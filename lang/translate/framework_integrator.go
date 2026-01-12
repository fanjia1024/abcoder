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
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

// FrameworkIntegrator handles web framework integration
type FrameworkIntegrator struct {
	targetLang     uniast.Language
	framework      string
	generatedFiles map[string]string
	routes         []RouteInfo
}

// RouteInfo contains information about a detected route
type RouteInfo struct {
	Method      string // GET, POST, PUT, DELETE, etc.
	Path        string
	HandlerName string
	SourceType  string // Original controller class name
}

// NewFrameworkIntegrator creates a new FrameworkIntegrator
func NewFrameworkIntegrator(targetLang uniast.Language, framework string) *FrameworkIntegrator {
	return &FrameworkIntegrator{
		targetLang:     targetLang,
		framework:      framework,
		generatedFiles: make(map[string]string),
		routes:         make([]RouteInfo, 0),
	}
}

// Integrate performs framework integration
func (f *FrameworkIntegrator) Integrate(repo *uniast.Repository) (*uniast.Repository, error) {
	// Detect REST endpoints
	f.detectRoutes(repo)

	// Generate framework-specific code
	switch f.targetLang {
	case uniast.Golang:
		return f.integrateGo(repo)
	case uniast.Rust:
		return f.integrateRust(repo)
	case uniast.Python:
		return f.integratePython(repo)
	default:
		return repo, nil
	}
}

// GetFiles returns generated framework files
func (f *FrameworkIntegrator) GetFiles() map[string]string {
	return f.generatedFiles
}

// GetDependencies returns dependencies for the framework
func (f *FrameworkIntegrator) GetDependencies() []string {
	switch f.targetLang {
	case uniast.Golang:
		return f.getGoDependencies()
	case uniast.Rust:
		return f.getRustDependencies()
	case uniast.Python:
		return f.getPythonDependencies()
	default:
		return nil
	}
}

// detectRoutes finds all REST endpoints in the repository
func (f *FrameworkIntegrator) detectRoutes(repo *uniast.Repository) {
	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}

		for _, pkg := range mod.Packages {
			for _, typ := range pkg.Types {
				f.detectRoutesFromType(typ)
			}
			for _, fn := range pkg.Functions {
				f.detectRoutesFromFunction(fn)
			}
		}
	}
}

// detectRoutesFromType detects routes from type annotations
func (f *FrameworkIntegrator) detectRoutesFromType(typ *uniast.Type) {
	content := typ.Content

	// Check for Spring REST annotations
	if !strings.Contains(content, "@RestController") &&
		!strings.Contains(content, "@Controller") {
		return
	}

	// Extract base path from @RequestMapping
	basePath := "/"
	if idx := strings.Index(content, "@RequestMapping"); idx != -1 {
		basePath = extractAnnotationValue(content[idx:], "value", "/")
	}

	// Find method mappings
	mappings := []struct {
		annotation string
		method     string
	}{
		{"@GetMapping", "GET"},
		{"@PostMapping", "POST"},
		{"@PutMapping", "PUT"},
		{"@DeleteMapping", "DELETE"},
		{"@PatchMapping", "PATCH"},
		{"@RequestMapping", "GET"}, // Default to GET
	}

	for _, m := range mappings {
		if m.annotation == "@RequestMapping" {
			continue // Skip general RequestMapping, already handled
		}

		// Find all occurrences
		idx := 0
		for {
			pos := strings.Index(content[idx:], m.annotation)
			if pos == -1 {
				break
			}
			idx += pos

			// Extract path
			path := extractAnnotationValue(content[idx:], "value", "")
			if path == "" {
				path = extractAnnotationValue(content[idx:], "", "")
			}

			// Extract method name (simplified)
			methodName := extractNextMethodName(content[idx:])

			fullPath := basePath
			if path != "" {
				if !strings.HasSuffix(fullPath, "/") && !strings.HasPrefix(path, "/") {
					fullPath += "/"
				}
				fullPath += strings.TrimPrefix(path, "/")
			}

			f.routes = append(f.routes, RouteInfo{
				Method:      m.method,
				Path:        fullPath,
				HandlerName: methodName,
				SourceType:  typ.Name,
			})

			idx++
		}
	}
}

// detectRoutesFromFunction detects routes from function annotations
func (f *FrameworkIntegrator) detectRoutesFromFunction(fn *uniast.Function) {
	// Routes in functions are typically already detected from types
	// This can be extended for other patterns
}

// integrateGo generates Go web framework integration
func (f *FrameworkIntegrator) integrateGo(repo *uniast.Repository) (*uniast.Repository, error) {
	switch f.framework {
	case "gin":
		return f.integrateGin(repo)
	case "echo":
		return f.integrateEcho(repo)
	default:
		return f.integrateGin(repo) // Default to Gin
	}
}

// integrateGin generates Gin framework code
func (f *FrameworkIntegrator) integrateGin(repo *uniast.Repository) (*uniast.Repository, error) {
	// Generate main.go with Gin setup
	mainContent := `package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	
	// Register routes
	registerRoutes(r)
	
	// Start server
	r.Run(":8080")
}
`

	// Generate routes.go
	routesContent := f.generateGinRoutes()

	f.generatedFiles["cmd/main.go"] = mainContent
	f.generatedFiles["internal/router/routes.go"] = routesContent

	return repo, nil
}

// generateGinRoutes generates Gin route registration code
func (f *FrameworkIntegrator) generateGinRoutes() string {
	var sb strings.Builder
	sb.WriteString(`package router

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
`)

	for _, route := range f.routes {
		method := strings.ToLower(route.Method)
		sb.WriteString(fmt.Sprintf("\t\tapi.%s(\"%s\", %sHandler)\n",
			strings.Title(method), route.Path, frameworkToCamelCase(route.HandlerName)))
	}

	sb.WriteString(`	}
}

`)

	// Generate handler stubs
	for _, route := range f.routes {
		sb.WriteString(fmt.Sprintf(`// %sHandler handles %s %s
func %sHandler(c *gin.Context) {
	// TODO: Implement handler logic
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

`, frameworkToCamelCase(route.HandlerName), route.Method, route.Path, frameworkToCamelCase(route.HandlerName)))
	}

	return sb.String()
}

// integrateEcho generates Echo framework code
func (f *FrameworkIntegrator) integrateEcho(repo *uniast.Repository) (*uniast.Repository, error) {
	mainContent := `package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	
	// Register routes
	registerRoutes(e)
	
	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
`

	routesContent := f.generateEchoRoutes()

	f.generatedFiles["cmd/main.go"] = mainContent
	f.generatedFiles["internal/router/routes.go"] = routesContent

	return repo, nil
}

// generateEchoRoutes generates Echo route registration code
func (f *FrameworkIntegrator) generateEchoRoutes() string {
	var sb strings.Builder
	sb.WriteString(`package router

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(e *echo.Echo) {
	api := e.Group("/api")
`)

	for _, route := range f.routes {
		sb.WriteString(fmt.Sprintf("\tapi.%s(\"%s\", %sHandler)\n",
			strings.ToUpper(route.Method), route.Path, frameworkToCamelCase(route.HandlerName)))
	}

	sb.WriteString("}\n\n")

	// Generate handler stubs
	for _, route := range f.routes {
		sb.WriteString(fmt.Sprintf(`// %sHandler handles %s %s
func %sHandler(c echo.Context) error {
	// TODO: Implement handler logic
	return c.JSON(http.StatusOK, map[string]string{"message": "success"})
}

`, frameworkToCamelCase(route.HandlerName), route.Method, route.Path, frameworkToCamelCase(route.HandlerName)))
	}

	return sb.String()
}

// integrateRust generates Rust web framework integration
func (f *FrameworkIntegrator) integrateRust(repo *uniast.Repository) (*uniast.Repository, error) {
	switch f.framework {
	case "actix":
		return f.integrateActix(repo)
	case "axum":
		return f.integrateAxum(repo)
	default:
		return f.integrateActix(repo) // Default to Actix
	}
}

// integrateActix generates Actix-web code
func (f *FrameworkIntegrator) integrateActix(repo *uniast.Repository) (*uniast.Repository, error) {
	mainContent := `use actix_web::{web, App, HttpServer, HttpResponse, Responder};

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| {
        App::new()
            .configure(configure_routes)
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}

fn configure_routes(cfg: &mut web::ServiceConfig) {
`

	for _, route := range f.routes {
		method := strings.ToLower(route.Method)
		mainContent += fmt.Sprintf("    cfg.route(\"%s\", web::%s().to(%s));\n",
			route.Path, method, frameworkToSnakeCase(route.HandlerName))
	}

	mainContent += "}\n\n"

	// Add handler functions
	for _, route := range f.routes {
		mainContent += fmt.Sprintf(`async fn %s() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({"message": "success"}))
}

`, frameworkToSnakeCase(route.HandlerName))
	}

	f.generatedFiles["src/main.rs"] = mainContent

	return repo, nil
}

// integrateAxum generates Axum code
func (f *FrameworkIntegrator) integrateAxum(repo *uniast.Repository) (*uniast.Repository, error) {
	mainContent := `use axum::{routing::get, routing::post, Router, Json};
use serde_json::json;
use std::net::SocketAddr;

#[tokio::main]
async fn main() {
    let app = Router::new()
`

	for _, route := range f.routes {
		method := strings.ToLower(route.Method)
		mainContent += fmt.Sprintf("        .route(\"%s\", %s(%s))\n",
			route.Path, method, frameworkToSnakeCase(route.HandlerName))
	}

	mainContent += `;

    let addr = SocketAddr::from(([127, 0, 0, 1], 8080));
    axum::Server::bind(&addr)
        .serve(app.into_make_service())
        .await
        .unwrap();
}

`

	// Add handler functions
	for _, route := range f.routes {
		mainContent += fmt.Sprintf(`async fn %s() -> Json<serde_json::Value> {
    Json(json!({"message": "success"}))
}

`, frameworkToSnakeCase(route.HandlerName))
	}

	f.generatedFiles["src/main.rs"] = mainContent

	return repo, nil
}

// integratePython generates Python web framework integration
func (f *FrameworkIntegrator) integratePython(repo *uniast.Repository) (*uniast.Repository, error) {
	switch f.framework {
	case "fastapi":
		return f.integrateFastAPI(repo)
	case "flask":
		return f.integrateFlask(repo)
	default:
		return f.integrateFastAPI(repo) // Default to FastAPI
	}
}

// integrateFastAPI generates FastAPI code
func (f *FrameworkIntegrator) integrateFastAPI(repo *uniast.Repository) (*uniast.Repository, error) {
	mainContent := `from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

`

	for _, route := range f.routes {
		method := strings.ToLower(route.Method)
		mainContent += fmt.Sprintf(`@app.%s("%s")
async def %s():
    """Handler for %s %s"""
    return {"message": "success"}

`, method, route.Path, frameworkToSnakeCase(route.HandlerName), route.Method, route.Path)
	}

	mainContent += `
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
`

	f.generatedFiles["src/main.py"] = mainContent

	return repo, nil
}

// integrateFlask generates Flask code
func (f *FrameworkIntegrator) integrateFlask(repo *uniast.Repository) (*uniast.Repository, error) {
	mainContent := `from flask import Flask, jsonify

app = Flask(__name__)

`

	for _, route := range f.routes {
		methods := fmt.Sprintf("[\"%s\"]", route.Method)
		mainContent += fmt.Sprintf(`@app.route("%s", methods=%s)
def %s():
    """Handler for %s %s"""
    return jsonify({"message": "success"})

`, route.Path, methods, frameworkToSnakeCase(route.HandlerName), route.Method, route.Path)
	}

	mainContent += `
if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080, debug=True)
`

	f.generatedFiles["src/main.py"] = mainContent

	return repo, nil
}

// getGoDependencies returns Go dependencies
func (f *FrameworkIntegrator) getGoDependencies() []string {
	switch f.framework {
	case "gin":
		return []string{"github.com/gin-gonic/gin v1.9.1"}
	case "echo":
		return []string{"github.com/labstack/echo/v4 v4.11.4"}
	default:
		return []string{"github.com/gin-gonic/gin v1.9.1"}
	}
}

// getRustDependencies returns Rust dependencies
func (f *FrameworkIntegrator) getRustDependencies() []string {
	switch f.framework {
	case "actix":
		return []string{
			"actix-web = \"4\"",
			"serde = { version = \"1\", features = [\"derive\"] }",
			"serde_json = \"1\"",
			"tokio = { version = \"1\", features = [\"full\"] }",
		}
	case "axum":
		return []string{
			"axum = \"0.7\"",
			"serde = { version = \"1\", features = [\"derive\"] }",
			"serde_json = \"1\"",
			"tokio = { version = \"1\", features = [\"full\"] }",
		}
	default:
		return []string{"actix-web = \"4\"", "serde_json = \"1\""}
	}
}

// getPythonDependencies returns Python dependencies
func (f *FrameworkIntegrator) getPythonDependencies() []string {
	switch f.framework {
	case "fastapi":
		return []string{"fastapi>=0.100.0", "uvicorn>=0.23.0", "pydantic>=2.0"}
	case "flask":
		return []string{"flask>=3.0.0"}
	default:
		return []string{"fastapi>=0.100.0", "uvicorn>=0.23.0"}
	}
}

// Helper functions

func extractAnnotationValue(content, key, defaultValue string) string {
	// Simplified annotation value extraction
	// Look for patterns like: @Mapping("value") or @Mapping(value = "value")
	start := strings.Index(content, "(")
	if start == -1 {
		return defaultValue
	}
	end := strings.Index(content[start:], ")")
	if end == -1 {
		return defaultValue
	}

	params := content[start+1 : start+end]
	params = strings.Trim(params, " \"'")

	if key == "" {
		// Direct value
		if !strings.Contains(params, "=") {
			return strings.Trim(params, " \"'")
		}
	}

	// Look for key = "value"
	if idx := strings.Index(params, key); idx != -1 {
		valueStart := strings.Index(params[idx:], "=")
		if valueStart != -1 {
			rest := params[idx+valueStart+1:]
			rest = strings.TrimLeft(rest, " \"'")
			end := strings.IndexAny(rest, "\",)")
			if end != -1 {
				return rest[:end]
			}
			return strings.Trim(rest, " \"'")
		}
	}

	return defaultValue
}

func extractNextMethodName(content string) string {
	// Find the next method name after an annotation
	// Look for pattern: public ReturnType methodName(
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i == 0 {
			continue // Skip annotation line
		}
		line = strings.TrimSpace(line)
		if strings.Contains(line, "(") && !strings.HasPrefix(line, "@") {
			// Extract method name
			parts := strings.Fields(line)
			for j, part := range parts {
				if strings.Contains(part, "(") {
					name := strings.Split(part, "(")[0]
					return name
				}
				if j > 0 && len(parts) > j+1 && strings.Contains(parts[j+1], "(") {
					return part
				}
			}
		}
	}
	return "handler"
}

func frameworkToCamelCase(s string) string {
	if s == "" {
		return s
	}
	// First letter lowercase
	return strings.ToLower(s[:1]) + s[1:]
}

func frameworkToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
