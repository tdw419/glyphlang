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
