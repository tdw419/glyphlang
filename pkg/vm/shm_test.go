package vm

import (
	"encoding/binary"
	"testing"
)

// helper: set up a VM with a process table and a registered process.
func newShmTestVM(pid uint32) (*VM, *ProcessTable) {
	vm := NewVM()
	pt := NewProcessTable()
	vm.processTable = pt
	vm.pid = pid
	proc := &Process{PID: pid, PPID: 0, State: ProcessRunning, VM: vm}
	pt.Register(proc)
	return vm, pt
}

// --- SharedMemoryTable Tests (Step 2.1) ---

func TestSharedMemoryTableCreate(t *testing.T) {
	st := NewSharedMemoryTable()

	r := st.Create(1024, 1)
	if r == nil {
		t.Fatal("Create returned nil")
	}
	if r.Handle != 1 {
		t.Errorf("expected first handle to be 1, got %d", r.Handle)
	}
	if r.Size != 1024 {
		t.Errorf("expected size 1024, got %d", r.Size)
	}
	if r.Owner != 1 {
		t.Errorf("expected owner PID 1, got %d", r.Owner)
	}
	if len(r.Data) != 1024 {
		t.Errorf("expected data buffer of 1024 bytes, got %d", len(r.Data))
	}

	r2 := st.Create(512, 2)
	if r2.Handle != 2 {
		t.Errorf("expected second handle to be 2, got %d", r2.Handle)
	}
}

func TestSharedMemoryTableGet(t *testing.T) {
	st := NewSharedMemoryTable()
	st.Create(256, 1)

	r, ok := st.Get(1)
	if !ok {
		t.Fatal("expected to find region 1")
	}
	if r.Size != 256 {
		t.Errorf("expected size 256, got %d", r.Size)
	}

	_, ok = st.Get(999)
	if ok {
		t.Error("expected Get(999) to return false")
	}
}

func TestSharedMemoryTableRemove(t *testing.T) {
	st := NewSharedMemoryTable()
	st.Create(256, 1)

	st.Remove(1)
	_, ok := st.Get(1)
	if ok {
		t.Error("expected region to be removed")
	}
}

// --- Access Control Tests (Step 2.3) ---

func TestSharedMemoryRegionOwnerHasAccess(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)
	if !r.HasAccess(10) {
		t.Error("owner should have access")
	}
}

func TestSharedMemoryRegionNonOwnerNoAccess(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)
	if r.HasAccess(20) {
		t.Error("non-owner should not have access by default")
	}
}

func TestSharedMemoryRegionGrantAccess(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)

	err := r.Grant(10, 20)
	if err != nil {
		t.Fatalf("Grant error: %v", err)
	}
	if !r.HasAccess(20) {
		t.Error("pid 20 should have access after grant")
	}
}

func TestSharedMemoryRegionGrantNonOwnerDenied(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)

	err := r.Grant(20, 30) // pid 20 is not the owner
	if err == nil {
		t.Error("expected error when non-owner tries to grant")
	}
}

func TestSharedMemoryRegionRevokeAccess(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)
	r.Grant(10, 20)

	err := r.Revoke(10, 20)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}
	if r.HasAccess(20) {
		t.Error("pid 20 should not have access after revoke")
	}
}

func TestSharedMemoryRegionRevokeNonOwnerDenied(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)
	r.Grant(10, 20)

	err := r.Revoke(20, 10) // pid 20 is not the owner
	if err == nil {
		t.Error("expected error when non-owner tries to revoke")
	}
}

func TestSharedMemoryRegionCannotRevokeOwner(t *testing.T) {
	r := newSharedMemoryRegion(1, 64, 10)

	err := r.Revoke(10, 10)
	if err == nil {
		t.Error("expected error when trying to revoke owner's access")
	}
}

// --- OpShmCreate Tests (Step 2.1) ---

func TestOpShmCreate(t *testing.T) {
	vm, pt := newShmTestVM(1)

	// Set up code: size operand = 1024
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, 1024)
	vm.code = sizeBytes
	vm.pc = 0

	err := vm.executeInstruction(OpShmCreate)
	if err != nil {
		t.Fatalf("executeInstruction(OpShmCreate) error: %v", err)
	}

	handle := mustPopInt(t, vm)
	if handle != 1 {
		t.Errorf("expected handle 1, got %d", handle)
	}

	// Region should exist in table
	region, ok := pt.shmTable.Get(1)
	if !ok {
		t.Fatal("expected region 1 in shared memory table")
	}
	if region.Size != 1024 {
		t.Errorf("expected size 1024, got %d", region.Size)
	}
	if region.Owner != 1 {
		t.Errorf("expected owner PID 1, got %d", region.Owner)
	}
}

func TestOpShmCreateNoProcessTable(t *testing.T) {
	vm := NewVM()
	vm.code = []byte{byte(OpShmCreate), 0, 0, 0, 0}
	vm.pc = 0

	err := vm.executeInstruction(OpShmCreate)
	if err == nil {
		t.Fatal("expected error without process table")
	}
}

func TestOpShmCreateZeroSize(t *testing.T) {
	vm, _ := newShmTestVM(1)

	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, 0)
	vm.code = sizeBytes
	vm.pc = 0

	err := vm.executeInstruction(OpShmCreate)
	if err == nil {
		t.Fatal("expected error for zero size")
	}
}

func TestOpShmCreateMultipleRegions(t *testing.T) {
	vm, _ := newShmTestVM(1)

	// Create first region
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, 256)
	vm.code = sizeBytes
	vm.pc = 0
	vm.executeInstruction(OpShmCreate)

	// Create second region
	binary.LittleEndian.PutUint32(sizeBytes, 512)
	vm.code = sizeBytes
	vm.pc = 0
	vm.executeInstruction(OpShmCreate)

	// Second handle should be 2
	handle2 := mustPopInt(t, vm)
	if handle2 != 2 {
		t.Errorf("expected handle 2, got %d", handle2)
	}

	// First handle should be 1
	handle1 := mustPopInt(t, vm)
	if handle1 != 1 {
		t.Errorf("expected handle 1, got %d", handle1)
	}
}

// --- OpShmMap Tests (Step 2.2) ---

func TestOpShmMapOwnerAllowed(t *testing.T) {
	vm, pt := newShmTestVM(1)

	// Create a shared memory region
	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(1024, 1)

	// Write some test data to verify shared access
	region.Data[0] = 0xAB
	region.Data[1] = 0xCD

	// Push handle and execute OpShmMap
	vm.Push(IntValue{Val: int64(region.Handle)})

	err := vm.executeInstruction(OpShmMap)
	if err != nil {
		t.Fatalf("OpShmMap error: %v", err)
	}

	ptr := mustPopInt(t, vm)
	if ptr != int64(region.Handle) {
		t.Errorf("expected pointer %d, got %d", region.Handle, ptr)
	}
}

func TestOpShmMapAccessDenied(t *testing.T) {
	vm, pt := newShmTestVM(2) // PID 2, not the owner

	// Create a region owned by PID 1
	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(1024, 1)

	// PID 2 tries to map a region owned by PID 1 (should fail)
	vm.Push(IntValue{Val: int64(region.Handle)})

	err := vm.executeInstruction(OpShmMap)
	if err == nil {
		t.Fatal("expected error when non-owner tries to map without grant")
	}
}

func TestOpShmMapInvalidHandle(t *testing.T) {
	vm, pt := newShmTestVM(1)
	pt.shmTable = NewSharedMemoryTable()

	vm.Push(IntValue{Val: 999}) // invalid handle

	err := vm.executeInstruction(OpShmMap)
	if err == nil {
		t.Fatal("expected error for invalid handle")
	}
}

func TestOpShmMapNoProcessTable(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: 1})

	err := vm.executeInstruction(OpShmMap)
	if err == nil {
		t.Fatal("expected error without process table")
	}
}

// --- OpShmGrant Tests (Step 2.3) ---

func TestOpShmGrantOwnerCanGrant(t *testing.T) {
	vm, pt := newShmTestVM(1)

	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(256, 1)

	// Grant PID 2 access: stack = [handle, target_pid]
	vm.Push(IntValue{Val: int64(region.Handle)})
	vm.Push(IntValue{Val: 2})

	err := vm.executeInstruction(OpShmGrant)
	if err != nil {
		t.Fatalf("OpShmGrant error: %v", err)
	}

	if !region.HasAccess(2) {
		t.Error("pid 2 should have access after grant")
	}
}

func TestOpShmGrantNonOwnerDenied(t *testing.T) {
	vm, pt := newShmTestVM(2) // PID 2, not the owner

	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(256, 1) // owned by PID 1

	// PID 2 tries to grant PID 3 access (should fail)
	vm.Push(IntValue{Val: int64(region.Handle)})
	vm.Push(IntValue{Val: 3})

	err := vm.executeInstruction(OpShmGrant)
	if err == nil {
		t.Fatal("expected error when non-owner tries to grant")
	}
}

// --- OpShmRevoke Tests (Step 2.3) ---

func TestOpShmRevokeOwnerCanRevoke(t *testing.T) {
	vm, pt := newShmTestVM(1)

	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(256, 1)
	region.Grant(1, 2)

	// Revoke PID 2's access: stack = [handle, target_pid]
	vm.Push(IntValue{Val: int64(region.Handle)})
	vm.Push(IntValue{Val: 2})

	err := vm.executeInstruction(OpShmRevoke)
	if err != nil {
		t.Fatalf("OpShmRevoke error: %v", err)
	}

	if region.HasAccess(2) {
		t.Error("pid 2 should not have access after revoke")
	}
}

func TestOpShmRevokeCannotRevokeOwner(t *testing.T) {
	vm, pt := newShmTestVM(1)

	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(256, 1)

	// Try to revoke owner's own access
	vm.Push(IntValue{Val: int64(region.Handle)})
	vm.Push(IntValue{Val: 1})

	err := vm.executeInstruction(OpShmRevoke)
	if err == nil {
		t.Fatal("expected error when trying to revoke owner's access")
	}
}

// --- Integration: Create -> Grant -> Map (cross-process) ---

func TestShmCreateGrantMapIntegration(t *testing.T) {
	ownerVM, pt := newShmTestVM(1)

	// Step 1: Owner creates shared memory
	pt.shmTable = NewSharedMemoryTable()
	region := pt.shmTable.Create(64, 1)

	// Owner writes data to the region
	region.Data[0] = 0xDE
	region.Data[1] = 0xAD

	// Step 2: Owner grants PID 2 access
	ownerVM.Push(IntValue{Val: int64(region.Handle)})
	ownerVM.Push(IntValue{Val: 2})
	err := ownerVM.executeInstruction(OpShmGrant)
	if err != nil {
		t.Fatalf("Grant error: %v", err)
	}

	// Step 3: PID 2 maps the region
	receiverVM := NewVM()
	receiverVM.processTable = pt
	receiverVM.pid = 2
	receiver := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: receiverVM}
	pt.Register(receiver)

	receiverVM.Push(IntValue{Val: int64(region.Handle)})
	err = receiverVM.executeInstruction(OpShmMap)
	if err != nil {
		t.Fatalf("OpShmMap error: %v", err)
	}

	ptr := mustPopInt(t, receiverVM)
	if ptr != int64(region.Handle) {
		t.Errorf("expected pointer %d, got %d", region.Handle, ptr)
	}

	// Verify that PID 2 can read the data the owner wrote
	if region.Data[0] != 0xDE || region.Data[1] != 0xAD {
		t.Errorf("shared memory data mismatch: got %x %x", region.Data[0], region.Data[1])
	}
}
