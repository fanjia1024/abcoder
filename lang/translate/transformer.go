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

	// Global translate context for tracking all translated nodes
	globalCtx := &TranslateContext{
		SourceRepo:      src,
		TargetRepo:      targetRepo,
		TranslatedNodes: make(map[string]uniast.Identity),
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
			}

			// Translate in order: Types -> Functions -> Vars
			// This ensures dependencies are translated first

			// Translate types
			if err := t.translateTypes(ctx, srcPkg, targetPkg, pkgCtx); err != nil {
				return nil, err
			}

			// Translate functions
			if err := t.translateFunctions(ctx, srcPkg, targetPkg, pkgCtx); err != nil {
				return nil, err
			}

			// Translate variables
			if err := t.translateVars(ctx, srcPkg, targetPkg, pkgCtx); err != nil {
				return nil, err
			}

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

// translateTypes translates all types in a package
func (t *BaseTransformer) translateTypes(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		return t.translateTypesParallel(ctx, srcPkg, targetPkg, tctx)
	}
	return t.translateTypesSequential(ctx, srcPkg, targetPkg, tctx)
}

func (t *BaseTransformer) translateTypesSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	for name, srcType := range srcPkg.Types {
		targetType, err := t.nodeTranslator.TranslateType(ctx, srcType, tctx)
		if err != nil {
			return fmt.Errorf("translate type %s failed: %w", name, err)
		}
		targetPkg.Types[targetType.Name] = targetType

		// Record translated node
		tctx.AddTranslatedNode(srcType.Identity, targetType.Identity)
	}
	return nil
}

func (t *BaseTransformer) translateTypesParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(srcPkg.Types))
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for name, srcType := range srcPkg.Types {
		wg.Add(1)
		go func(name string, srcType *uniast.Type) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			targetType, err := t.nodeTranslator.TranslateType(ctx, srcType, tctx)
			if err != nil {
				errCh <- fmt.Errorf("translate type %s failed: %w", name, err)
				return
			}

			mu.Lock()
			targetPkg.Types[targetType.Name] = targetType
			tctx.AddTranslatedNode(srcType.Identity, targetType.Identity)
			mu.Unlock()
		}(name, srcType)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// translateFunctions translates all functions in a package
func (t *BaseTransformer) translateFunctions(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		return t.translateFunctionsParallel(ctx, srcPkg, targetPkg, tctx)
	}
	return t.translateFunctionsSequential(ctx, srcPkg, targetPkg, tctx)
}

func (t *BaseTransformer) translateFunctionsSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	for name, srcFunc := range srcPkg.Functions {
		targetFunc, err := t.nodeTranslator.TranslateFunction(ctx, srcFunc, tctx)
		if err != nil {
			return fmt.Errorf("translate function %s failed: %w", name, err)
		}
		targetPkg.Functions[targetFunc.Name] = targetFunc

		// Record translated node
		tctx.AddTranslatedNode(srcFunc.Identity, targetFunc.Identity)
	}
	return nil
}

func (t *BaseTransformer) translateFunctionsParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(srcPkg.Functions))
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for name, srcFunc := range srcPkg.Functions {
		wg.Add(1)
		go func(name string, srcFunc *uniast.Function) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			targetFunc, err := t.nodeTranslator.TranslateFunction(ctx, srcFunc, tctx)
			if err != nil {
				errCh <- fmt.Errorf("translate function %s failed: %w", name, err)
				return
			}

			mu.Lock()
			targetPkg.Functions[targetFunc.Name] = targetFunc
			tctx.AddTranslatedNode(srcFunc.Identity, targetFunc.Identity)
			mu.Unlock()
		}(name, srcFunc)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// translateVars translates all variables in a package
func (t *BaseTransformer) translateVars(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	if t.opts.Parallel && t.opts.Concurrency > 1 {
		return t.translateVarsParallel(ctx, srcPkg, targetPkg, tctx)
	}
	return t.translateVarsSequential(ctx, srcPkg, targetPkg, tctx)
}

func (t *BaseTransformer) translateVarsSequential(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	for name, srcVar := range srcPkg.Vars {
		targetVar, err := t.nodeTranslator.TranslateVar(ctx, srcVar, tctx)
		if err != nil {
			return fmt.Errorf("translate var %s failed: %w", name, err)
		}
		targetPkg.Vars[targetVar.Name] = targetVar

		// Record translated node
		tctx.AddTranslatedNode(srcVar.Identity, targetVar.Identity)
	}
	return nil
}

func (t *BaseTransformer) translateVarsParallel(ctx context.Context, srcPkg, targetPkg *uniast.Package, tctx *TranslateContext) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(srcPkg.Vars))
	semaphore := make(chan struct{}, t.opts.Concurrency)

	for name, srcVar := range srcPkg.Vars {
		wg.Add(1)
		go func(name string, srcVar *uniast.Var) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			targetVar, err := t.nodeTranslator.TranslateVar(ctx, srcVar, tctx)
			if err != nil {
				errCh <- fmt.Errorf("translate var %s failed: %w", name, err)
				return
			}

			mu.Lock()
			targetPkg.Vars[targetVar.Name] = targetVar
			tctx.AddTranslatedNode(srcVar.Identity, targetVar.Identity)
			mu.Unlock()
		}(name, srcVar)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}
