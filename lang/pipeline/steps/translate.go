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
	"strconv"
	"time"

	"github.com/cloudwego/abcoder/lang/pipeline"
	"github.com/cloudwego/abcoder/lang/translate"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// TranslateStep runs LLM-based UniAST transformation with retry and rollback.
// Input: state.SourceUniAST. Output: state.TargetUniAST.
// On validation failure: Recoverable -> retry full transform (up to MaxRetry); Fatal -> rollback to previous TargetUniAST and return.
type TranslateStep struct {
	Options   translate.TranslateOptions
	MaxRetry  int  // per full-repo attempt (1 = no retry)
	Persist   bool // if true and state.OutputDir set, write target_ast_<version>.json after success
}

// Name implements pipeline.Step.
func (s *TranslateStep) Name() string { return "translate" }

// Run implements pipeline.Step.
func (s *TranslateStep) Run(ctx context.Context, state *pipeline.PipelineState) (*pipeline.PipelineState, error) {
	start := time.Now()
	next := state.Clone()
	rec := pipeline.StepRecord{
		StepID:    fmt.Sprintf("%s-%s", state.RunID, s.Name()),
		StepName:  s.Name(),
		StartedAt: start,
	}

	if state.SourceUniAST == nil || state.SourceUniAST.Repo == nil {
		rec.EndedAt = time.Now()
		rec.Status = "failed"
		rec.Err = "SourceUniAST is nil or has no Repo"
		next.History = append(next.History, rec)
		return next, fmt.Errorf("translate: %s", rec.Err)
	}

	opts := s.Options
	opts.SourceLanguage = state.SourceLang
	opts.TargetLanguage = state.TargetLang

	// Keep previous snapshot for rollback on Fatal
	previousTarget := state.TargetUniAST
	maxAttempts := s.MaxRetry
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastValidationErr *ValidationFailedError
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		transformer := translate.NewTransformer(opts)
		targetRepo, err := transformer.Transform(ctx, state.SourceUniAST.Repo)
		if err != nil {
			rec.EndedAt = time.Now()
			rec.Status = "failed"
			rec.Err = err.Error()
			next.History = append(next.History, rec)
			return next, fmt.Errorf("translate (attempt %d): %w", attempt, err)
		}

		res := uniast.ValidateRepositoryWithResult(targetRepo)
		if res.Ok {
			version := strconv.Itoa(len(next.History) + 1)
			next.TargetUniAST = pipeline.NewUniASTSnapshot(version, targetRepo)
			rec.EndedAt = time.Now()
			rec.Status = "ok"
			rec.Snapshot = "target_ast_v" + version
			if s.Persist && next.OutputDir != "" {
				path, err := persistRepoJSON(next.OutputDir, "target_ast_"+version+".json", targetRepo)
				if err == nil {
					next.Artifacts["target_ast_"+version] = pipeline.Artifact{Path: path, Kind: "uniast_json"}
				}
			}
			next.History = append(next.History, rec)
			return next, nil
		}

		if res.Severity == uniast.SeverityFatal {
			// Rollback: restore previous target snapshot
			next.TargetUniAST = previousTarget
			rec.EndedAt = time.Now()
			rec.Status = "failed"
			rec.Err = fmt.Sprintf("validation fatal (attempt %d): %d errors", attempt, len(res.Errors))
			next.History = append(next.History, rec)
			return next, &ValidationFailedError{Result: res}
		}

		// Recoverable: will retry next attempt
		lastValidationErr = &ValidationFailedError{Result: res}
	}

	// All retries exhausted (recoverable failures)
	next.TargetUniAST = previousTarget
	rec.EndedAt = time.Now()
	rec.Status = "failed"
	rec.Err = fmt.Sprintf("validation failed after %d attempts (recoverable)", maxAttempts)
	next.History = append(next.History, rec)
	if lastValidationErr != nil {
		return next, lastValidationErr
	}
	return next, fmt.Errorf("translate: %s", rec.Err)
}
