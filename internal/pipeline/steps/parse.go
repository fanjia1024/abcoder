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

// ParseStep loads source from SourceCodePath (directory or JSON file) and
// produces a source-uniast snapshot. Parse failures are not recoverable.
type ParseStep struct{}

// Name implements pipeline.Step.
func (s *ParseStep) Name() string { return "parse" }

// Run implements pipeline.Step.
func (s *ParseStep) Run(ctx context.Context, st *pipeline.PipelineState) (*pipeline.StepResult, error) {
	if st.SourceCodePath == "" {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("SourceCodePath is empty")
	}

	repo, err := translate.LoadRepository(ctx, st.SourceCodePath, st.SourceLang)
	if err != nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, err
	}

	if st.SourceLang == uniast.Unknown {
		st.SourceLang = inferLanguage(repo)
	}

	raw, err := json.Marshal(repo)
	if err != nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, err
	}

	snap := pipeline.NewSnapshot("source-uniast", repo, raw)
	return &pipeline.StepResult{
		Status:   pipeline.StepOK,
		Snapshot: snap,
	}, nil
}

func inferLanguage(repo *uniast.Repository) uniast.Language {
	for _, mod := range repo.Modules {
		if !mod.IsExternal() && mod.Language != uniast.Unknown {
			return mod.Language
		}
	}
	return uniast.Unknown
}
