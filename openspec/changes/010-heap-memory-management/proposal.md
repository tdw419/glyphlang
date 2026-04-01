# Dynamic Heap Memory Management

## Why

The GlyphLang VM is currently a stack machine with no dynamic memory allocation. All values live on the operand stack or in fixed registers. There is no way to allocate a buffer at runtime, build a data structure that outlives a function call, or pass a pointer to shared data. This severely limits what programs can do:

- No arrays, lists, or dynamic collections.
- No string buffers that grow (needed by the self-hosting compiler).
- No tree or graph structures.
- No way to pass large data without copying it on the stack.

The string support from change 002 introduced tagged value types. Heap allocation extends this with dynamically-sized, pointer-accessible memory.

## What Changes

1. **Heap region per VM instance**: Reserve a contiguous region of VM memory as the heap. Each process (once change 008 lands) gets its own isolated heap. The heap grows upward from a base address. A heap pointer (HP) register tracks the next free address.

2. **Allocation opcodes**:
   - `OpAlloc(size)` — allocate `size` bytes on the heap. Return a pointer (address) to the allocated block. Blocks are aligned to 8-byte boundaries.
   - `OpFree(ptr)` — mark a heap block as freed. Coalesce adjacent free blocks.
   - `OpLoadPtr(ptr, offset)` — load a value from heap memory at `ptr + offset`. Supports loading i32, i64, f64, and tagged value types.
   - `OpStorePtr(ptr, offset, value)` — store a value to heap memory at `ptr + offset`.

3. **Reference counting for managed values**: Every heap allocation has a header with a reference count. When a tagged value (string, array) is stored in a new location, its refcount increments. When the last reference is dropped, the block is freed. This prevents memory leaks without requiring a full garbage collector.

4. **Heap layout**:
   ```
   [block header: size(4) | refcount(4) | type_tag(2)] [data ...] [padding to 8-byte align]
   ```

## Impact

- Enables dynamic data structures: arrays, maps, strings that grow, AST nodes for the self-hosting compiler.
- Required for IPC shared memory (change 009) and VFS file buffers (change 007).
- Adds 4 new opcodes and a heap allocator to the VM.
- Reference counting adds overhead on every pointer store but avoids GC pauses.
- Must be implemented carefully to avoid use-after-free and double-free bugs.
