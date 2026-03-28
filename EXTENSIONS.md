# GlyphLang Orchestrator Extensions

Extensions to the official GlyphLang specification, optimized for AI self-orchestration and the Ouroboros architecture.

## Philosophy

Official GlyphLang targets **backend services** (CRUD APIs, WebSockets). Our extensions target **AI self-orchestration** — the AI reading, writing, and modifying its own execution logic.

**Token efficiency is paramount:** Every extension reduces context window consumption.

---

## Syntax Extensions

### `∞` — Infinite Loop

**Official:** `L` opcode for range generation (`1 10 L`)
**Ours:** `∞ { ... }` for semantic infinite loops

```
# Official (low-level)
0 1000000 L { body }

# Ours (orchestrator)
∞ {
  fetch_next_prompt()
  process()
  sleep(60)
}
```

**Rationale:** `∞` is single-token, semantic, and immediately clear to LLMs. Used in orchestrator main loops.

---

### `ROUTE` — Entry Point Declaration

**Official:** `@` for HTTP endpoints (`@ GET /users`)
**Ours:** `ROUTE name::route -> { }` for orchestrator arms

```
ROUTE core::orchestrator -> {
  match args() {
    ("--run") => { ... }
    ("--status") => { ... }
    _ => { FAIL }
  }
}
```

**Rationale:** Orchestrators aren't HTTP servers — they're CLI dispatchers. `ROUTE` clarifies intent.

---

### `match args()` — CLI Pattern Matching

**Official:** `match x { 200 => "OK" }` for value matching
**Ours:** `match args() { ("--flag", var) => { } }` for CLI dispatch

```
match args() {
  ("--run", project) => { process_project(project) }
  ("--status") => { print_status() }
  ("--evolve", code) => { M[self] ← code }
  _ => { PRINT "Usage: ..." }
}
```

**Rationale:** Direct mapping to CLI argument parsing without argparse boilerplate.

---

## Syscall Extensions

### SQL — Database Operations

**Official:** `% db: Database` with `db.query()`
**Ours:** Native `SQL.query()`, `SQL.insert()`, `SQL.update()` opcodes

```
# Query with binding
SQL.query("SELECT * FROM prompt_queue WHERE status='pending' LIMIT 1") → prompt

# Insert with field pack
SQL.insert("prompt_queue", %{
  id=human_abc123,
  prompt="Fix the bug",
  priority=1,
  source=human
})

# Update with where clause
SQL.update("prompt_queue", %{status="completed"}, prompt.id)
```

**Rationale:** Direct SQL without ORM overhead. LLMs already know SQL — no abstraction layer needed.

---

### FS — Filesystem Operations

**Official:** Not specified
**Ours:** `FS.read()`, `FS.write()` with JSON auto-parsing

```
# Read control file
FS.read(".loop.control") → cmd

# Write with field pack
FS.write(".loop.status", %{
  processed=processed_count,
  queue_pending=10828,
  uptime_seconds=3600
})
```

**Rationale:** Status and control files are JSON. Auto-parsing eliminates serialization code.

---

### MODEL — LLM Invocation

**Official:** Not specified
**Ours:** `MODEL.invoke()` with 3-tier failover

```
MODEL.invoke(prompt, context, model="qwen2.5-coder-7b") → result

# Result structure:
# {
#   success: bool,
#   error: str?,
#   content: str,
#   latency_ms: int,
#   model: str,
#   provider: str
# }
```

**Failover chain:**
1. LM Studio (`localhost:1234`)
2. Ollama (`localhost:11434`)
3. Return error (no external APIs)

**Rationale:** The orchestrator IS an LLM calling itself. Native model invocation is core infrastructure.

---

### OUTPUT — Content Analysis

**Official:** Not specified
**Ours:** `OUTPUT.analyze()` for mutation detection

```
OUTPUT.analyze(result.content) → analysis

# Analysis structure:
# {
#   type: "mutation" | "text",
#   has_code: bool,
#   has_error: bool,
#   code: str  # Extracted from ``` blocks
# }
```

**Rationale:** Detects when LLM output contains code mutations (for M opcode validation).

---

### ROADMAP / RAG — Context Retrieval

**Official:** Not specified
**Ours:** Context synchronization from project files

```
ROADMAP.sync("docs/ROADMAP.md") → roadmap_context
RAG.retrieve(query, codebase_context) → relevant_snippets
```

**Rationale:** AI needs project context. Embedding retrieval directly in the language eliminates external tools.

---

## Conditional Extensions

### `?Q∅` — Queue Empty Check

**Official:** Not specified
**Ours:** `?Q∅(var) → action_empty | action_full`

```
SQL.query("SELECT * FROM queue LIMIT 1") → q
?Q∅(q) → 
  PRINT "Queue empty"
|
  PROCESS q
```

**Rationale:** Queue processing is 80% of orchestrator logic. Specialized syntax reduces tokens.

---

### `?cond →` — Inline Conditional

**Official:** `? ( c t f -- r)` stack-based
**Ours:** `?condition → action | else_action`

```
?autonomous → 
  SQL.query("... WHERE priority <= 2")
|
  SQL.query("... WHERE priority <= 10")
```

**Rationale:** Readable conditionals without stack manipulation for high-level logic.

---

## Ouroboros Extensions

### `M[self]` — Self-Modification

**Official:** `M ( v o -- )` — mutate at IP + offset
**Ours:** `M[self] ← code` — semantic self-modification

```
# Official (low-level)
"99" 2 M  # Overwrite instruction 2 ticks ahead

# Ours (high-level)
M[self] ← "1 10 L { dup . 1 + }"  # Replace loop body
```

**Rationale:** LLMs think in code blocks, not instruction offsets. `M[self]` is semantic.

---

### `S(x, y, [args])` — Mitosis with Args

**Official:** `S ( o -- id)` — spawn at offset
**Ours:** `S(x, y, [args]) → id` — spawn with spatial coordinates + arguments

```
# Spawn 3 parallel agents
S(0.25, 0.75, ["--status"]) → agent_1
S(0.50, 0.25, ["--once"]) → agent_2
S(0.75, 0.75, ["--once"]) → agent_3

# Wait for all
S_WAIT() → results
```

**Rationale:** Spatial coordinates enable visual placement on infinite map. Args enable task dispatch.

---

### VCC — Visual Consistency Contract

**Official:** Green/red flash concept
**Ours:** `VCC → ?GREEN → commit | RED → revert`

```
M[self] ← mutation_code
VCC → 
  ?GREEN → 
    PRINT "Mutation committed"
    COMMIT
  | RED → 
    PRINT "Mutation rejected (compilation failed)"
    REVERT
```

**Rationale:** VCC is the immune system. Explicit GREEN/RED branching makes validation visible.

---

## Opcode Mapping

| Extension | Byte | Official | Status |
|-----------|------|----------|--------|
| `∞` | 0x13 | `L` (range) | Extended |
| `SQL` | 0x1D | Not specified | New |
| `FS` | 0x1E | Not specified | New |
| `MODEL` | 0x1F | Not specified | New |
| `OUTPUT` | 0x20 | Not specified | New |
| `STATUS` | 0x21 | Not specified | New |
| `ROADMAP` | 0x22 | Not specified | New |
| `RAG` | 0x23 | Not specified | New |
| `COND_QUEUE` | 0x30 | Not specified | New |
| `COND_BRANCH` | 0x31 | `?` (stack) | Extended |
| `CALL` | 0x32 | Not specified | New |
| `VCC_CHECK` | 0x40 | Concept only | New |
| `SELF_MODIFY` | 0x41 | `M` | Extended |
| `SLEEP` | 0x50 | Not specified | New |
| `SPAWN` | 0x51 | `S` | Extended |
| `SPAWN_WAIT` | 0x52 | Not specified | New |

---

## Compatibility Notes

### Using Official GlyphLang CLI

Our extensions are **not** recognized by the official `glyph` CLI. For standard backend services, use official syntax:

```bash
# Official syntax for APIs
glyph run api.glyph

# Our extensions for orchestrators
python3 geo_exec.py orchestrator.glyphbin --run
cargo run --bin sovereign_vm orchestrator.glyphbin -- --run
```

### Interop Strategy

1. **Backend APIs** → Official GlyphLang (`@ GET /users`)
2. **AI Orchestrators** → Our extensions (`∞`, `SQL`, `MODEL`)
3. **Visual Shell** → Mix both (official for HTTP, ours for state)

---

## Implementation

- **Compiler:** `geo_cc.py` — Python-based, emits `.glyphbin` + `.glyph.meta.json`
- **Python VM:** `geo_exec.py` — 550 lines, 21 tests
- **Rust VM:** `sovereign_vm` crate — 1200+ lines, 18 tests

---

## Future Extensions

| Extension | Purpose | Status |
|-----------|---------|--------|
| `CHAN` | Inter-agent messaging | Planned |
| `MSG` | Send to spatial address | Planned |
| `TEXTURE` | GPU texture update | Planned |
| `EMBED` | Embed .glyph in texture | Planned |

---

## Session 2026-03-27 Summary

**Extensions documented:**
- `∞` infinite loop (vs official `L`)
- `ROUTE` orchestrator entry points
- `SQL/FS/MODEL/OUTPUT/STATUS/ROADMAP/RAG` syscalls
- `?Q∅` queue empty check
- `M[self]` semantic self-modification
- `S(x, y, [args])` mitosis with spatial coords

**Total opcodes:** 23 (10 official + 13 extensions)

**Token savings:** 90% vs Python orchestrator

---

## References

- Official: https://github.com/GlyphLang/GlyphLang
- Spatial Assembly: `references/spatial-assembly.md`
- Extensions: `EXTENSIONS.md` (this file)
- Our VM: `~/zion/projects/geometry_os/systems/sovereign_vm/`
