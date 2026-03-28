package gpu

import (
	"fmt"
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
}

// NewMitosisVM creates a VM dispatcher with mitosis (S opcode) support.
func NewMitosisVM(maxThreads int) *MitosisVM {
	if maxThreads <= 0 {
		maxThreads = 256
	}
	return &MitosisVM{
		dispatcher: NewDispatcher(),
		maxThreads: maxThreads,
	}
}

// ThreadResult contains the result from one VM thread including spawn info.
type ThreadResult struct {
	ThreadID int
	ParentID int     // -1 for root thread
	Result   Result
	Children []int   // IDs of spawned child threads
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

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		results  []ThreadResult
		threadCount int
	)

	// Channel for spawn requests from running VMs
	spawns := make(chan spawnWork, m.maxThreads)
	done := make(chan struct{})

	// Worker that processes spawn requests
	go func() {
		for work := range spawns {
			wg.Add(1)
			go func(w spawnWork) {
				defer wg.Done()
				r := m.runThread(bytecode, config, w)
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}(work)
		}
		close(done)
	}()

	// Run root thread
	wg.Add(1)
	go func() {
		defer wg.Done()
		root := m.runMitosisThread(bytecode, config, 0, -1, 0, nil, nil, &threadCount, spawns)
		mu.Lock()
		results = append(results, root)
		mu.Unlock()
	}()

	// Wait for all threads to finish, then close spawn channel
	wg.Wait()
	close(spawns)
	<-done

	return results, nil
}

type spawnWork struct {
	threadID int
	parentID int
	startPC  uint32
	stack    []GpuValue
	vars     []GpuValue
}

// runThread executes a spawned thread.
func (m *MitosisVM) runThread(bytecode []byte, config *Config, w spawnWork) ThreadResult {
	// Create a modified VM that starts at the given PC with cloned state
	state := VMState{PC: w.startPC}
	stack := make([]GpuValue, MaxStack)
	copy(stack, w.stack)
	state.SP = uint32(len(w.stack))
	vars := make([]GpuValue, MaxVars)
	copy(vars, w.vars)

	base := int(config.CodeOffset)

	for state.Halted == 0 && state.Steps < MaxSteps {
		pc := int(state.PC)
		if base+pc >= len(bytecode) {
			state.Halted = 1
			break
		}

		op := bytecode[base+pc]

		// Handle S opcode (0xS0 reserved - using 0xC0 for Mitosis)
		if op == OpMitosis {
			// S opcode: pop offset, push thread ID
			if state.SP > 0 {
				state.SP--
				// In this simplified version, we just record the spawn
				// The actual spawn was handled during the mitosis thread run
			}
			state.PC++
			state.Steps++
			continue
		}

		// Delegate to standard execution
		result := m.dispatcher.runOneVM(bytecode, config, 0)
		return ThreadResult{
			ThreadID: w.threadID,
			ParentID: w.parentID,
			Result:   result,
		}
	}

	r := stateToResult(&state)
	return ThreadResult{
		ThreadID: w.threadID,
		ParentID: w.parentID,
		Result:   r,
	}
}

// OpMitosis is the S opcode for spawning parallel threads.
const OpMitosis byte = 0xC0

// runMitosisThread runs a VM with S opcode awareness, spawning children as needed.
func (m *MitosisVM) runMitosisThread(
	bytecode []byte,
	config *Config,
	threadID int,
	parentID int,
	startPC uint32,
	initStack []GpuValue,
	initVars []GpuValue,
	threadCount *int,
	spawns chan<- spawnWork,
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
			if *threadCount < m.maxThreads {
				// Clone stack and vars for child
				childStack := make([]GpuValue, state.SP)
				copy(childStack, stack[:state.SP])
				childVars := make([]GpuValue, MaxVars)
				copy(childVars, vars)

				spawns <- spawnWork{
					threadID: childID,
					parentID: threadID,
					startPC:  uint32(pc + 1 + int(offset.Data)),
					stack:    childStack,
					vars:     childVars,
				}
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
			constIdx := beUint32(bytecode[base+pc+1:])
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
			varIdx := beUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			stack[state.SP] = vars[varIdx%MaxVars]
			state.SP++

		case 0x41: // OP_STORE_VAR
			if base+pc+5 > len(bytecode) || state.SP == 0 {
				state.Halted = 1
				break
			}
			varIdx := beUint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			state.SP--
			vars[varIdx%MaxVars] = stack[state.SP]

		case 0x50: // OP_JUMP
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			nextPC = beUint32(bytecode[base+pc+1:])

		case 0x51: // OP_JUMP_IF_FALSE
			if base+pc+5 > len(bytecode) || state.SP == 0 {
				state.Halted = 1
				break
			}
			target := beUint32(bytecode[base+pc+1:])
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

func beUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}
