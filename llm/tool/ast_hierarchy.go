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
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

const (
	ToolGetASTHierarchy       = "get_ast_hierarchy"
	DescGetASTHierarchy       = "get the AST hierarchy (leveled directory) of a repository: Level 0=repo overview, 1=modules, 2=packages with counts, 3=files, 4=node list. Use max_depth to limit depth (0-4, default 4)."
	ToolGetTargetLanguageSpec = "get_target_language_spec"
	DescGetTargetLanguageSpec = "get target language specification summary: type mapping, naming conventions, error handling. Input target_language e.g. go, java, rust. Use before translating to confirm target language habits."
)

var (
	SchemaGetASTHierarchy       = GetJSONSchema(GetASTHierarchyReq{})
	SchemaGetTargetLanguageSpec = GetJSONSchema(GetTargetLanguageSpecReq{})
)

// GetASTHierarchyReq is the request for get_ast_hierarchy.
type GetASTHierarchyReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository"`
	MaxDepth int    `json:"max_depth,omitempty" jsonschema:"description=maximum level to include (0=repo only, 1=+modules, 2=+packages, 3=+files, 4=+nodes; default 4)"`
}

// GetASTHierarchyResp is the response for get_ast_hierarchy.
type GetASTHierarchyResp struct {
	Hierarchy *HierarchyNode `json:"hierarchy,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// HierarchyNode is one node in the AST hierarchy tree.
type HierarchyNode struct {
	Level    int              `json:"level"`
	Kind     string           `json:"kind"`
	Path     string           `json:"path,omitempty"`
	Name     string           `json:"name,omitempty"`
	Counts   *HierarchyCounts `json:"counts,omitempty"`
	Children []*HierarchyNode `json:"children,omitempty"`
}

// HierarchyCounts holds counts at a level.
type HierarchyCounts struct {
	Modules   int `json:"modules,omitempty"`
	Packages  int `json:"packages,omitempty"`
	Files     int `json:"files,omitempty"`
	Types     int `json:"types,omitempty"`
	Functions int `json:"functions,omitempty"`
	Vars      int `json:"vars,omitempty"`
}

// GetTargetLanguageSpecReq is the request for get_target_language_spec.
type GetTargetLanguageSpecReq struct {
	TargetLanguage string `json:"target_language" jsonschema:"description=target language e.g. go, java, rust, python"`
}

// GetTargetLanguageSpecResp is the response for get_target_language_spec.
type GetTargetLanguageSpecResp struct {
	Spec  string `json:"spec"`
	Error string `json:"error,omitempty"`
}

// BuildASTHierarchy builds the hierarchy tree from a repository. maxDepth 0-4; if < 0, treated as 4.
func BuildASTHierarchy(repo *uniast.Repository, maxDepth int) *GetASTHierarchyResp {
	if maxDepth < 0 || maxDepth > 4 {
		maxDepth = 4
	}
	root := &HierarchyNode{
		Level: 0,
		Kind:  "repository",
		Name:  repo.Name,
		Path:  repo.Path,
		Counts: &HierarchyCounts{},
	}
	var modCount int
	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		modCount++
	}
	root.Counts.Modules = modCount
	if maxDepth < 1 {
		return &GetASTHierarchyResp{Hierarchy: root}
	}
	for modPath, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		modNode := &HierarchyNode{
			Level: 1,
			Kind:  "module",
			Path:  string(modPath),
			Name:  mod.Name,
			Counts: &HierarchyCounts{Packages: len(mod.Packages)},
		}
		if maxDepth >= 2 {
			for pkgPath, pkg := range mod.Packages {
				files := mod.GetPkgFiles(pkgPath)
				pkgNode := &HierarchyNode{
					Level: 2,
					Kind:  "package",
					Path:  string(pkgPath),
					Name:  string(pkgPath),
					Counts: &HierarchyCounts{
						Files:     len(files),
						Types:     len(pkg.Types),
						Functions: len(pkg.Functions),
						Vars:      len(pkg.Vars),
					},
				}
				if maxDepth >= 3 {
					for _, f := range files {
						pkgNode.Children = append(pkgNode.Children, &HierarchyNode{
							Level: 3,
							Kind:  "file",
							Path:  f.Path,
							Name:  f.Path,
						})
					}
				}
				modNode.Children = append(modNode.Children, pkgNode)
			}
		}
		root.Children = append(root.Children, modNode)
	}
	return &GetASTHierarchyResp{Hierarchy: root}
}

// GetTargetLanguageSpecContent returns the spec text for the given target language.
func GetTargetLanguageSpecContent(targetLanguage string) *GetTargetLanguageSpecResp {
	lang := uniast.NewLanguage(strings.TrimSpace(targetLanguage))
	switch lang {
	case uniast.Golang:
		return &GetTargetLanguageSpecResp{Spec: goLanguageSpec}
	case uniast.Java:
		return &GetTargetLanguageSpecResp{Spec: javaLanguageSpec}
	case uniast.Rust:
		return &GetTargetLanguageSpecResp{Spec: rustLanguageSpec}
	case uniast.Python:
		return &GetTargetLanguageSpecResp{Spec: pythonLanguageSpec}
	default:
		return &GetTargetLanguageSpecResp{
			Spec:  "Target language not found. Supported: go, java, rust, python.",
			Error: "unsupported target_language: " + targetLanguage,
		}
	}
}

const goLanguageSpec = `# Go Language Spec (target)

## Type Mapping (from Java)
- Primitives: int→int, long→int64, String→string, boolean→bool, float→float32, double→float64, void→(none)
- Collections: List<T>→[]T, Map<K,V>→map[K]V, Set<T>→[]T or map[T]bool
- Optional<T>→(T, bool)
- Objects: Object→interface{}, custom classes→structs/interfaces

## Naming
- Types: PascalCase (exported)
- Functions: PascalCase (exported) or camelCase (unexported)
- Fields: PascalCase/camelCase
- Constants: UPPER_SNAKE_CASE

## Error Handling
- Prefer (result, error) return; use "if err != nil { return err }"; avoid panic for control flow.

## Structure
- One package per directory; package name = last path segment
- Module path in go.mod (e.g. github.com/example/project)
`

const javaLanguageSpec = `# Java Language Spec (target)

## Type Mapping (from Go)
- int→int/Integer, int64→long/Long, string→String, bool→boolean/Boolean
- []T→List<T>, map[K]V→Map<K,V>
- (T, bool)→Optional<T>
- interface{}→Object

## Naming
- Classes: PascalCase
- Methods/fields: camelCase
- Constants: UPPER_SNAKE_CASE

## Error Handling
- Exceptions: throws, try-catch; Optional for nullable returns.
`

const rustLanguageSpec = `# Rust Language Spec (target)

## Type Mapping
- Primitives: int→i32, int64→i64, string→String/&str, bool→bool
- Collections: List→Vec<T>, Map→HashMap<K,V>
- Optional→Option<T>; errors→Result<T, E>

## Naming
- Types: PascalCase
- Functions/variables: snake_case
- Constants: SCREAMING_SNAKE_CASE

## Error Handling
- Result<T, E>; ? operator; no exceptions.
`

const pythonLanguageSpec = `# Python Language Spec (target)

## Type Mapping
- Primitives: int→int, int64→int, string→str, bool→bool
- Collections: list, dict, set
- Optional→Optional[T] (typing) or None

## Naming
- Classes: PascalCase
- Functions/variables: snake_case
- Constants: UPPER_SNAKE_CASE

## Error Handling
- Exceptions: raise, try/except.
`
