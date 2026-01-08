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
	"strings"
)

// TypeMapping defines Java to Go type mappings
type TypeMapping struct {
	JavaType string
	GoType   string
}

// GetGoType maps Java type to Go type
func GetGoType(javaType string) string {
	// Remove generics for basic mapping
	baseType := strings.Split(javaType, "<")[0]
	baseType = strings.TrimSpace(baseType)

	// Basic type mappings
	typeMap := map[string]string{
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
		"List":       "[]",
		"ArrayList":  "[]",
		"LinkedList": "[]",
		"Set":        "[]",
		"HashSet":    "[]",
		"Map":        "map[string]",
		"HashMap":    "map[string]",
		"Optional":   "interface{}", // Will be converted to (T, bool) pattern

		// Common types
		"BigInteger": "big.Int",
		"BigDecimal": "big.Float",
	}

	if mapped, ok := typeMap[baseType]; ok {
		// Handle generics for collections
		if strings.Contains(javaType, "<") {
			genericPart := extractGeneric(javaType)
			if strings.HasPrefix(mapped, "[]") {
				// List<T> -> []T
				return mapped[:2] + GetGoType(genericPart)
			} else if strings.HasPrefix(mapped, "map[") {
				// Map<K, V> -> map[K]V
				parts := splitGeneric(genericPart)
				if len(parts) == 2 {
					return "map[" + GetGoType(parts[0]) + "]" + GetGoType(parts[1])
				}
			}
		}
		return mapped
	}

	// If not found, assume it's a custom type - convert package path
	return ConvertPackagePath(baseType)
}

// extractGeneric extracts the generic type from Java type like "List<String>"
func extractGeneric(javaType string) string {
	start := strings.Index(javaType, "<")
	end := strings.LastIndex(javaType, ">")
	if start >= 0 && end > start {
		return strings.TrimSpace(javaType[start+1 : end])
	}
	return ""
}

// splitGeneric splits "K, V" into ["K", "V"]
func splitGeneric(generic string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, r := range generic {
		switch r {
		case '<':
			depth++
			current.WriteRune(r)
		case '>':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, strings.TrimSpace(current.String()))
	}
	return parts
}

// ConvertPackagePath converts Java package path to Go module path
// Example: com.example.project -> github.com/example/project
func ConvertPackagePath(javaPath string) string {
	// Remove common Java prefixes
	javaPath = strings.TrimPrefix(javaPath, "java.lang.")
	javaPath = strings.TrimPrefix(javaPath, "java.util.")

	// Split by dots
	parts := strings.Split(javaPath, ".")
	if len(parts) == 0 {
		return javaPath
	}

	// Convert com.example.project to github.com/example/project
	if parts[0] == "com" && len(parts) > 1 {
		// Skip "com" and join the rest
		return "github.com/" + strings.Join(parts[1:], "/")
	}

	// For other cases, just join with slashes
	return strings.Join(parts, "/")
}

// ConvertJavaPackageToGoModule converts Java package name to simplified Go package name
// Example: com.example.common.model -> model
// Example: com.example.core.service -> service
// Takes only the last segment as the package name for flat structure
func ConvertJavaPackageToGoModule(javaPackage string) string {
	parts := strings.Split(javaPackage, ".")
	if len(parts) == 0 {
		return javaPackage
	}

	// Take only the last segment as the package name
	// This creates a flat structure: model, service, repository, etc.
	return parts[len(parts)-1]
}

// GetGoModuleNameFromGroupId converts Java groupId to Go module name
// Example: com.example.test -> example.com/test
// Example: com.haier.ai.gencode -> haier.com/ai/gencode
func GetGoModuleNameFromGroupId(groupId string) string {
	parts := strings.Split(groupId, ".")
	if len(parts) == 0 {
		return groupId
	}

	// Handle common patterns: com.example.test -> example.com/test
	if len(parts) >= 2 && parts[0] == "com" {
		// com.example.test -> example.com/test
		if len(parts) == 2 {
			return parts[1] + ".com"
		}
		return parts[1] + ".com/" + strings.Join(parts[2:], "/")
	}

	// For other patterns, join with slashes
	return strings.Join(parts, "/")
}

// ConvertMethodName converts Java method name to Go function name
// Java: camelCase -> Go: PascalCase (exported) or camelCase (unexported)
func ConvertMethodName(javaName string, exported bool) string {
	if len(javaName) == 0 {
		return javaName
	}

	if exported {
		// Convert to PascalCase
		return strings.ToUpper(javaName[:1]) + javaName[1:]
	}
	// Keep camelCase for unexported
	return javaName
}

// ConvertFieldName converts Java field name to Go field name
// Similar to method name conversion
func ConvertFieldName(javaName string, exported bool) string {
	return ConvertMethodName(javaName, exported)
}

// ConvertClassName converts Java class name to Go type name
// Java: PascalCase -> Go: PascalCase (always exported)
func ConvertClassName(javaName string) string {
	if len(javaName) == 0 {
		return javaName
	}
	// Java class names are already PascalCase, just ensure first letter is uppercase
	return strings.ToUpper(javaName[:1]) + javaName[1:]
}
