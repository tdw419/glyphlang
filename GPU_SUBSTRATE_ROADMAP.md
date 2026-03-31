# GlyphLang GPU Substrate Roadmap

## Problem Statement

The GPU substrate works in tests (4096 VMs, 100% grid saturation verified on RTX 5090)
but the CLI never actually touches the GPU. `NewDispatcher()` hardcodes `hasGPU: false`,
so `glyph gpu` and `glyph run --gpu` always execute on CPU goroutines. The multi-pass
Mitosis loop, VCC texture bridge, and persistent IPC daemon are all functional but
disconnected from the user-facing commands.

## Phase 1: Wire GPU to CLI (Critical Path)

Unblock `glyph gpu` and `glyph run --gpu --exec` from actually using the GPU daemon.

### 1a. Enable GPU flag in Dispatcher
- `NewDispatcher()` must detect if the Rust runner binary exists and wgpu is available
- Add `NewGPUDispatcher()` that sets `hasGPU: true` and initializes the PersistentRunner
- The `glyph gpu` command should use `NewGPUDispatcher()`, not `NewDispatcher()`
- Fallback to CPU with a clear log message: `[GPU] No wgpu device available, using CPU fallback`

### 1b. Wire multi-pass Mitosis into the CLI execution path
- The CPU `executeCPU` path runs all VMs in parallel goroutines but ignores Mitosis
- The GPU path via `dispatcher.Execute()` → `executeGPU()` → `ExecuteMultiWGSL()` 
  already has multi-pass in the Rust daemon
- Verify that `glyph gpu colony.glyph --vms 1` with real GPU produces >1 VM results

### 1c. Add --vms flag to `glyph run`
- `glyph run file.glyph --gpu --exec --vms 4` should work
- Currently only `glyph gpu` has `--vms`

### 1d. Execution mode indicator
- Print whether execution is on CPU or GPU: `[GPU] Executing on RTX 5090 via IPC daemon`
- The current "18µs" timing is misleading when it's actually CPU fallback

## Phase 2: Substrate Correctness

Make the substrate behave correctly, not just architecturally.

### 2a. CPU Mitosis implementation
- `executeCPU` / `runOneVM` silently ignores OP_MITOSIS (no spawn behavior)
- Either implement Mitosis in the CPU path (goroutine spawning) or return an error:
  `[CPU] OP_MITOSIS requires GPU execution. Use --gpu flag.`

### 2b. Spawn offset semantics
- `spawn(offset)` pushes the offset for the child's PC = parent.PC + offset
- Verify the compiler emits correct offset values for labels and function addresses
- The decompile shows `PUSH 0; MITOSIS` — offset 0 means child starts at parent.PC
  which may cause infinite spawn loops

### 2c. Depth guard testing
- The MAX_PASSES=8 and MAX_VMS=65536 guards exist in Rust but are untested at boundary
- Write a test that spawns until hitting MAX_VMS and verify clean termination

### 2d. Error propagation from Rust daemon
- If the Rust daemon crashes, the Go PersistentRunner should detect it and restart
- Currently it probably hangs waiting on stdout

## Phase 3: Developer Experience

Make the substrate usable for development.

### 3a. REPL GPU mode
- `glyph repl --gpu` should execute expressions on the GPU daemon
- Even simple expressions should work: `1 2 +` → 3 via GPU compute

### 3b. Live VCC streaming
- Replace `std::fs::write("/tmp/vcc_colony.rgba")` with in-memory pipe or SHM
- Go HTTP server should serve live-updating texture data, not a static file
- Target: 30Hz minimum for visual feedback

### 3c. Terminal spatial visualization
- `glyph gpu colony.glyph --vms 64 --spatial` should render the Hilbert grid in terminal
- The `--spatial` flag exists but uses static bytecode analysis, not live GPU state
- Update to read from VCC texture output after execution

### 3d. Glyph file examples for substrate
- Create `examples/colony_linear.glyph` — linear spawn (1 parent → N children)
- Create `examples/colony_recursive.glyph` — exponential spawn (each child spawns more)
- Create `examples/colony_conditional.glyph` — spawn based on computation result

## Phase 4: Performance & Scale

Make it fast enough to be impressive.

### 4a. Pipeline caching verification
- The daemon caches pipeline/buffers at startup (commit 6cf5a4c)
- Benchmark: run 1000 consecutive jobs and measure P50/P99 latency
- Target: <1ms per job for cached pipeline

### 4b. Grid scaling (64x64 → 256x256)
- Update `VCC_SIZE` constant in vm.wgsl from 64 to 256
- Update texture and buffer allocations in Rust runner
- 256x256 = 65,536 VMs — test that multi-pass Mitosis can saturate it
- Verify Hilbert d2xy correctness for VM IDs > 4096

### 4c. Persistent daemon across CLI invocations
- Currently the daemon starts/stops per `glyph gpu` invocation
- Implement a Unix socket or named pipe so the daemon stays alive between runs
- First invocation: start daemon. Subsequent: reuse running daemon.

### 4d. Asynchronous job results
- For 60Hz streaming, the Go side needs non-blocking job submission
- Submit job → continue → poll for results on next frame
- Requires restructuring PersistentRunner from sync.Mutex to a job queue

## Phase 5: Integration with Geometry OS

Connect the substrate to the broader ecosystem.

### 5a. PixiJS/WebGPU consumer
- JavaScript client that connects to :8080/vcc/colony.rgba
- Renders the texture as a live canvas using WebGPU
- Color legend: green=running, blue=halted, red=error, dark=idle

### 5b. Area Agent API
- Expose substrate control via a programmatic Go API (not just CLI)
- `agent.Spawn("compute.glyph", 100)` → submits job to GPU daemon
- `agent.Query(vmID)` → returns current VM state
- `agent.Watch()` → streams VCC texture updates

### 5c. Hot-reload during execution
- Modify .glyph source → compiler recompiles → daemon hot-swaps shader
- The Rust runner already has `GLYPH_WATCH` mode for shader hot-reload
- Wire `glyph dev file.glyph --gpu --watch` to use it

## Dependency Order

```
Phase 1a → 1b → 1c → 1d (sequential, enables everything else)
Phase 2a-2d (parallel, after Phase 1)
Phase 3a-3d (parallel, after Phase 2)
Phase 4a-4d (parallel, after Phase 3)
Phase 5a-5c (parallel, after Phase 4)
```

## Success Metrics

- `glyph gpu colony.glyph --vms 1` spawns >1 VM on real GPU (not CPU fallback)
- `glyph gpu colony.glyph --vms 256 --spatial` renders populated Hilbert grid in <100ms
- Continuous 60Hz execution for 60 seconds without crash or memory leak
- 256x256 grid (65,536 VMs) saturating in <500ms on RTX 5090
