// Package gpu provides WebGPU compute shader execution for GlyphLang bytecode.
//
// The GPU backend runs GLYP bytecode on GPU compute units, enabling massive
// parallelism for request handling. Each workgroup thread executes an independent
// VM instance with its own stack and variable space.
//
// Architecture:
//
//	GLYP bytecode → GPU storage buffer → WGSL compute shader → results readback
//
// The compute shader (vm.wgsl) implements the same opcode set as pkg/vm/vm.go,
// supporting arithmetic, comparisons, variables, control flow, and stack ops.
package gpu

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

//go:embed vm.wgsl
var shaderSource string

const (
	MaxStack   = 256
	MaxVars    = 64
	MaxVMs     = 4096
	MaxSteps   = 100000
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
	ErrNone          uint32 = 0
	ErrStackOverflow uint32 = 1
	ErrStackUnderflow uint32 = 2
	ErrDivByZero     uint32 = 3
	ErrMaxSteps      uint32 = 4
)

// Config matches the WGSL Config struct layout (8 x u32 = 32 bytes)
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

// VMState matches the WGSL VMState struct layout (8 x u32 = 32 bytes)
type VMState struct {
	PC        uint32
	SP        uint32
	Halted    uint32
	Error     uint32
	Steps     uint32
	ResultTag uint32
	ResultData int32
	Pad       uint32
}

// GpuValue matches the WGSL GpuValue struct (tag u32 + data i32 = 8 bytes)
type GpuValue struct {
	Tag  uint32
	Data int32
}

// Result represents the output of a single GPU VM execution
type Result struct {
	Tag    uint32
	IntVal int64
	FloatVal float64
	BoolVal bool
	Error  error
	Steps  uint32
}

// Dispatcher manages GPU compute shader execution.
// In environments without WebGPU, it falls back to CPU simulation.
type Dispatcher struct {
	mu       sync.Mutex
	shader   string
	hasGPU   bool
}

// NewDispatcher creates a GPU compute dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		shader: shaderSource,
		hasGPU: false, // CPU fallback until wgpu-native bindings are integrated
	}
}

// ShaderSource returns the embedded WGSL shader source.
func (d *Dispatcher) ShaderSource() string {
	return d.shader
}

// Execute runs GLYP bytecode on multiple parallel VM instances.
// Each VM executes the same bytecode independently (SIMD-style).
// Returns one Result per VM instance.
func (d *Dispatcher) Execute(bytecode []byte, numVMs int) ([]Result, error) {
	if numVMs <= 0 || numVMs > MaxVMs {
		return nil, fmt.Errorf("numVMs must be 1-%d, got %d", MaxVMs, numVMs)
	}
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	// Parse the bytecode to find constant pool and code boundaries
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

// ParseBytecodeLayoutPublic is the exported version of parseBytecodeLayout.
func ParseBytecodeLayoutPublic(bytecode []byte) (*Config, error) {
	return parseBytecodeLayout(bytecode)
}

// parseBytecodeLayout reads the GLYP bytecode format to determine offsets.
// Format: "GLYP" + version(4 LE) + constCount(4 LE) + constants... + instrCount(4 LE) + code...
func parseBytecodeLayout(bytecode []byte) (*Config, error) {
	if len(bytecode) < 16 {
		return nil, fmt.Errorf("bytecode too short: need at least 16 bytes, got %d", len(bytecode))
	}

	offset := 4 // skip GLYP magic

	// Read version (4 bytes, little-endian)
	// version := binary.LittleEndian.Uint32(bytecode[offset:])
	offset += 4

	// Read constant count (4 bytes, little-endian)
	constCount := binary.LittleEndian.Uint32(bytecode[offset:])
	offset += 4

	constStart := uint32(offset)

	// Skip over constants
	for i := uint32(0); i < constCount; i++ {
		if offset >= len(bytecode) {
			return nil, fmt.Errorf("truncated constant pool at index %d", i)
		}
		ctype := bytecode[offset]
		offset++
		switch ctype {
		case 0x00: // null — no data
		case 0x01: // int64
			if offset+8 > len(bytecode) {
				return nil, fmt.Errorf("truncated int constant at index %d", i)
			}
			offset += 8
		case 0x02: // float64
			if offset+8 > len(bytecode) {
				return nil, fmt.Errorf("truncated float constant at index %d", i)
			}
			offset += 8
		case 0x03: // bool
			if offset+1 > len(bytecode) {
				return nil, fmt.Errorf("truncated bool constant at index %d", i)
			}
			offset++
		case 0x04: // string
			if offset+4 > len(bytecode) {
				return nil, fmt.Errorf("truncated string length at index %d", i)
			}
			strLen := int(binary.LittleEndian.Uint32(bytecode[offset:]))
			offset += 4
			if offset+strLen > len(bytecode) {
				return nil, fmt.Errorf("truncated string data at index %d", i)
			}
			offset += strLen
		default:
			return nil, fmt.Errorf("unknown constant type 0x%02x at index %d", ctype, i)
		}
	}

	// Read instruction count (4 bytes, little-endian)
	if offset+4 > len(bytecode) {
		return nil, fmt.Errorf("missing instruction count")
	}
	// instrCount := binary.LittleEndian.Uint32(bytecode[offset:])
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

// executeCPU simulates the GPU compute shader on the CPU.
// This matches the WGSL shader behavior exactly for testing and fallback.
func (d *Dispatcher) executeCPU(bytecode []byte, config *Config) ([]Result, error) {
	numVMs := int(config.NumVMs)
	results := make([]Result, numVMs)

	var wg sync.WaitGroup
	wg.Add(numVMs)

	for i := 0; i < numVMs; i++ {
		go func(vmID int) {
			defer wg.Done()
			results[vmID] = d.runOneVM(bytecode, config, vmID)
		}(i)
	}

	wg.Wait()
	return results, nil
}

// runOneVM executes bytecode on a single simulated GPU VM instance.
func (d *Dispatcher) runOneVM(bytecode []byte, config *Config, vmID int) Result {
	state := VMState{}
	stack := make([]GpuValue, MaxStack)
	vars := make([]GpuValue, MaxVars)

	base := int(config.CodeOffset)

	for state.Halted == 0 && state.Steps < MaxSteps {
		pc := int(state.PC)
		if base+pc >= len(bytecode) {
			state.Halted = 1
			break
		}

		op := bytecode[base+pc]
		nextPC := uint32(pc + 1)

		switch op {
		case 0x01: // OP_PUSH
			if base+pc+5 > len(bytecode) {
				state.Error = ErrStackOverflow
				state.Halted = 1
				break
			}
			constIdx := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
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
			if a.Tag == TagFloat || b.Tag == TagFloat {
				af := math.Float32frombits(uint32(a.Data))
				bf := math.Float32frombits(uint32(b.Data))
				stack[state.SP] = GpuValue{TagFloat, int32(math.Float32bits(af + bf))}
			} else {
				stack[state.SP] = GpuValue{TagInt, a.Data + b.Data}
			}
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

		case 0x14: // OP_MOD
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
			stack[state.SP] = GpuValue{TagInt, a.Data % b.Data}
			state.SP++

		case 0x20: // OP_EQ
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data == b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x21: // OP_NE
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data != b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x22: // OP_LT
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data < b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x23: // OP_GT
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data > b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x24: // OP_GE
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data >= b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x25: // OP_LE
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data <= b.Data {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x26: // OP_AND
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data != 0 && b.Data != 0 {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x27: // OP_OR
			if state.SP < 2 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			b := stack[state.SP-1]
			a := stack[state.SP-2]
			state.SP -= 2
			r := int32(0)
			if a.Data != 0 || b.Data != 0 {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x28: // OP_NOT
			if state.SP < 1 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			a := stack[state.SP-1]
			state.SP--
			r := int32(0)
			if a.Data == 0 {
				r = 1
			}
			stack[state.SP] = GpuValue{TagBool, r}
			state.SP++

		case 0x29: // OP_NEG
			if state.SP < 1 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			a := stack[state.SP-1]
			state.SP--
			stack[state.SP] = GpuValue{a.Tag, -a.Data}
			state.SP++

		case 0x40: // OP_LOAD_VAR
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			varIdx := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			vidx := varIdx % MaxVars
			val := vars[vidx]
			if state.SP >= MaxStack {
				state.Error = ErrStackOverflow
				state.Halted = 1
				break
			}
			stack[state.SP] = val
			state.SP++

		case 0x41: // OP_STORE_VAR
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			varIdx := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			if state.SP == 0 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			state.SP--
			vars[varIdx%MaxVars] = stack[state.SP]

		case 0x50: // OP_JUMP
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			target := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
			nextPC = target

		case 0x51: // OP_JUMP_IF_FALSE
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			target := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			if state.SP == 0 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			state.SP--
			if stack[state.SP].Data == 0 {
				nextPC = target
			}

		case 0x52: // OP_JUMP_IF_TRUE
			if base+pc+5 > len(bytecode) {
				state.Halted = 1
				break
			}
			target := binary.LittleEndian.Uint32(bytecode[base+pc+1:])
			nextPC = uint32(pc + 5)
			if state.SP == 0 {
				state.Error = ErrStackUnderflow
				state.Halted = 1
				break
			}
			state.SP--
			if stack[state.SP].Data != 0 {
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
			// Skip unknown opcodes - try to advance past operands
			// Opcodes 0x40-0x52 have 4-byte operands, others have none
			if op >= 0x40 && op <= 0x52 {
				nextPC = uint32(pc + 5)
			}
		}

		state.PC = nextPC
		state.Steps++
	}

	if state.Steps >= MaxSteps {
		state.Error = ErrMaxSteps
	}

	return stateToResult(&state)
}

// loadConstant reads a constant from the bytecode constant pool.
func loadConstant(bytecode []byte, config *Config, idx uint32) GpuValue {
	offset := int(config.ConstantsOffset)

	for i := uint32(0); i < idx && offset < len(bytecode); i++ {
		ctype := bytecode[offset]
		offset++
		switch ctype {
		case 0x00: // null
		case 0x01: // int64
			offset += 8
		case 0x02: // float64
			offset += 8
		case 0x03: // bool
			offset++
		case 0x04: // string
			if offset+4 > len(bytecode) {
				return GpuValue{TagNull, 0}
			}
			strLen := int(binary.LittleEndian.Uint32(bytecode[offset:]))
			offset += 4 + strLen
		default:
			return GpuValue{TagNull, 0}
		}
	}

	if offset >= len(bytecode) {
		return GpuValue{TagNull, 0}
	}

	ctype := bytecode[offset]
	offset++

	switch ctype {
	case 0x00:
		return GpuValue{TagNull, 0}
	case 0x01: // int64
		if offset+8 > len(bytecode) {
			return GpuValue{TagNull, 0}
		}
		val := int64(binary.LittleEndian.Uint64(bytecode[offset:]))
		return GpuValue{TagInt, int32(val)}
	case 0x02: // float64
		if offset+8 > len(bytecode) {
			return GpuValue{TagNull, 0}
		}
		bits := binary.LittleEndian.Uint64(bytecode[offset:])
		f := math.Float64frombits(bits)
		return GpuValue{TagFloat, int32(math.Float32bits(float32(f)))}
	case 0x03: // bool
		if offset >= len(bytecode) {
			return GpuValue{TagNull, 0}
		}
		return GpuValue{TagBool, int32(bytecode[offset])}
	case 0x04: // string — return index as reference
		return GpuValue{TagInt, int32(idx)}
	}

	return GpuValue{TagNull, 0}
}

func stateToResult(state *VMState) Result {
	r := Result{
		Tag:   state.ResultTag,
		Steps: state.Steps,
	}

	switch state.Error {
	case ErrNone:
		// ok
	case ErrStackOverflow:
		r.Error = fmt.Errorf("stack overflow")
	case ErrStackUnderflow:
		r.Error = fmt.Errorf("stack underflow")
	case ErrDivByZero:
		r.Error = fmt.Errorf("division by zero")
	case ErrMaxSteps:
		r.Error = fmt.Errorf("max execution steps exceeded (%d)", MaxSteps)
	default:
		r.Error = fmt.Errorf("GPU VM error code %d", state.Error)
	}

	switch state.ResultTag {
	case TagInt:
		r.IntVal = int64(state.ResultData)
	case TagFloat:
		r.FloatVal = float64(math.Float32frombits(uint32(state.ResultData)))
	case TagBool:
		r.BoolVal = state.ResultData != 0
	}

	return r
}

// executeGPU runs bytecode on actual WebGPU hardware.
// This is a placeholder for when wgpu-native bindings are integrated.
func (d *Dispatcher) executeGPU(bytecode []byte, config *Config) ([]Result, error) {
	// TODO: Integrate wgpu-native bindings
	// 1. Create GPU device and command encoder
	// 2. Create storage buffers for config, bytecode, vm_states, stacks, vars
	// 3. Upload bytecode and config
	// 4. Create compute pipeline from vm.wgsl
	// 5. Dispatch ceil(numVMs / 64) workgroups
	// 6. Read back vm_states buffer
	// 7. Convert VMState results to Result slice
	return d.executeCPU(bytecode, config)
}

// ErrorString returns a human-readable error description.
func ErrorString(code uint32) string {
	switch code {
	case ErrNone:
		return "ok"
	case ErrStackOverflow:
		return "stack overflow"
	case ErrStackUnderflow:
		return "stack underflow"
	case ErrDivByZero:
		return "division by zero"
	case ErrMaxSteps:
		return "max execution steps exceeded"
	default:
		return fmt.Sprintf("unknown error %d", code)
	}
}
