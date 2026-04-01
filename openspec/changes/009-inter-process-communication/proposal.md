# Inter-Process Communication (IPC) Channels

## Why

With a formal process model (change 008), processes are isolated — they have separate memory spaces and cannot interact. But real programs need to communicate. A shell needs to pipe output to another command. A server needs to send requests to worker processes. The self-hosting compiler might delegate compilation of individual modules to separate processes and collect results.

Without IPC:
- Processes are completely siloed — no coordination is possible.
- There is no way to build pipelines, request/response patterns, or shared state.
- The OS vision stalls because a multi-process system cannot function without communication.

## What Changes

1. **Channel-based messaging (OpSend/OpRecv)**: Add two core opcodes for typed message passing between processes:
   - `OpSend(channel_id, value)` — send a value to a channel. Blocks if the channel buffer is full.
   - `OpRecv(channel_id)` — receive a value from a channel. Blocks if the channel buffer is empty.
   - Channels are typed buffers with configurable capacity (0 for synchronous rendezvous, N for buffered async).

2. **Shared memory regions**: Allow processes to explicitly map shared memory regions:
   - `OpShmCreate(size)` — create a named shared memory region, return a handle.
   - `OpShmMap(handle)` — map the shared region into the process's address space, return a pointer.
   - Shared memory is the high-throughput path for large data transfers between processes.

3. **Synchronization primitives**: Provide blocking primitives for coordination:
   - Mutex (lock/unlock) — mutual exclusion for shared resources.
   - Semaphore (wait/signal) — counting semaphore for resource pools.
   - Barrier — synchronize N processes at a rendezvous point.

## Impact

- Enables cooperative multi-process programs: pipelines, worker pools, producer-consumer patterns.
- Required for the self-hosting compiler to parallelize module compilation across processes.
- Shared memory provides a zero-copy fast path for GPU data sharing between processes.
- Adds ~6 new opcodes and a channel subsystem to the VM runtime.
- Depends on change 008 for PID addressing and process table awareness.
