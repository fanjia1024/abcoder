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

//go:embed code-analysis/SKILL.md code-translation/SKILL.md code-review/SKILL.md doc-generation/SKILL.md test-generation/SKILL.md
var EmbeddedFS embed.FS

// SkillPaths 返回所有内置 skill 的路径
func SkillPaths() []string {
	return []string{
		"code-analysis/SKILL.md",
		"code-translation/SKILL.md",
		"code-review/SKILL.md",
		"doc-generation/SKILL.md",
		"test-generation/SKILL.md",
	}
}
