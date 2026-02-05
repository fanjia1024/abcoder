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
	"fmt"
	"time"
)

// Pipeline runs steps in sequence with retry/rollback driven by the Agent.
type Pipeline struct {
	Steps []Step
	Agent Agent
}

// Run executes all steps. For each step it may retry or rollback based on
// Agent.OnStepFailure. State is mutated only via applySnapshot and rollback.
func (p *Pipeline) Run(ctx context.Context, st *PipelineState) error {
	if p.Agent == nil {
		p.Agent = &DefaultAgent{MaxRetry: 1}
	}
	for _, step := range p.Steps {
		if err := p.runStep(ctx, step, st); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) runStep(ctx context.Context, step Step, st *PipelineState) error {
	// Save current TargetUniAST so we can rollback to it on DecisionRollback.
	prevTargetUniAST := st.TargetUniAST

	attempt := 0
	for {
		attempt++
		result, err := step.Run(ctx, st)
		if err == nil && result != nil && result.Status == StepOK {
			if result.Snapshot != nil {
				applySnapshot(st, result.Snapshot)
			}
			st.History = append(st.History, StepRecord{
				StepName: step.Name(),
				Attempt:  attempt,
				Status:   StepOK,
				Time:     time.Now(),
			})
			return nil
		}

		// Build result for Agent if step returned nil result
		if result == nil {
			result = &StepResult{Status: StepFailed, Recoverable: true}
		}
		if result.Status == StepOK {
			result = &StepResult{Status: StepFailed, Recoverable: false}
		}

		st.History = append(st.History, StepRecord{
			StepName: step.Name(),
			Attempt:  attempt,
			Status:   result.Status,
			Error:    errStr(err),
			Time:     time.Now(),
		})

		decision := p.Agent.OnStepFailure(ctx, step, st, result, attempt)
		switch decision {
		case DecisionRetry:
			continue
		case DecisionRollback:
			rollback(st, prevTargetUniAST)
			continue
		case DecisionAbort:
			if err != nil {
				return fmt.Errorf("step %s: %w", step.Name(), err)
			}
			return fmt.Errorf("step %s failed (abort)", step.Name())
		}
	}
}

// applySnapshot updates state from a step-produced snapshot by kind.
func applySnapshot(st *PipelineState, snap *Snapshot) {
	if st == nil || snap == nil {
		return
	}
	switch snap.Kind {
	case "ast":
		st.SourceAST = snap
	case "source-uniast":
		st.SourceUniAST = snap
	case "target-uniast":
		st.TargetUniAST = snap
	}
}

// rollback restores TargetUniAST to the previous snapshot (e.g. before TranslateStep).
func rollback(st *PipelineState, prevTargetUniAST *Snapshot) {
	if st == nil {
		return
	}
	st.TargetUniAST = prevTargetUniAST
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
