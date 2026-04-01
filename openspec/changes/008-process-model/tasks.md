# Tasks: Formal Process Abstraction

## 1. Process table and PID assignment
- [x] 1.1 Define the Process struct: PID (uint32), parent PID, state (ready/running/blocked/zombie), exit code, register file, stack pointer, heap base, fd table, spawn timestamp.
- [x] 1.2 Implement the global process table as a concurrent-safe map protected by a mutex. Add PID allocation (incrementing counter starting at 1) and reclamation for zombie processes.
- [x] 1.3 Wire the process table into the VM runner: when a VM starts, it registers as PID 1. When Mitosis spawns instances, each gets a unique PID from the table.

## 2. Lifecycle opcodes
- [x] 2.1 Implement `OpSpawn`: fork or create a new VM instance with a fresh register file and stack. Copy the bytecode. Assign a PID. Set parent PID. Place the child PID on the parent's stack as the return value. Place 0 on the child's stack.
- [x] 2.2 Implement `OpKill`: look up the target PID, set its state to zombie, release its file descriptors and heap memory, reparent its children to PID 1.
- [x] 2.3 Implement `OpWait`: block the calling process until the target process enters zombie state. Read the exit code from the zombie's process struct. Clean up the zombie entry.

## 3. Process isolation and hierarchy
- [x] 3.1 Ensure each spawned VM instance has a fully independent register file, stack, and heap region. No memory region overlaps between processes.
- [x] 3.2 Implement parent-child tracking: each process records its parent PID and a list of child PIDs. On child exit, notify the parent (wake from OpWait if blocked).
- [x] 3.3 Implement reparenting: when a process dies, move all its children to PID 1's child list. PID 1's OpWait loop adopts and reaps these orphans.
