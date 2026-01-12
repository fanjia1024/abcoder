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

package translate

import (
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// StructureAdapter adapts source language structure to target language
type StructureAdapter struct {
	source uniast.Language
	target uniast.Language
}

// NewStructureAdapter creates a new StructureAdapter
func NewStructureAdapter(source, target uniast.Language) *StructureAdapter {
	return &StructureAdapter{
		source: source,
		target: target,
	}
}

// AdaptRepository creates a target Repository structure from source
func (a *StructureAdapter) AdaptRepository(src *uniast.Repository, targetModName string) *uniast.Repository {
	repo := uniast.NewRepository(targetModName)
	repo.Path = src.Path
	repo.ASTVersion = src.ASTVersion
	repo.ToolVersion = src.ToolVersion
	return &repo
}

// AdaptModule creates a target Module structure from source
func (a *StructureAdapter) AdaptModule(src *uniast.Module) *uniast.Module {
	modName := a.convertModuleName(src.Name)
	mod := uniast.NewModule(modName, src.Dir, a.target)

	// Copy and convert dependencies
	for depName, depVersion := range src.Dependencies {
		// For now, keep dependencies as is
		// In the future, we could map known dependencies to equivalents
		mod.Dependencies[depName] = depVersion
	}

	return mod
}

// AdaptPackage creates a target Package structure from source
func (a *StructureAdapter) AdaptPackage(src *uniast.Package) *uniast.Package {
	pkgPath := a.convertPackagePath(string(src.PkgPath))
	return &uniast.Package{
		IsMain:    src.IsMain,
		IsTest:    src.IsTest,
		PkgPath:   uniast.PkgPath(pkgPath),
		Functions: make(map[string]*uniast.Function),
		Types:     make(map[string]*uniast.Type),
		Vars:      make(map[string]*uniast.Var),
	}
}

// AdaptFile creates a target File structure from source
func (a *StructureAdapter) AdaptFile(src *uniast.File) *uniast.File {
	return &uniast.File{
		Path:    a.convertFilePath(src.Path),
		Package: uniast.PkgPath(a.convertPackagePath(string(src.Package))),
		Imports: a.convertImports(src.Imports),
	}
}

// convertModuleName converts a module name to target language convention
func (a *StructureAdapter) convertModuleName(name string) string {
	switch {
	case a.source == uniast.Java && a.target == uniast.Golang:
		// com.example.project -> github.com/example/project
		return GetGoModuleNameFromGroupId(name)
	case a.source == uniast.Java && a.target == uniast.Rust:
		// com.example.project -> example_project
		parts := strings.Split(name, ".")
		if len(parts) > 1 {
			return strings.Join(parts[1:], "_")
		}
		return toSnakeCase(name)
	case a.source == uniast.Java && a.target == uniast.Python:
		// com.example.project -> example.project
		parts := strings.Split(name, ".")
		if len(parts) > 1 {
			return strings.Join(parts[1:], ".")
		}
		return name
	case a.source == uniast.Golang && a.target == uniast.Java:
		// github.com/example/project -> com.example.project
		name = strings.TrimPrefix(name, "github.com/")
		parts := strings.Split(name, "/")
		return "com." + strings.Join(parts, ".")
	case a.source == uniast.Golang && a.target == uniast.Rust:
		// github.com/example/project -> example_project
		parts := strings.Split(name, "/")
		if len(parts) > 0 {
			return toSnakeCase(parts[len(parts)-1])
		}
		return toSnakeCase(name)
	case a.source == uniast.Golang && a.target == uniast.Python:
		// github.com/example/project -> example.project
		name = strings.TrimPrefix(name, "github.com/")
		return strings.ReplaceAll(name, "/", ".")
	case a.source == uniast.Python && a.target == uniast.Golang:
		// example.project -> github.com/example/project
		return "github.com/" + strings.ReplaceAll(name, ".", "/")
	case a.source == uniast.Python && a.target == uniast.Java:
		// example.project -> com.example.project
		return "com." + name
	case a.source == uniast.Python && a.target == uniast.Rust:
		// example.project -> example_project
		return toSnakeCase(strings.ReplaceAll(name, ".", "_"))
	case a.source == uniast.Rust && a.target == uniast.Golang:
		// example_project -> github.com/example/project
		return "github.com/" + strings.ReplaceAll(name, "_", "/")
	case a.source == uniast.Rust && a.target == uniast.Java:
		// example_project -> com.example.project
		return "com." + strings.ReplaceAll(name, "_", ".")
	case a.source == uniast.Rust && a.target == uniast.Python:
		// example_project -> example.project
		return strings.ReplaceAll(name, "_", ".")
	default:
		return name
	}
}

// convertPackagePath converts a package path to target language convention
func (a *StructureAdapter) convertPackagePath(path string) string {
	switch {
	case a.source == uniast.Java && a.target == uniast.Golang:
		// com.example.project.model -> model
		return ConvertJavaPackageToGoModule(path)
	case a.source == uniast.Java && a.target == uniast.Rust:
		// com.example.project.model -> model
		parts := strings.Split(path, ".")
		if len(parts) > 0 {
			return toSnakeCase(parts[len(parts)-1])
		}
		return path
	case a.source == uniast.Java && a.target == uniast.Python:
		// com.example.project.model -> model
		parts := strings.Split(path, ".")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return path
	case a.source == uniast.Golang && a.target == uniast.Java:
		// github.com/example/project/model -> com.example.project.model
		path = strings.TrimPrefix(path, "github.com/")
		return "com." + strings.ReplaceAll(path, "/", ".")
	case a.source == uniast.Golang && a.target == uniast.Rust:
		// github.com/example/project/model -> model
		return strings.ReplaceAll(path, "/", "::")
	case a.source == uniast.Golang && a.target == uniast.Python:
		// github.com/example/project/model -> model
		return strings.ReplaceAll(path, "/", ".")
	case a.source == uniast.Python && a.target == uniast.Golang:
		// example.model -> model
		parts := strings.Split(path, ".")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return path
	case a.source == uniast.Python && a.target == uniast.Java:
		// example.model -> com.example.model
		return "com." + path
	case a.source == uniast.Python && a.target == uniast.Rust:
		// example.model -> model
		return toSnakeCase(strings.ReplaceAll(path, ".", "::"))
	case a.source == uniast.Rust && a.target == uniast.Golang:
		// crate::module -> module
		return strings.ReplaceAll(path, "::", "/")
	case a.source == uniast.Rust && a.target == uniast.Java:
		// crate::module -> com.crate.module
		return "com." + strings.ReplaceAll(path, "::", ".")
	case a.source == uniast.Rust && a.target == uniast.Python:
		// crate::module -> crate.module
		return strings.ReplaceAll(path, "::", ".")
	default:
		return path
	}
}

// convertFilePath converts a file path to target language convention
func (a *StructureAdapter) convertFilePath(path string) string {
	if path == "" {
		return path
	}

	ext := a.getFileExtension()
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	var newBase string
	switch a.target {
	case uniast.Golang:
		newBase = strings.ToLower(base)
	case uniast.Rust:
		newBase = toSnakeCase(base)
	case uniast.Python:
		newBase = toSnakeCase(base)
	case uniast.Java:
		newBase = toPascalCase(base)
	default:
		newBase = base
	}

	if dir == "." || dir == "" {
		return newBase + ext
	}
	return filepath.Join(dir, newBase+ext)
}

// convertImports converts imports to target language format
func (a *StructureAdapter) convertImports(imports []uniast.Import) []uniast.Import {
	if len(imports) == 0 {
		return imports
	}

	result := make([]uniast.Import, 0, len(imports))
	for _, imp := range imports {
		converted := a.convertImport(imp)
		if converted.Path != "" {
			result = append(result, converted)
		}
	}
	return result
}

// convertImport converts a single import to target language format
func (a *StructureAdapter) convertImport(imp uniast.Import) uniast.Import {
	// For now, just return the import as is
	// In the future, we could map known imports to equivalents
	return imp
}

// getFileExtension returns the file extension for the target language
func (a *StructureAdapter) getFileExtension() string {
	switch a.target {
	case uniast.Golang:
		return ".go"
	case uniast.Rust:
		return ".rs"
	case uniast.Python:
		return ".py"
	case uniast.Java:
		return ".java"
	case uniast.TypeScript:
		return ".ts"
	case uniast.Cxx:
		return ".cpp"
	default:
		return ""
	}
}
