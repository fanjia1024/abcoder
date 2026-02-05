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
	"path/filepath"
	"testing"

	"github.com/cloudwego/abcoder/internal/pipeline"
	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestParseStep_Run_WithTestdataJSON(t *testing.T) {
	astPath := filepath.Join(testutils.GetTestDataRoot(), "asts", "localsession.json")
	st := &pipeline.PipelineState{
		RunID:          "test",
		SourceCodePath: astPath,
		SourceLang:     uniast.Golang,
		TargetLang:     uniast.Golang,
	}

	step := &ParseStep{}
	result, err := step.Run(context.Background(), st)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != pipeline.StepOK {
		t.Errorf("status: got %s", result.Status)
	}
	if result.Snapshot == nil {
		t.Fatal("expected Snapshot")
	}
	if result.Snapshot.Kind != "source-uniast" {
		t.Errorf("snapshot kind: got %s", result.Snapshot.Kind)
	}
	if _, ok := result.Snapshot.Payload.(*uniast.Repository); !ok {
		t.Errorf("Payload type: %T", result.Snapshot.Payload)
	}
}
