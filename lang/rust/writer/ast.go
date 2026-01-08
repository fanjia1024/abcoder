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

package writer

import (
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
	rustast "github.com/cloudwego/abcoder/lang/rust"
)

func writeImport(sb *strings.Builder, impts []uniast.Import) {
	if len(impts) == 0 {
		return
	}

	// Use existing Rust import merging logic
	fileContents := make([]string, len(impts))
	for i, imp := range impts {
		fileContents[i] = imp.Path
	}

	merged, err := rustast.GetAndMergeUse(fileContents)
	if err != nil {
		// Fallback: just write imports as-is
		for _, imp := range impts {
			writeSingleImport(sb, imp)
		}
		return
	}

	// Write merged use statements
	for _, imp := range merged {
		writeSingleImport(sb, imp)
	}
}

func writeSingleImport(sb *strings.Builder, v uniast.Import) {
	path := strings.TrimSpace(v.Path)
	// If path already contains "use ", use it as is
	if strings.HasPrefix(path, "use ") {
		sb.WriteString(path)
		if !strings.HasSuffix(path, ";") {
			sb.WriteString(";")
		}
		sb.WriteString("\n")
	} else {
		// Otherwise, format as use statement
		sb.WriteString("use ")
		sb.WriteString(path)
		if !strings.HasSuffix(path, ";") {
			sb.WriteString(";")
		}
		sb.WriteString("\n")
	}
}

// mergeImports merges two import lists, prioritizing priors (file-level imports)
func mergeImports(priors []uniast.Import, subs []uniast.Import) (ret []uniast.Import) {
	visited := make(map[string]bool, len(priors)+len(subs))
	ret = make([]uniast.Import, 0, len(priors)+len(subs))

	// Add priors first (they may have aliases)
	for _, v := range priors {
		key := strings.TrimSpace(v.Path)
		if !visited[key] {
			visited[key] = true
			ret = append(ret, v)
		}
	}

	// Add subs if not already present
	for _, v := range subs {
		key := strings.TrimSpace(v.Path)
		if !visited[key] {
			visited[key] = true
			ret = append(ret, v)
		}
	}

	return ret
}
