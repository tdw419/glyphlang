# Fix #88: Mitosis Children Not Executing

## Why

The GPU substrate is achieving 70fps live mode on RTX 5090, but Mitosis (parallel child spawn) is broken. Children are spawned but never execute because of a PC=0 initialization bug in the Rust runner. The program counter is not being set correctly when child processes are dispatched to GPU compute units.

This is a critical blocker for GPU-parallel GlyphLang execution. Mitosis is the core parallelism primitive -- without it, GPU execution is limited to single-threaded kernel launches.

Related issues:
- #78: CPU Mitosis fallback (depends on this fix)
- #85: Integration tests for GPU execution path (needs working Mitosis)
- #79/#76: --vms flag for glyph run (needs working Mitosis for multi-VM)

## What Changes

1. **Root cause**: The Rust runner initializes child process PC to 0 (start of bytecode) instead of the correct entry point for the spawned child function. Fix the PC initialization to use the child's actual code offset.
2. **Integration test**: Add a test that spawns 2+ Mitosis children, each computing a value, and verifies all children complete and produce correct results.
3. **CPU fallback stub**: Add a flag/path that detects Mitosis failure and falls back to CPU execution (prerequisite for #78).

## Impact

- Unblocks GPU-parallel execution
- Required for #78, #85, #79, #76
- No impact on non-Mitosis GPU execution (70fps live mode stays working)
- This is on the parallel GPU track and does not block bootstrap/self-hosting work
