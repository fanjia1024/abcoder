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
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func writeImport(sb *strings.Builder, impts []uniast.Import) {
	if len(impts) == 0 {
		return
	}

	// Group includes: system includes (<>) vs local includes ("")
	systemIncludes := []uniast.Import{}
	localIncludes := []uniast.Import{}

	for _, imp := range impts {
		path := strings.TrimSpace(imp.Path)
		if strings.HasPrefix(path, "<") && strings.HasSuffix(path, ">") {
			systemIncludes = append(systemIncludes, imp)
		} else {
			localIncludes = append(localIncludes, imp)
		}
	}

	// Sort each group
	sortImports(systemIncludes)
	sortImports(localIncludes)

	// Write system includes first, then local includes with blank line between
	writeImportGroup(sb, systemIncludes)
	if len(systemIncludes) > 0 && len(localIncludes) > 0 {
		sb.WriteString("\n")
	}
	writeImportGroup(sb, localIncludes)
}

func writeImportGroup(sb *strings.Builder, impts []uniast.Import) {
	if len(impts) == 0 {
		return
	}

	for _, imp := range impts {
		writeSingleImport(sb, imp)
	}
}

func writeSingleImport(sb *strings.Builder, v uniast.Import) {
	path := strings.TrimSpace(v.Path)
	// If path already contains #include, use it as is
	if strings.HasPrefix(path, "#include") {
		sb.WriteString(path)
		if !strings.HasSuffix(path, "\n") {
			sb.WriteString("\n")
		}
	} else {
		// Otherwise, format as #include directive
		sb.WriteString("#include ")
		// Determine if it's a system or local include
		if strings.Contains(path, ".") && !strings.HasPrefix(path, "<") {
			// Local include
			if !strings.HasPrefix(path, "\"") {
				sb.WriteString("\"")
			}
			sb.WriteString(path)
			if !strings.HasSuffix(path, "\"") {
				sb.WriteString("\"")
			}
		} else {
			// System include
			if !strings.HasPrefix(path, "<") {
				sb.WriteString("<")
			}
			sb.WriteString(path)
			if !strings.HasSuffix(path, ">") {
				sb.WriteString(">")
			}
		}
		sb.WriteString("\n")
	}
}

func sortImports(impts []uniast.Import) {
	sort.Slice(impts, func(i, j int) bool {
		return impts[i].Path < impts[j].Path
	})
}

// mergeImports merges two import lists, prioritizing priors (file-level imports)
func mergeImports(priors []uniast.Import, subs []uniast.Import) (ret []uniast.Import) {
	visited := make(map[string]bool, len(priors)+len(subs))
	ret = make([]uniast.Import, 0, len(priors)+len(subs))

	// Add priors first (they may have specific formats)
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
