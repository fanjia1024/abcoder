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
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// Translate executes the complete code translation flow:
// 1. Parse source code to generate UniAST (if input is a path)
// 2. Use LLM to transform UniAST to target language UniAST
// 3. Use target language Writer to output code
func Translate(ctx context.Context, input interface{}, opts TranslateOptions) error {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	var srcRepo *uniast.Repository
	var err error

	switch v := input.(type) {
	case string:
		// Input is a path - could be a directory or a JSON file
		srcRepo, err = LoadRepository(ctx, v, opts.SourceLanguage)
		if err != nil {
			return fmt.Errorf("load repository failed: %w", err)
		}
	case *uniast.Repository:
		srcRepo = v
	default:
		return fmt.Errorf("unsupported input type: %T, expected string (path) or *uniast.Repository", input)
	}

	// Infer source language if not specified
	if opts.SourceLanguage == uniast.Unknown {
		opts.SourceLanguage = inferLanguage(srcRepo)
	}

	// Stage 2: Transform - Use LLM to convert AST
	transformer := NewTransformer(opts)
	targetRepo, err := transformer.Transform(ctx, srcRepo)
	if err != nil {
		return fmt.Errorf("transform AST failed: %w", err)
	}

	// Validate target UniAST before writing; reject invalid LLM output to avoid writing bad code
	if err := uniast.ValidateRepository(targetRepo); err != nil {
		return fmt.Errorf("UniAST validation failed (rejecting output): %w", err)
	}

	// Stage 3: Write - Use existing writer to output code
	if opts.OutputDir != "" {
		err = lang.Write(ctx, targetRepo, lang.WriteOptions{
			OutputDir: opts.OutputDir,
		})
		if err != nil {
			return fmt.Errorf("write target code failed: %w", err)
		}
	}

	return nil
}

// TranslateAST performs only AST transformation without parsing or writing
// Useful when you already have an AST and don't need file output
func TranslateAST(ctx context.Context, srcRepo *uniast.Repository, opts TranslateOptions) (*uniast.Repository, error) {
	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Infer source language if not specified
	if opts.SourceLanguage == uniast.Unknown {
		opts.SourceLanguage = inferLanguage(srcRepo)
	}

	transformer := NewTransformer(opts)
	return transformer.Transform(ctx, srcRepo)
}

// validateOptions checks if the required options are provided
func validateOptions(opts TranslateOptions) error {
	if opts.TargetLanguage == uniast.Unknown {
		return fmt.Errorf("TargetLanguage is required")
	}
	if opts.LLMTranslator == nil {
		return fmt.Errorf("LLMTranslator callback is required")
	}
	return nil
}

// LoadRepository loads a repository from a path.
// The path can be either a directory (to parse) or a JSON file (pre-parsed AST).
// Exported for use by pipeline steps.
func LoadRepository(ctx context.Context, path string, language uniast.Language) (*uniast.Repository, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if !info.IsDir() && (len(path) > 5 && path[len(path)-5:] == ".json") {
		// Load from JSON file
		return loadRepositoryFromJSON(path)
	}

	// Parse from directory using lang.Parse
	parseOpts := lang.ParseOptions{
		CollectOption: collect.CollectOption{
			Language: language,
		},
	}

	data, err := lang.Parse(ctx, path, parseOpts)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	var repo uniast.Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("unmarshal repository failed: %w", err)
	}

	return &repo, nil
}

// loadRepositoryFromJSON loads a repository from a JSON file
func loadRepositoryFromJSON(path string) (*uniast.Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	var repo uniast.Repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("unmarshal repository failed: %w", err)
	}

	return &repo, nil
}

// inferLanguage tries to infer the source language from the repository
func inferLanguage(repo *uniast.Repository) uniast.Language {
	for _, mod := range repo.Modules {
		if !mod.IsExternal() && mod.Language != uniast.Unknown {
			return mod.Language
		}
	}
	return uniast.Unknown
}
