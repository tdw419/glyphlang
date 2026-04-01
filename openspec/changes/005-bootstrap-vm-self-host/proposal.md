# Bootstrap VM Self-Hosting Milestone

## Why

The ultimate goal of the bootstrap phase is a self-hosting GlyphLang: the interpreter written in .glyph can load and execute .glyph source files, including itself. This milestone proves that:

1. The language is expressive enough to implement its own interpreter
2. The bootstrap VM is complete enough to run real programs (not just toy examples)
3. The toolchain (lexer, parser, compiler, VM) is correct enough to handle meta-circular execution

This is the "logo on the flag" moment for the project. Once achieved, all subsequent language development can be done in GlyphLang itself.

## What Changes

1. **Dynamic eval**: Ensure `eval_source(source_string)` works by calling the interpreter recursively. The interpreter's own `eval_source` function must be callable from within interpreted code.
2. **String-intensive operations**: The interpreter does heavy string operations (lexing, parsing). Verify the string support from change 002 handles multi-character tokens, string slicing, and character comparison.
3. **Recursive interpretation**: The interpreter.glyph file imports parser.glyph and lexer.glyph (via module imports from change 004). When run, it reads a test .glyph file, lexes, parses, and interprets it.
4. **Meta-circular test**: The final test: `glyph run interpreter.glyph -- test_program.glyph` where `test_program.glyph` computes something verifiable (e.g., fibonacci(10) == 55).

## Impact

- Proves the language is self-hosting
- Validates all previous changes end-to-end
- Enables future development in GlyphLang itself (no more Go toolchain needed for language work)
- Likely to surface edge cases in VM, compiler, and interpreter that tests didn't catch
- Performance will be slow (interpreted interpreter interpreting programs) but correctness is the goal
