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
	"fmt"
	"sync"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/skill"
	"github.com/cloudwego/abcoder/llm/tool"
)

// Coordinator 协调多个 SkillAgent 协作处理任务
type Coordinator struct {
	registry *skill.Registry
	agents   map[string]*SkillAgent // skill name -> agent
	model    llm.ChatModel
	tools    map[string]tool.Tool
	opts     CoordinatorOptions
	mu       sync.RWMutex
}

// CoordinatorOptions 是 Coordinator 的配置选项
type CoordinatorOptions struct {
	ASTsDir  string // AST 目录
	MaxSteps int    // 每个 agent 的最大步数
	Retries  int    // 重试次数
	Timeout  int    // 超时时间（秒）
}

// NewCoordinator 创建新的 Coordinator
func NewCoordinator(ctx context.Context, registry *skill.Registry, model llm.ChatModel, opts CoordinatorOptions) (*Coordinator, error) {
	// 收集所有可用工具
	// 注意：这里我们存储工具名称到工具的映射，用于后续在 skill_agent 中过滤
	allTools := make(map[string]tool.Tool)

	// AST 读取工具
	if opts.ASTsDir != "" {
		astTools := tool.NewASTReadTools(tool.ASTReadToolsOptions{
			RepoASTsDir: opts.ASTsDir,
		})
		// 使用工具名称常量来映射
		allTools[tool.ToolListRepos] = astTools.GetTool(tool.ToolListRepos)
		allTools[tool.ToolGetRepoStructure] = astTools.GetTool(tool.ToolGetRepoStructure)
		allTools[tool.ToolGetASTHierarchy] = astTools.GetTool(tool.ToolGetASTHierarchy)
		allTools[tool.ToolGetTargetLanguageSpec] = astTools.GetTool(tool.ToolGetTargetLanguageSpec)
		allTools[tool.ToolGetPackageStructure] = astTools.GetTool(tool.ToolGetPackageStructure)
		allTools[tool.ToolGetFileStructure] = astTools.GetTool(tool.ToolGetFileStructure)
		allTools[tool.ToolGetASTNode] = astTools.GetTool(tool.ToolGetASTNode)
	}

	// Sequential thinking 工具（通常只有一个，名称固定为 sequential_thinking）
	thinkingTools, err := tool.GetSequentialThinkingTools(ctx)
	if err == nil {
		for _, t := range thinkingTools {
			allTools["sequential_thinking"] = t
		}
	}

	// 注意：AST 翻译工具暂时不在这里添加
	// 因为 ASTTranslateTools 返回的是 InvokableTool，需要特殊处理
	// 如果需要翻译工具，可以在 skill_agent 中单独处理

	coordinator := &Coordinator{
		registry: registry,
		agents:   make(map[string]*SkillAgent),
		model:    model,
		tools:    allTools,
		opts:     opts,
	}

	// 确保 registry 已初始化（发现所有 skills）
	if registry.Count() == 0 {
		if err := registry.Initialize(); err != nil {
			log.Error("Failed to initialize skill registry: %v", err)
		}
	}

	return coordinator, nil
}

// GetAvailableSkillsXML 获取所有可用 skills 的 XML 表示
// 用于注入到系统提示中
func (c *Coordinator) GetAvailableSkillsXML() string {
	return c.registry.GetAvailableSkillsXML()
}

// Process 处理用户输入，自动选择并执行相关 skills
func (c *Coordinator) Process(ctx context.Context, input string) (string, error) {
	// 自动匹配相关 skills
	matchedSkills := c.SelectSkills(input)
	if len(matchedSkills) == 0 {
		return "", fmt.Errorf("no matching skills found for input: %s", input)
	}

	log.Info("Matched %d skills for input: %v", len(matchedSkills), getSkillNames(matchedSkills))

	// 如果只有一个 skill，直接执行
	if len(matchedSkills) == 1 {
		return c.executeSkill(ctx, matchedSkills[0], input)
	}

	// 多个 skills，协调执行
	return c.Orchestrate(ctx, matchedSkills, input)
}

// SelectSkills 根据输入自动选择相关 skills
func (c *Coordinator) SelectSkills(input string) []*skill.Skill {
	return c.registry.Search(input)
}

// executeSkill 执行单个 skill
func (c *Coordinator) executeSkill(ctx context.Context, s *skill.Skill, input string) (string, error) {
	agent, err := c.getOrCreateAgent(ctx, s)
	if err != nil {
		return "", fmt.Errorf("failed to get agent for skill %s: %w", s.Name, err)
	}

	log.Info("Executing skill: %s", s.Name)
	return agent.Call(ctx, input)
}

// getOrCreateAgent 获取或创建 skill 对应的 agent
func (c *Coordinator) getOrCreateAgent(ctx context.Context, s *skill.Skill) (*SkillAgent, error) {
	c.mu.RLock()
	if agent, exists := c.agents[s.Name]; exists {
		c.mu.RUnlock()
		return agent, nil
	}
	c.mu.RUnlock()

	// 创建新 agent
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if agent, exists := c.agents[s.Name]; exists {
		return agent, nil
	}

	agent, err := NewSkillAgent(ctx, SkillAgentOptions{
		Skill:    s,
		Model:    c.model,
		AllTools: c.tools,
		ASTsDir:  c.opts.ASTsDir,
		MaxSteps: c.opts.MaxSteps,
		Retries:  c.opts.Retries,
		Timeout:  c.opts.Timeout,
	})
	if err != nil {
		return nil, err
	}

	c.agents[s.Name] = agent
	return agent, nil
}

// Orchestrate 协调多个 skills 协作处理任务
func (c *Coordinator) Orchestrate(ctx context.Context, skills []*skill.Skill, input string) (string, error) {
	// 简单策略：按顺序执行，每个 skill 处理输入并产生输出
	// 后续可以扩展为更复杂的协作策略（并行、管道等）

	var results []string
	for i, s := range skills {
		log.Info("Executing skill %d/%d: %s", i+1, len(skills), s.Name)
		
		result, err := c.executeSkill(ctx, s, input)
		if err != nil {
			log.Error("Skill %s failed: %v", s.Name, err)
			// 继续执行其他 skills
			continue
		}
		
		results = append(results, fmt.Sprintf("## %s\n\n%s", s.Name, result))
		
		// 将前一个 skill 的输出作为下一个 skill 的输入（可选）
		// input = result
	}

	if len(results) == 0 {
		return "", fmt.Errorf("all skills failed to process input")
	}

	// 合并结果
	finalResult := fmt.Sprintf("Processed by %d skill(s):\n\n%s", len(results), joinResults(results))
	return finalResult, nil
}

// getSkillNames 获取 skill 名称列表
func getSkillNames(skills []*skill.Skill) []string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return names
}

// joinResults 合并多个结果
func joinResults(results []string) string {
	joined := ""
	for i, result := range results {
		if i > 0 {
			joined += "\n\n---\n\n"
		}
		joined += result
	}
	return joined
}

// GetAgent 获取指定 skill 的 agent（如果不存在则创建）
func (c *Coordinator) GetAgent(skillName string) (*SkillAgent, error) {
	c.mu.RLock()
	agent, exists := c.agents[skillName]
	c.mu.RUnlock()

	if exists {
		return agent, nil
	}

	// 激活 skill
	skill, err := c.registry.ActivateSkill(skillName)
	if err != nil {
		return nil, fmt.Errorf("failed to activate skill: %w", err)
	}

	// 创建 agent
	agent, err = NewSkillAgent(context.Background(), SkillAgentOptions{
		Skill:    skill,
		Model:    c.model,
		AllTools: c.tools,
		ASTsDir:  c.opts.ASTsDir,
		MaxSteps: c.opts.MaxSteps,
		Retries:  c.opts.Retries,
		Timeout:  c.opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create skill agent: %w", err)
	}

	// 缓存 agent
	c.mu.Lock()
	c.agents[skillName] = agent
	c.mu.Unlock()

	return agent, nil
}

// ListAgents 列出所有已创建的 agents
func (c *Coordinator) ListAgents() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	names := make([]string, 0, len(c.agents))
	for name := range c.agents {
		names = append(names, name)
	}
	return names
}
