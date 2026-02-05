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

package steps

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/abcoder/lang/pipeline"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// ValidationFailedError is returned by ValidateStep when validation fails.
// Callers can use errors.As to get the result and decide retry (Recoverable) vs rollback (Fatal).
type ValidationFailedError struct {
	Result uniast.ValidationResult
}

func (e *ValidationFailedError) Error() string {
	return fmt.Sprintf("UniAST validation failed: severity=%s, %d errors", e.Result.Severity, len(e.Result.Errors))
}

// ValidateStep runs uniast.ValidateRepositoryWithResult on state.TargetUniAST.
// It does not mutate state; if validation fails it returns ValidationFailedError.
type ValidateStep struct{}

// Name implements pipeline.Step.
func (s *ValidateStep) Name() string { return "validate" }

// Run implements pipeline.Step.
func (s *ValidateStep) Run(ctx context.Context, state *pipeline.PipelineState) (*pipeline.PipelineState, error) {
	start := time.Now()
	next := state.Clone()
	rec := pipeline.StepRecord{
		StepID:    fmt.Sprintf("%s-%s", state.RunID, s.Name()),
		StepName:  s.Name(),
		StartedAt: start,
	}

	if state.TargetUniAST == nil || state.TargetUniAST.Repo == nil {
		rec.EndedAt = time.Now()
		rec.Status = "failed"
		rec.Err = "TargetUniAST is nil or has no Repo"
		next.History = append(next.History, rec)
		return next, fmt.Errorf("validate: %s", rec.Err)
	}

	res := uniast.ValidateRepositoryWithResult(state.TargetUniAST.Repo)
	rec.EndedAt = time.Now()
	if res.Ok {
		rec.Status = "ok"
		next.History = append(next.History, rec)
		return next, nil
	}

	rec.Status = "failed"
	rec.Err = fmt.Sprintf("validation failed: severity=%s, %d errors", res.Severity, len(res.Errors))
	next.History = append(next.History, rec)
	return next, &ValidationFailedError{Result: res}
}
