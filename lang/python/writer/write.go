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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

var _ uniast.Writer = (*Writer)(nil)

type Options struct {
	CompilerPath string
}

type Writer struct {
	Options
	visited map[string]map[string]*fileNode // module -> filename -> fileNode
}

type fileNode struct {
	chunks []chunk
	impts  []uniast.Import
}

type chunk struct {
	codes string
	line  int
}

func NewWriter(opts Options) *Writer {
	if opts.CompilerPath == "" {
		opts.CompilerPath = "python"
	}
	return &Writer{
		Options: opts,
		visited: make(map[string]map[string]*fileNode),
	}
}

func (w *Writer) WriteModule(repo *uniast.Repository, modPath string, outDir string) error {
	mod := repo.Modules[modPath]
	if mod == nil {
		return fmt.Errorf("module %s not found", modPath)
	}

	// Collect all packages
	for _, pkg := range mod.Packages {
		if err := w.appendPackage(repo, pkg); err != nil {
			return fmt.Errorf("write package %s failed: %v", pkg.PkgPath, err)
		}
	}

	// Write files
	outdir := filepath.Join(outDir, mod.Dir)
	for modulePath, module := range w.visited {
		// Convert module path to directory structure
		pkgDir := filepath.Join(outdir, strings.ReplaceAll(modulePath, ".", string(filepath.Separator)))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s failed: %v", pkgDir, err)
		}

		// Create __init__.py if package has multiple files
		if len(module) > 1 {
			initPath := filepath.Join(pkgDir, "__init__.py")
			if _, err := os.Stat(initPath); os.IsNotExist(err) {
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					return fmt.Errorf("write __init__.py failed: %v", err)
				}
			}
		}

		for filename, f := range module {
			var sb strings.Builder

			// Python doesn't have explicit package declaration
			// But we can add module docstring if needed

			// Merge imports
			var fimpts []uniast.Import
			if fi, ok := mod.Files[filepath.Join(mod.Dir, filename)]; ok && fi.Imports != nil {
				fimpts = fi.Imports
			}
			impts := mergeImports(fimpts, f.impts)
			if len(impts) > 0 {
				writeImport(&sb, impts)
			}

			// Sort chunks by line number
			sort.SliceStable(f.chunks, func(i, j int) bool {
				return f.chunks[i].line < f.chunks[j].line
			})

			// Write code chunks
			for _, c := range f.chunks {
				sb.WriteString(c.codes)
				sb.WriteString("\n\n")
			}

			// Ensure .py extension
			if !strings.HasSuffix(filename, ".py") {
				filename = filename + ".py"
			}
			fpath := filepath.Join(pkgDir, filename)

			if err := os.WriteFile(fpath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// Generate pyproject.toml if needed
	if err := w.generatePyProjectToml(mod, outdir); err != nil {
		log.Error("generate pyproject.toml failed: %v", err)
	}

	return nil
}

func (w *Writer) appendPackage(repo *uniast.Repository, pkg *uniast.Package) error {
	for _, v := range pkg.Vars {
		n := repo.GetNode(v.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, v.File, v.Line, v.Content); err != nil {
			return fmt.Errorf("append chunk for var %s failed: %v", v.Name, err)
		}
	}
	for _, f := range pkg.Functions {
		if f.IsInterfaceMethod {
			continue
		}
		n := repo.GetNode(f.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, f.File, f.Line, f.Content); err != nil {
			return fmt.Errorf("append chunk for function %s failed: %v", f.Name, err)
		}
	}
	for _, t := range pkg.Types {
		n := repo.GetNode(t.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, t.File, t.Line, t.Content); err != nil {
			return fmt.Errorf("append chunk for type %s failed: %v", t.Name, err)
		}
	}
	return nil
}

func (w *Writer) appendNode(node *uniast.Node, pkg string, isMain bool, file string, line int, src string) error {
	module := pkg
	if module == "" {
		module = "main"
	}

	m := w.visited[module]
	if m == nil {
		m = make(map[string]*fileNode)
		w.visited[module] = m
	}

	var filename string
	if file == "" {
		if isMain {
			filename = "main.py"
		} else {
			filename = "lib.py"
		}
	} else {
		filename = filepath.Base(file)
		if !strings.HasSuffix(filename, ".py") {
			filename = filename + ".py"
		}
	}

	fs := m[filename]
	if fs == nil {
		fs = &fileNode{
			chunks: make([]chunk, 0),
			impts:  make([]uniast.Import, 0),
		}
		m[filename] = fs
	}

	// Collect dependencies as imports
	for _, v := range node.Dependencies {
		if v.PkgPath == "" || v.PkgPath == pkg {
			continue
		}
		// Convert to Python import format
		importPath := strings.ReplaceAll(v.PkgPath, "/", ".")
		fs.impts = append(fs.impts, uniast.Import{Path: importPath})
	}

	// Extract imports from source code
	if cs, impts, err := w.SplitImportsAndCodes(src); err == nil {
		src = cs
		for _, v := range impts {
			fs.impts = append(fs.impts, v)
		}
	}

	fs.chunks = append(fs.chunks, chunk{
		codes: src,
		line:  line,
	})
	return nil
}

func (w *Writer) SplitImportsAndCodes(src string) (codes string, imports []uniast.Import, err error) {
	// Use regex patterns from python/spec.go
	patterns := []string{
		// Matches: import <anything> (on a single line)
		`(?m)^import\s+(.*)$`,
		// Matches: from <anything> import <anything> (on a single line, without parentheses)
		`(?m)^from\s+(.*?)\s+import\s+([^()\n]*)$`,
		// Matches: from <anything> import ( <anything> ) where <anything> can span multiple lines
		`(?m)^from\s+(.*?)\s+import\s+\(([\s\S]*?)\)$`,
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		matches := re.FindAllStringSubmatch(src, -1)
		for _, match := range matches {
			if len(match) > 0 {
				imports = append(imports, uniast.Import{Path: match[0]})
			}
		}
	}

	// Remove import statements from source
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		src = re.ReplaceAllString(src, "")
	}

	codes = strings.TrimSpace(src)
	return codes, imports, nil
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	// Convert package path to Python import format
	importPath := strings.ReplaceAll(id.PkgPath, "/", ".")
	return uniast.Import{Path: importPath}, nil
}

func (w *Writer) PatchImports(impts []uniast.Import, file []byte) ([]byte, error) {
	// Parse existing imports
	existingImports, err := w.parseImportsFromFile(file)
	if err != nil {
		return nil, utils.WrapError(err, "fail parse imports from file")
	}

	// Merge imports
	merged := mergeImports(existingImports, impts)
	if len(merged) == len(existingImports) {
		return file, nil
	}

	// Find import section
	content := string(file)
	importStart := strings.Index(content, "import ")
	fromStart := strings.Index(content, "from ")
	
	var firstImport int
	if importStart == -1 && fromStart == -1 {
		firstImport = 0
	} else if importStart == -1 {
		firstImport = fromStart
	} else if fromStart == -1 {
		firstImport = importStart
	} else {
		if importStart < fromStart {
			firstImport = importStart
		} else {
			firstImport = fromStart
		}
	}

	// Find end of import section
	importEnd := firstImport
	lines := strings.Split(content[firstImport:], "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i > 0 && !strings.HasPrefix(trimmed, "import ") && 
			!strings.HasPrefix(trimmed, "from ") && 
			trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			importEnd = firstImport + strings.Index(content[firstImport:], line)
			break
		}
	}
	if importEnd == firstImport {
		importEnd = len(content)
	}

	// Replace import section
	var sb strings.Builder
	sb.WriteString(content[:firstImport])
	writeImport(&sb, merged)
	if firstImport > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(content[importEnd:])
	return []byte(sb.String()), nil
}

func (w *Writer) parseImportsFromFile(file []byte) ([]uniast.Import, error) {
	_, imports, err := w.SplitImportsAndCodes(string(file))
	return imports, err
}

func (w *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder

	// Python doesn't have explicit package declaration
	// But we can add module docstring if needed

	// Write imports
	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	return []byte(sb.String()), nil
}

func (w *Writer) generatePyProjectToml(mod *uniast.Module, outdir string) error {
	var sb strings.Builder
	sb.WriteString("[build-system]\n")
	sb.WriteString("requires = [\"setuptools>=61.0\"]\n")
	sb.WriteString("build-backend = \"setuptools.build_meta\"\n\n")
	sb.WriteString("[project]\n")
	sb.WriteString("name = \"")
	sb.WriteString(mod.Name)
	sb.WriteString("\"\n")
	sb.WriteString("version = \"1.0.0\"\n")
	sb.WriteString("description = \"Generated project\"\n\n")

	if len(mod.Dependencies) > 0 {
		sb.WriteString("dependencies = [\n")
		for name, dep := range mod.Dependencies {
			depParts := strings.Split(dep, "@")
			depName := name
			if len(depParts) >= 2 && depParts[1] != "" {
				depName = name + "==" + depParts[1]
			}
			sb.WriteString("    \"")
			sb.WriteString(depName)
			sb.WriteString("\",\n")
		}
		sb.WriteString("]\n\n")
	}

	tomlPath := filepath.Join(outdir, "pyproject.toml")
	if err := os.WriteFile(tomlPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write pyproject.toml failed: %v", err)
	}

	// Try to format with black if available
	cmd := exec.Command("black", ".")
	cmd.Dir = outdir
	if err := cmd.Run(); err != nil {
		log.Error("black formatting failed: %v", err)
	}

	return nil
}
