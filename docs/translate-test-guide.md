# Java 到 Go 转换测试指南

## 快速开始

### 1. 设置 LLM 环境变量

选择一个 LLM 提供商并设置相应的环境变量：

#### 使用 OpenAI
```bash
export API_TYPE='openai'
export API_KEY='your-openai-api-key'
export MODEL_NAME='gpt-4o'
```

#### 使用 Claude
```bash
export API_TYPE='claude'
export API_KEY='your-anthropic-api-key'
export MODEL_NAME='claude-sonnet-4-20250514'
```

#### 使用通义千问 (DashScope/Qwen)
```bash
export API_TYPE='dashscope'
export API_KEY='your-dashscope-api-key'
export MODEL_NAME='qwen-max'
```

#### 使用 DeepSeek
```bash
export API_TYPE='deepseek'
export API_KEY='your-deepseek-api-key'
export MODEL_NAME='deepseek-chat'
```

#### 使用豆包 (ARK)
```bash
export API_TYPE='ark'
export API_KEY='your-ark-api-key'
export MODEL_NAME='doubao-pro-32k'
```

#### 使用 Ollama (本地模型)
```bash
export API_TYPE='ollama'
export MODEL_NAME='llama3:70b'
# Ollama 不需要 API_KEY
```

### 2. 运行转换测试

#### 方法 1: 使用测试脚本（推荐）

```bash
cd /Users/jiafan/Desktop/work-code/abcoder
./test_translate.sh
```

#### 方法 2: 直接使用命令

```bash
cd /Users/jiafan/Desktop/work-code/abcoder

# 转换简单项目
abcoder translate java go testdata/java/0_simple -o test-output/go-simple -verbose

# 转换高级特性项目
abcoder translate java go testdata/java/1_advanced -o test-output/go-advanced -verbose

# 转换继承示例项目
abcoder translate java go testdata/java/2_inheritance -o test-output/go-inheritance -verbose
```

### 3. 查看转换结果

```bash
# 查看输出目录
ls -la test-output/go-simple/

# 查看转换后的 Go 代码
cat test-output/go-simple/*.go

# 尝试编译 Go 代码
cd test-output/go-simple
go mod tidy
go build
```

## 测试项目说明

### 0_simple - 简单示例
包含基本的类、方法和内部类示例。

**文件:**
- `HelloWorld.java` - 包含 main 方法、重载方法和内部类
- `AdvancedFeatures.java` - 空类示例

### 1_advanced - 高级特性
包含接口和实现类。

**文件:**
- `Animal.java` - 接口定义
- `Cat.java` - 实现类
- `Dog.java` - 实现类

### 2_inheritance - 继承示例
包含继承和多态示例。

**文件:**
- `Shape.java` - 基类
- `Circle.java` - 子类
- `Rectangle.java` - 子类

### 4_full_maven_repo - 完整 Maven 项目
多模块 Maven 项目，包含：
- common-module - 公共模块
- core-module - 核心模块
- service-module - 服务模块
- web-module - Web 模块

## 故障排除

### 问题 1: API_TYPE 未设置
```
错误: env API_TYPE is required
```
**解决:** 设置 `export API_TYPE='your-provider'`

### 问题 2: API_KEY 未设置
```
错误: env API_KEY is required
```
**解决:** 设置 `export API_KEY='your-api-key'`

### 问题 3: MODEL_NAME 未设置
```
错误: env MODEL_NAME is required
```
**解决:** 设置 `export MODEL_NAME='your-model-name'`

### 问题 4: Java 解析失败
如果 Java 项目解析失败，检查：
- Java 项目结构是否正确
- 是否有 pom.xml 或 build.gradle（Maven/Gradle 项目）
- Java 版本是否兼容

### 问题 5: 转换结果无法编译
转换后的 Go 代码可能需要手动调整：
- 检查 import 语句
- 检查类型映射
- 检查错误处理

## 完整测试示例

```bash
# 1. 设置环境变量（以 Claude 为例）
export API_TYPE='claude'
export API_KEY='sk-ant-...'
export MODEL_NAME='claude-sonnet-4-20250514'

# 2. 运行转换
cd /Users/jiafan/Desktop/work-code/abcoder
abcoder translate java go testdata/java/0_simple -o test-output/go-simple -verbose

# 3. 查看结果
ls -la test-output/go-simple/
cat test-output/go-simple/*.go

# 4. 尝试编译（如果生成了 go.mod）
cd test-output/go-simple
go mod tidy 2>&1 || echo "需要手动创建 go.mod"
go build 2>&1 || echo "编译失败，需要手动修复"
```

## 注意事项

1. **首次运行**: 首次解析 Java 项目时，ABCoder 会自动下载 JDT Language Server，可能需要一些时间。

2. **LLM 成本**: 转换大型项目会消耗 LLM API 调用，请注意成本。

3. **转换质量**: 转换结果可能需要人工审查和调整，特别是：
   - 错误处理模式
   - 并发模型
   - 第三方库依赖

4. **输出目录**: 如果输出目录已存在，转换会覆盖其中的文件。
