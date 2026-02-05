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
	"time"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// PipelineState is the Agent's single source of truth. All intermediate
// results are carried as snapshots; rollback = restore a previous snapshot.
type PipelineState struct {
	RunID      string
	SourceLang uniast.Language
	TargetLang uniast.Language

	SourceCodePath string
	OutputPath     string

	SourceAST    *Snapshot // optional; nil when going straight to UniAST
	SourceUniAST *Snapshot
	TargetUniAST *Snapshot

	History []StepRecord
}

// StepRecord is an immutable log entry for one step execution.
type StepRecord struct {
	StepName string
	Attempt  int
	Status   StepStatus
	Error    string
	Time     time.Time
}

// StepStatus is the outcome of a step run.
type StepStatus string

const (
	StepOK     StepStatus = "ok"
	StepFailed StepStatus = "failed"
	StepRetry  StepStatus = "retry"
)
