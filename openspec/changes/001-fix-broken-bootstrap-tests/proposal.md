# Fix Broken Bootstrap Tests

## Why

Two bootstrap test files are currently broken and blocking all downstream progress:

1. **test_compiler_mutable_closure.glyph**: Parse error from `> false` on same line as `print()` inside an if block. The test file structure triggers a parser ambiguity.
2. **test_vm_bootstrap.glyph**: Bare `run_tests()` call at module level is rejected by the Go parser. Needs the `@` command run wrapper syntax.

Additionally, there is no end-to-end test that exercises nested functions through the full pipeline (parse -> compile -> vm_exec). This is needed to validate the nested `!` function support just added to the Go toolchain.

Without these tests passing, we cannot verify that fixes from this session (ITER_NEXT width, jump patching, nested functions) actually work end-to-end.

## What Changes

1. Restructure `test_compiler_mutable_closure.glyph` to avoid the `> false` parse ambiguity. Move the comparison to a separate variable assignment or restructure the if-block.
2. Wrap `run_tests()` in `test_vm_bootstrap.glyph` with `@ run_tests()` so the Go parser accepts it as a command entry point.
3. Add a new test file `test_nested_functions_e2e.glyph` that defines a nested `! inner()` function, compiles it, and verifies correct execution through the VM.

## Impact

- Unblocks all Phase 2+ work that depends on a green test suite
- Validates the 3 fixes from this session (ITER_NEXT width, jump patching, nested functions)
- Establishes the compile-and-execute test pattern for future VM tests
- No breaking changes to existing passing tests (230/230 remain green)
