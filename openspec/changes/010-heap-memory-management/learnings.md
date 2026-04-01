# Learnings: 010-heap-memory-management

## discovery

- **[discovery]** (from SEC-1) Agent strategy: no-action (first attempt didn't produce code)

## SEC-1 implementation

- Opcode space 0xD0-0xDC is already taken by IPC opcodes (channels, shm, mutex, semaphore). Used 0xE0-0xE3 for heap opcodes instead.
- The VM is a Go-level Value stack machine (not byte-addressed memory). The heap allocator manages Go objects (HeapBlock structs) with address handles, not raw memory. This is cleaner than trying to overlay onto a byte array.
- `vm.hp` field mirrors `heap.hp` for external register access. Must sync after alloc operations.
- `headerSize` is 16 bytes (aligned), covering size(4) + refcount(4) + type_tag(2) + padding(6).
- Free list uses best-fit search. Coalescing checks both forward and backward adjacent blocks.
- Clone() gives the new VM a fresh heap (not shared state) — correct for fork semantics.
