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
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// Transformer defines the interface for AST transformation
type Transformer interface {
	// Transform converts source AST to target AST
	Transform(ctx context.Context, src *uniast.Repository) (*uniast.Repository, error)
}

// BaseTransformer is the base implementation of Transformer
type BaseTransformer struct {
	opts           TranslateOptions
	nodeTranslator *NodeTranslator
	structAdapter  *StructureAdapter
	promptBuilder  *PromptBuilder
}

// NewTransformer creates a new BaseTransformer
func NewTransformer(opts TranslateOptions) *BaseTransformer {
	typeHints := NewTypeHints(opts.SourceLanguage, opts.TargetLanguage)
	return &BaseTransformer{
		opts:           opts,
		nodeTranslator: NewNodeTranslator(opts, typeHints),
		structAdapter:  NewStructureAdapter(opts.SourceLanguage, opts.TargetLanguage),
		promptBuilder:  NewPromptBuilder(opts.SourceLanguage, opts.TargetLanguage, typeHints),
	}
}

// Transform converts source AST to target AST
func (t *BaseTransformer) Transform(ctx context.Context, src *uniast.Repository) (*uniast.Repository, error) {
	// 1. Determine target module name
	targetModName := t.opts.TargetModuleName
	if targetModName == "" {
		// Try to derive from source repository name
		targetModName = t.structAdapter.convertModuleName(src.Name)
	}

	// Sanitize module name (remove invalid characters like colons)
	targetModName = sanitizeModuleName(targetModName)

	// 2. Create target Repository with single module
	targetRepo := t.structAdapter.AdaptRepository(src, targetModName)

	// Create a single target module for Go (merge all source modules)
	// Go projects typically have one go.mod at the root
	// Use "." as Dir to indicate this is a local module (not external)
	targetMod := uniast.NewModule(targetModName, ".", t.opts.TargetLanguage)

	// Optional result for node-granular outcome (failed nodes, success cache)
	if t.opts.Result != nil {
		if t.opts.Result.TranslatedIDs == nil {
			t.opts.Result.TranslatedIDs = make(map[string]struct{})
		}
		t.opts.Result.FailedNodes = nil
	}
	maxRetry := t.opts.MaxRetryPerNode
	if maxRetry < 1 {
		maxRetry = 1
	}

	// Global translate context for tracking all translated nodes
	globalCtx := &TranslateContext{
		SourceRepo:      src,
		TargetRepo:      targetRepo,
		TranslatedNodes: make(map[string]uniast.Identity),
		Result:          t.opts.Result,
	}

	// 3. Traverse all source modules and merge their packages into the single target module
	for _, srcMod := range src.Modules {
		if srcMod.IsExternal() {
			continue
		}

		for pkgPath, srcPkg := range srcMod.Packages {
			// Convert package path for target language (flat structure)
			targetPkgPath := t.structAdapter.convertPackagePath(string(pkgPath))

			// Check if package already exists (from another source module)
			targetPkg, exists := targetMod.Packages[uniast.PkgPath(targetPkgPath)]
			if !exists {
				targetPkg = t.structAdapter.AdaptPackage(srcPkg)
				targetPkg.PkgPath = uniast.PkgPath(targetPkgPath)
			}

			// Create package-level context
			pkgCtx := &TranslateContext{
				SourceRepo:      src,
				TargetRepo:      targetRepo,
				Module:          targetMod,
				Package:         targetPkg,
				TranslatedNodes: globalCtx.TranslatedNodes,
				Result:          globalCtx.Result,
			}

			// Translate in order: Types -> Functions -> Vars (one node = one retry unit; failures recorded, continue)
			t.translateTypes(ctx, srcPkg, targetPkg, pkgCtx, maxRetry)
			t.translateFunctions(ctx, srcPkg, targetPkg, pkgCtx, maxRetry)
			t.translateVars(ctx, srcPkg, targetPkg, pkgCtx, maxRetry)

			targetMod.Packages[uniast.PkgPath(targetPkgPath)] = targetPkg
		}
	}

	// 4. Add the single merged module to the repository
	targetRepo.Modules[targetModName] = targetMod

	// 5. Rebuild dependency graph
	if err := targetRepo.BuildGraph(); err != nil {
		return nil, fmt.Errorf("build graph failed: %w", err)
	}

	// 6. Post-processing: entry points, framework integration, config generation
	postProcessor := NewPostProcessor(t.opts.TargetLanguage, PostProcessOptions{
		GenerateEntryPoint: t.opts.GenerateEntryPoint,
		WebFramework:       t.opts.WebFramework,
		GenerateConfig:     t.opts.GenerateConfig,
		ModuleName:         targetModName,
		OutputDir:          t.opts.OutputDir,
	})

	targetRepo, err := postProcessor.Process(targetRepo)
	if err != nil {
		return nil, fmt.Errorf("post-process failed: %w", err)
	}

	return targetRepo, nil
}

// sanitizeModuleName removes invalid characters from a module name
func sanitizeModuleName(name string) string {
	// Handle filesystem paths - extract just the project name
	if strings.HasPrefix(name, "/") || strings.Contains(name, "/Users/") || strings.Contains(name, "/home/") {
		// Extract the last meaningful directory name (project name)
		parts := strings.Split(name, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			if part != "" && part != "." && part != ".." {
				name = part
				break
			}
		}
	}

	// Remove Maven version suffix (e.g., :1.0.0-SNAPSHOT)
	if idx := strings.LastIndex(name, ":"); idx > 0 {
		// Check if after : is a version number
		suffix := name[idx+1:]
		if len(suffix) > 0 && (suffix[0] >= '0' && suffix[0] <= '9') {
			name = name[:idx]
		}
	}

	// Replace colons with slashes (Maven groupId:artifactId -> path)
	name = strings.ReplaceAll(name, ":", "/")

	// Convert to lowercase for Go
	name = strings.ToLower(name)

	// Replace underscores with hyphens for Go module names
	name = strings.ReplaceAll(name, "_", "-")

	// Ensure valid Go module name
	// Remove leading/trailing slashes
	name = strings.Trim(name, "/")

	// If it doesn't look like a proper module path, add a domain prefix
	if !strings.Contains(name, ".") && !strings.Contains(name, "/") {
		name = "github.com/example/" + name
	}

	return name
}

// translateTypes translates all types in a package. One node = one retry unit; failures are recorded, translation continues.
func (t *BaseTransformer) translateTypes(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		t.translateTypesParallel(ctx, srcPkg, targetPkg, tctx, maxRetry)
		return
	}
	t.translateTypesSequential(ctx, srcPkg, targetPkg, tctx, maxRetry)
}

func (t *BaseTransformer) translateTypesSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	for _, srcType := range srcPkg.Types {
		if tctx.Result != nil && t.opts.AlreadyTranslatedIDs != nil {
			if _, ok := t.opts.AlreadyTranslatedIDs[srcType.Identity.Full()]; ok {
				continue
			}
		}
		var targetType *uniast.Type
		var err error
		for attempt := 0; attempt < maxRetry; attempt++ {
			targetType, err = t.nodeTranslator.TranslateType(ctx, srcType, tctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			if tctx.Result != nil {
				tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
					NodeID: srcType.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
				})
			}
			continue
		}
		targetPkg.Types[targetType.Name] = targetType
		tctx.AddTranslatedNode(srcType.Identity, targetType.Identity)
		if tctx.Result != nil {
			tctx.Result.TranslatedIDs[srcType.Identity.Full()] = struct{}{}
		}
	}
}

func (t *BaseTransformer) translateTypesParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for _, srcType := range srcPkg.Types {
		wg.Add(1)
		go func(srcType *uniast.Type) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var targetType *uniast.Type
			var err error
			for attempt := 0; attempt < maxRetry; attempt++ {
				targetType, err = t.nodeTranslator.TranslateType(ctx, srcType, tctx)
				if err == nil {
					break
				}
			}
			if err != nil {
				if tctx.Result != nil {
					mu.Lock()
					tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
						NodeID: srcType.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
					})
					mu.Unlock()
				}
				return
			}
			mu.Lock()
			targetPkg.Types[targetType.Name] = targetType
			tctx.AddTranslatedNode(srcType.Identity, targetType.Identity)
			if tctx.Result != nil {
				tctx.Result.TranslatedIDs[srcType.Identity.Full()] = struct{}{}
			}
			mu.Unlock()
		}(srcType)
	}
	wg.Wait()
}

// translateFunctions translates all functions in a package. One node = one retry unit; failures recorded, continue.
func (t *BaseTransformer) translateFunctions(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		t.translateFunctionsParallel(ctx, srcPkg, targetPkg, tctx, maxRetry)
		return
	}
	t.translateFunctionsSequential(ctx, srcPkg, targetPkg, tctx, maxRetry)
}

func (t *BaseTransformer) translateFunctionsSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	for _, srcFunc := range srcPkg.Functions {
		if tctx.Result != nil && t.opts.AlreadyTranslatedIDs != nil {
			if _, ok := t.opts.AlreadyTranslatedIDs[srcFunc.Identity.Full()]; ok {
				continue
			}
		}
		var targetFunc *uniast.Function
		var err error
		for attempt := 0; attempt < maxRetry; attempt++ {
			targetFunc, err = t.nodeTranslator.TranslateFunction(ctx, srcFunc, tctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			if tctx.Result != nil {
				tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
					NodeID: srcFunc.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
				})
			}
			continue
		}
		targetPkg.Functions[targetFunc.Name] = targetFunc
		tctx.AddTranslatedNode(srcFunc.Identity, targetFunc.Identity)
		if tctx.Result != nil {
			tctx.Result.TranslatedIDs[srcFunc.Identity.Full()] = struct{}{}
		}
	}
}

func (t *BaseTransformer) translateFunctionsParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for _, srcFunc := range srcPkg.Functions {
		wg.Add(1)
		go func(srcFunc *uniast.Function) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var targetFunc *uniast.Function
			var err error
			for attempt := 0; attempt < maxRetry; attempt++ {
				targetFunc, err = t.nodeTranslator.TranslateFunction(ctx, srcFunc, tctx)
				if err == nil {
					break
				}
			}
			if err != nil {
				if tctx.Result != nil {
					mu.Lock()
					tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
						NodeID: srcFunc.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
					})
					mu.Unlock()
				}
				return
			}
			mu.Lock()
			targetPkg.Functions[targetFunc.Name] = targetFunc
			tctx.AddTranslatedNode(srcFunc.Identity, targetFunc.Identity)
			if tctx.Result != nil {
				tctx.Result.TranslatedIDs[srcFunc.Identity.Full()] = struct{}{}
			}
			mu.Unlock()
		}(srcFunc)
	}
	wg.Wait()
}

// translateVars translates all variables in a package. One node = one retry unit; failures recorded, continue.
func (t *BaseTransformer) translateVars(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		t.translateVarsParallel(ctx, srcPkg, targetPkg, tctx, maxRetry)
		return
	}
	t.translateVarsSequential(ctx, srcPkg, targetPkg, tctx, maxRetry)
}

func (t *BaseTransformer) translateVarsSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	for _, srcVar := range srcPkg.Vars {
		if tctx.Result != nil && t.opts.AlreadyTranslatedIDs != nil {
			if _, ok := t.opts.AlreadyTranslatedIDs[srcVar.Identity.Full()]; ok {
				continue
			}
		}
		var targetVar *uniast.Var
		var err error
		for attempt := 0; attempt < maxRetry; attempt++ {
			targetVar, err = t.nodeTranslator.TranslateVar(ctx, srcVar, tctx)
			if err == nil {
				break
			}
		}
		if err != nil {
			if tctx.Result != nil {
				tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
					NodeID: srcVar.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
				})
			}
			continue
		}
		targetPkg.Vars[targetVar.Name] = targetVar
		tctx.AddTranslatedNode(srcVar.Identity, targetVar.Identity)
		if tctx.Result != nil {
			tctx.Result.TranslatedIDs[srcVar.Identity.Full()] = struct{}{}
		}
	}
}

func (t *BaseTransformer) translateVarsParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext, maxRetry int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for _, srcVar := range srcPkg.Vars {
		wg.Add(1)
		go func(srcVar *uniast.Var) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var targetVar *uniast.Var
			var err error
			for attempt := 0; attempt < maxRetry; attempt++ {
				targetVar, err = t.nodeTranslator.TranslateVar(ctx, srcVar, tctx)
				if err == nil {
					break
				}
			}
			if err != nil {
				if tctx.Result != nil {
					mu.Lock()
					tctx.Result.FailedNodes = append(tctx.Result.FailedNodes, FailedNodeInfo{
						NodeID: srcVar.Identity.Full(), Attempts: maxRetry, Err: err.Error(),
					})
					mu.Unlock()
				}
				return
			}
			mu.Lock()
			targetPkg.Vars[targetVar.Name] = targetVar
			tctx.AddTranslatedNode(srcVar.Identity, targetVar.Identity)
			if tctx.Result != nil {
				tctx.Result.TranslatedIDs[srcVar.Identity.Full()] = struct{}{}
			}
			mu.Unlock()
		}(srcVar)
	}
	wg.Wait()
}
