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
	"errors"
	"fmt"
)

type GitHubBackend struct {
}

var _ Backend = (*GitHubBackend)(nil)

func (gh *GitHubBackend) ListObjects(prefix string) ([]Object, error) {
	fmt.Printf("[DEBUG] ListObjects: prefix=%q\n", prefix)

	return nil, errors.New("not implemented")
}

func (gh *GitHubBackend) GetObject(path string) (Object, error) {
	fmt.Printf("[DEBUG] GetObject: path=%q\n", path)

	return Object{}, errors.New("not implemented")
}

func (gh *GitHubBackend) PutObject(path string, content []byte) error {
	fmt.Printf("[DEBUG] PutObject: path=%q, content=%d\n", path, len(content))

	return errors.New("not implemented")
}

func (gh *GitHubBackend) DeleteObject(path string) error {
	fmt.Printf("[DEBUG] DeleteObject: path=%q\n", path)

	return errors.New("not implemented")
}
