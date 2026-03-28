# GlyphLang Bootstrap

Self-hosting components for GlyphLang — parsers and compilers written in GlyphLang itself.

## ✅ MILESTONE: Self-Hosting Parser

**The GlyphLang parser can now parse itself!**

```
$ ./glyph exec bootstrap/test_self_host.glyph run

=== Self-Hosting Test ===
Parsing parser.glyph (29929 chars)... 
SUCCESS! Parsed 75 items 

=== BOOTSTRAP COMPLETE ===
The GlyphLang parser can parse itself!
```

## Status

| Component | Lines | Status | Description |
|-----------|-------|--------|-------------|
| token.glyph | 79 | ✅ Complete | Token type definitions |
| lexer.glyph | 344 | ✅ Complete | Self-hosting tokenizer |
| ast.glyph | 131 | ✅ Complete | AST node definitions |
| parser.glyph | 1085 | ✅ Complete | **Parses itself!** |
| compiler.glyph | 372 | ✅ Complete | Bytecode emitter |
| test_self_host.glyph | 45 | ✅ Working | Self-host verification |

**Total: 2,056 lines of self-hosting GlyphLang**

## v0.7.0 Milestones

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ Complete | Lexer tokenizes GlyphLang |
| Phase 2 | ✅ Complete | Parser produces correct AST |
| Phase 2.5 | ✅ **COMPLETE** | **Parser parses itself** |
| Phase 3 | 🚧 Next | Compiler emits bytecode |
| Phase 4 | 📋 Planned | Full bootstrap (compile compiler with itself) |

## Key Features

### Qualified Type Names

```glyph
: Parser {
  tokens: [token.Token]!  # Works now!
  pos: int!
}
```

### Bare Reassignment

```glyph
$ x = 10    # Declaration
x = 20        # Reassignment (no $ needed)
```

### Self-Hosting Test

```bash
./glyph exec bootstrap/test_self_host.glyph run
```

## Architecture

```
bootstrap/
├── token.glyph          (79 lines) — Token constants
├── lexer.glyph          (344 lines) — Tokenizer
├── ast.glyph            (131 lines) — AST nodes
├── parser.glyph         (1085 lines) — Parser ✅ PARSES ITSELF
├── compiler.glyph       (372 lines) — Bytecode emitter
├── test_lexer.glyph     (22 lines) — Lexer tests
├── test_parser.glyph    (102 lines) — Parser tests
├── test_compiler.glyph  (98 lines) — Compiler tests
├── test_self_host.glyph (45 lines) — Self-host verification
└── README.md            — This file
```

## Next Steps

1. **Bytecode emission** — compiler.glyph produces VM instructions
2. **VM integration** — Run the the compiled bytecode
3. **Full bootstrap** — Compile compiler.glyph with itself
4. **Delete Go compiler** — True self-hosting

---

**Milestone:** v0.7.0-alpha Phase 2.5 Complete
**Date:** 2026-03-28
**Lines:** 2,056 GlyphLang
**Status:** PARSER PARSES ITSELF ✅
