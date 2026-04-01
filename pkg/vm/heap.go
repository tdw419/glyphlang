package vm

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// heapBase is the starting address for the heap region.
// Set to 0x10000 (65536), leaving room for stack and globals below.
const heapBase uint32 = 0x10000

// headerSize is the size of a heap block header in bytes.
// Layout: [size:4 bytes][refcount:4 bytes][type_tag:2 bytes] = 10 bytes, aligned to 16.
const headerSize uint32 = 16

// Type tag constants for heap blocks.
const (
	TypeTagRaw    uint16 = 0 // Raw bytes / default
	TypeTagI32    uint16 = 1 // 4-byte signed integers
	TypeTagI64    uint16 = 2 // 8-byte signed integers
	TypeTagF64    uint16 = 3 // 8-byte floating point
	TypeTagPtr    uint16 = 4 // 8-byte pointer addresses
	TypeTagString uint16 = 5 // Variable-length string data
)

// typeTagElementSize returns the size in bytes of a single element for a given type tag.
func typeTagElementSize(tag uint16) uint32 {
	switch tag {
	case TypeTagI32:
		return 4
	case TypeTagI64, TypeTagF64, TypeTagPtr:
		return 8
	case TypeTagString:
		return 1 // byte-addressable
	default:
		return 8 // raw default: i64-sized slots
	}
}

// HeapBlock represents an allocated block on the heap.
type HeapBlock struct {
	Address  uint32 // Start address (where header begins)
	Size     uint32 // Data size requested by the user
	RefCount int    // Reference count
	TypeTag  uint16 // Type tag for the block (0 = raw bytes)
	Freed    bool   // Whether this block has been freed
	Data     []byte // Raw data storage
}

// PointerValue represents a pointer to a heap allocation.
type PointerValue struct {
	Address uint32
}

func (v PointerValue) Type() string { return "ptr" }

func (v PointerValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":    "ptr",
		"address": v.Address,
	})
}

// Heap manages dynamic memory allocations for a VM instance.
//
// Memory management uses reference counting. Every heap block starts with
// RefCount=1 on allocation. Storing a pointer into a TypeTagPtr block
// increments the target's refcount; overwriting a pointer slot decrements
// the old target's refcount. When a function frame is popped (execReturn),
// all PointerValues in the frame's local environment have their refcounts
// decremented. Blocks reaching refcount 0 are freed automatically.
//
// Known limitation: reference counting cannot collect cyclic data structures.
// If block A holds a pointer to block B, and B holds a pointer back to A,
// neither will ever reach refcount 0 even when no external references remain.
// GlyphLang programs should prefer acyclic data structures (trees, flat lists,
// etc.). A future mark-and-sweep garbage collection pass can be added to
// reclaim cycles if needed.
type Heap struct {
	blocks map[uint32]*HeapBlock // address -> block
	hp     uint32                // heap pointer (bump allocator)
	freeList []uint32            // addresses of freed blocks
}

// NewHeap creates a new heap instance.
func NewHeap() *Heap {
	return &Heap{
		blocks:   make(map[uint32]*HeapBlock),
		hp:       heapBase,
		freeList: make([]uint32, 0),
	}
}

// align8 rounds up to the next multiple of 8.
func align8(size uint32) uint32 {
	return (size + 7) & ^uint32(7)
}

// Alloc allocates a block of the given size on the heap.
// Returns the data address (header address + headerSize).
func (h *Heap) Alloc(size uint32) uint32 {
	totalSize := headerSize + align8(size)

	// Search free list for best fit
	bestIdx := -1
	bestSize := uint32(0)
	for i, addr := range h.freeList {
		block := h.blocks[addr]
		blockSize := headerSize + align8(block.Size)
		if blockSize >= totalSize {
			if bestIdx == -1 || blockSize < bestSize {
				bestIdx = i
				bestSize = blockSize
			}
		}
	}

	if bestIdx != -1 {
		// Reuse freed block
		addr := h.freeList[bestIdx]
		block := h.blocks[addr]
		block.Size = size
		block.RefCount = 1
		block.Freed = false
		if cap(block.Data) >= int(size) {
			block.Data = block.Data[:size]
		} else {
			block.Data = make([]byte, size)
		}
		// Remove from free list
		h.freeList = append(h.freeList[:bestIdx], h.freeList[bestIdx+1:]...)
		return addr + headerSize
	}

	// Bump allocate
	addr := h.hp
	h.hp += totalSize

	h.blocks[addr] = &HeapBlock{
		Address:  addr,
		Size:     size,
		RefCount: 1,
		TypeTag:  0,
		Freed:    false,
		Data:     make([]byte, size),
	}

	return addr + headerSize
}

// Free marks a heap block as freed. The data address (ptr) should be the
// value returned by Alloc (header address + headerSize).
func (h *Heap) Free(dataAddr uint32) error {
	headerAddr := dataAddr - headerSize

	block, exists := h.blocks[headerAddr]
	if !exists {
		return fmt.Errorf("invalid heap pointer: %d", dataAddr)
	}

	if block.Freed {
		return fmt.Errorf("double free at address %d", dataAddr)
	}

	block.Freed = true
	block.RefCount = 0
	block.TypeTag = 0

	// Try to coalesce with adjacent freed blocks
	h.coalesce(headerAddr)

	return nil
}

// coalesce merges adjacent freed blocks.
func (h *Heap) coalesce(addr uint32) {
	block, exists := h.blocks[addr]
	if !exists || !block.Freed {
		return
	}

	// Check the block immediately after this one
	nextAddr := addr + headerSize + align8(block.Size)
	if nextBlock, exists := h.blocks[nextAddr]; exists && nextBlock.Freed {
		// Merge: expand current block to cover next
		block.Size = nextAddr + align8(nextBlock.Size) - addr - headerSize
		delete(h.blocks, nextAddr)
		// Remove nextAddr from free list
		h.removeFreeList(nextAddr)
	}

	// Check the block immediately before this one
	for candidateAddr, candidate := range h.blocks {
		if candidate.Freed && candidateAddr < addr {
			endAddr := candidateAddr + headerSize + align8(candidate.Size)
			if endAddr == addr {
				// Merge: expand the predecessor to cover current
				candidate.Size = addr + align8(block.Size) - candidateAddr - headerSize
				delete(h.blocks, addr)
				h.removeFreeList(addr)
				// Keep candidate in free list
				return
			}
		}
	}

	// If not merged into predecessor, add to free list
	h.freeList = append(h.freeList, addr)
}

// removeFreeList removes an address from the free list.
func (h *Heap) removeFreeList(addr uint32) {
	for i, a := range h.freeList {
		if a == addr {
			h.freeList = append(h.freeList[:i], h.freeList[i+1:]...)
			return
		}
	}
}

// decrementRefcount decrements the reference count of the block at the given
// data address. If refcount reaches zero, the block is freed automatically.
func (h *Heap) decrementRefcount(dataAddr uint32) {
	block, err := h.GetBlock(dataAddr)
	if err != nil || block.Freed {
		return
	}
	block.RefCount--
	if block.RefCount <= 0 {
		h.Free(dataAddr)
	}
}

// ReleaseEnv scans an environment for heap pointers and decrements their
// refcounts. Blocks reaching refcount 0 are freed automatically.
// This should be called when a function frame is popped.
func (h *Heap) ReleaseEnv(env *Environment) {
	if env == nil {
		return
	}
	for _, val := range env.Values {
		if ptr, ok := val.(PointerValue); ok {
			h.decrementRefcount(ptr.Address)
		}
	}
}

// GetBlock returns the heap block for a given data address.
func (h *Heap) GetBlock(dataAddr uint32) (*HeapBlock, error) {
	headerAddr := dataAddr - headerSize
	block, exists := h.blocks[headerAddr]
	if !exists {
		return nil, fmt.Errorf("invalid heap pointer: %d", dataAddr)
	}
	return block, nil
}

// execAlloc implements OpAlloc: pops a size from the stack, allocates a heap block,
// and pushes a PointerValue onto the stack.
// Delegates to the syscall table's SysAlloc handler.
func (vm *VM) execAlloc() error {
	handler := syscallTable[SysAlloc]
	if handler == nil {
		return fmt.Errorf("ENOSYS: syscall 0x%02x not registered", SysAlloc)
	}
	result, err := handler(vm)
	if err != nil {
		return err
	}
	vm.Push(result)
	return nil
}

// execFree implements OpFree: pops a pointer from the stack and frees the
// corresponding heap block.
// Delegates to the syscall table's SysFree handler.
func (vm *VM) execFree() error {
	handler := syscallTable[SysFree]
	if handler == nil {
		return fmt.Errorf("ENOSYS: syscall 0x%02x not registered", SysFree)
	}
	result, err := handler(vm)
	if err != nil {
		return err
	}
	vm.Push(result)
	return nil
}

// Load reads a value from a heap block at the given byte offset.
// The value type is determined by the block's TypeTag.
func (h *Heap) Load(dataAddr uint32, offset uint32) (Value, error) {
	block, err := h.GetBlock(dataAddr)
	if err != nil {
		return nil, err
	}
	if block.Freed {
		return nil, fmt.Errorf("SEGFAULT: use-after-free at address %d", dataAddr)
	}

	elemSize := typeTagElementSize(block.TypeTag)
	if offset+elemSize > block.Size {
		return nil, fmt.Errorf("SEGFAULT: out-of-bounds read at offset %d (block size %d)", offset, block.Size)
	}

	switch block.TypeTag {
	case TypeTagI32:
		val := int32(binary.LittleEndian.Uint32(block.Data[offset : offset+4]))
		return IntValue{Val: int64(val)}, nil
	case TypeTagF64:
		bits := binary.LittleEndian.Uint64(block.Data[offset : offset+8])
		return FloatValue{Val: math.Float64frombits(bits)}, nil
	case TypeTagPtr:
		addr := binary.LittleEndian.Uint64(block.Data[offset : offset+8])
		return PointerValue{Address: uint32(addr)}, nil
	default:
		// TypeTagRaw / TypeTagI64: read 8-byte signed integer
		val := int64(binary.LittleEndian.Uint64(block.Data[offset : offset+8]))
		return IntValue{Val: val}, nil
	}
}

// Store writes a value to a heap block at the given byte offset.
// The value is encoded according to the block's TypeTag.
// If the value is a heap pointer, refcount is incremented for the new target.
func (h *Heap) Store(dataAddr uint32, offset uint32, value Value) error {
	block, err := h.GetBlock(dataAddr)
	if err != nil {
		return err
	}
	if block.Freed {
		return fmt.Errorf("SEGFAULT: use-after-free at address %d", dataAddr)
	}

	elemSize := typeTagElementSize(block.TypeTag)
	if offset+elemSize > block.Size {
		return fmt.Errorf("SEGFAULT: out-of-bounds write at offset %d (block size %d)", offset, block.Size)
	}

	switch block.TypeTag {
	case TypeTagI32:
		var intVal int64
		switch v := value.(type) {
		case IntValue:
			intVal = v.Val
		case FloatValue:
			intVal = int64(v.Val)
		default:
			return fmt.Errorf("store: cannot store %s into i32 block", value.Type())
		}
		binary.LittleEndian.PutUint32(block.Data[offset:offset+4], uint32(int32(intVal)))
	case TypeTagF64:
		var floatVal float64
		switch v := value.(type) {
		case IntValue:
			floatVal = float64(v.Val)
		case FloatValue:
			floatVal = v.Val
		default:
			return fmt.Errorf("store: cannot store %s into f64 block", value.Type())
		}
		binary.LittleEndian.PutUint64(block.Data[offset:offset+8], math.Float64bits(floatVal))
	case TypeTagPtr:
		ptr, ok := value.(PointerValue)
		if !ok {
			return fmt.Errorf("store: cannot store %s into ptr block", value.Type())
		}

		// Decrement refcount of the old value at this slot (if it was a valid pointer)
		oldAddr := binary.LittleEndian.Uint64(block.Data[offset : offset+8])
		if oldAddr != 0 {
			h.decrementRefcount(uint32(oldAddr))
		}

		binary.LittleEndian.PutUint64(block.Data[offset:offset+8], uint64(ptr.Address))
		// Increment refcount of the pointed-to block
		targetBlock, err := h.GetBlock(ptr.Address)
		if err == nil && !targetBlock.Freed {
			targetBlock.RefCount++
		}
	default:
		// TypeTagRaw / TypeTagI64: store as 8-byte signed integer
		var intVal int64
		switch v := value.(type) {
		case IntValue:
			intVal = v.Val
		case FloatValue:
			intVal = int64(v.Val)
		case BoolValue:
			if v.Val {
				intVal = 1
			}
		default:
			return fmt.Errorf("store: cannot store %s into raw/i64 block", value.Type())
		}
		binary.LittleEndian.PutUint64(block.Data[offset:offset+8], uint64(intVal))
	}
	return nil
}

// execLoadPtr implements OpLoadPtr: pops offset and pointer from the stack,
// reads a value from the heap block at ptr+offset, pushes the decoded value.
func (vm *VM) execLoadPtr() error {
	offsetVal, err := vm.Pop()
	if err != nil {
		return err
	}
	ptrVal, err := vm.Pop()
	if err != nil {
		return err
	}

	offsetInt, ok := offsetVal.(IntValue)
	if !ok {
		return fmt.Errorf("loadptr: offset must be integer, got %s", offsetVal.Type())
	}

	ptr, ok := ptrVal.(PointerValue)
	if !ok {
		return fmt.Errorf("loadptr: requires a pointer, got %s", ptrVal.Type())
	}

	if ptr.Address < heapBase+headerSize {
		return fmt.Errorf("SEGFAULT: invalid heap pointer %d", ptr.Address)
	}

	result, err := vm.heap.Load(ptr.Address, uint32(offsetInt.Val))
	if err != nil {
		return err
	}

	vm.Push(result)
	return nil
}

// execStorePtr implements OpStorePtr: pops value, offset, and pointer from the stack,
// encodes the value and writes it to the heap block at ptr+offset.
func (vm *VM) execStorePtr() error {
	value, err := vm.Pop()
	if err != nil {
		return err
	}
	offsetVal, err := vm.Pop()
	if err != nil {
		return err
	}
	ptrVal, err := vm.Pop()
	if err != nil {
		return err
	}

	offsetInt, ok := offsetVal.(IntValue)
	if !ok {
		return fmt.Errorf("storeptr: offset must be integer, got %s", offsetVal.Type())
	}

	ptr, ok := ptrVal.(PointerValue)
	if !ok {
		return fmt.Errorf("storeptr: requires a pointer, got %s", ptrVal.Type())
	}

	if ptr.Address < heapBase+headerSize {
		return fmt.Errorf("SEGFAULT: invalid heap pointer %d", ptr.Address)
	}

	return vm.heap.Store(ptr.Address, uint32(offsetInt.Val), value)
}
