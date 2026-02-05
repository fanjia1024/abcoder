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
	"encoding/json"
	"fmt"

	"github.com/cloudwego/abcoder/internal/pipeline"
	"github.com/cloudwego/abcoder/lang/translate"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// TranslateStep runs UniAST → UniAST transformation via the existing translator.
// LLM failures and validation failures are Recoverable so the Agent can retry.
type TranslateStep struct {
	Options translate.TranslateOptions
}

// Name implements pipeline.Step.
func (s *TranslateStep) Name() string { return "translate-uniast" }

// Run implements pipeline.Step.
func (s *TranslateStep) Run(ctx context.Context, st *pipeline.PipelineState) (*pipeline.StepResult, error) {
	if st.SourceUniAST == nil || st.SourceUniAST.Payload == nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("SourceUniAST is nil or has no Payload")
	}

	repo, ok := st.SourceUniAST.Payload.(*uniast.Repository)
	if !ok {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("SourceUniAST.Payload is not *uniast.Repository")
	}

	opts := s.Options
	opts.SourceLanguage = st.SourceLang
	opts.TargetLanguage = st.TargetLang

	targetRepo, err := translate.TranslateAST(ctx, repo, opts)
	if err != nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: true,
		}, err
	}

	res := uniast.ValidateRepositoryWithResult(targetRepo)
	if !res.Ok {
		recoverable := res.Severity == uniast.SeverityRecoverable
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: recoverable,
		}, fmt.Errorf("validation failed: %w", &validationResultErr{res})
	}

	raw, err := json.Marshal(targetRepo)
	if err != nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, err
	}

	snap := pipeline.NewSnapshot("target-uniast", targetRepo, raw)
	return &pipeline.StepResult{
		Status:   pipeline.StepOK,
		Snapshot: snap,
	}, nil
}

type validationResultErr struct {
	Result uniast.ValidationResult
}

func (e *validationResultErr) Error() string {
	return fmt.Sprintf("severity=%s, %d errors", e.Result.Severity, len(e.Result.Errors))
}
