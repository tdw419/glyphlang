# GlyphLang Bootstrap

Self-hosting components for GlyphLang — parsers, compilers, and VM written in GlyphLang itself.

## ✅ v0.8.6 BOOTSTRAP VM WORKING

**The GlyphLang VM now runs in GlyphLang itself with 68% opcode coverage!**

```bash
$ ./glyph exec bootstrap/vm.glyph run

=== Bootstrap VM Tests (Extended) ===
 
✅ test_simple: 42 
✅ test_arithmetic: (5+3)*2 = 16 
✅ test_comparison: 5 < 10 = 1 
✅ test_conditional: if(5>3) = 1 
✅ test_logic: (5>3) AND (10>5) = 1 
✅ test_or: (5<3) OR (10>5) = 1 
✅ test_load_var: LOAD_VAR 0 = 42 
✅ test_loop (unrolled): 1+2+3+4+5 = 15 
 
Results: 8/8 passed
🎉 All tests passed!
[INFO] Execution time: 517.804µs
```

## 🧬 v0.9.0 GOAL: FULL SELF-HOSTING

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
| **vm.glyph** | **550** | **Self-hosted VM** | ✅ **v0.8.6** |

**Total: 2,639 lines of self-hosting GlyphLang**

## Bootstrap VM (`vm.glyph`)

The self-hosted VM executes GlyphLang bytecode without Go dependency.

### Opcode Coverage: 23/34 (68%)

| Category | Opcodes | Status |
|----------|---------|--------|
| Stack | PUSH, POP | ✅ |
| Arithmetic | ADD, SUB, MUL, DIV, MOD | ✅ |
| Comparison | EQ, NE, LT, GT, GE, LE | ✅ |
| Logic | AND, OR, NOT, NEG | ✅ |
| Variables | LOAD_VAR, STORE_VAR | ✅ |
| Control Flow | JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE, HALT | ✅ |
| Functions | CALL, RETURN | ⏳ Next |
| Arrays | BUILD_ARRAY | ⏳ |
| Objects | BUILD_OBJECT, GET_FIELD, SET_FIELD, DEF_FUNC | ⏳ |
| Iterators | GET_ITER, ITER_NEXT, ITER_HAS_NEXT | ⏳ |

### Architecture

```
Source (.glyph)
    ↓
bootstrap/lexer.glyph → Tokens
    ↓
bootstrap/parser.glyph → AST
    ↓
bootstrap/compiler.glyph → Bytecode (.glyphbin)
    ↓
bootstrap/vm.glyph → Execution ✅ WORKING
```

### Design Principles

1. **Immutable Structs** - All VM state is immutable, functions return new states
2. **Functional Style** - No mutation, only transformation
3. **Helper Functions** - Complex conditionals extracted to avoid scoping issues
4. **4-byte Addresses** - Little-endian for jump targets

### Example

```glyph
# Compute (5+3)*2 = 16
$ code = [
  OP_PUSH, 0,      # Push constants[0] = 5
  OP_PUSH, 1,      # Push constants[1] = 3
  OP_ADD,          # 5 + 3 = 8
  OP_PUSH, 2,      # Push constants[2] = 2
  OP_MUL,          # 8 * 2 = 16
  OP_HALT
]
$ constants = [5, 3, 2]
$ result = exec(code, constants)
print(result)  # Output: 16
```

## Running Tests

```bash
# Bootstrap VM tests
./glyph exec bootstrap/vm.glyph run

# Self-compilation
./glyph exec bootstrap/test_self_compile.glyph run

# All bootstrap tests
./glyph exec bootstrap/test.glyph run
```

## Milestones

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ | Lexer tokenizes GlyphLang |
| Phase 2 | ✅ | Parser produces correct AST |
| Phase 2.5 | ✅ | Parser parses itself |
| Phase 3 | ✅ | Compiler emits bytecode |
| Phase 3.5 | ✅ | Compiler compiles itself |
| **Phase 4** | ✅ | **VM executes bytecode (68% coverage)** |
| Phase 5 | ⚪ | CALL/RETURN for functions |
| Phase 6 | ⚪ | Full bootstrap cycle complete |
| Phase 7 | ⚪ | GPU-native execution |

## Recent Commits

- `ee1bfda` - Extended bootstrap VM (23 opcodes, 68% coverage)
- `vm-bootstrap` - Self-hosted VM working (v0.8.5)
- `90ff7d3` - Self-compilation achieved (v0.8.0)
- `4a3a594` - Enhanced VM and interpreter

---

**Milestone:** v0.8.6 Bootstrap VM Extended ✅
**Next:** v0.9.0 CALL/RETURN for Full Self-Hosting ⚪
**Date:** 2026-03-29
**Lines:** 2,639 GlyphLang
**Status:** 68% OPCODE COVERAGE
