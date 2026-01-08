---
name: code-analysis
description: 分析代码结构、理解代码逻辑、追踪依赖关系。用于理解代码库、定位问题、回答代码相关问题。
allowed-tools: list_repos get_repo_structure get_package_structure get_ast_node get_file_structure sequential_thinking
compatibility: Requires abcoder MCP server with AST tools
---

# Code Analysis Skill

## Purpose

This skill enables comprehensive code analysis, helping to understand code structure, logic flow, and dependencies. Use this skill when you need to:

- Understand how a codebase is organized
- Trace code execution paths
- Find relationships between code components
- Answer questions about code functionality
- Locate specific code patterns or implementations

## Available Tools

- `list_repos`: List all available repositories
- `get_repo_structure`: Get repository structure including modules and packages
- `get_package_structure`: Get package structure including files and node names
- `get_ast_node`: Get complete AST node information including code, type, location, and relationships
- `get_file_structure`: Get file structure including node names, types, and signatures
- `sequential_thinking`: Tool for step-by-step thinking and context storage

## Workflow

1. **Start with Repository Overview**: Use `list_repos` to see available repositories, then `get_repo_structure` to understand the overall organization.

2. **Navigate by Package**: Use `get_package_structure` to explore packages and identify relevant files.

3. **Deep Dive into Code**: Use `get_ast_node` to get detailed information about specific code elements, including:
   - Code content
   - Type information
   - Location (file path and line numbers)
   - Dependencies and references
   - Inheritance and implementation relationships

4. **Trace Relationships**: Follow dependencies, references, and relationships to understand code flow and connections.

5. **Use Sequential Thinking**: Break down complex analysis tasks using `sequential_thinking` to maintain context and avoid information loss.

## Best Practices

- Always start with a high-level view before diving into details
- Use `sequential_thinking` to record findings and build understanding incrementally
- Check test files (`*_test.*`) for examples and usage patterns
- Follow dependency chains to understand code relationships
- Provide accurate file locations and line numbers in responses
