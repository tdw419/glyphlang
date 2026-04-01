# Learnings: SEC-1 Syscall Dispatch Mechanism

## What worked
- Placing the syscall dispatch in a separate file (`syscall.go`) kept the change surgically separated from the massive `vm.go` (2900+ lines).
- Using `sync.Once` to lazily initialize the global syscall table avoids init-order issues.
- Reading the syscall number as a single byte (not a 4-byte operand) keeps the encoding compact: `[0xDD, num]` = 2 bytes.

## What didn't match the spec
- **Opcode collision**: The spec proposed `0xD0` for OpSyscall, but `0xD0`-`0xDC` are already occupied by IPC/sync opcodes (OpChannelCreate, OpSend, OpRecv, OpShmCreate, etc.). Used `0xDD` instead. The spec in proposal.md should be updated to reflect this.
- **Step 1.3 (compiler migration)** is a larger change that should be a separate focused task. The compiler's `compileFunctionCall` already handles special cases (spawn, mutate, telemetry, ws.*) — adding syscall-based dispatch requires careful mapping of each builtin name to a syscall number. Doing this without a migration plan would break existing tests.

## What would I do differently
- The stub handlers for VFS, process, IPC, and GPU syscalls could be more useful if they popped the expected number of arguments before returning "not implemented". Currently they return errors immediately, which means the stack isn't cleaned up properly if someone pushes args before calling an unimplemented syscall. This is acceptable for stubs but worth noting.

## Files changed
- `pkg/vm/syscall.go` — new: syscall number constants, dispatch table, handler registration, `execSyscall` method
- `pkg/vm/syscall_test.go` — new: comprehensive tests for opcode, numbers, ENOSYS, truncation, stubs, alloc/free, time, print, exit
- `pkg/vm/vm.go` — added `OpSyscall = 0xDD` opcode constant, dispatch case in `executeInstruction`, `sync.Once` init in `NewVM`
