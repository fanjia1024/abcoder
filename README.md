# ABCoder: AI-Based Coder(AKA: A Brand-new Coder)

![ABCoder](images/ABCoder.png)

# Overview
ABCoder, an AI-oriented Code-processing **Framework**, is designed to enhance and extend the coding context for Large-Language-Model (LLM), finally boosting the development of AI-assisted-programming applications. 


## Features

- Universal Abstract-Syntax-Tree (UniAST), a language-independent, AI-friendly specification of code information, providing a boundless, flexible and structural coding context for both AI and humans.
  
- General Parser, parses arbitrary-language codes to UniAST.

- General Writer transforms UniAST back to code.

- **Code-Retrieval-Augmented-Generation (Code-RAG)**, provides a set of MCP tools to help the LLM understand code repositories precisely and locally. And it can support both in-workspace and out-of-workspace third-party libraries simultaneously -- I guess you are thinking about [DeepWiki](https://deepwiki.org) and [context7](https://github.com/upstash/context7), but ABCoder is more reliable and confidential -- no need to wait for their services to be done, and no worry about your codes will be uploaded! 

Based on these features, developers can easily implement or enhance their AI-assisted programming applications, such as reviewing, optimizing, translating, etc.


## Universal Abstract-Syntax-Tree Specification

see [UniAST Specification](docs/uniast-zh.md)


# Quick Start
## Use ABCoder as a MCP server

1. Install ABCoder:

    ```bash
    go install github.com/cloudwego/abcoder@latest
    ```

2. Use ABCoder to parse a repository to UniAST (JSON)

    ```bash
    abcoder parse {language} {repo-path} -o xxx.json
    ```

    ABCoder will try to install any dependency automatically.
    In case of failure (or if you want to customize installation), refer to the [docs](./docs/lsp-installation-en.md).

    For example, to parse a Go repository:

    ```bash
    git clone https://github.com/cloudwego/localsession.git localsession
    abcoder parse go localsession -o /abcoder-asts/localsession.json
    ```


3. Integrate ABCoder's MCP tools into your AI agent.

    ```json
    {
        "mcpServers": {
            "abcoder": {
                "command": "abcoder",
                "args": [
                    "mcp",
                    "{the-AST-directory}"
                ]
            }
        }
    }
    ```


4. Enjoy it!
   
   Try to click and watch the video below:

   <div align="center">
   
   [<img src="images/abcoder-hertz-trae.png" alt="MCP" width="500"/>](https://www.bilibili.com/video/BV14ggJzCEnK)
   
   </div>

    
## Tips:
    
- You can add more repo ASTs into the AST directory without restarting abcoder MCP server.
    
- Try to use [the recommended prompt](llm/prompt/analyzer.md) and combine planning/memory tools like [sequential-thinking](https://github.com/modelcontextprotocol/servers/tree/main/src/sequentialthinking) in your AI agent.


## Translate Code Between Languages

ABCoder can translate code from one language to another using LLM. Supported translation pairs include **Java to Go** and **TypeScript to Go** (and other combinations among Java, Go, Python, Rust, C++).

```bash
export API_TYPE='{openai|ollama|ark|claude|dashscope|deepseek}' 
export API_KEY='{your-api-key}' 
export MODEL_NAME='{model-endpoint}' 
abcoder translate <src-lang> <dst-lang> <project-path> -o <output-dir>
```

**TypeScript to Go** (requires `abcoder-ts-parser` installed, e.g. `npm install -g abcoder-ts-parser`):

```bash
abcoder translate ts go /path/to/your-ts-project -o ./output-go-project
```

For example, translating a local TypeScript project to Go:

```bash
export API_TYPE='claude'
export API_KEY='your-anthropic-api-key'
export MODEL_NAME='claude-sonnet-4-20250514'
abcoder translate ts go /Users/jiafan/Desktop/poc/opencode -o ./opencode-go
```

**Java to Go** — for example, using Claude:

```bash
export API_TYPE='claude'
export API_KEY='your-anthropic-api-key'
export MODEL_NAME='claude-sonnet-4-20250514'
abcoder translate java go ./my-java-project -o ./my-go-project
```

Using Qwen (DashScope):

```bash
export API_TYPE='dashscope'
export API_KEY='your-dashscope-api-key'
export MODEL_NAME='qwen-max'
abcoder translate java go ./my-java-project -o ./my-go-project
```

Using DeepSeek:

```bash
export API_TYPE='deepseek'
export API_KEY='your-deepseek-api-key'
export MODEL_NAME='deepseek-chat'
abcoder translate java go ./my-java-project -o ./my-go-project
```

**Supported LLM Providers:**
- OpenAI (GPT-4o, GPT-4, etc.)
- Claude (Claude 3.5/4, etc.)
- ARK/Doubao (火山引擎/豆包)
- DashScope/Qwen (阿里云/通义千问)
- DeepSeek
- Ollama (local models)
- Any OpenAI-compatible API (via `BASE_URL`)

## Use ABCoder as an Agent (WIP)

You can also use ABCoder as a command-line Agent like:

```bash
export API_TYPE='{openai|ollama|ark|claude|dashscope|deepseek}' 
export API_KEY='{your-api-key}' 
export MODEL_NAME='{model-endpoint}' 
abcoder agent {the-AST-directory}
```
For example:

```bash
$ API_TYPE='ark' API_KEY='xxx' MODEL_NAME='zzz' abcoder agent ./testdata/asts

Hello! I'm ABCoder, your coding assistant. What can I do for you today?

$ What does the repo 'localsession' do?

The `localsession` repository appears to be a Go module (`github.com/cloudwego/localsession`) that provides functionality related to managing local sessions. Here's a breakdown of its structure and purpose:
...
If you'd like to explore specific functionalities or code details, let me know, and I can dive deeper into the relevant files or nodes. For example:
- What does `session.go` or `manager.go` implement?
- How is the backup functionality used?

$ exit
```

- NOTICE: This feature is Work-In-Progress. It only supports code analysis at present.

# Supported Languages

ABCoder currently supports the following languages:

| Language | Parser | Writer      |
| -------- | ------ | ----------- |
| Go       | ✅      | ✅           |
| Rust     | ✅      | Coming Soon |
| C        | ✅      | Coming Soon |
| Python   | ✅      | Coming Soon |
| JS/TS    | ✅      | Coming Soon |
| Java     | ✅      | Coming Soon |


# Getting Involved

We encourage developers to contribute and make this tool more powerful. If you are interested in contributing to ABCoder
project, kindly check out our guide:
- [Parser Extension](docs/parser-zh.md)

> Note: This is a dynamic README and is subject to changes as the project evolves.


# Contact Us
- How to become a member: [COMMUNITY MEMBERSHIP](https://github.com/cloudwego/community/blob/main/COMMUNITY_MEMBERSHIP.md)
- Issues: [Issues](https://github.com/cloudwego/abcoder/issues)
- Lark: Scan the QR code below with [Register Feishu](https://www.feishu.cn/en/) to join our CloudWeGo/abcoder user group.

&ensp;&ensp;&ensp; <img src="images/lark_group_zh.png" alt="LarkGroup" width="200"/>

# Contributors
Thank you for your contribution to ABCoder!

[![Contributors](https://contrib.rocks/image?repo=cloudwego/abcoder)](https://github.com/cloudwego/abcoder/graphs/contributors)

# License
This project is licensed under the [Apache-2.0 License](LICENSE-APACHE).
