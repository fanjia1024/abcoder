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

	// Group imports by category
	javaImports := []uniast.Import{}
	javaxImports := []uniast.Import{}
	orgImports := []uniast.Import{}
	comImports := []uniast.Import{}
	otherImports := []uniast.Import{}

	for _, imp := range impts {
		path := strings.TrimSpace(imp.Path)
		if strings.HasPrefix(path, "java.") {
			javaImports = append(javaImports, imp)
		} else if strings.HasPrefix(path, "javax.") {
			javaxImports = append(javaxImports, imp)
		} else if strings.HasPrefix(path, "org.") {
			orgImports = append(orgImports, imp)
		} else if strings.HasPrefix(path, "com.") {
			comImports = append(comImports, imp)
		} else {
			otherImports = append(otherImports, imp)
		}
	}

	// Sort each group
	sortImports(javaImports)
	sortImports(javaxImports)
	sortImports(orgImports)
	sortImports(comImports)
	sortImports(otherImports)

	// Write imports in order with blank lines between groups
	writeImportGroup(sb, javaImports)
	writeImportGroup(sb, javaxImports)
	writeImportGroup(sb, orgImports)
	writeImportGroup(sb, comImports)
	writeImportGroup(sb, otherImports)
}

func writeImportGroup(sb *strings.Builder, impts []uniast.Import) {
	if len(impts) == 0 {
		return
	}

	for _, imp := range impts {
		writeSingleImport(sb, imp)
	}
	sb.WriteString("\n")
}

func writeSingleImport(sb *strings.Builder, v uniast.Import) {
	sb.WriteString("import ")
	if v.Alias != nil {
		sb.WriteString("static ")
	}
	sb.WriteString(v.Path)
	if !strings.HasSuffix(v.Path, ";") {
		sb.WriteString(";")
	}
	sb.WriteString("\n")
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
