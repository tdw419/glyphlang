# GlyphLang Bootstrap

Self-hosting components for GlyphLang — parsers and compilers written in GlyphLang itself.

## Status

| Component | Lines | Status | Description |
|-----------|-------|--------|-------------|
| token.glyph | 79 | ✅ Complete | Token type definitions |
| lexer.glyph | 344 | ✅ Complete | Self-hosting tokenizer |
| ast.glyph | 344 | ✅ Complete | AST node definitions |
| parser.glyph | 1079 | ✅ Complete | Recursive descent parser |
| test_parser.glyph | 102 | ✅ Complete | Parser test suite |
| compiler.glyph | - | 📋 Next | Bytecode emitter |

**Total: 1,757 lines of self-hosting GlyphLang**

## v0.7.0 Milestone 2 Complete

✅ **Lexer** — Tokenizes GlyphLang source including new keywords (ASSERT, BREAK, CONTINUE)
✅ **AST** — Defines 100% of nodes required for self-hosting
✅ **Parser** — Recursive descent parser written in GlyphLang, produces correct AST trees
✅ **Core Engine** — Precise lexical scoping, debugging builtins

### Test Results

```
=== Test 1: Const declarations === 
Parsed 3 items 
const ILLEGAL = ILLEGAL 
const EOF = EOF 
const NEWLINE = NEWLINE 

=== Test 2: Type definition === 
type Token (4 fields) 
  type: named 
  literal: named 
  line: named 
  col: named 

=== Test 3: Function definition === 
fn is_digit (1 params, 1 stmts) 
  stmt: return 

=== Test 4: Complex snippet === 
Parsed 3 items 
import "./token" 
type Lexer (2 fields) 
fn new_lexer (1 params, 1 stmts) 

Self-hosted parser: ALL TESTS PASSED 
```

## Usage

```bash
# Run parser tests
./glyph exec bootstrap/test_parser.glyph run

# Use in code
import "./parser"

$ p = parser.new_parser("@ GET /users/:id -> User")
$ result = parser.parse(p)
if result.error != "" {
  print("Error: " + result.error)
} else {
  print("Parsed " + str(len(result.mod.items)) + " items")
}
```

## Architecture

```
bootstrap/
├── token.glyph      # Token type constants
├── lexer.glyph      # Tokenizer
├── ast.glyph        # AST node definitions
├── parser.glyph     # Recursive descent parser
├── test_parser.glyph # Test suite
├── test_lexer.glyph # Lexer tests
└── README.md        # This file
```

## Key Design Decisions

### Helper Functions (Go Parser Bug Workaround)

The Go parser has a bug where `func_call(args).field` breaks inside `if/while` conditions. All field access in conditions uses helper functions:

```glyph
# Instead of:
if current(p).type == "IDENT" { ... }

# Use:
! cur_type(p: Parser) -> str {
  $ tok = current(p)
  > tok.type
}

if cur_type(p) == "IDENT" { ... }
```

### Scoping Rules

- `$ x = value` — Declares new variable in current scope
- `x = value` — Reassigns existing variable (searches up the scope chain)
- Proper shadowing: inner `$ x` shadows outer `x`

### Named Result Types

All parser functions return named result structs for clarity:

```glyph
: ParseResult {
  mod: ast.Module!
  error: str!
}

: ExprResult {
  parser: Parser!
  expr: ast.Expr!
  error: str!
}
```

## Bootstrapping Path

1. ✅ **Stage 0:** Write lexer.glyph using Go compiler
2. ✅ **Stage 1:** Write parser.glyph using lexer.glyph
3. 📋 **Stage 2:** Write compiler.glyph using parser.glyph
4. 📋 **Stage 3:** Compile compiler.glyph with itself
5. 📋 **Stage 4:** Delete Go compiler, keep only .glyph

## Next: Phase 3 - Bytecode Emission

`compiler.glyph` will transform AST into VM instructions:

- **Constants table** — String literals, numbers
- **Instruction stream** — Opcodes + operands
- **Jump targets** — Label resolution for branches

This is the final piece before full self-hosting.

---

**Milestone:** v0.7.0-alpha Phase 2 Complete
**Date:** 2026-03-28
**Lines:** 1,757 GlyphLang
**Tests:** ALL PASSED
