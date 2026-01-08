---
name: code-translation
description: 将代码从一种语言翻译到另一种语言，保持语义等价性。支持 Java->Go 等翻译。
allowed-tools: list_repos get_ast_node translate_node translate_type translate_file sequential_thinking
compatibility: Requires abcoder MCP server with AST and translation tools
---

# Code Translation Skill

## Purpose

This skill enables translation of code from one programming language to another while maintaining semantic equivalence. Currently supports Java to Go translation, with extensibility for other language pairs.

## Available Tools

- `list_repos`: List all available repositories
- `get_ast_node`: Get complete AST node information for source code
- `translate_node`: Translate a Java AST node to Go code
- `translate_type`: Translate Java type to Go type
- `translate_file`: Translate an entire Java file to Go code
- `sequential_thinking`: Tool for step-by-step thinking and context storage

## Translation Rules

### Type Mapping

- **Primitives**: `int` → `int`, `long` → `int64`, `String` → `string`, `boolean` → `bool`
- **Collections**: `List<T>` → `[]T`, `Map<K,V>` → `map[K]V`, `Set<T>` → `[]T` or `map[T]bool`
- **Optional**: `Optional<T>` → `(T, bool)` (Go idiom)
- **Objects**: `Object` → `interface{}`, Custom classes → structs or interfaces

### Package Path Conversion

- Java: `com.example.project` → Go: `github.com/example/project`
- Maintain package hierarchy relationships

### Naming Conventions

- **Classes**: `PascalCase` → `PascalCase` (Go types, always exported)
- **Methods**: `camelCase` → `PascalCase` (exported) or `camelCase` (unexported)
- **Fields**: `camelCase` → `PascalCase` (exported) or `camelCase` (unexported)
- **Constants**: `UPPER_SNAKE_CASE` → `UPPER_SNAKE_CASE`

### Error Handling

- Java `throws Exception` → Go `(result, error)` return pattern
- Java `try-catch` → Go `if err != nil { return err }`
- Java `RuntimeException` → Go `panic` (use sparingly)

### Class and Method Conversion

- `class MyClass` → `type MyClass struct`
- `interface MyInterface` → `type MyInterface interface`
- `public void method()` → `func (r *Receiver) Method()`
- `static method()` → `func Method()` (package-level function)

## Workflow

1. **Analyze Source Code**: Use `get_ast_node` to understand the structure of the source code
2. **Translate Types**: Use `translate_type` for type conversions
3. **Translate Nodes**: Use `translate_node` for individual code elements
4. **Translate Files**: Use `translate_file` for complete file translation
5. **Verify**: Ensure translated code follows Go conventions and is compilable

## Best Practices

- Use `sequential_thinking` to break down complex translations
- Check dependencies and imports carefully
- Handle edge cases (nil checks, error handling)
- Preserve code comments when possible
- Ensure type safety and correctness
- Follow Go idioms and best practices
