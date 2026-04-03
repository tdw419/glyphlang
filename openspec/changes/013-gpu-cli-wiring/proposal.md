# Wire GPU Execution to CLI

## Why
The GPU substrate works in tests (4096 VMs, 100% grid saturation on RTX 5090) but the CLI never touches the GPU. `NewDispatcher()` hardcodes `hasGPU: false`, so `glyph gpu` and `glyph run --gpu` always execute on CPU goroutines. This is the single highest-leverage change.

## Scope

### SEC-1: Enable GPU flag in Dispatcher
- Add `NewGPUDispatcher()` that sets `hasGPU: true` and initializes the PersistentRunner
- Detect if Rust runner binary exists before enabling GPU
- `glyph gpu` command uses `NewGPUDispatcher()`, not `NewDispatcher()`
- Fallback to CPU: `[GPU] No wgpu device available, using CPU fallback`

### SEC-2: Wire multi-pass Mitosis into CLI
- Verify `glyph gpu colony.glyph --vms 1` with real GPU produces >1 VM results

### SEC-3: Add --vms flag to glyph run
- `glyph run file.glyph --gpu --exec --vms 4` should work

### SEC-4: Execution mode indicator
- Print CPU or GPU: `[GPU] Executing on RTX 5090 via IPC daemon`

## Acceptance Criteria
- `glyph gpu colony.glyph --vms 1` spawns >1 VM on real GPU
- `go test ./...` passes
