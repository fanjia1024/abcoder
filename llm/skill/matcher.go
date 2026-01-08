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
	"unicode"
)

// Matcher 负责根据用户输入匹配最相关的 skills
type Matcher struct {
	// 可以扩展：添加 LLM 支持的语义匹配
}

// NewMatcher 创建新的 Matcher
func NewMatcher() *Matcher {
	return &Matcher{}
}

// Match 根据查询字符串匹配 skills，返回按相关性排序的结果
func (m *Matcher) Match(query string, skills []*Skill) []*Skill {
	if query == "" {
		return skills
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryWords := m.tokenize(query)

	// 计算每个 skill 的匹配分数
	type scoredSkill struct {
		skill *Skill
		score float64
	}

	scored := make([]scoredSkill, 0, len(skills))
	for _, skill := range skills {
		score := m.calculateScore(query, queryWords, skill)
		if score > 0 {
			scored = append(scored, scoredSkill{skill: skill, score: score})
		}
	}

	// 按分数排序（降序）
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 提取 skills
	result := make([]*Skill, 0, len(scored))
	for _, s := range scored {
		result = append(result, s.skill)
	}

	return result
}

// MatchMetadata 根据查询字符串匹配 skill metadata，返回按相关性排序的结果
func (m *Matcher) MatchMetadata(query string, metadataList []*SkillMetadata) []*SkillMetadata {
	if query == "" {
		return metadataList
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryWords := m.tokenize(query)

	// 计算每个 metadata 的匹配分数
	type scoredMeta struct {
		meta  *SkillMetadata
		score float64
	}

	scored := make([]scoredMeta, 0, len(metadataList))
	for _, meta := range metadataList {
		score := m.calculateMetadataScore(query, queryWords, meta)
		if score > 0 {
			scored = append(scored, scoredMeta{meta: meta, score: score})
		}
	}

	// 按分数排序（降序）
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 提取 metadata
	result := make([]*SkillMetadata, 0, len(scored))
	for _, s := range scored {
		result = append(result, s.meta)
	}

	return result
}

// calculateMetadataScore 计算 metadata 与查询的匹配分数
func (m *Matcher) calculateMetadataScore(query string, queryWords []string, meta *SkillMetadata) float64 {
	var score float64

	// 1. 名称完全匹配（最高分）
	nameLower := strings.ToLower(meta.Name)
	if nameLower == query {
		score += 100.0
	} else if strings.Contains(nameLower, query) {
		score += 50.0
	}

	// 2. 名称中的关键词匹配
	for _, word := range queryWords {
		if strings.Contains(nameLower, word) {
			score += 20.0
		}
	}

	// 3. 描述匹配
	descLower := strings.ToLower(meta.Description)
	if strings.Contains(descLower, query) {
		score += 30.0
	}

	// 4. 描述中的关键词匹配
	for _, word := range queryWords {
		if strings.Contains(descLower, word) {
			score += 10.0
		}
	}

	return score
}

// calculateScore 计算 skill 与查询的匹配分数
func (m *Matcher) calculateScore(query string, queryWords []string, skill *Skill) float64 {
	var score float64

	// 1. 名称完全匹配（最高分）
	nameLower := strings.ToLower(skill.Name)
	if nameLower == query {
		score += 100.0
	} else if strings.Contains(nameLower, query) {
		score += 50.0
	}

	// 2. 名称中的关键词匹配
	for _, word := range queryWords {
		if strings.Contains(nameLower, word) {
			score += 20.0
		}
	}

	// 3. 描述匹配
	descLower := strings.ToLower(skill.Description)
	if strings.Contains(descLower, query) {
		score += 30.0
	}

	// 4. 描述中的关键词匹配
	for _, word := range queryWords {
		if strings.Contains(descLower, word) {
			score += 10.0
		}
	}

	// 5. 指令内容匹配（权重较低）
	instructionsLower := strings.ToLower(skill.Instructions)
	for _, word := range queryWords {
		if strings.Contains(instructionsLower, word) {
			score += 5.0
		}
	}

	// 6. 元数据匹配
	for _, value := range skill.Metadata {
		valueLower := strings.ToLower(value)
		if strings.Contains(valueLower, query) {
			score += 15.0
		}
	}

	return score
}

// tokenize 将查询字符串分词
func (m *Matcher) tokenize(text string) []string {
	// 简单的分词：按空格和标点符号分割
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(unicode.ToLower(r))
		} else {
			if current.Len() > 0 {
				word := current.String()
				if len(word) > 2 { // 忽略太短的词
					words = append(words, word)
				}
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		word := current.String()
		if len(word) > 2 {
			words = append(words, word)
		}
	}

	return words
}

// MatchTopN 返回匹配度最高的 N 个 skills
func (m *Matcher) MatchTopN(query string, skills []*Skill, n int) []*Skill {
	matched := m.Match(query, skills)
	if len(matched) > n {
		return matched[:n]
	}
	return matched
}

// MatchMetadataTopN 返回匹配度最高的 N 个 skill metadata
func (m *Matcher) MatchMetadataTopN(query string, metadataList []*SkillMetadata, n int) []*SkillMetadata {
	matched := m.MatchMetadata(query, metadataList)
	if len(matched) > n {
		return matched[:n]
	}
	return matched
}
