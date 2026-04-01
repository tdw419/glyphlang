package vm

import (
	"fmt"
	"sync"
)

// MutexID is a unique identifier for a mutex.
type MutexID uint32

// SemaphoreID is a unique identifier for a semaphore.
type SemaphoreID uint32

// Mutex is a mutual exclusion lock for IPC synchronization.
// Only one process can hold the lock at a time. Other processes
// that attempt to lock are placed in a wait queue and block until
// the lock is released.
type Mutex struct {
	ID    MutexID
	owner uint32     // PID of the current lock holder (0 = unlocked)
	mu    sync.Mutex // protects mutable state
	queue []uint32   // PIDs waiting to acquire the lock
}

// newMutex creates an unlocked mutex with the given ID.
func newMutex(id MutexID) *Mutex {
	return &Mutex{ID: id}
}

// Lock attempts to acquire the mutex. Returns (acquired, wokenPID).
// If acquired=true, the caller now holds the lock.
// If acquired=false, the caller has been enqueued and should block.
// wokenPID is 0 unless this lock operation caused another process to be
// implicitly affected (currently always 0 for Lock).
func (m *Mutex) Lock(pid uint32) (acquired bool, wokenPID uint32, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.owner == pid {
		return false, 0, fmt.Errorf("mutex_lock: deadlock - pid %d already holds mutex %d", pid, m.ID)
	}

	if m.owner == 0 {
		// Unlocked: acquire it
		m.owner = pid
		return true, 0, nil
	}

	// Locked by someone else: enqueue and block
	m.queue = append(m.queue, pid)
	return false, 0, nil
}

// Unlock releases the mutex. Returns wokenPID — the PID of one
// waiting process that has been granted ownership (FIFO), or 0
// if no one was waiting.
func (m *Mutex) Unlock(pid uint32) (wokenPID uint32, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.owner != pid {
		return 0, fmt.Errorf("mutex_unlock: pid %d does not hold mutex %d (owner: %d)", pid, m.ID, m.owner)
	}

	if len(m.queue) == 0 {
		// No waiters: just release
		m.owner = 0
		return 0, nil
	}

	// Transfer ownership to first waiter (FIFO)
	wokenPID = m.queue[0]
	m.queue = m.queue[1:]
	m.owner = wokenPID
	return wokenPID, nil
}

// Semaphore is a counting semaphore for IPC synchronization.
// Wait decrements the count and blocks if it reaches zero.
// Signal increments the count and wakes one waiter.
type Semaphore struct {
	ID    SemaphoreID
	count int        // current count (available permits)
	mu    sync.Mutex // protects mutable state
	queue []uint32   // PIDs waiting for a permit
}

// newSemaphore creates a semaphore with the given ID and initial count.
func newSemaphore(id SemaphoreID, initialCount int) *Semaphore {
	return &Semaphore{ID: id, count: initialCount}
}

// Wait decrements the semaphore count. If count > 0, decrements and returns
// immediately. If count == 0, blocks the caller (enqueues it).
// Returns (acquired, wokenPID).
func (s *Semaphore) Wait(pid uint32) (acquired bool, wokenPID uint32, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.count > 0 {
		s.count--
		return true, 0, nil
	}

	// No permits available: enqueue and block
	s.queue = append(s.queue, pid)
	return false, 0, nil
}

// Signal increments the semaphore count. If there are waiters,
// one is woken (granted a permit directly, count stays the same).
// Returns wokenPID (0 if no one was waiting).
func (s *Semaphore) Signal() (wokenPID uint32, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queue) > 0 {
		// Transfer a permit directly to the first waiter
		wokenPID = s.queue[0]
		s.queue = s.queue[1:]
		// count stays the same: we consumed the increment by giving
		// it to the waiter
		return wokenPID, nil
	}

	// No waiters: just increment
	s.count++
	return 0, nil
}

// SyncTable is a concurrent-safe global namespace for mutexes and semaphores.
type SyncTable struct {
	mu        sync.RWMutex
	mutexes   map[MutexID]*Mutex
	semaphores map[SemaphoreID]*Semaphore
	nextMutexID MutexID
	nextSemID   SemaphoreID
}

// NewSyncTable creates an empty sync table.
func NewSyncTable() *SyncTable {
	return &SyncTable{
		mutexes:     make(map[MutexID]*Mutex),
		semaphores:  make(map[SemaphoreID]*Semaphore),
		nextMutexID: 1,
		nextSemID:   1,
	}
}

// CreateMutex allocates and registers a new mutex.
func (st *SyncTable) CreateMutex() *Mutex {
	st.mu.Lock()
	defer st.mu.Unlock()

	id := st.nextMutexID
	st.nextMutexID++

	m := newMutex(id)
	st.mutexes[id] = m
	return m
}

// GetMutex retrieves a mutex by ID.
func (st *SyncTable) GetMutex(id MutexID) (*Mutex, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	m, ok := st.mutexes[id]
	return m, ok
}

// CreateSemaphore allocates and registers a new semaphore with the given initial count.
func (st *SyncTable) CreateSemaphore(initialCount int) *Semaphore {
	st.mu.Lock()
	defer st.mu.Unlock()

	id := st.nextSemID
	st.nextSemID++

	s := newSemaphore(id, initialCount)
	st.semaphores[id] = s
	return s
}

// GetSemaphore retrieves a semaphore by ID.
func (st *SyncTable) GetSemaphore(id SemaphoreID) (*Semaphore, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	s, ok := st.semaphores[id]
	return s, ok
}

// --- Synchronization Opcodes ---

// OpMutexCreate (0xD7): create a mutex. Pushes the mutex ID (IntValue).
const OpMutexCreate Opcode = 0xD7

// OpMutexLock (0xD8): acquire a mutex.
// Stack: [mutex_id] -> pops mutex_id. Blocks if already held.
const OpMutexLock Opcode = 0xD8

// OpMutexUnlock (0xD9): release a mutex. Wakes one waiter.
// Stack: [mutex_id] -> pops mutex_id.
const OpMutexUnlock Opcode = 0xD9

// OpSemCreate (0xDA): create a semaphore with initial count.
// Operand: initial_count (uint32). Pushes the semaphore ID (IntValue).
const OpSemCreate Opcode = 0xDA

// OpSemWait (0xDB): wait (decrement) on a semaphore. Blocks at zero.
// Stack: [sem_id] -> pops sem_id.
const OpSemWait Opcode = 0xDB

// OpSemSignal (0xDC): signal (increment) a semaphore. Wakes one waiter.
// Stack: [sem_id] -> pops sem_id.
const OpSemSignal Opcode = 0xDC

// execMutexCreate creates a new mutex.
func (vm *VM) execMutexCreate() error {
	if vm.processTable == nil {
		return fmt.Errorf("mutex_create: no process table attached to VM")
	}

	// Lazily initialize sync table
	if vm.processTable.syncTable == nil {
		vm.processTable.syncTable = NewSyncTable()
	}

	m := vm.processTable.syncTable.CreateMutex()
	vm.Push(IntValue{Val: int64(m.ID)})
	return nil
}

// execMutexLock acquires a mutex. Blocks if already held.
func (vm *VM) execMutexLock() error {
	if vm.processTable == nil {
		return fmt.Errorf("mutex_lock: no process table attached to VM")
	}

	mutexIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	mutexIDInt, ok := mutexIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("mutex_lock: mutex ID must be an integer")
	}
	mutexID := MutexID(mutexIDInt.Val)

	if vm.processTable.syncTable == nil {
		return fmt.Errorf("mutex_lock: no sync table")
	}

	m, exists := vm.processTable.syncTable.GetMutex(mutexID)
	if !exists {
		return fmt.Errorf("mutex_lock: mutex %d not found", mutexID)
	}

	acquired, _, err := m.Lock(vm.pid)
	if err != nil {
		return err
	}

	if !acquired {
		// Block the calling process
		if proc, ok := vm.processTable.Get(vm.pid); ok {
			proc.State = ProcessBlocked
		}
	}

	return nil
}

// execMutexUnlock releases a mutex. Wakes one waiter.
func (vm *VM) execMutexUnlock() error {
	if vm.processTable == nil {
		return fmt.Errorf("mutex_unlock: no process table attached to VM")
	}

	mutexIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	mutexIDInt, ok := mutexIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("mutex_unlock: mutex ID must be an integer")
	}
	mutexID := MutexID(mutexIDInt.Val)

	if vm.processTable.syncTable == nil {
		return fmt.Errorf("mutex_unlock: no sync table")
	}

	m, exists := vm.processTable.syncTable.GetMutex(mutexID)
	if !exists {
		return fmt.Errorf("mutex_unlock: mutex %d not found", mutexID)
	}

	wokenPID, err := m.Unlock(vm.pid)
	if err != nil {
		return err
	}

	// Wake the blocked process (move from blocked to ready)
	if wokenPID != 0 {
		if proc, ok := vm.processTable.Get(wokenPID); ok {
			proc.State = ProcessReady
		}
	}

	return nil
}

// execSemCreate creates a semaphore with the given initial count.
func (vm *VM) execSemCreate() error {
	if vm.processTable == nil {
		return fmt.Errorf("sem_create: no process table attached to VM")
	}

	count, err := vm.readOperand()
	if err != nil {
		return err
	}

	// Lazily initialize sync table
	if vm.processTable.syncTable == nil {
		vm.processTable.syncTable = NewSyncTable()
	}

	s := vm.processTable.syncTable.CreateSemaphore(int(count))
	vm.Push(IntValue{Val: int64(s.ID)})
	return nil
}

// execSemWait decrements a semaphore. Blocks at zero.
func (vm *VM) execSemWait() error {
	if vm.processTable == nil {
		return fmt.Errorf("sem_wait: no process table attached to VM")
	}

	semIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	semIDInt, ok := semIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("sem_wait: semaphore ID must be an integer")
	}
	semID := SemaphoreID(semIDInt.Val)

	if vm.processTable.syncTable == nil {
		return fmt.Errorf("sem_wait: no sync table")
	}

	s, exists := vm.processTable.syncTable.GetSemaphore(semID)
	if !exists {
		return fmt.Errorf("sem_wait: semaphore %d not found", semID)
	}

	acquired, _, err := s.Wait(vm.pid)
	if err != nil {
		return err
	}

	if !acquired {
		// Block the calling process
		if proc, ok := vm.processTable.Get(vm.pid); ok {
			proc.State = ProcessBlocked
		}
	}

	return nil
}

// execSemSignal increments a semaphore. Wakes one waiter.
func (vm *VM) execSemSignal() error {
	if vm.processTable == nil {
		return fmt.Errorf("sem_signal: no process table attached to VM")
	}

	semIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	semIDInt, ok := semIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("sem_signal: semaphore ID must be an integer")
	}
	semID := SemaphoreID(semIDInt.Val)

	if vm.processTable.syncTable == nil {
		return fmt.Errorf("sem_signal: no sync table")
	}

	s, exists := vm.processTable.syncTable.GetSemaphore(semID)
	if !exists {
		return fmt.Errorf("sem_signal: semaphore %d not found", semID)
	}

	wokenPID, err := s.Signal()
	if err != nil {
		return err
	}

	// Wake the blocked process
	if wokenPID != 0 {
		if proc, ok := vm.processTable.Get(wokenPID); ok {
			proc.State = ProcessReady
		}
	}

	return nil
}
