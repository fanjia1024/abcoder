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
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// mockStepOK returns StepOK with an optional snapshot.
type mockStepOK struct {
	name    string
	snap    *Snapshot
	payload any
	raw     []byte
}

func (m *mockStepOK) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock-ok"
}

func (m *mockStepOK) Run(ctx context.Context, st *PipelineState) (*StepResult, error) {
	if m.snap != nil {
		return &StepResult{Status: StepOK, Snapshot: m.snap}, nil
	}
	if m.payload != nil && m.raw != nil {
		snap := NewSnapshot("target-uniast", m.payload, m.raw)
		return &StepResult{Status: StepOK, Snapshot: snap}, nil
	}
	return &StepResult{Status: StepOK}, nil
}

// mockStepFail returns a failure with recoverable flag.
type mockStepFail struct {
	recoverable bool
}

func (m *mockStepFail) Name() string { return "mock-fail" }

func (m *mockStepFail) Run(ctx context.Context, st *PipelineState) (*StepResult, error) {
	return &StepResult{
		Status:      StepFailed,
		Recoverable: m.recoverable,
	}, nil
}

func TestPipeline_Run_Success(t *testing.T) {
	ctx := context.Background()
	st := &PipelineState{
		RunID:      "run-1",
		SourceLang: uniast.Golang,
		TargetLang: uniast.Golang,
	}
	payload := "hello"
	raw := []byte("hello")
	snap := NewSnapshot("source-uniast", payload, raw)

	pl := &Pipeline{
		Steps: []Step{&mockStepOK{name: "inject", snap: snap}},
		Agent: &DefaultAgent{MaxRetry: 1},
	}
	err := pl.Run(ctx, st)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st.SourceUniAST == nil {
		t.Fatal("expected SourceUniAST to be set")
	}
	if st.SourceUniAST.Payload != payload {
		t.Errorf("payload: got %v", st.SourceUniAST.Payload)
	}
	if len(st.History) != 1 {
		t.Errorf("expected 1 history record, got %d", len(st.History))
	}
	if st.History[0].Status != StepOK {
		t.Errorf("history status: got %s", st.History[0].Status)
	}
}

func TestPipeline_Run_AbortOnNonRecoverable(t *testing.T) {
	ctx := context.Background()
	st := &PipelineState{RunID: "run-1"}

	pl := &Pipeline{
		Steps: []Step{&mockStepFail{recoverable: false}},
		Agent: &DefaultAgent{MaxRetry: 3},
	}
	err := pl.Run(ctx, st)
	if err == nil {
		t.Fatal("expected error on non-recoverable failure")
	}
}

func TestDefaultAgent_OnStepFailure(t *testing.T) {
	ctx := context.Background()
	agent := &DefaultAgent{MaxRetry: 2}

	t.Run("abort when not recoverable", func(t *testing.T) {
		d := agent.OnStepFailure(ctx, nil, nil, &StepResult{Recoverable: false}, 1)
		if d != DecisionAbort {
			t.Errorf("got %s", d)
		}
	})

	t.Run("retry when recoverable and under max", func(t *testing.T) {
		d := agent.OnStepFailure(ctx, nil, nil, &StepResult{Recoverable: true}, 1)
		if d != DecisionRetry {
			t.Errorf("got %s", d)
		}
	})

	t.Run("rollback when recoverable and at max", func(t *testing.T) {
		d := agent.OnStepFailure(ctx, nil, nil, &StepResult{Recoverable: true}, 2)
		if d != DecisionRollback {
			t.Errorf("got %s", d)
		}
	})
}

func TestApplySnapshot(t *testing.T) {
	st := &PipelineState{}
	snap := NewSnapshot("target-uniast", "x", []byte("x"))
	applySnapshot(st, snap)
	if st.TargetUniAST != snap {
		t.Error("TargetUniAST not set")
	}
	st.TargetUniAST = nil
	snap2 := NewSnapshot("source-uniast", "y", []byte("y"))
	applySnapshot(st, snap2)
	if st.SourceUniAST != snap2 {
		t.Error("SourceUniAST not set")
	}
}

func TestRollback(t *testing.T) {
	prev := NewSnapshot("target-uniast", "prev", []byte("prev"))
	st := &PipelineState{TargetUniAST: NewSnapshot("target-uniast", "cur", []byte("cur"))}
	rollback(st, prev)
	if st.TargetUniAST != prev {
		t.Error("rollback did not restore")
	}
}
