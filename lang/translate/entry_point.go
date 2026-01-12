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
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// EntryPointType represents the type of entry point
type EntryPointType int

const (
	EntryPointMain EntryPointType = iota
	EntryPointSpringBoot
	EntryPointRestController
	EntryPointScheduledTask
)

// EntryPointInfo contains information about a detected entry point
type EntryPointInfo struct {
	Type       EntryPointType
	Name       string
	Package    string
	ClassName  string
	MethodName string
	Content    string
	Identity   uniast.Identity
}

// EntryPointHandler handles entry point detection and generation
type EntryPointHandler struct {
	targetLang uniast.Language
}

// NewEntryPointHandler creates a new EntryPointHandler
func NewEntryPointHandler(targetLang uniast.Language) *EntryPointHandler {
	return &EntryPointHandler{
		targetLang: targetLang,
	}
}

// DetectEntryPoints finds all entry points in the repository
func (h *EntryPointHandler) DetectEntryPoints(repo *uniast.Repository) []EntryPointInfo {
	var entryPoints []EntryPointInfo

	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}

		for _, pkg := range mod.Packages {
			// Check functions for main methods
			for _, fn := range pkg.Functions {
				if ep := h.detectFunctionEntryPoint(fn); ep != nil {
					entryPoints = append(entryPoints, *ep)
				}
			}

			// Check types for annotated classes (Spring Boot, etc.)
			for _, typ := range pkg.Types {
				if eps := h.detectTypeEntryPoints(typ); len(eps) > 0 {
					entryPoints = append(entryPoints, eps...)
				}
			}
		}
	}

	return entryPoints
}

// detectFunctionEntryPoint checks if a function is an entry point
func (h *EntryPointHandler) detectFunctionEntryPoint(fn *uniast.Function) *EntryPointInfo {
	// Check for main function
	if fn.Name == "main" || strings.HasSuffix(fn.Name, "::main") {
		return &EntryPointInfo{
			Type:       EntryPointMain,
			Name:       fn.Name,
			MethodName: "main",
			Content:    fn.Content,
			Identity:   fn.Identity,
		}
	}

	// Check for Java main method pattern
	if strings.Contains(fn.Name, "main(String[])") || fn.Name == "main" {
		return &EntryPointInfo{
			Type:       EntryPointMain,
			Name:       fn.Name,
			MethodName: "main",
			Content:    fn.Content,
			Identity:   fn.Identity,
		}
	}

	return nil
}

// detectTypeEntryPoints checks if a type contains entry point annotations
func (h *EntryPointHandler) detectTypeEntryPoints(typ *uniast.Type) []EntryPointInfo {
	var entryPoints []EntryPointInfo
	content := typ.Content

	// Check for Spring Boot Application
	if strings.Contains(content, "@SpringBootApplication") {
		entryPoints = append(entryPoints, EntryPointInfo{
			Type:      EntryPointSpringBoot,
			Name:      typ.Name,
			ClassName: typ.Name,
			Content:   content,
			Identity:  typ.Identity,
		})
	}

	// Check for REST Controller
	if strings.Contains(content, "@RestController") || strings.Contains(content, "@Controller") {
		entryPoints = append(entryPoints, EntryPointInfo{
			Type:      EntryPointRestController,
			Name:      typ.Name,
			ClassName: typ.Name,
			Content:   content,
			Identity:  typ.Identity,
		})
	}

	// Check for Scheduled tasks
	if strings.Contains(content, "@Scheduled") {
		entryPoints = append(entryPoints, EntryPointInfo{
			Type:      EntryPointScheduledTask,
			Name:      typ.Name,
			ClassName: typ.Name,
			Content:   content,
			Identity:  typ.Identity,
		})
	}

	return entryPoints
}

// GenerateDefaultEntry generates a default entry point for the target language
func (h *EntryPointHandler) GenerateDefaultEntry(repo *uniast.Repository) (*uniast.Repository, error) {
	// Find the first non-external module
	var targetMod *uniast.Module
	var modName string
	for name, mod := range repo.Modules {
		if !mod.IsExternal() {
			targetMod = mod
			modName = name
			break
		}
	}

	if targetMod == nil {
		return repo, nil
	}

	// Generate entry point based on target language
	entryContent := h.generateEntryContent()
	if entryContent == "" {
		return repo, nil
	}

	// Create or get main package
	mainPkgPath := h.getMainPackagePath()
	mainPkg, exists := targetMod.Packages[uniast.PkgPath(mainPkgPath)]
	if !exists {
		mainPkg = &uniast.Package{
			IsMain:    true,
			PkgPath:   uniast.PkgPath(mainPkgPath),
			Functions: make(map[string]*uniast.Function),
			Types:     make(map[string]*uniast.Type),
			Vars:      make(map[string]*uniast.Var),
		}
		targetMod.Packages[uniast.PkgPath(mainPkgPath)] = mainPkg
	}

	// Add main function
	mainFunc := &uniast.Function{
		Exported: true,
		Identity: uniast.Identity{
			ModPath: modName,
			PkgPath: mainPkgPath,
			Name:    "main",
		},
		FileLine: uniast.FileLine{
			File: h.getMainFileName(),
			Line: 1,
		},
		Content: entryContent,
	}
	mainPkg.Functions["main"] = mainFunc
	mainPkg.IsMain = true

	return repo, nil
}

// ConvertEntryPoints converts existing entry points to target language style
func (h *EntryPointHandler) ConvertEntryPoints(repo *uniast.Repository, entryPoints []EntryPointInfo) (*uniast.Repository, error) {
	// Entry points have already been translated by LLM
	// This method can be used for additional adjustments if needed
	return repo, nil
}

// generateEntryContent generates the entry point content for target language
func (h *EntryPointHandler) generateEntryContent() string {
	switch h.targetLang {
	case uniast.Golang:
		return `func main() {
	// Application entry point
	fmt.Println("Application started")
}`
	case uniast.Rust:
		return `fn main() {
    // Application entry point
    println!("Application started");
}`
	case uniast.Python:
		return `def main():
    """Application entry point"""
    print("Application started")

if __name__ == "__main__":
    main()`
	case uniast.Cxx:
		return `int main(int argc, char* argv[]) {
    // Application entry point
    std::cout << "Application started" << std::endl;
    return 0;
}`
	case uniast.Java:
		return `public static void main(String[] args) {
    // Application entry point
    System.out.println("Application started");
}`
	default:
		return ""
	}
}

// getMainPackagePath returns the main package path for target language
func (h *EntryPointHandler) getMainPackagePath() string {
	switch h.targetLang {
	case uniast.Golang:
		return "main"
	case uniast.Rust:
		return "src"
	case uniast.Python:
		return "__main__"
	case uniast.Cxx:
		return "src"
	case uniast.Java:
		return "com.example.app"
	default:
		return "main"
	}
}

// getMainFileName returns the main file name for target language
func (h *EntryPointHandler) getMainFileName() string {
	switch h.targetLang {
	case uniast.Golang:
		return "main.go"
	case uniast.Rust:
		return "main.rs"
	case uniast.Python:
		return "__main__.py"
	case uniast.Cxx:
		return "main.cpp"
	case uniast.Java:
		return "Application.java"
	default:
		return "main"
	}
}

// HasMainEntry checks if there's already a main entry point
func (h *EntryPointHandler) HasMainEntry(entryPoints []EntryPointInfo) bool {
	for _, ep := range entryPoints {
		if ep.Type == EntryPointMain || ep.Type == EntryPointSpringBoot {
			return true
		}
	}
	return false
}

// GetRestControllers returns all REST controller entry points
func (h *EntryPointHandler) GetRestControllers(entryPoints []EntryPointInfo) []EntryPointInfo {
	var controllers []EntryPointInfo
	for _, ep := range entryPoints {
		if ep.Type == EntryPointRestController {
			controllers = append(controllers, ep)
		}
	}
	return controllers
}
