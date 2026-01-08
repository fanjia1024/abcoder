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

import "testing"

func TestGetGoType(t *testing.T) {
	tests := []struct {
		javaType string
		expected string
	}{
		{"int", "int"},
		{"Integer", "int"},
		{"String", "string"},
		{"boolean", "bool"},
		{"List<String>", "[]string"},
		{"Map<String, Integer>", "map[string]int"},
		{"Object", "interface{}"},
	}

	for _, tt := range tests {
		result := GetGoType(tt.javaType)
		if result != tt.expected {
			t.Errorf("GetGoType(%q) = %q, want %q", tt.javaType, result, tt.expected)
		}
	}
}

func TestConvertJavaPackageToGoModule(t *testing.T) {
	tests := []struct {
		javaPackage string
		expected    string
	}{
		{"com.example.project", "github.com/example/project"},
		{"org.apache.commons", "github.com/apache/commons"},
		{"java.lang", "java/lang"},
	}

	for _, tt := range tests {
		result := ConvertJavaPackageToGoModule(tt.javaPackage)
		if result != tt.expected {
			t.Errorf("ConvertJavaPackageToGoModule(%q) = %q, want %q", tt.javaPackage, result, tt.expected)
		}
	}
}

func TestConvertMethodName(t *testing.T) {
	tests := []struct {
		javaName string
		exported bool
		expected string
	}{
		{"getUser", true, "GetUser"},
		{"getUser", false, "getUser"},
		{"calculateTotal", true, "CalculateTotal"},
	}

	for _, tt := range tests {
		result := ConvertMethodName(tt.javaName, tt.exported)
		if result != tt.expected {
			t.Errorf("ConvertMethodName(%q, %v) = %q, want %q", tt.javaName, tt.exported, result, tt.expected)
		}
	}
}
