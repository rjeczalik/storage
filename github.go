/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

// TODO: support more use-cases:
//
//       * no multitenancy --depth=0 (org, repo and branch configured at a startup)
//       * multitenancy --depth=2 (repo and branch as a part of a prefix)
//       * writing files (?) - currently the storage is read-only, as writing
//         (committing) usually is done elsewhere at github.com (GitOps)

var errReadOnly = errors.New("not implemented: read-only storage")

// GitHubBackend is a storage backend for a repository hosted at github.com.
//
// NOTE: Currently only single repository is supported - github.com/:org/:repo.
//       The chartmuseum must be started with --depth=1 flag to support passing
//       branch name as a prefix.
// NOTE: The storage is currently read-only, to function properly
//       all caches must be disabled (via --cache=none flag).
type GitHubBackend struct {
	org, repo string // required
	prefix    string // optional
	client    *github.Client
}

// NewGitHubBackend gives new instance of GitHubBackend.
func NewGitHubBackend(org, repo, token string, prefix string) *GitHubBackend {
	var (
		ctx = context.TODO()
		ts  = oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: token,
			},
		)
		tc = oauth2.NewClient(ctx, ts)
	)

	return &GitHubBackend{
		org:    org,
		repo:   repo,
		prefix: strings.Trim(prefix, "/"),
		client: github.NewClient(tc),
	}
}

var _ Backend = (*GitHubBackend)(nil)

// ListObjects recursively lists all blobs rooted at prefix (= :branch/:dir).
func (gh *GitHubBackend) ListObjects(prefix string) ([]Object, error) {
	ctx := context.TODO()

	branch, dir, err := gh.split(prefix)
	if err != nil {
		return nil, err
	}

	if dir != "" {
		dir = dir + "/"
	}

	tree, _, err := gh.client.Git.GetTree(ctx, gh.org, gh.repo, branch, true)
	if err != nil {
		return nil, err
	}

	var objs []Object

	for _, entry := range tree.Entries {
		if dir != "" && !strings.HasPrefix(entry.GetPath(), dir) {
			continue
		}

		if typ := entry.GetType(); typ != "blob" {
			return nil, fmt.Errorf("%q: unexpected type: %q", entry.GetPath(), typ)
		}

		objs = append(objs, Object{
			Path:         strings.TrimPrefix(entry.GetPath(), dir),
			Content:      []byte{},
			LastModified: time.Now(), // prevents caching for read-only storage
		})
	}

	return objs, nil
}

// GetObject retrieves a blob from prefix (= :branch/:file).
func (gh *GitHubBackend) GetObject(prefix string) (Object, error) {
	ctx := context.TODO()

	branch, path, err := gh.split(prefix)
	if err != nil {
		return Object{}, err
	}

	tree, _, err := gh.client.Git.GetTree(ctx, gh.org, gh.repo, branch, true)
	if err != nil {
		return Object{}, err
	}

	var sha string

	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" && strings.EqualFold(entry.GetPath(), path) {
			sha = entry.GetSHA()
			break
		}
	}

	if sha == "" {
		return Object{}, fmt.Errorf("%q: unable to find %q in the tree", prefix, path)
	}

	blob, _, err := gh.client.Git.GetBlob(ctx, gh.org, gh.repo, sha)
	if err != nil {
		return Object{}, err
	}

	if enc := blob.GetEncoding(); enc != "base64" {
		return Object{}, fmt.Errorf("unexpected encoding: %q", enc)
	}

	content, err := base64.StdEncoding.DecodeString(blob.GetContent())
	if err != nil {
		return Object{}, err
	}

	return Object{
		Path:         path,
		Content:      content,
		LastModified: time.Now(), // prevents caching for read-only storage
	}, nil
}

// PutObject is currently not implemented (read-only storage).
func (*GitHubBackend) PutObject(string, []byte) error { return errReadOnly }

// DeleteObject is currently not implemented (read-only storage).
func (*GitHubBackend) DeleteObject(string) error { return errReadOnly }

// ReadOnly allows for inspecting the storage via interface upgrade
// whether it is read-only.
func (*GitHubBackend) ReadOnly() bool { return true }

func (gh *GitHubBackend) split(prefix string) (branch, path string, err error) {
	s := strings.Trim(prefix, "/")

	if i := strings.IndexRune(s, '/'); i != -1 {
		branch, path = s[:i], s[i+1:]
	} else {
		branch = s
	}

	if branch, err = url.PathUnescape(branch); err != nil {
		return "", "", fmt.Errorf("%q: %w", prefix, err)
	}

	if branch == "" {
		return "", "", fmt.Errorf("%q: branch is empty or missing", prefix)
	}

	switch {
	case gh.prefix != "" && path != "":
		path = gh.prefix + "/" + path
	case gh.prefix != "":
		path = gh.prefix
	}

	return branch, path, nil
}
