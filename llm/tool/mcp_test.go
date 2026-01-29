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

package tool

import (
	"context"
	"fmt"
	"testing"
)

func TestMCPClient(t *testing.T) {
	cli, err := NewMCPClient(MCPConfig{
		Type:   MCPTypeStdio,
		Command: "npx",
		Args: []string{
			"-y",
			"@modelcontextprotocol/server-sequential-thinking",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	tools, err := cli.GetTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", tools)
}

func TestGetGitTools(t *testing.T) {
	got, err := GetGitTools(context.Background())
	if err != nil {
		t.Skipf("GetGitTools requires git MCP (uvx/mcp-server-git): %v", err)
	}
	if got == nil {
		t.Fatal("GetGitTools() returned nil slice")
	}
	// When MCP is available, tools may be non-empty; we only assert a valid response.
	if len(got) > 0 {
		t.Logf("GetGitTools() returned %d tool(s)", len(got))
	}
}
