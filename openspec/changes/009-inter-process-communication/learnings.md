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
