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
	"errors"
	"strings"
	"testing"
)

func TestValidateRepository_Nil(t *testing.T) {
	err := ValidateRepository(nil)
	if err == nil {
		t.Fatal("expected error for nil repository")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil: %v", err)
	}
}

func TestValidateRepository_EmptyName(t *testing.T) {
	repo := NewRepository("")
	mod := NewModule("m", ".", Golang)
	repo.Modules["m"] = mod
	err := ValidateRepository(&repo)
	if err == nil {
		t.Fatal("expected error for empty repository name")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError: %T", err)
	}
}

func TestValidateRepository_NoInternalModules(t *testing.T) {
	repo := NewRepository("myrepo")
	mod := NewModule("ext", "", Golang) // Dir empty => IsExternal() true
	repo.Modules["ext"] = mod
	err := ValidateRepository(&repo)
	if err == nil {
		t.Fatal("expected error when all modules are external")
	}
}

func TestValidateRepository_ValidMinimal(t *testing.T) {
	repo := NewRepository("myrepo")
	mod := NewModule("m", ".", Golang)
	pkg := NewPackage("pkg")
	pkg.Functions["f"] = &Function{
		Identity: NewIdentity("m", "pkg", "f"),
		FileLine: FileLine{File: "a.go", Line: 1},
		Content:  "func f() {}",
	}
	mod.Packages["pkg"] = pkg
	repo.Modules["m"] = mod
	err := ValidateRepository(&repo)
	if err != nil {
		t.Fatalf("expected no error for valid minimal repo: %v", err)
	}
}

func TestValidateRepository_EmptyContentRejected(t *testing.T) {
	repo := NewRepository("myrepo")
	mod := NewModule("m", ".", Golang)
	pkg := NewPackage("pkg")
	pkg.Functions["f"] = &Function{
		Identity: NewIdentity("m", "pkg", "f"),
		FileLine: FileLine{File: "a.go", Line: 1},
		Content:  "", // invalid
	}
	mod.Packages["pkg"] = pkg
	repo.Modules["m"] = mod
	err := ValidateRepository(&repo)
	if err == nil {
		t.Fatal("expected error for empty Content")
	}
	if !strings.Contains(err.Error(), "Content") {
		t.Errorf("error should mention Content: %v", err)
	}
}

func TestValidateRepository_DuplicateNameRejected(t *testing.T) {
	repo := NewRepository("myrepo")
	mod := NewModule("m", ".", Golang)
	pkg := NewPackage("pkg")
	pkg.Functions["Foo"] = &Function{
		Identity: NewIdentity("m", "pkg", "Foo"),
		FileLine: FileLine{File: "a.go", Line: 1},
		Content:  "func Foo() {}",
	}
	pkg.Types["Foo"] = &Type{
		Identity: NewIdentity("m", "pkg", "Foo"),
		FileLine: FileLine{File: "a.go", Line: 2},
		Content:  "type Foo struct{}",
	}
	mod.Packages["pkg"] = pkg
	repo.Modules["m"] = mod
	err := ValidateRepository(&repo)
	if err == nil {
		t.Fatal("expected error for duplicate name in package")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate: %v", err)
	}
}

func TestValidationError_Error(t *testing.T) {
	e := &ValidationError{Errs: []string{"a", "b"}}
	s := e.Error()
	if !strings.Contains(s, "2") || !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Errorf("Error() should list count and messages: %q", s)
	}
	e1 := &ValidationError{Errs: []string{"only one"}}
	if e1.Error() != "only one" {
		t.Errorf("single error should return that message: %q", e1.Error())
	}
}

func TestValidateRepositoryWithResult(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo := NewRepository("myrepo")
		mod := NewModule("m", ".", Golang)
		mod.Packages["pkg"] = NewPackage("pkg")
		mod.Packages["pkg"].Functions["f"] = &Function{
			Identity: NewIdentity("m", "pkg", "f"),
			FileLine: FileLine{File: "a.go", Line: 1},
			Content:  "func f() {}",
		}
		repo.Modules["m"] = mod
		res := ValidateRepositoryWithResult(&repo)
		if !res.Ok {
			t.Fatalf("expected Ok: %+v", res)
		}
		if len(res.Errors) != 0 {
			t.Fatalf("expected no errors: %v", res.Errors)
		}
	})

	t.Run("fatal", func(t *testing.T) {
		repo := NewRepository("myrepo")
		mod := NewModule("m", ".", Golang)
		mod.Packages["pkg"] = NewPackage("pkg")
		mod.Packages["pkg"].Functions["f"] = &Function{
			Identity: NewIdentity("m", "pkg", "f"),
			FileLine: FileLine{File: "a.go", Line: 1},
			Content:  "", // fatal: empty content
		}
		repo.Modules["m"] = mod
		res := ValidateRepositoryWithResult(&repo)
		if res.Ok {
			t.Fatal("expected !Ok")
		}
		if res.Severity != SeverityFatal {
			t.Errorf("expected SeverityFatal, got %q", res.Severity)
		}
		if len(res.Errors) == 0 {
			t.Fatal("expected at least one error")
		}
	})

	t.Run("recoverable", func(t *testing.T) {
		repo := NewRepository("myrepo")
		mod := NewModule("m", ".", Golang)
		mod.Packages["pkg"] = NewPackage("pkg")
		mod.Packages["pkg"].Functions["f"] = &Function{
			Identity: NewIdentity("m", "pkg", "f"),
			FileLine: FileLine{File: "a.go", Line: 1},
			Content:  "x", // very short => recoverable
		}
		repo.Modules["m"] = mod
		res := ValidateRepositoryWithResult(&repo)
		if res.Ok {
			t.Fatal("expected !Ok for very short content")
		}
		if res.Severity != SeverityRecoverable {
			t.Errorf("expected SeverityRecoverable, got %q", res.Severity)
		}
	})
}
