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

	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

type GitHubBackend struct {
	org, repo string
	client    *github.Client
}

func NewGitHubBackend(org, repo, token string) *GitHubBackend {
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
		client: github.NewClient(tc),
	}
}

var _ Backend = (*GitHubBackend)(nil)

func (gh *GitHubBackend) ListObjects(prefix string) ([]Object, error) {
	ctx := context.TODO()

	branch, path, err := gh.split(prefix)
	if err != nil {
		return nil, err
	}

	tree, _, err := gh.client.Git.GetTree(ctx, gh.org, gh.repo, branch, true)
	if err != nil {
		return nil, err
	}

	var (
		objs    []Object
		commits = make(map[string]*github.Commit)
	)

	for _, entry := range tree.Entries {
		if !strings.HasPrefix(entry.GetPath(), path+"/") {
			continue
		}

		if typ := entry.GetType(); typ != "blob" {
			return nil, fmt.Errorf("%q: unexpected type: %q", entry.GetPath(), typ)
		}

		commit, ok := commits[entry.GetSHA()]
		if !ok {
			if commit, _, err = gh.client.Git.GetCommit(ctx, gh.org, gh.repo, entry.GetSHA()); err != nil {
				return nil, fmt.Errorf("%q: %w", entry.GetPath(), err)
			}

			commits[entry.GetSHA()] = commit
		}

		objs = append(objs, Object{
			Path:         strings.TrimPrefix(entry.GetPath(), path+"/"),
			Content:      []byte{},
			LastModified: commit.GetAuthor().GetDate(),
		})
	}

	return objs, nil
}

func (gh *GitHubBackend) GetObject(prefix string) (Object, error) {
	ctx := context.TODO()

	branch, path, err := gh.split(prefix)
	if err != nil {
		return Object{}, err
	}

	blob, _, err := gh.client.Git.GetBlob(ctx, gh.org, gh.repo, branch+"/"+path)
	if err != nil {
		return Object{}, err
	}

	commit, _, err := gh.client.Git.GetCommit(ctx, gh.org, gh.repo, blob.GetSHA())
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
		LastModified: commit.GetAuthor().GetDate(),
	}, nil
}

func (gh *GitHubBackend) PutObject(path string, content []byte) error {
	return errors.New("not implemented")
}

func (gh *GitHubBackend) DeleteObject(path string) error {
	return errors.New("not implemented")
}

func (gh *GitHubBackend) split(prefix string) (branch, path string, err error) {
	if i := strings.IndexRune(prefix, '/'); i != -1 {
		if branch, err = url.PathUnescape(prefix[:i]); err != nil {
			return "", "", err
		}
		path = prefix[i+1:]
	}

	if branch == "" {
		return "", "", errors.New("branch is empty or missing")
	}

	return branch, path, nil
}
