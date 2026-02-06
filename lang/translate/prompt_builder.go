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
	"fmt"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// PromptBuilder builds prompts for LLM translation
type PromptBuilder struct {
	source    uniast.Language
	target    uniast.Language
	typeHints *TypeHints
}

// NewPromptBuilder creates a new PromptBuilder
func NewPromptBuilder(source, target uniast.Language, typeHints *TypeHints) *PromptBuilder {
	return &PromptBuilder{
		source:    source,
		target:    target,
		typeHints: typeHints,
	}
}

// BuildTypePrompt builds a prompt for translating a type
func (b *PromptBuilder) BuildTypePrompt(req *LLMTranslateRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Translate the following %s type/class to %s.\n\n", b.source, b.target))

	// Add type mapping reference
	sb.WriteString("## Type Mapping Reference\n")
	sb.WriteString(b.typeHints.FormatForPrompt())
	sb.WriteString("\n")

	// Add already translated dependencies
	if len(req.Dependencies) > 0 {
		sb.WriteString("## Already Translated Dependencies\n")
		b.writeDependencies(&sb, req.Dependencies)
		sb.WriteString("\n")
	}

	// Add source code
	sb.WriteString("## Source Code\n")
	if req.SourceTruncated {
		sb.WriteString("Note: Source was truncated for context limit; translate the visible part only.\n\n")
	}
	sb.WriteString("```")
	sb.WriteString(string(b.source))
	sb.WriteString("\n")
	sb.WriteString(req.SourceContent)
	sb.WriteString("\n```\n\n")

	// Add requirements
	sb.WriteString("## Requirements\n")
	sb.WriteString(b.getTypeRequirements())
	sb.WriteString("\n\n")

	// Add output format
	sb.WriteString("## Output\n")
	sb.WriteString("Return ONLY the translated code, no explanations or markdown formatting.\n")

	return sb.String()
}

// BuildFunctionPrompt builds a prompt for translating a function
func (b *PromptBuilder) BuildFunctionPrompt(req *LLMTranslateRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Translate the following %s function/method to %s.\n\n", b.source, b.target))

	// Add type mapping reference
	sb.WriteString("## Type Mapping Reference\n")
	sb.WriteString(b.typeHints.FormatForPrompt())
	sb.WriteString("\n")

	// Add already translated dependencies
	if len(req.Dependencies) > 0 {
		sb.WriteString("## Already Translated Dependencies\n")
		b.writeDependencies(&sb, req.Dependencies)
		sb.WriteString("\n")
	}

	// Add source code
	sb.WriteString("## Source Code\n")
	if req.SourceTruncated {
		sb.WriteString("Note: Source was truncated for context limit; translate the visible part only.\n\n")
	}
	sb.WriteString("```")
	sb.WriteString(string(b.source))
	sb.WriteString("\n")
	sb.WriteString(req.SourceContent)
	sb.WriteString("\n```\n\n")

	// Add requirements
	sb.WriteString("## Requirements\n")
	sb.WriteString(b.getFunctionRequirements())
	sb.WriteString("\n\n")

	// Add output format
	sb.WriteString("## Output\n")
	sb.WriteString("Return ONLY the translated code, no explanations or markdown formatting.\n")

	return sb.String()
}

// BuildVarPrompt builds a prompt for translating a variable
func (b *PromptBuilder) BuildVarPrompt(req *LLMTranslateRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Translate the following %s variable/constant to %s.\n\n", b.source, b.target))

	// Add type mapping reference
	sb.WriteString("## Type Mapping Reference\n")
	sb.WriteString(b.typeHints.FormatForPrompt())
	sb.WriteString("\n")

	// Add already translated dependencies
	if len(req.Dependencies) > 0 {
		sb.WriteString("## Already Translated Dependencies\n")
		b.writeDependencies(&sb, req.Dependencies)
		sb.WriteString("\n")
	}

	// Add source code
	sb.WriteString("## Source Code\n")
	if req.SourceTruncated {
		sb.WriteString("Note: Source was truncated for context limit; translate the visible part only.\n\n")
	}
	sb.WriteString("```")
	sb.WriteString(string(b.source))
	sb.WriteString("\n")
	sb.WriteString(req.SourceContent)
	sb.WriteString("\n```\n\n")

	// Add requirements
	sb.WriteString("## Requirements\n")
	sb.WriteString(b.getVarRequirements())
	sb.WriteString("\n\n")

	// Add output format
	sb.WriteString("## Output\n")
	sb.WriteString("Return ONLY the translated code, no explanations or markdown formatting.\n")

	return sb.String()
}

// writeDependencies writes dependency hints to the builder
func (b *PromptBuilder) writeDependencies(sb *strings.Builder, deps []DependencyHint) {
	for _, dep := range deps {
		sb.WriteString(fmt.Sprintf("- `%s` -> `%s`\n", dep.SourceIdentity.Name, dep.TargetIdentity.Name))
		if dep.TargetSignature != "" {
			sb.WriteString(fmt.Sprintf("  Signature: `%s`\n", dep.TargetSignature))
		}
	}
}

// getTypeRequirements returns language-specific requirements for type translation
func (b *PromptBuilder) getTypeRequirements() string {
	common := `- Preserve the semantics and functionality of the original type
- Use idiomatic patterns for the target language
- IMPORTANT: Do NOT redeclare types that already exist in the dependencies
- IMPORTANT: Include ALL necessary import statements at the TOP of the code
- IMPORTANT: For cross-package types, use full package prefix (e.g., model.User, not just User)
- Output ONLY the type definition, do NOT include duplicate struct/method definitions
`
	if b.source == uniast.TypeScript {
		common = `- Source is TypeScript: convert interface to Go struct or interface; convert class to struct with methods; use exported (PascalCase) for public, unexported for private
` + common
	}

	switch b.target {
	case uniast.Golang:
		return common + `- Use Go naming conventions (PascalCase for exported types)
- Convert class to struct with methods
- Convert inheritance to composition or embedding where appropriate
- Use pointer receivers for methods that modify state
- Add json tags if the original had serialization annotations
- Include required imports: "time" for time.Time, "sync" for sync.Mutex, etc.
- Do NOT define helper types like UserStatus multiple times`
	case uniast.Rust:
		return common + `- Use Rust naming conventions (PascalCase for types)
- Convert class to struct with impl block
- Use derive macros for common traits (Debug, Clone, etc.)
- Use Option<T> for nullable fields
- Implement proper ownership semantics`
	case uniast.Python:
		return common + `- Use Python naming conventions (PascalCase for classes)
- Add type hints for all fields and methods
- Use @dataclass decorator where appropriate
- Add __init__, __repr__, and other dunder methods as needed
- Use Optional[] for nullable types`
	case uniast.Java:
		return common + `- Use Java naming conventions (PascalCase for classes, camelCase for fields)
- Add appropriate access modifiers (public, private, protected)
- Generate getters and setters for private fields
- Add @Override annotations where appropriate
- Use Optional<T> for nullable fields`
	default:
		return common
	}
}

// getFunctionRequirements returns language-specific requirements for function translation
func (b *PromptBuilder) getFunctionRequirements() string {
	common := `- Preserve the semantics and functionality of the original function
- Use idiomatic patterns for the target language
- IMPORTANT: Do NOT redeclare functions/methods that already exist
- IMPORTANT: Include ALL necessary import statements at the TOP of the code
- IMPORTANT: Use types from dependencies with proper package prefix (e.g., model.User)
- Output ONLY the single function/method, do NOT include duplicate definitions
`
	if b.source == uniast.TypeScript {
		common = `- Source is TypeScript: convert Promise<T> to (T, error) or return T; use Go error as last return; map async/await to synchronous Go or goroutines where appropriate
` + common
	}

	switch b.target {
	case uniast.Golang:
		return common + `- Use Go naming conventions (PascalCase for exported, camelCase for unexported)
- Use Go error handling pattern (return error as last return value)
- Use multiple return values instead of out parameters
- Use Go slices instead of arrays where appropriate
- Add appropriate comments for exported functions
- Include required imports: "time" for time.Time, "fmt" for fmt.Sprintf, etc.
- For methods, output only the method signature and body, not the struct definition`
	case uniast.Rust:
		return common + `- Use Rust naming conventions (snake_case for functions)
- Use Result<T, E> for functions that can fail
- Use proper ownership and borrowing
- Use iterators and closures idiomatically
- Add lifetime annotations where necessary`
	case uniast.Python:
		return common + `- Use Python naming conventions (snake_case for functions)
- Add type hints for parameters and return type
- Use Pythonic patterns (list comprehensions, generators, etc.)
- Raise exceptions for error handling
- Add docstrings for documentation`
	case uniast.Java:
		return common + `- Use Java naming conventions (camelCase for methods)
- Use proper access modifiers
- Handle checked exceptions appropriately
- Use Optional<T> for methods that may not return a value
- Add Javadoc comments for public methods`
	default:
		return common
	}
}

// getVarRequirements returns language-specific requirements for variable translation
func (b *PromptBuilder) getVarRequirements() string {
	common := `- Preserve the value and meaning of the variable
- Use appropriate type for the target language
- IMPORTANT: Do NOT redeclare variables/constants that already exist
- Output ONLY the single variable/constant definition
`

	switch b.target {
	case uniast.Golang:
		return common + `- Use Go naming conventions (PascalCase for exported, camelCase for unexported)
- Use const for constants, var for variables
- Use appropriate Go types
- Use iota for enum-like constants if applicable
- Do NOT duplicate const blocks that define the same constants`
	case uniast.Rust:
		return common + `- Use Rust naming conventions (SCREAMING_SNAKE_CASE for constants, snake_case for variables)
- Use const for compile-time constants
- Use static for runtime constants
- Use appropriate Rust types`
	case uniast.Python:
		return common + `- Use Python naming conventions (SCREAMING_SNAKE_CASE for constants, snake_case for variables)
- Add type hints
- Use Final[] for constants where appropriate`
	case uniast.Java:
		return common + `- Use Java naming conventions (SCREAMING_SNAKE_CASE for constants, camelCase for fields)
- Use final for constants
- Use static final for class constants
- Add appropriate access modifiers`
	default:
		return common
	}
}
