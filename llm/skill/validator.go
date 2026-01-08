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

package skill

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ValidateName 验证 name 字段
// - 1-64 字符
// - 只能包含小写字母、数字、连字符
// - 不能以连字符开头/结尾
// - 不能有连续连字符
// - 必须匹配目录名
func ValidateName(name string, dirName string) error {
	// 检查长度
	if len(name) == 0 {
		return fmt.Errorf("skill name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("skill name must be 1-64 characters, got %d", len(name))
	}

	// 检查字符：只能包含小写字母、数字、连字符
	for _, r := range name {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '-' {
			return fmt.Errorf("skill name can only contain lowercase letters, numbers, and hyphens, got '%c'", r)
		}
	}

	// 不能以连字符开头/结尾
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("skill name cannot start with hyphen")
	}
	if strings.HasSuffix(name, "-") {
		return fmt.Errorf("skill name cannot end with hyphen")
	}

	// 不能有连续连字符
	if strings.Contains(name, "--") {
		return fmt.Errorf("skill name cannot contain consecutive hyphens")
	}

	// 必须匹配目录名
	baseDir := filepath.Base(dirName)
	if name != baseDir {
		return fmt.Errorf("skill name '%s' must match directory name '%s'", name, baseDir)
	}

	return nil
}

// ValidateDescription 验证 description 字段
// - 1-1024 字符
func ValidateDescription(desc string) error {
	if len(desc) == 0 {
		return fmt.Errorf("skill description cannot be empty")
	}
	if len(desc) > 1024 {
		return fmt.Errorf("skill description must be 1-1024 characters, got %d", len(desc))
	}
	return nil
}

// ValidateCompatibility 验证 compatibility 字段
// - 0-500 字符（可选字段，可以为空）
func ValidateCompatibility(compat string) error {
	if len(compat) > 500 {
		return fmt.Errorf("skill compatibility must be 0-500 characters, got %d", len(compat))
	}
	return nil
}

// ParseAllowedTools 解析空格分隔的 allowed-tools 字符串
// 例如: "Bash(git:*) Read Write" -> []string{"Bash(git:*)", "Read", "Write"}
// 支持括号内的内容（如 Bash(git:*)）
func ParseAllowedTools(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	var tools []string
	var current strings.Builder
	parenDepth := 0

	for _, r := range toolsStr {
		switch r {
		case '(':
			parenDepth++
			current.WriteRune(r)
		case ')':
			parenDepth--
			current.WriteRune(r)
		case ' ':
			if parenDepth == 0 {
				// 不在括号内，遇到空格表示一个工具结束
				if current.Len() > 0 {
					tools = append(tools, strings.TrimSpace(current.String()))
					current.Reset()
				}
			} else {
				// 在括号内，空格是工具名的一部分
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	// 添加最后一个工具
	if current.Len() > 0 {
		tools = append(tools, strings.TrimSpace(current.String()))
	}

	return tools
}
