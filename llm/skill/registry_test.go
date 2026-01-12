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

	"github.com/cloudwego/abcoder/llm/skill/embedded"
)

func TestRegistry_DiscoverSkills(t *testing.T) {
	registry := NewRegistry()
	err := registry.DiscoverSkills()
	if err != nil {
		t.Fatalf("DiscoverSkills failed: %v", err)
	}

	// 检查是否发现了预期数量的 skills
	count := registry.Count()
	expectedCount := len(embedded.SkillPaths())
	t.Logf("Discovered %d skills (expected %d)", count, expectedCount)

	if count == 0 {
		t.Error("No skills discovered")
	}

	// 列出所有发现的 skills
	metadataList := registry.ListMetadata()
	for _, meta := range metadataList {
		t.Logf("  - %s: %s", meta.Name, truncate(meta.Description, 50))
	}
}

func TestRegistry_ActivateSkill(t *testing.T) {
	registry := NewRegistry()
	err := registry.DiscoverSkills()
	if err != nil {
		t.Fatalf("DiscoverSkills failed: %v", err)
	}

	// 测试激活几个 skills
	testSkills := []string{"code-analysis", "mcp-builder", "pdf"}
	for _, skillName := range testSkills {
		skill, err := registry.ActivateSkill(skillName)
		if err != nil {
			t.Errorf("ActivateSkill(%s) failed: %v", skillName, err)
			continue
		}

		if skill.Name != skillName {
			t.Errorf("Expected skill name %s, got %s", skillName, skill.Name)
		}
		if skill.Description == "" {
			t.Errorf("Skill %s has empty description", skillName)
		}
		if skill.Instructions == "" {
			t.Errorf("Skill %s has empty instructions", skillName)
		}
		t.Logf("Activated skill: %s (tools: %v)", skill.Name, skill.AllowedTools)
	}
}

func TestRegistry_GetAvailableSkillsXML(t *testing.T) {
	registry := NewRegistry()
	err := registry.DiscoverSkills()
	if err != nil {
		t.Fatalf("DiscoverSkills failed: %v", err)
	}

	xml := registry.GetAvailableSkillsXML()
	if xml == "" {
		t.Error("GetAvailableSkillsXML returned empty string")
	}

	// 检查 XML 格式
	if !strings.Contains(xml, "<available_skills>") {
		t.Error("XML should contain <available_skills>")
	}
	if !strings.Contains(xml, "</available_skills>") {
		t.Error("XML should contain </available_skills>")
	}

	t.Logf("XML length: %d bytes", len(xml))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
