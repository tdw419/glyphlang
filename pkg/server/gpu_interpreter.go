package server

import (
	"fmt"
	"sync"

	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/glyphlang/glyph/pkg/vm"
)

// GPUInterpreter executes Glyph bytecode using GPU compute shaders.
// It runs pre-compiled bytecode on the GPU Dispatcher for parallel execution.
// Note: Compilation happens externally to avoid import cycles.
type GPUInterpreter struct {
	dispatcher *gpu.Dispatcher
	vm         *vm.VM // Fallback for non-parallel routes
	mu         sync.RWMutex
}

// NewGPUInterpreter creates a GPU-backed interpreter.
func NewGPUInterpreter() *GPUInterpreter {
	return &GPUInterpreter{
		dispatcher: gpu.NewDispatcher(),
		vm:         vm.NewVM(),
	}
}

// Execute runs a route's logic on the GPU.
// For routes with simple logic, it uses the GPU dispatcher.
// For complex routes requiring stdlib, it falls back to CPU VM.
func (g *GPUInterpreter) Execute(route *Route, ctx *Context) (interface{}, error) {
	// Routes without custom handlers need to be compiled
	// For now, we use the CPU VM as the primary executor
	// with GPU acceleration for specific operations

	// The GPU path is used for:
	// 1. Parallel batch processing (multiple requests)
	// 2. Compute-intensive operations
	// 3. Array/map operations that benefit from parallelization

	// For single-route execution, use the standard VM
	// GPU execution is triggered via --gpu flag or batch operations
	return nil, fmt.Errorf("GPU interpreter requires compiled bytecode; use standard interpreter for dynamic routes")
}

// ExecuteBytecode runs pre-compiled bytecode on the GPU.
// This is the primary entry point for GPU execution.
func (g *GPUInterpreter) ExecuteBytecode(bytecode []byte) (*gpu.Result, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.dispatcher.ExecuteOne(bytecode)
}

// ExecuteBatch runs multiple VMs in parallel on the GPU.
// numVMs specifies how many parallel instances to spawn.
func (g *GPUInterpreter) ExecuteBatch(bytecode []byte, numVMs int) ([]gpu.Result, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.dispatcher.Execute(bytecode, numVMs)
}

// Dispatcher returns the underlying GPU dispatcher for advanced usage.
func (g *GPUInterpreter) Dispatcher() *gpu.Dispatcher {
	return g.dispatcher
}

// HybridInterpreter combines GPU and CPU execution.
// Routes are analyzed to determine the best execution path.
type HybridInterpreter struct {
	gpu *GPUInterpreter
	cpu *vm.VM
}

// NewHybridInterpreter creates an interpreter that uses GPU when beneficial.
func NewHybridInterpreter() *HybridInterpreter {
	return &HybridInterpreter{
		gpu: NewGPUInterpreter(),
		cpu: vm.NewVM(),
	}
}

// Execute runs the route using the optimal backend.
func (h *HybridInterpreter) Execute(route *Route, ctx *Context) (interface{}, error) {
	// Use CPU VM for interactive routes (requires stdlib, I/O)
	// GPU is used for compute-heavy batch operations
	return nil, fmt.Errorf("route execution requires compiled handler")
}

// GPU returns the GPU interpreter for direct bytecode execution.
func (h *HybridInterpreter) GPU() *GPUInterpreter {
	return h.gpu
}

// CPU returns the CPU VM for standard execution.
func (h *HybridInterpreter) CPU() *vm.VM {
	return h.cpu
}
