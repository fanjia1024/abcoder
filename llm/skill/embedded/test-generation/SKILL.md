---
name: test-generation
description: 为代码生成单元测试、集成测试，包括测试用例设计和测试代码编写。
allowed-tools: list_repos get_ast_node get_package_structure sequential_thinking
compatibility: Requires abcoder MCP server with AST tools
---

# Test Generation Skill

## Purpose

This skill enables automatic generation of test cases for code, including unit tests, integration tests, test case design, and test code writing.

## Available Tools

- `list_repos`: List all available repositories
- `get_ast_node`: Get complete AST node information including code, type, location, and relationships
- `get_package_structure`: Get package structure including files and node names
- `sequential_thinking`: Tool for step-by-step thinking and context storage

## Test Types

### 1. Unit Tests
- Function-level testing
- Method testing
- Type testing
- Edge case coverage

### 2. Integration Tests
- Component interaction testing
- API endpoint testing
- Database integration testing
- External service integration

### 3. Test Cases
- Happy path scenarios
- Error conditions
- Edge cases
- Boundary value testing

## Workflow

1. **Analyze Code**: Use `get_ast_node` to understand the code structure and functionality
2. **Identify Test Targets**: Determine which functions, methods, or components need testing
3. **Design Test Cases**: Plan test scenarios including normal cases, edge cases, and error conditions
4. **Generate Test Code**: Write test code following the project's testing framework conventions
5. **Use Sequential Thinking**: Use `sequential_thinking` to organize test generation process

## Test Generation Guidelines

### Test Structure
- Setup/teardown procedures
- Test case organization
- Assertion patterns
- Mock/stub usage

### Coverage Goals
- Function coverage
- Branch coverage
- Edge case coverage
- Error path coverage

### Test Quality
- Clear test names
- Single responsibility per test
- Independent test cases
- Deterministic tests

## Best Practices

- Follow the project's existing test patterns
- Write clear, descriptive test names
- Test both success and failure paths
- Include edge cases and boundary conditions
- Use appropriate mocking/stubbing
- Ensure tests are independent and repeatable
- Maintain test code quality similar to production code
- Consider performance implications of tests
