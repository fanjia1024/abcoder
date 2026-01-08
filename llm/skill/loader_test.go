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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoader_ParseSkill(t *testing.T) {
	loader := NewLoader()

	// 创建临时测试目录（目录名必须匹配 skill name）
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}
	skillFile := filepath.Join(skillDir, "SKILL.md")
	
	skillContent := `---
name: test-skill
description: A test skill for unit testing
allowed-tools: tool1 tool2
metadata:
  category: test
---

# Test Skill

This is a test skill for unit testing.
`

	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write test skill file: %v", err)
	}

	// 加载 skill metadata（Phase 1）
	meta, err := loader.LoadMetadataOnly(skillDir, SourceLocal)
	if err != nil {
		t.Fatalf("Failed to load skill metadata: %v", err)
	}

	// 验证 metadata
	if meta.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", meta.Name)
	}
	if meta.Description != "A test skill for unit testing" {
		t.Errorf("Unexpected description: %s", meta.Description)
	}

	// 加载完整 skill（Phase 2）
	skill, err := loader.LoadFull(meta)
	if err != nil {
		t.Fatalf("Failed to load full skill: %v", err)
	}

	// 验证 skill 内容
	if skill.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Name)
	}
	if len(skill.AllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d", len(skill.AllowedTools))
	}
	if skill.Source != SourceLocal {
		t.Errorf("Expected source Local, got %v", skill.Source)
	}
	if !strings.Contains(skill.Instructions, "Test Skill") {
		t.Errorf("Instructions should contain 'Test Skill'")
	}
}

func TestLoader_ExtractFrontmatter(t *testing.T) {
	loader := NewLoader()

	content := `---
name: test
description: test desc
---

# Content

This is the content.
`

	frontmatter, body, err := loader.extractFrontmatter(content)
	if err != nil {
		t.Fatalf("Failed to extract frontmatter: %v", err)
	}

	if !strings.Contains(frontmatter, "name: test") {
		t.Errorf("Frontmatter should contain 'name: test'")
	}
	if !strings.Contains(body, "Content") {
		t.Errorf("Body should contain 'Content'")
	}
}
