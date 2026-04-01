package gpu

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

// Mitosis implements the S opcode: spawning parallel VM threads from a running VM.
// When a VM executes S(offset), it clones its state (stack, vars, PC) into a new
// VM thread that starts executing at PC + offset.

// SpawnRequest represents a request from a VM to spawn a child VM.
type SpawnRequest struct {
	ParentID int
	Offset   int // Spatial offset from parent's PC
}

// MitosisVM extends the basic Dispatcher with S opcode support.
type MitosisVM struct {
	dispatcher *Dispatcher
	maxThreads int
	nextID     atomic.Int64

	// ForceGPUError forces the GPU execution path to fail for testing.
	// When true, ExecuteWithMitosis will simulate a GPU failure, log a
	// warning, and fall back to CPU execution.
	ForceGPUError bool

	// fallbackWarnings collects warning messages from GPU fallback events.
	fallbackWarnings []string
	fallbackMu       sync.Mutex

	// logger is used for fallback warning output. Defaults to stdlib log.
	logger *log.Logger
}

// NewMitosisVM creates a VM dispatcher with mitosis (S opcode) support.
func NewMitosisVM(maxThreads int) *MitosisVM {
	if maxThreads <= 0 {
		maxThreads = 256
	}
	return &MitosisVM{
		dispatcher: NewDispatcher(),
		maxThreads: maxThreads,
		logger:     log.New(os.Stderr, "[mitosis] ", log.LstdFlags),
	}
}

// ThreadResult contains the result from one VM thread including spawn info.
type ThreadResult struct {
	ThreadID int
	ParentID int // -1 for root thread
	Result   Result
	Children []int // IDs of spawned child threads
}

// FallbackWarnings returns the list of GPU fallback warning messages collected
// during execution. Thread-safe.
func (m *MitosisVM) FallbackWarnings() []string {
	m.fallbackMu.Lock()
	defer m.fallbackMu.Unlock()
	out := make([]string, len(m.fallbackWarnings))
	copy(out, m.fallbackWarnings)
	return out
}

// logFallbackWarning records a fallback warning message and logs it.
func (m *MitosisVM) logFallbackWarning(msg string) {
	m.fallbackMu.Lock()
	m.fallbackWarnings = append(m.fallbackWarnings, msg)
	m.fallbackMu.Unlock()
	if m.logger != nil {
		m.logger.Print(msg)
	}
}

// ExecuteWithMitosis runs bytecode with S opcode support.
// Returns results from all threads (root + spawned children).
func (m *MitosisVM) ExecuteWithMitosis(bytecode []byte) ([]ThreadResult, error) {
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		return nil, err
	}

	// Attempt GPU execution first, fall back to CPU on failure.
	// This infrastructure supports issue #78 (full CPU fallback).
	if err := m.attemptGPUExecution(bytecode, config); err != nil {
		warning := fmt.Sprintf("GPU mitosis execution failed: %v; falling back to CPU", err)
		m.logFallbackWarning(warning)
	}

	return m.executeCPUFallback(bytecode, config), nil
}

// attemptGPUExecution tries to run the bytecode on GPU via the dispatcher.
// Returns nil if GPU execution succeeded, or an error describing why it failed.
// The GPU path does not yet support Mitosis child spawning — it only validates
// that the GPU is available and the bytecode is GPU-compatible.
func (m *MitosisVM) attemptGPUExecution(bytecode []byte, config *Config) error {
	// Forced failure for testing the fallback path
	if m.ForceGPUError {
		return fmt.Errorf("simulated GPU failure (ForceGPUError=true)")
	}

	// Check if bytecode is GPU-compatible
	if !IsGPUCompatible(bytecode) {
		return fmt.Errorf("bytecode contains opcodes not supported by GPU")
	}

	// Check if GPU is available via the dispatcher
	if !m.dispatcher.hasGPU {
		return fmt.Errorf("GPU not available (dispatcher.hasGPU=false)")
	}

	// GPU is available and bytecode is compatible.
	// Full GPU mitosis execution is not yet implemented (#78).
	// When implemented, this would dispatch child threads to GPU workgroups.
	return nil
}

// executeCPUFallback runs the full Mitosis execution on CPU.
func (m *MitosisVM) executeCPUFallback(bytecode []byte, config *Config) []ThreadResult {
	var (
		mu          sync.Mutex
		results     []ThreadResult
		threadCount int
	)

	// Collect spawn requests synchronously from the root thread,
	// then execute all children in parallel.
	var spawnRequests []spawnWork

	root := m.runMitosisThread(bytecode, config, 0, -1, 0, nil, nil,
		&threadCount, nil, &spawnRequests)

	mu.Lock()
	results = append(results, root)
	mu.Unlock()

	// Execute all children in parallel using a WaitGroup.
	// pending.Add is called before launching goroutines, so Wait won't return early.
	var pending sync.WaitGroup
	pending.Add(len(spawnRequests))

	for _, work := range spawnRequests {
		go func(w spawnWork) {
			defer pending.Done()
			r := m.runThread(bytecode, config, w)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(work)
	}

	pending.Wait()

	return results
}

type spawnWork struct {
	threadID int
	parentID int
	startPC  uint32
	stack    []GpuValue
	vars     []GpuValue
}

// runThread executes a spawned child thread.
// It reuses runMitosisThread with a nil spawns slice so the child
// cannot re-spawn further children.
func (m *MitosisVM) runThread(bytecode []byte, config *Config, w spawnWork) ThreadResult {
	// We pass nil for threadCount and spawns since children shouldn't
	// re-spawn in this simplified model. runMitosisThread handles nil
	// spawns gracefully by not attempting to collect spawn requests.
	var dummyThreadCount int
	return m.runMitosisThread(bytecode, config, w.threadID, w.parentID,
		w.startPC, w.stack, w.vars, &dummyThreadCount, nil, nil)
}

// OpMitosis is the S opcode for spawning parallel threads.
const OpMitosis byte = 0xC0

// runMitosisThread runs a VM with S opcode awareness, spawning children as needed.
// If spawnsOut is non-nil, spawn requests are appended to it (for synchronous collection).
// If spawnsOut is nil and spawns chan is non-nil, sends to the channel (legacy path).
func (m *MitosisVM) runMitosisThread(
	bytecode []byte,
	config *Config,
	threadID int,
	parentID int,
	startPC uint32,
	initStack []GpuValue,
	initVars []GpuValue,
	threadCount *int,
	_ chan<- spawnWork, // deprecated: channel-based spawns (kept for API compat)
	spawnsOut *[]spawnWork,
) ThreadResult {
	state := VMState{PC: startPC}
	stack := make([]GpuValue, MaxStack)
	if initStack != nil {
		copy(stack, initStack)
		state.SP = uint32(len(initStack))
	}
	vars := make([]GpuValue, MaxVars)
	if initVars != nil {
		copy(vars, initVars)
	}

	base := int(config.CodeOffset)
	var children []int

	for state.Halted == 0 && state.Steps < MaxSteps {
		pc := int(state.PC)
		if base+pc >= len(bytecode) {
			state.Halted = 1
			break
		}

		op := bytecode[base+pc]

		if op == OpMitosis {
			// S opcode: pop offset from stack, clone VM state, spawn child
			if state.SP == 0 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			state.SP--
			offset := stack[state.SP]

			*threadCount++
			childID := *threadCount
			if *threadCount < m.maxThreads && spawnsOut != nil {
				// Clone stack and vars for child
				childStack := make([]GpuValue, state.SP)
				copy(childStack, stack[:state.SP])
				childVars := make([]GpuValue, MaxVars)
				copy(childVars, vars)

				*spawnsOut = append(*spawnsOut, spawnWork{
					threadID: childID,
					parentID: threadID,
					startPC:  uint32(pc + 1 + int(offset.Data)),
					stack:    childStack,
					vars:     childVars,
				})
				children = append(children, childID)
			}

			// Push child thread ID onto parent's stack
			stack[state.SP] = GpuValue{TagInt, int32(childID)}
			state.SP++
			state.PC = uint32(pc + 1)
			state.Steps++
			continue
		}

		// Standard opcode execution (inline the critical path)
		nextPC := uint32(pc + 1)

		switch op {
		case 0x01: // OP_PUSH
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			constIdx := leUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			val := loadConstant(bytecode, config, constIdx)
			if state.SP >= MaxStack {
				state.Error = ErrStackOverflow
				state.Halted = 1
				break
			}
			stack[state.SP] = val
			state.SP++

		case 0x02: // OP_POP
			if state.SP == 0 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			state.SP--

		case 0x10: // OP_ADD
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data + b.Data}
			state.SP++

		case 0x11: // OP_SUB
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data - b.Data}
			state.SP++

		case 0x12: // OP_MUL
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data * b.Data}
			state.SP++

		case 0x13: // OP_DIV
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			if b.Data == 0 {
				state.Error = ErrDivByZero
				state.Halted = 1
				break
			}
			stack[state.SP] = GpuValue{TagInt, a.Data / b.Data}
			state.SP++

		case 0x40: // OP_LOAD_VAR
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			varIdx := leUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			stack[state.SP] = vars[varIdx%MaxVars]
			state.SP++

		case 0x41: // OP_STORE_VAR
			if base+pc+5 > len(bytecode) || state.SP == 0 {
				state.Halted = 1
				break
			}
			varIdx := leUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			state.SP--
			vars[varIdx%MaxVars] = stack[state.SP]

		case 0x50: // OP_JUMP
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			nextPC = leUint32(bytecode[base+pc+1:])

		case 0x51: // OP_JUMP_IF_FALSE
			if base+pc+5 > len(bytecode) || state.SP == 0 {
				state.Halted = 1
				break
			}
			target := leUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			state.SP--
			if stack[state.SP].Data == 0 {
				nextPC = target
			}

		case 0x61: // OP_RETURN
			if state.SP > 0 {
				val := stack[state.SP-1]
				state.ResultTag = val.Tag
				state.ResultData = val.Data
			}
			state.Halted = 1

		case 0xFF: // OP_HALT
			if state.SP > 0 {
				val := stack[state.SP-1]
				state.ResultTag = val.Tag
				state.ResultData = val.Data
			}
			state.Halted = 1

		default:
			// Skip unknown opcodes with operands
			if op >= 0x40 && op <= 0x52 {
				nextPC = uint32(pc + 5)
			}
		}

		state.PC = nextPC
		state.Steps++
	}

	r := stateToResult(&state)
	return ThreadResult{
		ThreadID: threadID,
		ParentID: parentID,
		Result:   r,
		Children: children,
	}
}

func leUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
