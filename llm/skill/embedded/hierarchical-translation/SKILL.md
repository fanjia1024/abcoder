---
name: hierarchical-translation
description: For large source ASTs, extract hierarchy first, confirm target language spec, then translate level by level (repo → modules → packages → Types/Functions/Vars). Use get_ast_hierarchy and get_target_language_spec before translating.
allowed-tools: list_repos get_repo_structure get_ast_hierarchy get_target_language_spec get_package_structure get_file_structure get_ast_node translate_type translate_node translate_file translate_package translate_repo sequential_thinking
compatibility: Requires abcoder with AST read and translation tools
---

# Hierarchical Translation Skill

## Purpose

When the source language AST is large, use **hierarchical translation**: first extract the AST hierarchy (leveled directory), confirm target language characteristics, then translate step by step to the target language AST. This avoids oversized context and omissions from translating the entire repo at once.

## Available Tools

- `list_repos`: List available repositories
- `get_repo_structure`: Repository structure (modules and packages)
- `get_ast_hierarchy`: **Leveled AST directory** (Level 0=repo, 1=modules, 2=packages with type/function/var counts, 3=files, 4=nodes). Use `max_depth` (0–4) to limit depth. Prefer this for large repos to plan translation.
- `get_target_language_spec`: **Target language spec** (type mapping, naming, error handling). Call before translating to confirm target language (e.g. go, java, rust).
- `get_package_structure`: Package structure (files and node list)
- `get_file_structure`: File structure (node id, type, signature)
- `get_ast_node`: Full AST node (code, type, location, relations)
- `translate_type`: Translate a type to target language
- `translate_node`: Translate one AST node to target code
- `translate_file`: Translate one file
- `translate_package`: Translate one package to target UniAST (use per package for large repos)
- `translate_repo`: Translate entire repo (for small repos only; for large repos use translate_package per package)
- `sequential_thinking`: Step-by-step thinking and context storage

## Workflow

1. **Confirm repository**: Use `list_repos` to get the repository name.
2. **Get AST hierarchy**: Use `get_ast_hierarchy(repo_name, max_depth)` to get the leveled directory. Use `max_depth` 2 or 4 for large repos (2 = repo + modules + packages with counts; 4 = full tree including files and node list).
3. **Confirm target language**: Use `get_target_language_spec(target_language)` (e.g. `go`, `java`, `rust`) to get type mapping, naming, and error-handling habits before translating.
4. **Translate by level**: Use `translate_package` for each package, or use `get_ast_node` and `translate_node` in order: **Types first**, then **Functions**, then **Vars** per package. Prefer per-package translation; avoid loading the entire AST at once. Use `sequential_thinking` to plan the order.
5. **Aggregate**: Combine translated packages into the target UniAST or output.

## Best Practices

- Always call `get_ast_hierarchy` and `get_target_language_spec` before translating when the repo has many modules or packages.
- Translate in dependency order: types → functions → vars within each package.
- Use `translate_package` for large repos instead of `translate_repo` to keep context manageable.
- Use `sequential_thinking` to record progress and plan the next package or level.

## Relation to Code Translation

- **code-translation**: General translation rules and tools (type mapping, naming, error handling). Use for any code translation task.
- **hierarchical-translation**: Recommended flow for **large** source ASTs: hierarchy extraction → target spec confirmation → level-by-level translation. Use when the repo has many packages or files.
