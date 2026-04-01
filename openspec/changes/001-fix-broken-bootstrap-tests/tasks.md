# Tasks: Fix Broken Bootstrap Tests

## 1. Fix test_compiler_mutable_closure.glyph parse error
- [x] 1.1 Restructure the if-block containing `> false` on same line as `print()` to eliminate parse ambiguity (move comparison to temp variable or split across lines)
- [x] 1.2 Run the test and verify it passes through the Go parser and bootstrap interpreter

## 2. Fix test_vm_bootstrap.glyph module-level call
- [ ] 2.1 Replace bare `run_tests()` with `@ run_tests()` wrapper syntax accepted by Go parser
- [ ] 2.2 Run the test and verify it passes

## 3. Add nested functions end-to-end test
- [ ] 3.1 Create `test_nested_functions_e2e.glyph` with a nested `! inner()` function, compile to bytecode, execute on VM, and assert correct output
- [ ] 3.2 Verify the test passes through full pipeline: parse -> compile -> vm_exec
