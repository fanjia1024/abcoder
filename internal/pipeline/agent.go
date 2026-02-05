// Copyright 2025 ByteDance Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipeline

import (
	"context"
)

// Agent decides what to do on step failure: retry, rollback, or abort.
// The Agent only schedules; it never edits code or AST.
type Agent interface {
	OnStepFailure(
		ctx context.Context,
		step Step,
		st *PipelineState,
		result *StepResult,
		attempt int,
	) AgentDecision
}

// AgentDecision is the action to take after a step failure.
type AgentDecision string

const (
	DecisionRetry    AgentDecision = "retry"
	DecisionRollback AgentDecision = "rollback"
	DecisionAbort    AgentDecision = "abort"
)

// DefaultAgent implements a minimal policy: abort if not recoverable,
// rollback if max retries exceeded, else retry.
type DefaultAgent struct {
	MaxRetry int
}

// OnStepFailure implements Agent.
func (a *DefaultAgent) OnStepFailure(
	ctx context.Context,
	step Step,
	st *PipelineState,
	result *StepResult,
	attempt int,
) AgentDecision {
	if result != nil && !result.Recoverable {
		return DecisionAbort
	}
	if attempt >= a.MaxRetry {
		return DecisionRollback
	}
	return DecisionRetry
}
