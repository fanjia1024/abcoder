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

	// Group imports: stdlib, third-party, local
	stdlibImports := []uniast.Import{}
	thirdPartyImports := []uniast.Import{}
	localImports := []uniast.Import{}

	for _, imp := range impts {
		path := strings.TrimSpace(imp.Path)
		// Check if it's a standard library import
		if isStdLibImport(path) {
			stdlibImports = append(stdlibImports, imp)
		} else if strings.Contains(path, ".") && !strings.HasPrefix(path, ".") {
			// Third-party import (has dots but doesn't start with dot)
			thirdPartyImports = append(thirdPartyImports, imp)
		} else {
			// Local import
			localImports = append(localImports, imp)
		}
	}

	// Sort each group
	sortImports(stdlibImports)
	sortImports(thirdPartyImports)
	sortImports(localImports)

	// Write imports in order with blank lines between groups
	writeImportGroup(sb, stdlibImports)
	if len(stdlibImports) > 0 && (len(thirdPartyImports) > 0 || len(localImports) > 0) {
		sb.WriteString("\n")
	}
	writeImportGroup(sb, thirdPartyImports)
	if len(thirdPartyImports) > 0 && len(localImports) > 0 {
		sb.WriteString("\n")
	}
	writeImportGroup(sb, localImports)
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
	// If path already contains the full import statement, use it as is
	if strings.HasPrefix(path, "import ") || strings.HasPrefix(path, "from ") {
		sb.WriteString(path)
		if !strings.HasSuffix(path, "\n") {
			sb.WriteString("\n")
		}
	} else {
		// Otherwise, treat as module name
		sb.WriteString("import ")
		sb.WriteString(path)
		sb.WriteString("\n")
	}
}

func sortImports(impts []uniast.Import) {
	sort.Slice(impts, func(i, j int) bool {
		return impts[i].Path < impts[j].Path
	})
}

func isStdLibImport(path string) bool {
	// Common Python standard library modules
	stdlibModules := []string{
		"os", "sys", "json", "re", "datetime", "time", "collections",
		"itertools", "functools", "operator", "copy", "pickle",
		"io", "pathlib", "shutil", "glob", "fnmatch", "linecache",
		"tempfile", "fileinput", "stat", "filecmp", "mmap",
		"codecs", "stringprep", "readline", "rlcompleter",
		"struct", "codecs", "encodings", "locale", "gettext",
		"unicodedata", "string", "textwrap", "difflib",
		"types", "copyreg", "pprint", "reprlib", "enum",
		"numbers", "math", "cmath", "decimal", "fractions",
		"statistics", "random", "secrets", "hashlib", "hmac",
		"base64", "binascii", "array", "weakref", "gc",
		"inspect", "site", "fpectl", "atexit", "traceback",
		"__future__", "warnings", "contextlib", "abc", "atexit",
		"traceback", "gc", "inspect", "site", "code", "codeop",
		"py_compile", "compileall", "dis", "pickletools",
		"argparse", "getopt", "logging", "getpass", "curses",
		"platform", "errno", "ctypes", "threading", "multiprocessing",
		"concurrent", "subprocess", "sched", "queue", "select",
		"selectors", "asyncio", "socket", "ssl", "email",
		"http", "urllib", "html", "xml", "csv", "netrc",
		"xdrlib", "plistlib", "configparser", "netrc",
		"turtle", "cmd", "shlex", "tkinter", "typing",
		"dataclasses", "contextvars", "asyncio",
	}

	// Extract module name (first part before dot)
	moduleName := path
	if idx := strings.Index(path, "."); idx > 0 {
		moduleName = path[:idx]
	}
	if idx := strings.Index(path, " "); idx > 0 {
		moduleName = path[:idx]
	}

	for _, stdlib := range stdlibModules {
		if moduleName == stdlib {
			return true
		}
	}
	return false
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
