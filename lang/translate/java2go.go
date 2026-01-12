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
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// convertJavaNameToGo converts a Java name to Go naming convention
func convertJavaNameToGo(name string, exported bool) string {
	if name == "" {
		return name
	}

	// Handle method names like "ClassName.methodName"
	if idx := strings.LastIndex(name, "."); idx != -1 {
		className := name[:idx]
		methodName := name[idx+1:]
		return convertJavaNameToGo(className, exported) + "." + convertJavaNameToGo(methodName, exported)
	}

	// Convert first letter based on visibility
	runes := []rune(name)
	if exported {
		runes[0] = unicode.ToUpper(runes[0])
	} else {
		runes[0] = unicode.ToLower(runes[0])
	}
	return string(runes)
}

// convertJavaFilePathToGo converts a Java file path to Go file path
// e.g., "src/main/java/com/example/model/User.java" -> "model/user.go"
func convertJavaFilePathToGo(javaPath string) string {
	if javaPath == "" {
		return javaPath
	}

	// Extract filename without path
	base := filepath.Base(javaPath)

	// Remove .java extension and add .go
	base = strings.TrimSuffix(base, ".java")

	// Convert to lowercase for Go convention (e.g., User.java -> user.go)
	base = strings.ToLower(base) + ".go"

	return base
}

// Java2GoOptions contains options for Java to Go translation (legacy)
// Deprecated: Use TranslateOptions with LLMTranslator for new code
type Java2GoOptions struct {
	GoModuleName string // Target Go module name (e.g., github.com/example/project)
	OutputDir    string // Output directory for Go code
}

// extractGroupIdFromModuleName extracts groupId from Maven module name
// Format: groupId:artifactId:version -> groupId
// Example: com.example.test:test-repo:1.0.0-SNAPSHOT -> com.example.test
func extractGroupIdFromModuleName(modName string) string {
	parts := strings.Split(modName, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return modName
}

// TranslateJava2Go converts a Java UniAST Repository to Go UniAST Repository
// This function merges all Java modules into a single Go module with flat package structure
// Deprecated: Use Translate with LLMTranslator for new code
func TranslateJava2Go(ctx context.Context, javaRepo *uniast.Repository, opts Java2GoOptions) (*uniast.Repository, error) {
	if javaRepo == nil {
		return nil, fmt.Errorf("java repository is nil")
	}

	// Determine Go module name from groupId
	goModuleName := opts.GoModuleName
	if goModuleName == "" {
		// Extract groupId from the first non-external module
		for modPath, javaMod := range javaRepo.Modules {
			if !javaMod.IsExternal() {
				groupId := extractGroupIdFromModuleName(modPath)
				goModuleName = GetGoModuleNameFromGroupId(groupId)
				break
			}
		}
		// Fallback if no modules found
		if goModuleName == "" {
			goModuleName = ConvertJavaPackageToGoModule(javaRepo.Name)
		}
	}

	goRepo := uniast.NewRepository(goModuleName)
	goRepo.Path = javaRepo.Path
	goRepo.ASTVersion = javaRepo.ASTVersion
	goRepo.ToolVersion = javaRepo.ToolVersion

	// Create a single Go module (merge all Java modules)
	goModName := goModuleName
	goMod := uniast.NewModule(goModName, "", uniast.Golang) // Empty dir for flat structure

	// Track package names to handle conflicts
	packageNames := make(map[string]int) // package name -> count

	// Convert all modules and merge packages
	for _, javaMod := range javaRepo.Modules {
		if javaMod.IsExternal() {
			continue
		}

		// Convert packages from this Java module
		for pkgPath, javaPkg := range javaMod.Packages {
			// Simplify package path: com.example.common.model -> model
			goPkgPath := ConvertJavaPackageToGoModule(string(pkgPath))
			if goPkgPath == "" {
				goPkgPath = string(pkgPath)
			}

			// Handle package name conflicts
			originalPkgPath := goPkgPath
			if count, exists := packageNames[goPkgPath]; exists {
				// Conflict detected, add suffix
				packageNames[goPkgPath] = count + 1
				goPkgPath = fmt.Sprintf("%s_%d", goPkgPath, count+1)
			} else {
				packageNames[originalPkgPath] = 1
			}

			// Check if package already exists (from another module)
			var goPkg *uniast.Package
			if existingPkg, exists := goMod.Packages[uniast.PkgPath(goPkgPath)]; exists {
				// Merge into existing package
				goPkg = existingPkg
			} else {
				// Create new Go package
				goPkg = &uniast.Package{
					PkgPath:   uniast.PkgPath(goPkgPath),
					IsMain:    javaPkg.IsMain,
					Functions: make(map[string]*uniast.Function),
					Types:     make(map[string]*uniast.Type),
					Vars:      make(map[string]*uniast.Var),
				}
				goMod.Packages[uniast.PkgPath(goPkgPath)] = goPkg
			}

			// Copy functions with Content (LLM will translate Content later)
			for name, javaFn := range javaPkg.Functions {
				goFn := &uniast.Function{
					Exported:          javaFn.Exported,
					IsMethod:          javaFn.IsMethod,
					IsInterfaceMethod: javaFn.IsInterfaceMethod,
					Identity: uniast.Identity{
						ModPath: goModName, // All use the same module
						PkgPath: goPkgPath,
						Name:    convertJavaNameToGo(name, javaFn.Exported),
					},
					FileLine: uniast.FileLine{
						File: convertJavaFilePathToGo(javaFn.File), // Convert .java to .go
						Line: javaFn.Line,
					},
					Content:   javaFn.Content, // Java code - to be translated by LLM
					Signature: javaFn.Signature,
				}
				// Handle function name conflicts within the same package
				funcName := goFn.Name
				counter := 1
				for _, exists := goPkg.Functions[funcName]; exists; {
					funcName = fmt.Sprintf("%s_%d", goFn.Name, counter)
					counter++
					_, exists = goPkg.Functions[funcName]
				}
				goFn.Name = funcName
				goFn.Identity.Name = funcName
				goPkg.Functions[funcName] = goFn
			}

			// Copy types with Content
			for name, javaType := range javaPkg.Types {
				goType := &uniast.Type{
					Exported: javaType.Exported,
					TypeKind: javaType.TypeKind,
					Identity: uniast.Identity{
						ModPath: goModName, // All use the same module
						PkgPath: goPkgPath,
						Name:    convertJavaNameToGo(name, javaType.Exported),
					},
					FileLine: uniast.FileLine{
						File: convertJavaFilePathToGo(javaType.File), // Convert .java to .go
						Line: javaType.Line,
					},
					Content: javaType.Content, // Java code - to be translated by LLM
				}
				// Handle type name conflicts within the same package
				typeName := goType.Name
				counter := 1
				for _, exists := goPkg.Types[typeName]; exists; {
					typeName = fmt.Sprintf("%s_%d", goType.Name, counter)
					counter++
					_, exists = goPkg.Types[typeName]
				}
				goType.Name = typeName
				goType.Identity.Name = typeName
				goPkg.Types[typeName] = goType
			}

			// Copy vars with Content
			for name, javaVar := range javaPkg.Vars {
				goVar := &uniast.Var{
					IsExported: javaVar.IsExported,
					IsConst:    javaVar.IsConst,
					IsPointer:  javaVar.IsPointer,
					Identity: uniast.Identity{
						ModPath: goModName, // All use the same module
						PkgPath: goPkgPath,
						Name:    convertJavaNameToGo(name, javaVar.IsExported),
					},
					FileLine: uniast.FileLine{
						File: convertJavaFilePathToGo(javaVar.File), // Convert .java to .go
						Line: javaVar.Line,
					},
					Content: javaVar.Content, // Java code - to be translated by LLM
				}
				// Handle var name conflicts within the same package
				varName := goVar.Name
				counter := 1
				for _, exists := goPkg.Vars[varName]; exists; {
					varName = fmt.Sprintf("%s_%d", goVar.Name, counter)
					counter++
					_, exists = goPkg.Vars[varName]
				}
				goVar.Name = varName
				goVar.Identity.Name = varName
				goPkg.Vars[varName] = goVar
			}
		}

		// Copy files structure with converted paths (merge into single module)
		for filePath, javaFile := range javaMod.Files {
			// Convert file path from .java to .go
			goFilePath := convertJavaFilePathToGo(filePath)
			// Convert package reference
			var goFilePkgPath uniast.PkgPath
			if javaFile.Package != "" {
				goFilePkgPath = uniast.PkgPath(ConvertJavaPackageToGoModule(string(javaFile.Package)))
			}
			// Check if file already exists (from another module)
			if existingFile, exists := goMod.Files[goFilePath]; exists {
				// Merge imports if needed
				existingFile.Imports = append(existingFile.Imports, javaFile.Imports...)
			} else {
				goFile := &uniast.File{
					Path:    goFilePath,
					Package: goFilePkgPath,
					Imports: []uniast.Import{},
				}
				goMod.Files[goFilePath] = goFile
			}
		}

		// Merge dependencies (avoid duplicates)
		for depName, depVersion := range javaMod.Dependencies {
			// Convert Java dependency to Go module if needed
			// For now, keep as is or convert based on naming
			if _, exists := goMod.Dependencies[depName]; !exists {
				goMod.Dependencies[depName] = depVersion
			}
		}
	}

	// Add the single merged module to the repository
	goRepo.Modules[goModName] = goMod

	return &goRepo, nil
}

// PrepareConversionContext prepares context information for LLM conversion
// This includes module structure, package information, etc.
type ConversionContext struct {
	RepoName    string
	ModulePath  string
	PackagePath string
	FilePath    string
	Nodes       []*uniast.Node
}

// PrepareContext prepares conversion context for a specific file
func PrepareContext(repo *uniast.Repository, modulePath string, packagePath string, filePath string) (*ConversionContext, error) {
	mod := repo.GetModule(uniast.ModPath(modulePath))
	if mod == nil {
		return nil, fmt.Errorf("module not found: %s", modulePath)
	}

	nodes := repo.GetFileNodes(filePath)

	return &ConversionContext{
		RepoName:    repo.Name,
		ModulePath:  modulePath,
		PackagePath: packagePath,
		FilePath:    filePath,
		Nodes:       nodes,
	}, nil
}
