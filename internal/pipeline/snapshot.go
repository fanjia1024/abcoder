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
)

// Snapshot is an immutable, versioned snapshot of an intermediate artifact.
// Each LLM output produces a new Snapshot; rollback restores a previous one.
type Snapshot struct {
	Kind    string // e.g. "source-uniast", "target-uniast"
	Hash    string // hex-encoded sha256 of raw bytes
	Payload any    // e.g. *uniast.Repository
}

// NewSnapshot creates a snapshot from a payload and its serialized form.
// raw is used only to compute the hash (e.g. json.Marshal(repo)).
func NewSnapshot(kind string, payload any, raw []byte) *Snapshot {
	h := sha256.Sum256(raw)
	return &Snapshot{
		Kind:    kind,
		Hash:    hex.EncodeToString(h[:]),
		Payload: payload,
	}
}
