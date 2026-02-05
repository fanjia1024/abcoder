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
	"fmt"
)

// RunPipeline runs steps in sequence. Each step receives the state produced by the previous step.
// Returns the final state and the first error encountered, if any.
func RunPipeline(ctx context.Context, state *PipelineState, steps []Step) (*PipelineState, error) {
	if state == nil {
		return nil, fmt.Errorf("pipeline: initial state is nil")
	}
	current := state
	for i, step := range steps {
		if step == nil {
			return current, fmt.Errorf("pipeline: step %d is nil", i)
		}
		next, err := step.Run(ctx, current)
		if err != nil {
			return next, fmt.Errorf("pipeline step %q: %w", step.Name(), err)
		}
		current = next
	}
	return current, nil
}
