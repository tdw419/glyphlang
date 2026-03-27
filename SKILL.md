---
name: glyphlang
description: AI-first backend programming language with spatial assembly substrate. Use when building APIs, backend services, or spatial computation programs with GlyphLang, compiling .glyph files, or working with the Geometry OS stack.
when_to_use: when writing GlyphLang code, compiling or running .glyph files, building AI-optimized APIs, using .glyph spatial assembly opcodes, working with the GlyphLang CLI, generating code from .glyph source, or integrating with Geometry OS / Ouroboros architecture
version: 1.0.0
languages: glyph
---

# GlyphLang

AI-first backend language that compiles to a single static binary. Designed for minimal token consumption and maximum LLM generation accuracy. ~5x fewer lines and tokens than equivalent Python/FastAPI.

## When to Use

- Building REST APIs, WebSockets, or backend services with AI code generation
- Minimizing LLM token costs for backend development
- Single-file service deployment (no container orchestration needed)
- Spatial computation or self-modifying programs (Ouroboros architecture)
- Polyglot code generation (one `.glyph` source → Python/TypeScript output)

## Quick Start

```bash
glyph init                    # Initialize project
glyph run hello.glyph         # Run server (default :3000)
glyph dev hello.glyph         # Dev server with hot reload
glyph validate src/ --ai      # Validate with JSON error output
glyph context --format compact # Project summary for AI context
```

## Symbol Reference

| Symbol | Name | Usage | Example |
|--------|------|-------|---------|
| `@` | Route/Endpoint | HTTP endpoint | `@ GET /users` |
| `:` | Type | Type definition | `: User { id: int }` |
| `$` | Variable | Variable declaration | `$ name = "Alice"` |
| `!` | Function | Function/CLI command | `! greet(name: str)` |
| `>` | Return | Return statement | `> {message: "ok"}` |
| `+` | Middleware | Apply middleware | `+ auth(jwt)` |
| `%` | Inject | Dependency injection | `% db: Database` |
| `?` | Optional | Optional type | `email: str?` |
| `*` | Cron | Scheduled task | `* "0 * * * *" cleanup` |
| `~` | Event | Event handler | `~ user.created` |
| `&` | Queue | Queue worker | `& emails processEmail` |
| `#` | Comment | Single-line comment | `# comment` |
| `->` | Arrow | Return type annotation | `-> User` |
| `\|` | Union | Union type | `str \| int` |

**Type modifiers:** `T!` (required), `T?` (optional), `[T]` (array)

## Core Patterns

### CRUD API

```glyph
: User {
  id: int!
  name: str!
  email: str?
}

@ GET /users -> [User] {
  % db: Database
  > db.query("SELECT * FROM users")
}

@ POST /users {
  % db: Database
  > db.insert("users", input)
}

@ GET /users/:id -> User | Error {
  % db: Database
  $ user = db.query("SELECT * FROM users WHERE id = ?", id)
  if user == null { > {error: "not found", code: 404} }
  > user
}
```

### Pattern Matching

```glyph
$ result = match code {
  200 => "OK"
  404 => "Not Found"
  n when n >= 500 => "Server Error"
  _ => "Unknown"
}
```

### Async/Await with Combinators

```glyph
@ GET /dashboard {
  $ user = async { > db.getUser(userId) }
  $ orders = async { > db.getOrders(userId) }
  > {user: await user, orders: await orders}
}
```

### WebSocket

```glyph
@ ws /chat/:room {
  on connect { ws.join(room) }
  on message { ws.broadcast_to_room(room, input) }
  on disconnect { ws.leave(room) }
}
```

### Generics

```glyph
! map<T, U>(arr: [T], fn: (T) -> U): [U] {
  $ result = []
  for item in arr { result = append(result, fn(item)) }
  > result
}
```

### Auth + Middleware Chain

```glyph
@ GET /api/profile -> User {
  + auth(jwt)
  + ratelimit(100/min)
  % db: Database
  > db.query("SELECT * FROM users WHERE id = ?", auth.user_id)
}
```

## Type System

| Type | Syntax | Notes |
|------|--------|-------|
| Primitives | `int`, `str`, `bool`, `float` | Built-in |
| Arrays | `[T]` | Generic collections |
| Objects | `{ field: Type }` | Inline or named via `:` |
| Optional | `T?` | Nullable |
| Union | `A \| B` | Either type |
| Generic | `T` | Type parameters on functions/types |

## Project Layout

```
my-project/
├── main.glyph        # Entry point with routes
├── types.glyph       # Type definitions (optional)
├── utils.glyph       # Utility functions (optional)
└── .glyph/           # Build artifacts
```

Import modules: `import "./utils"` → access as `utils.functionName()`

## AI Agent Workflow

```bash
# 1. Get project context (optimized for LLM context windows)
glyph context --format compact

# 2. Make changes, then validate
glyph validate src/ --ai    # Returns JSON errors with fix hints

# 3. Check what changed
glyph context --changed

# 4. Generate polyglot output if needed
glyph codegen main.glyph --lang typescript -o ./out
```

## Spatial Assembly Substrate (Low-Level)

For advanced use: Ouroboros Level 3 architecture with self-modifying programs. See [references/spatial-assembly.md](references/spatial-assembly.md) for the full opcode reference.

**Key opcodes:**

| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `0-9` | `( -- n)` | Push integer |
| `+ - * /` | `( a b -- r)` | Arithmetic |
| `> < =` | `( a b -- bool)` | Comparison (pushes 1/0) |
| `?` | `( c t f -- r)` | Conditional |
| `L` | `( s e -- [r])` | Range generator |
| `M` | `( v o -- )` | Mutator: overwrite code at IP+offset |
| `S` | `( o -- id)` | Mitosis: clone VM into parallel thread |
| `.` | `( v -- )` | Output to visual grid |
| `@` | `( -- )` | Terminate thread |
| `@>` | `( -- )` | Request natural language intervention |

**Register protocol:** Lowercase `a-z` pops stack → store. Uppercase `A-Z` loads register → push stack.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using `return` keyword | Use `>` for returns |
| Declaring types with `type` | Use `:` prefix: `: User { ... }` |
| Writing `function` | Use `!` prefix: `! myFunc()` |
| Multiple files for simple API | Keep in single `.glyph` file |
| Forgetting `!` on required fields | `T!` = required, `T?` = optional |
| Using `async/await` keywords | Use `async { }` blocks and `await` expression |
| Missing dependency injection | Use `% db: Database` to inject |

## Performance Characteristics

- **Compilation:** ~867ns to static binary
- **Execution:** 2.95-37.6 ns/op instruction throughput
- **Token savings:** 23% vs FastAPI, 36% vs Flask, 57% vs Spring/Java
- **Deployment:** Single static binary, built-in HTTP server + DB access

## References

- [Spatial Assembly Opcode Reference](references/spatial-assembly.md) - Full low-level instruction set
- [GitHub Repository](https://github.com/GlyphLang/GlyphLang) - Source, issues, discussions
- [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=GlyphLang.GlyphLang) - LSP + syntax highlighting
