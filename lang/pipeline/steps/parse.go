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
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/abcoder/lang/pipeline"
	"github.com/cloudwego/abcoder/lang/translate"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// ParseStep loads source from InputPath (directory or JSON file) and sets SourceUniAST.
// No retry; failure is fatal.
type ParseStep struct {
	// PersistSourceJSON if non-empty writes source repo JSON to this path under state.OutputDir (or current dir).
	PersistSourceJSON string
}

// Name implements pipeline.Step.
func (s *ParseStep) Name() string { return "parse" }

// Run implements pipeline.Step.
func (s *ParseStep) Run(ctx context.Context, state *pipeline.PipelineState) (*pipeline.PipelineState, error) {
	start := time.Now()
	next := state.Clone()
	rec := pipeline.StepRecord{
		StepID:    fmt.Sprintf("%s-%s", state.RunID, s.Name()),
		StepName:  s.Name(),
		StartedAt: start,
	}

	if state.InputPath == "" {
		rec.EndedAt = time.Now()
		rec.Status = "failed"
		rec.Err = "InputPath is empty"
		next.History = append(next.History, rec)
		return next, fmt.Errorf("parse: %s", rec.Err)
	}

	repo, err := translate.LoadRepository(ctx, state.InputPath, state.SourceLang)
	if err != nil {
		rec.EndedAt = time.Now()
		rec.Status = "failed"
		rec.Err = err.Error()
		next.History = append(next.History, rec)
		return next, fmt.Errorf("parse: %w", err)
	}

	if state.SourceLang == uniast.Unknown {
		next.SourceLang = inferLanguage(repo)
	} else {
		next.SourceLang = state.SourceLang
	}

	next.SourceUniAST = pipeline.NewUniASTSnapshot("1", repo)
	rec.EndedAt = time.Now()
	rec.Status = "ok"

	if s.PersistSourceJSON != "" {
		dir := state.OutputDir
		if dir == "" {
			dir = filepath.Dir(state.InputPath)
		}
		path, err := persistRepoJSON(dir, s.PersistSourceJSON, repo)
		if err != nil {
			rec.Err = err.Error()
			// non-fatal: still append artifact if path was set
		} else {
			next.Artifacts["source_ast.json"] = pipeline.Artifact{Path: path, Kind: "uniast_json"}
			rec.Snapshot = "source_ast.json"
		}
	}
	next.History = append(next.History, rec)
	return next, nil
}

func inferLanguage(repo *uniast.Repository) uniast.Language {
	for _, mod := range repo.Modules {
		if !mod.IsExternal() && mod.Language != uniast.Unknown {
			return mod.Language
		}
	}
	return uniast.Unknown
}

// persistRepoJSON writes repo as JSON to dir/name; returns the full path or error.
func persistRepoJSON(dir, name string, repo *uniast.Repository) (string, error) {
	if repo == nil {
		return "", fmt.Errorf("repo is nil")
	}
	data, err := json.MarshalIndent(repo, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}
