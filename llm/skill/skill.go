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

package skill

// Skill 表示一个可加载的技能（完整加载，Phase 2）
type Skill struct {
	// 必需字段
	Name        string // 1-64字符，小写+数字+连字符
	Description string // 1-1024字符

	// 可选字段
	License       string            // 许可证
	Compatibility string            // 环境要求，1-500字符
	AllowedTools  []string          // 空格分隔字符串解析后的列表
	Metadata      map[string]string // 任意键值对

	// 内容（渐进式加载）
	Instructions string // Markdown 正文（Phase 2 加载）

	// 资源目录（Phase 3 按需加载）
	ScriptsDir    string // scripts/ 目录路径
	ReferencesDir string // references/ 目录路径
	AssetsDir     string // assets/ 目录路径

	// 元信息
	Source   SkillSource // 来源类型
	BasePath string      // skill 根目录
	Path     string      // SKILL.md 文件路径（向后兼容）
}

// SkillMetadata 用于 Phase 1 只加载元数据
type SkillMetadata struct {
	Name        string // skill 名称
	Description string // skill 描述
	Path        string // SKILL.md 文件路径
	BasePath    string // skill 根目录
	Source      SkillSource
}

// SkillSource 表示 skill 的来源类型
type SkillSource int

const (
	SourceEmbedded SkillSource = iota // 内置
	SourceLocal                       // 本地目录
	SourceRemote                      // GitHub 远程
)

// String 返回 skill source 的字符串表示
func (s SkillSource) String() string {
	switch s {
	case SourceEmbedded:
		return "embedded"
	case SourceLocal:
		return "local"
	case SourceRemote:
		return "remote"
	default:
		return "unknown"
	}
}
