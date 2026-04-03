# GPU Shader Optimization Pipeline

## Why

The GPU execution path works -- `glyph repl --gpu` evaluates `1+2 => 3` through the standard compiler and GPU VM with real timing data (Init 2.5ms, Compute 1ms). The SSA compiler runs constant folding. But the GPU pipeline currently treats every expression the same way: compile to bytecode, ship to GPU, execute one step, read back.

Real GPU performance comes from batching work and running compute shaders that process many operations in parallel. The infrastructure exists (VCC textures, shared memory, the GPU VM) but nothing uses it efficiently yet.

## Problem

- Every GPU evaluation is a single dispatch: compile -> upload -> execute -> readback. No batching.
- The SSA compiler does constant folding but doesn't generate WGSL (WebGPU Shading Language). Shader compilation is stubbed.
- No way to run a full program on GPU -- only single expressions.
- The ExperimentLoop at `scripts/experiment_loop.py` is wired but has no experiments to run.
- GPU VM timing shows 2.5ms init overhead for 1ms of compute. The ratio gets worse for trivial expressions.

## Solution

Build a shader optimization pipeline:

1. **SSA -> WGSL backend**: Extend the SSA compiler to emit WGSL compute shaders instead of just bytecode. A function like `fn add(a, b) { return a + b }` becomes a WGSL compute shader with `@compute @workgroup_size(64)`.

2. **Batch dispatch**: When the REPL evaluates multiple expressions or a program runs, batch them into a single GPU dispatch instead of N separate round-trips.

3. **ExperimentLoop experiments**: Wire real shader optimization experiments -- compare SSA-compiled bytecode vs WGSL shader vs standard compiler on the same workloads, measure timing, keep the winner.

4. **GPU program execution**: Extend `glyph run --gpu` to execute entire programs on GPU, not just REPL expressions.

## Success Criteria

- [ ] SSA compiler can emit valid WGSL for arithmetic expressions and function calls
- [ ] GPU batch dispatch reduces N expressions to a single GPU round-trip
- [ ] ExperimentLoop runs at least 1 shader optimization experiment with timing comparison
- [ ] `glyph run --gpu examples/hello-world/main.glyph` executes the full program on GPU
- [ ] All 55 Go packages still pass
