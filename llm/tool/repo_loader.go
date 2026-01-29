/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tool

import (
	"path/filepath"
	"sync"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// TestRepoASTsDir is the default testdata directory for AST JSON files; used by tests in this package and others (e.g. llm/agent, llm/mcp).
const TestRepoASTsDir = "../../testdata/asts"

// LoadReposIntoMap loads all *.json repository files from dir into m (repo name -> *uniast.Repository).
// If onLoadError is non-nil, it is called for each file that fails to load and loading continues;
// otherwise the first load error is returned.
func LoadReposIntoMap(dir string, m *sync.Map, onLoadError func(file string, err error)) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	for _, f := range files {
		repo, err := uniast.LoadRepo(f)
		if err != nil {
			if onLoadError != nil {
				onLoadError(f, err)
				continue
			}
			return err
		}
		m.Store(repo.Name, repo)
	}
	return nil
}
