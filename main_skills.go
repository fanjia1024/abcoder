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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/agent"
	"github.com/cloudwego/abcoder/llm/skill"
)

// handleSkillsCommand 处理 skills 子命令
func handleSkillsCommand(flags *flag.FlagSet, flagHelp *bool, flagVerbose *bool) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: abcoder skills <subcommand> [options]\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  list                    List all available skills\n")
		fmt.Fprintf(os.Stderr, "  install <repo> <path>   Install skill from GitHub\n")
		fmt.Fprintf(os.Stderr, "  show <name>             Show skill details\n")
		os.Exit(1)
	}

	subcommand := strings.ToLower(os.Args[2])

	// 初始化 registry
	registry := skill.NewRegistry()
	if err := registry.Initialize(); err != nil {
		log.Error("Failed to initialize skill registry: %v", err)
		os.Exit(1)
	}

	switch subcommand {
	case "list":
		// 使用 ListMetadata 只显示元数据（更快）
		metadataList := registry.ListMetadata()
		fmt.Printf("Available skills (%d):\n\n", len(metadataList))
		for _, meta := range metadataList {
			fmt.Printf("  %s (%s)\n", meta.Name, meta.Source.String())
			fmt.Printf("    %s\n", meta.Description)
			fmt.Println()
		}

	case "install":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "Usage: abcoder skills install <repo> <path>\n")
			fmt.Fprintf(os.Stderr, "Example: abcoder skills install anthropics/skills code-reviewer\n")
			os.Exit(1)
		}
		repo := os.Args[3]
		path := os.Args[4]

		// 获取本地 skills 目录
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Error("Failed to get home directory: %v", err)
			os.Exit(1)
		}
		localDir := filepath.Join(homeDir, ".abcoder", "skills")
		if err := os.MkdirAll(localDir, 0755); err != nil {
			log.Error("Failed to create skills directory: %v", err)
			os.Exit(1)
		}

		if err := skill.InstallSkillFromGitHub(repo, path, localDir); err != nil {
			log.Error("Failed to install skill: %v", err)
			os.Exit(1)
		}
		fmt.Printf("Skill installed successfully to %s\n", localDir)

	case "show":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: abcoder skills show <name>\n")
			os.Exit(1)
		}
		name := os.Args[3]
		// 激活 skill 以获取完整内容
		s, err := registry.ActivateSkill(name)
		if err != nil {
			log.Error("Skill '%s' not found: %v", name, err)
			os.Exit(1)
		}

		fmt.Printf("Skill: %s\n", s.Name)
		fmt.Printf("Source: %s\n", s.Source.String())
		fmt.Printf("Description: %s\n\n", s.Description)
		if s.Compatibility != "" {
			fmt.Printf("Compatibility: %s\n\n", s.Compatibility)
		}
		if len(s.AllowedTools) > 0 {
			fmt.Printf("Allowed Tools:\n")
			for _, tool := range s.AllowedTools {
				fmt.Printf("  - %s\n", tool)
			}
			fmt.Println()
		}
		fmt.Printf("Instructions:\n%s\n", s.Instructions)

	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

// runSkillAgent 运行指定 skill 的 agent
func runSkillAgent(ctx context.Context, astsDir, skillName string, aopts agent.AgentOptions) {
	// 初始化 registry
	registry := skill.NewRegistry()
	if err := registry.Initialize(); err != nil {
		log.Error("Failed to initialize skill registry: %v", err)
		os.Exit(1)
	}

	// 获取 skill
	s := registry.Get(skillName)
	if s == nil {
		log.Error("Skill '%s' not found", skillName)
		os.Exit(1)
	}

	// 创建 model
	model := llm.NewChatModel(aopts.Model)

	// 创建 coordinator（用于获取工具和创建 agent）
	coordinator, err := agent.NewCoordinator(ctx, registry, model, agent.CoordinatorOptions{
		ASTsDir:  astsDir,
		MaxSteps: aopts.MaxSteps,
		Retries:  3,
		Timeout:  600,
	})
	if err != nil {
		log.Error("Failed to create coordinator: %v", err)
		os.Exit(1)
	}

	// 获取或创建指定 skill 的 agent
	skillAgent, err := coordinator.GetAgent(skillName)
	if err != nil {
		log.Error("Failed to get skill agent: %v", err)
		os.Exit(1)
	}

	// 运行 REPL
	fmt.Fprintf(os.Stdout, "Hello! I'm ABCoder with skill '%s'. What can I do for you today?\n", skillName)

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		query := strings.TrimSpace(sc.Text())
		if query == "" {
			continue
		}
		if query == "exit" {
			break
		}

		// 使用 skill agent 直接调用
		resp, err := skillAgent.Call(ctx, query)
		if err != nil {
			log.Error("Failed to run agent: %v\n", err)
			continue
		}

		fmt.Fprintf(os.Stdout, "\n%s\n", resp)
	}
}

// runCoordinatorAgent 运行 coordinator agent（自动匹配 skills）
func runCoordinatorAgent(ctx context.Context, astsDir string, aopts agent.AgentOptions) {
	// 初始化 registry
	registry := skill.NewRegistry()
	if err := registry.Initialize(); err != nil {
		log.Error("Failed to initialize skill registry: %v", err)
		os.Exit(1)
	}

	// 创建 model
	model := llm.NewChatModel(aopts.Model)

	// 创建 coordinator
	coordinator, err := agent.NewCoordinator(ctx, registry, model, agent.CoordinatorOptions{
		ASTsDir:  astsDir,
		MaxSteps: aopts.MaxSteps,
		Retries:  3,
		Timeout:  600,
	})
	if err != nil {
		log.Error("Failed to create coordinator: %v", err)
		os.Exit(1)
	}

	// 运行 REPL
	fmt.Fprintf(os.Stdout, "Hello! I'm ABCoder with skills support. What can I do for you today?\n")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}
		if query == "exit" {
			break
		}

		resp, err := coordinator.Process(ctx, query)
		if err != nil {
			log.Error("Failed to process: %v\n", err)
			continue
		}

		fmt.Fprintf(os.Stdout, "\n%s\n", resp)
	}
}
