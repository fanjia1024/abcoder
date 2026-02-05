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

	"github.com/cloudwego/abcoder/internal/pipeline"
	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// WriteStep writes TargetUniAST to disk via lang.Write. Failures are not
// recoverable; the Agent should not intervene on write.
type WriteStep struct {
	Compiler string // optional, for lang.WriteOptions
}

// Name implements pipeline.Step.
func (s *WriteStep) Name() string { return "write-code" }

// Run implements pipeline.Step.
func (s *WriteStep) Run(ctx context.Context, st *pipeline.PipelineState) (*pipeline.StepResult, error) {
	if st.TargetUniAST == nil || st.TargetUniAST.Payload == nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("TargetUniAST is nil or has no Payload")
	}
	if st.OutputPath == "" {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("OutputPath is empty")
	}

	repo, ok := st.TargetUniAST.Payload.(*uniast.Repository)
	if !ok {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, fmt.Errorf("TargetUniAST.Payload is not *uniast.Repository")
	}

	err := lang.Write(ctx, repo, lang.WriteOptions{
		OutputDir: st.OutputPath,
		Compiler:  s.Compiler,
	})
	if err != nil {
		return &pipeline.StepResult{
			Status:      pipeline.StepFailed,
			Recoverable: false,
		}, err
	}

	return &pipeline.StepResult{Status: pipeline.StepOK}, nil
}
