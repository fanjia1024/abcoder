# Java → Go Translation Trace

This document traces a **single translation pair (Java → Go)** through the full pipeline: from source code to UniAST, LLM node transformation, target UniAST, and final emitted code. It serves as a technical reference and debugging aid.

## Pipeline Overview

```
Java source (.java)
       ↓  [Parser: javaparser / LSP]
   Java AST
       ↓  [Collect / Export]
   Source UniAST (Repository JSON)
       ↓  [Transformer: per-node]
   LLM request (one Type / Function / Var)
       ↓  [LLM: IR transformer only]
   LLM response (TargetContent)
       ↓  [Framework: assemble node]
   Target UniAST (Repository)
       ↓  [UniAST Validator]
   Validated target repo
       ↓  [Writer: deterministic]
   Go source (.go)
```

**Important:** The LLM never generates final source code. It only returns **UniAST node content** (e.g. the body of a type or function). The Writer turns that content into files.

---

## Example: One Type Node (BaseEntity)

We trace the Java class `BaseEntity` from `com.example.common.model` through to the Go struct in `model/baseentity.go`.

### 1. Input: Java source

File: `com/example/common/model/BaseEntity.java` (simplified snippet)

```java
package com.example.common.model;

import java.time.LocalDateTime;

public abstract class BaseEntity {

    private Long id;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
    private String createdBy;
    private String updatedBy;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    // ... getters/setters ...
}
```

### 2. Source UniAST (after Parse + Collect)

The parser and collector produce a `uniast.Repository`. The node for `BaseEntity` appears inside a module and package. Relevant **Type** node in the source repo (conceptual JSON):

```json
{
  "ModPath": "com.example.common@1.0",
  "PkgPath": "com/example/common/model",
  "Name": "BaseEntity",
  "File": "BaseEntity.java",
  "Line": 5,
  "Content": "public abstract class BaseEntity {\n    private Long id;\n    ...\n}",
  "TypeKind": "struct",
  "Exported": true
}
```

Identity: `ModPath` + `PkgPath` + `Name` uniquely identify the node. `Content` holds the full class body (signature + fields + methods) that the Writer would have emitted for Java; for translation we send this to the LLM.

### 3. LLM request (single node)

The Transformer calls `NodeTranslator.TranslateType` for this node. The **LLMTranslateRequest** contains:

| Field            | Example value |
|------------------|----------------|
| SourceLanguage   | `java`        |
| TargetLanguage   | `go`          |
| NodeType         | `TYPE`        |
| SourceContent    | The Java class body (same as `Content` above) |
| Identity         | `{ ModPath, PkgPath, Name }` for BaseEntity |
| TypeHints        | e.g. `Long → int64`, `LocalDateTime → time.Time`, `String → string` |
| Dependencies     | Already-translated types in the same package (if any) |
| Prompt           | Full prompt: type mapping, dependencies, source code, requirements, output format |

The prompt instructs the LLM to return **only** the translated code (Go struct + methods), no markdown or explanation.

### 4. LLM response

The **LLMTranslateResponse** for this type node:

| Field             | Example value |
|-------------------|----------------|
| TargetContent     | Go struct and method bodies (see below) |
| TargetSignature   | (empty for types; used for functions) |
| AdditionalImports | e.g. `"time"` |
| Error             | `""` on success |

Example `TargetContent` (what the LLM returns):

```go
type BaseEntity struct {
	ID        *int64    `json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

func (b *BaseEntity) GetID() *int64 { return b.ID }
func (b *BaseEntity) SetID(id *int64) { b.ID = id }
// ...
```

### 5. Target UniAST (assembled node)

The framework builds the target **uniast.Type** and inserts it into the target repository:

- **Identity**: `ModPath` = target module (e.g. `example.com/myapp`), `PkgPath` = `model`, `Name` = `BaseEntity`
- **FileLine**: `File` = `baseentity.go` (or derived from source), `Line` = 1
- **Content**: exactly `resp.TargetContent` from the LLM
- **TypeKind** / **Exported**: copied or adapted from source

No re-parsing of `TargetContent` is done here; the Writer will emit `Content` as-is (with import handling).

### 6. UniAST validation

Before any write, `uniast.ValidateRepository(targetRepo)` runs:

- Required fields present (Identity, non-empty Content, valid FileLine)
- Invariants (e.g. no duplicate node names per package)

If validation fails, the pipeline **rejects** the output and does not call the Writer.

### 7. Final Go code (Writer output)

The Go Writer visits each package and each Type/Function/Var, and appends `node.Content` to the appropriate file (with import collection). For this node, the emitted file is e.g. `model/baseentity.go`:

```go
package model

import (
	"time"
)

type BaseEntity struct {
	ID        *int64    `json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	// ...
}

func (b *BaseEntity) GetID() *int64 { return b.ID }
// ...
```

The Writer is **deterministic**: the same target UniAST always produces the same Go files.

---

## Summary Table (single node)

| Stage           | Input / Output |
|----------------|----------------|
| Input AST      | Java `BaseEntity.java` (source text) |
| Source UniAST  | One Type node: Identity `com.example.common.model#BaseEntity`, Content = Java class body |
| LLM request    | NodeType=TYPE, SourceContent=that body, TypeHints, Prompt |
| LLM response   | TargetContent = Go struct + methods (plain text) |
| Target UniAST  | One Type node: Identity `module#model#BaseEntity`, Content = TargetContent |
| Validator      | Checks repo; rejects if any node has empty Content or invalid Identity |
| Writer         | Writes `Content` to `model/baseentity.go` with package and imports |

---

## References

- [UniAST specification](uniast-zh.md)
- [Translation / architecture](../../README.md#translate-code-between-languages) (README)
- Translate implementation: `lang/translate/` (Transformer, NodeTranslator, options)
- UniAST validation: `lang/uniast/validate.go`
