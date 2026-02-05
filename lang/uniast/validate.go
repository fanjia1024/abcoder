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

package uniast

import (
	"fmt"
	"strings"
)

// ValidationSeverity indicates whether a validation failure is recoverable (retry) or fatal (rollback).
type ValidationSeverity string

const (
	SeverityFatal       ValidationSeverity = "fatal"
	SeverityRecoverable ValidationSeverity = "recoverable"
)

// ValidationErrorItem is a single validation issue with optional node identity for reporting.
type ValidationErrorItem struct {
	Message  string
	Severity ValidationSeverity
	NodeID   string // optional, e.g. Identity.Full() for node-level reporting
}

// ValidationResult is the result of ValidateRepositoryWithResult. Agent uses it to decide retry vs rollback.
type ValidationResult struct {
	Ok       bool
	Errors   []ValidationErrorItem
	Severity ValidationSeverity // overall: if any Fatal, result is Fatal; else Recoverable
}

// ValidationError collects multiple validation failures so callers can see all issues at once.
type ValidationError struct {
	Errs []string
}

func (e *ValidationError) Error() string {
	if len(e.Errs) == 0 {
		return "validation failed"
	}
	if len(e.Errs) == 1 {
		return e.Errs[0]
	}
	return fmt.Sprintf("validation failed (%d errors): %s", len(e.Errs), strings.Join(e.Errs, "; "))
}

// ValidateRepository checks that a Repository is well-formed before passing to the Writer.
// It performs schema checks (required fields, non-empty content) and simple invariants
// (e.g. unique node names per package). Validation failure causes a fatal error so that
// bad LLM output is never written to disk.
func ValidateRepository(repo *Repository) error {
	res := ValidateRepositoryWithResult(repo)
	if res.Ok {
		return nil
	}
	var errs []string
	for _, e := range res.Errors {
		errs = append(errs, e.Message)
	}
	return &ValidationError{Errs: errs}
}

// ValidateRepositoryWithResult returns a structured result so the Agent can decide retry (Recoverable) vs rollback (Fatal).
func ValidateRepositoryWithResult(repo *Repository) ValidationResult {
	if repo == nil {
		return ValidationResult{
			Ok:       false,
			Errors:   []ValidationErrorItem{{Message: "repository is nil", Severity: SeverityFatal}},
			Severity: SeverityFatal,
		}
	}
	var items []ValidationErrorItem

	// Schema: repository must have a name and at least one non-external module
	if repo.Name == "" {
		items = append(items, ValidationErrorItem{Message: "repository name is empty", Severity: SeverityFatal})
	}
	if repo.Modules == nil {
		items = append(items, ValidationErrorItem{Message: "repository Modules is nil", Severity: SeverityFatal})
	} else {
		hasInternal := false
		for modName, mod := range repo.Modules {
			if mod == nil {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("module %q is nil", modName),
					Severity: SeverityFatal,
				})
				continue
			}
			if !mod.IsExternal() {
				hasInternal = true
			}
			modItems := validateModuleWithResult(modName, mod)
			items = append(items, modItems...)
		}
		if !hasInternal && len(repo.Modules) > 0 {
			items = append(items, ValidationErrorItem{
				Message:  "repository has no internal modules (all are external)",
				Severity: SeverityFatal,
			})
		}
	}

	if len(items) == 0 {
		return ValidationResult{Ok: true}
	}
	severity := SeverityFatal
	for _, it := range items {
		if it.Severity == SeverityFatal {
			severity = SeverityFatal
			break
		}
		severity = SeverityRecoverable
	}
	return ValidationResult{Ok: false, Errors: items, Severity: severity}
}

func validateModuleWithResult(modName string, mod *Module) []ValidationErrorItem {
	var items []ValidationErrorItem
	if mod.Name == "" {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("module %q has empty Name", modName),
			Severity: SeverityFatal,
		})
	}
	if mod.Packages == nil {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("module %q has nil Packages", modName),
			Severity: SeverityFatal,
		})
		return items
	}
	for pkgPath, pkg := range mod.Packages {
		if pkg == nil {
			items = append(items, ValidationErrorItem{
				Message:  fmt.Sprintf("module %q package %q is nil", modName, pkgPath),
				Severity: SeverityFatal,
			})
			continue
		}
		items = append(items, validatePackageWithResult(modName, pkgPath, pkg)...)
	}
	return items
}

func validatePackageWithResult(modName, pkgPath string, pkg *Package) []ValidationErrorItem {
	var items []ValidationErrorItem
	seenNames := make(map[string]string)

	if pkg.Functions != nil {
		for name, f := range pkg.Functions {
			if f == nil {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s function %q is nil", modName, pkgPath, name),
					Severity: SeverityFatal,
				})
				continue
			}
			items = append(items, validateIdentityAndContentWithResult("function", modName, pkgPath, name, f.Identity, f.Content, f.FileLine)...)
			if prev, ok := seenNames[f.Name]; ok {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s duplicate name %q (already used as %s)", modName, pkgPath, f.Name, prev),
					Severity: SeverityFatal,
					NodeID:   f.Identity.Full(),
				})
			} else {
				seenNames[f.Name] = "function"
			}
		}
	}

	if pkg.Types != nil {
		for name, t := range pkg.Types {
			if t == nil {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s type %q is nil", modName, pkgPath, name),
					Severity: SeverityFatal,
				})
				continue
			}
			items = append(items, validateIdentityAndContentWithResult("type", modName, pkgPath, name, t.Identity, t.Content, t.FileLine)...)
			if prev, ok := seenNames[t.Name]; ok {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s duplicate name %q (already used as %s)", modName, pkgPath, t.Name, prev),
					Severity: SeverityFatal,
					NodeID:   t.Identity.Full(),
				})
			} else {
				seenNames[t.Name] = "type"
			}
		}
	}

	if pkg.Vars != nil {
		for name, v := range pkg.Vars {
			if v == nil {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s var %q is nil", modName, pkgPath, name),
					Severity: SeverityFatal,
				})
				continue
			}
			items = append(items, validateIdentityAndContentWithResult("var", modName, pkgPath, name, v.Identity, v.Content, v.FileLine)...)
			if prev, ok := seenNames[v.Name]; ok {
				items = append(items, ValidationErrorItem{
					Message:  fmt.Sprintf("package %s#%s duplicate name %q (already used as %s)", modName, pkgPath, v.Name, prev),
					Severity: SeverityFatal,
					NodeID:   v.Identity.Full(),
				})
			} else {
				seenNames[v.Name] = "var"
			}
		}
	}

	return items
}

// minContentLengthForRecoverable: content shorter than this may be incomplete LLM output (recoverable by retry).
const minContentLengthForRecoverable = 3

func validateIdentityAndContentWithResult(kind, modName, pkgPath, keyName string, id Identity, content string, fl FileLine) []ValidationErrorItem {
	var items []ValidationErrorItem
	nodeID := id.Full()
	if id.ModPath == "" {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has empty Identity.ModPath", modName, pkgPath, kind, keyName),
			Severity: SeverityFatal,
			NodeID:   nodeID,
		})
	}
	if id.PkgPath == "" {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has empty Identity.PkgPath", modName, pkgPath, kind, keyName),
			Severity: SeverityFatal,
			NodeID:   nodeID,
		})
	}
	if id.Name == "" {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has empty Identity.Name", modName, pkgPath, kind, keyName),
			Severity: SeverityFatal,
			NodeID:   nodeID,
		})
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has empty or whitespace-only Content", modName, pkgPath, kind, keyName),
			Severity: SeverityFatal,
			NodeID:   nodeID,
		})
	} else if len(trimmed) < minContentLengthForRecoverable {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has very short Content (possible incomplete LLM output)", modName, pkgPath, kind, keyName),
			Severity: SeverityRecoverable,
			NodeID:   nodeID,
		})
	}
	if fl.Line < 0 {
		items = append(items, ValidationErrorItem{
			Message:  fmt.Sprintf("package %s#%s %s %q has negative FileLine.Line", modName, pkgPath, kind, keyName),
			Severity: SeverityFatal,
			NodeID:   nodeID,
		})
	}
	return items
}
