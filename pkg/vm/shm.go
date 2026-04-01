package vm

import (
	"fmt"
	"sync"
)

// ShmHandle is a unique identifier for a shared memory region.
type ShmHandle uint32

// SharedMemoryRegion represents a block of memory shared between processes.
type SharedMemoryRegion struct {
	Handle ShmHandle
	Size   int
	Data   []byte
	Owner  uint32    // PID of the creator
	mu     sync.RWMutex
	// Access control: tracks which PIDs have access to this region.
	// The owner always has access. Other PIDs must be explicitly granted.
	allowed map[uint32]bool
}

// newSharedMemoryRegion creates a new shared memory region owned by the given PID.
func newSharedMemoryRegion(handle ShmHandle, size int, ownerPID uint32) *SharedMemoryRegion {
	return &SharedMemoryRegion{
		Handle:  handle,
		Size:    size,
		Data:    make([]byte, size),
		Owner:   ownerPID,
		allowed: map[uint32]bool{ownerPID: true},
	}
}

// HasAccess returns true if the given PID is allowed to access this region.
func (r *SharedMemoryRegion) HasAccess(pid uint32) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.allowed[pid]
}

// Grant gives a PID access to this region. Only the owner can grant access.
// Returns an error if the caller is not the owner.
func (r *SharedMemoryRegion) Grant(callerPID, targetPID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Owner != callerPID {
		return fmt.Errorf("shm_grant: only owner (pid %d) can grant access, caller is pid %d", r.Owner, callerPID)
	}

	r.allowed[targetPID] = true
	return nil
}

// Revoke removes a PID's access to this region. Only the owner can revoke.
// The owner cannot revoke their own access. Returns an error if the caller
// is not the owner or if trying to revoke the owner.
func (r *SharedMemoryRegion) Revoke(callerPID, targetPID uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Owner != callerPID {
		return fmt.Errorf("shm_revoke: only owner (pid %d) can revoke access, caller is pid %d", r.Owner, callerPID)
	}
	if targetPID == r.Owner {
		return fmt.Errorf("shm_revoke: cannot revoke owner access")
	}

	delete(r.allowed, targetPID)
	return nil
}

// SharedMemoryTable is a concurrent-safe global namespace for shared memory regions.
type SharedMemoryTable struct {
	mu      sync.RWMutex
	regions map[ShmHandle]*SharedMemoryRegion
	nextID  ShmHandle
}

// NewSharedMemoryTable creates an empty shared memory table with handle allocation starting at 1.
func NewSharedMemoryTable() *SharedMemoryTable {
	return &SharedMemoryTable{
		regions: make(map[ShmHandle]*SharedMemoryRegion),
		nextID:  1,
	}
}

// Create allocates a new shared memory region of the given size, owned by ownerPID.
func (t *SharedMemoryTable) Create(size int, ownerPID uint32) *SharedMemoryRegion {
	t.mu.Lock()
	defer t.mu.Unlock()

	handle := t.nextID
	t.nextID++

	region := newSharedMemoryRegion(handle, size, ownerPID)
	t.regions[handle] = region
	return region
}

// Get retrieves a shared memory region by handle. Returns nil, false if not found.
func (t *SharedMemoryTable) Get(handle ShmHandle) (*SharedMemoryRegion, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	r, ok := t.regions[handle]
	return r, ok
}

// Remove deletes a shared memory region from the table.
func (t *SharedMemoryTable) Remove(handle ShmHandle) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.regions, handle)
}

// --- Opcodes for shared memory ---

// OpShmCreate (0xD3): allocate a shared memory region.
// Operand: size (uint32). Pushes the handle (IntValue) onto the stack.
const OpShmCreate Opcode = 0xD3

// OpShmMap (0xD4): map a shared memory region into the process's address space.
// Stack: [handle] -> pops handle, pushes pointer offset (IntValue).
// Validates that the calling process has access to the region.
const OpShmMap Opcode = 0xD4

// OpShmGrant (0xD5): grant another process access to a shared memory region.
// Stack: [target_pid, handle] -> pops both.
// Only the owner can grant access.
const OpShmGrant Opcode = 0xD5

// OpShmRevoke (0xD6): revoke another process's access to a shared memory region.
// Stack: [target_pid, handle] -> pops both.
// Only the owner can revoke access. Cannot revoke the owner.
const OpShmRevoke Opcode = 0xD6

// execShmCreate allocates a new shared memory region of the given size.
func (vm *VM) execShmCreate() error {
	if vm.processTable == nil {
		return fmt.Errorf("shm_create: no process table attached to VM")
	}

	size, err := vm.readOperand()
	if err != nil {
		return err
	}

	if size == 0 {
		return fmt.Errorf("shm_create: size must be > 0")
	}

	// Lazily initialize shared memory table on the process table
	if vm.processTable.shmTable == nil {
		vm.processTable.shmTable = NewSharedMemoryTable()
	}

	region := vm.processTable.shmTable.Create(int(size), vm.pid)
	vm.Push(IntValue{Val: int64(region.Handle)})
	return nil
}

// execShmMap maps a shared memory region into the process's address space.
// Stack: [handle] -> pops handle, pushes pointer offset.
func (vm *VM) execShmMap() error {
	if vm.processTable == nil {
		return fmt.Errorf("shm_map: no process table attached to VM")
	}

	handleVal, err := vm.Pop()
	if err != nil {
		return err
	}

	handleInt, ok := handleVal.(IntValue)
	if !ok {
		return fmt.Errorf("shm_map: handle must be an integer")
	}
	handle := ShmHandle(handleInt.Val)

	if vm.processTable.shmTable == nil {
		return fmt.Errorf("shm_map: no shared memory table")
	}

	region, exists := vm.processTable.shmTable.Get(handle)
	if !exists {
		return fmt.Errorf("shm_map: region %d not found", handle)
	}

	if !region.HasAccess(vm.pid) {
		return fmt.Errorf("shm_map: pid %d does not have access to region %d", vm.pid, handle)
	}

	// Return a pointer offset into the region's data.
	// For now, the "pointer" is an opaque integer that the VM can use
	// to reference the shared memory. We use the handle itself as the
	// address, since the region data is accessible via the table.
	vm.Push(IntValue{Val: int64(handle)})
	return nil
}

// execShmGrant grants another process access to a shared memory region.
// Stack: [target_pid, handle] -> pops both.
func (vm *VM) execShmGrant() error {
	if vm.processTable == nil {
		return fmt.Errorf("shm_grant: no process table attached to VM")
	}

	targetPIDVal, err := vm.Pop()
	if err != nil {
		return err
	}
	targetPIDInt, ok := targetPIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("shm_grant: target PID must be an integer")
	}
	targetPID := uint32(targetPIDInt.Val)

	handleVal, err := vm.Pop()
	if err != nil {
		return err
	}
	handleInt, ok := handleVal.(IntValue)
	if !ok {
		return fmt.Errorf("shm_grant: handle must be an integer")
	}
	handle := ShmHandle(handleInt.Val)

	if vm.processTable.shmTable == nil {
		return fmt.Errorf("shm_grant: no shared memory table")
	}

	region, exists := vm.processTable.shmTable.Get(handle)
	if !exists {
		return fmt.Errorf("shm_grant: region %d not found", handle)
	}

	return region.Grant(vm.pid, targetPID)
}

// execShmRevoke revokes another process's access to a shared memory region.
// Stack: [target_pid, handle] -> pops both.
func (vm *VM) execShmRevoke() error {
	if vm.processTable == nil {
		return fmt.Errorf("shm_revoke: no process table attached to VM")
	}

	targetPIDVal, err := vm.Pop()
	if err != nil {
		return err
	}
	targetPIDInt, ok := targetPIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("shm_revoke: target PID must be an integer")
	}
	targetPID := uint32(targetPIDInt.Val)

	handleVal, err := vm.Pop()
	if err != nil {
		return err
	}
	handleInt, ok := handleVal.(IntValue)
	if !ok {
		return fmt.Errorf("shm_revoke: handle must be an integer")
	}
	handle := ShmHandle(handleInt.Val)

	if vm.processTable.shmTable == nil {
		return fmt.Errorf("shm_revoke: no shared memory table")
	}

	region, exists := vm.processTable.shmTable.Get(handle)
	if !exists {
		return fmt.Errorf("shm_revoke: region %d not found", handle)
	}

	return region.Revoke(vm.pid, targetPID)
}
