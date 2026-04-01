# Tasks: Fix Broken GPU Execution Path

## 1. Fix constant encoding mismatch
- [x] 1.1 Audit the Go compiler's constant encoding in `pkg/compiler/`: identify the exact format and byte width of emitted constants. Document the current encoding (8-byte u64 per constant, little-endian).
- [x] 1.2 Audit the WGSL shader's constant loading in `vm.wgsl`: identify how it read from the constant table. Document the current decoding (likely 4-byte u32 readss with wrong stride).
- [x] 1.3 Align the two: update the WGSL shader to read 8-byte constants with the correct stride and byte offset. Update the Rust runner's constant buffer preparation to match. Verify with a simple test program that prints a constant.

## 2. Complete missing WGSL opcodes
- [ ] 2.1 Diff the Go compiler's opcode set (`pkg/compiler/opcodes.go` or equivalent) against the WGSL shader's switch/match statement. List all opcodes emitted by the compiler that are not handled by the shader.
- [ ] 2.2 Implement the missing opcodes in vm.wgsl. For each missing opcode, add a case to the main dispatch switch that performs the same operation as the CPU VM. Use WGSL-compatible operations (no heap allocation, no syscalls).
- [ ] 2.3 Handle opcodes that cannot run on GPU gracefully: add a trap/error code for "unsupported on GPU" and ensure the Rust runner can detect and report this to the caller.

## 3. GPU verification tests
- [ ] 3.1 Create a test harness that compiles a GlyphLang program to GLYP bytecode, executes it on the CPU VM, executes it on the GPU via the Rust runner, and compares the results (register state, output values).
- [ ] 3.2 Write 5 verification programs: (1) arithmetic (add/mul/div), (2) comparisons and branches, (3) function calls, (4) constant loading (the encoding fix), (5) loops with ITER_NEXT. Each must produce identical results on CPU and GPU.
- [ ] 3.3 Run the full existing test suite. Confirm all 57 test packages pass (the 3 GPU failures should now be green). No regressions in the 54 already-passing packages.
