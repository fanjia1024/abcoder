#!/bin/bash

# Java to Go 转换测试脚本

set -e

echo "=========================================="
echo "ABCoder Java to Go 转换测试"
echo "=========================================="
echo ""

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

# 选择你的 LLM 提供商 (取消注释对应的配置块)

# --- DashScope (通义千问) ---
export API_TYPE="dashscope"
export API_KEY="sk-810e9c55ef5948f58837c90eed07b8bc"
export MODEL_NAME="qwen3-max"
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
# export API_TYPE="ollama"
# export API_KEY=""
# export MODEL_NAME="llama3"
# export BASE_URL="http://localhost:11434"

# ============================================
# 配置验证
# ============================================

if [ -z "$API_TYPE" ] || [ -z "$API_KEY" ] || [ -z "$MODEL_NAME" ]; then
    echo "错误: 请在脚本中配置 LLM 服务"
    echo "打开 test_translate.sh 并修改 LLM 配置部分"
    exit 1
fi

if [ "$API_KEY" = "sk-your-dashscope-api-key" ] || \
   [ "$API_KEY" = "sk-your-openai-api-key" ] || \
   [ "$API_KEY" = "sk-your-claude-api-key" ] || \
   [ "$API_KEY" = "sk-your-deepseek-api-key" ] || \
   [ "$API_KEY" = "your-ark-api-key" ]; then
    echo "错误: 请在脚本中设置真实的 API_KEY"
    echo "打开 test_translate.sh 并修改 API_KEY 值"
    exit 1
fi

echo "LLM 配置:"
echo "  API_TYPE:   $API_TYPE"
echo "  MODEL_NAME: $MODEL_NAME"
echo "  API_KEY:    ${API_KEY:0:10}..."
if [ -n "$BASE_URL" ]; then
    echo "  BASE_URL:   $BASE_URL"
fi
echo ""

# 设置测试项目路径
TEST_PROJECT="testdata/java/4_full_maven_repo"
OUTPUT_DIR="test-output/go-simple"
LOCAL_BIN="./abcoder_local"

echo "测试项目: $TEST_PROJECT"
echo "输出目录: $OUTPUT_DIR"
echo ""

# 清理旧输出
if [ -d "$OUTPUT_DIR" ]; then
  echo "清理之前的输出目录..."
  rm -rf "$OUTPUT_DIR"
fi

# 构建本地二进制，避免使用旧版本
echo "构建本地 abcoder..."
if ! go build -o "$LOCAL_BIN" ./main.go; then
  echo "构建失败，退出"
  exit 1
fi

# 运行转换
echo "开始转换..."
echo ""

"$LOCAL_BIN" translate java go "$TEST_PROJECT" -o "$OUTPUT_DIR" -verbose

echo ""
echo "=========================================="
echo "转换完成！"
echo "=========================================="
echo ""

# 显示 AST 文件路径
TEMP_AST_DIR="/tmp/abcoder-translate-asts"
if [ -d "$TEMP_AST_DIR" ]; then
    echo "AST 文件位置:"
    if [ -f "$TEMP_AST_DIR/java-repo.json" ]; then
        echo "  Java UniAST: $TEMP_AST_DIR/java-repo.json"
    fi
    if [ -f "$TEMP_AST_DIR/go-repo.json" ]; then
        echo "  Go UniAST: $TEMP_AST_DIR/go-repo.json"
    fi
    echo ""
fi

# 检查输出目录是否存在
if [ ! -d "$OUTPUT_DIR" ]; then
    echo "错误: 输出目录不存在: $OUTPUT_DIR"
    echo "转换可能失败，请检查上面的错误信息"
    exit 1
fi

# 检查是否有 Go 文件生成
GO_FILES=$(find "$OUTPUT_DIR" -name "*.go" 2>/dev/null | wc -l | tr -d ' ')
if [ "$GO_FILES" -eq 0 ]; then
    echo "警告: 没有生成任何 Go 文件"
    echo "输出目录内容:"
    ls -la "$OUTPUT_DIR"
else
    echo "成功生成 $GO_FILES 个 Go 文件"
fi

# 检查是否有 go.mod
if [ -f "$OUTPUT_DIR/go.mod" ]; then
    echo "成功生成 go.mod"
    echo ""
    echo "go.mod 内容:"
    cat "$OUTPUT_DIR/go.mod"
    echo ""
else
    echo "警告: 没有生成 go.mod"
fi

echo ""
echo "输出目录: $OUTPUT_DIR"
echo ""
echo "查看转换结果:"
echo "  ls -la $OUTPUT_DIR"
echo "  find $OUTPUT_DIR -name '*.go' -exec cat {} \\;"
echo ""
