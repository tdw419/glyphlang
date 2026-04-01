# Tasks: Formal Syscall Interface

## 1. Syscall dispatch mechanism
- [x] 1.1 Add `OpSyscall` (0xDD) to the VM opcode table. When decoded, read the next byte as the syscall number (0x00–0xFF). Look up the syscall handler in the dispatch table. Call it with arguments from the operand stack.
- [x] 1.2 Implement the syscall dispatch table as a function array indexed by syscall number. Register handlers for syscalls 0x00–0x0F. Unregistered entries return ENOSYS.
- [ ] 1.3 Update the Go compiler: change code generation for builtins like `print()`, `len()`, and file operations to emit `[0xDD, syscall_number]` instead of dedicated opcodes. Keep old opcodes working during migration with a deprecation path.

## 2. Capability-based permissions
- [ ] 2.1 Add a capability bitmask (uint16) to the Process struct (from change 008). Define the 7 capability bits (CAP_FS, CAP_PROC, CAP_MEM, CAP_IPC, CAP_IO, CAP_TIME, CAP_GPU).
- [ ] 2.2 Before dispatching a syscall, check the calling process's capability bitmask against the syscall's required capability group. If the bit is not set, trap with EPERM and do not execute the syscall.
- [ ] 2.3 Implement capability inheritance: when `sys_spawn` is called, the child process's capabilities default to a subset of the parent's. The parent can explicitly restrict capabilities via a spawn flag.

## 3. Migration and testing
- [ ] 3.1 Migrate existing ad-hoc builtins one by one: print → sys_print, file ops → sys_read/write/open/close, alloc → sys_alloc. Ensure each migration preserves existing behavior (all 54 passing test packages remain green).
- [ ] 3.2 Add a comprehensive syscall test suite: test each syscall number, test capability rejection, test ENOSYS for unregistered numbers, test that the Go compiler produces correct bytecode for syscall-encoded builtins.
- [ ] 3.3 Update the WGSL shader (vm.wgsl) to recognize `OpSyscall` if GPU-side execution needs to handle syscalls. For now, GPU execution can trap on syscalls since they require host runtime support.
