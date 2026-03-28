# GlyphLang

The first self-hosting, AI-native backend language with GPU-native bytecode execution.

GlyphLang is a programming language designed for AI agents to rapidly build secure, high-performance backend applications. Its symbol-based syntax uses 35% fewer tokens than Python, making it the most efficient language for AI code generation. The entire VM runs as a WebGPU compute shader — not just specialized kernels, but the full general-purpose stack machine, massively parallelized across GPU threads.

## What makes GlyphLang different

| | Traditional | GPU Languages | GlyphLang |
|---|---|---|---|
| **Execution** | CPU interpreter | GPU kernels (CUDA, WGSL) | Full VM on GPU compute |
| **AI generation** | Verbose syntax | Not designed for AI | 35% fewer tokens, AI-native |
| **Self-hosting** | Rare for new langs | N/A | Compiler written in itself |
| **Debugging** | Text logs | printf debugging | Spatial grid via Hilbert curves |
| **Parallelism** | Threads/goroutines | Explicit kernel launch | S opcode: VM clones itself |

**Key milestones:**
- **v0.8.0** — Self-hosting: `compiler.glyph` compiles itself (14K chars → 12K bytes bytecode)
- **v0.9.0** — GPU native: WGSL compute shader VM, Hilbert spatial mapping, S opcode mitosis

## Quick start

```bash
# Build from source (Go 1.21+)
git clone https://github.com/glyphlang/glyphlang.git
cd glyphlang
make build

# Run a GlyphLang application
glyph run app.glyph

# AI generates and executes GlyphLang
glyph ai "create a REST API for a todo app"

# Execute bytecode on GPU compute backend
glyph compile app.glyph -o app.glyphc
glyph gpu app.glyphc --vms 1000
```

## Hello World

```glyph
@ GET /hello {
    > {message: "Hello, World!"}
}
```

```bash
$ glyph run hello.glyph
[SUCCESS] Server listening on http://localhost:3000

$ curl localhost:3000/hello
{"message": "Hello, World!"}
```

## REST API in 10 lines

```glyph
use db: sqlite("app.db")

@ GET /users {
    $ users = db.All("users")
    > users
}

@ POST /users {
    auth()
    $ user = db.Create("users", body)
    > user
}
```

## Symbol syntax

GlyphLang replaces keywords with symbols for token efficiency:

| Symbol | Meaning | Example |
|--------|---------|---------|
| `@` | Route/command | `@ GET /api` |
| `$` | Variable declaration | `$ x = 42` |
| `>` | Return | `> {ok: true}` |
| `!` | Function definition | `! add(a: int, b: int) -> int` |
| `:` | Type definition | `: User { name: str, age: int }` |
| `#` | Comment | `# this is a comment` |

## Architecture

```
Source (.glyph)
    |
    +--[Parser]--> AST
    |                |
    |    +-----------+-----------+
    |    |           |           |
    | [Interpreter] [Compiler]  [WASM]
    | (tree-walk)    |         (future)
    |                |
    |           Bytecode (.glyphc)
    |                |
    |    +-----------+-----------+
    |    |           |           |
    |   [VM]       [GPU]       [JIT]
    |  (CPU)    (compute)   (optimized)
    |    |           |           |
    |    +-----------+-----------+
    |                |
    |          HTTP Server / WebSocket / gRPC
    |
    +--[AI Pipeline]--> Generate --> Compile --> Execute
```

### Execution paths

1. **Interpreted** — Tree-walking interpreter for development (`glyph run --interpret`)
2. **Compiled** — Bytecode VM for production (`glyph run` default)
3. **GPU** — Compute shader VM for massive parallelism (`glyph gpu`)
4. **AI** — LLM generates GlyphLang, compiles and executes (`glyph ai`)

## GPU native execution

GlyphLang runs its entire bytecode VM as a WebGPU compute shader. Each GPU thread executes an independent VM instance with its own stack and variables.

```bash
# Run 1000 parallel VM instances on GPU
glyph gpu app.glyphc --vms 1000

# View Hilbert spatial grid visualization
glyph gpu app.glyphc --spatial

# Print the WGSL compute shader source
glyph gpu --shader
```

**How it works:**
- GLYP bytecode is uploaded to a GPU storage buffer
- A WGSL compute shader interprets opcodes on each GPU thread (64 threads/workgroup)
- Each thread runs a full stack-based VM with 256-slot stack and 64 variables
- Results are read back to CPU

**S opcode (Mitosis):** A running VM can clone its entire state — stack, variables, program counter — and spawn a child thread at a spatial offset. Biological parallelism.

**Spatial debugging:** Bytecode maps to a 2D grid via Hilbert space-filling curves, preserving locality. Adjacent instructions stay spatially adjacent. Designed for VLM (vision language model) observation.

## Self-hosting

GlyphLang's compiler is written in GlyphLang itself. The bootstrap chain:

```
bootstrap/lexer.glyph    (344 lines)  - Tokenizer
bootstrap/parser.glyph   (1,141 lines) - Recursive descent parser
bootstrap/compiler.glyph (394 lines)  - Bytecode emitter
```

Self-compilation test:
```
Source: 14,395 characters
Parsed: 48 top-level items (31 constants, 2 types, 14 functions)
Output: 12,225 bytes of GLYP bytecode (392 constants, 8,412 instructions)
```

```bash
glyph exec bootstrap/test_self_compile.glyph run
# === SELF-COMPILATION SUCCEEDED ===
# compiler.glyph -> parser -> compiler -> 12225 bytes of bytecode
```

## AI pipeline

```bash
# Generate and execute GlyphLang from a prompt
glyph ai "create an API that converts celsius to fahrenheit"

# Show generated code without executing
glyph ai --code-only "create a user authentication system"

# Use a local LLM
glyph ai --provider ollama --model llama3 "build a REST API"

# Use a custom endpoint (LM Studio, vLLM, etc.)
glyph ai --base-url http://localhost:1234/v1 "create a todo API"
```

The AI pipeline: LLM outputs GlyphLang in `@ command run { }` blocks, which are parsed, compiled to bytecode, and executed through the VM — or directly on GPU.

## CLI reference

| Command | Description |
|---------|-------------|
| `glyph run <file>` | Run a GlyphLang application |
| `glyph dev <file>` | Development server with hot reload |
| `glyph compile <file>` | Compile to bytecode (.glyphc) |
| `glyph gpu <file>` | Execute bytecode on GPU compute |
| `glyph ai "<prompt>"` | AI generates and executes GlyphLang |
| `glyph exec <file> <cmd>` | Execute a CLI command defined in a .glyph file |
| `glyph test <file>` | Run tests |
| `glyph repl` | Interactive REPL |
| `glyph decompile <file>` | Decompile bytecode to source |
| `glyph docs <file>` | Generate API documentation |
| `glyph openapi <file>` | Generate OpenAPI 3.0 spec |
| `glyph lsp` | Start Language Server Protocol server |
| `glyph validate <file>` | Validate source file |

## Benchmarks

GPU compute backend (Intel Ultra 9 275HX):

| Workload | Time | Throughput |
|----------|------|------------|
| Single VM, 1000-iteration loop | 19us | 52K loops/sec |
| 1000 parallel VMs | 327us | 327ns/VM |
| Hilbert coordinate mapping | 13ns | 77M maps/sec |
| Self-compilation | 720ms | 14K chars/sec |

## Project structure

```
cmd/glyph/          CLI entry point
pkg/
  parser/            Lexer + recursive descent parser
  compiler/          Bytecode compiler (Go)
  vm/                Stack-based bytecode VM
  gpu/               WebGPU compute shader backend
    vm.wgsl          WGSL compute shader (full VM)
    hilbert.go       Hilbert curve spatial mapping
    mitosis.go       S opcode parallel VM spawning
  interpreter/       Tree-walking interpreter (fallback)
  ai/                AI -> GlyphLang -> Execute pipeline
  jit/               JIT compilation framework
  server/            HTTP server + middleware
  websocket/         WebSocket hub/room model
  database/          Multi-database provider
  security/          XSS, CSRF, rate limiting, path traversal
  lsp/               Language Server Protocol
bootstrap/
  lexer.glyph        Self-hosted tokenizer
  parser.glyph       Self-hosted parser
  compiler.glyph     Self-hosted compiler
  ast.glyph          AST node definitions
  token.glyph        Token type definitions
examples/            43 example applications
docs/                Language spec, architecture, API reference
```

## Documentation

- [Language Specification](docs/LANGUAGE_SPECIFICATION.md)
- [Architecture Design](docs/ARCHITECTURE_DESIGN.md)
- [API Reference](docs/API_REFERENCE.md)
- [Quick Start Guide](docs/QUICKSTART.md)
- [CLI Reference](docs/CLI.md)
- [Binary Format](docs/BINARY_FORMAT.md)
- [Performance Guide](docs/PERFORMANCE.md)
- [Contributing](CONTRIBUTING.md)
- [Roadmap](ROADMAP.md)

## License

See [LICENSE](LICENSE) for details.
