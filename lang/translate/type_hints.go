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

// TypeHints provides type mapping hints for LLM translation
type TypeHints struct {
	source   uniast.Language
	target   uniast.Language
	mappings map[string]string
}

// NewTypeHints creates a new TypeHints instance
func NewTypeHints(source, target uniast.Language) *TypeHints {
	h := &TypeHints{
		source:   source,
		target:   target,
		mappings: make(map[string]string),
	}
	h.loadMappings()
	return h
}

// loadMappings loads the type mappings for the source-target language pair
func (h *TypeHints) loadMappings() {
	key := fmt.Sprintf("%s->%s", h.source, h.target)
	switch key {
	case "java->go":
		h.mappings = javaToGoMappings()
	case "go->java":
		h.mappings = goToJavaMappings()
	case "java->rust":
		h.mappings = javaToRustMappings()
	case "rust->java":
		h.mappings = rustToJavaMappings()
	case "go->rust":
		h.mappings = goToRustMappings()
	case "rust->go":
		h.mappings = rustToGoMappings()
	case "python->go":
		h.mappings = pythonToGoMappings()
	case "go->python":
		h.mappings = goToPythonMappings()
	case "java->python":
		h.mappings = javaToPythonMappings()
	case "python->java":
		h.mappings = pythonToJavaMappings()
	case "python->rust":
		h.mappings = pythonToRustMappings()
	case "rust->python":
		h.mappings = rustToPythonMappings()
	case "typescript->go", "ts->go":
		h.mappings = typescriptToGoMappings()
	default:
		h.mappings = make(map[string]string)
	}
}

// FormatForPrompt formats the type mappings as a markdown table for inclusion in prompts
func (h *TypeHints) FormatForPrompt() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("| %s | %s |\n", h.source, h.target))
	sb.WriteString("|---|---|\n")
	for src, tgt := range h.mappings {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", src, tgt))
	}
	return sb.String()
}

// GetMapping returns the target type for a source type
func (h *TypeHints) GetMapping(sourceType string) (string, bool) {
	target, ok := h.mappings[sourceType]
	return target, ok
}

// AddMapping adds a custom mapping
func (h *TypeHints) AddMapping(sourceType, targetType string) {
	h.mappings[sourceType] = targetType
}

// Java -> Go type mappings
func javaToGoMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":       "int",
		"Integer":   "int",
		"long":      "int64",
		"Long":      "int64",
		"short":     "int16",
		"Short":     "int16",
		"byte":      "byte",
		"Byte":      "byte",
		"float":     "float32",
		"Float":     "float32",
		"double":    "float64",
		"Double":    "float64",
		"boolean":   "bool",
		"Boolean":   "bool",
		"char":      "rune",
		"Character": "rune",
		"String":    "string",
		"void":      "",
		"Object":    "interface{}",

		// Collections
		"List<T>":      "[]T",
		"ArrayList<T>": "[]T",
		"LinkedList<T>": "[]T",
		"Set<T>":       "map[T]struct{}",
		"HashSet<T>":   "map[T]struct{}",
		"Map<K,V>":     "map[K]V",
		"HashMap<K,V>": "map[K]V",
		"Optional<T>":  "*T",

		// Common types
		"BigInteger": "*big.Int",
		"BigDecimal": "*big.Float",
		"Date":       "time.Time",
		"LocalDate":  "time.Time",
		"LocalDateTime": "time.Time",
		"Instant":    "time.Time",
	}
}

// TypeScript -> Go type mappings
func typescriptToGoMappings() map[string]string {
	return map[string]string{
		// Primitives
		"string":   "string",
		"number":   "int64",
		"boolean":  "bool",
		"void":     "",
		"null":     "nil",
		"undefined": "zero value or omit",

		// TS built-in / common
		"any":       "any",
		"unknown":   "interface{}",
		"object":    "map[string]interface{}",
		"never":     "// no Go equivalent",

		// Arrays and collections
		"Array<T>": "[]T",
		"T[]":       "[]T",
		"ReadonlyArray<T>": "[]T",
		"Record<K,V>": "map[K]V",
		"Map<K,V>":  "map[K]V",
		"Set<T>":    "map[T]struct{}",

		// Promise -> suggest goroutine/channel or return type
		"Promise<T>": "T (or use goroutine/channel)",

		// Common TS/JS
		"Date":     "time.Time",
		"Error":    "error",
		"Promise":  "// use goroutine or return value",
	}
}

// Go -> Java type mappings
func goToJavaMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":     "int",
		"int8":    "byte",
		"int16":   "short",
		"int32":   "int",
		"int64":   "long",
		"uint":    "int",
		"uint8":   "byte",
		"uint16":  "int",
		"uint32":  "long",
		"uint64":  "long",
		"float32": "float",
		"float64": "double",
		"bool":    "boolean",
		"string":  "String",
		"byte":    "byte",
		"rune":    "char",

		// Composite types
		"[]T":         "List<T>",
		"map[K]V":     "Map<K,V>",
		"*T":          "T",
		"error":       "Exception",
		"interface{}": "Object",
		"any":         "Object",

		// Common types
		"time.Time":     "LocalDateTime",
		"time.Duration": "Duration",
	}
}

// Java -> Rust type mappings
func javaToRustMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":       "i32",
		"Integer":   "i32",
		"long":      "i64",
		"Long":      "i64",
		"short":     "i16",
		"Short":     "i16",
		"byte":      "i8",
		"Byte":      "i8",
		"float":     "f32",
		"Float":     "f32",
		"double":    "f64",
		"Double":    "f64",
		"boolean":   "bool",
		"Boolean":   "bool",
		"char":      "char",
		"Character": "char",
		"String":    "String",
		"void":      "()",
		"Object":    "Box<dyn Any>",

		// Collections
		"List<T>":      "Vec<T>",
		"ArrayList<T>": "Vec<T>",
		"Set<T>":       "HashSet<T>",
		"HashSet<T>":   "HashSet<T>",
		"Map<K,V>":     "HashMap<K,V>",
		"HashMap<K,V>": "HashMap<K,V>",
		"Optional<T>":  "Option<T>",
	}
}

// Rust -> Java type mappings
func rustToJavaMappings() map[string]string {
	return map[string]string{
		// Primitives
		"i8":    "byte",
		"i16":   "short",
		"i32":   "int",
		"i64":   "long",
		"u8":    "byte",
		"u16":   "int",
		"u32":   "long",
		"u64":   "long",
		"f32":   "float",
		"f64":   "double",
		"bool":  "boolean",
		"char":  "char",
		"String": "String",
		"&str":  "String",
		"()":    "void",

		// Composite types
		"Vec<T>":        "List<T>",
		"HashSet<T>":    "Set<T>",
		"HashMap<K,V>":  "Map<K,V>",
		"Option<T>":     "Optional<T>",
		"Result<T,E>":   "T throws Exception",
		"Box<T>":        "T",
	}
}

// Go -> Rust type mappings
func goToRustMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":     "i64",
		"int8":    "i8",
		"int16":   "i16",
		"int32":   "i32",
		"int64":   "i64",
		"uint":    "u64",
		"uint8":   "u8",
		"uint16":  "u16",
		"uint32":  "u32",
		"uint64":  "u64",
		"float32": "f32",
		"float64": "f64",
		"bool":    "bool",
		"string":  "String",
		"byte":    "u8",
		"rune":    "char",

		// Composite types
		"[]T":         "Vec<T>",
		"map[K]V":     "HashMap<K,V>",
		"*T":          "Option<Box<T>>",
		"error":       "Result<T, Error>",
		"interface{}": "Box<dyn Any>",
		"any":         "Box<dyn Any>",
	}
}

// Rust -> Go type mappings
func rustToGoMappings() map[string]string {
	return map[string]string{
		// Primitives
		"i8":    "int8",
		"i16":   "int16",
		"i32":   "int32",
		"i64":   "int64",
		"u8":    "uint8",
		"u16":   "uint16",
		"u32":   "uint32",
		"u64":   "uint64",
		"f32":   "float32",
		"f64":   "float64",
		"bool":  "bool",
		"char":  "rune",
		"String": "string",
		"&str":  "string",
		"()":    "",

		// Composite types
		"Vec<T>":       "[]T",
		"HashSet<T>":   "map[T]struct{}",
		"HashMap<K,V>": "map[K]V",
		"Option<T>":    "*T",
		"Result<T,E>":  "(T, error)",
		"Box<T>":       "*T",
	}
}

// Python -> Go type mappings
func pythonToGoMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":   "int",
		"float": "float64",
		"str":   "string",
		"bool":  "bool",
		"None":  "nil",
		"bytes": "[]byte",

		// Composite types
		"list":  "[]",
		"List":  "[]",
		"dict":  "map",
		"Dict":  "map",
		"set":   "map[T]struct{}",
		"Set":   "map[T]struct{}",
		"tuple": "struct",
		"Tuple": "struct",
		"Optional": "*",
		"Any":   "interface{}",
	}
}

// Go -> Python type mappings
func goToPythonMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":     "int",
		"int8":    "int",
		"int16":   "int",
		"int32":   "int",
		"int64":   "int",
		"uint":    "int",
		"uint8":   "int",
		"uint16":  "int",
		"uint32":  "int",
		"uint64":  "int",
		"float32": "float",
		"float64": "float",
		"bool":    "bool",
		"string":  "str",
		"byte":    "bytes",
		"[]byte":  "bytes",

		// Composite types
		"[]T":         "List[T]",
		"map[K]V":     "Dict[K, V]",
		"*T":          "Optional[T]",
		"error":       "Exception",
		"interface{}": "Any",
		"any":         "Any",
	}
}

// Java -> Python type mappings
func javaToPythonMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":       "int",
		"Integer":   "int",
		"long":      "int",
		"Long":      "int",
		"short":     "int",
		"Short":     "int",
		"byte":      "int",
		"Byte":      "int",
		"float":     "float",
		"Float":     "float",
		"double":    "float",
		"Double":    "float",
		"boolean":   "bool",
		"Boolean":   "bool",
		"char":      "str",
		"Character": "str",
		"String":    "str",
		"void":      "None",
		"Object":    "Any",

		// Collections
		"List<T>":      "List[T]",
		"ArrayList<T>": "List[T]",
		"Set<T>":       "Set[T]",
		"HashSet<T>":   "Set[T]",
		"Map<K,V>":     "Dict[K, V]",
		"HashMap<K,V>": "Dict[K, V]",
		"Optional<T>":  "Optional[T]",
	}
}

// Python -> Java type mappings
func pythonToJavaMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":   "int",
		"float": "double",
		"str":   "String",
		"bool":  "boolean",
		"None":  "void",
		"bytes": "byte[]",

		// Composite types
		"list":     "List",
		"List":     "List",
		"dict":     "Map",
		"Dict":     "Map",
		"set":      "Set",
		"Set":      "Set",
		"tuple":    "List",
		"Tuple":    "List",
		"Optional": "Optional",
		"Any":      "Object",
	}
}

// Python -> Rust type mappings
func pythonToRustMappings() map[string]string {
	return map[string]string{
		// Primitives
		"int":   "i64",
		"float": "f64",
		"str":   "String",
		"bool":  "bool",
		"None":  "()",
		"bytes": "Vec<u8>",

		// Composite types
		"list":     "Vec",
		"List":     "Vec",
		"dict":     "HashMap",
		"Dict":     "HashMap",
		"set":      "HashSet",
		"Set":      "HashSet",
		"Optional": "Option",
		"Any":      "Box<dyn Any>",
	}
}

// Rust -> Python type mappings
func rustToPythonMappings() map[string]string {
	return map[string]string{
		// Primitives
		"i8":    "int",
		"i16":   "int",
		"i32":   "int",
		"i64":   "int",
		"u8":    "int",
		"u16":   "int",
		"u32":   "int",
		"u64":   "int",
		"f32":   "float",
		"f64":   "float",
		"bool":  "bool",
		"char":  "str",
		"String": "str",
		"&str":  "str",
		"()":    "None",

		// Composite types
		"Vec<T>":       "List[T]",
		"HashSet<T>":   "Set[T]",
		"HashMap<K,V>": "Dict[K, V]",
		"Option<T>":    "Optional[T]",
		"Result<T,E>":  "T",
		"Box<T>":       "T",
	}
}
