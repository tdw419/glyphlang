# Formal Syscall Interface

## Why

The GlyphLang VM currently exposes runtime functionality through ad-hoc builtins scattered across the codebase. Print is one opcode. String operations are another. GPU dispatch is yet another path. There is no unified, numbered, capability-controlled interface between user programs and the VM runtime. This is the definition of a syscall layer in a real OS.

Without a formal syscall interface:
- Every new runtime feature requires a new opcode and special-case logic in the VM dispatch loop.
- There is no way to control which operations a process can perform (no security model).
- The opcode space is being consumed by high-level operations that should be syscalls, not instructions.
- The Go compiler and Rust runner have inconsistent ideas about what is a primitive vs. a runtime call.

## What Changes

1. **Numbered syscall range (0xD0–0xDF)**: Reserve a contiguous range in the opcode space for syscall dispatch. `0xD0` is `OpSyscall`. The next byte encodes the syscall number (0x00–0xFF, giving 256 possible syscalls). This packs into a 2-byte sequence: `[0xD0, syscall_number]`.

2. **Syscall dispatch table**: Replace ad-hoc builtins with a numbered dispatch table:
   ```
   0x00 — sys_read(fd, buf, count)    // VFS read
   0x01 — sys_write(fd, buf, count)   // VFS write
   0x02 — sys_open(path, flags)       // VFS open
   0x03 — sys_close(fd)               // VFS close
   0x04 — sys_spawn(bytecode_ptr)     // Process spawn
   0x05 — sys_kill(pid)               // Process kill
   0x06 — sys_wait(pid)               // Process wait
   0x07 — sys_signal(pid, sig)        // Process signal
   0x08 — sys_alloc(size)             // Heap allocate
   0x09 — sys_free(ptr)               // Heap free
   0x0A — sys_send(channel, value)    // IPC send
   0x0B — sys_recv(channel)           // IPC receive
   0x0C — sys_print(value)            // Console output
   0x0D — sys_time()                  // Current timestamp
   0x0E — sys_exit(code)              // Process exit
   0x0F — sys_gpu_dispatch(...)       // GPU kernel dispatch
   ```
   Entries 0x10–0xFF are reserved for future expansion.

3. **Capability-based permissions**: Each process has a capability bitmap. When a process is spawned, the parent specifies which syscall groups the child may invoke. Groups:
   - `CAP_FS` (bit 0) — file operations (0x00–0x03)
   - `CAP_PROC` (bit 1) — process management (0x04–0x07)
   - `CAP_MEM` (bit 2) — heap operations (0x08–0x09)
   - `CAP_IPC` (bit 3) — inter-process communication (0x0A–0x0B)
   - `CAP_IO` (bit 4) — console I/O (0x0C)
   - `CAP_TIME` (bit 5) — time access (0x0D)
   - `CAP_GPU` (bit 6) — GPU dispatch (0x0F)

   A process calling a syscall without the corresponding capability gets a permission-denied trap.

## Impact

- Unifies all runtime interaction through a single dispatch mechanism.
- Reclaims opcode space by consolidating high-level operations into syscall numbers.
- Enables a security model: sandboxed processes can be spawned with restricted capabilities.
- Depends on VFS (007), process model (008), and heap memory (010) since syscalls delegate to those subsystems.
- The Go compiler's code generation changes: builtin calls emit `OpSyscall` + syscall number instead of dedicated opcodes.
