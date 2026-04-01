# Learnings: 008-process-model


## pattern

- **[pattern]** (from SEC-1) [modified] glyph

- **[pattern]** (from SEC-1) [modified] pkg/compiler/compiler.go

## discovery

- **[discovery]** (from SEC-1) Agent strategy: modified 2 files

- **[discovery]** (from SEC-1) Tests improved by 8 (46 -> 54)

- **[discovery]** (from SEC-2) Agent strategy: created 3 files, modified 3 files, added tests, fix attempt

- **[discovery]** (from SEC-2) Tests regressed by 1 (54 -> 53)
- **[discovery]** (from SEC-2 fix) Root cause: previous attempt modified cmd/glyph/commands.go (unrelated to process model) switching from manual route iteration to c.Compile(module), which changed the error message for empty modules from "no runnable item" to "empty module: no items to compile", breaking TestCompileNoRoutes.
- **[discovery]** (from SEC-2 fix) Fix: reverted commands.go to pre-SEC-2 version. Process model code in pkg/vm/ was correct and unrelated to the regression.
- **[learning]** (from SEC-2) Don't bundle unrelated refactors with feature work. The commands.go change had nothing to do with lifecycle opcodes.
