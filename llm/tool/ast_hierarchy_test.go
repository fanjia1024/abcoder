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
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestBuildASTHierarchy(t *testing.T) {
	// empty repo: NewRepository, no modules
	emptyRepo := uniast.NewRepository("r")

	// one internal module, one package with types/funcs/vars; module has a file for that package
	oneModOnePkgRepo := uniast.NewRepository("r")
	mod := uniast.NewModule("m", ".", uniast.Golang)
	mod.Packages["pkg/a"] = uniast.NewPackage("pkg/a")
	pkg := mod.Packages["pkg/a"]
	pkg.Types["T"] = &uniast.Type{}
	pkg.Functions["f"] = &uniast.Function{}
	pkg.Vars["v"] = &uniast.Var{}
	f := uniast.NewFile("path/to/foo.go")
	f.Package = "pkg/a"
	if mod.Files == nil {
		mod.Files = map[string]*uniast.File{}
	}
	mod.Files["path/to/foo.go"] = f
	oneModOnePkgRepo.Modules["m"] = mod

	// external module (Dir "") + one internal module
	externalPlusInternalRepo := uniast.NewRepository("r")
	externalPlusInternalRepo.Modules["ext"] = uniast.NewModule("ext", "", uniast.Golang)
	internalMod := uniast.NewModule("internal", ".", uniast.Golang)
	internalMod.Packages["p"] = uniast.NewPackage("p")
	externalPlusInternalRepo.Modules["internal"] = internalMod

	tests := []struct {
		name     string
		repo     *uniast.Repository
		maxDepth int
		assert   func(t *testing.T, got *GetASTHierarchyResp)
	}{
		{
			name:     "empty_repo",
			repo:     &emptyRepo,
			maxDepth: 0,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil {
					t.Fatal("Hierarchy must be non-nil")
				}
				if got.Hierarchy.Level != 0 || got.Hierarchy.Kind != "repository" {
					t.Errorf("Level=0 Kind=repository, got Level=%d Kind=%s", got.Hierarchy.Level, got.Hierarchy.Kind)
				}
				if got.Hierarchy.Counts == nil || got.Hierarchy.Counts.Modules != 0 {
					t.Errorf("Counts.Modules=0, got %v", got.Hierarchy.Counts)
				}
				if len(got.Hierarchy.Children) != 0 {
					t.Errorf("Children should be empty, got len=%d", len(got.Hierarchy.Children))
				}
			},
		},
		{
			name:     "empty_repo_depth2",
			repo:     &emptyRepo,
			maxDepth: 2,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil {
					t.Fatal("Hierarchy must be non-nil")
				}
				if got.Hierarchy.Level != 0 || got.Hierarchy.Kind != "repository" {
					t.Errorf("Level=0 Kind=repository, got Level=%d Kind=%s", got.Hierarchy.Level, got.Hierarchy.Kind)
				}
				if len(got.Hierarchy.Children) != 0 {
					t.Errorf("Children should be empty, got len=%d", len(got.Hierarchy.Children))
				}
			},
		},
		{
			name:     "one_module_one_package",
			repo:     &oneModOnePkgRepo,
			maxDepth: 0,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil {
					t.Fatal("Hierarchy must be non-nil")
				}
				if got.Hierarchy.Counts == nil || got.Hierarchy.Counts.Modules != 1 {
					t.Errorf("Counts.Modules=1, got %v", got.Hierarchy.Counts)
				}
				if len(got.Hierarchy.Children) != 0 {
					t.Errorf("maxDepth 0: Children should be empty, got len=%d", len(got.Hierarchy.Children))
				}
			},
		},
		{
			name:     "one_module_one_package_depth1",
			repo:     &oneModOnePkgRepo,
			maxDepth: 1,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil || len(got.Hierarchy.Children) != 1 {
					t.Fatalf("root.Children len=1, got %d", len(got.Hierarchy.Children))
				}
				c := got.Hierarchy.Children[0]
				if c.Level != 1 || c.Kind != "module" {
					t.Errorf("Level=1 Kind=module, got Level=%d Kind=%s", c.Level, c.Kind)
				}
				if c.Counts == nil || c.Counts.Packages != 1 {
					t.Errorf("module Counts.Packages=1, got %v", c.Counts)
				}
			},
		},
		{
			name:     "one_module_one_package_depth2",
			repo:     &oneModOnePkgRepo,
			maxDepth: 2,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil || len(got.Hierarchy.Children) != 1 {
					t.Fatalf("root.Children len=1, got %d", len(got.Hierarchy.Children))
				}
				modNode := got.Hierarchy.Children[0]
				if len(modNode.Children) != 1 {
					t.Fatalf("module.Children len=1, got %d", len(modNode.Children))
				}
				pkgNode := modNode.Children[0]
				if pkgNode.Level != 2 || pkgNode.Kind != "package" {
					t.Errorf("Level=2 Kind=package, got Level=%d Kind=%s", pkgNode.Level, pkgNode.Kind)
				}
				if pkgNode.Counts == nil || pkgNode.Counts.Types != 1 || pkgNode.Counts.Functions != 1 || pkgNode.Counts.Vars != 1 {
					t.Errorf("package Counts Types=1 Functions=1 Vars=1, got %v", pkgNode.Counts)
				}
				if pkgNode.Counts.Files != 1 {
					t.Errorf("package Counts.Files=1, got %d", pkgNode.Counts.Files)
				}
			},
		},
		{
			name:     "one_module_one_package_depth3",
			repo:     &oneModOnePkgRepo,
			maxDepth: 3,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				modNode := got.Hierarchy.Children[0]
				pkgNode := modNode.Children[0]
				if len(pkgNode.Children) != 1 {
					t.Fatalf("package.Children len=1 (file), got %d", len(pkgNode.Children))
				}
				fileNode := pkgNode.Children[0]
				if fileNode.Level != 3 || fileNode.Kind != "file" {
					t.Errorf("Level=3 Kind=file, got Level=%d Kind=%s", fileNode.Level, fileNode.Kind)
				}
				if fileNode.Path != "path/to/foo.go" {
					t.Errorf("file Path=path/to/foo.go, got %s", fileNode.Path)
				}
			},
		},
		{
			name:     "external_module_skipped",
			repo:     &externalPlusInternalRepo,
			maxDepth: 1,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy.Counts == nil || got.Hierarchy.Counts.Modules != 1 {
					t.Errorf("Counts.Modules=1 (only internal), got %v", got.Hierarchy.Counts)
				}
				if len(got.Hierarchy.Children) != 1 {
					t.Errorf("root.Children len=1 (only internal module), got %d", len(got.Hierarchy.Children))
				}
				if got.Hierarchy.Children[0].Name != "internal" {
					t.Errorf("only internal module expected, got %s", got.Hierarchy.Children[0].Name)
				}
			},
		},
		{
			name:     "invalid_maxDepth_negative",
			repo:     &oneModOnePkgRepo,
			maxDepth: -1,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil {
					t.Fatal("Hierarchy must be non-nil")
				}
				if len(got.Hierarchy.Children) == 0 {
					t.Error("maxDepth -1 treated as 4: should have module children")
				}
			},
		},
		{
			name:     "invalid_maxDepth_over4",
			repo:     &oneModOnePkgRepo,
			maxDepth: 5,
			assert: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Hierarchy == nil {
					t.Fatal("Hierarchy must be non-nil")
				}
				if len(got.Hierarchy.Children) == 0 {
					t.Error("maxDepth 5 treated as 4: should have module children")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildASTHierarchy(tt.repo, tt.maxDepth)
			tt.assert(t, got)
		})
	}
}

func TestGetTargetLanguageSpecContent(t *testing.T) {
	tests := []struct {
		name           string
		targetLanguage string
		assert         func(t *testing.T, got *GetTargetLanguageSpecResp)
	}{
		{
			name:           "supported_go",
			targetLanguage: "go",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if !strings.Contains(strings.ToLower(got.Spec), "go") {
					t.Errorf("Spec should contain go, got %s", got.Spec[:min(80, len(got.Spec))])
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "supported_java",
			targetLanguage: "java",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "supported_rust",
			targetLanguage: "rust",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "supported_python",
			targetLanguage: "python",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "normalized_go",
			targetLanguage: "Go",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "normalized_golang",
			targetLanguage: "golang",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("Error must be empty, got %s", got.Error)
				}
			},
		},
		{
			name:           "unknown_language",
			targetLanguage: "xyz",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Error == "" {
					t.Error("Error must be non-empty for unknown language")
				}
				if !strings.Contains(got.Spec, "not found") && !strings.Contains(got.Spec, "Supported") {
					t.Errorf("Spec should contain 'not found' or 'Supported', got %s", got.Spec)
				}
			},
		},
		{
			name:           "unknown_unknown",
			targetLanguage: "unknown",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Error == "" {
					t.Error("Error must be non-empty for unknown language")
				}
			},
		},
		{
			name:           "empty_input",
			targetLanguage: "",
			assert: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Error == "" {
					t.Error("Error must be non-empty for empty input")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTargetLanguageSpecContent(tt.targetLanguage)
			tt.assert(t, got)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
