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

package lang

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

func TestWrite_UnsupportedLanguage(t *testing.T) {
	repo := &uniast.Repository{
		Modules: map[string]*uniast.Module{
			"test": {
				Name:     "test",
				Dir:      "test",
				Language: uniast.Language("unsupported"),
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: t.TempDir(),
	})

	if err == nil {
		t.Error("expected error for unsupported language, got nil")
	}
}

func TestWrite_SkipExternalModules(t *testing.T) {
	repo := &uniast.Repository{
		Modules: map[string]*uniast.Module{
			"external": {
				Name:     "external",
				Dir:      "", // Empty dir means external
				Language: uniast.Golang,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: t.TempDir(),
	})

	if err != nil {
		t.Errorf("expected no error for external modules, got %v", err)
	}
}

func TestWrite_GoModule(t *testing.T) {
	tmpDir := t.TempDir()
	funcId := uniast.NewIdentity("github.com/example/test", "github.com/example/test", "main")
	repo := &uniast.Repository{
		Name: "test-repo",
		Modules: map[string]*uniast.Module{
			"github.com/example/test": {
				Name:     "github.com/example/test",
				Dir:      "test",
				Language: uniast.Golang,
				Packages: map[uniast.PkgPath]*uniast.Package{
					"github.com/example/test": {
						PkgPath: "github.com/example/test",
						IsMain:  true,
						Functions: map[string]*uniast.Function{
							"main": {
								Identity: funcId,
								Content:  "func main() {\n\tprintln(\"Hello\")\n}",
							},
						},
						Types: map[string]*uniast.Type{},
						Vars:  map[string]*uniast.Var{},
					},
				},
			},
		},
		Graph: map[string]*uniast.Node{
			funcId.Full(): {
				Identity: funcId,
				Type:     uniast.FUNC,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
		Compiler:  "true", // Use 'true' to skip go mod tidy
	})

	if err != nil {
		t.Errorf("expected no error for Go module, got %v", err)
	}

	// Check if go.mod was created
	goModPath := filepath.Join(tmpDir, "test", "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("expected go.mod to be created at %s", goModPath)
	}
}

func TestWrite_JavaModule(t *testing.T) {
	tmpDir := t.TempDir()
	typeId := uniast.NewIdentity("com.example:test:1.0.0", "com.example.test", "Main")
	repo := &uniast.Repository{
		Name: "test-repo",
		Modules: map[string]*uniast.Module{
			"com.example:test:1.0.0": {
				Name:     "com.example:test:1.0.0",
				Dir:      "test",
				Language: uniast.Java,
				Packages: map[uniast.PkgPath]*uniast.Package{
					"com.example.test": {
						PkgPath:   "com.example.test",
						Functions: map[string]*uniast.Function{},
						Types: map[string]*uniast.Type{
							"Main": {
								Identity: typeId,
								TypeKind: uniast.TypeKindStruct,
								Content:  "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello\");\n    }\n}",
							},
						},
						Vars: map[string]*uniast.Var{},
					},
				},
			},
		},
		Graph: map[string]*uniast.Node{
			typeId.Full(): {
				Identity: typeId,
				Type:     uniast.TYPE,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
	})

	if err != nil {
		t.Errorf("expected no error for Java module, got %v", err)
	}

	// Check if pom.xml was created
	pomPath := filepath.Join(tmpDir, "test", "pom.xml")
	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		t.Errorf("expected pom.xml to be created at %s", pomPath)
	}
}

func TestWrite_PythonModule(t *testing.T) {
	tmpDir := t.TempDir()
	funcId := uniast.NewIdentity("test_module", "test_module", "main")
	repo := &uniast.Repository{
		Name: "test-repo",
		Modules: map[string]*uniast.Module{
			"test_module": {
				Name:     "test_module",
				Dir:      "test",
				Language: uniast.Python,
				Packages: map[uniast.PkgPath]*uniast.Package{
					"test_module": {
						PkgPath: "test_module",
						Functions: map[string]*uniast.Function{
							"main": {
								Identity: funcId,
								Content:  "def main():\n    print(\"Hello\")",
							},
						},
						Types: map[string]*uniast.Type{},
						Vars:  map[string]*uniast.Var{},
					},
				},
			},
		},
		Graph: map[string]*uniast.Node{
			funcId.Full(): {
				Identity: funcId,
				Type:     uniast.FUNC,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
	})

	if err != nil {
		t.Errorf("expected no error for Python module, got %v", err)
	}

	// Check if pyproject.toml was created
	pyprojectPath := filepath.Join(tmpDir, "test", "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
		t.Errorf("expected pyproject.toml to be created at %s", pyprojectPath)
	}
}

func TestWrite_RustModule(t *testing.T) {
	tmpDir := t.TempDir()
	funcId := uniast.NewIdentity("test_crate", "test_crate", "main")
	repo := &uniast.Repository{
		Name: "test-repo",
		Modules: map[string]*uniast.Module{
			"test_crate": {
				Name:     "test_crate",
				Dir:      "test",
				Language: uniast.Rust,
				Packages: map[uniast.PkgPath]*uniast.Package{
					"test_crate": {
						PkgPath: "test_crate",
						IsMain:  true,
						Functions: map[string]*uniast.Function{
							"main": {
								Identity: funcId,
								Content:  "fn main() {\n    println!(\"Hello\");\n}",
							},
						},
						Types: map[string]*uniast.Type{},
						Vars:  map[string]*uniast.Var{},
					},
				},
			},
		},
		Graph: map[string]*uniast.Node{
			funcId.Full(): {
				Identity: funcId,
				Type:     uniast.FUNC,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
	})

	if err != nil {
		t.Errorf("expected no error for Rust module, got %v", err)
	}

	// Check if Cargo.toml was created
	cargoPath := filepath.Join(tmpDir, "test", "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		t.Errorf("expected Cargo.toml to be created at %s", cargoPath)
	}
}

func TestWrite_CxxModule(t *testing.T) {
	tmpDir := t.TempDir()
	funcId := uniast.NewIdentity("test_cxx", "test_cxx", "main")
	repo := &uniast.Repository{
		Name: "test-repo",
		Modules: map[string]*uniast.Module{
			"test_cxx": {
				Name:     "test_cxx",
				Dir:      "test",
				Language: uniast.Cxx,
				Packages: map[uniast.PkgPath]*uniast.Package{
					"test_cxx": {
						PkgPath: "test_cxx",
						Functions: map[string]*uniast.Function{
							"main": {
								Identity: funcId,
								Content:  "int main() {\n    return 0;\n}",
							},
						},
						Types: map[string]*uniast.Type{},
						Vars:  map[string]*uniast.Var{},
					},
				},
			},
		},
		Graph: map[string]*uniast.Node{
			funcId.Full(): {
				Identity: funcId,
				Type:     uniast.FUNC,
			},
		},
	}

	err := Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
	})

	if err != nil {
		t.Errorf("expected no error for C++ module, got %v", err)
	}

	// Check if CMakeLists.txt was created
	cmakePath := filepath.Join(tmpDir, "test", "CMakeLists.txt")
	if _, err := os.Stat(cmakePath); os.IsNotExist(err) {
		t.Errorf("expected CMakeLists.txt to be created at %s", cmakePath)
	}
}

func TestWrite_WithExistingAst(t *testing.T) {
	astFile := testutils.GetTestAstFile("localsession")
	repo, err := uniast.LoadRepo(astFile)
	if err != nil {
		t.Skipf("skip test: AST file not found: %v", err)
		return
	}

	tmpDir := testutils.MakeTmpTestdir(true)
	err = Write(context.Background(), repo, WriteOptions{
		OutputDir: tmpDir,
		Compiler:  "true", // Use 'true' to skip go mod tidy
	})

	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
}

func TestWriteOptions(t *testing.T) {
	opts := WriteOptions{
		OutputDir: "/tmp/test",
		Compiler:  "go",
	}

	if opts.OutputDir != "/tmp/test" {
		t.Errorf("OutputDir = %v, want %v", opts.OutputDir, "/tmp/test")
	}
	if opts.Compiler != "go" {
		t.Errorf("Compiler = %v, want %v", opts.Compiler, "go")
	}
}
