# Why GlyphLang?

A programming language designed for AI code generation. Built for LLMs, optimized for backend APIs.

## The Problem

Modern programming languages were designed for humans. Every syntax decision — from `public static void` to `def init() -> None:` — prioritizes human readability over machine generation.

**This is a problem for AI developers.**

When an LLM writes backend code, it spends 60-80% of its token budget on **boilerplate** — type declarations, import statements, framework scaffolding — rather than business logic.

| Language | Lines for CRUD API | Token Cost (approx) |
|----------|-------------------|---------------------|
| Python/FastAPI | 300-400 | 4,000-6,000 |
| Go/Gin | 250-350 | 3,500-5,000 |
| TypeScript/Express | 350-450 | 5,000-7,000 |
| **GlyphLang** | **60-120** | **800-1,500** |

## The Solution: Token-Efficient Design

GlyphLang removes the ceremony:

```glyph
@ GET /users/:id -> User | Error {
  % db: Database
  $ user = try db.get("users", id)
  > user
}
```

Equivalent FastAPI:
```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional

app = FastAPI()

class User(BaseModel):
    id: int
    name: str
    email: str

class Error(BaseModel):
    code: str
    message: str

@app.get("/users/{id}")
def get_user(id: int) -> User | Error:
    user = db.query("SELECT * FROM users WHERE id = ?", id)
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user
```

**5x fewer lines. 5x fewer tokens. Same functionality.**

## Why This Matters for AI Development

1. **Fewer context window issues** — Less code means the LLM can see the entire program at once
2. **Faster iteration** — Edit → validate → run cycle completes in sub-second
3. **Higher correctness** — Shorter programs have fewer opportunities for bugs
4. **Cleaner diffs** — Semantic changes are easier to review

## But It's Not Just About Compactness

GlyphLang provides full production capability:

- ✅ Static binary compilation (single file deployment)
- ✅ HTTP server with middleware, auth, WebSockets
- ✅ PostgreSQL, MySQL, SQLite, MongoDB support
- ✅ Redis caching, message queues, cron jobs
- ✅ OpenTelemetry tracing, Prometheus metrics
- ✅ LSP for VS Code and Neovim
- ✅ Polyglot codegen: `.glyph` → Python/FastAPI, TypeScript/Express

## The Missing Piece: Error Handling

Every real program needs error handling. GlyphLang implements `try` propagation:

```glyph
$ result = try db.query("SELECT * FROM users WHERE id = ?", id)
# If db.query returns an Error, it automatically propagates as HTTP 500
# The route return type `-> User | Error` handles it automatically
```

## Getting Started

```bash
# Install
go install github.com/glyphlang/glyph/cmd/glyph@latest

# Create a new API
glyph new my-api
cd my-api
glyph run

# Or run a single file
glyph run hello.glyph
```

## Comparison

| Feature | GlyphLang | Python | Go | Rust |
|---------|-----------|--------|-----|------|
| Token efficiency | ✅ 5x | ❌ | ❌ | ❌ |
| Single binary | ✅ | ❌ | ✅ | ✅ |
| Built-in HTTP | ✅ | FastAPI | Gin | Axum |
| Type system | ✅ Strict | ✅ Dynamic | ✅ | ✅ |
| Error handling | try | try/except | Result | Result |
| AI-optimized | ✅ | ❌ | ❌ | ❌ |

## The Bottom Line

If you're building backend APIs with AI assistance, GlyphLang is purpose-built for that workflow. It's not a replacement for Python or Go — it's a different tool designed for a different workflow.

**Try it:** `go install github.com/glyphlang/glyph/cmd/glyph@latest`