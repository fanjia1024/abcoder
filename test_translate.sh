#!/bin/bash

# ABCoder 多语言转换测试脚本
# 支持: Java, Go, Python, Rust, TypeScript 之间的互相转换

set -e

# ============================================
# 颜色定义
# ============================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# LLM 配置 - 在这里修改你的 LLM 服务配置
# ============================================

# 支持的 API_TYPE:
#   - openai     (OpenAI)
#   - claude     (Claude)
#   - ark        (豆包/火山引擎)
#   - dashscope  (通义千问/DashScope)
#   - deepseek   (DeepSeek)
#   - ollama     (本地模型)

# --- DashScope (通义千问) ---
# export API_TYPE="dashscope"
# export API_KEY="sk-810e9c55ef5948f58837c90eed07b8bc"
# export MODEL_NAME="qwen3-max"
# export BASE_URL=""  # 可选，使用默认值

# --- OpenAI ---
# export API_TYPE="openai"
# export API_KEY="sk-your-openai-api-key"
# export MODEL_NAME="gpt-4o"
# export BASE_URL="https://api.openai.com/v1"

# --- Claude ---
# export API_TYPE="claude"
# export API_KEY="sk-your-claude-api-key"
# export MODEL_NAME="claude-opus-4-20250514"

# --- DeepSeek ---
# export API_TYPE="deepseek"
# export API_KEY="sk-your-deepseek-api-key"
# export MODEL_NAME="deepseek-chat"
# export BASE_URL="https://api.deepseek.com/v1"

# --- 豆包/火山引擎 (ARK) ---
# export API_TYPE="ark"
# export API_KEY="your-ark-api-key"
# export MODEL_NAME="your-endpoint-id"

# --- Ollama (本地模型) ---
export API_TYPE="ollama"
export API_KEY="demo"
export MODEL_NAME="gpt-oss:120b"
export BASE_URL="http://10.135.4.11:11434"

# ============================================
# 测试数据目录配置
# ============================================
# 若项目目录下存在 uniast.json，translate 会直接使用并跳过解析，便于复用（如先 parse 生成后再多次 translate）。
JAVA_TEST_PROJECT="testdata/java/4_full_maven_repo"
GO_TEST_PROJECT="testdata/go/0_goland"      # TODO: 添加 Go 测试项目
PYTHON_TEST_PROJECT="testdata/python/7_reexport"      # TODO: 添加 Python 测试项目
RUST_TEST_PROJECT="testdata/rust/1_simpleobj"          # TODO: 添加 Rust 测试项目
# TypeScript 项目路径（需已安装 abcoder-ts-parser: npm install -g abcoder-ts-parser）
TS_TEST_PROJECT="/Users/jiafan/Desktop/poc/opencode"

OUTPUT_BASE_DIR="/Users/jiafan/Desktop/test/output"
LOCAL_BIN="./abcoder_local"

# ============================================
# 辅助函数
# ============================================

print_header() {
    echo ""
    echo -e "${BLUE}==========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}==========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# 验证 LLM 配置
validate_llm_config() {
    if [ -z "$API_TYPE" ] || [ -z "$MODEL_NAME" ]; then
        print_error "请在脚本中配置 LLM 服务"
        echo "打开 test_translate.sh 并修改 LLM 配置部分"
        exit 1
    fi

    # Ollama 不需要 API_KEY
    if [ "$API_TYPE" != "ollama" ] && [ -z "$API_KEY" ]; then
        print_error "请设置 API_KEY (除非使用 Ollama)"
        exit 1
    fi

    if [ "$API_KEY" = "sk-your-dashscope-api-key" ] || \
       [ "$API_KEY" = "sk-your-openai-api-key" ] || \
       [ "$API_KEY" = "sk-your-claude-api-key" ] || \
       [ "$API_KEY" = "sk-your-deepseek-api-key" ] || \
       [ "$API_KEY" = "your-ark-api-key" ]; then
        print_error "请在脚本中设置真实的 API_KEY"
        exit 1
    fi

    echo "LLM 配置:"
    echo "  API_TYPE:   $API_TYPE"
    echo "  MODEL_NAME: $MODEL_NAME"
    if [ -n "$API_KEY" ]; then
        echo "  API_KEY:    ${API_KEY:0:10}..."
    fi
    if [ -n "$BASE_URL" ]; then
        echo "  BASE_URL:   $BASE_URL"
    fi
    echo ""
}

# 构建本地二进制
build_binary() {
    print_info "构建本地 abcoder..."
    if ! go build -o "$LOCAL_BIN" .; then
        print_error "构建失败"
        exit 1
    fi
    print_success "构建成功"
}

# 获取文件扩展名
get_file_extension() {
    case "$1" in
        go|golang) echo "go" ;;
        java) echo "java" ;;
        python|py) echo "py" ;;
        rust|rs) echo "rs" ;;
        cpp|cxx|c++) echo "cpp" ;;
        ts|typescript|js) echo "ts" ;;
        *) echo "$1" ;;
    esac
}

# 运行单个转换测试
run_translation_test() {
    local src_lang="$1"
    local dst_lang="$2"
    local test_project="$3"
    local output_dir="$4"

    print_header "测试: $src_lang → $dst_lang"

    echo "源项目: $test_project"
    echo "输出目录: $output_dir"
    echo ""

    # 清理旧输出
    if [ -d "$output_dir" ]; then
        echo "清理之前的输出目录..."
        rm -rf "$output_dir"
    fi

    # 运行转换
    echo "开始转换..."
    echo ""

    if "$LOCAL_BIN" translate "$src_lang" "$dst_lang" "$test_project" -o "$output_dir" -verbose; then
        print_success "转换命令执行成功"
    else
        print_error "转换命令执行失败"
        return 1
    fi

    echo ""

    # 显示 UniAST 文件位置
    show_ast_files "$src_lang" "$dst_lang"

    # 检查输出
    check_output "$dst_lang" "$output_dir"
}

# 显示 UniAST JSON 文件位置
show_ast_files() {
    local src_lang="$1"
    local dst_lang="$2"
    
    # 查找 AST 临时目录
    local ast_dir=$(find /var/folders -name "abcoder-translate-asts" -type d 2>/dev/null | head -1)
    if [ -z "$ast_dir" ]; then
        ast_dir="/tmp/abcoder-translate-asts"
    fi
    
    if [ -d "$ast_dir" ]; then
        echo "UniAST JSON 文件:"
        # TypeScript 源在 Go 端生成的文件名为 typescript-repo.json
        local src_ast="$ast_dir/${src_lang}-repo.json"
        [ "$src_lang" = "ts" ] && src_ast="$ast_dir/typescript-repo.json"
        local dst_ast="$ast_dir/${dst_lang}-repo.json"

        if [ -f "$src_ast" ]; then
            local src_size=$(du -h "$src_ast" | cut -f1)
            print_success "源语言 ($src_lang): $src_ast ($src_size)"
        fi
        
        if [ -f "$dst_ast" ]; then
            local dst_size=$(du -h "$dst_ast" | cut -f1)
            print_success "目标语言 ($dst_lang): $dst_ast ($dst_size)"
        else
            print_warning "目标语言 AST 未生成: $dst_ast"
        fi
        echo ""
    fi
}

# 检查输出结果
check_output() {
    local lang="$1"
    local output_dir="$2"
    local ext=$(get_file_extension "$lang")

    echo "检查输出结果..."

    # 检查输出目录是否存在
    if [ ! -d "$output_dir" ]; then
        print_error "输出目录不存在: $output_dir"
        return 1
    fi

    # 检查生成的文件数量
    local file_count=$(find "$output_dir" -name "*.$ext" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$file_count" -eq 0 ]; then
        print_warning "没有生成任何 .$ext 文件"
        echo "输出目录内容:"
        ls -la "$output_dir"
        return 1
    else
        print_success "生成 $file_count 个 .$ext 文件"
    fi

    # 检查项目配置文件和入口点
    case "$lang" in
        go|golang)
            echo ""
            echo "📦 项目配置检查:"
            if [ -f "$output_dir/go.mod" ]; then
                print_success "生成 go.mod"
                echo ""
                echo "go.mod 内容:"
                cat "$output_dir/go.mod"
                echo ""
            else
                print_warning "没有生成 go.mod"
            fi
            
            # 检查入口点
            echo ""
            echo "🚀 入口点检查:"
            local main_go=$(find "$output_dir" -name "main.go" 2>/dev/null | head -1)
            if [ -n "$main_go" ] && [ -f "$main_go" ]; then
                print_success "生成入口点: $main_go"
                # 检查是否包含 Gin 框架
                if grep -q "gin" "$main_go" 2>/dev/null; then
                    print_success "包含 Gin 框架集成"
                fi
            else
                print_warning "没有生成 main.go 入口点"
            fi
            
            # 检查路由文件
            local router_go=$(find "$output_dir" -name "routes.go" -o -name "router.go" 2>/dev/null | head -1)
            if [ -n "$router_go" ] && [ -f "$router_go" ]; then
                print_success "生成路由配置: $router_go"
            fi
            ;;
        python|py)
            echo ""
            echo "📦 项目配置检查:"
            if [ -f "$output_dir/pyproject.toml" ]; then
                print_success "生成 pyproject.toml"
                echo ""
                echo "pyproject.toml 内容:"
                head -20 "$output_dir/pyproject.toml"
                echo ""
            elif [ -f "$output_dir/requirements.txt" ]; then
                print_success "生成 requirements.txt"
            elif [ -f "$output_dir/setup.py" ]; then
                print_success "生成 setup.py"
            else
                print_warning "没有生成 Python 项目配置文件"
            fi
            
            # 检查入口点
            echo ""
            echo "🚀 入口点检查:"
            local main_py=$(find "$output_dir" -name "main.py" -o -name "__main__.py" 2>/dev/null | head -1)
            if [ -n "$main_py" ] && [ -f "$main_py" ]; then
                print_success "生成入口点: $main_py"
                # 检查是否包含 FastAPI
                if grep -q "fastapi\|FastAPI" "$main_py" 2>/dev/null; then
                    print_success "包含 FastAPI 框架集成"
                elif grep -q "flask\|Flask" "$main_py" 2>/dev/null; then
                    print_success "包含 Flask 框架集成"
                fi
            else
                print_warning "没有生成 main.py 入口点"
            fi
            ;;
        rust|rs)
            echo ""
            echo "📦 项目配置检查:"
            if [ -f "$output_dir/Cargo.toml" ]; then
                print_success "生成 Cargo.toml"
                echo ""
                echo "Cargo.toml 内容:"
                cat "$output_dir/Cargo.toml"
                echo ""
            else
                print_warning "没有生成 Cargo.toml"
            fi
            
            # 检查入口点
            echo ""
            echo "🚀 入口点检查:"
            local main_rs=$(find "$output_dir" -name "main.rs" 2>/dev/null | head -1)
            if [ -n "$main_rs" ] && [ -f "$main_rs" ]; then
                print_success "生成入口点: $main_rs"
                # 检查是否包含 Actix
                if grep -q "actix" "$main_rs" 2>/dev/null; then
                    print_success "包含 Actix-web 框架集成"
                elif grep -q "axum" "$main_rs" 2>/dev/null; then
                    print_success "包含 Axum 框架集成"
                fi
            else
                print_warning "没有生成 main.rs 入口点"
            fi
            
            # 检查 lib.rs
            local lib_rs=$(find "$output_dir" -name "lib.rs" 2>/dev/null | head -1)
            if [ -n "$lib_rs" ] && [ -f "$lib_rs" ]; then
                print_success "生成库入口: $lib_rs"
            fi
            ;;
        java)
            echo ""
            echo "📦 项目配置检查:"
            if [ -f "$output_dir/pom.xml" ]; then
                print_success "生成 pom.xml"
                echo ""
                echo "pom.xml 部分内容:"
                head -30 "$output_dir/pom.xml"
                echo ""
            elif [ -f "$output_dir/build.gradle" ]; then
                print_success "生成 build.gradle"
            else
                print_warning "没有生成 Java 项目配置文件"
            fi
            
            # 检查 Application 入口
            echo ""
            echo "🚀 入口点检查:"
            local app_java=$(find "$output_dir" -name "Application.java" -o -name "*Application.java" 2>/dev/null | head -1)
            if [ -n "$app_java" ] && [ -f "$app_java" ]; then
                print_success "生成入口点: $app_java"
            else
                print_warning "没有生成 Application.java 入口点"
            fi
            ;;
        cpp|cxx|c++)
            echo ""
            echo "📦 项目配置检查:"
            if [ -f "$output_dir/CMakeLists.txt" ]; then
                print_success "生成 CMakeLists.txt"
                echo ""
                echo "CMakeLists.txt 内容:"
                cat "$output_dir/CMakeLists.txt"
                echo ""
            elif [ -f "$output_dir/Makefile" ]; then
                print_success "生成 Makefile"
            else
                print_warning "没有生成 C++ 项目配置文件 (CMakeLists.txt 或 Makefile)"
            fi
            
            # 检查入口点
            echo ""
            echo "🚀 入口点检查:"
            local main_cpp=$(find "$output_dir" -name "main.cpp" -o -name "main.c" 2>/dev/null | head -1)
            if [ -n "$main_cpp" ] && [ -f "$main_cpp" ]; then
                print_success "生成入口点: $main_cpp"
            else
                print_warning "没有生成 main.cpp 入口点"
            fi
            
            # 检查 include 目录
            if [ -d "$output_dir/include" ]; then
                print_success "生成 include 目录"
            fi
            ;;
    esac

    # 显示文件结构
    echo ""
    echo "📁 文件结构:"
    find "$output_dir" -type f \( -name "*.$ext" -o -name "*.h" -o -name "*.hpp" -o -name "go.mod" -o -name "Cargo.toml" -o -name "*.toml" -o -name "pom.xml" -o -name "CMakeLists.txt" -o -name "Makefile" \) | head -20
    
    local total_files=$(find "$output_dir" -type f \( -name "*.$ext" -o -name "*.h" -o -name "*.hpp" -o -name "go.mod" -o -name "Cargo.toml" -o -name "*.toml" -o -name "pom.xml" -o -name "CMakeLists.txt" -o -name "Makefile" \) | wc -l | tr -d ' ')
    if [ "$total_files" -gt 20 ]; then
        echo "... 还有 $((total_files - 20)) 个文件"
    fi

    # 验证项目构建（可选）
    echo ""
    echo "🔨 构建验证:"
    verify_build "$lang" "$output_dir"

    return 0
}

# 验证项目能否构建
verify_build() {
    local lang="$1"
    local output_dir="$2"

    case "$lang" in
        go|golang)
            if [ -f "$output_dir/go.mod" ]; then
                echo "尝试运行 go mod tidy..."
                if (cd "$output_dir" && go mod tidy 2>&1); then
                    print_success "go mod tidy 成功"
                else
                    print_warning "go mod tidy 失败 (可能需要手动修复依赖)"
                fi
                
                echo "尝试编译检查..."
                if (cd "$output_dir" && go build ./... 2>&1); then
                    print_success "go build 成功"
                else
                    print_warning "go build 失败 (可能需要手动修复语法错误)"
                fi
            fi
            ;;
        rust|rs)
            if [ -f "$output_dir/Cargo.toml" ]; then
                echo "尝试运行 cargo check..."
                if (cd "$output_dir" && cargo check 2>&1); then
                    print_success "cargo check 成功"
                else
                    print_warning "cargo check 失败 (可能需要手动修复语法错误)"
                fi
            fi
            ;;
        python|py)
            local main_py=$(find "$output_dir" -name "main.py" -o -name "__main__.py" 2>/dev/null | head -1)
            if [ -n "$main_py" ]; then
                echo "尝试语法检查..."
                if python3 -m py_compile "$main_py" 2>&1; then
                    print_success "Python 语法检查成功"
                else
                    print_warning "Python 语法检查失败"
                fi
            fi
            ;;
        cpp|cxx|c++)
            if [ -f "$output_dir/CMakeLists.txt" ]; then
                echo "尝试运行 cmake..."
                local build_dir="$output_dir/_build_check"
                mkdir -p "$build_dir"
                if (cd "$build_dir" && cmake .. 2>&1 | head -10); then
                    print_success "cmake 配置成功"
                else
                    print_warning "cmake 配置失败"
                fi
                rm -rf "$build_dir"
            fi
            ;;
        java)
            if [ -f "$output_dir/pom.xml" ]; then
                echo "尝试运行 mvn compile..."
                if (cd "$output_dir" && mvn compile -q 2>&1 | head -10); then
                    print_success "mvn compile 成功"
                else
                    print_warning "mvn compile 失败 (可能需要手动修复)"
                fi
            fi
            ;;
        *)
            print_info "不支持该语言的构建验证"
            ;;
    esac
}

# 显示使用帮助
show_usage() {
    echo "ABCoder 多语言转换测试脚本"
    echo ""
    echo "用法:"
    echo "  $0 [选项] [测试类型]"
    echo ""
    echo "测试类型:"
    echo "  all           运行所有可用的转换测试"
    echo "  java2go       Java → Go 转换测试"
    echo "  java2python   Java → Python 转换测试"
    echo "  java2rust     Java → Rust 转换测试"
    echo "  java2cpp      Java → C++ 转换测试"
    echo "  ts2go         TypeScript → Go 转换测试（需 TS_TEST_PROJECT 与 abcoder-ts-parser）"
    echo "  quick         仅运行 Java → Go 快速测试"
    echo ""
    echo "选项:"
    echo "  -h, --help    显示此帮助信息"
    echo "  -v, --verbose 显示详细输出"
    echo ""
    echo "示例:"
    echo "  $0              # 运行默认测试 (Java → Go)"
    echo "  $0 all          # 运行所有测试"
    echo "  $0 ts2go        # TypeScript → Go（项目路径见 TS_TEST_PROJECT）"
    echo "  $0 java2python  # 仅运行 Java → Python 测试"
    echo ""
}

# 运行 Java → Go 测试
test_java2go() {
    run_translation_test "java" "go" "$JAVA_TEST_PROJECT" "$OUTPUT_BASE_DIR/java2go"
}

# 运行 Java → Python 测试
test_java2python() {
    run_translation_test "java" "python" "$JAVA_TEST_PROJECT" "$OUTPUT_BASE_DIR/java2python"
}

# 运行 Java → Rust 测试
test_java2rust() {
    run_translation_test "java" "rust" "$JAVA_TEST_PROJECT" "$OUTPUT_BASE_DIR/java2rust"
}

# 运行 Java → C++ 测试
test_java2cpp() {
    run_translation_test "java" "cxx" "$JAVA_TEST_PROJECT" "$OUTPUT_BASE_DIR/java2cpp"
}

# 运行 TypeScript → Go 测试（需安装 abcoder-ts-parser）
test_ts2go() {
    if [ ! -d "$TS_TEST_PROJECT" ]; then
        print_warning "TypeScript 测试项目不存在: $TS_TEST_PROJECT"
        print_info "请设置 TS_TEST_PROJECT 或创建该目录后重试"
        return 1
    fi

    if ! command -v abcoder-ts-parser &>/dev/null; then
        if ! command -v npm &>/dev/null; then
            print_error "未找到 npm，无法安装 abcoder-ts-parser"
            print_info "请先安装 Node.js/npm，或手动安装后重试: npm install -g abcoder-ts-parser"
            return 1
        fi
        print_info "正在安装 abcoder-ts-parser..."
        if ! npm install -g abcoder-ts-parser; then
            print_error "安装 abcoder-ts-parser 失败"
            print_info "请手动执行: npm install -g abcoder-ts-parser"
            return 1
        fi
        if ! command -v abcoder-ts-parser &>/dev/null; then
            print_error "安装后仍未找到 abcoder-ts-parser，请检查 PATH 或手动执行: npm install -g abcoder-ts-parser"
            return 1
        fi
        print_success "abcoder-ts-parser 已安装"
    fi

    run_translation_test "ts" "go" "$TS_TEST_PROJECT" "$OUTPUT_BASE_DIR/ts2go"
}

# 运行所有测试
test_all() {
    local passed=0
    local failed=0
    local total=0

    print_header "运行所有转换测试"

    # Java → Go
    ((total++))
    if test_java2go; then
        ((passed++))
        print_success "Java → Go 测试通过"
    else
        ((failed++))
        print_error "Java → Go 测试失败"
    fi

    # Java → Python
    ((total++))
    if test_java2python; then
        ((passed++))
        print_success "Java → Python 测试通过"
    else
        ((failed++))
        print_error "Java → Python 测试失败"
    fi

    # Java → Rust
    ((total++))
    if test_java2rust; then
        ((passed++))
        print_success "Java → Rust 测试通过"
    else
        ((failed++))
        print_error "Java → Rust 测试失败"
    fi

    # Java → C++
    ((total++))
    if test_java2cpp; then
        ((passed++))
        print_success "Java → C++ 测试通过"
    else
        ((failed++))
        print_error "Java → C++ 测试失败"
    fi

    # TypeScript → Go（若 TS_TEST_PROJECT 存在且 abcoder-ts-parser 可用）
    ((total++))
    if [ -d "$TS_TEST_PROJECT" ] && command -v abcoder-ts-parser &>/dev/null; then
        if test_ts2go; then
            ((passed++))
            print_success "TypeScript → Go 测试通过"
        else
            ((failed++))
            print_error "TypeScript → Go 测试失败"
        fi
    else
        print_warning "跳过 TypeScript → Go（TS_TEST_PROJECT 不存在或未安装 abcoder-ts-parser）"
    fi

    # 汇总结果
    print_header "测试汇总"
    echo "总计: $total"
    echo -e "${GREEN}通过: $passed${NC}"
    if [ "$failed" -gt 0 ]; then
        echo -e "${RED}失败: $failed${NC}"
    else
        echo "失败: $failed"
    fi
    echo ""

    if [ "$failed" -gt 0 ]; then
        return 1
    fi
    return 0
}

# ============================================
# 主程序
# ============================================

main() {
    print_header "ABCoder 多语言转换测试"

    # 解析命令行参数
    local test_type="quick"
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--verbose)
                export VERBOSE=1
                shift
                ;;
            all|java2go|java2python|java2rust|java2cpp|ts2go|quick)
                test_type="$1"
                shift
                ;;
            *)
                print_error "未知参数: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # 验证 LLM 配置
    validate_llm_config

    # 构建二进制
    build_binary

    # 运行测试
    case "$test_type" in
        all)
            test_all
            ;;
        java2go|quick)
            test_java2go
            ;;
        java2python)
            test_java2python
            ;;
        java2rust)
            test_java2rust
            ;;
        java2cpp)
            test_java2cpp
            ;;
        ts2go)
            test_ts2go
            ;;
        *)
            print_error "未知测试类型: $test_type"
            exit 1
            ;;
    esac

    local result=$?

    print_header "完成"
    echo "输出目录: $OUTPUT_BASE_DIR"
    echo ""
    echo "查看结果:"
    echo "  ls -la $OUTPUT_BASE_DIR"
    echo "  tree $OUTPUT_BASE_DIR  # 如果安装了 tree"
    echo ""

    exit $result
}

# 运行主程序
main "$@"
