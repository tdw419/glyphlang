# Learnings: SEC-2 Mitosis Integration Test

## What worked
- Debugging with targeted debug tests that isolate one component at a time
- Tracing bytecode execution manually to verify opcode layout and offset calculations
- Using `runMitosisThread` directly to verify child execution before testing the full pipeline
- Replacing channel-based spawn collection with synchronous slice collection in `ExecuteWithMitosis` — eliminates WaitGroup race conditions entirely

## What didn't
- Initial bytecode design didn't account for MITOSIS pushing childID onto the parent stack (stack leak between spawns). Fixed by adding POP after each MITOSIS.
- Original `runThread` delegated to `runOneVM` with fresh state (PC=0, nil stack) — the exact PC=0 bug from #88. Fixed by delegating to `runMitosisThread` instead.
- Original `ExecuteWithMitosis` had a WaitGroup race: `pending.Add(1)` was called in a goroutine that might not execute before `pending.Wait()`. Fixed by switching to synchronous spawn collection.

## What would I do differently
- Design the bytecode layout with POP from the start when multiple spawns are needed
- Test the full pipeline (ExecuteWithMitosis) before the individual components — the WaitGroup race would have been caught earlier
- The channel-based architecture was overengineered for the actual workload; synchronous collection with parallel child execution is simpler and correct

## pattern

- **[pattern]** (from SEC-2) [modified] bootstrap/interpreter.glyph

- **[pattern]** (from SEC-2) [added] bootstrap/test_self_host.glyph

- **[pattern]** (from SEC-2) [modified] pkg/gpu/mitosis.go

- **[pattern]** (from SEC-2) [added] pkg/gpu/mitosis_integration_test.go

- **[pattern]** (from SEC-2) [modified] openspec/changes/006-gpu-mitosis-fix/learnings.md

## discovery

- **[discovery]** (from SEC-2) Agent strategy: created 2 files, modified 4 files, refactored, added tests, fix attempt

---

# Learnings: SEC-3 CPU Mitosis Fallback Detection

## What worked
- Adding `ForceGPUError` field for testability — allows testing fallback path without mocking GPU hardware
- Separating `attemptGPUExecution` from `executeCPUFallback` makes the fallback boundary explicit and testable
- Collecting warnings in a thread-safe slice (`fallbackWarnings` + mutex) allows tests to inspect what was logged without capturing stderr
- The `attemptGPUExecution` currently does validation-only (checks GPU availability and bytecode compatibility) — this is the right scope for "infrastructure" without overcommitting to #78's full implementation

## What didn't
- Nothing unexpected. This was a clean, well-scoped task.

## What would I do differently
- Could have used an interface for the GPU executor instead of a bool flag, but the bool is simpler and sufficient for the current scope. An interface would be warranted when #78 implements actual GPU dispatch.

## Files changed

- **[modified]** pkg/gpu/mitosis.go — added `ForceGPUError` field, `FallbackWarnings()`, `logFallbackWarning()`, `attemptGPUExecution()`, `executeCPUFallback()`; refactored `ExecuteWithMitosis` into GPU-attempt + CPU-fallback
- **[added]** pkg/gpu/mitosis_fallback_test.go — 3 tests: fallback warning emitted, no warning without error, correct results on fallback
