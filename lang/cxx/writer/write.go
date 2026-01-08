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
	visited map[string]map[string]*fileNode // namespace -> filename -> fileNode
	headers map[string]map[string]*headerInfo // namespace -> typename -> header
}

type fileNode struct {
	chunks []chunk
	impts  []uniast.Import
}

type headerInfo struct {
	declaration string
	includes    []uniast.Import
}

type chunk struct {
	codes string
	line  int
}

func NewWriter(opts Options) *Writer {
	if opts.CompilerPath == "" {
		opts.CompilerPath = "g++"
	}
	return &Writer{
		Options: opts,
		visited: make(map[string]map[string]*fileNode),
		headers: make(map[string]map[string]*headerInfo),
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
	includeDir := filepath.Join(outdir, "include")
	srcDir := filepath.Join(outDir, mod.Dir, "src")

	// Create directories
	if err := os.MkdirAll(includeDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s failed: %v", includeDir, err)
	}
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s failed: %v", srcDir, err)
	}

	// Write header files first
	for namespace, nsHeaders := range w.headers {
		nsDir := includeDir
		if namespace != "" {
			nsDir = filepath.Join(includeDir, strings.ReplaceAll(namespace, "::", string(filepath.Separator)))
			if err := os.MkdirAll(nsDir, 0755); err != nil {
				return fmt.Errorf("mkdir %s failed: %v", nsDir, err)
			}
		}

		for typeName, header := range nsHeaders {
			headerPath := filepath.Join(nsDir, typeName+".h")
			if err := w.writeHeaderFile(headerPath, namespace, typeName, header); err != nil {
				return fmt.Errorf("write header %s failed: %v", headerPath, err)
			}
		}
	}

	// Write implementation files
	for namespace, nsFiles := range w.visited {
		nsDir := srcDir
		if namespace != "" {
			nsDir = filepath.Join(srcDir, strings.ReplaceAll(namespace, "::", string(filepath.Separator)))
			if err := os.MkdirAll(nsDir, 0755); err != nil {
				return fmt.Errorf("mkdir %s failed: %v", nsDir, err)
			}
		}

		for filename, f := range nsFiles {
			var sb strings.Builder

			// Write includes
			if len(f.impts) > 0 {
				writeImport(&sb, f.impts)
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

			// Determine file extension
			if !strings.HasSuffix(filename, ".cpp") && !strings.HasSuffix(filename, ".c") {
				filename = filename + ".cpp"
			}
			fpath := filepath.Join(nsDir, filename)

			if err := os.WriteFile(fpath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// Generate CMakeLists.txt if needed
	if err := w.generateCMakeLists(mod, outdir); err != nil {
		log.Error("generate CMakeLists.txt failed: %v", err)
	}

	return nil
}

func (w *Writer) writeHeaderFile(path, namespace, typeName string, header *headerInfo) error {
	var sb strings.Builder

	// Include guard
	guardName := strings.ToUpper(strings.ReplaceAll(namespace+"_"+typeName, "::", "_"))
	guardName = strings.ReplaceAll(guardName, "-", "_")
	guardName = strings.ReplaceAll(guardName, ".", "_")
	sb.WriteString("#ifndef ")
	sb.WriteString(guardName)
	sb.WriteString("_H\n")
	sb.WriteString("#define ")
	sb.WriteString(guardName)
	sb.WriteString("_H\n\n")

	// Includes
	if len(header.includes) > 0 {
		writeImport(&sb, header.includes)
	}

	// Namespace
	if namespace != "" {
		sb.WriteString("namespace ")
		sb.WriteString(strings.ReplaceAll(namespace, "::", " {\nnamespace "))
		sb.WriteString(" {\n\n")
	}

	// Declaration
	sb.WriteString(header.declaration)
	sb.WriteString("\n")

	// Close namespace
	if namespace != "" {
		parts := strings.Split(namespace, "::")
		for range parts {
			sb.WriteString("}\n")
		}
		sb.WriteString("\n")
	}

	// Close include guard
	sb.WriteString("#endif // ")
	sb.WriteString(guardName)
	sb.WriteString("_H\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func (w *Writer) appendPackage(repo *uniast.Repository, pkg *uniast.Package) error {
	for _, t := range pkg.Types {
		n := repo.GetNode(t.Identity)
		if err := w.appendType(n, pkg.PkgPath, t.File, t.Line, t.Content); err != nil {
			return fmt.Errorf("append type %s failed: %v", t.Name, err)
		}
	}
	for _, f := range pkg.Functions {
		if f.IsInterfaceMethod {
			continue
		}
		n := repo.GetNode(f.Identity)
		if err := w.appendFunction(n, pkg.PkgPath, f.File, f.Line, f.Content); err != nil {
			return fmt.Errorf("append function %s failed: %v", f.Name, err)
		}
	}
	for _, v := range pkg.Vars {
		n := repo.GetNode(v.Identity)
		if err := w.appendVar(n, pkg.PkgPath, v.File, v.Line, v.Content); err != nil {
			return fmt.Errorf("append var %s failed: %v", v.Name, err)
		}
	}
	return nil
}

func (w *Writer) appendType(node *uniast.Node, pkg string, file string, line int, src string) error {
	namespace := w.extractNamespace(src, pkg)
	typeName := w.extractTypeName(src)

	ns := w.headers[namespace]
	if ns == nil {
		ns = make(map[string]*headerInfo)
		w.headers[namespace] = ns
	}

	header := ns[typeName]
	if header == nil {
		header = &headerInfo{
			includes: make([]uniast.Import, 0),
		}
		ns[typeName] = header
	}

	// Extract declaration (class/struct definition without implementation)
	header.declaration = w.extractDeclaration(src)

	// Collect dependencies as includes
	for _, v := range node.Dependencies {
		if v.PkgPath == "" || v.PkgPath == pkg {
			continue
		}
		header.includes = append(header.includes, uniast.Import{Path: v.PkgPath})
	}

	return nil
}

func (w *Writer) appendFunction(node *uniast.Node, pkg string, file string, line int, src string) error {
	namespace := w.extractNamespace(src, pkg)

	ns := w.visited[namespace]
	if ns == nil {
		ns = make(map[string]*fileNode)
		w.visited[namespace] = ns
	}

	var filename string
	if file == "" {
		filename = "functions.cpp"
	} else {
		filename = filepath.Base(file)
		if !strings.HasSuffix(filename, ".cpp") && !strings.HasSuffix(filename, ".c") {
			filename = filename + ".cpp"
		}
	}

	fs := ns[filename]
	if fs == nil {
		fs = &fileNode{
			chunks: make([]chunk, 0),
			impts:  make([]uniast.Import, 0),
		}
		ns[filename] = fs
	}

	// Collect dependencies as includes
	for _, v := range node.Dependencies {
		if v.PkgPath == "" || v.PkgPath == pkg {
			continue
		}
		fs.impts = append(fs.impts, uniast.Import{Path: v.PkgPath})
	}

	// Extract includes from source code
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

func (w *Writer) appendVar(node *uniast.Node, pkg string, file string, line int, src string) error {
	// Variables typically go in header files if they're constants, or in .cpp if they're globals
	// For simplicity, put them in implementation files
	return w.appendFunction(node, pkg, file, line, src)
}

func (w *Writer) extractNamespace(src, pkg string) string {
	// Try to extract namespace from source
	namespaceRegex := regexp.MustCompile(`namespace\s+(\w+(?:::\w+)*)`)
	matches := namespaceRegex.FindStringSubmatch(src)
	if len(matches) > 1 {
		return matches[1]
	}
	// Fallback to package path
	return strings.ReplaceAll(pkg, "/", "::")
}

func (w *Writer) extractTypeName(src string) string {
	// Match: class ClassName
	classRegex := regexp.MustCompile(`(?:class|struct|enum)\s+(\w+)`)
	matches := classRegex.FindStringSubmatch(src)
	if len(matches) > 1 {
		return matches[1]
	}
	return "Unknown"
}

func (w *Writer) extractDeclaration(src string) string {
	// Extract class/struct declaration without implementation
	// This is a simplified version - in practice, you'd need a proper C++ parser
	lines := strings.Split(src, "\n")
	var decl strings.Builder
	inBody := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "{") {
			inBody = true
			braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		}
		if inBody {
			decl.WriteString(line)
			decl.WriteString("\n")
			if braceCount <= 0 && strings.Contains(trimmed, "}") {
				break
			}
		} else {
			decl.WriteString(line)
			decl.WriteString("\n")
		}
	}

	result := decl.String()
	if result == "" {
		return src
	}
	return result
}

func (w *Writer) SplitImportsAndCodes(src string) (codes string, imports []uniast.Import, err error) {
	// Parse C++ include directives
	includeRegex := regexp.MustCompile(`(?m)^\s*#include\s+([<"].*?[>"])`)
	matches := includeRegex.FindAllStringSubmatch(src, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, uniast.Import{Path: match[1]})
		}
	}

	// Remove include directives from source
	codes = includeRegex.ReplaceAllString(src, "")
	codes = strings.TrimSpace(codes)

	return codes, imports, nil
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	// Convert to C++ include format
	// For now, assume it's a local header
	includePath := strings.ReplaceAll(id.PkgPath, "/", "/")
	return uniast.Import{Path: "\"" + includePath + ".h\""}, nil
}

func (w *Writer) PatchImports(impts []uniast.Import, file []byte) ([]byte, error) {
	// Parse existing includes
	existingImports, err := w.parseImportsFromFile(file)
	if err != nil {
		return nil, utils.WrapError(err, "fail parse includes from file")
	}

	// Merge imports
	merged := mergeImports(existingImports, impts)
	if len(merged) == len(existingImports) {
		return file, nil
	}

	// Find include section
	content := string(file)
	includeStart := strings.Index(content, "#include")
	if includeStart == -1 {
		// No includes found, insert at beginning
		var sb strings.Builder
		writeImport(&sb, merged)
		sb.WriteString("\n")
		sb.WriteString(content)
		return []byte(sb.String()), nil
	}

	// Find end of include section
	includeEnd := includeStart
	lines := strings.Split(content[includeStart:], "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i > 0 && !strings.HasPrefix(trimmed, "#include") && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			includeEnd = includeStart + strings.Index(content[includeStart:], line)
			break
		}
	}
	if includeEnd == includeStart {
		includeEnd = len(content)
	}

	// Replace include section
	var sb strings.Builder
	sb.WriteString(content[:includeStart])
	writeImport(&sb, merged)
	sb.WriteString("\n")
	sb.WriteString(content[includeEnd:])
	return []byte(sb.String()), nil
}

func (w *Writer) parseImportsFromFile(file []byte) ([]uniast.Import, error) {
	_, imports, err := w.SplitImportsAndCodes(string(file))
	return imports, err
}

func (w *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder

	// Write includes
	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	return []byte(sb.String()), nil
}

func (w *Writer) generateCMakeLists(mod *uniast.Module, outdir string) error {
	var sb strings.Builder
	sb.WriteString("cmake_minimum_required(VERSION 3.10)\n")
	sb.WriteString("project(")
	sb.WriteString(mod.Name)
	sb.WriteString(")\n\n")
	sb.WriteString("set(CMAKE_CXX_STANDARD 17)\n")
	sb.WriteString("set(CMAKE_CXX_STANDARD_REQUIRED ON)\n\n")
	sb.WriteString("include_directories(include)\n\n")
	sb.WriteString("file(GLOB_RECURSE SOURCES \"src/*.cpp\" \"src/*.c\")\n\n")
	sb.WriteString("add_executable(")
	sb.WriteString(mod.Name)
	sb.WriteString(" ${SOURCES})\n")

	cmakePath := filepath.Join(outdir, "CMakeLists.txt")
	if err := os.WriteFile(cmakePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write CMakeLists.txt failed: %v", err)
	}

	// Try to format with clang-format if available
	cmd := exec.Command("clang-format", "-i", "src/**/*.cpp", "include/**/*.h")
	cmd.Dir = outdir
	if err := cmd.Run(); err != nil {
		log.Error("clang-format failed: %v", err)
	}

	return nil
}
