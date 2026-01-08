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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/llm/log"
)

const (
	GitHubAPIBase = "https://api.github.com"
	GitHubRawBase = "https://raw.githubusercontent.com"
)

// fetchSkillFromGitHub 从 GitHub 获取 skill
func fetchSkillFromGitHub(repo, path string) (*Skill, error) {
	// 解析 repo（格式：owner/repo 或 owner/repo@branch）
	parts := strings.Split(repo, "@")
	repoPath := parts[0]
	branch := "main"
	if len(parts) > 1 {
		branch = parts[1]
	}

	// 构建 SKILL.md 的 URL
	skillURL := fmt.Sprintf("%s/%s/%s/%s", GitHubRawBase, repoPath, branch, path)
	if !strings.HasSuffix(path, SkillFileName) {
		if strings.HasSuffix(path, "/") {
			skillURL = skillURL + SkillFileName
		} else {
			skillURL = skillURL + "/" + SkillFileName
		}
	}

	log.Debug("Fetching skill from GitHub: %s", skillURL)

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(skillURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill content: %w", err)
	}

	// 解析 skill
	loader := NewLoader()
	// 从 URL 推断 basePath（远程 skill 的目录结构）
	basePath := filepath.Dir(path)
	if basePath == "." {
		basePath = ""
	}
	skill, err := loader.ParseSkill(data, SourceRemote, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}
	skill.Path = skillURL // 保存完整 URL

	return skill, nil
}

// InstallSkillFromGitHub 从 GitHub 安装 skill 到本地目录
func InstallSkillFromGitHub(repo, path, localDir string) error {
	// 获取 skill
	skill, err := fetchSkillFromGitHub(repo, path)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}

	// 创建本地目录
	skillDir := filepath.Join(localDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// 保存 SKILL.md
	skillPath := filepath.Join(skillDir, SkillFileName)
	
	// 重新获取原始内容（包含 frontmatter）
	parts := strings.Split(repo, "@")
	repoPath := parts[0]
	branch := "main"
	if len(parts) > 1 {
		branch = parts[1]
	}

	skillURL := fmt.Sprintf("%s/%s/%s/%s", GitHubRawBase, repoPath, branch, path)
	if !strings.HasSuffix(path, SkillFileName) {
		if strings.HasSuffix(path, "/") {
			skillURL = skillURL + SkillFileName
		} else {
			skillURL = skillURL + "/" + SkillFileName
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(skillURL)
	if err != nil {
		return fmt.Errorf("failed to fetch skill content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	file, err := os.Create(skillPath)
	if err != nil {
		return fmt.Errorf("failed to create skill file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	// 保存元数据（可选）
	metaPath := filepath.Join(skillDir, ".skill-meta.json")
	meta := map[string]interface{}{
		"name":      skill.Name,
		"source":    "github",
		"repo":      repo,
		"path":      path,
		"installed": time.Now().Format(time.RFC3339),
		"url":       skillURL,
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err == nil {
		os.WriteFile(metaPath, metaData, 0644)
	}

	log.Info("Installed skill %s to %s", skill.Name, skillDir)
	return nil
}

// ListGitHubSkills 列出 GitHub 仓库中的 skills（需要 GitHub API）
func ListGitHubSkills(repo, path string) ([]string, error) {
	// 解析 repo
	parts := strings.Split(repo, "@")
	repoPath := parts[0]
	branch := "main"
	if len(parts) > 1 {
		branch = parts[1]
	}

	// 使用 GitHub API 列出目录内容
	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", GitHubAPIBase, repoPath, path, branch)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	var skills []string
	for _, item := range contents {
		if item.Type == "dir" {
			// 检查目录中是否有 SKILL.md
			skillPath := filepath.Join(path, item.Name, SkillFileName)
			if _, err := fetchSkillFromGitHub(repo, skillPath); err == nil {
				skills = append(skills, item.Name)
			}
		}
	}

	return skills, nil
}
