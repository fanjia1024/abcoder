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
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SkillFileName        = "SKILL.md"
	FrontMatterDelimiter = "---"
	ScriptsDir           = "scripts"
	ReferencesDir        = "references"
	AssetsDir            = "assets"
)

// ResourceType 表示资源类型
type ResourceType int

const (
	ResourceTypeScript ResourceType = iota
	ResourceTypeReference
	ResourceTypeAsset
)

// Loader 负责加载和解析 SKILL.md 文件
type Loader struct{}

// NewLoader 创建新的 Loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadMetadataOnly 只加载元数据（Phase 1）
// 用于启动时发现所有可用 skills
func (l *Loader) LoadMetadataOnly(dir string, source SkillSource) (*SkillMetadata, error) {
	skillPath := filepath.Join(dir, SkillFileName)
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file %s: %w", skillPath, err)
	}

	// 只解析 frontmatter
	frontmatter, _, err := l.extractFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// 验证必需字段
	if err := ValidateDescription(meta.Description); err != nil {
		return nil, err
	}

	// 验证 name 并匹配目录名
	baseDir := filepath.Base(dir)
	if err := ValidateName(meta.Name, baseDir); err != nil {
		return nil, err
	}

	return &SkillMetadata{
		Name:     meta.Name,
		Description: meta.Description,
		Path:     skillPath,
		BasePath: dir,
		Source:   source,
	}, nil
}

// LoadFull 加载完整 skill（Phase 2）
// 当 skill 被激活时调用
func (l *Loader) LoadFull(meta *SkillMetadata) (*Skill, error) {
	data, err := os.ReadFile(meta.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file %s: %w", meta.Path, err)
	}

	skill, err := l.ParseSkill(data, meta.Source, meta.BasePath)
	if err != nil {
		return nil, err
	}

	// 检查并设置资源目录
	skill.ScriptsDir = filepath.Join(meta.BasePath, ScriptsDir)
	skill.ReferencesDir = filepath.Join(meta.BasePath, ReferencesDir)
	skill.AssetsDir = filepath.Join(meta.BasePath, AssetsDir)

	// 检查目录是否存在
	if _, err := os.Stat(skill.ScriptsDir); os.IsNotExist(err) {
		skill.ScriptsDir = ""
	}
	if _, err := os.Stat(skill.ReferencesDir); os.IsNotExist(err) {
		skill.ReferencesDir = ""
	}
	if _, err := os.Stat(skill.AssetsDir); os.IsNotExist(err) {
		skill.AssetsDir = ""
	}

	return skill, nil
}

// LoadResource 按需加载资源（Phase 3）
// 从 scripts/, references/, assets/ 加载指定文件
func (l *Loader) LoadResource(skill *Skill, resourceType ResourceType, filename string) ([]byte, error) {
	var baseDir string
	switch resourceType {
	case ResourceTypeScript:
		baseDir = skill.ScriptsDir
	case ResourceTypeReference:
		baseDir = skill.ReferencesDir
	case ResourceTypeAsset:
		baseDir = skill.AssetsDir
	default:
		return nil, fmt.Errorf("unknown resource type: %v", resourceType)
	}

	if baseDir == "" {
		return nil, fmt.Errorf("resource directory not found for type %v", resourceType)
	}

	// 安全检查：防止路径遍历攻击
	if strings.Contains(filename, "..") || strings.HasPrefix(filename, "/") {
		return nil, fmt.Errorf("invalid filename: %s", filename)
	}

	filePath := filepath.Join(baseDir, filename)
	return os.ReadFile(filePath)
}

// LoadFromFile 从文件路径加载 skill（向后兼容，内部调用 LoadFull）
func (l *Loader) LoadFromFile(path string, source SkillSource) (*Skill, error) {
	dir := filepath.Dir(path)
	meta, err := l.LoadMetadataOnly(dir, source)
	if err != nil {
		return nil, err
	}
	return l.LoadFull(meta)
}

// LoadFromFS 从 embed.FS 加载 skill（向后兼容，内部使用 ParseSkill）
func (l *Loader) LoadFromFS(fsys embed.FS, path string, source SkillSource) (*Skill, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill from embed FS %s: %w", path, err)
	}
	// 提取 basePath（目录路径）
	basePath := filepath.Dir(path)
	return l.ParseSkill(data, source, basePath)
}

// LoadFromDir 从目录加载 skill（查找 SKILL.md）
func (l *Loader) LoadFromDir(dir string, source SkillSource) (*Skill, error) {
	skillPath := filepath.Join(dir, SkillFileName)
	return l.LoadFromFile(skillPath, source)
}

// ParseSkill 解析 SKILL.md 内容（Phase 2）
func (l *Loader) ParseSkill(data []byte, source SkillSource, basePath string) (*Skill, error) {
	content := string(data)
	
	// 分离 frontmatter 和 markdown 内容
	frontmatter, instructions, err := l.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// 解析 YAML frontmatter
	var meta struct {
		Name          string            `yaml:"name"`
		Description   string            `yaml:"description"`
		AllowedTools  interface{}       `yaml:"allowed-tools"` // 可能是字符串或数组
		License       string            `yaml:"license"`
		Compatibility string            `yaml:"compatibility"`
		Metadata      map[string]string `yaml:"metadata"`
	}

	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// 验证必需字段
	if err := ValidateDescription(meta.Description); err != nil {
		return nil, err
	}

	// 验证 name（需要目录名）
	baseDir := filepath.Base(basePath)
	if err := ValidateName(meta.Name, baseDir); err != nil {
		return nil, err
	}

	// 验证 compatibility
	if err := ValidateCompatibility(meta.Compatibility); err != nil {
		return nil, err
	}

	// 解析 allowed-tools（支持字符串和数组两种格式）
	var allowedTools []string
	if meta.AllowedTools != nil {
		switch v := meta.AllowedTools.(type) {
		case string:
			// 官方规范：空格分隔字符串
			allowedTools = ParseAllowedTools(v)
		case []interface{}:
			// 向后兼容：YAML 数组格式
			for _, item := range v {
				if str, ok := item.(string); ok {
					allowedTools = append(allowedTools, str)
				}
			}
		}
	}

	// 合并 metadata
	metadata := make(map[string]string)
	if meta.Metadata != nil {
		for k, v := range meta.Metadata {
			metadata[k] = v
		}
	}
	if meta.License != "" {
		metadata["license"] = meta.License
	}

	skillPath := filepath.Join(basePath, SkillFileName)
	skill := &Skill{
		Name:          meta.Name,
		Description:   meta.Description,
		License:       meta.License,
		Compatibility: meta.Compatibility,
		AllowedTools:  allowedTools,
		Metadata:      metadata,
		Instructions:  strings.TrimSpace(instructions),
		Source:        source,
		BasePath:      basePath,
		Path:          skillPath,
	}

	return skill, nil
}

// extractFrontmatter 从 markdown 文件中提取 YAML frontmatter
func (l *Loader) extractFrontmatter(content string) (frontmatter string, body string, err error) {
	content = strings.TrimSpace(content)
	
	// 检查是否有 frontmatter
	if !strings.HasPrefix(content, FrontMatterDelimiter) {
		return "", content, fmt.Errorf("no frontmatter found (expected '---' at start)")
	}

	// 找到第一个 --- 后的内容
	firstDelim := strings.Index(content, "\n"+FrontMatterDelimiter)
	if firstDelim == -1 {
		// 尝试查找单独的 ---
		lines := strings.Split(content, "\n")
		if len(lines) < 2 || lines[0] != FrontMatterDelimiter {
			return "", content, fmt.Errorf("invalid frontmatter format")
		}
		// 查找第二个 ---
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == FrontMatterDelimiter {
				frontmatter = strings.Join(lines[1:i], "\n")
				body = strings.Join(lines[i+1:], "\n")
				return frontmatter, body, nil
			}
		}
		return "", content, fmt.Errorf("frontmatter not closed")
	}

	// 提取 frontmatter（跳过第一个 ---）
	frontmatter = content[len(FrontMatterDelimiter)+1 : firstDelim]
	
	// 提取 body（跳过第二个 ---）
	bodyStart := firstDelim + len(FrontMatterDelimiter) + 1
	if bodyStart < len(content) {
		body = content[bodyStart:]
	}

	return strings.TrimSpace(frontmatter), strings.TrimSpace(body), nil
}

// LoadAllMetadataFromDir 从目录加载所有 skills 的元数据（Phase 1）
// 递归查找所有包含 SKILL.md 的目录
func (l *Loader) LoadAllMetadataFromDir(rootDir string, source SkillSource) ([]*SkillMetadata, error) {
	var metadataList []*SkillMetadata
	
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() {
			return nil
		}
		
		// 检查目录中是否有 SKILL.md
		skillPath := filepath.Join(path, SkillFileName)
		if _, err := os.Stat(skillPath); err == nil {
			meta, err := l.LoadMetadataOnly(path, source)
			if err != nil {
				// 记录错误但继续处理其他目录
				return nil
			}
			metadataList = append(metadataList, meta)
		}
		
		return nil
	})
	
	return metadataList, err
}

// LoadAllFromDir 从目录加载所有 skills（向后兼容，内部使用 LoadAllMetadataFromDir）
func (l *Loader) LoadAllFromDir(rootDir string, source SkillSource) ([]*Skill, error) {
	metadataList, err := l.LoadAllMetadataFromDir(rootDir, source)
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, meta := range metadataList {
		skill, err := l.LoadFull(meta)
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}
