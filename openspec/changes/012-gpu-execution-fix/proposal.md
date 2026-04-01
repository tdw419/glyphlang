# Fix Broken GPU Execution Path

## Why

Three test packages are failing, and all three are GPU-related. The GPU execution path — where the Rust runner dispatches GlyphLang bytecode to the RTX 5090 via WebGPU/WGSL shaders — is broken. This is the highest-impact bug in the codebase right now because:

1. **Constant encoding mismatch**: The Go compiler writes constants as 8-byte (u64) values, but the WGSL shader reads them as 4-byte (u32) values. This causes every constant to be misaligned, producing garbage values in GPU execution.

2. **Missing WGSL opcodes**: The vm.wgsl shader has incomplete opcode handling. Several opcodes that the Go compiler emits are not implemented in the shader, causing unhandled-opcode traps on the GPU.

3. **No verification**: Even when the GPU path "runs," there is no end-to-end test that verifies the GPU produces the same results as the CPU VM for a given bytecode program.

This change has no dependencies and is the critical-path blocker. GPU execution must work before Mitosis (change 006 fix) and the process model (change 008) can leverage GPU parallelism.

## What Changes

1. **Fix constant encoding mismatch**: Align the Go compiler's constant encoding with the WGSL shader's expectations. Two options:
   - Option A: Change the Go compiler to emit 4-byte constants (breaks CPU VM).
   - Option B: Change the WGSL shader to read 8-byte constants (requires adjusting the constant table stride in the shader).
   - Option B is preferred because it does not break the CPU path.

2. **Complete missing WGSL opcodes**: Audit the Go compiler's opcode set against vm.wgsl. Implement any missing opcodes in the shader. Key candidates:
   - String operations (from change 002)
   - Comparison operations
   - Jump/call operations that may be missing

3. **Add GPU verification tests**: Create test programs that execute on both CPU VM and GPU, then compare results. This catches encoding mismatches and opcode bugs that would otherwise go undetected.

## Impact

- Fixes the 3 failing GPU test packages, bringing the test suite to 57/57 green.
- Unblocks Mitosis GPU parallelism (change 006 depends on GPU working correctly).
- Enables the process model (change 008) to use GPU for parallel process execution.
- Critical path with no dependencies — should be the first change implemented.
- No changes to the Go compiler's code generation logic (fix is in the shader and Rust runner).
