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
	_ "embed"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/prompt"
	"github.com/cloudwego/abcoder/llm/tool"
	etool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

type TranslatorOptions struct {
	llm.ModelConfig
	MaxSteps int    `json:"max_steps"`
	ASTsDir  string `json:"asts_dir"`
}

func NewTranslatorAgent(ctx context.Context, opts TranslatorOptions) *llm.ReactAgent {
	log.Debug("NewTranslatorAgent, opts: %+v", opts)

	exeModel := llm.NewChatModel(opts.ModelConfig)
	ast := tool.NewASTReadTools(tool.ASTReadToolsOptions{
		RepoASTsDir: opts.ASTsDir,
	})

	// AST read tools
	ts := ast.GetTools()
	log.Debug("NewTranslatorAgent, get AST tools: %#v", ts)
	tcfg := compose.ToolsNodeConfig{}
	for _, t := range ts {
		tcfg.Tools = append(tcfg.Tools, t.(etool.BaseTool))
	}

	// Translation tools
	translateTools := tool.NewASTTranslateTools(tool.ASTTranslateToolsOptions{
		RepoASTsDir: opts.ASTsDir,
	})
	translateTs := translateTools.GetTools()
	log.Debug("NewTranslatorAgent, get translation tools: %#v", translateTs)
	for _, t := range translateTs {
		tcfg.Tools = append(tcfg.Tools, t.(etool.BaseTool))
	}

	// Sequential thinking tools
	thinkingTools, err := tool.GetSequentialThinkingTools(ctx)
	log.Debug("NewTranslatorAgent, get sequential-thinking tools: %#v", thinkingTools)
	if err != nil {
		panic(err)
	}
	for _, t := range thinkingTools {
		tcfg.Tools = append(tcfg.Tools, t.(etool.BaseTool))
	}

	return llm.NewReactAgent("translator", llm.ReactAgentOptions{
		SysPrompt: prompt.NewTextPrompt(prompt.PromptTranslator),
		AgentConfig: &react.AgentConfig{
			ToolCallingModel: exeModel,
			ToolsConfig:      tcfg,
			MaxStep:          opts.MaxSteps,
		},
		Retries: opts.Retries,
		Timeout: opts.Timeout,
	})
}
