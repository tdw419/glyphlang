package vm

import (
	"encoding/json"
	"fmt"
)

// heapBase is the starting address for the heap region.
// Set to 0x10000 (65536), leaving room for stack and globals below.
const heapBase uint32 = 0x10000

// headerSize is the size of a heap block header in bytes.
// Layout: [size:4 bytes][refcount:4 bytes][type_tag:2 bytes] = 10 bytes, aligned to 16.
const headerSize uint32 = 16

// HeapBlock represents an allocated block on the heap.
type HeapBlock struct {
	Address  uint32 // Start address (where header begins)
	Size     uint32 // Data size requested by the user
	RefCount int    // Reference count
	TypeTag  uint16 // Type tag for the block (0 = raw bytes)
	Freed    bool   // Whether this block has been freed
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
func (vm *VM) execAlloc() error {
	sizeVal, err := vm.Pop()
	if err != nil {
		return err
	}

	sizeInt, ok := sizeVal.(IntValue)
	if !ok {
		return fmt.Errorf("alloc requires an integer size, got %s", sizeVal.Type())
	}

	if sizeInt.Val < 0 {
		return fmt.Errorf("alloc size must be positive, got %d", sizeInt.Val)
	}

	ptr := vm.heap.Alloc(uint32(sizeInt.Val))
	vm.hp = vm.heap.hp // sync HP register
	vm.Push(PointerValue{Address: ptr})
	return nil
}

// execFree implements OpFree: pops a pointer from the stack and frees the
// corresponding heap block.
func (vm *VM) execFree() error {
	ptrVal, err := vm.Pop()
	if err != nil {
		return err
	}

	ptr, ok := ptrVal.(PointerValue)
	if !ok {
		return fmt.Errorf("free requires a pointer, got %s", ptrVal.Type())
	}

	if ptr.Address < heapBase+headerSize {
		return fmt.Errorf("invalid heap pointer: %d", ptr.Address)
	}

	return vm.heap.Free(ptr.Address)
}
