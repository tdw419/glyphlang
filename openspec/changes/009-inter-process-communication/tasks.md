# Tasks: Inter-Process Communication

## 1. Channel-based messaging
- [x] 1.1 Implement channel data structure: a typed circular buffer with sender/receiver wait queues. Channel IDs are allocated from a global namespace.
- [x] 1.2 Add `OpSend` opcode: serialize the value into the channel buffer. If buffer is full, block the sending process (set state to blocked, add to channel's sender wait queue). If buffer has space, enqueue and wake any blocked receiver.
- [x] 1.3 Add `OpRecv` opcode: dequeue a value from the channel buffer. If buffer is empty, block the receiving process. If buffer has data, dequeue and wake any blocked sender. Place the received value on the operand stack.

## 2. Shared memory regions
- [x] 2.1 Implement `OpShmCreate(size)`: allocate a shared memory region of the given size. Return a handle (integer ID). Register the region with the creator's PID as owner.
- [x] 2.2 Implement `OpShmMap(handle)`: validate that the calling process has access to the region. Map the shared bytes into the process's address space at a reserved heap offset. Return a pointer value.
- [x] 2.3 Implement access control: track which PIDs have access to each shared region. Support explicit grant/revoke of access between processes.

## 3. Synchronization primitives
- [x] 3.1 Implement mutex: `OpMutexCreate` / `OpMutexLock` / `OpMutexUnlock`. Lock blocks if already held; unlock wakes one waiting process.
- [x] 3.2 Implement semaphore: `OpSemCreate(n)` / `OpSemWait` / `OpSemSignal`. Wait decrements (blocks at zero), signal increments (wakes one waiter).
- [x] 3.3 Wire synchronization state into the process scheduler: when a process is unblocked (lock released, semaphore signaled), move it from blocked to ready state.
