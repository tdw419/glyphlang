# Tasks: Dynamic Heap Memory Management

## 1. Heap allocator implementation
- [x] 1.1 Define the heap region in VM memory layout. Add a HeapPointer (HP) register to the VM state. Initialize HP to heap_base. Define heap_base as a configurable constant (default: address 0x10000, leaving room for stack and globals below).
- [x] 1.2 Implement `OpAlloc(size)`: compute total block size (header + data + padding). Search free list for a best-fit block. If none found, bump HP. Write block header (size, refcount=1, type_tag). Return the data pointer (header address + header_size).
- [x] 1.3 Implement `OpFree(ptr)`: look up the block header at (ptr - header_size). If refcount is 0, add to free list. Coalesce with adjacent free blocks. Guard against double-free by zeroing the type_tag.

## 2. Pointer load/store opcodes
- [ ] 2.1 Implement `OpLoadPtr(ptr, offset)`: compute effective address (ptr + offset). Read the value from VM memory at that address. Decode based on the block's type_tag (i32, i64, f64, string, ptr). Push the decoded value onto the operand stack.
- [ ] 2.2 Implement `OpStorePtr(ptr, offset, value)`: pop value from operand stack. Compute effective address. Encode the value according to the target type_tag. Write to VM memory. Increment refcount if the value is a heap reference; decrement the old value's refcount if overwritten.
- [ ] 2.3 Add bounds checking: verify that (ptr + offset) falls within the heap region. Trap with SEGFAULT on out-of-bounds access.

## 3. Reference counting
- [ ] 3.1 Add refcount increment/decrement to the block header. On every `OpStorePtr` that writes a heap pointer, increment the new target's refcount and decrement the old target's refcount.
- [ ] 3.2 Implement automatic cleanup: when a function frame is popped, scan its local variables for heap pointers and decrement refcounts. When refcount reaches zero, trigger `OpFree` on the block.
- [ ] 3.3 Handle cyclic references with a documented limitation: reference counting cannot collect cycles. Document that the language should prefer acyclic data structures, and that a future mark-and-sweep GC pass can be added if needed.
