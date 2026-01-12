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
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// NodeTranslator handles translation of individual AST nodes
type NodeTranslator struct {
	opts          TranslateOptions
	promptBuilder *PromptBuilder
	typeHints     *TypeHints
}

// NewNodeTranslator creates a new NodeTranslator
func NewNodeTranslator(opts TranslateOptions, typeHints *TypeHints) *NodeTranslator {
	return &NodeTranslator{
		opts:          opts,
		promptBuilder: NewPromptBuilder(opts.SourceLanguage, opts.TargetLanguage, typeHints),
		typeHints:     typeHints,
	}
}

// TranslateType translates a Type node
func (t *NodeTranslator) TranslateType(ctx context.Context, src *uniast.Type, tctx *TranslateContext) (*uniast.Type, error) {
	// 1. Build LLM request
	req := &LLMTranslateRequest{
		SourceLanguage: t.opts.SourceLanguage,
		TargetLanguage: t.opts.TargetLanguage,
		NodeType:       uniast.TYPE,
		SourceContent:  src.Content,
		Identity:       src.Identity,
		TypeHints:      t.typeHints,
		Dependencies:   t.collectDependencyHints(src.Identity, tctx),
	}
	req.Prompt = t.promptBuilder.BuildTypePrompt(req)

	// 2. Call LLM
	resp, err := t.opts.LLMTranslator(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("LLM error: %s", resp.Error)
	}

	// 3. Build target Type
	targetName := t.convertTypeName(src.Name, src.Exported)
	targetType := &uniast.Type{
		Exported: src.Exported,
		TypeKind: src.TypeKind,
		Identity: uniast.Identity{
			ModPath: tctx.Module.Name,
			PkgPath: string(tctx.Package.PkgPath),
			Name:    targetName,
		},
		FileLine: uniast.FileLine{
			File: t.convertFilePath(src.File),
			Line: src.Line,
		},
		Content: resp.TargetContent,
	}

	return targetType, nil
}

// TranslateFunction translates a Function node
func (t *NodeTranslator) TranslateFunction(ctx context.Context, src *uniast.Function, tctx *TranslateContext) (*uniast.Function, error) {
	// 1. Build LLM request
	req := &LLMTranslateRequest{
		SourceLanguage: t.opts.SourceLanguage,
		TargetLanguage: t.opts.TargetLanguage,
		NodeType:       uniast.FUNC,
		SourceContent:  src.Content,
		Identity:       src.Identity,
		TypeHints:      t.typeHints,
		Dependencies:   t.collectDependencyHints(src.Identity, tctx),
	}
	req.Prompt = t.promptBuilder.BuildFunctionPrompt(req)

	// 2. Call LLM
	resp, err := t.opts.LLMTranslator(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("LLM error: %s", resp.Error)
	}

	// 3. Build target Function
	targetName := t.convertFunctionName(src.Name, src.Exported)
	targetFunc := &uniast.Function{
		Exported:          src.Exported,
		IsMethod:          src.IsMethod,
		IsInterfaceMethod: src.IsInterfaceMethod,
		Identity: uniast.Identity{
			ModPath: tctx.Module.Name,
			PkgPath: string(tctx.Package.PkgPath),
			Name:    targetName,
		},
		FileLine: uniast.FileLine{
			File: t.convertFilePath(src.File),
			Line: src.Line,
		},
		Content:   resp.TargetContent,
		Signature: resp.TargetSignature,
	}

	return targetFunc, nil
}

// TranslateVar translates a Var node
func (t *NodeTranslator) TranslateVar(ctx context.Context, src *uniast.Var, tctx *TranslateContext) (*uniast.Var, error) {
	// 1. Build LLM request
	req := &LLMTranslateRequest{
		SourceLanguage: t.opts.SourceLanguage,
		TargetLanguage: t.opts.TargetLanguage,
		NodeType:       uniast.VAR,
		SourceContent:  src.Content,
		Identity:       src.Identity,
		TypeHints:      t.typeHints,
		Dependencies:   t.collectDependencyHints(src.Identity, tctx),
	}
	req.Prompt = t.promptBuilder.BuildVarPrompt(req)

	// 2. Call LLM
	resp, err := t.opts.LLMTranslator(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("LLM error: %s", resp.Error)
	}

	// 3. Build target Var
	targetName := t.convertVarName(src.Name, src.IsExported)
	targetVar := &uniast.Var{
		IsExported: src.IsExported,
		IsConst:    src.IsConst,
		IsPointer:  src.IsPointer,
		Identity: uniast.Identity{
			ModPath: tctx.Module.Name,
			PkgPath: string(tctx.Package.PkgPath),
			Name:    targetName,
		},
		FileLine: uniast.FileLine{
			File: t.convertFilePath(src.File),
			Line: src.Line,
		},
		Content: resp.TargetContent,
	}

	return targetVar, nil
}

// collectDependencyHints collects hints about already translated dependencies
func (t *NodeTranslator) collectDependencyHints(srcID uniast.Identity, tctx *TranslateContext) []DependencyHint {
	var hints []DependencyHint

	// Get the node from source repository
	node := tctx.SourceRepo.GetNode(srcID)
	if node == nil {
		return hints
	}

	// Collect hints for each dependency
	for _, dep := range node.Dependencies {
		if targetID, ok := tctx.GetTranslatedNode(dep.Identity); ok {
			hint := DependencyHint{
				SourceIdentity: dep.Identity,
				TargetIdentity: targetID,
			}

			// Try to get the signature of the translated dependency
			if targetNode := tctx.TargetRepo.GetNode(targetID); targetNode != nil {
				hint.TargetSignature = targetNode.Signature()
			}

			hints = append(hints, hint)
		}
	}

	return hints
}

// convertTypeName converts a type name to target language convention
func (t *NodeTranslator) convertTypeName(name string, exported bool) string {
	switch t.opts.TargetLanguage {
	case uniast.Golang:
		// Go: PascalCase for exported, camelCase for unexported
		if exported {
			return toPascalCase(name)
		}
		return toCamelCase(name)
	case uniast.Rust:
		// Rust: PascalCase for types
		return toPascalCase(name)
	case uniast.Python:
		// Python: PascalCase for classes
		return toPascalCase(name)
	case uniast.Java:
		// Java: PascalCase for classes
		return toPascalCase(name)
	default:
		return name
	}
}

// convertFunctionName converts a function name to target language convention
func (t *NodeTranslator) convertFunctionName(name string, exported bool) string {
	switch t.opts.TargetLanguage {
	case uniast.Golang:
		// Go: PascalCase for exported, camelCase for unexported
		if exported {
			return toPascalCase(name)
		}
		return toCamelCase(name)
	case uniast.Rust:
		// Rust: snake_case for functions
		return toSnakeCase(name)
	case uniast.Python:
		// Python: snake_case for functions
		return toSnakeCase(name)
	case uniast.Java:
		// Java: camelCase for methods
		return toCamelCase(name)
	default:
		return name
	}
}

// convertVarName converts a variable name to target language convention
func (t *NodeTranslator) convertVarName(name string, exported bool) string {
	switch t.opts.TargetLanguage {
	case uniast.Golang:
		// Go: PascalCase for exported, camelCase for unexported
		if exported {
			return toPascalCase(name)
		}
		return toCamelCase(name)
	case uniast.Rust:
		// Rust: snake_case for variables, SCREAMING_SNAKE for constants
		return toSnakeCase(name)
	case uniast.Python:
		// Python: snake_case for variables
		return toSnakeCase(name)
	case uniast.Java:
		// Java: camelCase for fields
		return toCamelCase(name)
	default:
		return name
	}
}

// convertFilePath converts a file path to target language convention
func (t *NodeTranslator) convertFilePath(path string) string {
	if path == "" {
		return path
	}

	ext := t.getFileExtension()
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	switch t.opts.TargetLanguage {
	case uniast.Golang:
		return strings.ToLower(base) + ext
	case uniast.Rust:
		return toSnakeCase(base) + ext
	case uniast.Python:
		return toSnakeCase(base) + ext
	case uniast.Java:
		return toPascalCase(base) + ext
	default:
		return base + ext
	}
}

// getFileExtension returns the file extension for the target language
func (t *NodeTranslator) getFileExtension() string {
	switch t.opts.TargetLanguage {
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

// Naming convention conversion helpers

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Handle snake_case and kebab-case
	words := splitWords(s)
	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}
	return result.String()
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == ' ' {
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// splitWords splits a string into words based on case changes and separators
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if unicode.IsUpper(r) && i > 0 {
			// New word starts with uppercase
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}
