// Copyright 2025 CloudWeGo Authors
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

/**
 * Copyright 2024 ByteDance Inc.
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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang"
	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/translate"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/agent"
	"github.com/cloudwego/abcoder/llm/mcp"
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/cloudwego/abcoder/version"
	"github.com/cloudwego/eino/schema"
)

const Usage = `abcoder <Action> [Language] <Path> [Flags]
Action:
   parse        parse the specific repo and write its UniAST (to stdout by default)
   write        write the specific UniAST back to codes
   translate    translate code from one language to another (e.g., java to go)
   mcp          run as a MCP server for all repo ASTs (*.json) in the specific directory
   agent        run as an Agent for all repo ASTs (*.json) in the specific directory. WIP: only support code-analyzing at present.
   skills       manage skills (list, install, etc.)
   version      print the version of abcoder
Language:
   go           for golang codes
   rust         for rust codes
   cxx          for c codes (cpp support is on the way)
   python       for python codes
   ts           for typescript codes
   js           for javascript codes
   java         for java codes
`

func main() {
	flags := flag.NewFlagSet("abcoder", flag.ExitOnError)

	flagHelp := flags.Bool("h", false, "Show help message.")
	flagVerbose := flags.Bool("verbose", false, "Verbose mode.")
	flagOutput := flags.String("o", "", "Output path.")
	flagLsp := flags.String("lsp", "", "Specify the language server path.")
	javaHome := flags.String("java-home", "", "java home")

	var opts lang.ParseOptions
	flags.BoolVar(&opts.LoadExternalSymbol, "load-external-symbol", false, "load external symbols into results")
	flags.BoolVar(&opts.NoNeedComment, "no-need-comment", false, "not need comment (only works for Go now)")
	flags.BoolVar(&opts.NotNeedTest, "no-need-test", false, "not need parse test files (only works for Go now)")
	flags.BoolVar(&opts.LoadByPackages, "load-by-packages", false, "load by packages (only works for Go now)")
	flags.Var((*StringArray)(&opts.Excludes), "exclude", "exclude files or directories, support multiple values")
	flags.StringVar(&opts.RepoID, "repo-id", "", "specify the repo id")
	flags.StringVar(&opts.TSConfig, "tsconfig", "", "tsconfig path (only works for TS now)")
	flags.Var((*StringArray)(&opts.TSSrcDir), "ts-src-dir", "src-dir path (only works for TS now)")

	var wopts lang.WriteOptions
	flags.StringVar(&wopts.Compiler, "compiler", "", "destination compiler path.")

	var aopts agent.AgentOptions
	flags.IntVar(&aopts.MaxSteps, "agent-max-steps", 50, "specify the max steps that the agent can run for each time")
	flags.IntVar(&aopts.MaxHistories, "agent-max-histories", 10, "specify the max histories that the agent can use")

	var skillName string
	flags.StringVar(&skillName, "skill", "", "specify skill name to use (empty for auto-match)")

	flags.Usage = func() {
		fmt.Fprint(os.Stderr, Usage)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flags.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flags.Usage()
		os.Exit(1)
	}
	action := strings.ToLower(os.Args[1])

	switch action {
	case "version":
		fmt.Fprintf(os.Stdout, "%s\n", version.Version)

	case "parse":
		language, uri := parseArgsAndFlags(flags, true, flagHelp, flagVerbose)

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
			opts.Verbose = true
		}

		opts.Language = language

		if language == uniast.TypeScript {
			if err := parseTSProject(context.Background(), uri, opts, flagOutput); err != nil {
				log.Error("Failed to parse: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if flagLsp != nil {
			opts.LSP = *flagLsp
		}

		lspOptions := make(map[string]string)
		if javaHome != nil {
			lspOptions["java.home"] = *javaHome
		}
		opts.LspOptions = lspOptions

		out, err := lang.Parse(context.Background(), uri, opts)
		if err != nil {
			log.Error("Failed to parse: %v\n", err)
			os.Exit(1)
		}

		if flagOutput != nil && *flagOutput != "" {
			if err := utils.MustWriteFile(*flagOutput, out); err != nil {
				log.Error("Failed to write output: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", out)
		}

	case "write":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Argument Path is required\n")
			os.Exit(1)
		}

		repo, err := uniast.LoadRepo(uri)
		if err != nil {
			log.Error("Failed to load repo: %v\n", err)
			os.Exit(1)
		}

		if flagOutput != nil && *flagOutput != "" {
			wopts.OutputDir = *flagOutput
		} else {
			wopts.OutputDir = filepath.Base(repo.Path)
		}

		if err := lang.Write(context.Background(), repo, wopts); err != nil {
			log.Error("Failed to write: %v\n", err)
			os.Exit(1)
		}

	case "mcp":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Argument Path is required\n")
			os.Exit(1)
		}

		svr := mcp.NewServer(mcp.ServerOptions{
			ServerName:    "abcoder",
			ServerVersion: version.Version,
			Verbose:       *flagVerbose,
			ASTReadToolsOptions: tool.ASTReadToolsOptions{
				RepoASTsDir: uri,
			},
		})
		if err := svr.ServeStdio(); err != nil {
			log.Error("Failed to run MCP server: %v\n", err)
			os.Exit(1)
		}

	case "translate":
		srcLang, dstLang, uri := parseTranslateArgs(flags, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Argument Path is required\n")
			os.Exit(1)
		}

		// Validate destination language (currently only java to go is supported)
		if srcLang != uniast.Java {
			log.Error("Currently only Java to Go translation is supported. Source language must be 'java'\n")
			os.Exit(1)
		}
		if dstLang != uniast.Golang {
			log.Error("Currently only Java to Go translation is supported. Destination language must be 'go'\n")
			os.Exit(1)
		}

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
		}

		// Parse Java project to UniAST
		parseOpts := lang.ParseOptions{
			CollectOption: collect.CollectOption{
				Language: srcLang,
			},
		}
		if flagLsp != nil {
			parseOpts.LSP = *flagLsp
		}
		lspOptions := make(map[string]string)
		if javaHome != nil {
			lspOptions["java.home"] = *javaHome
		}
		parseOpts.LspOptions = lspOptions

		astJSON, err := lang.Parse(context.Background(), uri, parseOpts)
		if err != nil {
			log.Error("Failed to parse Java project: %v\n", err)
			os.Exit(1)
		}

		// Write AST to temp file for agent
		tempASTDir := filepath.Join(os.TempDir(), "abcoder-translate-asts")
		os.MkdirAll(tempASTDir, 0755)
		tempASTFile := filepath.Join(tempASTDir, "java-repo.json")
		if err := utils.MustWriteFile(tempASTFile, astJSON); err != nil {
			log.Error("Failed to write AST file: %v\n", err)
			os.Exit(1)
		}

		// Load Java repository
		javaRepo, err := uniast.LoadRepo(tempASTFile)
		if err != nil {
			log.Error("Failed to load Java repository: %v\n", err)
			os.Exit(1)
		}

		// Prepare translation
		// GoModuleName will be auto-derived from groupId in Translate function
		// If empty, Translate will extract groupId from the first module
		translateOpts := translate.TranslateOptions{
			GoModuleName: "", // Auto-derive from groupId
			OutputDir:    "",
		}
		if flagOutput != nil && *flagOutput != "" {
			translateOpts.OutputDir = *flagOutput
		} else {
			translateOpts.OutputDir = filepath.Base(uri) + "-go"
		}

		log.Info("Java UniAST generated and saved to: %s\n", tempASTFile)

		// Create translator agent (LLM)
		translatorOpts := agent.TranslatorOptions{
			ModelConfig: llm.ModelConfig{
				APIType:   llm.NewModelType(os.Getenv("API_TYPE")),
				APIKey:    os.Getenv("API_KEY"),
				ModelName: os.Getenv("MODEL_NAME"),
				BaseURL:   os.Getenv("BASE_URL"),
			},
			MaxSteps: 100,
			ASTsDir:  tempASTDir,
		}

		if translatorOpts.APIType == llm.ModelTypeUnknown {
			log.Error("env API_TYPE is required for translation")
			os.Exit(1)
		}
		if translatorOpts.APIKey == "" {
			log.Error("env API_KEY is required for translation")
			os.Exit(1)
		}
		if translatorOpts.ModelName == "" {
			log.Error("env MODEL_NAME is required for translation")
			os.Exit(1)
		}

		// Create output directory
		outputDir := translateOpts.OutputDir
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Error("Failed to create output directory: %v\n", err)
			os.Exit(1)
		}

		// Step 2: Convert Java UniAST to Go UniAST (structure-preserving)
		log.Info("Converting Java UniAST to Go UniAST...\n")

		// Use translate package to create Go UniAST structure from Java UniAST
		goRepo, err := translate.Translate(context.Background(), javaRepo, translateOpts)
		if err != nil {
			log.Error("Failed to create Go UniAST structure: %v\n", err)
			os.Exit(1)
		}

		// Step 3: Translate Content fields per-package using LLM
		log.Info("Translating code content per package using LLM...\n")

		successCount := 0
		totalPkgs := 0

		for goModPath, goMod := range goRepo.Modules {
			if goMod.IsExternal() {
				continue
			}
			for goPkgPath, goPkg := range goMod.Packages {
				totalPkgs++
				log.Info("[Package %d] Translating content: %s\n", totalPkgs, goPkgPath)

				// Collect all code content that needs translation
				var contentToTranslate []struct {
					nodeType string // "function", "type", "var"
					name     string
					javaCode string
				}

				for name, fn := range goPkg.Functions {
					if fn.Content != "" {
						contentToTranslate = append(contentToTranslate, struct {
							nodeType string
							name     string
							javaCode string
						}{"function", name, fn.Content})
					}
				}
				for name, tp := range goPkg.Types {
					if tp.Content != "" {
						contentToTranslate = append(contentToTranslate, struct {
							nodeType string
							name     string
							javaCode string
						}{"type", name, tp.Content})
					}
				}
				for name, v := range goPkg.Vars {
					if v.Content != "" {
						contentToTranslate = append(contentToTranslate, struct {
							nodeType string
							name     string
							javaCode string
						}{"var", name, v.Content})
					}
				}

				if len(contentToTranslate) == 0 {
					log.Info("  Skipping: No code content to translate\n")
					continue
				}

				// Build combined prompt for all content in this package
				var contentList string
				for i, item := range contentToTranslate {
					contentList += fmt.Sprintf("\n--- %s: %s ---\n%s\n", item.nodeType, item.name, item.javaCode)
					if i > 10 { // Limit to avoid context overflow
						contentList += fmt.Sprintf("\n... and %d more items\n", len(contentToTranslate)-i-1)
						break
					}
				}

				translationPrompt := fmt.Sprintf(`You are a code translator. Translate Java code to idiomatic Go code.

IMPORTANT:
- Output ONLY a JSON object, no tool calls
- Do NOT use any tools like translate_node
- Return translations in this exact format:

{"translations": [{"name": "item_name", "go_code": "translated code"}]}

Target Go package: %s
Target Go module: %s

Translation rules:
1. String → string, int → int, long → int64, boolean → bool
2. List<T> → []T, Map<K,V> → map[K]V, Set<T> → map[T]bool
3. public → PascalCase (exported), private → camelCase (unexported)
4. class → struct, interface → interface
5. throws Exception → (result, error) return pattern
6. static methods → package-level functions
7. this.field → receiver.field

Java code to translate:
%s

Output ONLY valid JSON. Do not use any tools.`, goPkgPath, goModPath, contentList)

				// Call LLM directly without tools
				response, err := callLLMWithoutTools(context.Background(), translatorOpts.ModelConfig, translationPrompt)
				if err != nil {
					log.Info("  Error: LLM translation failed: %v\n", err)
					continue
				}

				// Parse translations
				translations := parseTranslations(response)
				if len(translations) == 0 {
					log.Info("  Warning: No translations parsed\n")
					continue
				}

				// Apply translations to UniAST with flexible name matching
				appliedCount := 0
				for _, tr := range translations {
					if tr.GoCode == "" {
						continue
					}

					// Try to match functions with flexible name matching
					for fnName, fn := range goPkg.Functions {
						if matchTranslationName(tr.Name, fnName) {
							fn.Content = tr.GoCode
							appliedCount++
							break
						}
					}

					// Try to match types
					for typeName, tp := range goPkg.Types {
						if matchTranslationName(tr.Name, typeName) {
							tp.Content = tr.GoCode
							appliedCount++
							break
						}
					}

					// Try to match vars
					for varName, v := range goPkg.Vars {
						if matchTranslationName(tr.Name, varName) {
							v.Content = tr.GoCode
							appliedCount++
							break
						}
					}
				}

				log.Info("  Applied %d/%d translations\n", appliedCount, len(contentToTranslate))
				if appliedCount > 0 {
					successCount++
				}
			}
		}

		log.Info("Content translation completed: %d/%d packages with translations\n", successCount, totalPkgs)

		// Save Go UniAST to file for debugging
		goASTFile := filepath.Join(tempASTDir, "go-repo.json")
		goASTBytes, _ := json.MarshalIndent(goRepo, "", "  ")
		if err := utils.MustWriteFile(goASTFile, goASTBytes); err != nil {
			log.Info("Warning: Failed to save Go UniAST: %v\n", err)
		} else {
			log.Info("Go UniAST saved to: %s\n", goASTFile)
		}

		// Step 4: Use Go Writer to write code from UniAST
		log.Info("Writing Go code from UniAST to: %s\n", outputDir)
		writeOpts := lang.WriteOptions{
			OutputDir: outputDir,
			Compiler:  "go",
		}
		if err := lang.Write(context.Background(), goRepo, writeOpts); err != nil {
			log.Error("Failed to write Go code from UniAST: %v\n", err)
			// Don't exit, try to continue with go.mod generation
		}

		// Generate go.mod (optional, Writer may handle this)
		if modName, err := generateGoMod(outputDir, translateOpts.GoModuleName); err != nil {
			log.Info("Failed to generate go.mod: %v\n", err)
		} else {
			log.Info("go.mod generated with module name: %s\n", modName)
		}

		// Run go mod tidy
		if err := runGoModTidy(outputDir); err != nil {
			log.Info("Failed to run go mod tidy: %v\n", err)
		}

		// Try to build
		if err := runGoBuild(outputDir); err != nil {
			log.Info("Go build failed: %v\n", err)
		}

		log.Info("Translation completed successfully!\n")
		log.Info("Java UniAST: %s\n", tempASTFile)
		log.Info("Go code written to: %s\n", outputDir)

	case "agent":
		_, uri := parseArgsAndFlags(flags, false, flagHelp, flagVerbose)
		if uri == "" {
			log.Error("Argument Path is required\n")
			os.Exit(1)
		}

		aopts.ASTsDir = uri
		aopts.Model.APIType = llm.NewModelType(os.Getenv("API_TYPE"))
		if aopts.Model.APIType == llm.ModelTypeUnknown {
			log.Error("env API_TYPE is required")
			os.Exit(1)
		}
		aopts.Model.APIKey = os.Getenv("API_KEY")
		if aopts.Model.APIKey == "" {
			log.Error("env API_KEY is required")
			os.Exit(1)
		}
		aopts.Model.ModelName = os.Getenv("MODEL_NAME")
		if aopts.Model.ModelName == "" {
			log.Error("env MODEL_NAME is required")
			os.Exit(1)
		}
		aopts.Model.BaseURL = os.Getenv("BASE_URL")

		// 如果指定了 skill，使用 skill-based agent
		if skillName != "" {
			runSkillAgent(context.Background(), uri, skillName, aopts)
		} else {
			// 使用 coordinator 自动匹配 skill
			runCoordinatorAgent(context.Background(), uri, aopts)
		}

	case "skills":
		handleSkillsCommand(flags, flagHelp, flagVerbose)

	}
}

func parseArgsAndFlags(flags *flag.FlagSet, needLang bool, flagHelp *bool, flagVerbose *bool) (language uniast.Language, uri string) {
	if len(os.Args) < 3 {
		flags.Usage()
		os.Exit(1)
	}

	if needLang {
		language = uniast.NewLanguage(os.Args[2])
		if language == uniast.Unknown {
			fmt.Fprintf(os.Stderr, "unsupported language: %s\n", os.Args[2])
			os.Exit(1)
		}
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "argument Path is required\n")
			os.Exit(1)
		}
		uri = os.Args[3]
		if len(os.Args) > 4 {
			flags.Parse(os.Args[4:])
		}
	} else {
		uri = os.Args[2]
		if len(os.Args) > 3 {
			flags.Parse(os.Args[3:])
		}
	}

	if flagHelp != nil && *flagHelp {
		flags.Usage()
		os.Exit(0)
	}

	if flagVerbose != nil && *flagVerbose {
		log.SetLogLevel(log.DebugLevel)
	}

	return language, uri
}

func parseTranslateArgs(flags *flag.FlagSet, flagHelp *bool, flagVerbose *bool) (srcLang uniast.Language, dstLang uniast.Language, uri string) {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: abcoder translate <src-lang> <dst-lang> <path>\n")
		fmt.Fprintf(os.Stderr, "Example: abcoder translate java go ./my-java-project\n")
		os.Exit(1)
	}

	srcLang = uniast.NewLanguage(os.Args[2])
	if srcLang == uniast.Unknown {
		fmt.Fprintf(os.Stderr, "unsupported source language: %s\n", os.Args[2])
		os.Exit(1)
	}

	dstLang = uniast.NewLanguage(os.Args[3])
	if dstLang == uniast.Unknown {
		fmt.Fprintf(os.Stderr, "unsupported destination language: %s\n", os.Args[3])
		os.Exit(1)
	}

	uri = os.Args[4]
	if len(os.Args) > 5 {
		flags.Parse(os.Args[5:])
	}

	if flagHelp != nil && *flagHelp {
		flags.Usage()
		os.Exit(0)
	}

	if flagVerbose != nil && *flagVerbose {
		log.SetLogLevel(log.DebugLevel)
	}

	return srcLang, dstLang, uri
}

type StringArray []string

func (s *StringArray) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *StringArray) String() string {
	return strings.Join(*s, ",")
}

func parseTSProject(ctx context.Context, repoPath string, opts lang.ParseOptions, outputFlag *string) error {
	if outputFlag == nil {
		return fmt.Errorf("output path is required")
	}

	parserPath, err := exec.LookPath("abcoder-ts-parser")
	if err != nil {
		log.Info("abcoder-ts-parser not found, installing...")
		cmd := exec.Command("npm", "install", "-g", "abcoder-ts-parser")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install abcoder-ts-parser: %v", err)
		}
		parserPath, err = exec.LookPath("abcoder-ts-parser")
		if err != nil {
			return fmt.Errorf("failed to find abcoder-ts-parser after installation: %v", err)
		}
	}

	args := []string{"parse", repoPath}
	if len(opts.TSSrcDir) > 0 {
		args = append(args, "--src", strings.Join(opts.TSSrcDir, ","))
	}
	if opts.TSConfig != "" {
		args = append(args, "--tsconfig", opts.TSConfig)
	}
	if *outputFlag != "" {
		args = append(args, "--output", *outputFlag)
	}

	cmd := exec.CommandContext(ctx, parserPath, args...)
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=65536")
	cmd.Env = append(cmd.Env, "ABCODER_TOOL_VERSION="+version.Version)
	cmd.Env = append(cmd.Env, "ABCODER_AST_VERSION="+uniast.Version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info("Running abcoder-ts-parser with args: %v", args)

	return cmd.Run()
}

// extractGoCode extracts Go code from LLM response, removing markdown code blocks if present
func extractGoCode(response string) string {
	response = strings.TrimSpace(response)

	// Remove markdown code blocks
	if strings.HasPrefix(response, "```go") {
		response = strings.TrimPrefix(response, "```go")
		response = strings.TrimSuffix(response, "```")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		idx := strings.Index(response, "\n")
		if idx >= 0 {
			response = response[idx+1:]
		}
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

// extractGoUniASTJSON extracts Go UniAST JSON from LLM response
func extractGoUniASTJSON(response string) string {
	response = strings.TrimSpace(response)

	// Try to find JSON in the response
	// Look for JSON object start
	startIdx := strings.Index(response, "{")
	if startIdx == -1 {
		return ""
	}

	// Find matching closing brace
	braceCount := 0
	endIdx := startIdx
	for i := startIdx; i < len(response); i++ {
		if response[i] == '{' {
			braceCount++
		} else if response[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx > startIdx {
		jsonStr := response[startIdx:endIdx]
		// Validate it's valid JSON by trying to parse it
		var test interface{}
		if err := json.Unmarshal([]byte(jsonStr), &test); err == nil {
			return jsonStr
		}
	}

	// Fallback: try to extract from markdown code blocks
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // len("```json")
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Last resort: return the whole response if it looks like JSON
	if strings.HasPrefix(response, "{") && strings.HasSuffix(response, "}") {
		return response
	}

	return ""
}

// determineGoFilePath determines the output Go file path based on package path and original file path
func determineGoFilePath(outputDir, goPkgPath, originalPath string) string {
	// Get base filename and change extension to .go
	baseName := filepath.Base(originalPath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".go"

	// If package path is different, create package directory structure
	if goPkgPath != "" && goPkgPath != "." {
		// Convert package path to directory structure
		// For simple packages like "simple", just use the package name as directory
		// For full paths like "github.com/example/project", extract the relative part
		pkgDir := goPkgPath

		// If it's a full module path, extract the relative part
		if strings.HasPrefix(pkgDir, "github.com/") {
			parts := strings.Split(pkgDir, "/")
			if len(parts) > 2 {
				// Use parts after github.com/org/repo
				pkgDir = strings.Join(parts[2:], "/")
			} else {
				// Just use the last part
				pkgDir = parts[len(parts)-1]
			}
		} else if strings.Contains(pkgDir, "/") {
			// Already a path, use as is
		} else {
			// Simple package name, use as directory
		}

		// Create directory structure
		if pkgDir != "" && pkgDir != "." {
			return filepath.Join(outputDir, pkgDir, baseName)
		}
	}

	return filepath.Join(outputDir, baseName)
}

// packageMiniAST represents a minimal UniAST structure for a single package
type packageMiniAST struct {
	Package   *uniast.Package             `json:"package,omitempty"`
	Functions map[string]*uniast.Function `json:"functions,omitempty"`
	Types     map[string]*uniast.Type     `json:"types,omitempty"`
	Vars      map[string]*uniast.Var      `json:"vars,omitempty"`
	Files     map[string]*uniast.File     `json:"files,omitempty"`
	Graph     map[string]*uniast.Node     `json:"graph,omitempty"`
}

// buildPackageUniAST builds a mini UniAST containing only one package
func buildPackageUniAST(repo *uniast.Repository, modName, pkgPath string, pkg *uniast.Package, mod *uniast.Module) *packageMiniAST {
	mini := &packageMiniAST{
		Package:   pkg,
		Functions: make(map[string]*uniast.Function),
		Types:     make(map[string]*uniast.Type),
		Vars:      make(map[string]*uniast.Var),
		Files:     make(map[string]*uniast.File),
		Graph:     make(map[string]*uniast.Node),
	}

	// Copy functions
	for name, fn := range pkg.Functions {
		mini.Functions[name] = fn
	}

	// Copy types
	for name, tp := range pkg.Types {
		mini.Types[name] = tp
	}

	// Copy vars
	for name, v := range pkg.Vars {
		mini.Vars[name] = v
	}

	// Copy relevant files
	for path, file := range mod.Files {
		if file.Package == pkgPath {
			mini.Files[path] = file
		}
	}

	// Copy relevant graph nodes
	for id, node := range repo.Graph {
		identity := uniast.NewIdentityFromString(id)
		if identity.ModPath == modName && identity.PkgPath == pkgPath {
			mini.Graph[id] = node
		}
	}

	return mini
}

// mergePackageIntoRepo merges a translated package mini AST into the Go repository
func mergePackageIntoRepo(goRepo *uniast.Repository, goMod *uniast.Module, goPkgPath string, mini *packageMiniAST) {
	// Create or get the package
	goPkg, exists := goMod.Packages[goPkgPath]
	if !exists {
		goPkg = uniast.NewPackage(goPkgPath)
		goMod.Packages[goPkgPath] = goPkg
	}

	// Merge functions
	if mini.Functions != nil {
		for name, fn := range mini.Functions {
			if fn != nil {
				// Update identity to use Go module
				fn.ModPath = goMod.Name
				fn.PkgPath = goPkgPath
				goPkg.Functions[name] = fn
			}
		}
	}
	if mini.Package != nil && mini.Package.Functions != nil {
		for name, fn := range mini.Package.Functions {
			if fn != nil {
				fn.ModPath = goMod.Name
				fn.PkgPath = goPkgPath
				goPkg.Functions[name] = fn
			}
		}
	}

	// Merge types
	if mini.Types != nil {
		for name, tp := range mini.Types {
			if tp != nil {
				tp.ModPath = goMod.Name
				tp.PkgPath = goPkgPath
				goPkg.Types[name] = tp
			}
		}
	}
	if mini.Package != nil && mini.Package.Types != nil {
		for name, tp := range mini.Package.Types {
			if tp != nil {
				tp.ModPath = goMod.Name
				tp.PkgPath = goPkgPath
				goPkg.Types[name] = tp
			}
		}
	}

	// Merge vars
	if mini.Vars != nil {
		for name, v := range mini.Vars {
			if v != nil {
				v.ModPath = goMod.Name
				v.PkgPath = goPkgPath
				goPkg.Vars[name] = v
			}
		}
	}
	if mini.Package != nil && mini.Package.Vars != nil {
		for name, v := range mini.Package.Vars {
			if v != nil {
				v.ModPath = goMod.Name
				v.PkgPath = goPkgPath
				goPkg.Vars[name] = v
			}
		}
	}

	// Merge files
	if mini.Files != nil {
		for path, file := range mini.Files {
			if file != nil {
				// Convert file path to Go style
				goFilePath := convertJavaFilePathToGo(path, goPkgPath)
				file.Package = goPkgPath
				goMod.Files[goFilePath] = file
			}
		}
	}

	// Merge graph nodes
	if mini.Graph != nil {
		for id, node := range mini.Graph {
			if node != nil {
				// Update node identity
				newId := goMod.Name + "?" + goPkgPath + "#" + node.Name
				goRepo.Graph[newId] = node
			} else {
				goRepo.Graph[id] = node
			}
		}
	}
}

// convertJavaFilePathToGo converts a Java file path to Go file path
func convertJavaFilePathToGo(javaPath, goPkgPath string) string {
	// Extract filename and change extension
	baseName := filepath.Base(javaPath)
	baseName = strings.TrimSuffix(baseName, ".java")
	// Convert to snake_case for Go files
	goFileName := strings.ToLower(baseName) + ".go"

	// Create path based on package
	pkgDir := filepath.Base(goPkgPath)
	return filepath.Join(pkgDir, goFileName)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// translation represents a single code translation result
type translation struct {
	Name   string `json:"name"`
	GoCode string `json:"go_code"`
}

// translationsResponse represents the LLM response format
type translationsResponse struct {
	Translations []translation `json:"translations"`
}

// callLLMWithoutTools calls LLM directly without tools for simple chat completion
// This avoids tool calling interference when we just need JSON output
func callLLMWithoutTools(ctx context.Context, modelConfig llm.ModelConfig, prompt string) (string, error) {
	// Create ChatModel
	chatModel := llm.NewChatModel(modelConfig)

	// Build messages
	messages := []*schema.Message{
		schema.UserMessage(prompt),
	}

	// Call Generate directly (no tools)
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM Generate failed: %w", err)
	}

	if response == nil {
		return "", fmt.Errorf("LLM returned nil response")
	}

	return response.Content, nil
}

// matchTranslationName checks if a translation name matches a UniAST node name
// Supports flexible matching because LLM may return different name formats
// e.g., "GetCreatedAt" should match "BaseEntity.GetCreatedAt()" or "BaseEntity.GetCreatedAt"
func matchTranslationName(translationName, nodeName string) bool {
	// Exact match
	if translationName == nodeName {
		return true
	}

	// Case-insensitive exact match
	if strings.EqualFold(translationName, nodeName) {
		return true
	}

	// Remove parameters from both names for comparison
	// e.g., "GetCreatedAt()" -> "GetCreatedAt"
	cleanTranslation := strings.TrimSuffix(translationName, "()")
	cleanTranslation = strings.Split(cleanTranslation, "(")[0]

	cleanNode := strings.TrimSuffix(nodeName, "()")
	cleanNode = strings.Split(cleanNode, "(")[0]

	// Match if the translation name equals the full node name or just the method part
	if cleanTranslation == cleanNode {
		return true
	}

	// Check if translation name matches just the method/type name (after last dot)
	// e.g., "GetCreatedAt" matches "BaseEntity.GetCreatedAt"
	if idx := strings.LastIndex(cleanNode, "."); idx != -1 {
		methodName := cleanNode[idx+1:]
		if cleanTranslation == methodName || strings.EqualFold(cleanTranslation, methodName) {
			return true
		}
	}

	// Check if translation name matches just the method/type name (after ::)
	// e.g., "isEmpty" matches "StringUtils::isEmpty"
	if idx := strings.LastIndex(cleanNode, "::"); idx != -1 {
		methodName := cleanNode[idx+2:]
		if cleanTranslation == methodName || strings.EqualFold(cleanTranslation, methodName) {
			return true
		}
	}

	// Check if node name ends with translation name (for class methods)
	if strings.HasSuffix(strings.ToLower(cleanNode), strings.ToLower(cleanTranslation)) {
		return true
	}

	return false
}

// parseTranslations parses the LLM response to extract code translations
func parseTranslations(response string) []translation {
	response = strings.TrimSpace(response)

	// Try to extract JSON from markdown code blocks
	if idx := strings.Index(response, "```json"); idx != -1 {
		start := idx + 7
		end := strings.Index(response[start:], "```")
		if end != -1 {
			response = strings.TrimSpace(response[start : start+end])
		}
	} else if idx := strings.Index(response, "```"); idx != -1 {
		start := idx + 3
		// Skip language tag if present
		if nl := strings.Index(response[start:], "\n"); nl != -1 {
			start += nl + 1
		}
		end := strings.Index(response[start:], "```")
		if end != -1 {
			response = strings.TrimSpace(response[start : start+end])
		}
	}

	// Find JSON object start
	startIdx := strings.Index(response, "{")
	if startIdx == -1 {
		return nil
	}

	// Find matching closing brace
	braceCount := 0
	endIdx := startIdx
	for i := startIdx; i < len(response); i++ {
		if response[i] == '{' {
			braceCount++
		} else if response[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx <= startIdx {
		return nil
	}

	jsonStr := response[startIdx:endIdx]

	// Try to parse as translationsResponse
	var resp translationsResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		log.Debug("Failed to parse translations JSON: %v\n", err)
		return nil
	}

	return resp.Translations
}

// determineGoPackageFilePath determines the output Go file path for a package
func determineGoPackageFilePath(outputDir, goPkgPath string) string {
	// Convert package path to directory structure
	pkgDir := goPkgPath

	// Extract package name (last part of the path)
	pkgName := filepath.Base(goPkgPath)
	if pkgName == "" || pkgName == "." {
		pkgName = "main"
	}

	// If it's a full module path like "github.com/example/project/subpkg"
	// Extract the relative part after the module prefix
	if strings.HasPrefix(pkgDir, "github.com/") {
		parts := strings.Split(pkgDir, "/")
		if len(parts) > 3 {
			// Use parts after github.com/org/repo
			pkgDir = strings.Join(parts[3:], "/")
		} else if len(parts) > 2 {
			// github.com/org/repo -> repo
			pkgDir = parts[len(parts)-1]
		} else {
			pkgDir = pkgName
		}
	} else if strings.Contains(pkgDir, ".") {
		// com.example.package -> com/example/package
		pkgDir = strings.ReplaceAll(pkgDir, ".", "/")
	}

	// Clean up the package directory
	pkgDir = strings.TrimPrefix(pkgDir, "/")
	pkgDir = strings.TrimSuffix(pkgDir, "/")

	// File name is the package name + ".go"
	fileName := pkgName + ".go"

	if pkgDir != "" && pkgDir != "." {
		return filepath.Join(outputDir, pkgDir, fileName)
	}
	return filepath.Join(outputDir, fileName)
}

// writeGoFile writes Go code to a file
func writeGoFile(filePath, goCode string) error {
	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Write file
	return os.WriteFile(filePath, []byte(goCode), 0644)
}

// formatGoFile formats a Go file using gofmt
func formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofmt failed: %v", err)
	}
	return nil
}

// generateGoMod generates a go.mod file
func generateGoMod(outputDir, moduleName string) (string, error) {
	sanitizedModule := sanitizeModuleName(moduleName, outputDir)

	goModPath := filepath.Join(outputDir, "go.mod")
	goVersion := detectGoVersion()

	content := fmt.Sprintf("module %s\n\ngo %s\n", sanitizedModule, goVersion)
	return sanitizedModule, os.WriteFile(goModPath, []byte(content), 0644)
}

// runGoModTidy runs go mod tidy in the output directory
func runGoModTidy(outputDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %v", err)
	}
	return nil
}

// runGoBuild runs go build in the output directory
func runGoBuild(outputDir string) error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %v", err)
	}
	return nil
}

// sanitizeModuleName ensures the module name is valid for go.mod
func sanitizeModuleName(moduleName, outputDir string) string {
	mod := strings.TrimSpace(moduleName)
	if mod == "" {
		mod = filepath.Base(outputDir)
	}
	// If module name is absolute path or starts with '/', convert to example.com/<basename>
	if strings.HasPrefix(mod, "/") || strings.Contains(mod, string(os.PathSeparator)) {
		mod = filepath.Base(mod)
	}
	// Replace spaces with underscore
	mod = strings.ReplaceAll(mod, " ", "_")
	// If no dot, prepend example.com/
	if !strings.Contains(mod, ".") {
		mod = "example.com/" + mod
	}
	return mod
}

// detectGoVersion extracts the major.minor version from `go version`
func detectGoVersion() string {
	// default fallback
	goVersion := "1.21"
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return goVersion
	}
	// Sample: "go version go1.22.5 darwin/arm64"
	toks := strings.Fields(string(out))
	for _, t := range toks {
		if strings.HasPrefix(t, "go1.") {
			ver := strings.TrimPrefix(t, "go")
			parts := strings.Split(ver, ".")
			if len(parts) >= 2 {
				return parts[0] + "." + parts[1]
			}
			return ver
		}
	}
	return goVersion
}

// buildStubGoCode builds a simple Go file content from Java code (fallback when LLM translation fails)
func buildStubGoCode(goPkgPath string, javaCode string, filePath string) string {
	pkgDir := filepath.Base(goPkgPath)
	if pkgDir == "" || pkgDir == "." || pkgDir == "/" {
		pkgDir = "main"
	}
	return fmt.Sprintf(`package %s

// Translated from Java file: %s
// Fallback stub (no LLM translation). Review and refine manually.
/*
%s
*/
`, pkgDir, filePath, javaCode)
}

// scanJavaFiles scans a directory recursively for .java files
func scanJavaFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".java") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// detectJavaPackage detects package name from file path and content
func detectJavaPackage(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return detectJavaPackageFromContent(string(content))
}

// detectJavaPackageFromContent extracts package name from Java source content
func detectJavaPackageFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, ln := range lines {
		line := strings.TrimSpace(ln)
		if strings.HasPrefix(line, "package ") {
			line = strings.TrimPrefix(line, "package ")
			line = strings.TrimSuffix(line, ";")
			return strings.TrimSpace(line)
		}
	}
	return ""
}
