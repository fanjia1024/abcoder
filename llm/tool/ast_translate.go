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
	"context"
	"fmt"
	"strings"
	"sync"

	abutil "github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/lang/translate"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	ToolTranslateNode    = "translate_node"
	DescTranslateNode    = "translate a Java AST node to Go code. Input: repo_name, node_id. Output: Go code string"
	ToolTranslateType    = "translate_type"
	DescTranslateType    = "translate a Java type to Go type. Input: java_type. Output: Go type string"
	ToolTranslateFile    = "translate_file"
	DescTranslateFile    = "translate an entire Java file to Go code. Input: repo_name, file_path. Output: Go code string"
	ToolTranslateRepo    = "translate_repo"
	DescTranslateRepo    = "translate an entire Java UniAST repository to Go UniAST repository. Input: repo_name, go_module_name. Output: Go UniAST JSON"
	ToolTranslatePackage = "translate_package"
	DescTranslatePackage = "translate a single Java package to Go package. Input: repo_name, module_name, package_path, go_module_name. Output: Go UniAST Package JSON"
)

var (
	SchemaTranslateNode    = GetJSONSchema(TranslateNodeReq{})
	SchemaTranslateType    = GetJSONSchema(TranslateTypeReq{})
	SchemaTranslateFile    = GetJSONSchema(TranslateFileReq{})
	SchemaTranslateRepo    = GetJSONSchema(TranslateRepoReq{})
	SchemaTranslatePackage = GetJSONSchema(TranslatePackageReq{})
)

type ASTTranslateToolsOptions struct {
	RepoASTsDir string
}

type ASTTranslateTools struct {
	opts  ASTTranslateToolsOptions
	repos sync.Map
	tools map[string]tool.InvokableTool
}

func NewASTTranslateTools(opts ASTTranslateToolsOptions) *ASTTranslateTools {
	ret := &ASTTranslateTools{
		opts:  opts,
		tools: map[string]tool.InvokableTool{},
	}

	// load all *.json repos from RepoASTsDir (lenient: log and skip failed files)
	if err := LoadReposIntoMap(opts.RepoASTsDir, &ret.repos, func(file string, err error) {
		log.Error("Load Uniast JSON file failed: %v", err)
	}); err != nil {
		panic(err)
	}

	// Register tools
	tt, err := utils.InferTool(ToolTranslateType,
		DescTranslateType,
		ret.TranslateType, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTranslateType] = tt

	tt, err = utils.InferTool(ToolTranslateNode,
		DescTranslateNode,
		ret.TranslateNode, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTranslateNode] = tt

	tt, err = utils.InferTool(ToolTranslateFile,
		DescTranslateFile,
		ret.TranslateFile, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTranslateFile] = tt

	tt, err = utils.InferTool(ToolTranslateRepo,
		DescTranslateRepo,
		ret.TranslateRepo, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTranslateRepo] = tt

	tt, err = utils.InferTool(ToolTranslatePackage,
		DescTranslatePackage,
		ret.TranslatePackage, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTranslatePackage] = tt

	return ret
}

func (a *ASTTranslateTools) GetTools() []tool.InvokableTool {
	var ret []tool.InvokableTool
	for _, t := range a.tools {
		ret = append(ret, t)
	}
	return ret
}

type TranslateTypeReq struct {
	JavaType string `json:"java_type"`
}

type TranslateTypeResp struct {
	GoType string `json:"go_type"`
}

func (a *ASTTranslateTools) TranslateType(ctx context.Context, req TranslateTypeReq) (*TranslateTypeResp, error) {
	goType := translate.GetGoType(req.JavaType)
	return &TranslateTypeResp{
		GoType: goType,
	}, nil
}

type TranslateNodeReq struct {
	RepoName string `json:"repo_name"`
	NodeID   string `json:"node_id"`
}

type TranslateNodeResp struct {
	GoCode string `json:"go_code"`
	Note   string `json:"note,omitempty"`
}

// normalizeNodeID normalizes node ID formats to the standard format: modPath?pkgPath#name
// Supports multiple input formats:
//   - Standard: modPath?pkgPath#name
//   - LLM format: modPath/pkgPath/name or modPath/pkgPath/name.method
func normalizeNodeID(nodeID string) string {
	// If already in correct format, return as-is
	if strings.Contains(nodeID, "?") && strings.Contains(nodeID, "#") {
		return nodeID
	}

	// Try to convert from / separator format to ? and # format
	// Example: com.example.test:core-module:1.0.0-SNAPSHOT/com.example.core.service/UserService.deleteUser(Long)
	// Should become: com.example.test:core-module:1.0.0-SNAPSHOT?com.example.core.service#UserService.deleteUser(Long)
	if strings.Contains(nodeID, "/") {
		parts := strings.Split(nodeID, "/")
		if len(parts) >= 2 {
			modPath := parts[0]
			// The rest should be pkgPath/name or just name
			if len(parts) == 2 {
				// Format: modPath/pkgPath#name or modPath/name
				// Check if it contains a dot (likely a method name)
				if strings.Contains(parts[1], ".") {
					// Assume it's pkgPath.name format
					lastDot := strings.LastIndex(parts[1], ".")
					if lastDot > 0 {
						pkgPath := parts[1][:lastDot]
						name := parts[1][lastDot+1:]
						return modPath + "?" + pkgPath + "#" + name
					}
				}
				// Simple case: modPath/name
				return modPath + "?#" + parts[1]
			} else if len(parts) >= 3 {
				// Format: modPath/pkgPath/name
				pkgPath := parts[1]
				name := strings.Join(parts[2:], ".")
				return modPath + "?" + pkgPath + "#" + name
			}
		}
	}

	// If we can't parse it, try using uniast.NewIdentityFromString as fallback
	// but first log a warning
	log.Debug("Could not normalize node ID format: %s, using as-is", nodeID)
	return nodeID
}

func (a *ASTTranslateTools) TranslateNode(ctx context.Context, req TranslateNodeReq) (*TranslateNodeResp, error) {
	// Get repo - try to find by name, or use the first available repo if name doesn't match
	var repo *uniast.Repository
	repoVal, ok := a.repos.Load(req.RepoName)
	if !ok {
		// Try to find any repo (fallback for name mismatch)
		a.repos.Range(func(key, value interface{}) bool {
			repo = value.(*uniast.Repository)
			return false // stop iteration
		})
		if repo == nil {
			return nil, fmt.Errorf("repository not found: %s", req.RepoName)
		}
		log.Info("Repository name mismatch: requested %s, using %s", req.RepoName, repo.Name)
	} else {
		repo = repoVal.(*uniast.Repository)
	}

	// Ensure Graph is built
	if len(repo.Graph) == 0 {
		if err := repo.BuildGraph(); err != nil {
			log.Error("Failed to build graph: %v", err)
			return nil, fmt.Errorf("failed to build graph: %v", err)
		}
	}

	// Normalize node ID to handle various formats from LLM
	normalizedID := normalizeNodeID(req.NodeID)
	if normalizedID != req.NodeID {
		log.Debug("Normalized node ID: %s -> %s", req.NodeID, normalizedID)
	}

	// Parse node ID using the correct format: {ModPath}?{PkgPath}#{Name}
	id := uniast.NewIdentityFromString(normalizedID)
	log.Debug("Looking for node: %s (parsed: ModPath=%s, PkgPath=%s, Name=%s, Full=%s)",
		normalizedID, id.ModPath, id.PkgPath, id.Name, id.Full())
	log.Debug("Graph size: %d", len(repo.Graph))

	// This is a helper tool - actual translation will be done by LLM
	// This tool provides the Java code context for LLM to translate
	node := repo.GetNode(id)

	if node == nil {
		// Try to find similar nodes for debugging
		var similarKeys []string
		for key := range repo.Graph {
			if len(similarKeys) < 5 {
				similarKeys = append(similarKeys, key)
			}
		}
		log.Error("Node not found: %s (normalized: %s). Graph has %d nodes. Sample keys: %v", req.NodeID, normalizedID, len(repo.Graph), similarKeys)
		return nil, fmt.Errorf("node not found: %s", req.NodeID)
	}

	// Return Java code for LLM to translate
	// The actual translation will be done by the LLM based on the prompt
	return &TranslateNodeResp{
		GoCode: "", // LLM will generate this
		Note:   fmt.Sprintf("Node found: %s, type: %s. Please translate the Java code to Go using the translation rules.", req.NodeID, node.Type),
	}, nil
}

type TranslateFileReq struct {
	RepoName string `json:"repo_name"`
	FilePath string `json:"file_path"`
}

type TranslateFileResp struct {
	GoCode string `json:"go_code"`
	Note   string `json:"note,omitempty"`
}

func (a *ASTTranslateTools) TranslateFile(ctx context.Context, req TranslateFileReq) (*TranslateFileResp, error) {
	// Get repo
	repoVal, ok := a.repos.Load(req.RepoName)
	if !ok {
		return nil, fmt.Errorf("repository not found: %s", req.RepoName)
	}
	repo := repoVal.(*uniast.Repository)

	// Get file nodes
	nodes := repo.GetFileNodes(req.FilePath)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("file not found or empty: %s", req.FilePath)
	}

	// This is a helper tool - actual translation will be done by LLM
	// Return note for LLM to translate
	return &TranslateFileResp{
		GoCode: "", // LLM will generate this
		Note:   fmt.Sprintf("File found: %s with %d nodes. Please translate all Java code in this file to Go using the translation rules.", req.FilePath, len(nodes)),
	}, nil
}

type TranslateRepoReq struct {
	RepoName     string `json:"repo_name" jsonschema:"description=The name of the Java repository to translate"`
	GoModuleName string `json:"go_module_name" jsonschema:"description=The desired Go module name for the output"`
}

type TranslateRepoResp struct {
	GoUniASTJSON string `json:"go_uniast_json" jsonschema:"description=The translated Go UniAST repository in JSON format"`
	Note         string `json:"note,omitempty" jsonschema:"description=Additional information or context"`
	Error        string `json:"error,omitempty" jsonschema:"description=Error message if translation failed"`
}

func (a *ASTTranslateTools) TranslateRepo(ctx context.Context, req TranslateRepoReq) (*TranslateRepoResp, error) {
	// Get Java repo
	repoVal, ok := a.repos.Load(req.RepoName)
	if !ok {
		return &TranslateRepoResp{
			Error: fmt.Sprintf("repository not found: %s", req.RepoName),
		}, nil
	}
	javaRepo := repoVal.(*uniast.Repository)

	// Count modules and packages for info
	var moduleCount, packageCount int
	for _, mod := range javaRepo.Modules {
		if !mod.IsExternal() {
			moduleCount++
			packageCount += len(mod.Packages)
		}
	}

	// Instead of returning the entire UniAST, return a summary and ask LLM to use translate_package for each
	return &TranslateRepoResp{
		GoUniASTJSON: "", // LLM should use translate_package for each package
		Note: fmt.Sprintf(`Java UniAST repository found: %s
Modules: %d, Packages: %d

This repository is too large to translate at once. Please use the translate_package tool to translate each package individually.

Repository structure:
%s`, req.RepoName, moduleCount, packageCount, getRepoStructureSummary(javaRepo)),
	}, nil
}

// getRepoStructureSummary returns a compact summary of the repository structure
func getRepoStructureSummary(repo *uniast.Repository) string {
	var result string
	for modName, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		result += fmt.Sprintf("- Module: %s\n", modName)
		for pkgPath := range mod.Packages {
			result += fmt.Sprintf("  - Package: %s\n", pkgPath)
		}
	}
	return result
}

type TranslatePackageReq struct {
	RepoName     string `json:"repo_name" jsonschema:"description=The name of the repository"`
	ModuleName   string `json:"module_name" jsonschema:"description=The name of the module containing the package"`
	PackagePath  string `json:"package_path" jsonschema:"description=The package path to translate"`
	GoModuleName string `json:"go_module_name" jsonschema:"description=The desired Go module name for the output"`
}

type TranslatePackageResp struct {
	GoPackageJSON string `json:"go_package_json" jsonschema:"description=The translated Go package in UniAST JSON format"`
	GoModuleName  string `json:"go_module_name" jsonschema:"description=The Go module name"`
	GoPackagePath string `json:"go_package_path" jsonschema:"description=The Go package path"`
	Note          string `json:"note,omitempty" jsonschema:"description=Additional information"`
	Error         string `json:"error,omitempty" jsonschema:"description=Error message if translation failed"`
}

func (a *ASTTranslateTools) TranslatePackage(ctx context.Context, req TranslatePackageReq) (*TranslatePackageResp, error) {
	// Get Java repo
	var repo *uniast.Repository
	repoVal, ok := a.repos.Load(req.RepoName)
	if !ok {
		// Try to find any repo (fallback for name mismatch)
		a.repos.Range(func(key, value interface{}) bool {
			repo = value.(*uniast.Repository)
			return false // stop iteration
		})
		if repo == nil {
			return &TranslatePackageResp{
				Error: fmt.Sprintf("repository not found: %s", req.RepoName),
			}, nil
		}
		log.Info("Repository name mismatch: requested %s, using %s", req.RepoName, repo.Name)
	} else {
		repo = repoVal.(*uniast.Repository)
	}

	// Get the module
	mod := repo.Modules[req.ModuleName]
	if mod == nil {
		// Try to find the module by iterating
		for modName, m := range repo.Modules {
			if !m.IsExternal() {
				mod = m
				req.ModuleName = modName
				break
			}
		}
		if mod == nil {
			return &TranslatePackageResp{
				Error: fmt.Sprintf("module not found: %s", req.ModuleName),
			}, nil
		}
	}

	// Get the package
	pkg := mod.Packages[req.PackagePath]
	if pkg == nil {
		return &TranslatePackageResp{
			Error: fmt.Sprintf("package not found: %s", req.PackagePath),
		}, nil
	}

	// Build a minimal repository containing only this package for context
	miniRepo := &uniast.Repository{
		Name:    repo.Name,
		Path:    repo.Path,
		Modules: map[string]*uniast.Module{},
		Graph:   uniast.NodeGraph{},
	}

	// Copy module with only this package
	miniMod := &uniast.Module{
		Language: mod.Language,
		Version:  mod.Version,
		Name:     mod.Name,
		Dir:      mod.Dir,
		Packages: map[uniast.PkgPath]*uniast.Package{
			req.PackagePath: pkg,
		},
		Files: map[string]*uniast.File{},
	}

	// Copy relevant files
	for path, file := range mod.Files {
		if file.Package == req.PackagePath {
			miniMod.Files[path] = file
		}
	}

	miniRepo.Modules[req.ModuleName] = miniMod

	// Copy relevant nodes to graph
	for id, node := range repo.Graph {
		identity := uniast.NewIdentityFromString(id)
		if identity.ModPath == req.ModuleName && identity.PkgPath == req.PackagePath {
			miniRepo.Graph[id] = node
		}
	}

	// Marshal the mini repo
	pkgJSON, err := abutil.MarshalJSONIndent(miniRepo)
	if err != nil {
		return &TranslatePackageResp{
			Error: fmt.Sprintf("failed to marshal package: %v", err),
		}, nil
	}

	// Compute Go package path
	goPackagePath := translate.ConvertJavaPackageToGoModule(req.PackagePath)

	return &TranslatePackageResp{
		GoPackageJSON: "", // LLM will generate this
		GoModuleName:  req.GoModuleName,
		GoPackagePath: goPackagePath,
		Note: fmt.Sprintf(`Java package found: %s
Module: %s
Functions: %d, Types: %d, Vars: %d

Please translate this Java package to Go UniAST format. The Java package UniAST JSON:

%s`, req.PackagePath, req.ModuleName, len(pkg.Functions), len(pkg.Types), len(pkg.Vars), pkgJSON),
	}, nil
}
