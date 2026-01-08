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
	"strings"
	"testing"
)

func TestGenerateAvailableSkillsXML(t *testing.T) {
	skills := []*SkillMetadata{
		{
			Name:        "test-skill",
			Description: "A test skill",
			Path:        "/path/to/test-skill/SKILL.md",
		},
		{
			Name:        "another-skill",
			Description: "Another test skill",
			Path:        "/path/to/another-skill/SKILL.md",
		},
	}

	xml := GenerateAvailableSkillsXML(skills)

	// 验证 XML 结构
	if !strings.Contains(xml, "<available_skills>") {
		t.Error("XML should contain <available_skills>")
	}
	if !strings.Contains(xml, "</available_skills>") {
		t.Error("XML should contain </available_skills>")
	}
	if !strings.Contains(xml, "<name>test-skill</name>") {
		t.Error("XML should contain test-skill name")
	}
	if !strings.Contains(xml, "<description>A test skill</description>") {
		t.Error("XML should contain test skill description")
	}
	if !strings.Contains(xml, "<location>/path/to/test-skill/SKILL.md</location>") {
		t.Error("XML should contain test skill location")
	}
}

func TestGenerateSkillInstructionsPrompt(t *testing.T) {
	skill := &Skill{
		Name:          "test-skill",
		Description:   "A test skill",
		Compatibility: "Requires test environment",
		AllowedTools:  []string{"tool1", "tool2"},
		Instructions:  "# Instructions\n\nDo something.",
	}

	prompt := GenerateSkillInstructionsPrompt(skill)

	if !strings.Contains(prompt, "## Skill: test-skill") {
		t.Error("Prompt should contain skill name")
	}
	if !strings.Contains(prompt, "A test skill") {
		t.Error("Prompt should contain description")
	}
	if !strings.Contains(prompt, "Requires test environment") {
		t.Error("Prompt should contain compatibility")
	}
	if !strings.Contains(prompt, "tool1, tool2") {
		t.Error("Prompt should contain allowed tools")
	}
	if !strings.Contains(prompt, "# Instructions") {
		t.Error("Prompt should contain instructions")
	}
}
