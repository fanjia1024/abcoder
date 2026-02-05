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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// PipelineState is the single ground truth the Agent reads. All intermediate
// results are serializable so the Agent never reasons from in-memory pointers.
type PipelineState struct {
	RunID        string
	SourceLang   uniast.Language
	TargetLang   uniast.Language

	InputPath  string // source path (dir or JSON file) for ParseStep
	OutputDir  string // for WriteStep and artifact persistence

	SourceUniAST *UniASTSnapshot // after Parse + Collect (or load from JSON)
	TargetUniAST *UniASTSnapshot // after Translate; rollback = replace with previous snapshot

	Artifacts map[string]Artifact // e.g. "source_ast.json", "target_ast_v1.json"
	History   []StepRecord
}

// UniASTSnapshot is a versioned, serializable snapshot of a Repository for rollback.
type UniASTSnapshot struct {
	Version string             // e.g. "1" or timestamp
	Hash    string             // content hash for quick equality
	Repo    *uniast.Repository // the actual data; JSON-serializable
}

// StepRecord is an immutable log entry for one step execution.
type StepRecord struct {
	StepID    string
	StepName  string
	StartedAt time.Time
	EndedAt   time.Time
	Status    string // "ok", "failed", "skipped"
	Err       string // if failed
	Snapshot  string // optional ref to Artifact key (e.g. "target_ast_v2")
}

// Artifact is a named blob (path on disk or kind); used for "every step persisted".
type Artifact struct {
	Path string // file path if persisted to disk
	Kind string // "uniast_json", "log", etc.
}

// NewPipelineState returns an initial state with empty snapshots and history.
func NewPipelineState(runID string, sourceLang, targetLang uniast.Language) *PipelineState {
	return &PipelineState{
		RunID:        runID,
		SourceLang:   sourceLang,
		TargetLang:   targetLang,
		Artifacts:    make(map[string]Artifact),
		History:      nil,
		SourceUniAST: nil,
		TargetUniAST: nil,
	}
}

// Clone returns a shallow copy of state. Caller can replace TargetUniAST for rollback.
func (s *PipelineState) Clone() *PipelineState {
	if s == nil {
		return nil
	}
	out := *s
	out.Artifacts = make(map[string]Artifact)
	for k, v := range s.Artifacts {
		out.Artifacts[k] = v
	}
	out.History = append([]StepRecord(nil), s.History...)
	return &out
}

// SaveToFile writes a JSON snapshot of the pipeline state to path (e.g. .abcoder/pipeline_state.json).
// Snapshots' Repo are serialized inline; file may be large. For resume/inspection only.
func (s *PipelineState) SaveToFile(path string) error {
	if s == nil {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// NewUniASTSnapshot creates a snapshot with optional hash. If version is empty, a default is set.
func NewUniASTSnapshot(version string, repo *uniast.Repository) *UniASTSnapshot {
	if version == "" {
		version = "1"
	}
	s := &UniASTSnapshot{Version: version, Repo: repo}
	s.Hash = computeRepoHash(repo)
	return s
}

func computeRepoHash(repo *uniast.Repository) string {
	if repo == nil {
		return ""
	}
	data, err := json.Marshal(repo)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
