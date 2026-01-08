---
name: code-review
description: 审查代码质量、检查最佳实践、发现潜在问题和安全隐患。
allowed-tools: list_repos get_repo_structure get_ast_node get_file_structure sequential_thinking
compatibility: Requires abcoder MCP server with AST tools
---

# Code Review Skill

## Purpose

This skill enables comprehensive code review, checking for best practices, potential issues, and security concerns. Use this skill when you need to:

- Review code quality and maintainability
- Check adherence to coding standards
- Identify potential bugs and issues
- Detect security vulnerabilities
- Suggest improvements and optimizations

## Available Tools

- `list_repos`: List all available repositories
- `get_repo_structure`: Get repository structure including modules and packages
- `get_ast_node`: Get complete AST node information including code, type, location, and relationships
- `get_file_structure`: Get file structure including node names, types, and signatures
- `sequential_thinking`: Tool for step-by-step thinking and context storage

## Review Checklist

### 1. Code Organization and Structure
- Module and package organization
- File structure and naming conventions
- Separation of concerns
- Code duplication

### 2. Error Handling
- Proper error handling patterns
- Error propagation
- Exception handling (if applicable)
- Error messages and logging

### 3. Performance Considerations
- Algorithm efficiency
- Resource usage (memory, CPU)
- Unnecessary computations
- Caching opportunities

### 4. Security Concerns
- Input validation
- Authentication and authorization
- Sensitive data handling
- SQL injection and XSS vulnerabilities
- Secure coding practices

### 5. Code Quality
- Code readability and maintainability
- Naming conventions
- Code comments and documentation
- Complexity and cyclomatic complexity

### 6. Testing
- Test coverage
- Test quality
- Edge cases handling

## Workflow

1. **Get Overview**: Use `get_repo_structure` to understand the codebase organization
2. **Review Files**: Use `get_file_structure` to identify files to review
3. **Deep Analysis**: Use `get_ast_node` to examine specific code elements
4. **Check Relationships**: Follow dependencies and references to understand context
5. **Document Findings**: Use `sequential_thinking` to record review findings
6. **Provide Feedback**: Give detailed, actionable feedback with specific file locations

## Best Practices

- Be constructive and specific in feedback
- Provide code examples for suggested improvements
- Prioritize security and critical issues
- Consider the context and purpose of the code
- Reference specific file locations and line numbers
- Balance between perfectionism and practicality
