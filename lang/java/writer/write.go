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
	visited map[string]map[string]*fileNode // pkg -> className -> fileNode
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
		opts.CompilerPath = "javac"
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
	for pkgPath, pkg := range w.visited {
		// Convert package path to directory structure
		pkgDir := filepath.Join(outdir, strings.ReplaceAll(pkgPath, ".", string(filepath.Separator)))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s failed: %v", pkgDir, err)
		}

		for className, f := range pkg {
			var sb strings.Builder

			// Write package declaration
			if pkgPath != "" {
				sb.WriteString("package ")
				sb.WriteString(pkgPath)
				sb.WriteString(";\n\n")
			}

			// Merge imports
			var fimpts []uniast.Import
			if fi, ok := mod.Files[filepath.Join(mod.Dir, className)]; ok && fi.Imports != nil {
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

			// Determine file name (Java: one public class per file)
			filename := className
			if !strings.HasSuffix(filename, ".java") {
				filename = filename + ".java"
			}
			fpath := filepath.Join(pkgDir, filename)

			if err := os.WriteFile(fpath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// Generate pom.xml if needed
	if err := w.generatePomXml(mod, outdir); err != nil {
		log.Error("generate pom.xml failed: %v", err)
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
	p := w.visited[pkg]
	if p == nil {
		p = make(map[string]*fileNode)
		w.visited[pkg] = p
	}

	// Extract class name from content for Java (one class per file)
	className := w.extractClassName(src, file)
	if className == "" {
		className = "Default"
	}

	fs := p[className]
	if fs == nil {
		fs = &fileNode{
			chunks: make([]chunk, 0),
			impts:  make([]uniast.Import, 0),
		}
		p[className] = fs
	}

	// Collect dependencies as imports
	for _, v := range node.Dependencies {
		if v.PkgPath == "" || v.PkgPath == pkg {
			continue
		}
		// Convert to Java import format
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

func (w *Writer) extractClassName(src, file string) string {
	// Try to extract from file name first
	if file != "" {
		base := filepath.Base(file)
		if strings.HasSuffix(base, ".java") {
			return strings.TrimSuffix(base, ".java")
		}
	}

	// Try to extract from source code
	// Match: public class ClassName
	classRegex := regexp.MustCompile(`(?:public\s+)?(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	matches := classRegex.FindStringSubmatch(src)
	if len(matches) > 1 {
		return matches[1]
	}

	// Match: public interface InterfaceName
	interfaceRegex := regexp.MustCompile(`(?:public\s+)?interface\s+(\w+)`)
	matches = interfaceRegex.FindStringSubmatch(src)
	if len(matches) > 1 {
		return matches[1]
	}

	// Match: public enum EnumName
	enumRegex := regexp.MustCompile(`(?:public\s+)?enum\s+(\w+)`)
	matches = enumRegex.FindStringSubmatch(src)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (w *Writer) SplitImportsAndCodes(src string) (codes string, imports []uniast.Import, err error) {
	// Parse Java import statements using regex
	importRegex := regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([\w.]+(?:\.[*])?)\s*;`)
	matches := importRegex.FindAllStringSubmatch(src, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, uniast.Import{Path: match[1]})
		}
	}

	// Remove import statements from source
	codes = importRegex.ReplaceAllString(src, "")
	codes = strings.TrimSpace(codes)

	return codes, imports, nil
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	// Convert package path to Java import format
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
	if importStart == -1 {
		// No imports found, insert after package
		packageEnd := strings.Index(content, ";")
		if packageEnd == -1 {
			packageEnd = 0
		} else {
			packageEnd++
		}
		var sb strings.Builder
		sb.WriteString(content[:packageEnd])
		sb.WriteString("\n")
		writeImport(&sb, merged)
		sb.WriteString("\n")
		sb.WriteString(content[packageEnd:])
		return []byte(sb.String()), nil
	}

	// Find end of import section
	importEnd := importStart
	lines := strings.Split(content[importStart:], "\n")
	for i, line := range lines {
		if i > 0 && !strings.Contains(line, "import ") && strings.TrimSpace(line) != "" {
			importEnd = importStart + strings.Index(content[importStart:], line)
			break
		}
	}
	if importEnd == importStart {
		importEnd = len(content)
	}

	// Replace import section
	var sb strings.Builder
	sb.WriteString(content[:importStart])
	writeImport(&sb, merged)
	sb.WriteString("\n")
	sb.WriteString(content[importEnd:])
	return []byte(sb.String()), nil
}

func (w *Writer) parseImportsFromFile(file []byte) ([]uniast.Import, error) {
	_, imports, err := w.SplitImportsAndCodes(string(file))
	return imports, err
}

func (w *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder

	// Write package declaration
	if fi.Package != "" {
		sb.WriteString("package ")
		sb.WriteString(string(fi.Package))
		sb.WriteString(";\n\n")
	}

	// Write imports
	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	return []byte(sb.String()), nil
}

func (w *Writer) generatePomXml(mod *uniast.Module, outdir string) error {
	// Extract groupId and artifactId from module name
	// Format: groupId:artifactId:version
	parts := strings.Split(mod.Name, ":")
	groupId := "com.example"
	artifactId := mod.Name
	version := "1.0.0"

	if len(parts) >= 1 {
		groupId = parts[0]
	}
	if len(parts) >= 2 {
		artifactId = parts[1]
	}
	if len(parts) >= 3 {
		version = parts[2]
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<project xmlns=\"http://maven.apache.org/POM/4.0.0\"\n")
	sb.WriteString("         xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\n")
	sb.WriteString("         xsi:schemaLocation=\"http://maven.apache.org/POM/4.0.0\n")
	sb.WriteString("         http://maven.apache.org/xsd/maven-4.0.0.xsd\">\n")
	sb.WriteString("    <modelVersion>4.0.0</modelVersion>\n\n")
	sb.WriteString("    <groupId>")
	sb.WriteString(groupId)
	sb.WriteString("</groupId>\n")
	sb.WriteString("    <artifactId>")
	sb.WriteString(artifactId)
	sb.WriteString("</artifactId>\n")
	sb.WriteString("    <version>")
	sb.WriteString(version)
	sb.WriteString("</version>\n\n")
	sb.WriteString("    <properties>\n")
	sb.WriteString("        <maven.compiler.source>17</maven.compiler.source>\n")
	sb.WriteString("        <maven.compiler.target>17</maven.compiler.target>\n")
	sb.WriteString("        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>\n")
	sb.WriteString("    </properties>\n\n")

	if len(mod.Dependencies) > 0 {
		sb.WriteString("    <dependencies>\n")
		for name, dep := range mod.Dependencies {
			depParts := strings.Split(dep, "@")
			depName := name
			depVersion := "LATEST"
			if len(depParts) >= 2 {
				depVersion = depParts[1]
			}
			depGroupId := strings.Split(depName, ":")[0]
			depArtifactId := depName
			if len(strings.Split(depName, ":")) >= 2 {
				depArtifactId = strings.Split(depName, ":")[1]
			}

			sb.WriteString("        <dependency>\n")
			sb.WriteString("            <groupId>")
			sb.WriteString(depGroupId)
			sb.WriteString("</groupId>\n")
			sb.WriteString("            <artifactId>")
			sb.WriteString(depArtifactId)
			sb.WriteString("</artifactId>\n")
			sb.WriteString("            <version>")
			sb.WriteString(depVersion)
			sb.WriteString("</version>\n")
			sb.WriteString("        </dependency>\n")
		}
		sb.WriteString("    </dependencies>\n\n")
	}

	sb.WriteString("</project>\n")

	pomPath := filepath.Join(outdir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write pom.xml failed: %v", err)
	}

	// Try to format with maven if available
	cmd := exec.Command("mvn", "fmt:format")
	cmd.Dir = outdir
	if err := cmd.Run(); err != nil {
		log.Error("mvn fmt:format failed: %v", err)
	}

	return nil
}
