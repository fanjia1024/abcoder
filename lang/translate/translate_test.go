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
	"strings"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// mockLLMTranslator is a mock LLM translator for testing
func mockLLMTranslator(ctx context.Context, req *LLMTranslateRequest) (*LLMTranslateResponse, error) {
	// Simple mock: just return the source content with a comment indicating translation
	return &LLMTranslateResponse{
		TargetContent: "// Translated from " + string(req.SourceLanguage) + "\n" + req.SourceContent,
	}, nil
}

func TestTranslateAST(t *testing.T) {
	// Create a simple Java repository for testing
	srcRepo := createTestJavaRepo()

	opts := TranslateOptions{
		SourceLanguage:   uniast.Java,
		TargetLanguage:   uniast.Golang,
		TargetModuleName: "github.com/example/test",
		LLMTranslator:    mockLLMTranslator,
	}

	ctx := context.Background()
	targetRepo, err := TranslateAST(ctx, srcRepo, opts)
	if err != nil {
		t.Fatalf("TranslateAST failed: %v", err)
	}

	if targetRepo == nil {
		t.Fatal("targetRepo is nil")
	}

	if targetRepo.Name != "github.com/example/test" {
		t.Errorf("unexpected repo name: %s", targetRepo.Name)
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    TranslateOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: TranslateOptions{
				TargetLanguage: uniast.Golang,
				LLMTranslator:  mockLLMTranslator,
			},
			wantErr: false,
		},
		{
			name: "missing target language",
			opts: TranslateOptions{
				LLMTranslator: mockLLMTranslator,
			},
			wantErr: true,
		},
		{
			name: "missing LLM translator",
			opts: TranslateOptions{
				TargetLanguage: uniast.Golang,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTypeHints(t *testing.T) {
	tests := []struct {
		source uniast.Language
		target uniast.Language
		input  string
		want   string
	}{
		{uniast.Java, uniast.Golang, "String", "string"},
		{uniast.Java, uniast.Golang, "int", "int"},
		{uniast.Java, uniast.Golang, "List<T>", "[]T"},
		{uniast.Golang, uniast.Rust, "string", "String"},
		{uniast.Golang, uniast.Rust, "[]T", "Vec<T>"},
		{uniast.Python, uniast.Golang, "str", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hints := NewTypeHints(tt.source, tt.target)
			got, ok := hints.GetMapping(tt.input)
			if !ok {
				t.Errorf("GetMapping(%q) not found", tt.input)
				return
			}
			if got != tt.want {
				t.Errorf("GetMapping(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTypeHintsFormatForPrompt(t *testing.T) {
	hints := NewTypeHints(uniast.Java, uniast.Golang)
	prompt := hints.FormatForPrompt()

	if !strings.Contains(prompt, "java") {
		t.Error("prompt should contain 'java'")
	}
	if !strings.Contains(prompt, "go") {
		t.Error("prompt should contain 'go'")
	}
	if !strings.Contains(prompt, "String") {
		t.Error("prompt should contain 'String'")
	}
}

func TestStructureAdapter(t *testing.T) {
	adapter := NewStructureAdapter(uniast.Java, uniast.Golang)

	// Test module name conversion
	modName := adapter.convertModuleName("com.example.project")
	if modName != "example.com/project" {
		t.Errorf("convertModuleName() = %q, want %q", modName, "example.com/project")
	}

	// Test package path conversion
	pkgPath := adapter.convertPackagePath("com.example.project.model")
	if pkgPath != "model" {
		t.Errorf("convertPackagePath() = %q, want %q", pkgPath, "model")
	}

	// Test file path conversion
	filePath := adapter.convertFilePath("User.java")
	if filePath != "user.go" {
		t.Errorf("convertFilePath() = %q, want %q", filePath, "user.go")
	}
}

func TestNamingConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		toPascal string
		toCamel  string
		toSnake  string
	}{
		{"simple", "hello", "Hello", "hello", "hello"},
		{"camelCase", "helloWorld", "HelloWorld", "helloWorld", "hello_world"},
		{"PascalCase", "HelloWorld", "HelloWorld", "helloWorld", "hello_world"},
		{"snake_case", "hello_world", "HelloWorld", "helloWorld", "hello_world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toPascalCase(tt.input); got != tt.toPascal {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.toPascal)
			}
			if got := toCamelCase(tt.input); got != tt.toCamel {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.toCamel)
			}
			if got := toSnakeCase(tt.input); got != tt.toSnake {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.toSnake)
			}
		})
	}
}

func TestPromptBuilder(t *testing.T) {
	hints := NewTypeHints(uniast.Java, uniast.Golang)
	builder := NewPromptBuilder(uniast.Java, uniast.Golang, hints)

	req := &LLMTranslateRequest{
		SourceLanguage: uniast.Java,
		TargetLanguage: uniast.Golang,
		NodeType:       uniast.FUNC,
		SourceContent:  "public void hello() { }",
		Identity: uniast.Identity{
			ModPath: "com.example",
			PkgPath: "com.example.test",
			Name:    "hello",
		},
	}

	// Test function prompt
	prompt := builder.BuildFunctionPrompt(req)
	if !strings.Contains(prompt, "java") {
		t.Error("function prompt should contain 'java'")
	}
	if !strings.Contains(prompt, "go") {
		t.Error("function prompt should contain 'go'")
	}
	if !strings.Contains(prompt, "public void hello()") {
		t.Error("function prompt should contain source content")
	}

	// Test type prompt
	req.NodeType = uniast.TYPE
	req.SourceContent = "public class User { }"
	typePrompt := builder.BuildTypePrompt(req)
	if !strings.Contains(typePrompt, "type/class") {
		t.Error("type prompt should contain 'type/class'")
	}

	// Test var prompt
	req.NodeType = uniast.VAR
	req.SourceContent = "public static final int MAX = 100;"
	varPrompt := builder.BuildVarPrompt(req)
	if !strings.Contains(varPrompt, "variable/constant") {
		t.Error("var prompt should contain 'variable/constant'")
	}
}

func TestNodeTranslator(t *testing.T) {
	opts := TranslateOptions{
		SourceLanguage: uniast.Java,
		TargetLanguage: uniast.Golang,
		LLMTranslator:  mockLLMTranslator,
	}

	hints := NewTypeHints(uniast.Java, uniast.Golang)
	translator := NewNodeTranslator(opts, hints)

	// Create test context
	srcRepo := createTestJavaRepo()
	targetRepo := uniast.NewRepository("github.com/example/test")
	targetMod := uniast.NewModule("github.com/example/test", "", uniast.Golang)
	targetPkg := &uniast.Package{
		PkgPath:   "model",
		Functions: make(map[string]*uniast.Function),
		Types:     make(map[string]*uniast.Type),
		Vars:      make(map[string]*uniast.Var),
	}

	tctx := &TranslateContext{
		SourceRepo:      srcRepo,
		TargetRepo:      &targetRepo,
		Module:          targetMod,
		Package:         targetPkg,
		TranslatedNodes: make(map[string]uniast.Identity),
	}

	// Test TranslateType
	srcType := &uniast.Type{
		Exported: true,
		TypeKind: uniast.TypeKindStruct,
		Identity: uniast.Identity{
			ModPath: "com.example:test:1.0",
			PkgPath: "com.example.model",
			Name:    "User",
		},
		FileLine: uniast.FileLine{
			File: "User.java",
			Line: 1,
		},
		Content: "public class User { private String name; }",
	}

	ctx := context.Background()
	targetType, err := translator.TranslateType(ctx, srcType, tctx)
	if err != nil {
		t.Fatalf("TranslateType failed: %v", err)
	}
	if targetType.Name != "User" {
		t.Errorf("unexpected type name: %s", targetType.Name)
	}

	// Test TranslateFunction
	srcFunc := &uniast.Function{
		Exported: true,
		IsMethod: false,
		Identity: uniast.Identity{
			ModPath: "com.example:test:1.0",
			PkgPath: "com.example.service",
			Name:    "getUser",
		},
		FileLine: uniast.FileLine{
			File: "UserService.java",
			Line: 10,
		},
		Content: "public User getUser(String id) { return null; }",
	}

	targetFunc, err := translator.TranslateFunction(ctx, srcFunc, tctx)
	if err != nil {
		t.Fatalf("TranslateFunction failed: %v", err)
	}
	if targetFunc.Name != "GetUser" {
		t.Errorf("unexpected function name: %s", targetFunc.Name)
	}
}

// createTestJavaRepo creates a test Java repository
func createTestJavaRepo() *uniast.Repository {
	repo := uniast.NewRepository("com.example:test:1.0")

	mod := uniast.NewModule("com.example:test:1.0", "src/main/java", uniast.Java)

	pkg := &uniast.Package{
		PkgPath:   "com.example.model",
		Functions: make(map[string]*uniast.Function),
		Types:     make(map[string]*uniast.Type),
		Vars:      make(map[string]*uniast.Var),
	}

	pkg.Types["User"] = &uniast.Type{
		Exported: true,
		TypeKind: uniast.TypeKindStruct,
		Identity: uniast.Identity{
			ModPath: "com.example:test:1.0",
			PkgPath: "com.example.model",
			Name:    "User",
		},
		Content: "public class User { private String name; }",
	}

	mod.Packages["com.example.model"] = pkg
	repo.Modules["com.example:test:1.0"] = mod

	return &repo
}
