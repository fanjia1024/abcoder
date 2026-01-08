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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/llm/log"
)

// ScriptExecutor 脚本执行器（带安全措施）
type ScriptExecutor struct {
	// 安全配置
	AllowedCommands []string      // 白名单命令
	Timeout         time.Duration // 执行超时
	WorkDir         string        // 工作目录

	// 交互式确认
	RequireConfirmation bool // 是否需要用户确认

	// 日志
	Logger func(string, ...interface{}) // 日志函数
}

// ExecutionResult 脚本执行结果
type ExecutionResult struct {
	Output   string        // 标准输出
	Error    string        // 标准错误
	ExitCode int           // 退出码
	Duration time.Duration // 执行时长
}

// NewScriptExecutor 创建新的脚本执行器
func NewScriptExecutor() *ScriptExecutor {
	return &ScriptExecutor{
		AllowedCommands: []string{
			"python", "python3",
			"bash", "sh",
			"node", "nodejs",
			"go", "run",
		},
		Timeout:             30 * time.Second,
		RequireConfirmation: true,
		Logger:              log.Info,
	}
}

// Execute 执行脚本（带安全措施）
func (e *ScriptExecutor) Execute(ctx context.Context, skill *Skill, script string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()

	// 安全检查：验证脚本路径
	scriptPath := filepath.Join(skill.ScriptsDir, script)
	if !strings.HasPrefix(scriptPath, skill.BasePath) {
		return nil, fmt.Errorf("script path outside skill directory: %s", scriptPath)
	}

	// 检查文件是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", scriptPath)
	}

	// 确定执行命令
	ext := filepath.Ext(script)
	var cmd *exec.Cmd
	switch ext {
	case ".py":
		if !e.isCommandAllowed("python3") && !e.isCommandAllowed("python") {
			return nil, fmt.Errorf("python not in allowed commands")
		}
		cmdName := "python3"
		if !e.isCommandAllowed("python3") {
			cmdName = "python"
		}
		cmd = exec.CommandContext(ctx, cmdName, append([]string{scriptPath}, args...)...)
	case ".sh", ".bash":
		if !e.isCommandAllowed("bash") && !e.isCommandAllowed("sh") {
			return nil, fmt.Errorf("bash/sh not in allowed commands")
		}
		cmdName := "bash"
		if !e.isCommandAllowed("bash") {
			cmdName = "sh"
		}
		cmd = exec.CommandContext(ctx, cmdName, scriptPath)
		cmd.Args = append(cmd.Args, args...)
	case ".js":
		if !e.isCommandAllowed("node") && !e.isCommandAllowed("nodejs") {
			return nil, fmt.Errorf("node not in allowed commands")
		}
		cmdName := "node"
		if !e.isCommandAllowed("node") {
			cmdName = "nodejs"
		}
		cmd = exec.CommandContext(ctx, cmdName, append([]string{scriptPath}, args...)...)
	case ".go":
		if !e.isCommandAllowed("go") {
			return nil, fmt.Errorf("go not in allowed commands")
		}
		cmd = exec.CommandContext(ctx, "go", append([]string{"run", scriptPath}, args...)...)
	default:
		return nil, fmt.Errorf("unsupported script type: %s", ext)
	}

	// 设置工作目录
	if e.WorkDir != "" {
		cmd.Dir = e.WorkDir
	} else {
		cmd.Dir = skill.BasePath
	}

	// 限制环境变量（只保留必要的）
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
	}

	// 用户确认
	if e.RequireConfirmation {
		if !e.RequestConfirmation(skill.Name, script, args) {
			return nil, fmt.Errorf("user denied script execution")
		}
	}

	// 记录执行日志
	e.Logger("Executing script: %s %v (skill: %s)", script, args, skill.Name)

	// 执行命令
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := &ExecutionResult{
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = err.Error()
		}
		e.Logger("Script execution failed: %v (duration: %v)", err, duration)
	} else {
		result.ExitCode = 0
		e.Logger("Script execution succeeded (duration: %v)", duration)
	}

	return result, nil
}

// RequestConfirmation 请求用户确认
func (e *ScriptExecutor) RequestConfirmation(skillName, script string, args []string) bool {
	fmt.Printf("\n[Security] Skill '%s' wants to execute:\n", skillName)
	fmt.Printf("  Script: %s\n", script)
	if len(args) > 0 {
		fmt.Printf("  Args: %v\n", args)
	}
	fmt.Print("Allow execution? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// isCommandAllowed 检查命令是否在白名单中
func (e *ScriptExecutor) isCommandAllowed(cmd string) bool {
	for _, allowed := range e.AllowedCommands {
		if allowed == cmd {
			return true
		}
	}
	return false
}

// SetAllowedCommands 设置允许的命令列表
func (e *ScriptExecutor) SetAllowedCommands(commands []string) {
	e.AllowedCommands = commands
}

// SetTimeout 设置执行超时
func (e *ScriptExecutor) SetTimeout(timeout time.Duration) {
	e.Timeout = timeout
}

// SetRequireConfirmation 设置是否需要用户确认
func (e *ScriptExecutor) SetRequireConfirmation(require bool) {
	e.RequireConfirmation = require
}
