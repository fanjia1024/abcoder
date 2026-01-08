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
	"strings"
	"html"
)

// GenerateAvailableSkillsXML 生成用于系统提示的 XML 格式
// 遵循官方推荐的 <available_skills> 格式
func GenerateAvailableSkillsXML(skills []*SkillMetadata) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	for _, skill := range skills {
		sb.WriteString("  <skill>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", html.EscapeString(skill.Name)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", html.EscapeString(skill.Description)))
		sb.WriteString(fmt.Sprintf("    <location>%s</location>\n", html.EscapeString(skill.Path)))
		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}

// GenerateSkillInstructionsPrompt 生成 skill 指令的提示文本
// 用于在激活 skill 时注入到系统提示中
func GenerateSkillInstructionsPrompt(skill *Skill) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Skill: %s\n\n", skill.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", skill.Description))

	if skill.Compatibility != "" {
		sb.WriteString(fmt.Sprintf("**Compatibility**: %s\n\n", skill.Compatibility))
	}

	if len(skill.AllowedTools) > 0 {
		sb.WriteString("**Allowed Tools**: ")
		sb.WriteString(strings.Join(skill.AllowedTools, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString(skill.Instructions)

	return sb.String()
}
