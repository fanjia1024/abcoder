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

package translate

import (
	"context"
	"sync"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// TranslateOptions holds the configuration for code translation
type TranslateOptions struct {
	// SourceLanguage specifies the source language (inferred from Repository if not specified)
	SourceLanguage uniast.Language
	// TargetLanguage specifies the target language (required)
	TargetLanguage uniast.Language
	// TargetModuleName specifies the module name for the target code
	TargetModuleName string
	// OutputDir specifies the output directory for generated code
	OutputDir string
	// LLMTranslator is the callback function for LLM translation (required)
	LLMTranslator LLMTranslateFunc
	// Parallel enables parallel translation (default: false)
	Parallel bool
	// Concurrency specifies the number of concurrent translations
	Concurrency int

	// Post-processing options
	// WebFramework specifies the web framework to integrate: "gin", "echo", "actix", "fastapi", "none"
	WebFramework string
	// GenerateEntryPoint enables generation of entry point if missing (default: true)
	GenerateEntryPoint bool
	// GenerateConfig enables generation of project config files (default: true)
	GenerateConfig bool

	// MaxRetryPerNode is the number of retries per node on translate failure (default: 1). One node = one retry unit.
	MaxRetryPerNode int
	// Result, if non-nil, is filled with FailedNodes and TranslatedIDs after Transform (for observability and resume).
	Result *TranslateResult
	// AlreadyTranslatedIDs is an optional set of source node IDs to skip (success cache for resume).
	AlreadyTranslatedIDs map[string]struct{}
	// ProgressCallback is optional; called after each node is processed (done, total, kind, nodeID) for real-time progress.
	ProgressCallback ProgressCallbackFunc
}

// ProgressCallbackFunc is called after each node is processed. done = processed count, total = CountTranslatableNodes, kind = "type"|"func"|"var", nodeID = Identity.Full().
type ProgressCallbackFunc func(done, total int, currentKind, currentNodeID string)

// FailedNodeInfo records a node that failed translation after max retries.
type FailedNodeInfo struct {
	NodeID   string // source Identity.Full()
	Attempts int
	Err      string
}

// TranslateResult is filled by Transform when opts.Result is non-nil (node-granular outcome and cache).
// TotalNodes and ProcessedNodes are set at end for stats and checkpoint/resume.
type TranslateResult struct {
	FailedNodes     []FailedNodeInfo
	TranslatedIDs   map[string]struct{} // source Identity.Full() of successfully translated nodes
	TotalNodes      int                 // CountTranslatableNodes at start
	ProcessedNodes  int                 // done count at end (success + failed)
	CheckpointPath  string              // reserved: path to checkpoint file for resume
}

// LLMTranslateFunc is the callback function type for LLM translation
// Caller is responsible for implementing the actual LLM call logic
type LLMTranslateFunc func(ctx context.Context, req *LLMTranslateRequest) (*LLMTranslateResponse, error)

// LLMTranslateRequest represents a request to the LLM for code translation
type LLMTranslateRequest struct {
	// SourceLanguage is the source programming language
	SourceLanguage uniast.Language
	// TargetLanguage is the target programming language
	TargetLanguage uniast.Language
	// NodeType is the type of AST node: FUNC, TYPE, VAR
	NodeType uniast.NodeType
	// SourceContent is the source code content (from node's Content field)
	SourceContent string
	// Identity contains the node identification information
	Identity uniast.Identity
	// TypeHints provides type mapping hints to help LLM understand type conversions
	TypeHints *TypeHints
	// Dependencies contains information about already translated dependencies
	Dependencies []DependencyHint
	// Prompt is the complete prompt built by PromptBuilder
	Prompt string
}

// LLMTranslateResponse represents the response from the LLM
type LLMTranslateResponse struct {
	// TargetContent is the translated code content
	TargetContent string
	// TargetSignature is the translated signature (optional, for functions)
	TargetSignature string
	// AdditionalImports contains additional imports required (optional)
	AdditionalImports []uniast.Import
	// Error contains error message if translation failed
	Error string
}

// DependencyHint provides information about an already translated dependency
type DependencyHint struct {
	// SourceIdentity is the identity in source language
	SourceIdentity uniast.Identity
	// TargetIdentity is the identity in target language
	TargetIdentity uniast.Identity
	// TargetSignature is the signature/content in target language
	TargetSignature string
}

// TranslateContext holds the context during translation
type TranslateContext struct {
	// SourceRepo is the source repository being translated
	SourceRepo *uniast.Repository
	// TargetRepo is the target repository being built
	TargetRepo *uniast.Repository
	// Module is the current target module
	Module *uniast.Module
	// Package is the current target package
	Package *uniast.Package
	// TranslatedNodes maps source identity to target identity for already translated nodes
	// Access via AddTranslatedNode/GetTranslatedNode when used from parallel translation.
	TranslatedNodes map[string]uniast.Identity
	// mu protects TranslatedNodes for concurrent read/write
	mu sync.RWMutex
	// Result, if non-nil, receives FailedNodes and TranslatedIDs (one node = one retry unit).
	Result *TranslateResult
	// Progress is optional; when set, ReportNodeDone is called after each node for real-time progress.
	Progress *ProgressState
}

// ProgressState holds total/done and callback for thread-safe progress reporting.
type ProgressState struct {
	Total    int
	done     int
	mu       sync.Mutex
	Callback ProgressCallbackFunc
}

// ReportNodeDone increments done and invokes Callback (if set). Safe for concurrent use.
func (p *ProgressState) ReportNodeDone(currentKind, currentNodeID string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.done++
	done := p.done
	total := p.Total
	cb := p.Callback
	p.mu.Unlock()
	if cb != nil && total > 0 {
		cb(done, total, currentKind, currentNodeID)
	}
}

// Done returns the current processed count. Safe for concurrent use.
func (p *ProgressState) Done() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done
}

// NewTranslateContext creates a new TranslateContext
func NewTranslateContext(srcRepo, targetRepo *uniast.Repository, mod *uniast.Module, pkg *uniast.Package) *TranslateContext {
	return &TranslateContext{
		SourceRepo:      srcRepo,
		TargetRepo:      targetRepo,
		Module:          mod,
		Package:         pkg,
		TranslatedNodes: make(map[string]uniast.Identity),
	}
}

// AddTranslatedNode records a translated node mapping (safe for concurrent use)
func (c *TranslateContext) AddTranslatedNode(sourceID, targetID uniast.Identity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TranslatedNodes[sourceID.Full()] = targetID
}

// GetTranslatedNode returns the target identity for a source identity (safe for concurrent use)
func (c *TranslateContext) GetTranslatedNode(sourceID uniast.Identity) (uniast.Identity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	targetID, ok := c.TranslatedNodes[sourceID.Full()]
	return targetID, ok
}
