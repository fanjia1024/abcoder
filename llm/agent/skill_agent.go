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

package agent

import (
	"context"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/prompt"
	"github.com/cloudwego/abcoder/llm/skill"
	"github.com/cloudwego/abcoder/llm/tool"
	etool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

// SkillAgent 是基于 Skill 的 Agent
type SkillAgent struct {
	*llm.ReactAgent
	skill *skill.Skill
	tools map[string]tool.Tool // 根据 skill.AllowedTools 过滤的工具集
}

// SkillAgentOptions 是创建 SkillAgent 的选项
type SkillAgentOptions struct {
	Skill         *skill.Skill
	Model         llm.ChatModel
	AllTools      map[string]tool.Tool // 所有可用工具
	ASTsDir       string               // AST 目录
	MaxSteps      int                  // 最大步数
	Retries       int                  // 重试次数
	Timeout       int                  // 超时时间（秒）
}

// NewSkillAgent 创建新的 SkillAgent
func NewSkillAgent(ctx context.Context, opts SkillAgentOptions) (*SkillAgent, error) {
	log.Debug("NewSkillAgent, skill: %s, opts: %+v", opts.Skill.Name, opts)

	// 根据 skill.AllowedTools 过滤工具
	filteredTools := make(map[string]tool.Tool)
	allowedToolsMap := make(map[string]bool)
	for _, toolName := range opts.Skill.AllowedTools {
		allowedToolsMap[toolName] = true
	}

	// 过滤工具
	for toolName, tool := range opts.AllTools {
		if allowedToolsMap[toolName] {
			filteredTools[toolName] = tool
		}
	}

	// 构建工具配置
	tcfg := compose.ToolsNodeConfig{}
	for _, t := range filteredTools {
		tcfg.Tools = append(tcfg.Tools, t.(etool.BaseTool))
	}

	// 创建 system prompt（结合 skill instructions）
	// 注意：这里可以添加可用 skills 的 XML 格式，但当前 skill agent 只关注单个 skill
	sysPrompt := buildSkillPrompt(opts.Skill)

	// 创建 ReactAgent
	reactAgent := llm.NewReactAgent(opts.Skill.Name, llm.ReactAgentOptions{
		SysPrompt: prompt.NewTextPrompt(sysPrompt),
		AgentConfig: &react.AgentConfig{
			ToolCallingModel: opts.Model,
			ToolsConfig:      tcfg,
			MaxStep:          opts.MaxSteps,
		},
		Retries: opts.Retries,
	})

	return &SkillAgent{
		ReactAgent: reactAgent,
		skill:      opts.Skill,
		tools:      filteredTools,
	}, nil
}

// buildSkillPrompt 构建 skill 的 system prompt
func buildSkillPrompt(s *skill.Skill) string {
	// 使用官方推荐的格式生成提示
	return skill.GenerateSkillInstructionsPrompt(s)
}

// GetSkill 返回关联的 skill
func (a *SkillAgent) GetSkill() *skill.Skill {
	return a.skill
}

// GetTools 返回可用的工具列表
func (a *SkillAgent) GetTools() map[string]tool.Tool {
	return a.tools
}
