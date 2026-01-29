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

package embedded

import (
	"embed"
)

// EmbeddedFS 包含所有内置 skills 的文件系统
// 使用 all: 前缀来包含所有文件（包括以 . 开头的文件）
//
//go:embed all:algorithmic-art all:brand-guidelines all:canvas-design
//go:embed all:code-analysis all:code-review all:code-translation all:hierarchical-translation
//go:embed all:doc-coauthoring all:doc-generation all:docx
//go:embed all:frontend-design all:internal-comms all:mcp-builder
//go:embed all:pdf all:pptx all:skill-creator all:slack-gif-creator
//go:embed all:test-generation all:theme-factory all:web-artifacts-builder
//go:embed all:webapp-testing all:xlsx
var EmbeddedFS embed.FS

// SkillPaths 返回所有内置 skill 的 SKILL.md 路径
func SkillPaths() []string {
	return []string{
		// abcoder 原生 skills（代码相关）
		"code-analysis/SKILL.md",
		"code-translation/SKILL.md",
		"hierarchical-translation/SKILL.md",
		"code-review/SKILL.md",
		"doc-generation/SKILL.md",
		"test-generation/SKILL.md",

		// 官方 skills - Creative & Design
		"algorithmic-art/SKILL.md",
		"canvas-design/SKILL.md",
		"frontend-design/SKILL.md",
		"slack-gif-creator/SKILL.md",
		"theme-factory/SKILL.md",

		// 官方 skills - Development & Technical
		"mcp-builder/SKILL.md",
		"skill-creator/SKILL.md",
		"web-artifacts-builder/SKILL.md",
		"webapp-testing/SKILL.md",

		// 官方 skills - Enterprise & Communication
		"brand-guidelines/SKILL.md",
		"doc-coauthoring/SKILL.md",
		"internal-comms/SKILL.md",

		// 官方 skills - Document Skills
		"docx/SKILL.md",
		"pdf/SKILL.md",
		"pptx/SKILL.md",
		"xlsx/SKILL.md",
	}
}

// SkillCategories 返回 skill 分类信息
func SkillCategories() map[string][]string {
	return map[string][]string{
		"Code & Development": {
			"code-analysis",
			"code-translation",
			"hierarchical-translation",
			"code-review",
			"doc-generation",
			"test-generation",
		},
		"Creative & Design": {
			"algorithmic-art",
			"canvas-design",
			"frontend-design",
			"slack-gif-creator",
			"theme-factory",
		},
		"Development & Technical": {
			"mcp-builder",
			"skill-creator",
			"web-artifacts-builder",
			"webapp-testing",
		},
		"Enterprise & Communication": {
			"brand-guidelines",
			"doc-coauthoring",
			"internal-comms",
		},
		"Document Skills": {
			"docx",
			"pdf",
			"pptx",
			"xlsx",
		},
	}
}
