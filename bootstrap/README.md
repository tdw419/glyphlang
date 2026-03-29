# GlyphLang Bootstrap

Self-hosting components for GlyphLang — parsers, compilers, and VM written in GlyphLang itself.

## ✅ v0.9.0 SELF-HOSTING VM

**The GlyphLang VM runs in GlyphLang!**

```
$ ./glyph exec bootstrap/vm.glyph run

=== Bootstrap VM Tests ===

✅ test_simple: 42
✅ test_arithmetic: (5+3)*2 = 16
✅ test_subtraction: 10-7 = 3

Results: 3/3 passed
🎉 All tests passed!
```

## 🧬 v1.0.0 GOAL: TRIPLE CROWN

**The Triple Crown Test:**
```
Stage 1: go-compiler compiles compiler.glyph → compiler.bin
Stage 2: go-vm runs compiler.bin to compile compiler.glyph → compiler_self.bin
Stage 3: glyph-vm runs compiler_self.bin to compile compiler.glyph → compiler_final.bin
Verify: diff compiler_self.bin compiler_final.bin → IDENTICAL ✅
```

## Components

| Component | Lines | Description | Status |
|-----------|-------|-------------|--------|
| token.glyph | 79 | Token type constants | ✅ |
| lexer.glyph | 344 | Self-hosting tokenizer | ✅ |
| ast.glyph | 131 | AST node definitions | ✅ |
| parser.glyph | 1141 | Recursive descent parser | ✅ |
| compiler.glyph | 394 | Bytecode emitter | ✅ |
| **vm.glyph** | **200** | **Self-hosted VM** | ✅ **NEW** |

**Total: 2,289 lines of self-hosting GlyphLang**

## Bootstrap VM (`vm.glyph`)

The self-hosted VM executes GlyphLang bytecode without Go dependency.

### Supported Opcodes

| Category | Opcodes |
|----------|---------|
| Stack | `PUSH`, `POP` |
| Arithmetic | `ADD`, `SUB`, `MUL`, `DIV` |
| Control | `HALT` |

### Architecture

The VM uses a functional style with immutable state:

```glyph
: VM{
  stack: [int]!
  pc: int!
  halted: bool!
}

! vm_push(vm: VM, val: int) -> VM{
  > {
    stack: vm.stack + [val],
    pc: vm.pc,
    halted: vm.halted
  }
}

! vm_exec(vm: VM, code: [int], constants: [int]) -> int{
  $ current = vm
  while !current.halted && current.pc < length(code){
    $ op = code[current.pc]
    # ... execute op ...
  }
  > final.val
}
```

### Usage

```glyph
$ code = [OP_PUSH, 0, OP_PUSH, 1, OP_ADD, OP_HALT]
$ constants = [5, 3]
$ result = exec(code, constants)  # → 8
```

## Architecture

```
Source (.glyph)
    ↓
bootstrap/lexer.glyph → Tokens
    ↓
bootstrap/parser.glyph → AST
    ↓
bootstrap/compiler.glyph → Bytecode (.glyphbin)
    ↓
bootstrap/vm.glyph → Execution ✅ NEW
```

## Milestones

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ | Lexer tokenizes GlyphLang |
| Phase 2 | ✅ | Parser produces correct AST |
| Phase 2.5 | ✅ | Parser parses itself |
| Phase 3 | ✅ | Compiler emits bytecode |
| Phase 3.5 | ✅ | Compiler compiles itself |
| **Phase 4** | ✅ | **VM executes bytecode** ← **NEW** |
| Phase 5 | ⚪ | Bootstrap cycle complete |
| Phase 6 | ⚪ | GPU-native execution |

## Test Files

| Test | Description |
|------|-------------|
| test.glyph | Basic bootstrap tests |
| test_e2e.glyph | End-to-end compilation |
| test_self_compile.glyph | Self-compilation verification |
| test_vm_exec.glyph | VM execution tests |
| test_minimal_runtime.glyph | Minimal runtime tests |
| **vm.glyph (run command)** | **Self-hosted VM tests** ✅ **NEW** |

## Running Tests

```bash
# Bootstrap tests
./glyph exec bootstrap/test.glyph run

# Self-compilation
./glyph exec bootstrap/test_self_compile.glyph run

# Self-hosted VM tests
./glyph exec bootstrap/vm.glyph run
```

## Commits

- `90ff7d3` - Self-compilation achieved (v0.8.0)
- `4a3a594` - Enhanced VM and interpreter
- `232ef77` - Full bootstrap test
- `55df4a4` - Self-hosted compiler

## Next Steps

1. **Expand VM opcodes** - Add `JUMP`, `JUMP_IF_FALSE`, `CALL`, `RETURN`, etc.
2. **Triple Crown Test** - Verify bootstrap cycle
3. **GPU Lowering** - Compile GlyphLang → WGSL for RTX 5090 execution

---

**Milestone:** v0.9.0 Self-Hosting VM ✅ → v1.0.0 Triple Crown ⚪
**Date:** 2026-03-29
**Lines:** 2,289 GlyphLang
**Status:** BOOTSTRAP VM WORKING ✅
