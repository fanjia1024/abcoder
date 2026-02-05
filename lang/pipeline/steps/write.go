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

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/pipeline"
)

// WriteStep writes state.TargetUniAST.Repo to state.OutputDir using lang.Write.
// No retry; failure is fatal.
type WriteStep struct {
	Compiler string // optional, for lang.WriteOptions
}

// Name implements pipeline.Step.
func (s *WriteStep) Name() string { return "write" }

// Run implements pipeline.Step.
func (s *WriteStep) Run(ctx context.Context, state *pipeline.PipelineState) (*pipeline.PipelineState, error) {
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
		return next, fmt.Errorf("write: %s", rec.Err)
	}
	if state.OutputDir == "" {
		rec.EndedAt = time.Now()
		rec.Status = "failed"
		rec.Err = "OutputDir is empty"
		next.History = append(next.History, rec)
		return next, fmt.Errorf("write: %s", rec.Err)
	}

	err := lang.Write(ctx, state.TargetUniAST.Repo, lang.WriteOptions{
		OutputDir: state.OutputDir,
		Compiler:  s.Compiler,
	})
	rec.EndedAt = time.Now()
	if err != nil {
		rec.Status = "failed"
		rec.Err = err.Error()
		next.History = append(next.History, rec)
		return next, fmt.Errorf("write: %w", err)
	}
	rec.Status = "ok"
	next.History = append(next.History, rec)
	return next, nil
}
