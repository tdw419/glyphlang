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

## pattern

- **[pattern]** (from SEC-1) [added] pkg/vm/heap.go

- **[pattern]** (from SEC-1) [modified] pkg/vm/vm.go

- **[pattern]** (from SEC-1) [added] pkg/vm/heap_test.go

- **[pattern]** (from SEC-1) [modified] openspec/changes/010-heap-memory-management/learnings.md

- **[pattern]** (from SEC-1) [modified] openspec/changes/010-heap-memory-management/tasks.md

- **[discovery]** (from SEC-1) Agent strategy: created 2 files, modified 3 files, added tests

## SEC-2 implementation

- HeapBlock needed a `Data []byte` field for actual byte-level storage. Without it, LoadPtr/StorePtr had nowhere to read/write. Initialized in `Alloc()` and on free-list reuse.
- Type tags are critical for LoadPtr/StorePtr: the block's TypeTag determines how bytes are decoded on load and encoded on store. Default (TypeTagRaw=0) treats slots as i64. For f64 or ptr values, the caller must set block.TypeTag before using LoadPtr/StorePtr. This is a deliberate design: type is declared at allocation time, not inferred per-access.
- Stack order for StorePtr: pop value first, then offset, then ptr (LIFO). For LoadPtr: pop offset first, then ptr.
- Bounds checking uses `offset + elementSize > block.Size` to prevent both in-bounds partial reads and full out-of-bounds access.
- Added refcount increment on StorePtr when the value is a PointerValue and the target block has TypeTagPtr.
- Tests cover: i64 round-trip, f64 round-trip (with explicit TypeTag), ptr round-trip (with TypeTagPtr), multi-slot offsets, overwrite, out-of-bounds (SEGFAULT), invalid pointer, use-after-free, stack underflow.
- 16 new tests added (72 total in pkg/vm, up from 56).
