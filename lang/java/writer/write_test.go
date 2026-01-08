/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package writer

import (
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestWriter_SplitImportsAndCodes(t *testing.T) {
	w := NewWriter(Options{})
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantImps int
	}{
		{
			name:     "simple import",
			src:      "import java.util.List;\n\npublic class Test {}",
			wantCode: "public class Test {}",
			wantImps: 1,
		},
		{
			name:     "multiple imports",
			src:      "import java.util.List;\nimport java.util.Map;\n\npublic class Test {}",
			wantCode: "public class Test {}",
			wantImps: 2,
		},
		{
			name:     "no imports",
			src:      "public class Test {}",
			wantCode: "public class Test {}",
			wantImps: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotImps, err := w.SplitImportsAndCodes(tt.src)
			if err != nil {
				t.Errorf("SplitImportsAndCodes() error = %v", err)
				return
			}
			if gotCode != tt.wantCode {
				t.Errorf("SplitImportsAndCodes() code = %v, want %v", gotCode, tt.wantCode)
			}
			if len(gotImps) != tt.wantImps {
				t.Errorf("SplitImportsAndCodes() imports = %v, want %v", len(gotImps), tt.wantImps)
			}
		})
	}
}

func TestWriter_IdToImport(t *testing.T) {
	w := NewWriter(Options{})
	tests := []struct {
		name string
		id   uniast.Identity
		want string
	}{
		{
			name: "simple package",
			id: uniast.Identity{
				PkgPath: "com/example/model",
			},
			want: "com.example.model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.IdToImport(tt.id)
			if err != nil {
				t.Errorf("IdToImport() error = %v", err)
				return
			}
			if got.Path != tt.want {
				t.Errorf("IdToImport() = %v, want %v", got.Path, tt.want)
			}
		})
	}
}
