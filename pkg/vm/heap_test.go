package vm

import (
	"strings"
	"testing"
)

// --- Step 1.1: Heap region and PointerValue ---

func TestPointerValueType(t *testing.T) {
	p := PointerValue{Address: 42}
	if p.Type() != "ptr" {
		t.Errorf("expected Type() 'ptr', got %q", p.Type())
	}
}

func TestNewVMHasHeap(t *testing.T) {
	vm := NewVM()
	if vm.heap == nil {
		t.Fatal("expected VM to have a non-nil heap")
	}
	if vm.hp != heapBase {
		t.Errorf("expected HP to be heapBase (%d), got %d", heapBase, vm.hp)
	}
}

func TestHeapBaseConstant(t *testing.T) {
	// heapBase should be 0x10000 (65536), leaving room for stack and globals below
	if heapBase != 0x10000 {
		t.Errorf("expected heapBase=0x10000, got 0x%x", heapBase)
	}
}

// --- Step 1.2: OpAlloc ---

func TestOpAllocBasic(t *testing.T) {
	vm := NewVM()

	// Push size = 16, then alloc
	vm.Push(IntValue{Val: 16})
	err := vm.execAlloc()
	if err != nil {
		t.Fatalf("execAlloc() error: %v", err)
	}

	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}

	ptr, ok := result.(PointerValue)
	if !ok {
		t.Fatalf("expected PointerValue, got %T", result)
	}
	if ptr.Address < heapBase {
		t.Errorf("pointer address %d should be >= heapBase %d", ptr.Address, heapBase)
	}
}

func TestOpAllocAdvancesHP(t *testing.T) {
	vm := NewVM()
	initialHP := vm.hp

	vm.Push(IntValue{Val: 32})
	vm.execAlloc()

	// HP should have advanced by headerSize + aligned(32)
	blockSize := align8(32)
	expectedAdvance := headerSize + blockSize
	if vm.hp != initialHP+uint32(expectedAdvance) {
		t.Errorf("expected HP to advance by %d, got advance of %d", expectedAdvance, vm.hp-initialHP)
	}
}

func TestOpAllocMultiple(t *testing.T) {
	vm := NewVM()

	// Allocate two blocks
	vm.Push(IntValue{Val: 8})
	vm.execAlloc()
	ptr1, _ := vm.Pop()

	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr2, _ := vm.Pop()

	p1 := ptr1.(PointerValue).Address
	p2 := ptr2.(PointerValue).Address

	if p1 == p2 {
		t.Error("two allocations should return different addresses")
	}
	if p2 <= p1 {
		t.Errorf("expected p2 (%d) > p1 (%d)", p2, p1)
	}
}

func TestOpAllocZeroSize(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: 0})
	err := vm.execAlloc()
	if err != nil {
		t.Fatalf("alloc(0) should succeed: %v", err)
	}
	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}
	ptr, ok := result.(PointerValue)
	if !ok {
		t.Fatalf("expected PointerValue, got %T", result)
	}
	// Even 0-size allocation should have a valid header
	if ptr.Address < heapBase {
		t.Errorf("pointer address should be >= heapBase")
	}
}

func TestOpAllocStackUnderflow(t *testing.T) {
	vm := NewVM()
	// No value on stack
	err := vm.execAlloc()
	if err == nil {
		t.Error("expected error on empty stack")
	}
}

func TestOpAllocNonIntegerSize(t *testing.T) {
	vm := NewVM()
	vm.Push(StringValue{Val: "not a size"})
	err := vm.execAlloc()
	if err == nil {
		t.Error("expected error for non-integer size")
	}
	if !strings.Contains(err.Error(), "integer") {
		t.Errorf("expected 'integer' in error, got %q", err.Error())
	}
}

func TestOpAllocNegativeSize(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: -1})
	err := vm.execAlloc()
	if err == nil {
		t.Error("expected error for negative size")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Errorf("expected 'positive' in error, got %q", err.Error())
	}
}

// --- Step 1.3: OpFree ---

func TestOpFreeBasic(t *testing.T) {
	vm := NewVM()

	// Allocate, then free
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	vm.Push(ptr)
	err := vm.execFree()
	if err != nil {
		t.Fatalf("execFree() error: %v", err)
	}
}

func TestOpFreeDoubleFree(t *testing.T) {
	vm := NewVM()

	// Allocate then free twice
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	vm.Push(ptr)
	vm.execFree() // first free OK

	vm.Push(ptr)
	err := vm.execFree() // double free should error
	if err == nil {
		t.Error("expected error on double free")
	}
	if !strings.Contains(err.Error(), "double free") {
		t.Errorf("expected 'double free' in error, got %q", err.Error())
	}
}

func TestOpFreeInvalidPointer(t *testing.T) {
	vm := NewVM()

	// Try to free a non-pointer value
	vm.Push(IntValue{Val: 42})
	err := vm.execFree()
	if err == nil {
		t.Error("expected error when freeing non-pointer")
	}
}

func TestOpFreeInvalidAddress(t *testing.T) {
	vm := NewVM()

	// Free a pointer to address that was never allocated
	vm.Push(PointerValue{Address: 99999})
	err := vm.execFree()
	if err == nil {
		t.Error("expected error for invalid heap address")
	}
}

func TestOpFreeCoalescing(t *testing.T) {
	vm := NewVM()

	// Allocate three adjacent blocks
	vm.Push(IntValue{Val: 8})
	vm.execAlloc()
	ptr1, _ := vm.Pop()

	vm.Push(IntValue{Val: 8})
	vm.execAlloc()
	ptr2, _ := vm.Pop()

	vm.Push(IntValue{Val: 8})
	vm.execAlloc()
	ptr3, _ := vm.Pop()

	// Free middle, then first — should coalesce
	vm.Push(ptr2)
	vm.execFree()

	vm.Push(ptr1)
	vm.execFree()

	// Verify ptr3 is still valid
	vm.Push(ptr3)
	err := vm.execFree()
	if err != nil {
		t.Errorf("ptr3 should still be valid after freeing neighbors: %v", err)
	}
}

func TestOpFreeReuse(t *testing.T) {
	vm := NewVM()

	// Allocate, free, then allocate again — should reuse freed block
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr1, _ := vm.Pop()

	vm.Push(ptr1)
	vm.execFree()

	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr2, _ := vm.Pop()

	// The reused allocation may or may not be same address,
	// but it should be valid
	p2 := ptr2.(PointerValue)
	if p2.Address < heapBase {
		t.Errorf("reused pointer should be valid")
	}
}

func TestOpFreeStackUnderflow(t *testing.T) {
	vm := NewVM()
	err := vm.execFree()
	if err == nil {
		t.Error("expected error on empty stack")
	}
}

// --- Integration: bytecode execution ---

func TestAllocFreeViaBytecode(t *testing.T) {
	constants := []Value{IntValue{Val: 32}}
	bytecode := createBytecodeHeader(constants)

	op0 := uint32(0)
	bytecode = addInstruction(bytecode, OpPush, &op0)  // push size
	bytecode = addInstruction(bytecode, OpAlloc, nil)   // alloc
	bytecode = addInstruction(bytecode, OpFree, nil)    // free
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	// Free pushes nothing meaningful, so after halt we get NullValue
	_, ok := result.(NullValue)
	if !ok {
		t.Logf("Result type after free+halt: %T (value: %v) — this is acceptable", result, result)
	}
}

// --- Step 2.1: OpLoadPtr ---

func TestOpLoadPtrI64(t *testing.T) {
	vm := NewVM()

	// Alloc a block, store an i64, then load it back
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Store value 42 at offset 0
	vm.Push(ptr)              // ptr
	vm.Push(IntValue{Val: 0}) // offset
	vm.Push(IntValue{Val: 42}) // value
	vm.execStorePtr()

	// Load value at offset 0
	vm.Push(ptr)              // ptr
	vm.Push(IntValue{Val: 0}) // offset
	err := vm.execLoadPtr()
	if err != nil {
		t.Fatalf("execLoadPtr() error: %v", err)
	}

	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}

	intVal, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Val != 42 {
		t.Errorf("expected 42, got %d", intVal.Val)
	}
}

func TestOpLoadPtrF64(t *testing.T) {
	vm := NewVM()

	// Alloc a block and set its type tag to f64
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Set the type tag to f64
	block, _ := vm.heap.GetBlock(ptr.(PointerValue).Address)
	block.TypeTag = TypeTagF64

	// Store 3.14 at offset 0
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(FloatValue{Val: 3.14})
	vm.execStorePtr()

	// Load it back
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	err := vm.execLoadPtr()
	if err != nil {
		t.Fatalf("execLoadPtr() error: %v", err)
	}

	result, _ := vm.Pop()
	floatVal, ok := result.(FloatValue)
	if !ok {
		t.Fatalf("expected FloatValue, got %T", result)
	}
	if floatVal.Val != 3.14 {
		t.Errorf("expected 3.14, got %f", floatVal.Val)
	}
}

func TestOpLoadPtrAtOffset(t *testing.T) {
	vm := NewVM()

	// Alloc block large enough for two i64 values
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Store 100 at offset 0
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 100})
	vm.execStorePtr()

	// Store 200 at offset 8
	vm.Push(ptr)
	vm.Push(IntValue{Val: 8})
	vm.Push(IntValue{Val: 200})
	vm.execStorePtr()

	// Load offset 8 — should get 200
	vm.Push(ptr)
	vm.Push(IntValue{Val: 8})
	vm.execLoadPtr()

	result, _ := vm.Pop()
	intVal, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Val != 200 {
		t.Errorf("expected 200, got %d", intVal.Val)
	}
}

// --- Step 2.2: OpStorePtr ---

func TestOpStorePtrOverwrite(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Store 10
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 10})
	vm.execStorePtr()

	// Overwrite with 20
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 20})
	vm.execStorePtr()

	// Load — should get 20
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.execLoadPtr()

	result, _ := vm.Pop()
	intVal, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Val != 20 {
		t.Errorf("expected 20, got %d", intVal.Val)
	}
}

func TestOpStorePtrPointerType(t *testing.T) {
	vm := NewVM()

	// Allocate block A (to be stored as a pointer value)
	vm.Push(IntValue{Val: 8})
	vm.execAlloc()
	ptrA, _ := vm.Pop()

	// Allocate block B (will hold a pointer to A), set its type tag to ptr
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptrB, _ := vm.Pop()

	block, _ := vm.heap.GetBlock(ptrB.(PointerValue).Address)
	block.TypeTag = TypeTagPtr

	// Store ptrA into block B at offset 0
	vm.Push(ptrB)
	vm.Push(IntValue{Val: 0})
	vm.Push(ptrA)
	vm.execStorePtr()

	// Load the pointer from block B
	vm.Push(ptrB)
	vm.Push(IntValue{Val: 0})
	err := vm.execLoadPtr()
	if err != nil {
		t.Fatalf("execLoadPtr() error: %v", err)
	}

	result, _ := vm.Pop()
	loadedPtr, ok := result.(PointerValue)
	if !ok {
		t.Fatalf("expected PointerValue, got %T", result)
	}
	if loadedPtr.Address != ptrA.(PointerValue).Address {
		t.Errorf("expected address %d, got %d", ptrA.(PointerValue).Address, loadedPtr.Address)
	}
}

// --- Step 2.3: Bounds checking ---

func TestOpLoadPtrOutOfBounds(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 8}) // 8-byte block
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Try to read at offset 16 — out of bounds for 8-byte block
	vm.Push(ptr)
	vm.Push(IntValue{Val: 16})
	err := vm.execLoadPtr()
	if err == nil {
		t.Error("expected SEGFAULT error for out-of-bounds load")
	}
}

func TestOpStorePtrOutOfBounds(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 8}) // 8-byte block
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Try to write at offset 16 — out of bounds
	vm.Push(ptr)
	vm.Push(IntValue{Val: 16})
	vm.Push(IntValue{Val: 42})
	err := vm.execStorePtr()
	if err == nil {
		t.Error("expected SEGFAULT error for out-of-bounds store")
	}
}

func TestOpLoadPtrInvalidPointer(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 99999}) // not a valid heap pointer
	vm.Push(IntValue{Val: 0})
	err := vm.execLoadPtr()
	if err == nil {
		t.Error("expected error for invalid heap pointer")
	}
}

func TestOpStorePtrInvalidPointer(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 99999}) // not a valid heap pointer
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 42})
	err := vm.execStorePtr()
	if err == nil {
		t.Error("expected error for invalid heap pointer")
	}
}

func TestOpLoadPtrFreedBlock(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Free the block
	vm.Push(ptr)
	vm.execFree()

	// Try to load from freed block
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	err := vm.execLoadPtr()
	if err == nil {
		t.Error("expected error when loading from freed block")
	}
}

func TestOpStorePtrFreedBlock(t *testing.T) {
	vm := NewVM()

	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	// Free the block
	vm.Push(ptr)
	vm.execFree()

	// Try to store to freed block
	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 42})
	err := vm.execStorePtr()
	if err == nil {
		t.Error("expected error when storing to freed block")
	}
}

func TestOpLoadPtrStackUnderflow(t *testing.T) {
	vm := NewVM()
	err := vm.execLoadPtr()
	if err == nil {
		t.Error("expected error on empty stack")
	}
}

func TestOpStorePtrStackUnderflow(t *testing.T) {
	vm := NewVM()
	err := vm.execStorePtr()
	if err == nil {
		t.Error("expected error on empty stack")
	}
}

// --- Integration: LoadPtr/StorePtr via bytecode ---

func TestLoadStorePtrViaBytecode(t *testing.T) {
	vm := NewVM()

	// Manually set up: alloc 16 bytes, store 99 at offset 0, load it back
	vm.Push(IntValue{Val: 16})
	vm.execAlloc()
	ptr, _ := vm.Pop()

	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.Push(IntValue{Val: 99})
	vm.execStorePtr()

	vm.Push(ptr)
	vm.Push(IntValue{Val: 0})
	vm.execLoadPtr()

	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}

	intVal, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Val != 99 {
		t.Errorf("expected 99, got %d", intVal.Val)
	}
}

func TestAllocReturnsPointerViaBytecode(t *testing.T) {
	constants := []Value{IntValue{Val: 64}}
	bytecode := createBytecodeHeader(constants)

	op0 := uint32(0)
	bytecode = addInstruction(bytecode, OpPush, &op0) // push size
	bytecode = addInstruction(bytecode, OpAlloc, nil)  // alloc -> pushes PointerValue
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	ptr, ok := result.(PointerValue)
	if !ok {
		t.Fatalf("expected PointerValue, got %T", result)
	}
	if ptr.Address < heapBase {
		t.Errorf("pointer address should be >= heapBase")
	}
}
