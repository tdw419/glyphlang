package vm

import (
	"testing"
)

// --- Mutex Data Structure Tests ---

func TestMutexLockUnlocked(t *testing.T) {
	m := newMutex(1)

	acquired, _, err := m.Lock(10)
	if err != nil {
		t.Fatalf("Lock error: %v", err)
	}
	if !acquired {
		t.Error("expected lock to succeed on unlocked mutex")
	}
}

func TestMutexLockBlocksIfHeld(t *testing.T) {
	m := newMutex(1)

	// PID 10 acquires first
	m.Lock(10)

	// PID 20 tries to lock — should block
	acquired, _, err := m.Lock(20)
	if err != nil {
		t.Fatalf("Lock error: %v", err)
	}
	if acquired {
		t.Error("expected lock to fail on held mutex")
	}

	if len(m.queue) != 1 {
		t.Fatalf("expected 1 waiter, got %d", len(m.queue))
	}
	if m.queue[0] != 20 {
		t.Errorf("expected waiter PID 20, got %d", m.queue[0])
	}
}

func TestMutexLockDeadlockDetection(t *testing.T) {
	m := newMutex(1)
	m.Lock(10)

	_, _, err := m.Lock(10)
	if err == nil {
		t.Error("expected deadlock error when same PID locks twice")
	}
}

func TestMutexUnlockWakesWaiter(t *testing.T) {
	m := newMutex(1)

	m.Lock(10)   // PID 10 holds
	m.Lock(20)   // PID 20 waits

	wokenPID, err := m.Unlock(10)
	if err != nil {
		t.Fatalf("Unlock error: %v", err)
	}
	if wokenPID != 20 {
		t.Errorf("expected woken PID 20, got %d", wokenPID)
	}

	// Mutex should now be owned by PID 20
	if m.owner != 20 {
		t.Errorf("expected owner PID 20, got %d", m.owner)
	}
}

func TestMutexUnlockNoWaiter(t *testing.T) {
	m := newMutex(1)
	m.Lock(10)

	wokenPID, err := m.Unlock(10)
	if err != nil {
		t.Fatalf("Unlock error: %v", err)
	}
	if wokenPID != 0 {
		t.Errorf("expected no woken PID, got %d", wokenPID)
	}
	if m.owner != 0 {
		t.Errorf("expected mutex to be unlocked (owner=0), got owner=%d", m.owner)
	}
}

func TestMutexUnlockWrongOwner(t *testing.T) {
	m := newMutex(1)
	m.Lock(10)

	_, err := m.Unlock(99)
	if err == nil {
		t.Error("expected error when non-owner unlocks")
	}
}

func TestMutexFIFOOrder(t *testing.T) {
	m := newMutex(1)
	m.Lock(10)   // owner
	m.Lock(20)   // first waiter
	m.Lock(30)   // second waiter

	woken, _ := m.Unlock(10)
	if woken != 20 {
		t.Errorf("expected first waiter (PID 20) to be woken, got %d", woken)
	}

	woken, _ = m.Unlock(20)
	if woken != 30 {
		t.Errorf("expected second waiter (PID 30) to be woken, got %d", woken)
	}
}

// --- Semaphore Data Structure Tests ---

func TestSemaphoreWaitDecrements(t *testing.T) {
	s := newSemaphore(1, 3)

	acquired, _, err := s.Wait(10)
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if !acquired {
		t.Error("expected wait to succeed when count > 0")
	}
	if s.count != 2 {
		t.Errorf("expected count 2, got %d", s.count)
	}
}

func TestSemaphoreWaitBlocksAtZero(t *testing.T) {
	s := newSemaphore(1, 0)

	acquired, _, err := s.Wait(10)
	if err != nil {
		t.Fatalf("Wait error: %v", err)
	}
	if acquired {
		t.Error("expected wait to block when count == 0")
	}
	if len(s.queue) != 1 {
		t.Fatalf("expected 1 waiter, got %d", len(s.queue))
	}
}

func TestSemaphoreSignalIncrements(t *testing.T) {
	s := newSemaphore(1, 0)

	wokenPID, err := s.Signal()
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}
	if wokenPID != 0 {
		t.Errorf("expected no woken PID (no waiters), got %d", wokenPID)
	}
	if s.count != 1 {
		t.Errorf("expected count 1, got %d", s.count)
	}
}

func TestSemaphoreSignalWakesWaiter(t *testing.T) {
	s := newSemaphore(1, 1)

	s.Wait(10) // count -> 0
	s.Wait(20) // blocks, PID 20 enqueued

	wokenPID, err := s.Signal()
	if err != nil {
		t.Fatalf("Signal error: %v", err)
	}
	if wokenPID != 20 {
		t.Errorf("expected woken PID 20, got %d", wokenPID)
	}
	// Count should stay at 0 (permit transferred to waiter)
	if s.count != 0 {
		t.Errorf("expected count to stay 0 after waking waiter, got %d", s.count)
	}
}

func TestSemaphoreMultipleWaiters(t *testing.T) {
	s := newSemaphore(1, 0)

	s.Wait(10) // blocked
	s.Wait(20) // blocked
	s.Wait(30) // blocked

	woken, _ := s.Signal()
	if woken != 10 {
		t.Errorf("expected first waiter (PID 10), got %d", woken)
	}

	woken, _ = s.Signal()
	if woken != 20 {
		t.Errorf("expected second waiter (PID 20), got %d", woken)
	}

	woken, _ = s.Signal()
	if woken != 30 {
		t.Errorf("expected third waiter (PID 30), got %d", woken)
	}
}

// --- SyncTable Tests ---

func TestSyncTableCreateMutex(t *testing.T) {
	st := NewSyncTable()

	m := st.CreateMutex()
	if m == nil {
		t.Fatal("CreateMutex returned nil")
	}
	if m.ID != 1 {
		t.Errorf("expected mutex ID 1, got %d", m.ID)
	}

	m2 := st.CreateMutex()
	if m2.ID != 2 {
		t.Errorf("expected mutex ID 2, got %d", m2.ID)
	}
}

func TestSyncTableGetMutex(t *testing.T) {
	st := NewSyncTable()
	st.CreateMutex()

	m, ok := st.GetMutex(1)
	if !ok {
		t.Fatal("expected to find mutex 1")
	}
	if m.ID != 1 {
		t.Errorf("expected mutex ID 1, got %d", m.ID)
	}

	_, ok = st.GetMutex(999)
	if ok {
		t.Error("expected GetMutex(999) to return false")
	}
}

func TestSyncTableCreateSemaphore(t *testing.T) {
	st := NewSyncTable()

	s := st.CreateSemaphore(5)
	if s == nil {
		t.Fatal("CreateSemaphore returned nil")
	}
	if s.ID != 1 {
		t.Errorf("expected semaphore ID 1, got %d", s.ID)
	}
	if s.count != 5 {
		t.Errorf("expected count 5, got %d", s.count)
	}
}

func TestSyncTableGetSemaphore(t *testing.T) {
	st := NewSyncTable()
	st.CreateSemaphore(3)

	s, ok := st.GetSemaphore(1)
	if !ok {
		t.Fatal("expected to find semaphore 1")
	}
	if s.count != 3 {
		t.Errorf("expected count 3, got %d", s.count)
	}

	_, ok = st.GetSemaphore(999)
	if ok {
		t.Error("expected GetSemaphore(999) to return false")
	}
}

// --- OpMutexCreate Tests ---

func TestOpMutexCreate(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	err := testVM.executeInstruction(OpMutexCreate)
	if err != nil {
		t.Fatalf("executeInstruction(OpMutexCreate) error: %v", err)
	}

	mutexID := mustPopInt(t, testVM)
	if mutexID != 1 {
		t.Errorf("expected mutex ID 1, got %d", mutexID)
	}

	// Mutex should exist in sync table
	m, ok := pt.syncTable.GetMutex(1)
	if !ok {
		t.Fatal("expected mutex 1 in sync table")
	}
	if m.ID != 1 {
		t.Errorf("expected mutex ID 1, got %d", m.ID)
	}
}

func TestOpMutexCreateNoProcessTable(t *testing.T) {
	testVM := NewVM()

	err := testVM.executeInstruction(OpMutexCreate)
	if err == nil {
		t.Fatal("expected error without process table")
	}
}

// --- OpMutexLock Tests ---

func TestOpMutexLockAcquireUnlocked(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()
	pt.syncTable.CreateMutex() // mutex ID 1

	testVM.Push(IntValue{Val: 1}) // mutex ID

	err := testVM.executeInstruction(OpMutexLock)
	if err != nil {
		t.Fatalf("OpMutexLock error: %v", err)
	}

	// Process should still be running
	proc, _ := pt.Get(1)
	if proc.State != ProcessRunning {
		t.Errorf("expected process to remain running, got %v", proc.State)
	}
}

func TestOpMutexLockBlocksIfHeld(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 2

	proc1 := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: NewVM()}
	proc2 := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: testVM}
	pt.Register(proc1)
	pt.Register(proc2)

	pt.syncTable = NewSyncTable()
	m := pt.syncTable.CreateMutex() // mutex ID 1
	m.Lock(1)                       // PID 1 holds it

	testVM.Push(IntValue{Val: 1}) // mutex ID

	err := testVM.executeInstruction(OpMutexLock)
	if err != nil {
		t.Fatalf("OpMutexLock error: %v", err)
	}

	// Process 2 should be blocked
	proc, _ := pt.Get(2)
	if proc.State != ProcessBlocked {
		t.Errorf("expected process to be blocked, got %v", proc.State)
	}
}

func TestOpMutexLockInvalidID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpMutexLock)
	if err == nil {
		t.Fatal("expected error for invalid mutex ID")
	}
}

// --- OpMutexUnlock Tests ---

func TestOpMutexUnlockWakesWaiter(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	proc1 := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	proc2 := &Process{PID: 2, PPID: 1, State: ProcessBlocked, VM: NewVM()}
	pt.Register(proc1)
	pt.Register(proc2)

	pt.syncTable = NewSyncTable()
	m := pt.syncTable.CreateMutex() // mutex ID 1
	m.Lock(1)                       // PID 1 holds
	m.Lock(2)                       // PID 2 waits

	testVM.Push(IntValue{Val: 1}) // mutex ID

	err := testVM.executeInstruction(OpMutexUnlock)
	if err != nil {
		t.Fatalf("OpMutexUnlock error: %v", err)
	}

	// PID 2 should be woken (moved to Ready)
	proc, _ := pt.Get(2)
	if proc.State != ProcessReady {
		t.Errorf("expected PID 2 to be ready, got %v", proc.State)
	}
}

func TestOpMutexUnlockNoWaiter(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()
	m := pt.syncTable.CreateMutex()
	m.Lock(1)

	testVM.Push(IntValue{Val: 1})

	err := testVM.executeInstruction(OpMutexUnlock)
	if err != nil {
		t.Fatalf("OpMutexUnlock error: %v", err)
	}

	// Mutex should be unlocked
	got, _ := pt.syncTable.GetMutex(1)
	if got.owner != 0 {
		t.Errorf("expected mutex to be unlocked, owner=%d", got.owner)
	}
}

func TestOpMutexUnlockWrongOwner(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 2

	proc1 := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: NewVM()}
	proc2 := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: testVM}
	pt.Register(proc1)
	pt.Register(proc2)

	pt.syncTable = NewSyncTable()
	m := pt.syncTable.CreateMutex()
	m.Lock(1) // PID 1 holds

	testVM.Push(IntValue{Val: 1})

	err := testVM.executeInstruction(OpMutexUnlock)
	if err == nil {
		t.Fatal("expected error when non-owner unlocks")
	}
}

// --- OpSemCreate Tests ---

func TestOpSemCreate(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	// Set up operand: initial count = 3
	opBytes := make([]byte, 4)
	opBytes[0] = 3 // little-endian uint32 = 3
	testVM.code = opBytes
	testVM.pc = 0

	err := testVM.executeInstruction(OpSemCreate)
	if err != nil {
		t.Fatalf("OpSemCreate error: %v", err)
	}

	semID := mustPopInt(t, testVM)
	if semID != 1 {
		t.Errorf("expected semaphore ID 1, got %d", semID)
	}

	s, ok := pt.syncTable.GetSemaphore(1)
	if !ok {
		t.Fatal("expected semaphore 1 in sync table")
	}
	if s.count != 3 {
		t.Errorf("expected count 3, got %d", s.count)
	}
}

func TestOpSemCreateNoProcessTable(t *testing.T) {
	testVM := NewVM()
	testVM.code = make([]byte, 4)
	testVM.pc = 0

	err := testVM.executeInstruction(OpSemCreate)
	if err == nil {
		t.Fatal("expected error without process table")
	}
}

// --- OpSemWait Tests ---

func TestOpSemWaitDecrements(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()
	pt.syncTable.CreateSemaphore(3) // sem ID 1, count=3

	testVM.Push(IntValue{Val: 1}) // sem ID

	err := testVM.executeInstruction(OpSemWait)
	if err != nil {
		t.Fatalf("OpSemWait error: %v", err)
	}

	proc, _ := pt.Get(1)
	if proc.State != ProcessRunning {
		t.Errorf("expected process to remain running, got %v", proc.State)
	}

	s, _ := pt.syncTable.GetSemaphore(1)
	if s.count != 2 {
		t.Errorf("expected count 2 after wait, got %d", s.count)
	}
}

func TestOpSemWaitBlocksAtZero(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()
	pt.syncTable.CreateSemaphore(0) // sem ID 1, count=0

	testVM.Push(IntValue{Val: 1})

	err := testVM.executeInstruction(OpSemWait)
	if err != nil {
		t.Fatalf("OpSemWait error: %v", err)
	}

	proc, _ := pt.Get(1)
	if proc.State != ProcessBlocked {
		t.Errorf("expected process to be blocked, got %v", proc.State)
	}
}

func TestOpSemWaitInvalidID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpSemWait)
	if err == nil {
		t.Fatal("expected error for invalid semaphore ID")
	}
}

// --- OpSemSignal Tests ---

func TestOpSemSignalIncrements(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()
	pt.syncTable.CreateSemaphore(0) // sem ID 1, count=0

	testVM.Push(IntValue{Val: 1})

	err := testVM.executeInstruction(OpSemSignal)
	if err != nil {
		t.Fatalf("OpSemSignal error: %v", err)
	}

	s, _ := pt.syncTable.GetSemaphore(1)
	if s.count != 1 {
		t.Errorf("expected count 1 after signal, got %d", s.count)
	}
}

func TestOpSemSignalWakesWaiter(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 2

	proc1 := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: NewVM()}
	proc2 := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: testVM}
	pt.Register(proc1)
	pt.Register(proc2)

	pt.syncTable = NewSyncTable()
	s := pt.syncTable.CreateSemaphore(1) // sem ID 1, count=1
	s.Wait(1)                            // PID 1 decrements to 0
	s.Wait(1)                            // PID 1 blocks (but wait, same PID twice...)
	// Let me redo: PID 1 waits once (count -> 0), PID 1 tries again -> blocks
	// Actually we need different PIDs. Let me set it up properly.

	// Reset and redo
	pt2 := NewProcessTable()
	testVM.processTable = pt2
	testVM.pid = 2

	p1 := &Process{PID: 1, PPID: 0, State: ProcessBlocked, VM: NewVM()}
	p2 := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: testVM}
	pt2.Register(p1)
	pt2.Register(p2)

	pt2.syncTable = NewSyncTable()
	sem := pt2.syncTable.CreateSemaphore(0) // sem ID 1, count=0
	sem.Wait(1)                             // PID 1 blocks, enqueued

	testVM.Push(IntValue{Val: 1})

	err := testVM.executeInstruction(OpSemSignal)
	if err != nil {
		t.Fatalf("OpSemSignal error: %v", err)
	}

	// PID 1 should be woken
	proc, _ := pt2.Get(1)
	if proc.State != ProcessReady {
		t.Errorf("expected PID 1 to be ready, got %v", proc.State)
	}
}

func TestOpSemSignalInvalidID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.syncTable = NewSyncTable()

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpSemSignal)
	if err == nil {
		t.Fatal("expected error for invalid semaphore ID")
	}
}

// --- Integration: Mutex Lock/Unlock with scheduler state ---

func TestMutexLockUnlockIntegration(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	proc1 := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	proc2 := &Process{PID: 2, PPID: 1, State: ProcessBlocked, VM: NewVM()}
	pt.Register(proc1)
	pt.Register(proc2)

	// Create mutex
	pt.syncTable = NewSyncTable()
	m := pt.syncTable.CreateMutex()
	m.Lock(1)  // PID 1 holds
	m.Lock(2)  // PID 2 waits (blocked)

	// PID 1 unlocks
	testVM.Push(IntValue{Val: int64(m.ID)})
	err := testVM.executeInstruction(OpMutexUnlock)
	if err != nil {
		t.Fatalf("OpMutexUnlock error: %v", err)
	}

	// PID 2 should be Ready now
	p2, _ := pt.Get(2)
	if p2.State != ProcessReady {
		t.Errorf("expected PID 2 to be Ready after unlock, got %v", p2.State)
	}

	// Mutex should be owned by PID 2
	got, _ := pt.syncTable.GetMutex(m.ID)
	if got.owner != 2 {
		t.Errorf("expected mutex owner to be PID 2, got %d", got.owner)
	}
}

// --- Integration: Semaphore Wait/Signal with scheduler state ---

func TestSemaphoreWaitSignalIntegration(t *testing.T) {
	pt := NewProcessTable()
	pt.syncTable = NewSyncTable()

	s := pt.syncTable.CreateSemaphore(2) // count=2

	// Two processes wait (should succeed)
	s.Wait(1) // count -> 1
	s.Wait(2) // count -> 0

	// Third process blocks
	acquired, _, _ := s.Wait(3)
	if acquired {
		t.Error("expected third wait to block")
	}

	// Register processes in table
	proc3 := &Process{PID: 3, PPID: 0, State: ProcessBlocked, VM: NewVM()}
	pt.Register(proc3)

	// Signal from PID 1
	wokenPID, _ := s.Signal()
	if wokenPID != 3 {
		t.Errorf("expected PID 3 to be woken, got %d", wokenPID)
	}
}

// --- Edge case: multiple mutex creates increment IDs ---

func TestMultipleMutexCreates(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	err := testVM.executeInstruction(OpMutexCreate)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}
	id1 := mustPopInt(t, testVM)

	err = testVM.executeInstruction(OpMutexCreate)
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}
	id2 := mustPopInt(t, testVM)

	if id1 != 1 {
		t.Errorf("expected first mutex ID 1, got %d", id1)
	}
	if id2 != 2 {
		t.Errorf("expected second mutex ID 2, got %d", id2)
	}
}

// --- Edge case: no sync table when lock/unlock is called ---

func TestOpMutexLockNoSyncTable(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)
	// No syncTable initialized

	testVM.Push(IntValue{Val: 1})
	err := testVM.executeInstruction(OpMutexLock)
	if err == nil {
		t.Fatal("expected error when no sync table")
	}
}

func TestOpSemWaitNoSyncTable(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	testVM.Push(IntValue{Val: 1})
	err := testVM.executeInstruction(OpSemWait)
	if err == nil {
		t.Fatal("expected error when no sync table")
	}
}
