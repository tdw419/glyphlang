// Package gpu provides WebGPU compute shader execution for GlyphLang bytecode.
package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
			"sync"
		_ "embed"
)

//go:embed vm.wgsl
var shaderSource string

const (
	MaxStack      = 256
	MaxVars       = 64
	MaxVMs        = 4096
	MaxSteps      = 100000
	WorkgroupSize = 64
)

// Value tag constants matching WGSL shader
const (
	TagNull  uint32 = 0
	TagInt   uint32 = 1
	TagFloat uint32 = 2
	TagBool  uint32 = 3
)

// Error codes from GPU execution
const (
	ErrNone           uint32 = 0
	ErrStackOverflow  uint32 = 1
	ErrStackUnderflow uint32 = 2
	ErrDivByZero      uint32 = 3
	ErrMaxSteps       uint32 = 4
	ErrMutatorOOB     uint32 = 5 // mutator target out of bounds
)

// Exported opcode bytes for test construction.
const (
	OpMitosisByte byte = 0xC0
)

// ErrorString returns a human-readable description for a GPU error code.
func ErrorString(code uint32) string {
	switch code {
	case ErrNone:
		return "no error"
	case ErrStackOverflow:
		return "stack overflow"
	case ErrStackUnderflow:
		return "stack underflow"
	case ErrDivByZero:
		return "division by zero"
	case ErrMaxSteps:
		return "maximum steps exceeded"
	case ErrMutatorOOB:
		return "mutator target out of bounds"
	default:
		return fmt.Sprintf("unknown error code %d", code)
	}
}

// Config matches the WGSL Config struct layout
type Config struct {
	BytecodeLen     uint32
	NumConstants    uint32
	ConstantsOffset uint32
	CodeOffset      uint32
	NumVMs          uint32
	Pad1            uint32
	Pad2            uint32
	Pad3            uint32
}

// VMState matches the WGSL VMState struct layout
type VMState struct {
	PC         uint32
	SP         uint32
	Halted     uint32
	Error      uint32
	Steps      uint32
	ResultTag  uint32
	ResultData int32
	Pad        uint32
}

// GpuValue matches the WGSL GpuValue struct
type GpuValue struct {
	Tag  uint32
	Data int32
}

// Result represents the output of a single GPU VM execution
type Result struct {
	Tag      uint32
	IntVal   int64
	FloatVal float64
	BoolVal  bool
	Error    error
	Steps    uint32
}

// Dispatcher manages GPU compute shader execution.
type Dispatcher struct {
	mu     sync.Mutex
	shader string
	hasGPU bool
}

// NewDispatcher creates a GPU compute dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		shader: shaderSource,
		hasGPU: detectGPU(),
	}
}

// HasGPU reports whether the dispatcher detected GPU availability.
func (d *Dispatcher) HasGPU() bool {
	return d.hasGPU
}

// SetCPUFallback forces the dispatcher to use CPU execution.
func (d *Dispatcher) SetCPUFallback() {
	d.hasGPU = false
}

// ShaderSource returns the embedded WGSL shader source.
func (d *Dispatcher) ShaderSource() string {
	return d.shader
}

// Execute runs GLYP bytecode on multiple parallel VM instances.
func (d *Dispatcher) Execute(bytecode []byte, numVMs int) ([]Result, error) {
	if numVMs <= 0 || numVMs > MaxVMs {
		return nil, fmt.Errorf("numVMs must be 1-%d, got %d", MaxVMs, numVMs)
	}
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		return nil, err
	}
	config.NumVMs = uint32(numVMs)

	if d.hasGPU {
		return d.executeGPU(bytecode, config)
	}
	return d.executeCPU(bytecode, config)
}

// ExecuteOne runs bytecode on a single VM and returns the result.
func (d *Dispatcher) ExecuteOne(bytecode []byte) (*Result, error) {
	results, err := d.Execute(bytecode, 1)
	if err != nil {
		return nil, err
	}
	return &results[0], nil
}

func parseBytecodeLayout(bytecode []byte) (*Config, error) {
	if len(bytecode) < 16 {
		return nil, fmt.Errorf("bytecode too short")
	}

	offset := 4 // skip magic
	offset += 4 // skip version

	constCount := binary.LittleEndian.Uint32(bytecode[offset:])
	offset += 4
	constStart := uint32(offset)

	// Skip constants
	for i := uint32(0); i < constCount; i++ {
		ctype := bytecode[offset]
		offset++
		switch ctype {
		case 0x00: // null
		case 0x01, 0x02: // int64, float64
			offset += 8
		case 0x03: // bool
			offset++
		case 0x04: // string
			strLen := int(binary.LittleEndian.Uint32(bytecode[offset:]))
			offset += 4 + strLen
		}
	}

	// Read string pool count
	strPoolCount := binary.LittleEndian.Uint32(bytecode[offset:])
	offset += 4
	for i := uint32(0); i < strPoolCount; i++ {
		strLen := int(binary.LittleEndian.Uint32(bytecode[offset:]))
		offset += 4 + strLen
	}

	// Read instruction count
	offset += 4
	codeStart := uint32(offset)
	codeLen := uint32(len(bytecode)) - codeStart

	return &Config{
		BytecodeLen:     codeLen,
		NumConstants:    constCount,
		ConstantsOffset: constStart,
		CodeOffset:      codeStart,
	}, nil
}

func (d *Dispatcher) executeCPU(bytecode []byte, config *Config) ([]Result, error) {
	numVMs := int(config.NumVMs)
	allResults := make(map[int]Result)
	var resultsMu sync.Mutex

	type task struct {
		id    int
		pc    uint32
		stack []GpuValue
		vars  []GpuValue
	}

	queue := []task{}
	for i := 0; i < numVMs; i++ {
		queue = append(queue, task{
			id: i, pc: 0,
			stack: []GpuValue{},
			vars:  make([]GpuValue, MaxVars),
		})
	}

	nextID := numVMs
	for len(queue) > 0 && len(allResults) < 65536 {
		var nextQueue []task
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, t := range queue {
			wg.Add(1)
			go func(tsk task) {
				defer wg.Done()
				res, spawns := d.runOneVM(bytecode, config, tsk.id, tsk.pc, tsk.stack, tsk.vars)
				resultsMu.Lock()
				allResults[tsk.id] = res
				resultsMu.Unlock()

				if len(spawns) > 0 {
					mu.Lock()
					for _, s := range spawns {
						nextQueue = append(nextQueue, task{
							id:    nextID,
							pc:    s.PC,
							stack: s.Stack,
							vars:  s.Vars,
						})
						nextID++
					}
					mu.Unlock()
				}
			}(t)
		}
		wg.Wait()
		queue = nextQueue
	}

	results := make([]Result, len(allResults))
	for i := 0; i < len(allResults); i++ {
		results[i] = allResults[i]
	}
	return results, nil
}

func (d *Dispatcher) runOneVM(bytecode []byte, config *Config, vmID int, initialPC uint32, initialStack []GpuValue, initialVars []GpuValue) (Result, []CpuSpawnRequest) {
	state := VMState{PC: initialPC, SP: uint32(len(initialStack))}
	if state.PC == 0 {
		state.PC = config.CodeOffset
	}

	stack := make([]GpuValue, MaxStack)
	copy(stack, initialStack)
	vars := make([]GpuValue, MaxVars)
	copy(vars, initialVars)

	var spawns []CpuSpawnRequest

	for state.Halted == 0 && state.Steps < MaxSteps {
		pc := int(state.PC)
		if pc >= len(bytecode) {
			state.Halted = 1
			break
		}

		op := bytecode[pc]
		nextPC := uint32(pc + 1)

		switch op {
		case 0x01: // PUSH
			constIdx := binary.LittleEndian.Uint32(bytecode[pc+1:])
			nextPC = uint32(pc + 5)
			val := loadConstant(bytecode, config, constIdx)
			stack[state.SP] = val
			state.SP++
			break
		case 0x02: // POP
			state.SP--
			break
		case 0x10: // ADD
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data + b.Data}
			state.SP++
			break
		case 0x11: // SUB
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data - b.Data}
			state.SP++
			break
		case 0x12: // MUL
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			stack[state.SP] = GpuValue{TagInt, a.Data * b.Data}
			state.SP++
			break
		case 0x13: // DIV
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			if b.Data == 0 { state.Error = ErrDivByZero; state.Halted = 1; break }
			stack[state.SP] = GpuValue{TagInt, a.Data / b.Data}
			state.SP++
			break
		case 0x14: // MOD
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			if b.Data == 0 { state.Error = ErrDivByZero; state.Halted = 1; break }
			stack[state.SP] = GpuValue{TagInt, a.Data % b.Data}
			state.SP++
			break
		case 0x20: // EQ
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Tag == b.Tag && a.Data == b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x21: // NE
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Tag != b.Tag || a.Data != b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x22: // LT
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Data < b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x23: // GT
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Data > b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x24: // GE
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Data >= b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x25: // LE
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := 0; if a.Data <= b.Data { res = 1 }
			stack[state.SP] = GpuValue{TagBool, int32(res)}
			state.SP++
			break
		case 0x26: // AND
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := int32(0); if a.Data != 0 && b.Data != 0 { res = 1 }
			stack[state.SP] = GpuValue{TagBool, res}
			state.SP++
			break
		case 0x27: // OR
			b, a := stack[state.SP-1], stack[state.SP-2]
			state.SP -= 2
			res := int32(0); if a.Data != 0 || b.Data != 0 { res = 1 }
			stack[state.SP] = GpuValue{TagBool, res}
			state.SP++
			break
		case 0x28: // NOT
			a := stack[state.SP-1]
			state.SP--
			res := int32(0); if a.Data == 0 { res = 1 }
			stack[state.SP] = GpuValue{TagBool, res}
			state.SP++
			break
		case 0x29: // NEG
			a := stack[state.SP-1]
			state.SP--
			stack[state.SP] = GpuValue{a.Tag, -a.Data}
			state.SP++
			break
		case 0x40: // LOAD_VAR
			vidx := binary.LittleEndian.Uint32(bytecode[pc+1:])
			nextPC = uint32(pc + 5)
			stack[state.SP] = vars[vidx % MaxVars]
			state.SP++
			break
		case 0x41: // STORE_VAR
			vidx := binary.LittleEndian.Uint32(bytecode[pc+1:])
			nextPC = uint32(pc + 5)
			state.SP--
			vars[vidx % MaxVars] = stack[state.SP]
			break
		case 0x50: // JUMP — operand is relative to code section start
			nextPC = config.CodeOffset + binary.LittleEndian.Uint32(bytecode[pc+1:])
			break
		case 0x51: // JUMP_IF_FALSE — operand is relative to code section start
			target := binary.LittleEndian.Uint32(bytecode[pc+1:])
			state.SP--
			if stack[state.SP].Data == 0 { nextPC = config.CodeOffset + target } else { nextPC = uint32(pc + 5) }
			break
		case 0x61: // RETURN
			if state.SP > 0 {
				val := stack[state.SP-1]
				state.ResultTag, state.ResultData = val.Tag, val.Data
			}
			state.Halted = 1
			break
		case 0x62: // CALL
			nextPC = uint32(pc + 5)
			state.Error = 100 // Trap
			state.Halted = 1
			break
		case 0x73: // DEF_FUNC
			pCount := binary.LittleEndian.Uint32(bytecode[pc+5:])
			bLen := binary.LittleEndian.Uint32(bytecode[pc+9:])
			nextPC = uint32(pc) + 13 + (pCount * 4) + bLen
			break
		case 0xC0: // MITOSIS
			offset := stack[state.SP-1].Data
			state.SP--
			cStack := make([]GpuValue, state.SP)
			copy(cStack, stack[:state.SP])
			cVars := make([]GpuValue, MaxVars)
			copy(cVars, vars)
			cStackWithResult := append(cStack, GpuValue{TagBool, 0})
			spawns = append(spawns, CpuSpawnRequest{
				Offset: int32(offset),
				PC:     uint32(pc),
				Stack:  cStackWithResult,
				Vars:   cVars,
			})
			stack[state.SP] = GpuValue{TagBool, 1}
			state.SP++
			break
                case 0xC1: // MUTATOR
                        val := stack[state.SP-2].Data
                        offset := stack[state.SP-1].Data
                        state.SP -= 2
                        target := int(pc) + int(offset)
                        if target >= 0 && target < len(bytecode) {
                                bytecode[target] = byte(val)
                        } else {
                                state.Error = ErrMutatorOOB
                                state.Halted = 1
                        }
                        break
		case 0xC2: // TELEMETRY
			state.SP--
			break
		case 0xFF: // HALT
			if state.SP > 0 {
				val := stack[state.SP-1]
				state.ResultTag, state.ResultData = val.Tag, val.Data
			}
			state.Halted = 1
			break
		default:
			state.Halted = 1
			break
		}
		state.PC = nextPC
		state.Steps++
	}
	return stateToResult(&state), spawns
}

func loadConstant(bytecode []byte, config *Config, idx uint32) GpuValue {
	offset := int(config.ConstantsOffset)
	for i := uint32(0); i < idx; i++ {
		ctype := bytecode[offset]
		offset++
		switch ctype {
		case 0x01, 0x02: offset += 8
		case 0x03: offset++
		case 0x04: strLen := int(binary.LittleEndian.Uint32(bytecode[offset:])); offset += 4 + strLen
		}
	}
	ctype := bytecode[offset]
	offset++
	switch ctype {
	case 0x01: return GpuValue{TagInt, int32(binary.LittleEndian.Uint64(bytecode[offset:]))}
	case 0x02: bits := binary.LittleEndian.Uint64(bytecode[offset:]); return GpuValue{TagFloat, int32(math.Float32bits(float32(math.Float64frombits(bits))))}
	case 0x03: return GpuValue{TagBool, int32(bytecode[offset])}
	}
	return GpuValue{TagNull, 0}
}

func stateToResult(state *VMState) Result {
	r := Result{Tag: state.ResultTag, Steps: state.Steps}
	if state.Error != 0 { r.Error = fmt.Errorf("error %d", state.Error) }
	switch state.ResultTag {
	case TagInt: r.IntVal = int64(state.ResultData)
	case TagFloat: r.FloatVal = float64(math.Float32frombits(uint32(state.ResultData)))
	case TagBool: r.BoolVal = state.ResultData != 0
	}
	return r
}

func detectGPU() bool {
	if os.Getenv("GLYPH_NO_GPU") != "" { return false }
	return false // Stub for now
}

type CpuSpawnRequest struct {
	Offset int32
	PC     uint32
	Stack  []GpuValue
	Vars   []GpuValue
}

// IsGPUCompatible checks if the bytecode contains only opcodes supported by the GPU backend.
func IsGPUCompatible(bytecode []byte) bool {
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return false
	}

	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		return false
	}

	code := bytecode[config.CodeOffset:]
	for i := 0; i < len(code); {
		op := code[i]

		// Check if opcode is supported
		supported := false
		switch op {
		case 0x00, 0x01, 0x02, 0x10, 0x11, 0x12, 0x13, 0x14, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x40, 0x41, 0x50, 0x51, 0x52, 0x56, 0x57, 0x61, 0x62, 0x70, 0x71, 0x72, 0x73, 0x80, 0xC0, 0xC1, 0xC2, 0xFF:
			supported = true
		}

		if !supported {
			return false
		}

		// Advance i based on opcode operands
		if hasOperand(op) {
			i += 5
		} else if op == 0x73 { // DEF_FUNC
			if i+13 > len(code) { return false }
			pCount := binary.LittleEndian.Uint32(code[i+5:])
			bLen := binary.LittleEndian.Uint32(code[i+9:])
			i += 13 + int(pCount)*4 + int(bLen)
		} else {
			i += 1
		}
	}

	return true
}

func hasOperand(op byte) bool {
	withOperand := map[byte]bool{
		0x01: true, 0x40: true, 0x41: true, 0x50: true, 0x51: true, 0x52: true, 0x62: true, 0x70: true, 0x80: true, 0xB0: true,
	}
	return withOperand[op]
}

// ParseBytecodeLayoutPublic is the exported version of parseBytecodeLayout.
func ParseBytecodeLayoutPublic(bytecode []byte) (*Config, error) {
	return parseBytecodeLayout(bytecode)
}
