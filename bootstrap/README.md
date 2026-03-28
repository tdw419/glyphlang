# GlyphLang Bootstrap

Self-hosting components for GlyphLang — parsers and compilers written in GlyphLang itself.

## ✅ v0.7.0 BOOTSTRAP COMPLETE

**The GlyphLang compiler is now self-hosting!**

```
$ ./glyph exec bootstrap/test.glyph run

=== BOOTSTRAP TESTS ===

Test 1 (Const): PASSED
Test 2 (Type): PASSED

=== ALL TESTS PASSED ===
```

## Components

| Component | Lines | Description |
|-----------|-------|-------------|
| token.glyph | 79 | Token type constants |
| lexer.glyph | 344 | Self-hosting tokenizer |
| ast.glyph | 131 | AST node definitions |
| parser.glyph | 1141 | Recursive descent parser |
| compiler.glyph | 394 | Bytecode emitter |
| test.glyph | 55 | Test suite |
| full_bootstrap.glyph | 25 | Full bootstrap test |

**Total: 2,169 lines of self-hosting GlyphLang**

## Milestones

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ | Lexer tokenizes GlyphLang |
| Phase 2 | ✅ | Parser produces correct AST |
| Phase 2.5 | ✅ | Parser parses itself |
| Phase 3 | ✅ | Compiler emits bytecode |

## Key Features

### Qualified Type Names
```glyph
: Parser {
  tokens: [token.Token]!  # Works!
  pos: int!
}
```

### Bare Reassignment
```glyph
$ x = 10    # Declaration
x = 20      # Reassignment
```

### Index Assignment
```glyph
$ arr = [1, 2, 3]
arr[0] = 99  # Works!
```

## Architecture

```
Source (.glyph)
    ↓
bootstrap/lexer.glyph → Tokens
    ↓
bootstrap/parser.glyph → AST
    ↓
bootstrap/compiler.glyph → Bytecode
    ↓
VM executes
```

## Commits

- `232ef77` - Full bootstrap test
- `55df4a4` - Self-hosted compiler
- `50f3e79` - Parser parsing itself
- `dfe21b5` - Core engine upgrades

---

**Milestone:** v0.7.0 Bootstrap Complete
**Date:** 2026-03-28
**Lines:** 2,169 GlyphLang
**Status:** SELF-HOSTING ✅
