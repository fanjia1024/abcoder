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
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// ConfigGenerator generates project configuration files
type ConfigGenerator struct {
	targetLang     uniast.Language
	moduleName     string
	generatedFiles map[string]string
	dependencies   []string
}

// NewConfigGenerator creates a new ConfigGenerator
func NewConfigGenerator(targetLang uniast.Language, moduleName string) *ConfigGenerator {
	return &ConfigGenerator{
		targetLang:     targetLang,
		moduleName:     moduleName,
		generatedFiles: make(map[string]string),
		dependencies:   make([]string, 0),
	}
}

// AddDependency adds a dependency to be included in config
func (g *ConfigGenerator) AddDependency(dep string) {
	g.dependencies = append(g.dependencies, dep)
}

// Generate creates project configuration files
func (g *ConfigGenerator) Generate(repo *uniast.Repository, outputDir string) (*uniast.Repository, error) {
	// Determine module name if not set
	if g.moduleName == "" {
		g.moduleName = g.inferModuleName(repo)
	}

	// Generate config based on target language
	switch g.targetLang {
	case uniast.Golang:
		g.generateGoConfig(outputDir)
	case uniast.Rust:
		g.generateRustConfig(outputDir)
	case uniast.Python:
		g.generatePythonConfig(outputDir)
	case uniast.Cxx:
		g.generateCppConfig(outputDir)
	case uniast.Java:
		g.generateJavaConfig(outputDir)
	}

	// Write generated files to disk
	for path, content := range g.generatedFiles {
		fullPath := filepath.Join(outputDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return repo, nil
}

// GetFiles returns all generated configuration files
func (g *ConfigGenerator) GetFiles() map[string]string {
	return g.generatedFiles
}

// inferModuleName infers module name from repository
func (g *ConfigGenerator) inferModuleName(repo *uniast.Repository) string {
	if repo.Name != "" {
		return g.sanitizeModuleName(repo.Name)
	}

	// Try to get from first non-external module
	for name, mod := range repo.Modules {
		if !mod.IsExternal() {
			return g.sanitizeModuleName(name)
		}
	}

	return "translated-project"
}

// sanitizeModuleName cleans up module name for target language
func (g *ConfigGenerator) sanitizeModuleName(name string) string {
	// Remove version suffixes (Maven style)
	if idx := strings.LastIndex(name, ":"); idx > 0 {
		suffix := name[idx+1:]
		// Check if after : is a version number
		if len(suffix) > 0 && (suffix[0] >= '0' && suffix[0] <= '9') {
			name = name[:idx]
		}
	}

	// Handle filesystem paths - extract just the project name
	if strings.Contains(name, "/Users/") || strings.Contains(name, "/home/") || strings.HasPrefix(name, "/") {
		// Extract the last meaningful directory name
		parts := strings.Split(name, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			if part != "" && part != "." && part != ".." {
				name = part
				break
			}
		}
	}

	// Replace invalid characters
	name = strings.ReplaceAll(name, ":", "/")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Trim(name, "/")

	switch g.targetLang {
	case uniast.Golang:
		// Convert to valid Go module name
		name = strings.ToLower(name)
		name = strings.ReplaceAll(name, "_", "-")
		// If it doesn't look like a domain, add github.com/example prefix
		if !strings.Contains(name, ".") {
			return "github.com/example/" + name
		}
		return name
	case uniast.Rust:
		// Rust crate names: lowercase, underscores
		name = strings.ToLower(name)
		name = strings.ReplaceAll(name, "-", "_")
		name = strings.ReplaceAll(name, "/", "_")
		return name
	case uniast.Python:
		// Python package names: lowercase, underscores
		name = strings.ToLower(name)
		name = strings.ReplaceAll(name, "-", "_")
		name = strings.ReplaceAll(name, "/", "_")
		return name
	default:
		return name
	}
}

// generateGoConfig generates Go project configuration
func (g *ConfigGenerator) generateGoConfig(outputDir string) {
	moduleName := g.moduleName
	if moduleName == "" {
		moduleName = "github.com/example/translated"
	}

	// go.mod
	goMod := fmt.Sprintf(`module %s

go 1.21

require (
`, moduleName)

	// Add dependencies
	for _, dep := range g.dependencies {
		goMod += fmt.Sprintf("\t%s\n", dep)
	}
	goMod += ")\n"

	g.generatedFiles["go.mod"] = goMod

	// Create standard directory structure markers
	g.generatedFiles["cmd/.gitkeep"] = ""
	g.generatedFiles["internal/.gitkeep"] = ""
	g.generatedFiles["pkg/.gitkeep"] = ""
}

// generateRustConfig generates Rust project configuration
func (g *ConfigGenerator) generateRustConfig(outputDir string) {
	projectName := g.moduleName
	if projectName == "" {
		projectName = "translated"
	}
	// Rust package names must be valid identifiers
	projectName = strings.ReplaceAll(projectName, "-", "_")
	projectName = strings.ReplaceAll(projectName, "/", "_")

	cargoToml := fmt.Sprintf(`[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[dependencies]
`, projectName)

	// Add dependencies
	for _, dep := range g.dependencies {
		cargoToml += fmt.Sprintf("%s\n", dep)
	}

	g.generatedFiles["Cargo.toml"] = cargoToml

	// Create src directory structure
	g.generatedFiles["src/lib.rs"] = `// Library root
pub mod model;
pub mod service;
pub mod repository;
`

	// Create module files
	g.generatedFiles["src/model/mod.rs"] = "// Model module\n"
	g.generatedFiles["src/service/mod.rs"] = "// Service module\n"
	g.generatedFiles["src/repository/mod.rs"] = "// Repository module\n"
}

// generatePythonConfig generates Python project configuration
func (g *ConfigGenerator) generatePythonConfig(outputDir string) {
	projectName := g.moduleName
	if projectName == "" {
		projectName = "translated"
	}
	// Python package names
	projectName = strings.ReplaceAll(projectName, "-", "_")
	projectName = strings.ReplaceAll(projectName, "/", "_")
	projectName = strings.ToLower(projectName)

	// pyproject.toml
	pyprojectToml := fmt.Sprintf(`[build-system]
requires = ["setuptools>=61.0"]
build-backend = "setuptools.build_meta"

[project]
name = "%s"
version = "0.1.0"
description = "Translated project"
readme = "README.md"
requires-python = ">=3.8"
dependencies = [
`, projectName)

	// Add dependencies
	for _, dep := range g.dependencies {
		pyprojectToml += fmt.Sprintf("    \"%s\",\n", dep)
	}
	pyprojectToml += `]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "black>=23.0",
    "mypy>=1.0",
]
`
	g.generatedFiles["pyproject.toml"] = pyprojectToml

	// requirements.txt
	requirements := "# Project dependencies\n"
	for _, dep := range g.dependencies {
		requirements += dep + "\n"
	}
	g.generatedFiles["requirements.txt"] = requirements

	// Create package structure
	g.generatedFiles["src/__init__.py"] = `"""Main package"""

__version__ = "0.1.0"
`
	g.generatedFiles["src/model/__init__.py"] = `"""Model module"""
`
	g.generatedFiles["src/service/__init__.py"] = `"""Service module"""
`
	g.generatedFiles["src/repository/__init__.py"] = `"""Repository module"""
`
	g.generatedFiles["tests/__init__.py"] = `"""Test package"""
`
}

// generateCppConfig generates C++ project configuration
func (g *ConfigGenerator) generateCppConfig(outputDir string) {
	projectName := g.moduleName
	if projectName == "" {
		projectName = "translated"
	}
	// Clean project name for CMake
	projectName = strings.ReplaceAll(projectName, "/", "_")
	projectName = strings.ReplaceAll(projectName, "-", "_")

	cmakeLists := fmt.Sprintf(`cmake_minimum_required(VERSION 3.16)
project(%s VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Include directories
include_directories(${PROJECT_SOURCE_DIR}/include)

# Source files
file(GLOB_RECURSE SOURCES "src/*.cpp")
file(GLOB_RECURSE HEADERS "include/*.h" "include/*.hpp")

# Main executable
add_executable(${PROJECT_NAME} ${SOURCES} ${HEADERS})

# Install targets
install(TARGETS ${PROJECT_NAME} DESTINATION bin)
install(DIRECTORY include/ DESTINATION include)
`, projectName)

	g.generatedFiles["CMakeLists.txt"] = cmakeLists

	// Create directory structure
	g.generatedFiles["include/.gitkeep"] = ""
	g.generatedFiles["src/.gitkeep"] = ""

	// Create a basic main.cpp if not exists
	mainCpp := `#include <iostream>

int main(int argc, char* argv[]) {
    std::cout << "Application started" << std::endl;
    return 0;
}
`
	g.generatedFiles["src/main.cpp"] = mainCpp

	// Create build script
	buildScript := `#!/bin/bash
# Build script for the project

mkdir -p build
cd build
cmake ..
make -j$(nproc)
`
	g.generatedFiles["build.sh"] = buildScript
}

// generateJavaConfig generates Java project configuration (Maven)
func (g *ConfigGenerator) generateJavaConfig(outputDir string) {
	groupId := "com.example"
	artifactId := g.moduleName
	if artifactId == "" {
		artifactId = "translated"
	}
	// Clean artifact ID
	artifactId = strings.ReplaceAll(artifactId, "/", "-")
	artifactId = strings.ToLower(artifactId)

	pomXml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>%s</groupId>
    <artifactId>%s</artifactId>
    <version>0.1.0-SNAPSHOT</version>
    <packaging>jar</packaging>

    <properties>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
`, groupId, artifactId)

	// Add dependencies
	for _, dep := range g.dependencies {
		pomXml += fmt.Sprintf("        <!-- %s -->\n", dep)
	}

	pomXml += `    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-compiler-plugin</artifactId>
                <version>3.11.0</version>
            </plugin>
        </plugins>
    </build>
</project>
`
	g.generatedFiles["pom.xml"] = pomXml

	// Create directory structure
	g.generatedFiles["src/main/java/.gitkeep"] = ""
	g.generatedFiles["src/main/resources/.gitkeep"] = ""
	g.generatedFiles["src/test/java/.gitkeep"] = ""
}
