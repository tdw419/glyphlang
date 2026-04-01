# Learnings: SEC-1 Channel-based messaging

## What worked
- Channel as a standalone struct with its own mutex keeps concurrency concerns clean.
- Lazily initializing ChannelTable on ProcessTable avoids requiring setup changes in existing tests.
- Circular buffer with head/tail/count is simple and correct for the buffered case.
- Rendezvous channels (capacity 0) use the buf slice as a single-element staging area.
- Testing the Channel data structure separately from the VM opcodes made debugging easy.

## What didn't
- The `createBytecodeHeader` + `parseBytecodeLayout` helpers don't handle the string pool section correctly for direct runLoop execution. The parseBytecodeLayout skips the string pool count but createBytecodeHeader writes it. Tests that call `executeInstruction` directly avoid this mismatch.
- `readOperand()` reads from `vm.code[vm.pc:vm.pc+4]`, so when calling `executeInstruction` directly (bypassing `step()`), `vm.pc` must be positioned at the operand bytes, not at the opcode byte. The `step()` function normally does `vm.pc++` before delegating.

## What I'd do differently
- Consider adding a `stepOver` or `skipOpcode` helper for direct-testing of opcodes that read operands, to avoid the manual PC positioning.
- The `ChannelTable` could be a standalone field on VM instead of nested inside ProcessTable, but for now the lazy-init approach works and keeps the change minimal.

## pattern

- **[pattern]** (from SEC-1) [modified] glyph

- **[pattern]** (from SEC-1) [modified] bootstrap/test_imports.glyphc

- **[pattern]** (from SEC-1) [modified] pkg/vm/process.go

- **[pattern]** (from SEC-1) [added] pkg/vm/channel_test.go

- **[pattern]** (from SEC-1) [modified] pkg/vm/vm.go

## discovery

- **[discovery]** (from SEC-1) Agent strategy: created 3 files, modified 7 files, added tests, fix attempt

---

# Learnings: SEC-2 Shared memory regions

## What worked
- Following the exact same pattern as channels (standalone struct with mutex, lazily-initialized table on ProcessTable) made the implementation consistent and fast.
- Separating access control into the SharedMemoryRegion struct (Grant/Revoke/HasAccess) with its own lock keeps the concurrency model clean.
- Using opcodes 0xD3-0xD6 right after the channel opcodes 0xD0-0xD2 keeps the opcode space organized.
- Testing data structures (SharedMemoryTable, SharedMemoryRegion) separately from VM opcodes catches bugs early.

## What didn't
- OpShmMap currently returns the handle as the "pointer". In a real implementation this would return a heap offset. The spec says "reserved heap offset" but the VM doesn't have a linear heap address space yet, so the handle-as-pointer is a reasonable abstraction.
- The spec mentions "explicit grant/revoke of access between processes" but didn't specify opcodes. Added OpShmGrant (0xD5) and OpShmRevoke (0xD6) to fill this gap.

## What I'd do differently
- If the VM ever gets a real heap with addressable offsets, OpShmMap should map shared memory into a specific offset range and return that offset as the pointer value.
- Could add OpShmWrite/OpShmRead opcodes for direct byte-level access to shared memory, but that's out of scope for this task.

## Files touched
- **[added]** pkg/vm/shm.go — SharedMemoryRegion, SharedMemoryTable, OpShmCreate/Map/Grant/Revoke
- **[added]** pkg/vm/shm_test.go — 23 tests covering table ops, access control, opcodes, and integration
- **[modified]** pkg/vm/process.go — added shmTable field to ProcessTable
- **[modified]** pkg/vm/vm.go — wired 4 new opcodes into executeInstruction

- **[pattern]** (from SEC-2) [modified] pkg/vm/process.go

- **[pattern]** (from SEC-2) [added] pkg/vm/shm.go

- **[pattern]** (from SEC-2) [added] pkg/vm/shm_test.go

- **[pattern]** (from SEC-2) [modified] pkg/vm/vm.go

- **[pattern]** (from SEC-2) [modified] openspec/changes/009-inter-process-communication/learnings.md

- **[discovery]** (from SEC-2) Agent strategy: created 2 files, modified 4 files, added tests

- **[discovery]** (from SEC-3) Agent strategy: modified 1 file

- **[discovery]** (from SEC-3) Tests regressed by 7 (52 -> 45)

- **[discovery]** (from SEC-3) Agent strategy: no-action

---

# Learnings: SEC-3 Synchronization primitives

## What worked
- Following the exact same pattern as channels and shm: standalone data structures with their own mutexes, lazily-initialized SyncTable on ProcessTable.
- Putting Mutex and Semaphore in a single `sync.go` file keeps related code together without creating too many tiny files.
- Opcodes 0xD7-0xDC right after the shm opcodes keeps the opcode space sequential.
- Deadlock detection in Mutex.Lock (same PID tries to acquire twice) catches a common programming error.
- FIFO wait queues on both Mutex and Semaphore ensure fairness.
- Semaphore Signal transfers a permit directly to a waiting process (count stays same) rather than incrementing then decrementing — cleaner semantics.
- 36 tests covering data structures, tables, opcodes, error cases, and integration scenarios.

## What didn't
- Previous attempts (1 and 2) failed because they tried to modify the `builtins.go` file instead of adding proper opcodes. The existing IPC pattern is all opcode-based, not builtin-based.
- Attempt 1 regressed 7 tests — likely from modifying the wrong file or adding conflicting code.

## What I'd do differently
- Could add a Barrier primitive as mentioned in the proposal, but the tasks.md only specifies mutex and semaphore.
- Could add `OpMutexTryLock` (non-blocking variant) but it's out of scope.

## Files touched
- **[added]** pkg/vm/sync.go — Mutex, Semaphore, SyncTable, 6 opcodes (OpMutexCreate/Lock/Unlock, OpSemCreate/Wait/Signal)
- **[added]** pkg/vm/sync_test.go — 36 tests
- **[modified]** pkg/vm/process.go — added syncTable field to ProcessTable
- **[modified]** pkg/vm/vm.go — wired 6 new opcodes into executeInstruction

## discovery
- **[discovery]** (from SEC-3 attempt 3) Agent strategy: created 2 files, modified 2 files, added 36 tests. All existing tests pass, 0 regressions.
