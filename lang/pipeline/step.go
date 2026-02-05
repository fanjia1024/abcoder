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
)

// Step is one unit of work in the pipeline. Each step takes the current state
// and returns a new state (or error). The Agent only schedules steps and reads
// state; it never edits code or AST.
type Step interface {
	Name() string
	Run(ctx context.Context, state *PipelineState) (*PipelineState, error)
}
