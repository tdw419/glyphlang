# Tasks: GPU Shader Optimization Pipeline

## 1. SSA-to-WGSL Backend
- [ ] 1.1 Add `compileWGSL()` method to SSA compiler that emits WGSL compute shader from SSA IR
- [ ] 1.2 Handle arithmetic ops (ADD, SUB, MUL, DIV) in WGSL emitter with proper type annotations
- [ ] 1.3 Handle function calls in WGSL emitter -- emit as WGSL `fn` with `@compute` entry point
- [ ] 1.4 Add tests: given SSA IR for `1+2` and `fn add(a,b) { return a+b }`, verify WGSL output is valid

## 2. GPU Batch Dispatch
- [ ] 2.1 Add `BatchDispatcher` to GPU runtime that collects multiple expressions into a single upload buffer
- [ ] 2.2 Extend REPL GPU path to batch consecutive expressions when not in interactive mode
- [ ] 2.3 Measure and log batch vs single-dispatch timing in GPU VM
- [ ] 2.4 Add tests for batch dispatch with 2+ expressions

## 3. ExperimentLoop Experiments
- [ ] 3.1 Create first experiment: compare standard compiler bytecode vs SSA bytecode vs WGSL shader on arithmetic workload
- [ ] 3.2 ExperimentLoop runs experiment, captures timing, persists results
- [ ] 3.3 Add experiment results to `openspec/learnings.md` for strategist feedback

## 4. GPU Program Execution
- [ ] 4.1 Wire `glyph run --gpu` flag to execute full programs through GPU path
- [ ] 4.2 Handle control flow (if/else, loops) in GPU execution mode
- [ ] 4.3 Test with `examples/hello-world/main.glyph` running end-to-end on GPU

## 5. Verification
- [ ] 5.1 All 55 Go packages pass with new code
- [ ] 5.2 ExperimentLoop produces at least 1 timing comparison result
