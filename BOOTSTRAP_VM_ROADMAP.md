# Bootstrap VM Roadmap: v0.9.6 → v1.0.0

**Current State (v0.9.6):**
- 23/32 opcodes implemented in bootstrap VM
- Recursion working (factorial(5) = 120)
- Self-compilation works (compiler.glyph → bytecode via Go VM)
- **Gap:** bootstrap VM cannot yet execute compiler output (missing object/array/string ops)

---

## v0.9.7 — Object & Array Opcodes [PRIORITY: 3]

**Goal:** Implement 5 defined-but-unimplemented opcodes

### Tasks
- [ ] OP_BUILD_OBJECT (0x70) — construct {key: value} on the stack
- [ ] OP_GET_FIELD (0x71) — read obj.field
- [ ] OP_SET_FIELD (0x72) — write obj.field = val
- [ ] OP_BUILD_ARRAY (0x80) — construct [a, b, c] on the stack
- [ ] OP_DEF_FUNC (0x73) — define a named function

**Why first:** The compiler emits these opcodes. The bootstrap VM can't run any real GlyphLang program without them — every route handler returns an object.

**Success Test:** `glyph exec bootstrap/vm.glyph` runs a program that builds `{status: 200, body: "ok"}` and reads fields back.

---

## v0.9.8 — String & Index Operations [PRIORITY: 4]

**Goal:** Enable string manipulation and array indexing

### Tasks
- [ ] OP_GET_INDEX (0x56) — arr[i] / str[i]
- [ ] OP_SET_INDEX (0x57) — arr[i] = val
- [ ] String concatenation via OP_ADD type dispatch
- [ ] Value type tagging (int vs string vs object vs array)

**Why:** The compiler and parser both use string operations and array indexing heavily. Without this, the bootstrap VM can't run the lexer.

**Success Test:** Bootstrap VM lexes a simple token stream like `"const x = 42"`.

---

## v0.9.9 — Bootstrap Cycle (THE MILESTONE) [PRIORITY: 3]

**Goal:** Wire the full pipeline inside the bootstrap VM

### Tasks
- [ ] Load compiler.glyph source as a string
- [ ] Lex → Parse → Compile (all running in bootstrap VM)
- [ ] Compare output bytecode to Go-compiled reference
- [ ] Bit-identical match = bootstrap cycle complete

**Why:** This is the "compiler compiles itself inside itself" moment. Proves the bootstrap VM is functionally equivalent to the Go VM for the compiler's needs.

**Success Test:** `test_self_compile.glyph` runs entirely on the bootstrap VM (not the Go runtime).

---

## v0.9.10 — Iterator & Multi-Arg Support [PRIORITY: 5]

**Goal:** Enable for loops and multi-argument functions

### Tasks
- [ ] OP_GET_ITER (0x53), OP_ITER_NEXT (0x54), OP_ITER_HAS_NEXT (0x55)
- [ ] Multi-argument function calls (current CALL assumes 1 arg)
- [ ] OP_STORE_FP for local variable writes in stack frames

**Why:** for loops over arrays and multi-arg functions are used everywhere in the parser/compiler. Needed before the bootstrap VM can run production GlyphLang programs.

---

## v0.10.0 — Drop the Go Training Wheels [PRIORITY: 5]

**Goal:** Full self-hosting

### Tasks
- [ ] Bootstrap VM runs the full toolchain: lex → parse → compile → execute
- [ ] `glyph exec bootstrap/vm.glyph compile myapp.glyph` works end-to-end
- [ ] Go VM only needed for initial bootstrap (like GCC bootstrapping GCC)
- [ ] Target: <2ms compile time for simple programs

---

## v0.11.0 — GPU Dispatch (RTX 5090) [PRIORITY: 6]

**Goal:** GPU-accelerated execution

### Tasks
- [ ] Port bootstrap VM opcodes to vm.wgsl
- [ ] OP_MITOSIS (0xC0) — fork VM instance to GPU thread
- [ ] Route-level parallelism: each HTTP request runs on a separate GPU thread
- [ ] Hilbert curve memory layout for spatial locality

**Why deferred:** GPU execution is an optimization. The bootstrap cycle must be correct first, then fast.

---

## v1.0.0 — Production [PRIORITY: 6]

**Goal:** Production-ready

### Tasks
- [ ] Security audit of bytecode execution
- [ ] Bounded stack/memory per VM instance
- [ ] Error recovery (try/catch via OP_TRY)
- [ ] `glyph run app.glyph` serves production traffic
- [ ] Async I/O integration (OP_ASYNC/OP_AWAIT)

---

## Dependency Graph

```
v0.9.7 Objects/Arrays
        ↓
v0.9.8 Strings/Indexes
        ↓
v0.9.9 Bootstrap Cycle ← CRITICAL PATH
        ↓
v0.9.10 Iterators/Multi-arg
        ↓
v0.10.0 Drop Go dependency
      ↓         ↓
v0.11.0 GPU   v1.0.0 Production
```

**The critical path runs through v0.9.9.** Everything before it is a prerequisite; everything after it is either optimization (GPU) or hardening (production).

---

## Quick Commands

```bash
# Test bootstrap VM
cd ~/zion/projects/glyphlang
PATH="/home/jericho/zion/apps/linux/go/bin:$PATH" go run ./cmd/glyph exec bootstrap/vm.glyph run

# Check current opcode coverage
grep -E "^if op == OP_" bootstrap/vm.glyph | wc -l
```

---

## Tracking

- **Created:** 2026-03-29
- **Source:** Claude Code analysis session
- **Location:** `~/zion/projects/glyphlang/BOOTSTRAP_VM_ROADMAP.md`
