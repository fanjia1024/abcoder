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

package tool

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

func TestASTTool_ToEinoTool(t *testing.T) {
	type fields struct {
		repo string
	}
	tests := []struct {
		name   string
		fields fields
		want   tool.BaseTool
	}{
		{
			name: "test",
			fields: fields{
				repo: "../../tmp/localsession.json",
			},
			want: utils.NewTool(
				&schema.ToolInfo{
					Name: "query_ast_node",
					Desc: "query the info of a AST node",
					ParamsOneOf: schema.NewParamsOneOfByParams(
						map[string]*schema.ParameterInfo{
							"id": {
								Type:     schema.Object,
								Desc:     "the id of the ast node",
								Required: true,
								SubParams: map[string]*schema.ParameterInfo{
									"build_package": {
										Type: schema.String,
										Desc: "the building build of the ast node belongs to, e.g. github.com/bytedance/sonic",
									},
									"version": {
										Type:     schema.String,
										Desc:     "the version of the building build, e.g. v1.0.0",
										Required: false,
									},
									"namespace": {
										Type: schema.String,
										Desc: "the namespace of the ast node belongs to, e.g. encoder/vm",
									},
									"name": {
										Type: schema.String,
										Desc: "the name of the ast node, e.g. Node.String",
									},
								},
							},
						},
					),
				},
				(&ASTReadTools{}).GetASTNode,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(ASTReadToolsOptions{
				// PatchOptions: patch.Options{
				// 	DefaultLanuage: uniast.Golang,
				// 	OutDir:         "./tmp",
				// 	RepoDir:        "../../tmp/localsession",
				// },
				RepoASTsDir: TestRepoASTsDir,
			})
			for _, tool := range tr.tools {
				t.Logf("tool: %#v", tool)
			}
		})
	}
}

func TestASTTools_GetFileStructure(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0 context.Context
		req GetFileStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetFileStructResp)
	}{
		{
			name: "existing_file",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				in0: context.Background(),
				req: GetFileStructReq{
					RepoName: "localsession",
					FilePath: "backup/metainfo_test.go",
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetFileStructResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if got.FilePath != "backup/metainfo_test.go" {
					t.Errorf("got.FilePath = %q", got.FilePath)
				}
				if len(got.Nodes) == 0 {
					t.Error("got.Nodes should be non-empty")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetFileStructure(tt.args.in0, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetFileStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// var hertzRepo *uniast.Repository

// func TestMain(m *testing.M) {
// 	repox, err := uniast.LoadRepo("../../tmp/hertz.json")
// 	if err != nil {
// 		panic(err)
// 	}
// 	hertzRepo = repox
// 	os.Exit(m.Run())
// }

func TestASTTools_GetRepoStructure(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0 context.Context
		req GetRepoStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetRepoStructResp)
	}{
		{
			name: "existing_repo_metainfo",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName: "metainfo",
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetRepoStructResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if len(got.Modules) == 0 {
					t.Error("got.Modules should be non-empty")
				}
			},
		},
		{
			name: "existing_repo_localsession",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				in0: context.Background(),
				req: GetRepoStructReq{
					RepoName: "localsession",
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetRepoStructResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if len(got.Modules) == 0 {
					t.Error("got.Modules should be non-empty")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetRepoStructure(tt.args.in0, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetRepoStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestASTTools_GetPackageStructure(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		ctx context.Context
		req GetPackageStructReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetPackageStructResp)
	}{
		{
			name: "existing_package_localsession_backup",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetPackageStructReq{
					RepoName: "localsession",
					ModPath:  "github.com/cloudwego/localsession",
					PkgPath:  "github.com/cloudwego/localsession/backup",
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetPackageStructResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if len(got.Files) == 0 {
					t.Error("got.Files should be non-empty for known package")
				}
			},
		},
		{
			name: "existing_package_metainfo_kv",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetPackageStructReq{
					RepoName: "metainfo",
					ModPath:  "metainfo",
					PkgPath:  "metainfo::kv",
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetPackageStructResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if len(got.Files) == 0 {
					t.Error("got.Files should be non-empty for known package")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetPackageStructure(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetPackageStructure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestASTTools_GetASTNode(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		in0    context.Context
		params GetASTNodeReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetASTNodeResp)
	}{
		{
			name: "existing_nodes",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				in0: context.Background(),
				params: GetASTNodeReq{
					RepoName: "localsession",
					NodeIDs: []NodeID{
						{
							ModPath: "github.com/cloudwego/localsession",
							PkgPath: "github.com/cloudwego/localsession/backup",
							Name:    "RecoverCtxOnDemands",
						},
						{
							ModPath: "github.com/cloudwego/localsession",
							PkgPath: "github.com/cloudwego/localsession",
							Name:    "CurSession",
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetASTNodeResp) {
				if got == nil {
					t.Fatal("got must be non-nil")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if len(got.Nodes) == 0 {
					t.Error("got.Nodes should be non-empty for known nodes")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetASTNode(tt.args.in0, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ASTTools.GetASTNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestASTTools_GetASTHierarchy(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		ctx context.Context
		req GetASTHierarchyReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetASTHierarchyResp)
	}{
		{
			name: "existing_repo_localsession",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetASTHierarchyReq{
					RepoName: "localsession",
					MaxDepth: 2,
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if got.Hierarchy == nil {
					t.Fatal("got.Hierarchy must be non-nil")
				}
				if got.Hierarchy.Level != 0 || got.Hierarchy.Kind != "repository" {
					t.Errorf("Level=0 Kind=repository, got Level=%d Kind=%s", got.Hierarchy.Level, got.Hierarchy.Kind)
				}
				if got.Hierarchy.Counts == nil || got.Hierarchy.Counts.Modules < 1 {
					t.Errorf("Counts.Modules >= 1, got %v", got.Hierarchy.Counts)
				}
			},
		},
		{
			name: "existing_repo_metainfo",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetASTHierarchyReq{
					RepoName: "metainfo",
					MaxDepth: 1,
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
				if got.Hierarchy == nil {
					t.Fatal("got.Hierarchy must be non-nil")
				}
				if got.Hierarchy.Level != 0 || got.Hierarchy.Kind != "repository" {
					t.Errorf("Level=0 Kind=repository, got Level=%d Kind=%s", got.Hierarchy.Level, got.Hierarchy.Kind)
				}
			},
		},
		{
			name: "repo_not_found",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetASTHierarchyReq{
					RepoName: "nonexistent_repo",
					MaxDepth: 0,
				},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetASTHierarchyResp) {
				if got.Error == "" {
					t.Error("got.Error must be non-empty when repo not found")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetASTHierarchy(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetASTHierarchy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestASTTools_GetTargetLanguageSpec(t *testing.T) {
	type fields struct {
		opts ASTReadToolsOptions
	}
	type args struct {
		ctx context.Context
		req GetTargetLanguageSpecReq
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(t *testing.T, got *GetTargetLanguageSpecResp)
	}{
		{
			name: "target_go",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetTargetLanguageSpecReq{TargetLanguage: "go"},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("got.Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
			},
		},
		{
			name: "target_java",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetTargetLanguageSpecReq{TargetLanguage: "java"},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Spec == "" {
					t.Error("got.Spec must be non-empty")
				}
				if got.Error != "" {
					t.Errorf("got.Error = %q, want empty", got.Error)
				}
			},
		},
		{
			name: "target_unknown",
			fields: fields{
				opts: ASTReadToolsOptions{
					RepoASTsDir: TestRepoASTsDir,
				},
			},
			args: args{
				ctx: context.Background(),
				req: GetTargetLanguageSpecReq{TargetLanguage: "xyz"},
			},
			wantErr: false,
			check: func(t *testing.T, got *GetTargetLanguageSpecResp) {
				if got.Error == "" {
					t.Error("got.Error must be non-empty for unknown language")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewASTReadTools(tt.fields.opts)
			got, err := tr.GetTargetLanguageSpec(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTargetLanguageSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// func TestASTTools_WriteASTNode(t *testing.T) {
// 	type fields struct {
// 		opts    ASTToolsOptions
// 		repo    *uniast.Repository
// 		patcher *patch.Patcher
// 		tools   map[string]tool.InvokableTool
// 	}
// 	type args struct {
// 		in0 context.Context
// 		req WriteASTNodeReq
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    *WriteASTNodeResp
// 		wantErr bool
// 	}{
// 		{
// 			name: "add",
// 			fields: fields{
// 				opts: ASTToolsOptions{
// 					PatchOptions: patch.Options{
// 						DefaultLanuage: uniast.Golang,
// 						OutDir:         "../../tmp/hertz",
// 						RepoDir:        "../../tmp/hertz",
// 					},
// 				},
// 				repo: hertzRepo,
// 			},
// 			args: args{
// 				in0: context.Background(),
// 				req: WriteASTNodeReq{
// 					ID: uniast.Identity{
// 						ModPath: "github.com/cloudwego/hertz",
// 						PkgPath: "github.com/cloudwego/hertz/pkg/app",
// 						Name:    "RequestContext2",
// 					},
// 					Codes: `type RequestContext2 struct {
// 						RequestContext
// 					}`,
// 					File: "pkg/app/context.go",
// 					Type: "TYPE",
// 				},
// 			},
// 		},
// 		{
// 			name: "modify",
// 			fields: fields{
// 				opts: ASTToolsOptions{
// 					PatchOptions: patch.Options{
// 						DefaultLanuage: uniast.Golang,
// 						OutDir:         "../../tmp/hertz",
// 						RepoDir:        "../../tmp/hertz",
// 					},
// 				},
// 				repo: hertzRepo,
// 			},
// 			args: args{
// 				in0: context.Background(),
// 				req: WriteASTNodeReq{
// 					ID: uniast.Identity{
// 						ModPath: "github.com/cloudwego/hertz",
// 						PkgPath: "github.com/cloudwego/hertz",
// 						Name:    "Version",
// 					},
// 					Codes: `Version = "v2"`,
// 				},
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tr := NewASTTools(tt.fields.repo, ASTToolsOptions{
// 				PatchOptions: tt.fields.opts.PatchOptions,
// 			})
// 			got, err := tr.WriteASTNode(tt.args.in0, tt.args.req)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("ASTTools.WriteASTNode() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			_ = got
// 			// if !reflect.DeepEqual(got, tt.want) {
// 			// 	t.Errorf("ASTTools.WriteASTNode() = %v, want %v", got, tt.want)
// 			// }
// 		})
// 	}
// }
