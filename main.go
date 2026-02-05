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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/abcoder/internal/pipeline"
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

	// Translation post-processing options
	var webFramework string
	flags.StringVar(&webFramework, "framework", "", "web framework for translation: gin, echo, actix, fastapi, flask, none (default: auto)")
	var noEntryPoint bool
	flags.BoolVar(&noEntryPoint, "no-entry", false, "skip entry point generation")
	var noConfig bool
	flags.BoolVar(&noConfig, "no-config", false, "skip project config generation (go.mod, Cargo.toml, etc.)")

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

		// Validate source and destination languages
		supportedSrcLangs := []uniast.Language{uniast.Java, uniast.Golang, uniast.Python, uniast.Rust, uniast.Cxx, uniast.TypeScript}
		supportedDstLangs := []uniast.Language{uniast.Golang, uniast.Python, uniast.Rust, uniast.Java, uniast.Cxx}

		srcSupported := false
		for _, l := range supportedSrcLangs {
			if srcLang == l {
				srcSupported = true
				break
			}
		}
		if !srcSupported {
			log.Error("Unsupported source language: %s. Supported: java, go, python, rust, cxx, ts\n", srcLang)
			os.Exit(1)
		}

		dstSupported := false
		for _, l := range supportedDstLangs {
			if dstLang == l {
				dstSupported = true
				break
			}
		}
		if !dstSupported {
			log.Error("Unsupported destination language: %s. Supported: java, go, python, rust, cxx\n", dstLang)
			os.Exit(1)
		}

		if srcLang == dstLang {
			log.Error("Source and destination languages must be different\n")
			os.Exit(1)
		}

		log.Info("Translating %s → %s\n", srcLang, dstLang)

		if flagVerbose != nil && *flagVerbose {
			log.SetLogLevel(log.DebugLevel)
		}

		// Pipeline state for this run (which step failed, attempt N, status)
		pipelineState := &pipeline.PipelineState{
			RunID:          fmt.Sprintf("%d", time.Now().UnixNano()),
			SourceLang:     srcLang,
			TargetLang:     dstLang,
			SourceCodePath: uri,
			OutputPath:     "",
			History:        nil,
		}
		reportPipelineFailureAndExit := func() {
			if n := len(pipelineState.History); n > 0 {
				last := pipelineState.History[n-1]
				log.Info("Pipeline: last step=%s, attempt=%d, status=%s\n", last.StepName, last.Attempt, last.Status)
			}
			os.Exit(1)
		}

		// Parse source project to UniAST
		tempASTDir := filepath.Join(os.TempDir(), "abcoder-translate-asts")
		os.MkdirAll(tempASTDir, 0755)
		tempASTFile := filepath.Join(tempASTDir, fmt.Sprintf("%s-repo.json", srcLang))

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
		parseOpts.TSConfig = opts.TSConfig
		parseOpts.TSSrcDir = opts.TSSrcDir

		var srcRepo *uniast.Repository
		usedExistingUniAST := false
		var existingUniASTPath string
		if stat, err := os.Stat(uri); err == nil {
			var candidatePath string
			if stat.IsDir() {
				candidatePath = filepath.Join(uri, "uniast.json")
			} else if !stat.IsDir() && strings.HasSuffix(strings.ToLower(uri), ".json") {
				candidatePath = uri
			}
			if candidatePath != "" {
				if s, err := os.Stat(candidatePath); err == nil && s != nil && !s.IsDir() {
					loaded, err := uniast.LoadRepo(candidatePath)
					if err == nil {
						srcRepo = loaded
						usedExistingUniAST = true
						existingUniASTPath = candidatePath
						log.Info("Using existing UniAST: %s, skip parsing\n", candidatePath)
					} else {
						// User explicitly passed a .json path; failed load is fatal (don't fall back to parsing a file as directory)
						if candidatePath == uri {
							pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
								StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
							})
							log.Error("Failed to load UniAST from %s: %v\n", candidatePath, err)
							reportPipelineFailureAndExit()
						}
						log.Info("Failed to load existing UniAST, will parse: %v\n", err)
					}
				}
			}
		}
		if !usedExistingUniAST {
			if srcLang == uniast.TypeScript {
				if err := parseTSProject(context.Background(), uri, parseOpts, &tempASTFile); err != nil {
					pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
						StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
					})
					log.Error("Failed to parse TypeScript project: %v\n", err)
					reportPipelineFailureAndExit()
				}
				var err error
				srcRepo, err = uniast.LoadRepo(tempASTFile)
				if err != nil {
					pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
						StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
					})
					log.Error("Failed to load TypeScript repository: %v\n", err)
					reportPipelineFailureAndExit()
				}
			} else {
				astJSON, err := lang.Parse(context.Background(), uri, parseOpts)
				if err != nil {
					pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
						StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
					})
					log.Error("Failed to parse %s project: %v\n", srcLang, err)
					reportPipelineFailureAndExit()
				}
				if err := utils.MustWriteFile(tempASTFile, astJSON); err != nil {
					pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
						StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
					})
					log.Error("Failed to write AST file: %v\n", err)
					reportPipelineFailureAndExit()
				}
				srcRepo, err = uniast.LoadRepo(tempASTFile)
				if err != nil {
					pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
						StepName: "parse", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
					})
					log.Error("Failed to load %s repository: %v\n", srcLang, err)
					reportPipelineFailureAndExit()
				}
			}
		}
		pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
			StepName: "parse", Attempt: 1, Status: pipeline.StepOK, Time: time.Now(),
		})
		if usedExistingUniAST {
			log.Info("%s UniAST: %s\n", srcLang, existingUniASTPath)
		} else {
			log.Info("%s UniAST generated and saved to: %s\n", srcLang, tempASTFile)
		}

		// Determine output directory
		outputDir := ""
		if flagOutput != nil && *flagOutput != "" {
			outputDir = *flagOutput
		} else {
			outputDir = filepath.Base(uri) + "-" + string(dstLang)
		}
		pipelineState.OutputPath = outputDir

		// Create output directory
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Error("Failed to create output directory: %v\n", err)
			os.Exit(1)
		}

		// Setup LLM configuration
		modelConfig := llm.ModelConfig{
			APIType:   llm.NewModelType(os.Getenv("API_TYPE")),
			APIKey:    os.Getenv("API_KEY"),
			ModelName: os.Getenv("MODEL_NAME"),
			BaseURL:   os.Getenv("BASE_URL"),
		}

		if modelConfig.APIType == llm.ModelTypeUnknown {
			log.Error("env API_TYPE is required for translation")
			os.Exit(1)
		}
		if modelConfig.APIKey == "" {
			log.Error("env API_KEY is required for translation")
			os.Exit(1)
		}
		if modelConfig.ModelName == "" {
			log.Error("env MODEL_NAME is required for translation")
			os.Exit(1)
		}

		// Create LLM translator callback
		llmTranslator := createLLMTranslator(modelConfig)

		// Determine web framework if auto
		framework := webFramework
		if framework == "" {
			// Auto-detect based on target language
			switch dstLang {
			case uniast.Golang:
				framework = "gin"
			case uniast.Rust:
				framework = "actix"
			case uniast.Python:
				framework = "fastapi"
			default:
				framework = "none"
			}
		}

		// Prepare translation options using new API
		concurrency := 8
		if s := os.Getenv("TRANSLATE_CONCURRENCY"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 32 {
				concurrency = n
			}
		}
		translateResult := &translate.TranslateResult{}
		translateOpts := translate.TranslateOptions{
			SourceLanguage:     srcLang,
			TargetLanguage:     dstLang,
			TargetModuleName:   "", // Auto-derive from source
			OutputDir:          outputDir,
			LLMTranslator:      llmTranslator,
			Parallel:           true,
			Concurrency:        concurrency,
			WebFramework:       framework,
			GenerateEntryPoint: !noEntryPoint,
			GenerateConfig:     !noConfig,
			Result:             translateResult,
			ProgressCallback: func(done, total int, currentKind, currentNodeID string) {
				if total > 0 {
					pct := 100 * float64(done) / float64(total)
					log.Info("Progress: %d/%d (%.1f%%) current: %s %s\n", done, total, pct, currentKind, currentNodeID)
				}
			},
		}

		// Transform source UniAST to target UniAST with LLM content translation
		log.Info("Translating %s to %s using LLM (Parser → Transform → Writer flow)...\n", srcLang, dstLang)

		// Use TranslateAST to get the target repo (so we can save it)
		targetRepo, err := translate.TranslateAST(context.Background(), srcRepo, translateOpts)
		if err != nil {
			pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
				StepName: "transform", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
			})
			log.Error("Failed to translate: %v\n", err)
			reportPipelineFailureAndExit()
		}
		pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
			StepName: "transform", Attempt: 1, Status: pipeline.StepOK, Time: time.Now(),
		})

		// Save target UniAST to JSON file
		targetASTFile := filepath.Join(tempASTDir, fmt.Sprintf("%s-repo.json", dstLang))
		targetASTJSON, err := json.MarshalIndent(targetRepo, "", "  ")
		if err != nil {
			log.Error("Failed to marshal target AST: %v\n", err)
			os.Exit(1)
		}
		if err := utils.MustWriteFile(targetASTFile, targetASTJSON); err != nil {
			log.Error("Failed to write target AST file: %v\n", err)
			os.Exit(1)
		}
		log.Info("Target UniAST saved to: %s\n", targetASTFile)

		// Validate target UniAST before writing (reject invalid LLM output)
		if err := uniast.ValidateRepository(targetRepo); err != nil {
			pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
				StepName: "validate", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
			})
			log.Error("UniAST validation failed (rejecting output): %v\n", err)
			reportPipelineFailureAndExit()
		}
		pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
			StepName: "validate", Attempt: 1, Status: pipeline.StepOK, Time: time.Now(),
		})
		// Snapshot target UniAST so rollback (e.g. on later failure) can restore; on Fatal validation we never reach here.
		pipelineState.TargetUniAST = pipeline.NewSnapshot("target-uniast", targetRepo, targetASTJSON)

		// Write target code using lang.Write
		err = lang.Write(context.Background(), targetRepo, lang.WriteOptions{
			OutputDir: outputDir,
		})
		if err != nil {
			pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
				StepName: "write", Attempt: 1, Status: pipeline.StepFailed, Error: err.Error(), Time: time.Now(),
			})
			log.Error("Failed to write target code: %v\n", err)
			reportPipelineFailureAndExit()
		}
		pipelineState.History = append(pipelineState.History, pipeline.StepRecord{
			StepName: "write", Attempt: 1, Status: pipeline.StepOK, Time: time.Now(),
		})

		// Report pipeline outcome (last step, attempt, status)
		if n := len(pipelineState.History); n > 0 {
			last := pipelineState.History[n-1]
			log.Info("Pipeline: last step=%s, attempt=%d, status=%s\n", last.StepName, last.Attempt, last.Status)
		}
		if flagVerbose != nil && *flagVerbose {
			for _, rec := range pipelineState.History {
				if rec.Error != "" {
					log.Debug("  step=%s attempt=%d status=%s error=%s\n", rec.StepName, rec.Attempt, rec.Status, rec.Error)
				} else {
					log.Debug("  step=%s attempt=%d status=%s\n", rec.StepName, rec.Attempt, rec.Status)
				}
			}
		}
		// Persist pipeline report (StepHistory) for observability
		if reportPath := filepath.Join(outputDir, "abcoder-pipeline-report.json"); outputDir != "" {
			report := struct {
				RunID   string               `json:"run_id"`
				History []pipeline.StepRecord `json:"history"`
			}{RunID: pipelineState.RunID, History: pipelineState.History}
			if reportJSON, err := json.MarshalIndent(report, "", "  "); err == nil {
				_ = os.WriteFile(reportPath, reportJSON, 0644)
			}
		}
		// Lightweight checkpoint for future resume: translated_ids + source identifier
		if outputDir != "" && translateResult.TranslatedIDs != nil {
			ids := make([]string, 0, len(translateResult.TranslatedIDs))
			for id := range translateResult.TranslatedIDs {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			checkpoint := struct {
				SourcePath    string   `json:"source_path"`
				TranslatedIDs []string `json:"translated_ids"`
			}{SourcePath: uri, TranslatedIDs: ids}
			if checkpointJSON, err := json.MarshalIndent(checkpoint, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(outputDir, "abcoder-translate-checkpoint.json"), checkpointJSON, 0644)
			}
		}

		// Run target language specific post-processing
		switch dstLang {
		case uniast.Golang:
			// Fix invalid imports in generated Go files
			// First try to read module name from go.mod
			moduleName := readGoModuleName(outputDir)
			if moduleName == "" {
				moduleName = translateOpts.TargetModuleName
			}
			if moduleName == "" {
				moduleName = "github.com/example/" + filepath.Base(outputDir)
			}
			if err := fixGoImportsInFiles(outputDir, moduleName); err != nil {
				log.Info("Failed to fix imports: %v\n", err)
			}
			// Run goimports to fix any remaining import issues and format code
			if err := runGoimports(outputDir); err != nil {
				log.Info("Failed to run goimports: %v\n", err)
			}
			// Run go mod tidy
			if err := runGoModTidy(outputDir); err != nil {
				log.Info("Failed to run go mod tidy: %v\n", err)
			}
			// Try to build
			if err := runGoBuild(outputDir); err != nil {
				log.Info("Go build failed: %v\n", err)
			}
		case uniast.Rust:
			// Run cargo check
			if err := runCargoCheck(outputDir); err != nil {
				log.Info("Cargo check failed: %v\n", err)
			}
		case uniast.Python:
			// Python doesn't need compilation, but we can check syntax
			log.Info("Python code generated. Run 'python -m py_compile <file>' to check syntax.\n")
		case uniast.Java:
			// Java compilation would need maven or gradle
			log.Info("Java code generated. Run 'mvn compile' or 'gradle build' to compile.\n")
		case uniast.Cxx:
			// C++ compilation would need cmake or make
			log.Info("C++ code generated. Run 'cmake . && make' or 'g++ -o main *.cpp' to compile.\n")
		}

		log.Info("Translation completed successfully!\n")
		log.Info("Source UniAST: %s\n", tempASTFile)
		log.Info("Target UniAST: %s\n", targetASTFile)
		log.Info("%s code written to: %s\n", dstLang, outputDir)

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
	args = append(args, "--monorepo-mode", "combined")

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

// createLLMTranslator creates an LLM translator callback for the translate package
func createLLMTranslator(modelConfig llm.ModelConfig) translate.LLMTranslateFunc {
	return func(ctx context.Context, req *translate.LLMTranslateRequest) (*translate.LLMTranslateResponse, error) {
		// Use the pre-built prompt from PromptBuilder
		prompt := req.Prompt
		if prompt == "" {
			// Fallback: build a simple prompt
			prompt = fmt.Sprintf("Translate the following %s code to %s:\n\n%s\n\nReturn ONLY the translated code, no explanations.",
				req.SourceLanguage, req.TargetLanguage, req.SourceContent)
		}

		log.Debug("LLM Translation Request:\n  Node: %s\n  Type: %s\n", req.Identity.Name, req.NodeType)

		// Call LLM with retry logic for transient errors
		maxRetries := 3
		var response string
		var err error

		for attempt := 1; attempt <= maxRetries; attempt++ {
			response, err = callLLMWithoutTools(ctx, modelConfig, prompt)
			if err == nil {
				break
			}

			// Check if error is retryable (timeout, connection reset, etc.)
			errStr := err.Error()
			isRetryable := strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "timed out") ||
				strings.Contains(errStr, "connection reset") ||
				strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "EOF") ||
				strings.Contains(errStr, "temporary failure")

			if !isRetryable || attempt == maxRetries {
				log.Error("LLM call failed after %d attempts: %v\n", attempt, err)
				return &translate.LLMTranslateResponse{
					Error: fmt.Sprintf("LLM call failed: %v", err),
				}, nil
			}

			// Exponential backoff: 2s, 4s, 8s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Info("LLM call failed (attempt %d/%d), retrying in %v: %v\n", attempt, maxRetries, backoff, err)
			time.Sleep(backoff)
		}

		// Clean up response - remove markdown code fences if present
		content := cleanCodeResponse(response)

		log.Debug("LLM Translation Response:\n  Content length: %d\n", len(content))

		return &translate.LLMTranslateResponse{
			TargetContent: content,
		}, nil
	}
}

// cleanCodeResponse removes markdown code fences from LLM response
func cleanCodeResponse(response string) string {
	response = strings.TrimSpace(response)

	// Remove language-specific code fence prefixes
	prefixes := []string{
		"```go", "```golang",
		"```rust", "```rs",
		"```python", "```py",
		"```java",
		"```cpp", "```c++", "```cxx",
		"```",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(response, prefix) {
			response = strings.TrimPrefix(response, prefix)
			break
		}
	}

	// Remove trailing ```
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
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
// fixGoImportsInFiles fixes invalid imports in generated Go files
func fixGoImportsInFiles(outputDir, moduleName string) error {
	// First pass: identify main packages (can't be imported)
	mainPackages := make(map[string]bool)
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Check if this file has package main
		if strings.Contains(string(content), "package main") {
			rel, _ := filepath.Rel(outputDir, filepath.Dir(path))
			if rel != "" && rel != "." {
				mainPackages[rel] = true
			}
		}
		return nil
	})

	// Second pass: fix imports and clean up code
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s failed: %v", path, err)
		}

		original := string(content)

		// Step 1: Remove duplicate declarations and clean up
		fixed := cleanupGoCode(original)

		// Step 2: Consolidate and fix all imports (collect, fix paths, move to top)
		fixed = consolidateImports(fixed, moduleName)

		// Step 3: Fix invalid import paths that might remain
		fixed = fixGoImportString(fixed, moduleName, outputDir, mainPackages)

		// Step 4: Auto-fix standard library imports based on usage
		fixed = autoFixStandardImports(fixed)

		// Step 5: Fix cross-package type references
		// Determine current package from directory
		relDir, _ := filepath.Rel(outputDir, filepath.Dir(path))
		currentPkg := filepath.Base(relDir)
		if currentPkg == "." || currentPkg == "" {
			currentPkg = "main"
		}
		fixed = fixCrossPackageTypes(fixed, currentPkg, nil)

		// Only write if changed
		if fixed != original {
			if err := os.WriteFile(path, []byte(fixed), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", path, err)
			}
			log.Info("Fixed imports in: %s\n", path)
		}

		return nil
	})
}

// fixGoImportString fixes invalid imports in Go source code
func fixGoImportString(content, moduleName, outputDir string, mainPackages map[string]bool) string {
	// Get existing package directories
	pkgDirs := make(map[string]bool)
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != outputDir {
			rel, _ := filepath.Rel(outputDir, path)
			if rel != "" && !strings.HasPrefix(rel, ".") {
				pkgDirs[rel] = true
				pkgDirs[filepath.Base(rel)] = true
			}
		}
		return nil
	})

	// Common Java to Go package mappings
	javaToGo := map[string]string{
		"core": "repository", "domain": "model", "entity": "model",
		"entities": "model", "dto": "model", "vo": "model",
		"dao": "repository", "mapper": "repository", "repo": "repository",
		"api": "controller", "rest": "controller", "endpoint": "controller",
		"impl": "service", "business": "service",
		"common": "utils", "util": "utils", "helper": "utils",
		"application": "", // Remove application imports (doesn't exist)
	}

	// Process all import paths in the content
	importBlockRegex := regexp.MustCompile(`(?s)import\s*\(\s*(.*?)\s*\)`)
	singleImportRegex := regexp.MustCompile(`"([^"]+)"`)

	content = importBlockRegex.ReplaceAllStringFunc(content, func(block string) string {
		return singleImportRegex.ReplaceAllStringFunc(block, func(match string) string {
			importPath := strings.Trim(match, `"`)

			// Skip standard library and well-known packages
			if !strings.Contains(importPath, "/") && !strings.Contains(importPath, ".") {
				return match // Keep standard library imports
			}
			if strings.HasPrefix(importPath, "github.com/gin") ||
				strings.HasPrefix(importPath, "golang.org") {
				return match
			}

			// Extract the package name (last segment)
			parts := strings.Split(importPath, "/")
			pkgName := strings.ToLower(parts[len(parts)-1])

			// Check if mapped to empty (should be removed)
			if mapped, ok := javaToGo[pkgName]; ok && mapped == "" {
				return `"REMOVE_THIS_IMPORT"`
			}

			// Check if target is a main package (can't be imported)
			if mainPackages[pkgName] {
				return `"REMOVE_THIS_IMPORT"`
			}

			// If already using module path, check if it's valid
			if strings.HasPrefix(importPath, moduleName) {
				targetPkg := strings.TrimPrefix(importPath, moduleName+"/")
				if pkgDirs[targetPkg] && !mainPackages[targetPkg] {
					return match // Keep valid imports
				}
				pkgName = strings.ToLower(filepath.Base(targetPkg))
			}

			// Check if this package exists directly
			if pkgDirs[pkgName] && !mainPackages[pkgName] {
				return `"` + moduleName + "/" + pkgName + `"`
			}

			// Try common mappings
			if mappedPkg, ok := javaToGo[pkgName]; ok && mappedPkg != "" {
				if pkgDirs[mappedPkg] && !mainPackages[mappedPkg] {
					return `"` + moduleName + "/" + mappedPkg + `"`
				}
			}

			// Try to find a similar package
			for dir := range pkgDirs {
				if !mainPackages[dir] && (strings.Contains(dir, pkgName) || strings.Contains(pkgName, dir)) {
					return `"` + moduleName + "/" + dir + `"`
				}
			}

			// If no match found, remove the import
			return `"REMOVE_THIS_IMPORT"`
		})
	})

	// Remove marked imports and empty lines
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	for _, line := range lines {
		if !strings.Contains(line, "REMOVE_THIS_IMPORT") {
			cleanedLines = append(cleanedLines, line)
		}
	}
	content = strings.Join(cleanedLines, "\n")

	// Fix type references (core.X -> repository.X, etc.)
	content = strings.ReplaceAll(content, "core.", "repository.")
	content = strings.ReplaceAll(content, "web.EmailService", "service.EmailService")
	content = strings.ReplaceAll(content, "web.NewEmailService", "service.NewEmailService")
	content = strings.ReplaceAll(content, "web.UserRegistrationService", "service.UserRegistrationService")
	content = strings.ReplaceAll(content, "web.NewUserRegistrationService", "service.NewUserRegistrationService")

	return content
}

// cleanupGoCode cleans up common issues in generated Go code
func cleanupGoCode(content string) string {
	// Step 0: Fix invalid type declarations like "type pkg.TypeName struct{}" -> "type TypeName struct{}"
	content = fixInvalidTypeDeclarations(content)

	// Step 0.5: Fix common LLM syntax errors
	content = fixCommonLLMErrors(content)

	// Step 1: Remove duplicate declarations (types, structs, methods, functions)
	content = deduplicateDeclarations(content)

	// Step 2: Remove duplicate main functions (keep only the first one)
	mainFuncRegex := regexp.MustCompile(`func main\(\)\s*\{`)
	matches := mainFuncRegex.FindAllStringIndex(content, -1)
	if len(matches) > 1 {
		for i := len(matches) - 1; i > 0; i-- {
			start := matches[i][0]
			braceCount := 0
			funcEnd := start
			inFunc := false
			for j := start; j < len(content); j++ {
				if content[j] == '{' {
					braceCount++
					inFunc = true
				} else if content[j] == '}' {
					braceCount--
					if inFunc && braceCount == 0 {
						funcEnd = j + 1
						break
					}
				}
			}
			if funcEnd > start {
				content = content[:start] + content[funcEnd:]
			}
		}
	}

	// Step 3: Remove empty import blocks
	content = regexp.MustCompile(`import\s*\(\s*\)`).ReplaceAllString(content, "")

	// Step 4: Remove consecutive empty lines (more than 2)
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	return content
}

// fixCommonLLMErrors fixes common syntax errors generated by LLMs
func fixCommonLLMErrors(content string) string {
	// Fix "!strings.TrimSpace(str) == """ -> "strings.TrimSpace(str) != """
	content = regexp.MustCompile(`!\s*strings\.TrimSpace\(([^)]+)\)\s*==\s*""`).
		ReplaceAllString(content, `strings.TrimSpace($1) != ""`)

	// Fix "!StringUtils.IsEmpty(str)" -> "!IsEmpty(str)" (when StringUtils is a type, not instance)
	content = regexp.MustCompile(`!StringUtils\.IsEmpty\(`).
		ReplaceAllString(content, `!IsEmpty(`)

	// Fix "StringUtils.IsEmpty(" -> "IsEmpty(" (same issue)
	content = regexp.MustCompile(`StringUtils\.IsEmpty\(`).
		ReplaceAllString(content, `IsEmpty(`)

	// Fix "u.GetId()" -> "u.GetID()" (Go naming convention)
	content = strings.ReplaceAll(content, ".GetId()", ".GetID()")
	content = strings.ReplaceAll(content, ".SetId(", ".SetID(")

	return content
}

// fixInvalidTypeDeclarations fixes invalid Go type declarations like "type pkg.Name struct{}"
// These are generated when LLM produces Java-style qualified names
func fixInvalidTypeDeclarations(content string) string {
	// Fix "type pkg.TypeName" -> "type TypeName"
	// Match: type <word>.<word> <rest>
	typeRegex := regexp.MustCompile(`type\s+(\w+)\.(\w+)\s+(struct|interface|\w)`)
	content = typeRegex.ReplaceAllString(content, "type $2 $3")

	// Also fix "var pkg.VarName" -> "var VarName"
	varRegex := regexp.MustCompile(`var\s+(\w+)\.(\w+)\s+`)
	content = varRegex.ReplaceAllString(content, "var $2 ")

	// Fix "const pkg.ConstName" -> "const ConstName"
	constRegex := regexp.MustCompile(`const\s+(\w+)\.(\w+)\s*=`)
	content = constRegex.ReplaceAllString(content, "const $2 =")

	return content
}

// deduplicateDeclarations removes duplicate type, struct, const, var, and method declarations
func deduplicateDeclarations(content string) string {
	lines := strings.Split(content, "\n")

	// Track seen declarations
	seenTypes := make(map[string]bool)  // type Name ...
	seenConsts := make(map[string]bool) // const Name ...
	seenVars := make(map[string]bool)   // var Name ...
	seenFuncs := make(map[string]bool)  // func Name(...) or func (r Receiver) Name(...)

	var result []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check for type declaration: "type Name struct/interface/..."
		if typeMatch := regexp.MustCompile(`^type\s+(\w+)\s+`).FindStringSubmatch(trimmed); len(typeMatch) > 1 {
			typeName := typeMatch[1]
			if seenTypes[typeName] {
				// Skip this duplicate type declaration including its body
				i = skipDeclarationBlock(lines, i)
				continue
			}
			seenTypes[typeName] = true
		}

		// Check for const block or single const
		if constMatch := regexp.MustCompile(`^const\s+(\w+)\s*=`).FindStringSubmatch(trimmed); len(constMatch) > 1 {
			constName := constMatch[1]
			if seenConsts[constName] {
				i++
				continue
			}
			seenConsts[constName] = true
		}

		// Check for const block: "const ("
		if strings.HasPrefix(trimmed, "const (") {
			// Extract const names from block and check for duplicates
			blockStart := i
			blockEnd := findBlockEnd(lines, i)

			// Check if any const in this block is a duplicate
			hasDuplicate := false
			for j := blockStart + 1; j < blockEnd; j++ {
				constLine := strings.TrimSpace(lines[j])
				if constNameMatch := regexp.MustCompile(`^(\w+)\s*[=\s]`).FindStringSubmatch(constLine); len(constNameMatch) > 1 {
					if seenConsts[constNameMatch[1]] {
						hasDuplicate = true
					}
				}
			}

			if hasDuplicate {
				// Skip the entire duplicate const block
				i = blockEnd + 1
				continue
			}

			// Mark all consts in block as seen
			for j := blockStart + 1; j < blockEnd; j++ {
				constLine := strings.TrimSpace(lines[j])
				if constNameMatch := regexp.MustCompile(`^(\w+)\s*[=\s]`).FindStringSubmatch(constLine); len(constNameMatch) > 1 {
					seenConsts[constNameMatch[1]] = true
				}
			}
		}

		// Check for method declaration: "func (r *Receiver) Name(...)"
		if methodMatch := regexp.MustCompile(`^func\s+\((\w+)\s+\*?(\w+)\)\s+(\w+)\s*\(`).FindStringSubmatch(trimmed); len(methodMatch) > 3 {
			receiverType := methodMatch[2]
			methodName := methodMatch[3]
			key := receiverType + "." + methodName
			if seenFuncs[key] {
				// Skip this duplicate method
				i = skipDeclarationBlock(lines, i)
				continue
			}
			seenFuncs[key] = true
		} else if funcMatch := regexp.MustCompile(`^func\s+(\w+)\s*\(`).FindStringSubmatch(trimmed); len(funcMatch) > 1 {
			// Check for standalone function: "func Name(...)"
			funcName := funcMatch[1]
			if seenFuncs[funcName] {
				// Skip this duplicate function
				i = skipDeclarationBlock(lines, i)
				continue
			}
			seenFuncs[funcName] = true
		}

		// Check for var declaration
		if varMatch := regexp.MustCompile(`^var\s+(\w+)\s+`).FindStringSubmatch(trimmed); len(varMatch) > 1 {
			varName := varMatch[1]
			if seenVars[varName] {
				i++
				continue
			}
			seenVars[varName] = true
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// skipDeclarationBlock skips a declaration block (type, func, etc.) and returns the next line index
func skipDeclarationBlock(lines []string, startIdx int) int {
	if startIdx >= len(lines) {
		return startIdx + 1
	}

	line := lines[startIdx]

	// If the line contains '{', find the matching '}'
	if strings.Contains(line, "{") {
		braceCount := strings.Count(line, "{") - strings.Count(line, "}")
		if braceCount == 0 {
			return startIdx + 1
		}

		for i := startIdx + 1; i < len(lines); i++ {
			braceCount += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
			if braceCount <= 0 {
				return i + 1
			}
		}
	}

	return startIdx + 1
}

// findBlockEnd finds the end of a block (const, var, type) that starts with "("
func findBlockEnd(lines []string, startIdx int) int {
	parenCount := 0
	for i := startIdx; i < len(lines); i++ {
		parenCount += strings.Count(lines[i], "(") - strings.Count(lines[i], ")")
		if parenCount <= 0 && i > startIdx {
			return i
		}
	}
	return len(lines) - 1
}

// consolidateImports collects all imports from the file and moves them to the top
func consolidateImports(content, moduleName string) string {
	lines := strings.Split(content, "\n")

	// Find package declaration line and determine current package
	pkgLineIdx := -1
	currentPkg := ""
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			pkgLineIdx = i
			// Extract package name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPkg = parts[1]
			}
			break
		}
	}
	if pkgLineIdx == -1 {
		return content
	}

	// Calculate the import path for the current package (to avoid self-imports)
	currentPkgImport := moduleName + "/" + currentPkg

	// Collect all imports and remove them from their current positions
	imports := make(map[string]bool)
	var newLines []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check for import block: import (...)
		if strings.HasPrefix(trimmed, "import (") {
			// Collect imports from block
			i++
			for i < len(lines) {
				importLine := strings.TrimSpace(lines[i])
				if importLine == ")" {
					i++
					break
				}
				// Extract import path
				if importMatch := regexp.MustCompile(`"([^"]+)"`).FindStringSubmatch(importLine); len(importMatch) > 1 {
					importPath := fixImportPath(strings.TrimSpace(importMatch[1]), moduleName)
					// Skip self-imports (package importing itself)
					// Also skip imports that end with the current package name
					if importPath != "" && importPath != currentPkgImport && !strings.HasSuffix(importPath, "/"+currentPkg) {
						imports[importPath] = true
					}
				}
				i++
			}
			continue
		}

		// Check for single import: import "..."
		if strings.HasPrefix(trimmed, "import \"") || strings.HasPrefix(trimmed, "import \"") {
			if importMatch := regexp.MustCompile(`import\s+"([^"]+)"`).FindStringSubmatch(trimmed); len(importMatch) > 1 {
				importPath := fixImportPath(strings.TrimSpace(importMatch[1]), moduleName)
				// Skip self-imports (package importing itself)
				if importPath != "" && importPath != currentPkgImport && !strings.HasSuffix(importPath, "/"+currentPkg) {
					imports[importPath] = true
				}
			}
			i++
			continue
		}

		newLines = append(newLines, line)
		i++
	}

	// Build consolidated import block
	if len(imports) == 0 {
		return strings.Join(newLines, "\n")
	}

	// Separate standard library imports from third-party imports
	var stdImports, thirdPartyImports []string
	for importPath := range imports {
		if !strings.Contains(importPath, ".") {
			stdImports = append(stdImports, importPath)
		} else {
			thirdPartyImports = append(thirdPartyImports, importPath)
		}
	}

	// Sort imports
	sort.Strings(stdImports)
	sort.Strings(thirdPartyImports)

	// Build import block
	var importBlock strings.Builder
	importBlock.WriteString("\nimport (\n")
	for _, imp := range stdImports {
		importBlock.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	if len(stdImports) > 0 && len(thirdPartyImports) > 0 {
		importBlock.WriteString("\n")
	}
	for _, imp := range thirdPartyImports {
		importBlock.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	importBlock.WriteString(")\n")

	// Insert import block after package declaration
	var result []string
	for i, line := range newLines {
		result = append(result, line)
		if i == pkgLineIdx {
			result = append(result, importBlock.String())
		}
	}

	return strings.Join(result, "\n")
}

// fixImportPath fixes wrong import paths
func fixImportPath(importPath, moduleName string) string {
	// Skip empty imports
	if importPath == "" {
		return ""
	}

	// Fix common wrong standard library imports
	stdlibFixes := map[string]string{
		"big": "math/big",
	}
	if fixed, ok := stdlibFixes[importPath]; ok {
		return fixed
	}

	// Fix wrong module paths like "github.com/example/java2go/model" -> "github.com/example/4-full-maven-repo/model"
	wrongModules := []string{
		"github.com/example/java2go/",
		"github.com/example/java-to-go/",
		"github.com/yourusername/",
		"your-module/",
		"com.example/",
	}

	for _, wrong := range wrongModules {
		if strings.HasPrefix(importPath, wrong) {
			pkg := strings.TrimPrefix(importPath, wrong)
			return moduleName + "/" + pkg
		}
	}

	// Fix bare package imports like "model" -> "github.com/example/xxx/model"
	barePackages := []string{"model", "service", "repository", "controller", "config", "utils", "web"}
	for _, bare := range barePackages {
		if importPath == bare {
			return moduleName + "/" + bare
		}
	}

	return importPath
}

// autoFixStandardImports adds missing standard library imports based on usage
func autoFixStandardImports(content string) string {
	// Map of identifiers to their required imports
	stdLibImports := map[string]string{
		"time.Time":           "time",
		"time.Duration":       "time",
		"time.Now":            "time",
		"time.Sleep":          "time",
		"time.Second":         "time",
		"time.Millisecond":    "time",
		"atomic.Int64":        "sync/atomic",
		"atomic.Int32":        "sync/atomic",
		"atomic.Value":        "sync/atomic",
		"atomic.AddInt64":     "sync/atomic",
		"sync.Mutex":          "sync",
		"sync.RWMutex":        "sync",
		"sync.WaitGroup":      "sync",
		"sync.Map":            "sync",
		"context.Context":     "context",
		"context.Background":  "context",
		"context.WithCancel":  "context",
		"fmt.Println":         "fmt",
		"fmt.Printf":          "fmt",
		"fmt.Sprintf":         "fmt",
		"fmt.Errorf":          "fmt",
		"errors.New":          "errors",
		"errors.Is":           "errors",
		"strings.Contains":    "strings",
		"strings.TrimSpace":   "strings",
		"strings.Split":       "strings",
		"strings.Join":        "strings",
		"strings.ToLower":     "strings",
		"strings.ToUpper":     "strings",
		"strconv.Itoa":        "strconv",
		"strconv.Atoi":        "strconv",
		"regexp.MustCompile":  "regexp",
		"regexp.MatchString":  "regexp",
		"json.Marshal":        "encoding/json",
		"json.Unmarshal":      "encoding/json",
		"http.Get":            "net/http",
		"http.Post":           "net/http",
		"http.Handler":        "net/http",
		"http.Request":        "net/http",
		"http.ResponseWriter": "net/http",
		"log.Println":         "log",
		"log.Printf":          "log",
		"log.Fatal":           "log",
		"os.Open":             "os",
		"os.Create":           "os",
		"os.ReadFile":         "os",
		"os.WriteFile":        "os",
		"io.Reader":           "io",
		"io.Writer":           "io",
		"io.Copy":             "io",
		"math.Max":            "math",
		"math.Min":            "math",
		"math.Abs":            "math",
		"rand.Intn":           "math/rand",
		"rand.Int":            "math/rand",
		"uuid.New":            "github.com/google/uuid",
	}

	// Detect which imports are needed
	neededImports := make(map[string]bool)
	for usage, pkg := range stdLibImports {
		// Check for both "time.Time" style and just "time." prefix
		parts := strings.SplitN(usage, ".", 2)
		if len(parts) == 2 {
			prefix := parts[0] + "."
			if strings.Contains(content, prefix) {
				neededImports[pkg] = true
			}
		}
	}

	// Also check for common type usages without package prefix
	typeToImport := map[string]string{
		"Time ":     "time", // "Time " as type
		"Duration ": "time",
		"Context ":  "context",
	}
	for typeUsage, pkg := range typeToImport {
		if strings.Contains(content, typeUsage) && !strings.Contains(content, `"`+pkg+`"`) {
			neededImports[pkg] = true
		}
	}

	if len(neededImports) == 0 {
		return content
	}

	// Check which imports are already present
	existingImports := make(map[string]bool)
	importBlockRegex := regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
	if matches := importBlockRegex.FindStringSubmatch(content); len(matches) > 1 {
		importBlock := matches[1]
		for pkg := range neededImports {
			if strings.Contains(importBlock, `"`+pkg+`"`) {
				existingImports[pkg] = true
			}
		}
	}

	// Build list of imports to add
	var importsToAdd []string
	for pkg := range neededImports {
		if !existingImports[pkg] {
			importsToAdd = append(importsToAdd, `	"`+pkg+`"`)
		}
	}

	if len(importsToAdd) == 0 {
		return content
	}

	// Add missing imports to import block
	if importBlockRegex.MatchString(content) {
		// Add to existing import block
		content = importBlockRegex.ReplaceAllStringFunc(content, func(block string) string {
			// Insert before the closing parenthesis
			insertPos := strings.LastIndex(block, ")")
			if insertPos > 0 {
				newImports := strings.Join(importsToAdd, "\n")
				return block[:insertPos] + newImports + "\n" + block[insertPos:]
			}
			return block
		})
	} else {
		// Create new import block after package declaration
		pkgRegex := regexp.MustCompile(`(package\s+\w+\s*\n)`)
		newImportBlock := "\nimport (\n" + strings.Join(importsToAdd, "\n") + "\n)\n"
		content = pkgRegex.ReplaceAllString(content, "${1}"+newImportBlock)
	}

	return content
}

// fixCrossPackageTypes fixes type references that need package prefixes
func fixCrossPackageTypes(content, currentPkg string, pkgTypes map[string]string) string {
	// Common type to package mappings
	typeToPackage := map[string]string{
		"User":                    "model",
		"UserStatus":              "model",
		"BaseEntity":              "model",
		"UserRepository":          "repository",
		"InMemoryUserRepository":  "repository",
		"UserService":             "service",
		"EmailService":            "service",
		"UserRegistrationService": "service",
		"UserController":          "controller",
		"AppConfig":               "config",
	}

	// Merge with provided mappings
	for t, pkg := range pkgTypes {
		typeToPackage[t] = pkg
	}

	for typeName, pkg := range typeToPackage {
		// Skip if we're in the same package
		if pkg == currentPkg {
			continue
		}

		// Pattern to match unqualified type usage (not already prefixed)
		// Match: "User" but not "model.User", "*User" but not "*model.User"
		patterns := []string{
			`(\s)` + typeName + `(\s|,|\)|\{|\})`,            // Type in context
			`(\*)` + typeName + `(\s|,|\)|\{|\})`,            // Pointer type
			`(\[\])` + typeName + `(\s|,|\)|\{|\})`,          // Slice type
			`(map\[[^\]]+\])` + typeName + `(\s|,|\)|\{|\})`, // Map value type
			`(:\s*)` + typeName + `(\s|,|\)|\{|\})`,          // Field type
			`(\s)` + typeName + `$`,                          // Type at end of line
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			replacement := "${1}" + pkg + "." + typeName + "${2}"
			content = re.ReplaceAllString(content, replacement)
		}
	}

	return content
}

// readGoModuleName reads the module name from go.mod file
func readGoModuleName(outputDir string) string {
	goModPath := filepath.Join(outputDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	// Parse module line: "module github.com/example/xxx"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

func runGoModTidy(outputDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %v", err)
	}
	return nil
}

// runGoimports runs goimports on all Go files to fix imports and format code
func runGoimports(outputDir string) error {
	// First check if goimports is available
	_, err := exec.LookPath("goimports")
	if err != nil {
		// Try to use gofmt as fallback
		log.Info("goimports not found, using gofmt for formatting")
		return runGofmt(outputDir)
	}

	// Run goimports on all Go files
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		cmd := exec.Command("goimports", "-w", path)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Info("goimports warning for %s: %s", path, string(output))
			// Don't fail, just warn - the file might have syntax errors
		}
		return nil
	})
}

// runGofmt runs gofmt on all Go files as a fallback
func runGofmt(outputDir string) error {
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		cmd := exec.Command("gofmt", "-w", path)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Info("gofmt warning for %s: %s", path, string(output))
			// Don't fail, just warn - the file might have syntax errors
		}
		return nil
	})
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

// runCargoCheck runs cargo check in the output directory for Rust projects
func runCargoCheck(outputDir string) error {
	// Check if Cargo.toml exists
	cargoToml := filepath.Join(outputDir, "Cargo.toml")
	if _, err := os.Stat(cargoToml); os.IsNotExist(err) {
		return fmt.Errorf("Cargo.toml not found")
	}

	cmd := exec.Command("cargo", "check")
	cmd.Dir = outputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo check failed: %v", err)
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