# Formal Process Abstraction

## Why

GlyphLang has Mitosis — the ability to spawn parallel VM instances on the GPU. But Mitosis is a raw parallel execution primitive, not a process. There are no process identifiers, no lifecycle management, no isolation guarantees, and no way for a parent to manage its children. To build a real OS vision on top of the VM, we need a formal process model.

Without this:
- Spawned VM instances are fire-and-forget — there is no way to wait for completion or check exit status.
- There is no concept of process isolation — one misbehaving instance can corrupt shared state.
- IPC (change 009) cannot exist without identifiable, addressable processes.

Change 006 fixes the Mitosis GPU execution path. This change builds on top of that working Mitosis to add proper process semantics.

## What Changes

1. **PID assignment and process table**: Every VM instance gets a unique 32-bit process ID (PID). The VM runtime maintains a global process table mapping PID to VM instance state. PID 1 is reserved for the init/root process.

2. **Process lifecycle**: Implement full lifecycle management:
   - **Spawn** — create a new VM instance (extends Mitosis), assign PID, register in process table, begin execution.
   - **Kill** — terminate a process by PID. Clean up its resources (memory, file descriptors). Propagate signal to children.
   - **Wait** — block the calling process until a target process exits. Return the exit code.
   - **Signal** — send a signal (interrupt, terminate, custom) to a process.

3. **Process isolation**: Each VM instance gets its own:
   - Register file and stack
   - Heap region (isolated memory space)
   - File descriptor table
   - No direct memory access between processes (shared memory requires explicit IPC)

4. **Process hierarchy**: Track parent-child relationships. When a parent dies, its children are reparented to PID 1 (init).

## Impact

- Transforms Mitosis from a parallelism hack into a real process model.
- Enables IPC (change 009) by giving processes identity and isolation.
- Required foundation for the syscall interface (change 011).
- Adds ~4 new opcodes: OpSpawn, OpKill, OpWait, OpSignal.
- Changes the VM runner to manage a process table instead of just executing a single bytecode blob.
