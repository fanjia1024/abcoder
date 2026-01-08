---
name: doc-generation
description: 为代码生成文档，包括 API 文档、README、架构说明等。
allowed-tools: list_repos get_repo_structure get_package_structure get_ast_node sequential_thinking
compatibility: Requires abcoder MCP server with AST tools
---

# Documentation Generation Skill

## Purpose

This skill enables automatic generation of documentation for codebases, including API documentation, README files, architecture descriptions, and code comments.

## Available Tools

- `list_repos`: List all available repositories
- `get_repo_structure`: Get repository structure including modules and packages
- `get_package_structure`: Get package structure including files and node names
- `get_ast_node`: Get complete AST node information including code, type, location, and relationships
- `sequential_thinking`: Tool for step-by-step thinking and context storage

## Documentation Types

### 1. API Documentation
- Function/method signatures
- Parameter descriptions
- Return value descriptions
- Usage examples
- Error conditions

### 2. README Files
- Project overview
- Installation instructions
- Usage examples
- Configuration options
- Contributing guidelines

### 3. Architecture Documentation
- System design
- Component relationships
- Data flow diagrams
- Module descriptions

### 4. Code Comments
- Function comments
- Type comments
- Package comments
- Inline documentation

## Workflow

1. **Understand Structure**: Use `get_repo_structure` to understand the codebase organization
2. **Explore Packages**: Use `get_package_structure` to identify key components
3. **Analyze Code**: Use `get_ast_node` to understand function signatures and implementations
4. **Generate Documentation**: Create appropriate documentation based on the code structure
5. **Use Sequential Thinking**: Use `sequential_thinking` to organize documentation generation process

## Documentation Standards

### Function Documentation
- Clear description of purpose
- Parameter descriptions with types
- Return value descriptions
- Usage examples
- Error handling information

### Package Documentation
- Package purpose and overview
- Key types and functions
- Usage examples
- Related packages

### README Structure
- Project title and description
- Features
- Installation
- Quick start
- API overview
- Examples
- Contributing
- License

## Best Practices

- Write clear, concise documentation
- Include practical examples
- Keep documentation up-to-date with code
- Use consistent formatting
- Provide both high-level overview and detailed API docs
- Include code examples in documentation
- Document edge cases and error conditions
