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
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// PostProcessOptions contains options for post-translation processing
type PostProcessOptions struct {
	GenerateEntryPoint bool   // Whether to generate entry point if missing
	WebFramework       string // Web framework: "gin", "echo", "actix", "fastapi", "none"
	GenerateConfig     bool   // Whether to generate project config files
	ModuleName         string // Module name for config generation
	OutputDir          string // Output directory path
}

// PostProcessor handles post-translation processing
type PostProcessor struct {
	targetLang          uniast.Language
	opts                PostProcessOptions
	entryPointHandler   *EntryPointHandler
	configGenerator     *ConfigGenerator
	frameworkIntegrator *FrameworkIntegrator
}

// NewPostProcessor creates a new PostProcessor
func NewPostProcessor(targetLang uniast.Language, opts PostProcessOptions) *PostProcessor {
	return &PostProcessor{
		targetLang:          targetLang,
		opts:                opts,
		entryPointHandler:   NewEntryPointHandler(targetLang),
		configGenerator:     NewConfigGenerator(targetLang, opts.ModuleName),
		frameworkIntegrator: NewFrameworkIntegrator(targetLang, opts.WebFramework),
	}
}

// Process performs all post-translation processing steps
func (p *PostProcessor) Process(repo *uniast.Repository) (*uniast.Repository, error) {
	var err error

	// Step 0: Fix imports in translated code (Java-style to Go-style)
	if p.targetLang == uniast.Golang {
		repo = p.fixGoImports(repo)
	}

	// Step 1: Detect and handle entry points
	if p.opts.GenerateEntryPoint {
		repo, err = p.processEntryPoints(repo)
		if err != nil {
			return nil, fmt.Errorf("entry point processing failed: %w", err)
		}
	}

	// Step 2: Integrate web framework if specified
	if p.opts.WebFramework != "" && p.opts.WebFramework != "none" {
		repo, err = p.integrateFramework(repo)
		if err != nil {
			return nil, fmt.Errorf("framework integration failed: %w", err)
		}
	}

	// Step 3: Generate project configuration files
	if p.opts.GenerateConfig {
		repo, err = p.generateConfig(repo)
		if err != nil {
			return nil, fmt.Errorf("config generation failed: %w", err)
		}
	}

	return repo, nil
}

// processEntryPoints detects existing entry points and generates missing ones
func (p *PostProcessor) processEntryPoints(repo *uniast.Repository) (*uniast.Repository, error) {
	// Detect existing entry points
	entryPoints := p.entryPointHandler.DetectEntryPoints(repo)

	// If no entry point found, generate a default one
	if len(entryPoints) == 0 {
		return p.entryPointHandler.GenerateDefaultEntry(repo)
	}

	// Convert existing entry points to target language style
	return p.entryPointHandler.ConvertEntryPoints(repo, entryPoints)
}

// integrateFramework integrates web framework conventions
func (p *PostProcessor) integrateFramework(repo *uniast.Repository) (*uniast.Repository, error) {
	return p.frameworkIntegrator.Integrate(repo)
}

// generateConfig generates project configuration files
func (p *PostProcessor) generateConfig(repo *uniast.Repository) (*uniast.Repository, error) {
	return p.configGenerator.Generate(repo, p.opts.OutputDir)
}

// GetGeneratedFiles returns the list of additional files to be written
func (p *PostProcessor) GetGeneratedFiles() map[string]string {
	files := make(map[string]string)

	// Add config generator files
	for path, content := range p.configGenerator.GetFiles() {
		files[path] = content
	}

	// Add framework integrator files
	for path, content := range p.frameworkIntegrator.GetFiles() {
		files[path] = content
	}

	return files
}

// fixGoImports fixes Java-style imports in Go code to use proper Go module paths
func (p *PostProcessor) fixGoImports(repo *uniast.Repository) *uniast.Repository {
	moduleName := p.opts.ModuleName
	if moduleName == "" {
		moduleName = "github.com/example/translated"
	}

	// Build a map of existing packages and common Java->Go mappings
	existingPkgs := make(map[string]string) // Java-style -> Go-style
	goPkgNames := make([]string, 0)

	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		for pkgPath := range mod.Packages {
			goPath := string(pkgPath)
			parts := strings.Split(goPath, "/")
			if len(parts) > 0 {
				pkgName := parts[len(parts)-1]
				goPkgNames = append(goPkgNames, pkgName)
				// Map various Java-style imports to our module
				existingPkgs["com.example/"+pkgName] = moduleName + "/" + pkgName
				existingPkgs["com.example."+pkgName] = moduleName + "/" + pkgName
				existingPkgs["your-module/"+pkgName] = moduleName + "/" + pkgName
				existingPkgs["your-module/path/to/"+pkgName] = moduleName + "/" + pkgName
				existingPkgs["your-module/config"] = moduleName + "/" + pkgName // Explicit mapping
			}
		}
	}

	// Debug: Log the package mapping
	// fmt.Printf("[PostProcessor] Module: %s, Packages: %v\n", moduleName, goPkgNames)

	// Add common Java package name mappings
	// Java 'core' often maps to Go 'model' or 'repository'
	commonMappings := map[string]string{
		"core":        "model",
		"domain":      "model",
		"entity":      "model",
		"entities":    "model",
		"dto":         "model",
		"vo":          "model",
		"pojo":        "model",
		"bean":        "model",
		"beans":       "model",
		"dao":         "repository",
		"mapper":      "repository",
		"repo":        "repository",
		"persistence": "repository",
		"api":         "controller",
		"rest":        "controller",
		"endpoint":    "controller",
		"handler":     "controller",
		"impl":        "service",
		"business":    "service",
		"logic":       "service",
		"common":      "utils",
		"util":        "utils",
		"helper":      "utils",
		"helpers":     "utils",
	}

	for javaName, goName := range commonMappings {
		// Only map if the target package exists
		for _, existingPkg := range goPkgNames {
			if existingPkg == goName {
				existingPkgs["com.example/"+javaName] = moduleName + "/" + goName
				existingPkgs["com.example."+javaName] = moduleName + "/" + goName
				existingPkgs["your-module/"+javaName] = moduleName + "/" + goName
				break
			}
		}
	}

	// Regex patterns for import statements - match anything that looks like a Java package
	importRegex := regexp.MustCompile(`"(com\.example[^"]*|your-module[^"]*|[a-z]+\.[a-z]+\.[a-z]+[^"]*)"`)

	// Fix imports in all functions, types, vars
	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		for _, pkg := range mod.Packages {
			for _, fn := range pkg.Functions {
				fn.Content = p.fixImportsInContent(fn.Content, existingPkgs, importRegex, moduleName, goPkgNames)
			}
			for _, typ := range pkg.Types {
				typ.Content = p.fixImportsInContent(typ.Content, existingPkgs, importRegex, moduleName, goPkgNames)
			}
			for _, v := range pkg.Vars {
				v.Content = p.fixImportsInContent(v.Content, existingPkgs, importRegex, moduleName, goPkgNames)
			}
		}
		// Fix imports in files
		for _, file := range mod.Files {
			for i, imp := range file.Imports {
				file.Imports[i].Path = p.fixImportPath(imp.Path, existingPkgs, moduleName, goPkgNames)
			}
		}
	}

	return repo
}

// fixImportsInContent fixes import statements in code content
func (p *PostProcessor) fixImportsInContent(content string, existingPkgs map[string]string, importRegex *regexp.Regexp, moduleName string, goPkgNames []string) string {
	// Fix import statements
	content = importRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the import path (remove quotes)
		importPath := strings.Trim(match, `"`)

		// Check if we have a mapping for this import
		if newPath, ok := existingPkgs[importPath]; ok {
			return `"` + newPath + `"`
		}

		// Convert Java-style package to Go module path
		newPath := p.convertJavaImportToGo(importPath, moduleName, goPkgNames)
		return `"` + newPath + `"`
	})

	return content
}

// fixImportPath fixes a single import path
func (p *PostProcessor) fixImportPath(path string, existingPkgs map[string]string, moduleName string, goPkgNames []string) string {
	if newPath, ok := existingPkgs[path]; ok {
		return newPath
	}
	if strings.HasPrefix(path, "com.example") || strings.HasPrefix(path, "your-module") ||
		strings.Contains(path, ".") && !strings.Contains(path, "github.com") {
		return p.convertJavaImportToGo(path, moduleName, goPkgNames)
	}
	return path
}

// convertJavaImportToGo converts a Java-style import to Go module path
func (p *PostProcessor) convertJavaImportToGo(javaImport, moduleName string, goPkgNames []string) string {
	// Handle common patterns
	// com.example/core -> moduleName/model (mapped)
	// com.example.service -> moduleName/service
	// your-module/path/to/X -> moduleName/x

	original := javaImport

	// Remove common prefixes
	javaImport = strings.TrimPrefix(javaImport, "com.example/")
	javaImport = strings.TrimPrefix(javaImport, "com.example.")
	javaImport = strings.TrimPrefix(javaImport, "your-module/path/to/")
	javaImport = strings.TrimPrefix(javaImport, "your-module/")

	// Convert dots to slashes
	javaImport = strings.ReplaceAll(javaImport, ".", "/")

	// Lowercase
	javaImport = strings.ToLower(javaImport)

	// Get the last segment (package name)
	parts := strings.Split(javaImport, "/")
	pkgName := parts[len(parts)-1]

	// Check if this package name exists in our generated packages
	for _, existingPkg := range goPkgNames {
		if existingPkg == pkgName {
			return moduleName + "/" + pkgName
		}
	}

	// Try common mappings
	commonMappings := map[string]string{
		"core": "model", "domain": "model", "entity": "model",
		"entities": "model", "dto": "model", "vo": "model",
		"dao": "repository", "mapper": "repository", "repo": "repository",
		"api": "controller", "rest": "controller", "endpoint": "controller",
		"impl": "service", "business": "service",
		"common": "utils", "util": "utils", "helper": "utils",
	}

	if mappedName, ok := commonMappings[pkgName]; ok {
		for _, existingPkg := range goPkgNames {
			if existingPkg == mappedName {
				return moduleName + "/" + mappedName
			}
		}
	}

	// If still no match, try to find a similar package or use 'model' as fallback
	for _, existingPkg := range goPkgNames {
		if strings.Contains(pkgName, existingPkg) || strings.Contains(existingPkg, pkgName) {
			return moduleName + "/" + existingPkg
		}
	}

	// Last resort: comment out invalid import and log warning
	// For now, just return the original converted path
	if javaImport != "" {
		return moduleName + "/" + javaImport
	}

	// If nothing worked, return original (will cause build error but at least visible)
	return original
}
