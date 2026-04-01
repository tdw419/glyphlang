package vm

import (
	"testing"
)

// mustPop is a test helper that pops and type-asserts an IntValue.
func mustPopInt(t *testing.T, vm *VM) int64 {
	t.Helper()
	val, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}
	iv, ok := val.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	return iv.Val
}

// --- Process Table Tests (SEC-1 prerequisite) ---

func TestNewProcessTable(t *testing.T) {
	pt := NewProcessTable()
	if pt == nil {
		t.Fatal("NewProcessTable() returned nil")
	}
	if pt.nextPID != 1 {
		t.Errorf("expected nextPID to start at 1, got %d", pt.nextPID)
	}
	if len(pt.processes) != 0 {
		t.Errorf("expected empty process table, got %d entries", len(pt.processes))
	}
}

func TestProcessTableAllocatePID(t *testing.T) {
	pt := NewProcessTable()

	pid1 := pt.AllocatePID()
	if pid1 != 1 {
		t.Errorf("expected first PID to be 1, got %d", pid1)
	}

	pid2 := pt.AllocatePID()
	if pid2 != 2 {
		t.Errorf("expected second PID to be 2, got %d", pid2)
	}
}

func TestProcessTableRegisterAndGet(t *testing.T) {
	pt := NewProcessTable()

	proc := &Process{
		PID:   1,
		PPID:  0,
		State: ProcessReady,
	}

	pt.Register(proc)

	got, ok := pt.Get(1)
	if !ok {
		t.Fatal("expected to find process with PID 1")
	}
	if got.PID != 1 {
		t.Errorf("expected PID 1, got %d", got.PID)
	}
	if got.State != ProcessReady {
		t.Errorf("expected state ProcessReady, got %v", got.State)
	}
}

func TestProcessTableGetNonExistent(t *testing.T) {
	pt := NewProcessTable()

	_, ok := pt.Get(999)
	if ok {
		t.Error("expected Get(999) to return false for non-existent PID")
	}
}

func TestProcessTableRemove(t *testing.T) {
	pt := NewProcessTable()

	proc := &Process{PID: 1, State: ProcessZombie}
	pt.Register(proc)

	pt.Remove(1)

	_, ok := pt.Get(1)
	if ok {
		t.Error("expected process to be removed")
	}
}

func TestProcessTableChildren(t *testing.T) {
	pt := NewProcessTable()

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning}
	child := &Process{PID: 2, PPID: 1, State: ProcessReady}

	pt.Register(parent)
	pt.Register(child)

	children := pt.Children(1)
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].PID != 2 {
		t.Errorf("expected child PID 2, got %d", children[0].PID)
	}
}

func TestProcessTableReparent(t *testing.T) {
	pt := NewProcessTable()

	// PID 1 = init
	init := &Process{PID: 1, PPID: 0, State: ProcessRunning}
	// PID 5 = intermediate parent
	parent := &Process{PID: 5, PPID: 1, State: ProcessRunning}
	// PID 10, 11 = children of PID 5
	child1 := &Process{PID: 10, PPID: 5, State: ProcessReady}
	child2 := &Process{PID: 11, PPID: 5, State: ProcessReady}

	pt.Register(init)
	pt.Register(parent)
	pt.Register(child1)
	pt.Register(child2)

	// Reparent children of PID 5 to PID 1
	pt.Reparent(5, 1)

	// Children of PID 5 should now belong to PID 1
	children := pt.Children(1)
	// PID 5 itself is already a child of PID 1, plus PID 10 and 11 are reparented
	if len(children) != 3 {
		t.Fatalf("expected 3 children after reparent (PID 5, 10, 11), got %d", len(children))
	}

	// Only PID 10 and 11 should have updated PPID
	reparented := map[uint32]bool{10: false, 11: false}
	for _, c := range children {
		if _, ok := reparented[c.PID]; ok {
			if c.PPID != 1 {
				t.Errorf("expected reparented child PID %d to have PPID 1, got %d", c.PID, c.PPID)
			}
			reparented[c.PID] = true
		}
	}
	for pid, found := range reparented {
		if !found {
			t.Errorf("expected PID %d to be found in children of PID 1", pid)
		}
	}
}

// --- OpSpawn Tests (SEC-2 step 2.1) ---

func TestOpSpawnCreatesChildProcess(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt

	// Register the parent as PID 1
	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)
	testVM.pid = 1

	// Set up bytecode for parent: OpSpawn then Halt
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	testVM.code = bytecode
	layout, err := parseBytecodeLayout(bytecode)
	if err != nil {
		t.Fatalf("parseBytecodeLayout error: %v", err)
	}
	testVM.pc = int(layout.CodeOffset)

	// Execute spawn
	err = testVM.executeInstruction(OpSpawn)
	if err != nil {
		t.Fatalf("OpSpawn error: %v", err)
	}

	// Parent should have child PID on stack
	childPID := mustPopInt(t, testVM)
	if childPID != 2 {
		t.Errorf("expected child PID 2, got %d", childPID)
	}

	// Process table should have the child
	childProc, found := pt.Get(uint32(childPID))
	if !found {
		t.Fatal("child process not found in process table")
	}
	if childProc.PPID != 1 {
		t.Errorf("expected child PPID 1, got %d", childProc.PPID)
	}
	if childProc.State != ProcessReady {
		t.Errorf("expected child state ProcessReady, got %v", childProc.State)
	}

	// Child should have a VM with 0 on its stack
	if len(childProc.VM.stack) != 1 {
		t.Fatalf("expected child stack to have 1 element (0), got %d", len(childProc.VM.stack))
	}
	childStackVal := childProc.VM.stack[0].(IntValue)
	if childStackVal.Val != 0 {
		t.Errorf("expected child stack top to be 0, got %d", childStackVal.Val)
	}
}

// --- OpKill Tests (SEC-2 step 2.2) ---

func TestOpKillSetsZombie(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt

	// Register processes
	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	target := &Process{PID: 5, PPID: 1, State: ProcessRunning, VM: NewVM()}
	pt.Register(parent)
	pt.Register(target)
	testVM.pid = 1

	// Push target PID onto stack
	testVM.Push(IntValue{Val: 5})

	// Execute kill
	err := testVM.executeInstruction(OpKill)
	if err != nil {
		t.Fatalf("OpKill error: %v", err)
	}

	// Target should be zombie
	updated, ok := pt.Get(5)
	if !ok {
		t.Fatal("killed process should still exist in table")
	}
	if updated.State != ProcessZombie {
		t.Errorf("expected zombie state, got %v", updated.State)
	}
}

func TestOpKillReparentsChildren(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt

	// PID 1 = init
	init := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	// PID 3 = process to kill (has children)
	target := &Process{PID: 3, PPID: 1, State: ProcessRunning, VM: NewVM()}
	// PID 7, 8 = children of target
	grandchild1 := &Process{PID: 7, PPID: 3, State: ProcessRunning, VM: NewVM()}
	grandchild2 := &Process{PID: 8, PPID: 3, State: ProcessRunning, VM: NewVM()}

	pt.Register(init)
	pt.Register(target)
	pt.Register(grandchild1)
	pt.Register(grandchild2)
	testVM.pid = 1

	// Push target PID
	testVM.Push(IntValue{Val: 3})

	err := testVM.executeInstruction(OpKill)
	if err != nil {
		t.Fatalf("OpKill error: %v", err)
	}

	// Grandchildren should now be parented under PID 1
	for _, pid := range []uint32{7, 8} {
		proc, ok := pt.Get(pid)
		if !ok {
			t.Fatalf("process %d should still exist", pid)
		}
		if proc.PPID != 1 {
			t.Errorf("expected grandchild PID %d reparented to PID 1, got PPID %d", pid, proc.PPID)
		}
	}
}

func TestOpKillNonExistentPID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpKill)
	if err == nil {
		t.Fatal("expected error when killing non-existent PID")
	}
}

func TestOpKillCleansUpResources(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	targetVM := NewVM()
	targetVM.Push(IntValue{Val: 42})
	targetVM.Push(IntValue{Val: 99})
	target := &Process{PID: 5, PPID: 1, State: ProcessRunning, VM: targetVM}
	pt.Register(parent)
	pt.Register(target)

	testVM.Push(IntValue{Val: 5})

	err := testVM.executeInstruction(OpKill)
	if err != nil {
		t.Fatalf("OpKill error: %v", err)
	}

	// Target's VM should be cleaned up (stack cleared)
	killed, _ := pt.Get(5)
	if len(killed.VM.stack) != 0 {
		t.Errorf("expected killed process stack to be cleared, got %d items", len(killed.VM.stack))
	}
}

// --- OpWait Tests (SEC-2 step 2.3) ---

func TestOpWaitOnAlreadyZombie(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	// Child already exited with code 7
	child := &Process{
		PID:      2,
		PPID:     1,
		State:    ProcessZombie,
		ExitCode: 7,
		VM:       NewVM(),
	}
	pt.Register(parent)
	pt.Register(child)

	// Push child PID
	testVM.Push(IntValue{Val: 2})

	err := testVM.executeInstruction(OpWait)
	if err != nil {
		t.Fatalf("OpWait error: %v", err)
	}

	// Should get exit code on stack
	exitCode := mustPopInt(t, testVM)
	if exitCode != 7 {
		t.Errorf("expected exit code 7, got %d", exitCode)
	}

	// Zombie should be cleaned up
	_, found := pt.Get(2)
	if found {
		t.Error("zombie should be cleaned up after wait")
	}
}

func TestOpWaitBlocksUntilZombie(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	child := &Process{
		PID:   2,
		PPID:  1,
		State: ProcessRunning,
		VM:    NewVM(),
	}
	pt.Register(parent)
	pt.Register(child)

	// Push child PID
	testVM.Push(IntValue{Val: 2})

	// Simulate the child becoming a zombie asynchronously.
	done := make(chan struct{})
	go func() {
		// Simulate child finishing after a brief delay
		child.State = ProcessZombie
		child.ExitCode = 42
		pt.notifyWaiter(2)
		close(done)
	}()

	err := testVM.executeInstruction(OpWait)
	if err != nil {
		t.Fatalf("OpWait error: %v", err)
	}

	// Should get exit code on stack
	exitCode := mustPopInt(t, testVM)
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}

	<-done // Ensure goroutine completed
}

func TestOpWaitNonExistentPID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpWait)
	if err == nil {
		t.Fatal("expected error when waiting on non-existent PID")
	}
}

func TestOpWaitCleansUpZombie(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	child := &Process{
		PID:      3,
		PPID:     1,
		State:    ProcessZombie,
		ExitCode: 0,
		VM:       NewVM(),
	}
	pt.Register(parent)
	pt.Register(child)

	testVM.Push(IntValue{Val: 3})

	err := testVM.executeInstruction(OpWait)
	if err != nil {
		t.Fatalf("OpWait error: %v", err)
	}

	// Zombie should be removed
	_, found := pt.Get(3)
	if found {
		t.Error("expected zombie to be cleaned up after wait")
	}

	// Exit code 0 should be on stack
	exitCode := mustPopInt(t, testVM)
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

// --- Integration: Spawn -> Kill -> Wait ---

func TestSpawnKillWaitIntegration(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	// 1. Spawn a child
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	testVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	testVM.pc = int(layout.CodeOffset)

	testVM.executeInstruction(OpSpawn)

	// Get child PID from stack
	childPIDInt := mustPopInt(t, testVM)
	childPID := uint32(childPIDInt)

	// 2. Kill the child
	testVM.Push(IntValue{Val: int64(childPID)})
	testVM.executeInstruction(OpKill)

	// 3. Wait on the child (should be zombie already)
	testVM.Push(IntValue{Val: int64(childPID)})
	testVM.executeInstruction(OpWait)

	// Exit code should be 0 (killed processes get exit code 0)
	exitCode := mustPopInt(t, testVM)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for killed process, got %d", exitCode)
	}

	// Child should be cleaned up
	_, found := pt.Get(childPID)
	if found {
		t.Error("child should be cleaned up after wait")
	}
}

// --- SEC-3: Process Isolation and Hierarchy ---

// 3.1: Spawned VMs must have fully independent state.

func TestSpawnChildHasIndependentStack(t *testing.T) {
	parentVM := NewVM()
	pt := NewProcessTable()
	parentVM.processTable = pt
	parentVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: parentVM}
	pt.Register(parent)

	// Push something on parent stack before spawn
	parentVM.Push(IntValue{Val: 99})

	// Execute spawn
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)
	parentVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	parentVM.pc = int(layout.CodeOffset)

	if err := parentVM.executeInstruction(OpSpawn); err != nil {
		t.Fatalf("OpSpawn error: %v", err)
	}

	// Parent stack should have: 99 (original) + child PID (from spawn)
	if len(parentVM.stack) != 2 {
		t.Fatalf("expected parent stack size 2, got %d", len(parentVM.stack))
	}
	if parentVM.stack[0].(IntValue).Val != 99 {
		t.Errorf("expected parent stack[0] to be 99, got %v", parentVM.stack[0])
	}

	// Find child process
	childPID := uint32(parentVM.stack[1].(IntValue).Val)
	childProc, ok := pt.Get(childPID)
	if !ok {
		t.Fatal("child process not found")
	}

	// Child stack should be independent: only [0]
	if len(childProc.VM.stack) != 1 {
		t.Fatalf("expected child stack size 1, got %d", len(childProc.VM.stack))
	}
	if childProc.VM.stack[0].(IntValue).Val != 0 {
		t.Errorf("expected child stack[0] to be 0, got %v", childProc.VM.stack[0])
	}
}

func TestSpawnChildHasIndependentGlobals(t *testing.T) {
	parentVM := NewVM()
	pt := NewProcessTable()
	parentVM.processTable = pt
	parentVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: parentVM}
	pt.Register(parent)

	// Set a global on parent
	parentVM.globals["shared"] = IntValue{Val: 100}

	// Execute spawn
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)
	parentVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	parentVM.pc = int(layout.CodeOffset)

	if err := parentVM.executeInstruction(OpSpawn); err != nil {
		t.Fatalf("OpSpawn error: %v", err)
	}

	childPID := uint32(mustPopInt(t, parentVM))
	childProc, ok := pt.Get(childPID)
	if !ok {
		t.Fatal("child process not found")
	}

	// Modify child's globals — must NOT affect parent
	childProc.VM.globals["shared"] = IntValue{Val: 999}

	if parentVM.globals["shared"].(IntValue).Val != 100 {
		t.Errorf("parent global corrupted after child modification: got %v, want 100", parentVM.globals["shared"])
	}
}

func TestSpawnChildHasIndependentFunctions(t *testing.T) {
	parentVM := NewVM()
	pt := NewProcessTable()
	parentVM.processTable = pt
	parentVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: parentVM}
	pt.Register(parent)

	// Define a function on parent
	parentVM.functions["myFunc"] = FunctionValue{Name: "myFunc", ParamNames: []string{"x"}}

	// Execute spawn
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)
	parentVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	parentVM.pc = int(layout.CodeOffset)

	if err := parentVM.executeInstruction(OpSpawn); err != nil {
		t.Fatalf("OpSpawn error: %v", err)
	}

	childPID := uint32(mustPopInt(t, parentVM))
	childProc, ok := pt.Get(childPID)
	if !ok {
		t.Fatal("child process not found")
	}

	// Add a function to child — must NOT appear in parent
	childProc.VM.functions["childOnly"] = FunctionValue{Name: "childOnly"}

	if _, exists := parentVM.functions["childOnly"]; exists {
		t.Error("parent should not see child's new function")
	}
	// Parent's function should still be there
	if _, exists := parentVM.functions["myFunc"]; !exists {
		t.Error("parent's original function missing")
	}
}

// 3.2: Parent-child tracking with ChildPIDs list + notify on child exit.

func TestProcessTracksChildPIDs(t *testing.T) {
	pt := NewProcessTable()

	init := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: NewVM()}
	pt.Register(init)

	parentVM := NewVM()
	parentVM.processTable = pt
	parentVM.pid = 1

	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)
	parentVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	parentVM.pc = int(layout.CodeOffset)

	// Spawn two children
	parentVM.executeInstruction(OpSpawn)
	childPID1 := uint32(mustPopInt(t, parentVM))
	parentVM.pc = int(layout.CodeOffset) // reset pc
	parentVM.executeInstruction(OpSpawn)
	childPID2 := uint32(mustPopInt(t, parentVM))

	// Init should track both children
	initProc, _ := pt.Get(1)
	if len(initProc.ChildPIDs) != 2 {
		t.Fatalf("expected 2 children tracked, got %d", len(initProc.ChildPIDs))
	}

	found1, found2 := false, false
	for _, pid := range initProc.ChildPIDs {
		if pid == childPID1 { found1 = true }
		if pid == childPID2 { found2 = true }
	}
	if !found1 || !found2 {
		t.Errorf("expected child PIDs %d and %d in ChildPIDs, got %v", childPID1, childPID2, initProc.ChildPIDs)
	}
}

func TestSetZombieNotifiesParent(t *testing.T) {
	pt := NewProcessTable()

	parentVM := NewVM()
	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: parentVM}
	child := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: NewVM()}
	pt.Register(parent)
	pt.Register(child)

	// Register the parent as waiting on the child
	waitCh := pt.RegisterWaiter(2)

	// Set child to zombie — should notify
	go func() {
		pt.SetZombie(2, 0)
	}()

	// Parent should be woken up
	<-waitCh

	// Child should be zombie
	childProc, _ := pt.Get(2)
	if childProc.State != ProcessZombie {
		t.Error("expected child to be zombie")
	}
}

func TestRemoveChildPIDOnExit(t *testing.T) {
	pt := NewProcessTable()

	parentVM := NewVM()
	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: parentVM}
	child := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: NewVM()}
	pt.Register(parent)
	pt.Register(child)

	// Manually add child to parent's tracking
	parent.ChildPIDs = []uint32{2}

	// Kill the child — SetZombie should remove child from parent's ChildPIDs
	parentVM.processTable = pt
	parentVM.pid = 1
	parentVM.Push(IntValue{Val: 2})
	if err := parentVM.executeInstruction(OpKill); err != nil {
		t.Fatalf("OpKill error: %v", err)
	}

	// Parent should no longer track the dead child
	updated, _ := pt.Get(1)
	for _, pid := range updated.ChildPIDs {
		if pid == 2 {
			t.Error("dead child PID should be removed from parent's ChildPIDs")
		}
	}
}

// 3.3: Reparenting orphans to PID 1 on process death.

func TestReparentMovesChildrenToInit(t *testing.T) {
	pt := NewProcessTable()

	initVM := NewVM()
	init := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: initVM}
	parent := &Process{PID: 5, PPID: 1, State: ProcessRunning, VM: NewVM()}
	child1 := &Process{PID: 10, PPID: 5, State: ProcessRunning, VM: NewVM()}
	child2 := &Process{PID: 11, PPID: 5, State: ProcessRunning, VM: NewVM()}

	pt.Register(init)
	pt.Register(parent)
	pt.Register(child1)
	pt.Register(child2)

	// Set up parent-child tracking
	parent.ChildPIDs = []uint32{10, 11}
	init.ChildPIDs = []uint32{5}

	// Kill parent PID 5
	initVM.processTable = pt
	initVM.pid = 1
	initVM.Push(IntValue{Val: 5})
	if err := initVM.executeInstruction(OpKill); err != nil {
		t.Fatalf("OpKill error: %v", err)
	}

	// Children should be reparented to PID 1
	for _, pid := range []uint32{10, 11} {
		proc, ok := pt.Get(pid)
		if !ok {
			t.Fatalf("process %d should still exist", pid)
		}
		if proc.PPID != 1 {
			t.Errorf("expected PID %d reparented to PID 1, got PPID %d", pid, proc.PPID)
		}
	}

	// PID 1 should now track the adopted children
	initProc, _ := pt.Get(1)
	found10, found11 := false, false
	for _, pid := range initProc.ChildPIDs {
		if pid == 10 { found10 = true }
		if pid == 11 { found11 = true }
	}
	if !found10 || !found11 {
		t.Errorf("expected PID 1 to adopt orphans 10 and 11, got ChildPIDs %v", initProc.ChildPIDs)
	}
}

func TestPID1ReapsOrphanedZombies(t *testing.T) {
	pt := NewProcessTable()

	initVM := NewVM()
	init := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: initVM}
	// Orphan: already zombie, parent already dead
	orphan := &Process{PID: 7, PPID: 1, State: ProcessZombie, ExitCode: 3, VM: NewVM()}

	pt.Register(init)
	pt.Register(orphan)
	init.ChildPIDs = []uint32{7}

	// PID 1 does OpWait on the orphan — should reap immediately
	initVM.processTable = pt
	initVM.Push(IntValue{Val: 7})
	if err := initVM.executeInstruction(OpWait); err != nil {
		t.Fatalf("OpWait error: %v", err)
	}

	exitCode := mustPopInt(t, initVM)
	if exitCode != 3 {
		t.Errorf("expected exit code 3, got %d", exitCode)
	}

	// Orphan should be cleaned up
	if _, found := pt.Get(7); found {
		t.Error("orphan should be reaped after wait")
	}
}
