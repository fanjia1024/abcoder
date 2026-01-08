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
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		wantErr  bool
		errMsg   string
	}{
		{"valid-skill", "valid-skill", false, ""},
		{"test123", "test123", false, ""},
		{"a", "a", false, ""},
		{"valid-skill", "valid-skill", false, ""}, // 正常情况
		{"", "test", true, "cannot be empty"},
		{"toolongname" + strings.Repeat("x", 60), "test", true, "must be 1-64"},
		{"Invalid", "invalid", true, "lowercase"},
		{"-invalid", "-invalid", true, "cannot start"},
		{"invalid-", "invalid-", true, "cannot end"},
		{"in--valid", "in--valid", true, "consecutive"},
		{"test", "different", true, "must match"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name, tt.dirName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateName() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		desc    string
		wantErr bool
	}{
		{"Valid description", false},
		{"A", false}, // 1 char
		{string(make([]byte, 1024)), false}, // 1024 chars
		{"", true},
		{string(make([]byte, 1025)), true}, // 1025 chars
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := ValidateDescription(tt.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDescription() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseAllowedTools(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"tool1 tool2 tool3", []string{"tool1", "tool2", "tool3"}},
		{"Bash(git:*) Read Write", []string{"Bash(git:*)", "Read", "Write"}},
		{"single-tool", []string{"single-tool"}},
		{"", nil},
		{"  tool1  tool2  ", []string{"tool1", "tool2"}},
		{"Tool(Arg1:Value1) Tool(Arg2:Value2)", []string{"Tool(Arg1:Value1)", "Tool(Arg2:Value2)"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseAllowedTools(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseAllowedTools() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseAllowedTools()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
