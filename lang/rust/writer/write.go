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
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
	rustast "github.com/cloudwego/abcoder/lang/rust"
)

var _ uniast.Writer = (*Writer)(nil)

type Options struct {
	CompilerPath string
}

type Writer struct {
	Options
	visited map[string]map[string]*fileNode // crate -> mod_path -> fileNode
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
		opts.CompilerPath = "cargo"
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
	crateName := mod.Name

	// Create src/ directory structure
	srcDir := filepath.Join(outdir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s failed: %v", srcDir, err)
	}

	// Group files by module path
	modFiles := make(map[string][]*fileInfo)
	for modPath, mod := range w.visited {
		for filename, f := range mod {
			modFiles[modPath] = append(modFiles[modPath], &fileInfo{
				name:   filename,
				fileNode: f,
			})
		}
	}

	// Write module files
	for modPath, files := range modFiles {
		var filePath string
		if modPath == "" || modPath == crateName {
			// Root module -> lib.rs
			filePath = filepath.Join(srcDir, "lib.rs")
		} else {
			// Submodule -> mod/mod.rs or mod.rs
			modDir := filepath.Join(srcDir, strings.ReplaceAll(modPath, "::", string(filepath.Separator)))
			if err := os.MkdirAll(modDir, 0755); err != nil {
				return fmt.Errorf("mkdir %s failed: %v", modDir, err)
			}
			filePath = filepath.Join(modDir, "mod.rs")
		}

		// Merge all files in this module
		var allChunks []chunk
		var allImports []uniast.Import
		for _, file := range files {
			allChunks = append(allChunks, file.chunks...)
			allImports = append(allImports, file.impts...)
		}

		// Sort chunks by line
		sort.SliceStable(allChunks, func(i, j int) bool {
			return allChunks[i].line < allChunks[j].line
		})

		// Merge imports using existing Rust import logic
		mergedImports := mergeImports(nil, allImports)

		var sb strings.Builder

		// Write use statements
		if len(mergedImports) > 0 {
			writeImport(&sb, mergedImports)
		}

		// Write code chunks
		for _, c := range allChunks {
			sb.WriteString(c.codes)
			sb.WriteString("\n\n")
		}

		if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
			return fmt.Errorf("write file %s failed: %v", filePath, err)
		}
	}

	// Generate Cargo.toml
	if err := w.generateCargoToml(mod, outdir, crateName); err != nil {
		log.Error("generate Cargo.toml failed: %v", err)
	}

	// Try to format with rustfmt
	cmd := exec.Command("rustfmt", "src/**/*.rs")
	cmd.Dir = outdir
	if err := cmd.Run(); err != nil {
		log.Error("rustfmt failed: %v", err)
	}

	return nil
}

type fileInfo struct {
	name     string
	*fileNode
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
	// Convert package path to module path
	modPath := strings.ReplaceAll(pkg, "/", "::")
	if modPath == "" {
		modPath = "main"
	}

	m := w.visited[modPath]
	if m == nil {
		m = make(map[string]*fileNode)
		w.visited[modPath] = m
	}

	var filename string
	if file == "" {
		if isMain {
			filename = "main.rs"
		} else {
			filename = "lib.rs"
		}
	} else {
		filename = filepath.Base(file)
		if !strings.HasSuffix(filename, ".rs") {
			filename = filename + ".rs"
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
		// Convert to Rust use format
		importPath := strings.ReplaceAll(v.PkgPath, "/", "::")
		fs.impts = append(fs.impts, uniast.Import{Path: "use " + importPath + ";"})
	}

	// Extract use statements from source code using existing Rust parser
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
	// Use existing Rust use statement parser
	useStatements, err := rustast.ParseUseStatements(src)
	if err != nil {
		return src, nil, nil
	}

	imports = useStatements

	// Remove use statements from source using GetRustContentDefine
	codes, err = rustast.GetRustContentDefine("", src)
	if err != nil {
		codes = src
	}

	return codes, imports, nil
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	// Convert package path to Rust use format
	importPath := strings.ReplaceAll(id.PkgPath, "/", "::")
	return uniast.Import{Path: "use " + importPath + ";"}, nil
}

func (w *Writer) PatchImports(impts []uniast.Import, file []byte) ([]byte, error) {
	// Parse existing use statements
	existingImports, err := rustast.ParseUseStatements(string(file))
	if err != nil {
		return nil, utils.WrapError(err, "fail parse use statements from file")
	}

	// Merge imports
	merged := mergeImports(existingImports, impts)
	if len(merged) == len(existingImports) {
		return file, nil
	}

	// Get content without use statements
	content, err := rustast.GetRustContentDefine("", string(file))
	if err != nil {
		content = string(file)
	}

	// Rebuild file with merged imports
	var sb strings.Builder
	writeImport(&sb, merged)
	sb.WriteString("\n")
	sb.WriteString(content)
	return []byte(sb.String()), nil
}

func (w *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder

	// Rust doesn't have explicit package declaration at file level
	// Module declarations are done with `mod` keyword

	// Write use statements
	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	return []byte(sb.String()), nil
}

func (w *Writer) generateCargoToml(mod *uniast.Module, outdir string, crateName string) error {
	var sb strings.Builder
	sb.WriteString("[package]\n")
	sb.WriteString("name = \"")
	sb.WriteString(crateName)
	sb.WriteString("\"\n")
	sb.WriteString("version = \"1.0.0\"\n")
	sb.WriteString("edition = \"2021\"\n\n")

	if len(mod.Dependencies) > 0 {
		sb.WriteString("[dependencies]\n")
		for name, dep := range mod.Dependencies {
			depParts := strings.Split(dep, "@")
			depName := name
			depVersion := "1.0"
			if len(depParts) >= 2 && depParts[1] != "" {
				depVersion = depParts[1]
			}
			sb.WriteString(depName)
			sb.WriteString(" = \"")
			sb.WriteString(depVersion)
			sb.WriteString("\"\n")
		}
		sb.WriteString("\n")
	}

	tomlPath := filepath.Join(outdir, "Cargo.toml")
	if err := os.WriteFile(tomlPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write Cargo.toml failed: %v", err)
	}

	return nil
}
