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
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/abcoder/llm/skill/embedded"
	"gopkg.in/yaml.v3"
)

// Registry 管理所有可用的 skills（支持渐进式加载）
type Registry struct {
	mu       sync.RWMutex
	metadata map[string]*SkillMetadata // Phase 1: 只有元数据
	loaded   map[string]*Skill         // Phase 2: 完整加载的 skills
	executor *ScriptExecutor            // 脚本执行器
	localDir string                     // 本地 skills 目录
	loader   *Loader
}

// NewRegistry 创建新的 Registry
func NewRegistry() *Registry {
	return &Registry{
		metadata: make(map[string]*SkillMetadata),
		loaded:   make(map[string]*Skill),
		executor: NewScriptExecutor(),
		loader:   NewLoader(),
	}
}

// SetLocalDir 设置本地 skills 目录
func (r *Registry) SetLocalDir(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.localDir = dir
}

// DiscoverSkills 发现所有 skills（只加载元数据，Phase 1）
func (r *Registry) DiscoverSkills() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 发现内置 skills
	skillPaths := embedded.SkillPaths()
	for _, path := range skillPaths {
		// 从 embed.FS 读取文件
		data, err := embedded.EmbeddedFS.ReadFile(path)
		if err != nil {
			log.Error("Failed to read embedded skill %s: %v", path, err)
			continue
		}

		// 提取元数据
		frontmatter, _, err := r.loader.extractFrontmatter(string(data))
		if err != nil {
			log.Error("Failed to extract frontmatter from %s: %v", path, err)
			continue
		}

		var meta struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
			log.Error("Failed to parse frontmatter from %s: %v", path, err)
			continue
		}

		// 验证
		if err := ValidateDescription(meta.Description); err != nil {
			log.Error("Invalid description in %s: %v", path, err)
			continue
		}

		// 提取目录名（从路径推断）
		dirName := filepath.Dir(path)
		baseDirName := filepath.Base(dirName)
		if err := ValidateName(meta.Name, baseDirName); err != nil {
			log.Error("Invalid name in %s: %v", path, err)
			continue
		}

		skillMeta := &SkillMetadata{
			Name:        meta.Name,
			Description: meta.Description,
			Path:        path, // embed 路径
			BasePath:    dirName,
			Source:      SourceEmbedded,
		}

		r.metadata[meta.Name] = skillMeta
	}

	// 发现本地 skills
	if r.localDir != "" {
		if _, err := os.Stat(r.localDir); err == nil {
			metadataList, err := r.loader.LoadAllMetadataFromDir(r.localDir, SourceLocal)
			if err == nil {
				for _, meta := range metadataList {
					r.metadata[meta.Name] = meta
				}
			}
		}
	} else {
		// 尝试默认目录
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultDir := filepath.Join(homeDir, ".abcoder", "skills")
			if _, err := os.Stat(defaultDir); err == nil {
				metadataList, err := r.loader.LoadAllMetadataFromDir(defaultDir, SourceLocal)
				if err == nil {
					for _, meta := range metadataList {
						r.metadata[meta.Name] = meta
					}
				}
			}
		}
	}

	log.Info("Discovered %d skills", len(r.metadata))
	return nil
}

// ActivateSkill 激活 skill（加载完整内容，Phase 2）
func (r *Registry) ActivateSkill(name string) (*Skill, error) {
	r.mu.RLock()
	meta, exists := r.metadata[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	// 检查是否已加载
	r.mu.RLock()
	if skill, loaded := r.loaded[name]; loaded {
		r.mu.RUnlock()
		return skill, nil
	}
	r.mu.RUnlock()

	// 加载完整 skill
	var skill *Skill
	var err error

	if meta.Source == SourceEmbedded {
		// 从 embed.FS 加载
		data, err := embedded.EmbeddedFS.ReadFile(meta.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded skill: %w", err)
		}
		skill, err = r.loader.ParseSkill(data, SourceEmbedded, meta.BasePath)
		if err == nil {
			// 设置资源目录（对于 embed，这些目录可能不存在，但路径需要设置）
			skill.ScriptsDir = filepath.Join(meta.BasePath, ScriptsDir)
			skill.ReferencesDir = filepath.Join(meta.BasePath, ReferencesDir)
			skill.AssetsDir = filepath.Join(meta.BasePath, AssetsDir)
		}
	} else {
		// 从文件系统加载
		skill, err = r.loader.LoadFull(meta)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load skill: %w", err)
	}

	// 缓存加载的 skill
	r.mu.Lock()
	r.loaded[name] = skill
	r.mu.Unlock()

	return skill, nil
}

// GetAvailableSkillsXML 获取所有可用 skills 的 XML 表示
func (r *Registry) GetAvailableSkillsXML() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadataList := make([]*SkillMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		metadataList = append(metadataList, meta)
	}

	return GenerateAvailableSkillsXML(metadataList)
}

// LoadEmbedded 加载内置 skills（向后兼容，内部使用 DiscoverSkills）
func (r *Registry) LoadEmbedded() error {
	return r.DiscoverSkills()
}

// LoadFromDir 从目录加载 skills（向后兼容，内部使用 DiscoverSkills）
func (r *Registry) LoadFromDir(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadataList, err := r.loader.LoadAllMetadataFromDir(dir, SourceLocal)
	if err != nil {
		return fmt.Errorf("failed to load skills from directory %s: %w", dir, err)
	}

	for _, meta := range metadataList {
		r.metadata[meta.Name] = meta
	}

	log.Info("Discovered %d skills from directory %s", len(metadataList), dir)
	return nil
}

// LoadFromGitHub 从 GitHub 加载 skill
func (r *Registry) LoadFromGitHub(repo, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 使用 remote.go 中的功能
	skill, err := fetchSkillFromGitHub(repo, path)
	if err != nil {
		return fmt.Errorf("failed to fetch skill from GitHub %s/%s: %w", repo, path, err)
	}

	if err := r.registerSkill(skill); err != nil {
		return fmt.Errorf("failed to register skill from GitHub: %w", err)
	}

	log.Info("Loaded skill %s from GitHub %s/%s", skill.Name, repo, path)
	return nil
}

// registerSkill 注册一个 skill（内部方法，已废弃，保留用于向后兼容）
func (r *Registry) registerSkill(skill *Skill) error {
	// 转换为 metadata 并注册
	meta := &SkillMetadata{
		Name:        skill.Name,
		Description: skill.Description,
		Path:        skill.Path,
		BasePath:    skill.BasePath,
		Source:      skill.Source,
	}
	r.metadata[skill.Name] = meta
	r.loaded[skill.Name] = skill
	return nil
}

// Get 根据名称获取 skill（如果未加载则自动激活）
func (r *Registry) Get(name string) *Skill {
	r.mu.RLock()
	skill, loaded := r.loaded[name]
	r.mu.RUnlock()

	if loaded {
		return skill
	}

	// 自动激活
	skill, err := r.ActivateSkill(name)
	if err != nil {
		return nil
	}
	return skill
}

// List 返回所有 skills（自动激活未加载的）
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	metadataList := make([]*SkillMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		metadataList = append(metadataList, meta)
	}
	r.mu.RUnlock()

	skills := make([]*Skill, 0, len(metadataList))
	for _, meta := range metadataList {
		skill := r.Get(meta.Name)
		if skill != nil {
			skills = append(skills, skill)
		}
	}
	return skills
}

// ListMetadata 返回所有 skills 的元数据（不加载完整内容）
func (r *Registry) ListMetadata() []*SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadataList := make([]*SkillMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		metadataList = append(metadataList, meta)
	}
	return metadataList
}

// Search 根据查询字符串搜索 skills（使用 matcher）
func (r *Registry) Search(query string) []*Skill {
	r.mu.RLock()
	metadataList := make([]*SkillMetadata, 0, len(r.metadata))
	for _, meta := range r.metadata {
		metadataList = append(metadataList, meta)
	}
	r.mu.RUnlock()

	// 在元数据上匹配
	matcher := NewMatcher()
	matchedMetadata := matcher.MatchMetadata(query, metadataList)

	// 激活匹配的 skills
	skills := make([]*Skill, 0, len(matchedMetadata))
	for _, meta := range matchedMetadata {
		skill := r.Get(meta.Name)
		if skill != nil {
			skills = append(skills, skill)
		}
	}
	return skills
}

// Initialize 初始化 registry（加载内置和本地 skills）
func (r *Registry) Initialize() error {
	// 加载内置 skills
	if err := r.LoadEmbedded(); err != nil {
		return fmt.Errorf("failed to load embedded skills: %w", err)
	}

	// 加载本地 skills 目录（如果存在）
	if r.localDir != "" {
		if _, err := os.Stat(r.localDir); err == nil {
			if err := r.LoadFromDir(r.localDir); err != nil {
				log.Error("Failed to load skills from local directory: %v", err)
			}
		}
	} else {
		// 尝试默认目录
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultDir := filepath.Join(homeDir, ".abcoder", "skills")
			if _, err := os.Stat(defaultDir); err == nil {
				if err := r.LoadFromDir(defaultDir); err != nil {
					log.Debug("No skills found in default directory: %v", err)
				}
			}
		}
	}

	return nil
}

// Remove 移除一个 skill
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := false
	if _, exists := r.metadata[name]; exists {
		delete(r.metadata, name)
		removed = true
	}
	if _, exists := r.loaded[name]; exists {
		delete(r.loaded, name)
		removed = true
	}
	return removed
}

// Count 返回 skill 数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.metadata)
}

// GetExecutor 获取脚本执行器
func (r *Registry) GetExecutor() *ScriptExecutor {
	return r.executor
}
