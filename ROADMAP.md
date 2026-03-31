# GlyphLang Roadmap

Current: **v0.8.0** (self-hosting) + **v0.9.0** (GPU compute backend)

Production readiness: **7.5/10** — core pipeline solid, needs quality pass and CI/CD.

---

## Phase 1: Stabilization (v0.9.1)

Priority: Code quality, testing gaps, CI/CD. Unblock v1.0.

### P1: CI/CD Pipeline
- [x] GitHub Actions: test on push/PR (go test ./...)
- [ ] GitHub Actions: lint enforcement (gofmt, golangci-lint)
- [ ] GitHub Actions: build matrix (linux/mac/windows x amd64/arm64)
- [x] Coverage reporting (codecov or similar)
- [x] Release automation (goreleaser on tag push)

### P2: Code Quality (13 items from CODE_REVIEW_ISSUES.md)
- [x] P2-1: Run gofmt across all 72 files with violations
- [x] P2-2: Split cmd/glyph/main.go (459 lines) into separate command files
- [x] P2-3: Decompose evaluateFunctionCall() (618 lines) in interpreter
- [x] P2-4: Move AST types from pkg/interpreter to pkg/ast
- [x] P2-5: Move MockInterpreter to test-only file
- [ ] P2-6: Replace stub test assertions with real tests
- [x] P2-7: Resolve remaining TODO comments
- [ ] P2-8: Dead code cleanup
- [x] P2-9: Consistent pointer vs value types in compiler
- [x] P2-10: Inject version via ldflags (remove hardcoded string)
- [x] P2-11: Enforce gofmt in CI (covered by CI/CD above)
- [x] P2-12: Remove os.Exit() from non-main functions
- [x] P2-13: Scrub database passwords from log output

### P3: Test Coverage Gaps
- [x] Add tests for pkg/config (0 tests)
- [x] Add tests for pkg/runtime (0 tests)
- [x] Add tests for pkg/wasm (0 tests, stub package)
- [x] Strengthen pkg/jit tests (3 tests for 2,945 LOC)
- [x] Strengthen pkg/lsp tests (6 tests for 8,477 LOC)
- [ ] Replace stub assertions in existing tests (P2-6)

### P4: Documentation
- [x] Create README.md (project overview, quickstart, architecture)
- [ ] Create CLAUDE.md (AI assistant context file)
- [x] Document bootstrap design rationale (why self-hosting matters)
- [x] Document GPU spatial grid execution model
- [x] Document bytecode format (GLYP header, LE encoding, constant pool)

---

## Phase 2: Integration (v0.10.0)

Priority: Connect isolated subsystems into the main pipeline.

### GPU Integration
- [ ] Wire GPU dispatcher into route execution (auto-fallback to CPU)
- [ ] Add `--gpu` flag to `glyph run` for GPU-accelerated route handling
- [ ] Profile-guided GPU dispatch: auto-offload when parallelism > threshold
- [ ] Synchronize GPU opcodes with VM opcodes (single source of truth)
- [ ] Investigate wgpu-native Go bindings (or dawn/webgpu headers via CGo)

### JIT Activation
- [ ] Connect JIT profiler to VM execution loop
- [ ] Implement hot path detection trigger (execution count threshold)
- [ ] Complete type specialization code generation
- [ ] Add benchmarks proving JIT speedup vs baseline
- [ ] Tier progression: Interpreted → Baseline JIT → Optimized JIT

### WASM Target
- [ ] Implement WASM compilation backend in pkg/wasm
- [ ] `glyph compile --target wasm` CLI command
- [ ] WASM import/export interface for host environment
- [ ] Browser-compatible output (wasm-bindgen or raw exports)
- [ ] Tests for WASM output correctness

### Bootstrap Compiler Extensions
- [ ] Add iterator opcodes (GET_ITER, ITER_NEXT, ITER_HAS_NEXT)
- [ ] Add async/await opcodes (ASYNC, AWAIT, TRY)
- [ ] Add WebSocket opcodes (for full-stack self-hosting)
- [ ] Add GPU opcodes (MITOSIS, MUTATOR) to bootstrap compiler

---

## Phase 3: Production Hardening (v1.0.0)

Priority: Security, performance, reliability for production deployments.

### Security
- [ ] Third-party security audit
- [ ] Resolve remaining 3 P1 issues (P1-2, P1-8, P1-9)
- [ ] Fuzz testing for parser and VM (go-fuzz or native fuzzing)
- [ ] Dependency vulnerability scan (govulncheck)
- [ ] Secrets scanning in CI (prevent credential commits)

### Performance
- [ ] Comprehensive benchmark suite (parser, compiler, VM, HTTP)
- [ ] Regression testing: benchmarks in CI with threshold alerts
- [ ] Memory profiling under sustained load
- [ ] Garbage collection tuning for long-running servers
- [ ] Connection pool optimization for database providers

### Reliability
- [ ] Stress testing: 10K concurrent requests
- [ ] Large bytecode handling (>1MB programs)
- [ ] Graceful shutdown and signal handling
- [ ] Health check endpoint standardization
- [ ] Error recovery and panic handler hardening

### Ecosystem
- [ ] Package registry (`glyph mod publish`)
- [ ] VS Code extension (syntax highlighting, LSP integration)
- [ ] Playground (browser-based REPL using WASM target)
- [ ] Example gallery (expand from 43 examples)
- [ ] Community contribution guide updates

---

## Phase 4: Advanced Features (v1.x)

### Spatial Computing (from spatial-assembly spec)
- [ ] M opcode: self-modifying code (Ouroboros mutator)
- [ ] S opcode: full spatial grid mitosis with VLM feedback
- [ ] Visual Consistency Contract (VCC): green/red flash validation
- [ ] Hilbert curve instruction layout for VLM-observable debugging
- [ ] Multi-dimensional execution layers (1D→2D→4D→8D)

### AI-Native Features
- [ ] AIPM integration: GlyphLang compression for agent workflows
- [ ] AI pipeline improvements: multi-turn code generation
- [ ] Neuro-spatial fusion: spatial grid ↔ neural network bridge
- [ ] Agent-to-agent communication via CHAN/MSG opcodes

### Performance Tiers
- [ ] GPU-accelerated route handling (wgpu-native integration)
- [ ] JIT with aggressive optimization (inlining, devirtualization)
- [ ] AOT compilation to native binary (via LLVM or cranelift)
- [ ] SIMD vectorization for batch operations

---

## Metrics

| Metric | Current | Phase 1 Target | v1.0 Target |
|--------|---------|----------------|-------------|
| Test files | 177 | 185+ | 200+ |
| P2 issues | 13 open | 0 open | 0 open |
| CI/CD | None | Full pipeline | Full + release |
| Packages tested | 47/51 | 51/51 | 51/51 |
| Code formatted | ~72 violations | 0 | 0 |
| README | Missing | Complete | Complete |
| GPU integration | Isolated | Route-aware | Auto-dispatch |
| Production score | 7.5/10 | 8.5/10 | 9.5/10 |

---

## Architecture

```
Source (.glyph)
    │
    ├──[Parser]──► AST
    │                │
    │    ┌───────────┼───────────┐
    │    │           │           │
    │  [Interpreter] [Compiler]  [WASM]
    │  (tree-walk)   │          (future)
    │                │
    │           Bytecode (.glyphc)
    │                │
    │    ┌───────────┼───────────┐
    │    │           │           │
    │   [VM]       [GPU]       [JIT]
    │  (CPU)    (compute)   (optimized)
    │    │           │           │
    │    └───────────┼───────────┘
    │                │
    │           HTTP Server
    │           WebSocket
    │           gRPC / GraphQL
    │
    └──[AI Pipeline]──► Generate → Compile → Execute
```
