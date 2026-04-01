package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// Opcode represents a bytecode operation
type Opcode byte

const (
	OpPush        Opcode = 0x01
	OpPop         Opcode = 0x02
	OpAdd         Opcode = 0x10
	OpSub         Opcode = 0x11
	OpMul         Opcode = 0x12
	OpDiv         Opcode = 0x13
	OpMod         Opcode = 0x14
	OpEq          Opcode = 0x20
	OpNe          Opcode = 0x21
	OpLt          Opcode = 0x22
	OpGt          Opcode = 0x23
	OpGe          Opcode = 0x24
	OpLe          Opcode = 0x25
	OpAnd         Opcode = 0x26
	OpOr          Opcode = 0x27
	OpNot         Opcode = 0x28
	OpNeg         Opcode = 0x29 // Unary negation (-)
	OpLoadVar     Opcode = 0x40
	OpStoreVar    Opcode = 0x41
	OpJump        Opcode = 0x50
	OpJumpIfFalse Opcode = 0x51
	OpJumpIfTrue  Opcode = 0x52
	OpGetIter     Opcode = 0x53
	OpIterNext    Opcode = 0x54
	OpIterHasNext Opcode = 0x55
	OpGetIndex    Opcode = 0x56
	OpSetIndex    Opcode = 0x57 // Set array/map element: arr[idx] = val
	OpReturn      Opcode = 0x61
	OpCall        Opcode = 0x62
	OpBuildObject Opcode = 0x70
	OpGetField    Opcode = 0x71
	OpSetField    Opcode = 0x72 // Set object field: obj.field = val
	OpDefFunc     Opcode = 0x73 // Define a function: name_idx, param_count, body_length, [param_name_idxs...], body...
	OpBuildArray  Opcode = 0x80
	OpLoadString  Opcode = 0x91 // Load string from string pool by index
	OpHttpReturn  Opcode = 0x90

	// WebSocket opcodes
	OpWsSend          Opcode = 0xA0 // Send message to current connection
	OpWsBroadcast     Opcode = 0xA1 // Broadcast to all connections
	OpWsBroadcastRoom Opcode = 0xA2 // Broadcast to room
	OpWsJoinRoom      Opcode = 0xA3 // Join a room
	OpWsLeaveRoom     Opcode = 0xA4 // Leave a room
	OpWsClose         Opcode = 0xA5 // Close connection
	OpWsGetRooms      Opcode = 0xA6 // Get list of rooms
	OpWsGetClients    Opcode = 0xA7 // Get clients in room
	OpWsGetConnCount  Opcode = 0xA8 // Get total connection count
	OpWsGetUptime     Opcode = 0xA9 // Get server uptime in seconds

	// Async/await opcodes
	OpAsync Opcode = 0xB0 // Create async future (operand: body length)
	OpAwait Opcode = 0xB1 // Await a future
	OpTry   Opcode = 0xB2 // Try expression - unwrap result, propagate error

	// GPU/Parallel opcodes
	OpMitosis Opcode = 0xC0 // S opcode: clone VM state into new thread at spatial offset
	OpMutator Opcode = 0xC1 // M opcode: self-modify code at IP + offset
	OpTelemetry Opcode = 0xC2 // write to vm_stats[slot]

	// Process lifecycle opcodes
	OpSpawn Opcode = 0xC3 // Fork: create child VM, assign PID
	OpKill  Opcode = 0xC4 // Terminate process by PID
	OpWait  Opcode = 0xC5 // Block until target process becomes zombie, reap it

	OpHalt Opcode = 0xFF
)

// BuiltinFunc represents a built-in function
type BuiltinFunc func(args []Value) (Value, error)

// Iterator represents an iterator over a collection
type Iterator struct {
	collection Value
	index      int
	keys       []string // for object iteration
}

// WebSocketHandler defines the interface for WebSocket operations
// This allows the VM to be decoupled from the actual WebSocket implementation
type WebSocketHandler interface {
	Send(message interface{}) error
	Broadcast(message interface{}) error
	BroadcastToRoom(room string, message interface{}) error
	JoinRoom(room string) error
	LeaveRoom(room string) error
	Close(reason string) error
	GetRooms() []string
	GetRoomClients(room string) []string
	GetConnectionID() string
	GetConnectionCount() int
	GetUptime() int64 // uptime in seconds
}

// CallFrame saves the state when entering a function call.
type CallFrame struct {
	returnPC        int              // Program counter to return to
	returnBytecode  []byte           // Bytecode buffer to return to
	returnConstants []Value          // Constant pool to return to
	env             *Environment     // Lexical environment
}


// Clone creates a deep copy of the VM state for parallel execution (mitosis).
func (vm *VM) Clone() *VM {
	newVM := &VM{
		env:        vm.env.Clone(), // TODO: Deep copy environment if needed
		globals:    vm.globals,
		constants:  vm.constants,
		stringPool: vm.stringPool,
		builtins:   vm.builtins,
		functions:  vm.functions,
		pc:         vm.pc,
		code:       vm.code,
		halted:     vm.halted,
		maxSteps:   vm.maxSteps,
		profiler:   vm.profiler,
		nextIterID: vm.nextIterID,
	}

	// Deep copy stack
	newVM.stack = make([]Value, len(vm.stack))
	copy(newVM.stack, vm.stack)

	// Deep copy call stack
	newVM.callStack = make([]CallFrame, len(vm.callStack))
	copy(newVM.callStack, vm.callStack)

	// Deep copy iterators
	newVM.iterators = make(map[int]*Iterator)
	for k, v := range vm.iterators {
		newVM.iterators[k] = &Iterator{
			collection: v.collection,
			index:      v.index,
			keys:       v.keys,
		}
	}

	return newVM
}

// VM represents the virtual machine
type VM struct {
	stack      []Value
	env        *Environment
	globals    map[string]Value
	constants  []Value
	stringPool []string // String pool for OP_LOAD_STRING
	builtins   map[string]BuiltinFunc
	functions  map[string]FunctionValue // User-defined functions
	callStack  []CallFrame              // Call frame stack
	iterators  map[int]*Iterator        // track iterators by ID
	nextIterID int
	pc         int // program counter
	code       []byte
	halted     bool

	// WebSocket context (set when executing WebSocket handlers)
	wsHandler WebSocketHandler

	// Maximum number of execution steps (0 = unlimited)
	maxSteps int

	// JIT profiler for hot path optimization
	profiler ProfileRecorder

	// Process model: PID of this VM instance (0 = unregistered)
	pid uint32

	// Process table for lifecycle management (nil = no process model)
	processTable *ProcessTable
}

// SetProfiler sets the profiler for this VM.
func (vm *VM) SetProfiler(p ProfileRecorder) {
	vm.profiler = p
}

// NewVM creates a new virtual machine
func NewVM() *VM {
	vm := &VM{
		stack:      make([]Value, 0, 256),
		env:        NewEnvironment(nil),
		globals:    make(map[string]Value),
		constants:  make([]Value, 0),
		stringPool: make([]string, 0),
		builtins:   make(map[string]BuiltinFunc),
		functions:  make(map[string]FunctionValue),
		callStack:  make([]CallFrame, 0, 16),
		iterators:  make(map[int]*Iterator),
		nextIterID: 0,
		pc:         0,
		halted:     false,
		profiler:   NewSimpleProfiler(),
	}
	vm.registerBuiltins()
	return vm
}

// Execute runs bytecode
func (vm *VM) Execute(bytecode []byte) (Value, error) {
	start := time.Now()
	defer func() {
		if vm.profiler != nil {
			vm.profiler.Record("vm_execute", time.Since(start))
		}
	}()

	if len(bytecode) < 4 {
		return nil, fmt.Errorf("invalid bytecode: too short")
	}

	// Verify magic bytes
	if string(bytecode[0:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid bytecode: bad magic bytes")
	}

	// Parse bytecode
	offset := 4
	if err := vm.parseBytecode(bytecode, &offset); err != nil {
		return nil, err
	}

	vm.code = bytecode
	vm.pc = offset
	vm.halted = false

	return vm.runLoop()
}

// executeRaw runs raw instruction bytes without a bytecode header.
// Constants, locals, globals, and builtins must already be set on the VM.
func (vm *VM) executeRaw(instructions []byte) (Value, error) {
	vm.code = instructions
	vm.pc = 0
	vm.halted = false

	return vm.runLoop()
}

// runLoop is the core execution loop shared by Execute and executeRaw.
func (vm *VM) runLoop() (Value, error) {
	// Execute instructions with step limit to prevent infinite loops
	steps := 0
	for !vm.halted && vm.pc < len(vm.code) {
		if err := vm.step(); err != nil {
			return nil, err
		}
		steps++
		if vm.maxSteps > 0 && steps > vm.maxSteps {
			return nil, fmt.Errorf("execution exceeded maximum step limit (%d steps)", vm.maxSteps)
		}
	}

	// Return top of stack or null
	if len(vm.stack) > 0 {
		return vm.Pop()
	}
	return NullValue{}, nil
}

// parseBytecode parses the bytecode header and constants
func (vm *VM) parseBytecode(bytecode []byte, offset *int) error {
	// Read version (4 bytes)
	if *offset+4 > len(bytecode) {
		return fmt.Errorf("invalid bytecode: missing version")
	}
	version := binary.LittleEndian.Uint32(bytecode[*offset : *offset+4])
	*offset += 4

	if version != 1 {
		return fmt.Errorf("unsupported bytecode version: %d", version)
	}

	// Read constant count (4 bytes)
	if *offset+4 > len(bytecode) {
		return fmt.Errorf("invalid bytecode: missing constant count")
	}
	constCount := binary.LittleEndian.Uint32(bytecode[*offset : *offset+4])
	*offset += 4

	// Read constants
	for i := uint32(0); i < constCount; i++ {
		constant, err := vm.readConstant(bytecode, offset)
		if err != nil {
			return err
		}
		vm.constants = append(vm.constants, constant)
	}

	// Read string pool count (4 bytes)
	if *offset+4 > len(bytecode) {
		return fmt.Errorf("invalid bytecode: missing string pool count")
	}
	strPoolCount := binary.LittleEndian.Uint32(bytecode[*offset : *offset+4])
	*offset += 4

	// Read string pool entries
	for i := uint32(0); i < strPoolCount; i++ {
		if *offset+4 > len(bytecode) {
			return fmt.Errorf("invalid bytecode: truncated string pool entry")
		}
		strLen := binary.LittleEndian.Uint32(bytecode[*offset : *offset+4])
		*offset += 4
		if *offset+int(strLen) > len(bytecode) {
			return fmt.Errorf("invalid bytecode: truncated string pool data")
		}
		s := string(bytecode[*offset : *offset+int(strLen)])
		*offset += int(strLen)
		vm.stringPool = append(vm.stringPool, s)
	}

	// Read instruction count (4 bytes)
	if *offset+4 > len(bytecode) {
		return fmt.Errorf("invalid bytecode: missing instruction count")
	}
	*offset += 4 // Skip instruction count, we'll execute until halt or end

	return nil
}

// readConstant reads a constant from bytecode
func (vm *VM) readConstant(bytecode []byte, offset *int) (Value, error) {
	if *offset >= len(bytecode) {
		return nil, fmt.Errorf("invalid bytecode: unexpected end while reading constant")
	}

	constType := bytecode[*offset]
	*offset++

	switch constType {
	case 0x00: // Null
		return NullValue{}, nil
	case 0x01: // Int
		if *offset+8 > len(bytecode) {
			return nil, fmt.Errorf("invalid bytecode: truncated int constant")
		}
		val := int64(binary.LittleEndian.Uint64(bytecode[*offset : *offset+8]))
		*offset += 8
		return IntValue{Val: val}, nil
	case 0x02: // Float
		if *offset+8 > len(bytecode) {
			return nil, fmt.Errorf("invalid bytecode: truncated float constant")
		}
		bits := binary.LittleEndian.Uint64(bytecode[*offset : *offset+8])
		val := math.Float64frombits(bits)
		*offset += 8
		return FloatValue{Val: val}, nil
	case 0x03: // Bool
		if *offset >= len(bytecode) {
			return nil, fmt.Errorf("invalid bytecode: truncated bool constant")
		}
		val := bytecode[*offset] != 0
		*offset++
		return BoolValue{Val: val}, nil
	case 0x04: // String
		if *offset+4 > len(bytecode) {
			return nil, fmt.Errorf("invalid bytecode: truncated string length")
		}
		length := binary.LittleEndian.Uint32(bytecode[*offset : *offset+4])
		*offset += 4
		if *offset+int(length) > len(bytecode) {
			return nil, fmt.Errorf("invalid bytecode: truncated string data")
		}
		val := string(bytecode[*offset : *offset+int(length)])
		*offset += int(length)
		return StringValue{Val: val}, nil
	default:
		return nil, fmt.Errorf("unknown constant type: 0x%02x", constType)
	}
}

// step executes one instruction
func (vm *VM) step() error {
	if vm.pc >= len(vm.code) {
		return fmt.Errorf("program counter out of bounds")
	}

	opcode := Opcode(vm.code[vm.pc])
	vm.pc++

	return vm.executeInstruction(opcode)
}

// executeInstruction executes a single instruction
func (vm *VM) executeInstruction(opcode Opcode) error {
	switch opcode {
	case OpPush:
		return vm.execPush()
	case OpPop:
		_, err := vm.Pop()
		return err
	case OpAdd:
		return vm.execAdd()
	case OpSub:
		return vm.execSub()
	case OpMul:
		return vm.execMul()
	case OpDiv:
		return vm.execDiv()
	case OpMod:
		return vm.execMod()
	case OpEq:
		return vm.execEq()
	case OpNe:
		return vm.execNe()
	case OpLt:
		return vm.execLt()
	case OpGt:
		return vm.execGt()
	case OpGe:
		return vm.execGe()
	case OpLe:
		return vm.execLe()
	case OpAnd:
		return vm.execAnd()
	case OpOr:
		return vm.execOr()
	case OpNot:
		return vm.execNot()
	case OpNeg:
		return vm.execNeg()
	case OpLoadVar:
		return vm.execLoadVar()
	case OpStoreVar:
		return vm.execStoreVar()
	case OpJump:
		return vm.execJump()
	case OpJumpIfFalse:
		return vm.execJumpIfFalse()
	case OpJumpIfTrue:
		return vm.execJumpIfTrue()
	case OpGetIter:
		return vm.execGetIter()
	case OpIterNext:
		return vm.execIterNext()
	case OpIterHasNext:
		return vm.execIterHasNext()
	case OpGetIndex:
		return vm.execGetIndex()
	case OpSetIndex:
		return vm.execSetIndex()
	case OpReturn:
		return vm.execReturn()
	case OpCall:
		return vm.execCall()
	case OpBuildObject:
		return vm.execBuildObject()
	case OpGetField:
		return vm.execGetField()
	case OpSetField:
		return vm.execSetField()
	case OpDefFunc:
		return vm.execDefFunc()
	case OpBuildArray:
		return vm.execBuildArray()
	case OpLoadString:
		return vm.execLoadString()
	case OpHttpReturn:
		return vm.execHttpReturn()
	case OpWsSend:
		return vm.execWsSend()
	case OpWsBroadcast:
		return vm.execWsBroadcast()
	case OpWsBroadcastRoom:
		return vm.execWsBroadcastRoom()
	case OpWsJoinRoom:
		return vm.execWsJoinRoom()
	case OpWsLeaveRoom:
		return vm.execWsLeaveRoom()
	case OpWsClose:
		return vm.execWsClose()
	case OpWsGetRooms:
		return vm.execWsGetRooms()
	case OpWsGetClients:
		return vm.execWsGetClients()
	case OpWsGetConnCount:
		return vm.execWsGetConnCount()
	case OpWsGetUptime:
		return vm.execWsGetUptime()
	case OpAsync:
		return vm.execAsync()
	case OpAwait:
		return vm.execAwait()
	case OpTry:
		return vm.execTry()
	case OpMitosis:
		return vm.execMitosis()
	case OpMutator:
		return vm.execMutator()
	case OpTelemetry:
		return vm.execTelemetry()
	case OpSpawn:
		return vm.execSpawn()
	case OpKill:
		return vm.execKill()
	case OpWait:
		return vm.execWait()
	case OpHalt:
		vm.halted = true
		return nil
	default:
		return fmt.Errorf("unknown opcode: 0x%02x", opcode)
	}
}

// execLoadString loads a string from the string pool by index and pushes it onto the stack
func (vm *VM) execLoadString() error {
	idx, err := vm.readOperand()
	if err != nil {
		return err
	}

	if int(idx) >= len(vm.stringPool) {
		return fmt.Errorf("string pool index out of bounds: %d (pool size: %d)", idx, len(vm.stringPool))
	}

	vm.Push(StringValue{Val: vm.stringPool[idx]})
	return nil
}

// execPush pushes a constant onto the stack
func (vm *VM) execPush() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	if int(operand) >= len(vm.constants) {
		return fmt.Errorf("constant index out of bounds: %d", operand)
	}

	vm.Push(vm.constants[operand])
	return nil
}

// execAdd adds two values
func (vm *VM) execAdd() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(IntValue{Val: av.Val + bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: float64(av.Val) + bv.Val})
			return nil
		}
		if bv, ok := b.(StringValue); ok {
			// int + string: coerce int to string and concatenate
			vm.Push(StringValue{Val: intToString(av.Val) + bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: av.Val + bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(FloatValue{Val: av.Val + float64(bv.Val)})
			return nil
		}
		if bv, ok := b.(StringValue); ok {
			// float + string: coerce float to string and concatenate
			vm.Push(StringValue{Val: floatToString(av.Val) + bv.Val})
			return nil
		}
	case StringValue:
		// If either operand is a string, concatenate by converting both to string
		vm.Push(StringValue{Val: av.Val + valueToString(b)})
		return nil
	case BoolValue:
		if bv, ok := b.(StringValue); ok {
			// bool + string: coerce bool to string and concatenate
			s := "false"
			if av.Val {
				s = "true"
			}
			vm.Push(StringValue{Val: s + bv.Val})
			return nil
		}
	case ArrayValue:
		if bv, ok := b.(ArrayValue); ok {
			// Array concatenation
			result := make([]Value, len(av.Val)+len(bv.Val))
			copy(result, av.Val)
			copy(result[len(av.Val):], bv.Val)
			vm.Push(ArrayValue{Val: result})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot add %s and %s", a.Type(), b.Type())
}

// execSub subtracts two values
func (vm *VM) execSub() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(IntValue{Val: av.Val - bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: float64(av.Val) - bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: av.Val - bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(FloatValue{Val: av.Val - float64(bv.Val)})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot subtract %s and %s", a.Type(), b.Type())
}

// execMul multiplies two values
func (vm *VM) execMul() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(IntValue{Val: av.Val * bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: float64(av.Val) * bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(FloatValue{Val: av.Val * bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(FloatValue{Val: av.Val * float64(bv.Val)})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot multiply %s and %s", a.Type(), b.Type())
}

// execDiv divides two values
func (vm *VM) execDiv() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("division by zero")
			}
			vm.Push(IntValue{Val: av.Val / bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("division by zero")
			}
			vm.Push(FloatValue{Val: float64(av.Val) / bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("division by zero")
			}
			vm.Push(FloatValue{Val: av.Val / bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("division by zero")
			}
			vm.Push(FloatValue{Val: av.Val / float64(bv.Val)})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot divide %s and %s", a.Type(), b.Type())
}

// execMod performs modulo/remainder operation
func (vm *VM) execMod() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("modulo by zero")
			}
			vm.Push(IntValue{Val: av.Val % bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("modulo by zero")
			}
			vm.Push(FloatValue{Val: math.Mod(float64(av.Val), bv.Val)})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("modulo by zero")
			}
			vm.Push(FloatValue{Val: math.Mod(av.Val, bv.Val)})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			if bv.Val == 0 {
				return fmt.Errorf("modulo by zero")
			}
			vm.Push(FloatValue{Val: math.Mod(av.Val, float64(bv.Val))})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot compute modulo of %s and %s", a.Type(), b.Type())
}

// execEq checks equality
func (vm *VM) execEq() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	result := vm.valuesEqual(a, b)
	vm.Push(BoolValue{Val: result})
	return nil
}

// execNe checks inequality
func (vm *VM) execNe() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	result := !vm.valuesEqual(a, b)
	vm.Push(BoolValue{Val: result})
	return nil
}

// execLt checks less than
func (vm *VM) execLt() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val < bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: float64(av.Val) < bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: av.Val < bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val < float64(bv.Val)})
			return nil
		}
	case StringValue:
		if bv, ok := b.(StringValue); ok {
			vm.Push(BoolValue{Val: av.Val < bv.Val})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot compare %s and %s", a.Type(), b.Type())
}

// execLoadVar loads a variable
func (vm *VM) execLoadVar() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	if int(operand) >= len(vm.constants) {
		return fmt.Errorf("constant index out of bounds: %d", operand)
	}

	nameVal := vm.constants[operand]
	name, ok := nameVal.(StringValue)
	if !ok {
		return fmt.Errorf("variable name must be a string, got %T", nameVal)
	}

	// Try lexical environment first, then globals
	for cur := vm.env; cur != nil; cur = cur.Parent {
		if val, exists := cur.Values[name.Val]; exists {
			vm.Push(val)
			return nil
		}
	}
	if val, exists := vm.globals[name.Val]; exists {
		vm.Push(val)
		return nil
	}

	return fmt.Errorf("undefined variable: %s", name.Val)
}

// execStoreVar stores a variable
func (vm *VM) execStoreVar() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	if int(operand) >= len(vm.constants) {
		return fmt.Errorf("constant index out of bounds: %d", operand)
	}

	nameVal := vm.constants[operand]
	name, ok := nameVal.(StringValue)
	if !ok {
		return fmt.Errorf("variable name must be a string")
	}

	val, err := vm.Pop()
	if err != nil {
		return err
	}

	// Lexical scope update: search for existing variable to update
	for cur := vm.env; cur != nil; cur = cur.Parent {
		if _, exists := cur.Values[name.Val]; exists {
			cur.Values[name.Val] = val
			return nil
		}
	}
	// If not found in parent scopes, define in current local scope
	vm.env.Values[name.Val] = val
	return nil
}

// execJump performs an unconditional jump
func (vm *VM) execJump() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	vm.pc = int(operand)
	return nil
}

// execGetIter creates an iterator for a collection
func (vm *VM) execGetIter() error {
	collection, err := vm.Pop()
	if err != nil {
		return err
	}

	iter := &Iterator{
		collection: collection,
		index:      0,
	}

	// For objects, pre-compute keys
	if objVal, ok := collection.(ObjectValue); ok {
		iter.keys = make([]string, 0, len(objVal.Val))
		for k := range objVal.Val {
			iter.keys = append(iter.keys, k)
		}
	}

	// Store iterator and push ID
	iterID := vm.nextIterID
	vm.nextIterID++
	vm.iterators[iterID] = iter
	vm.Push(IntValue{Val: int64(iterID)})
	return nil
}

// execIterHasNext checks if iterator has more elements
func (vm *VM) execIterHasNext() error {
	iterIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	iterID, ok := iterIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("iterator ID must be an integer")
	}

	iter, exists := vm.iterators[int(iterID.Val)]
	if !exists {
		return fmt.Errorf("invalid iterator ID: %d", iterID.Val)
	}

	hasNext := false
	switch coll := iter.collection.(type) {
	case ArrayValue:
		hasNext = iter.index < len(coll.Val)
	case ObjectValue:
		hasNext = iter.index < len(iter.keys)
	default:
		return fmt.Errorf("cannot iterate over %s", coll.Type())
	}

	// Clean up exhausted iterators to prevent memory leaks
	if !hasNext {
		delete(vm.iterators, int(iterID.Val))
	}

	vm.Push(BoolValue{Val: hasNext})
	return nil
}

// execIterNext advances iterator and pushes key/value
func (vm *VM) execIterNext() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}
	hasKey := operand != 0

	iterIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	iterID, ok := iterIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("iterator ID must be an integer")
	}

	iter, exists := vm.iterators[int(iterID.Val)]
	if !exists {
		return fmt.Errorf("invalid iterator ID: %d", iterID.Val)
	}

	switch coll := iter.collection.(type) {
	case ArrayValue:
		if iter.index >= len(coll.Val) {
			return fmt.Errorf("iterator exhausted")
		}
		if hasKey {
			vm.Push(IntValue{Val: int64(iter.index)})
		}
		vm.Push(coll.Val[iter.index])
		iter.index++
	case ObjectValue:
		if iter.index >= len(iter.keys) {
			return fmt.Errorf("iterator exhausted")
		}
		key := iter.keys[iter.index]
		if hasKey {
			vm.Push(StringValue{Val: key})
		}
		vm.Push(coll.Val[key])
		iter.index++
	default:
		return fmt.Errorf("cannot iterate over %s", coll.Type())
	}

	return nil
}

// execGetIndex gets an element from an array by index
func (vm *VM) execGetIndex() error {
	index, err := vm.Pop()
	if err != nil {
		return err
	}
	arr, err := vm.Pop()
	if err != nil {
		return err
	}

	arrVal, ok := arr.(ArrayValue)
	if !ok {
		return fmt.Errorf("type error: can only index arrays, got %s", arr.Type())
	}

	indexInt, ok := index.(IntValue)
	if !ok {
		return fmt.Errorf("type error: array index must be an integer, got %s", index.Type())
	}

	if indexInt.Val < 0 || indexInt.Val >= int64(len(arrVal.Val)) {
		return fmt.Errorf("index out of bounds: %d", indexInt.Val)
	}

	vm.Push(arrVal.Val[indexInt.Val])
	return nil
}

// execBuildObject builds an object from stack values
func (vm *VM) execBuildObject() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	fieldCount := int(operand)
	obj := make(map[string]Value)

	for i := 0; i < fieldCount; i++ {
		val, err := vm.Pop()
		if err != nil {
			return err
		}
		key, err := vm.Pop()
		if err != nil {
			return err
		}

		keyStr, ok := key.(StringValue)
		if !ok {
			return fmt.Errorf("object key must be a string")
		}

		obj[keyStr.Val] = val
	}

	vm.Push(ObjectValue{Val: obj})
	return nil
}

// execBuildArray builds an array from stack values
func (vm *VM) execBuildArray() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	elemCount := int(operand)
	arr := make([]Value, elemCount)

	// Pop in reverse order
	for i := elemCount - 1; i >= 0; i-- {
		val, err := vm.Pop()
		if err != nil {
			return err
		}
		arr[i] = val
	}

	vm.Push(ArrayValue{Val: arr})
	return nil
}

// execHttpReturn handles HTTP return
func (vm *VM) execHttpReturn() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}

	// Push back for retrieval
	vm.Push(val)
	vm.halted = true
	return nil
}

// execGt checks greater than
func (vm *VM) execGt() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val > bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: float64(av.Val) > bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: av.Val > bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val > float64(bv.Val)})
			return nil
		}
	case StringValue:
		if bv, ok := b.(StringValue); ok {
			vm.Push(BoolValue{Val: av.Val > bv.Val})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot compare %s and %s", a.Type(), b.Type())
}

// execGe checks greater than or equal
func (vm *VM) execGe() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val >= bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: float64(av.Val) >= bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: av.Val >= bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val >= float64(bv.Val)})
			return nil
		}
	case StringValue:
		if bv, ok := b.(StringValue); ok {
			vm.Push(BoolValue{Val: av.Val >= bv.Val})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot compare %s and %s", a.Type(), b.Type())
}

// execLe checks less than or equal
func (vm *VM) execLe() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val <= bv.Val})
			return nil
		}
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: float64(av.Val) <= bv.Val})
			return nil
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			vm.Push(BoolValue{Val: av.Val <= bv.Val})
			return nil
		}
		if bv, ok := b.(IntValue); ok {
			vm.Push(BoolValue{Val: av.Val <= float64(bv.Val)})
			return nil
		}
	case StringValue:
		if bv, ok := b.(StringValue); ok {
			vm.Push(BoolValue{Val: av.Val <= bv.Val})
			return nil
		}
	}

	return fmt.Errorf("type error: cannot compare %s and %s", a.Type(), b.Type())
}

// execAnd performs logical AND
func (vm *VM) execAnd() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	av, aok := a.(BoolValue)
	bv, bok := b.(BoolValue)
	if !aok || !bok {
		return fmt.Errorf("type error: logical AND requires boolean operands")
	}

	vm.Push(BoolValue{Val: av.Val && bv.Val})
	return nil
}

// execOr performs logical OR
func (vm *VM) execOr() error {
	b, err := vm.Pop()
	if err != nil {
		return err
	}
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	av, aok := a.(BoolValue)
	bv, bok := b.(BoolValue)
	if !aok || !bok {
		return fmt.Errorf("type error: logical OR requires boolean operands")
	}

	vm.Push(BoolValue{Val: av.Val || bv.Val})
	return nil
}

// execNot performs logical NOT
func (vm *VM) execNot() error {
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	av, ok := a.(BoolValue)
	if !ok {
		return fmt.Errorf("type error: logical NOT requires boolean operand")
	}

	vm.Push(BoolValue{Val: !av.Val})
	return nil
}

// execNeg performs unary negation
func (vm *VM) execNeg() error {
	a, err := vm.Pop()
	if err != nil {
		return err
	}

	switch av := a.(type) {
	case IntValue:
		vm.Push(IntValue{Val: -av.Val})
	case FloatValue:
		vm.Push(FloatValue{Val: -av.Val})
	default:
		return fmt.Errorf("type error: unary negation requires numeric operand, got %T", a)
	}
	return nil
}

// execJumpIfFalse performs conditional jump if top of stack is false
func (vm *VM) execJumpIfFalse() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	val, err := vm.Pop()
	if err != nil {
		return err
	}

	boolVal, ok := val.(BoolValue)
	if !ok {
		return fmt.Errorf("type error: conditional jump requires boolean value")
	}

	if !boolVal.Val {
		vm.pc = int(operand)
	}

	return nil
}

// execJumpIfTrue performs conditional jump if top of stack is true
func (vm *VM) execJumpIfTrue() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	val, err := vm.Pop()
	if err != nil {
		return err
	}

	boolVal, ok := val.(BoolValue)
	if !ok {
		return fmt.Errorf("type error: conditional jump requires boolean value")
	}

	if boolVal.Val {
		vm.pc = int(operand)
	}

	return nil
}

// execGetField gets a field from an object
func (vm *VM) execGetField() error {
	key, err := vm.Pop()
	if err != nil {
		return err
	}
	obj, err := vm.Pop()
	if err != nil {
		return err
	}

	keyStr, ok := key.(StringValue)
	if !ok {
		return fmt.Errorf("type error: field name must be a string")
	}

	objVal, ok := obj.(ObjectValue)
	if !ok {
		return fmt.Errorf("type error: can only get field from object")
	}

	fieldVal, exists := objVal.Val[keyStr.Val]
	if !exists {
		return fmt.Errorf("field not found: %s", keyStr.Val)
	}

	vm.Push(fieldVal)
	return nil
}

// execSetField sets a field on an object.
// Stack: [obj, key, value] → pops value, key, obj; mutates obj; pushes obj back.
func (vm *VM) execSetField() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}
	key, err := vm.Pop()
	if err != nil {
		return err
	}
	obj, err := vm.Pop()
	if err != nil {
		return err
	}

	keyStr, ok := key.(StringValue)
	if !ok {
		return fmt.Errorf("type error: field name must be a string, got %s", key.Type())
	}

	objVal, ok := obj.(ObjectValue)
	if !ok {
		return fmt.Errorf("type error: can only set field on object, got %s", obj.Type())
	}

	objVal.Val[keyStr.Val] = val
	vm.Push(objVal)
	return nil
}

// execSetIndex sets an element in an array or map.
// Stack: [arr, index, value] → pops value, index, arr; mutates arr; pushes arr back.
func (vm *VM) execSetIndex() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}
	index, err := vm.Pop()
	if err != nil {
		return err
	}
	container, err := vm.Pop()
	if err != nil {
		return err
	}

	switch c := container.(type) {
	case ArrayValue:
		indexInt, ok := index.(IntValue)
		if !ok {
			return fmt.Errorf("type error: array index must be an integer, got %s", index.Type())
		}
		if indexInt.Val < 0 || indexInt.Val >= int64(len(c.Val)) {
			return fmt.Errorf("index out of bounds: %d (length %d)", indexInt.Val, len(c.Val))
		}
		c.Val[indexInt.Val] = val
		vm.Push(c)
	case ObjectValue:
		keyStr, ok := index.(StringValue)
		if !ok {
			return fmt.Errorf("type error: object key must be a string, got %s", index.Type())
		}
		c.Val[keyStr.Val] = val
		vm.Push(c)
	default:
		return fmt.Errorf("type error: can only set index on array or object, got %s", container.Type())
	}

	return nil
}

// execDefFunc defines a user function.
// Format: OP_DEF_FUNC, name_idx(4), param_count(4), body_length(4), [param_name_idx(4)]..., body...
func (vm *VM) execDefFunc() error {
	nameIdx, err := vm.readOperand()
	if err != nil {
		return err
	}
	if int(nameIdx) >= len(vm.constants) {
		return fmt.Errorf("function name constant out of bounds: %d", nameIdx)
	}
	nameVal, ok := vm.constants[nameIdx].(StringValue)
	if !ok {
		return fmt.Errorf("function name must be a string")
	}

	paramCount, err := vm.readOperand()
	if err != nil {
		return err
	}

	bodyLength, err := vm.readOperand()
	if err != nil {
		return err
	}

	// Read parameter names
	paramNames := make([]string, paramCount)
	for i := uint32(0); i < paramCount; i++ {
		pidx, err := vm.readOperand()
		if err != nil {
			return err
		}
		if int(pidx) >= len(vm.constants) {
			return fmt.Errorf("param name constant out of bounds: %d", pidx)
		}
		pval, ok := vm.constants[pidx].(StringValue)
		if !ok {
			return fmt.Errorf("param name must be a string")
		}
		paramNames[i] = pval.Val
	}

	// Register the function (body starts at current PC)
	// Capture current environment as closure
	fn := FunctionValue{
		Name:       nameVal.Val,
		ParamNames: paramNames,
		CodeOffset: vm.pc,
		CodeLength: int(bodyLength),
		Closure:    vm.env,
	}
	vm.functions[nameVal.Val] = fn
	// Also define it as a variable in current scope to support returning functions
	vm.env.Values[nameVal.Val] = fn

	// Skip over the function body
	vm.pc += int(bodyLength)

	return nil
}

// execReturn returns from a function call or halts if at top level.
func (vm *VM) execReturn() error {
	if len(vm.callStack) == 0 {
		// Top-level return → halt
		vm.halted = true
		return nil
	}

	// Pop call frame
	frame := vm.callStack[len(vm.callStack)-1]
	vm.callStack = vm.callStack[:len(vm.callStack)-1]

	// Restore caller state
	vm.pc = frame.returnPC
	vm.env = frame.env
	vm.code = frame.returnBytecode
	vm.constants = frame.returnConstants

	return nil
}

// execCall performs a function call

func (vm *VM) executeFunction(fn FunctionValue, args []Value) error {
	// Save current frame
	vm.callStack = append(vm.callStack, CallFrame{
		returnPC:        vm.pc,
		returnBytecode:  vm.code,
		returnConstants: vm.constants,
		env:             vm.env,
	})

	// Set up new environment with parameters, capturing closure if present
	vm.env = NewEnvironment(fn.Closure)
	for i, paramName := range fn.ParamNames {
		if i < len(args) {
			vm.env.Values[paramName] = args[i]
		}
	}

	if fn.Bytecode != nil {
		// Switch to modular bytecode and its constant pool
		vm.code = fn.Bytecode
		if fn.Constants != nil {
			vm.constants = fn.Constants
		}
		layout, err := parseBytecodeLayout(vm.code)
		if err != nil {
			return fmt.Errorf("failed to call modular function %s: %w", fn.Name, err)
		}
		vm.pc = int(layout.CodeOffset)
	} else {
		// Stay in current bytecode
		vm.pc = fn.CodeOffset
	}

	return nil
}

func (vm *VM) execCall() error {
	operand, err := vm.readOperand()
	if err != nil {
		return err
	}

	argCount := int(operand)

	// Pop arguments
	args := make([]Value, argCount)
	for i := argCount - 1; i >= 0; i-- {
		arg, err := vm.Pop()
		if err != nil {
			return err
		}
		args[i] = arg
	}

	// Pop function name/reference
	fnVal, err := vm.Pop()
	if err != nil {
		return err
	}

	// Handle direct function value (closures)
	if fn, ok := fnVal.(FunctionValue); ok {
		return vm.executeFunction(fn, args)
	}

	// Get function name
	fnName, ok := fnVal.(StringValue)
	if !ok {
		return fmt.Errorf("function name must be a string, got %T", fnVal)
	}

	// Look up built-in function
	if builtinFn, exists := vm.builtins[fnName.Val]; exists {
		result, err := builtinFn(args)
		if err != nil {
			return fmt.Errorf("built-in function %s failed: %w", fnName.Val, err)
		}
		vm.Push(result)
		return nil
	}

	// Look up in environment (for closures stored in variables)
	for cur := vm.env; cur != nil; cur = cur.Parent {
		if val, exists := cur.Values[fnName.Val]; exists {
			if fn, ok := val.(FunctionValue); ok {
				return vm.executeFunction(fn, args)
			}
		}
	}

	// Look up user-defined function
	if fn, exists := vm.functions[fnName.Val]; exists {
		// Save current frame
		vm.callStack = append(vm.callStack, CallFrame{
			returnPC:        vm.pc,
			returnBytecode:  vm.code,
			returnConstants: vm.constants,
			env:             vm.env,
		})

		// Set up new environment with parameters, capturing closure if present
		vm.env = NewEnvironment(fn.Closure)
		for i, paramName := range fn.ParamNames {
			if i < len(args) {
				vm.env.Values[paramName] = args[i]
			}
		}

		if fn.Bytecode != nil {
			// Switch to modular bytecode and its constant pool
			vm.code = fn.Bytecode
			if fn.Constants != nil {
				vm.constants = fn.Constants
			}
			layout, err := parseBytecodeLayout(vm.code)
			if err != nil {
				return fmt.Errorf("failed to call modular function %s: %w", fnName.Val, err)
			}
			vm.pc = int(layout.CodeOffset)
		} else {
			// Stay in current bytecode
			vm.pc = fn.CodeOffset
		}
		return nil
	}

	// Function not found
	return fmt.Errorf("undefined function: %s", fnName.Val)
}

// execTelemetry writes to the telemetry plane

// execMutator modifies the bytecode at a relative offset.
// This enables the Ouroboros pattern: self-modifying code.

// execMitosis clones the current VM and starts parallel execution.
// (Spatial S opcode)
func (vm *VM) execMitosis() error {
	_, err := vm.Pop() // spatial_offset
	if err != nil { return err }
	
	// TODO: Implement actual parallel spawning.
	// For now, just a stub that indicates success.
	vm.Push(BoolValue{Val: true})
	return nil
}

func (vm *VM) execMutator() error {
	offset, err := vm.Pop()
	if err != nil { return err }
	val, err := vm.Pop()
	if err != nil { return err }

	iv, ok1 := val.(IntValue)
	ov, ok2 := offset.(IntValue)
	if !ok1 || !ok2 {
		return fmt.Errorf("mutator expects integer value and offset")
	}

	target := vm.pc + int(ov.Val)
	if target < 0 || target >= len(vm.code) {
		return fmt.Errorf("mutator target out of bounds: %d", target)
	}

	vm.code[target] = byte(iv.Val)
	return nil
}

func (vm *VM) execTelemetry() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}
	slot, err := vm.Pop()
	if err != nil {
		return err
	}

	s, ok1 := slot.(IntValue)
	v, ok2 := val.(IntValue)
	if !ok1 || !ok2 {
		return fmt.Errorf("telemetry expects integer arguments")
	}

	if os.Getenv("GLYPH_DEBUG") == "true" {
		fmt.Printf("[TELEMETRY] Slot %d = %d\n", s.Val, v.Val)
	}
	return nil
}

// execSpawn creates a new child VM process (fork semantics).
// It copies the current bytecode, assigns a PID, sets the parent PID,
// pushes the child PID on the parent's stack, and pushes 0 on the child's stack.
func (vm *VM) execSpawn() error {
	if vm.processTable == nil {
		return fmt.Errorf("spawn: no process table attached to VM")
	}

	// Allocate a new PID for the child
	childPID := vm.processTable.AllocatePID()

	// Clone the current VM for the child
	childVM := vm.Clone()

	// Ensure full isolation: deep copy mutable state so parent and child
	// do not share any writable memory region.
	// Deep copy globals
	childVM.globals = make(map[string]Value, len(vm.globals))
	for k, v := range vm.globals {
		childVM.globals[k] = v
	}
	// Deep copy functions
	childVM.functions = make(map[string]FunctionValue, len(vm.functions))
	for k, v := range vm.functions {
		childVM.functions[k] = v
	}
	// Deep copy environment (Clone already does this, but ensure it's fresh)
	childVM.env = vm.env.Clone()

	// Fresh stack for child — push 0
	childVM.stack = make([]Value, 0, 256)
	childVM.Push(IntValue{Val: 0})
	childVM.pid = childPID
	childVM.processTable = vm.processTable

	// Create process entry
	childProc := &Process{
		PID:   childPID,
		PPID:  vm.pid,
		State: ProcessReady,
		VM:    childVM,
	}
	vm.processTable.Register(childProc)

	// Track child in parent's ChildPIDs (3.2 parent-child tracking)
	if parentProc, ok := vm.processTable.Get(vm.pid); ok {
		parentProc.ChildPIDs = append(parentProc.ChildPIDs, childPID)
	}

	// Parent gets the child PID on its stack
	vm.Push(IntValue{Val: int64(childPID)})

	return nil
}

// execKill terminates a process by PID.
// It sets the target to zombie, releases its resources, and reparents
// its children to PID 1.
func (vm *VM) execKill() error {
	if vm.processTable == nil {
		return fmt.Errorf("kill: no process table attached to VM")
	}

	targetPIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	targetPIDInt, ok := targetPIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("kill: target PID must be an integer")
	}
	targetPID := uint32(targetPIDInt.Val)

	// Verify target exists
	if _, exists := vm.processTable.Get(targetPID); !exists {
		return fmt.Errorf("kill: process %d not found", targetPID)
	}

	// Don't allow killing PID 1 (init)
	if targetPID == 1 {
		return fmt.Errorf("kill: cannot kill init process (PID 1)")
	}

	// Reparent target's children to PID 1
	vm.processTable.Reparent(targetPID, 1)

	// Set target to zombie with exit code 0 (killed)
	if err := vm.processTable.SetZombie(targetPID, 0); err != nil {
		return fmt.Errorf("kill: %w", err)
	}

	return nil
}

// execWait blocks the calling process until the target process enters zombie state.
// It reads the exit code from the zombie's process struct and cleans up the zombie entry.
func (vm *VM) execWait() error {
	if vm.processTable == nil {
		return fmt.Errorf("wait: no process table attached to VM")
	}

	targetPIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	targetPIDInt, ok := targetPIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("wait: target PID must be an integer")
	}
	targetPID := uint32(targetPIDInt.Val)

	// Verify target exists
	target, exists := vm.processTable.Get(targetPID)
	if !exists {
		return fmt.Errorf("wait: process %d not found", targetPID)
	}

	// If already zombie, reap immediately
	if target.State == ProcessZombie {
		exitCode := target.ExitCode
		vm.processTable.Remove(targetPID)
		vm.Push(IntValue{Val: exitCode})
		return nil
	}

	// Block until the process becomes zombie
	ch := vm.processTable.RegisterWaiter(targetPID)
	<-ch // blocks until the process transitions to zombie

	// Re-read the process (it should now be zombie)
	target, exists = vm.processTable.Get(targetPID)
	if !exists {
		// Process was reaped by someone else
		vm.Push(IntValue{Val: -1})
		return nil
	}

	exitCode := target.ExitCode
	vm.processTable.Remove(targetPID)
	vm.Push(IntValue{Val: exitCode})

	return nil
}

// readOperand reads a 4-byte operand
func (vm *VM) readOperand() (uint32, error) {
	if vm.pc+4 > len(vm.code) {
		return 0, fmt.Errorf("truncated operand at pc=%d", vm.pc)
	}

	operand := binary.LittleEndian.Uint32(vm.code[vm.pc : vm.pc+4])
	vm.pc += 4
	return operand, nil
}

// valuesEqual checks if two values are equal
func (vm *VM) valuesEqual(a, b Value) bool {
	switch av := a.(type) {
	case IntValue:
		if bv, ok := b.(IntValue); ok {
			return av.Val == bv.Val
		}
	case FloatValue:
		if bv, ok := b.(FloatValue); ok {
			return av.Val == bv.Val
		}
	case BoolValue:
		if bv, ok := b.(BoolValue); ok {
			return av.Val == bv.Val
		}
	case StringValue:
		if bv, ok := b.(StringValue); ok {
			return av.Val == bv.Val
		}
	case NullValue:
		_, ok := b.(NullValue)
		return ok
	}
	return false
}

// maxStackSize is the maximum stack depth to prevent unbounded memory usage.
const maxStackSize = 10000

// Push adds a value to the stack
func (vm *VM) Push(val Value) {
	if len(vm.stack) >= maxStackSize {
		// Silently drop to avoid panicking in hot paths; the step limit
		// will catch runaway programs. In future versions this could return an error.
		return
	}
	vm.stack = append(vm.stack, val)
}

// Pop removes and returns a value from the stack
func (vm *VM) Pop() (Value, error) {
	if len(vm.stack) == 0 {
		return nil, fmt.Errorf("stack underflow")
	}

	val := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return val, nil
}

type bytecodeLayout struct {
	CodeOffset uint32
}

func parseBytecodeLayout(bytecode []byte) (*bytecodeLayout, error) {
	if len(bytecode) < 16 {
		return nil, fmt.Errorf("bytecode too short")
	}
	if string(bytecode[:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid magic")
	}

	constCount := binary.LittleEndian.Uint32(bytecode[8:12])
	offset := 12
	for i := 0; i < int(constCount); i++ {
		if offset >= len(bytecode) {
			return nil, fmt.Errorf("invalid constant pool")
		}
		tag := bytecode[offset]
		offset++
		switch tag {
		case 0x00: // null
		case 0x01, 0x02: // int, float
			offset += 8
		case 0x03: // bool
			offset += 1
		case 0x04: // string
			if offset+4 > len(bytecode) {
				return nil, fmt.Errorf("invalid string constant")
			}
			length := binary.LittleEndian.Uint32(bytecode[offset : offset+4])
			offset += 4 + int(length)
		default:
			return nil, fmt.Errorf("unknown constant tag: 0x%02x", tag)
		}
	}

	if offset+4 > len(bytecode) {
		return nil, fmt.Errorf("missing instruction count")
	}
	// skip instrCount
	offset += 4

	return &bytecodeLayout{
		CodeOffset: uint32(offset),
	}, nil
}

// SetLocal sets a local variable value (used for route parameters and injections)
func (vm *VM) SetLocal(name string, value Value) {
	vm.env.Values[name] = value
}

// RegisterFunction registers a compiled function with the VM
func (vm *VM) RegisterFunction(name string, bytecode []byte) {
	fn := FunctionValue{
		Name:     name,
		Bytecode: bytecode,
	}

	// Extract constants from bytecode
	offset := 4 // skip magic
	// Read version
	if offset+4 <= len(bytecode) {
		offset += 4
		// Read constant count
		if offset+4 <= len(bytecode) {
			constCount := binary.LittleEndian.Uint32(bytecode[offset : offset+4])
			offset += 4
			for i := uint32(0); i < constCount; i++ {
				constant, err := vm.readConstant(bytecode, &offset)
				if err == nil {
					fn.Constants = append(fn.Constants, constant)
				}
			}
		}
	}

	vm.functions[name] = fn
}

// RegisterFunctionValue registers a pre-compiled function value with the VM
func (vm *VM) RegisterFunctionValue(name string, fn FunctionValue) {
	// If constants are not provided, extract them from bytecode
	if len(fn.Constants) == 0 && fn.Bytecode != nil {
		offset := 4 // skip magic
		if offset+4 <= len(fn.Bytecode) {
			offset += 4
			if offset+4 <= len(fn.Bytecode) {
				constCount := binary.LittleEndian.Uint32(fn.Bytecode[offset : offset+4])
				offset += 4
				for i := uint32(0); i < constCount; i++ {
					constant, err := vm.readConstant(fn.Bytecode, &offset)
					if err == nil {
						fn.Constants = append(fn.Constants, constant)
					}
				}
			}
		}
	}
	vm.functions[name] = fn
}

// StackSize returns the current size of the stack
func (vm *VM) StackSize() int {
	return len(vm.stack)
}

// IteratorCount returns the number of active iterators
func (vm *VM) IteratorCount() int {
	return len(vm.iterators)
}

// LocalsCount returns the number of local variables
func (vm *VM) LocalsCount() int {
	return len(vm.env.Values)
}

// Reset clears the VM state for reuse
func (vm *VM) Reset() {
	vm.stack = vm.stack[:0]
	vm.env = NewEnvironment(nil)
	vm.constants = vm.constants[:0]
	vm.iterators = make(map[int]*Iterator)
	vm.nextIterID = 0
	vm.pc = 0
	vm.code = nil
	vm.halted = false
}

// registerBuiltins registers all built-in functions
func (vm *VM) registerBuiltins() {
	// time.now() - returns current Unix timestamp
	vm.builtins["time.now"] = func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time.now() takes no arguments, got %d", len(args))
		}
		return IntValue{Val: time.Now().Unix()}, nil
	}

	// now() - alias for time.now()
	vm.builtins["now"] = func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("now() takes no arguments, got %d", len(args))
		}
		return IntValue{Val: time.Now().Unix()}, nil
	}

	// Ok(val) - creates a success ResultValue
	vm.builtins["Ok"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("Ok() takes exactly 1 argument, got %d", len(args))
		}
		return &ResultValue{Val: args[0], IsErr: false}, nil
	}

	// Err(msg) - creates an error ResultValue
	vm.builtins["Err"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("Err() takes exactly 1 argument, got %d", len(args))
		}
		var msg string
		switch v := args[0].(type) {
		case StringValue:
			msg = v.Val
		default:
			msg = fmt.Sprintf("%v", args[0])
		}
		return &ResultValue{ErrMsg: msg, IsErr: true}, nil
	}

	// length() - returns length of array or string
	vm.builtins["length"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("length() takes exactly 1 argument, got %d", len(args))
		}

		switch val := args[0].(type) {
		case ArrayValue:
			return IntValue{Val: int64(len(val.Val))}, nil
		case StringValue:
			return IntValue{Val: int64(len([]rune(val.Val)))}, nil
		default:
			return nil, fmt.Errorf("length() requires array or string, got %T", val)
		}
	}

	// upper() - convert string to uppercase
	vm.builtins["upper"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("upper() takes exactly 1 argument, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("upper() requires a string, got %T", args[0])
		}
		return StringValue{Val: strings.ToUpper(str.Val)}, nil
	}

	// lower() - convert string to lowercase
	vm.builtins["lower"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("lower() takes exactly 1 argument, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("lower() requires a string, got %T", args[0])
		}
		return StringValue{Val: strings.ToLower(str.Val)}, nil
	}

	// trim() - remove leading/trailing whitespace
	vm.builtins["trim"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("trim() takes exactly 1 argument, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("trim() requires a string, got %T", args[0])
		}
		return StringValue{Val: strings.TrimSpace(str.Val)}, nil
	}

	// split() - split string into array
	vm.builtins["split"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("split() takes exactly 2 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("split() first argument must be a string, got %T", args[0])
		}
		delim, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("split() second argument must be a string, got %T", args[1])
		}
		parts := strings.Split(str.Val, delim.Val)
		result := make([]Value, len(parts))
		for i, part := range parts {
			result[i] = StringValue{Val: part}
		}
		return ArrayValue{Val: result}, nil
	}

	// join() - join array into string
	vm.builtins["join"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("join() takes exactly 2 arguments, got %d", len(args))
		}
		arr, ok := args[0].(ArrayValue)
		if !ok {
			return nil, fmt.Errorf("join() first argument must be an array, got %T", args[0])
		}
		delim, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("join() second argument must be a string, got %T", args[1])
		}
		strParts := make([]string, len(arr.Val))
		for i, elem := range arr.Val {
			strParts[i] = valueToString(elem)
		}
		return StringValue{Val: strings.Join(strParts, delim.Val)}, nil
	}

	// contains() - check if string contains substring
	vm.builtins["contains"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("contains() takes exactly 2 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("contains() first argument must be a string, got %T", args[0])
		}
		substr, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("contains() second argument must be a string, got %T", args[1])
		}
		return BoolValue{Val: strings.Contains(str.Val, substr.Val)}, nil
	}

	// replace() - replace occurrences in string
	vm.builtins["replace"] = func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("replace() takes exactly 3 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("replace() first argument must be a string, got %T", args[0])
		}
		old, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("replace() second argument must be a string, got %T", args[1])
		}
		new, ok := args[2].(StringValue)
		if !ok {
			return nil, fmt.Errorf("replace() third argument must be a string, got %T", args[2])
		}
		return StringValue{Val: strings.ReplaceAll(str.Val, old.Val, new.Val)}, nil
	}

	// startsWith() - check if string starts with prefix
	vm.builtins["startsWith"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("startsWith() takes exactly 2 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("startsWith() first argument must be a string, got %T", args[0])
		}
		prefix, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("startsWith() second argument must be a string, got %T", args[1])
		}
		return BoolValue{Val: strings.HasPrefix(str.Val, prefix.Val)}, nil
	}

	// endsWith() - check if string ends with suffix
	vm.builtins["endsWith"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("endsWith() takes exactly 2 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("endsWith() first argument must be a string, got %T", args[0])
		}
		suffix, ok := args[1].(StringValue)
		if !ok {
			return nil, fmt.Errorf("endsWith() second argument must be a string, got %T", args[1])
		}
		return BoolValue{Val: strings.HasSuffix(str.Val, suffix.Val)}, nil
	}

	// repeat() - repeat string n times
	vm.builtins["repeat"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("repeat() takes exactly 2 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("repeat() first argument must be a string, got %T", args[0])
		}
		count, ok := args[1].(IntValue)
		if !ok {
			return nil, fmt.Errorf("repeat() second argument must be an integer, got %T", args[1])
		}
		if count.Val < 0 {
			return nil, fmt.Errorf("repeat() count must be non-negative, got %d", count.Val)
		}
		return StringValue{Val: strings.Repeat(str.Val, int(count.Val))}, nil
	}

	// reverse() - reverse a string or array
	vm.builtins["reverse"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("reverse() takes exactly 1 argument, got %d", len(args))
		}
		switch v := args[0].(type) {
		case StringValue:
			runes := []rune(v.Val)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return StringValue{Val: string(runes)}, nil
		case ArrayValue:
			reversed := make([]Value, len(v.Val))
			for i, j := 0, len(v.Val)-1; j >= 0; i, j = i+1, j-1 {
				reversed[i] = v.Val[j]
			}
			return ArrayValue{Val: reversed}, nil
		default:
			return nil, fmt.Errorf("reverse() argument must be a string or array, got %T", args[0])
		}
	}

	// substring() - get substring
	vm.builtins["substring"] = func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("substring() takes exactly 3 arguments, got %d", len(args))
		}
		str, ok := args[0].(StringValue)
		if !ok {
			return nil, fmt.Errorf("substring() first argument must be a string, got %T", args[0])
		}
		start, ok := args[1].(IntValue)
		if !ok {
			return nil, fmt.Errorf("substring() second argument must be an integer, got %T", args[1])
		}
		end, ok := args[2].(IntValue)
		if !ok {
			return nil, fmt.Errorf("substring() third argument must be an integer, got %T", args[2])
		}
		if start.Val < 0 || end.Val < 0 {
			return nil, fmt.Errorf("substring() indices must be non-negative")
		}
		if start.Val > end.Val {
			return nil, fmt.Errorf("substring() start index must be less than or equal to end index")
		}
		runes := []rune(str.Val)
		strLen := int64(len(runes))
		if start.Val > strLen {
			return nil, fmt.Errorf("substring() start index out of bounds: %d (length %d)", start.Val, strLen)
		}
		if end.Val > strLen {
			return nil, fmt.Errorf("substring() end index out of bounds: %d (length %d)", end.Val, strLen)
		}
		return StringValue{Val: string(runes[start.Val:end.Val])}, nil
	}

	// print(val...) - print values to stdout
	vm.builtins["print"] = func(args []Value) (Value, error) {
		for i, arg := range args {
			fmt.Printf("%v", valueToInterface(arg))
			if i < len(args)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
		return NullValue{}, nil
	}

	// toString(val) - convert value to string
	vm.builtins["toString"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toString() takes exactly 1 argument, got %d", len(args))
		}
		return StringValue{Val: valueToString(args[0])}, nil
	}

	// str(val) - alias for toString, convert value to string
	vm.builtins["str"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("str() takes exactly 1 argument, got %d", len(args))
		}
		return StringValue{Val: valueToString(args[0])}, nil
	}

	// typeOf(val) - get type of value as string
	vm.builtins["typeOf"] = func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("typeOf() takes exactly 1 argument, got %d", len(args))
		}
		return StringValue{Val: args[0].Type()}, nil
	}

	// writeFile(path, data) - write file to disk
	vm.builtins["writeFile"] = func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("writeFile() takes exactly 2 arguments, got %d", len(args))
		}
		pathVal := args[0]
		pathStr, ok := pathVal.(StringValue)
		if !ok {
			return nil, fmt.Errorf("writeFile() expects string path, got %T", pathVal)
		}

		var bytes []byte
		switch v := args[1].(type) {
		case StringValue:
			bytes = []byte(v.Val)
		case ArrayValue:
			bytes = make([]byte, len(v.Val))
			for i, val := range v.Val {
				if n, ok := val.(IntValue); ok {
					bytes[i] = byte(n.Val)
				} else {
					return nil, fmt.Errorf("writeFile() data array must contain ints, got %T at index %d", val, i)
				}
			}
		default:
			return nil, fmt.Errorf("writeFile() data must be str or [int], got %T", args[1])
		}

		if err := os.WriteFile(pathStr.Val, bytes, 0644); err != nil {
			return nil, err
		}
		return NullValue{}, nil
	}
}

// valueToString converts a Value to a string representation
func valueToString(val Value) string {
	switch v := val.(type) {
	case StringValue:
		return v.Val
	case IntValue:
		return intToString(v.Val)
	case FloatValue:
		return floatToString(v.Val)
	case BoolValue:
		if v.Val {
			return "true"
		}
		return "false"
	case NullValue:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// intToString converts an int64 to string
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// floatToString converts a float64 to string
func floatToString(f float64) string {
	return fmt.Sprintf("%g", f)
}

// SetMaxSteps sets the maximum number of execution steps.
// 0 means unlimited (default). Use this to prevent infinite loops.
func (vm *VM) SetMaxSteps(maxSteps int) {
	vm.maxSteps = maxSteps
}

// SetWebSocketHandler sets the WebSocket handler for WS operations
func (vm *VM) SetWebSocketHandler(handler WebSocketHandler) {
	vm.wsHandler = handler
}

// execWsSend sends a message to the current WebSocket connection
func (vm *VM) execWsSend() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	msg, err := vm.Pop()
	if err != nil {
		return err
	}

	// Convert Value to interface{} for sending
	data := valueToInterface(msg)
	if err := vm.wsHandler.Send(data); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsBroadcast broadcasts a message to all connections
func (vm *VM) execWsBroadcast() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	msg, err := vm.Pop()
	if err != nil {
		return err
	}

	data := valueToInterface(msg)
	if err := vm.wsHandler.Broadcast(data); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsBroadcastRoom broadcasts a message to a specific room
func (vm *VM) execWsBroadcastRoom() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	msg, err := vm.Pop()
	if err != nil {
		return err
	}

	roomVal, err := vm.Pop()
	if err != nil {
		return err
	}

	room, ok := roomVal.(StringValue)
	if !ok {
		return fmt.Errorf("room name must be a string, got %T", roomVal)
	}

	data := valueToInterface(msg)
	if err := vm.wsHandler.BroadcastToRoom(room.Val, data); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsJoinRoom joins a WebSocket room
func (vm *VM) execWsJoinRoom() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	roomVal, err := vm.Pop()
	if err != nil {
		return err
	}

	room, ok := roomVal.(StringValue)
	if !ok {
		return fmt.Errorf("room name must be a string, got %T", roomVal)
	}

	if err := vm.wsHandler.JoinRoom(room.Val); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsLeaveRoom leaves a WebSocket room
func (vm *VM) execWsLeaveRoom() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	roomVal, err := vm.Pop()
	if err != nil {
		return err
	}

	room, ok := roomVal.(StringValue)
	if !ok {
		return fmt.Errorf("room name must be a string, got %T", roomVal)
	}

	if err := vm.wsHandler.LeaveRoom(room.Val); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsClose closes the WebSocket connection
func (vm *VM) execWsClose() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	reasonVal, err := vm.Pop()
	if err != nil {
		return err
	}

	reason := ""
	if str, ok := reasonVal.(StringValue); ok {
		reason = str.Val
	}

	if err := vm.wsHandler.Close(reason); err != nil {
		return err
	}
	// Push null so POP after expression statement works correctly
	vm.Push(NullValue{})
	return nil
}

// execWsGetRooms gets the list of rooms
func (vm *VM) execWsGetRooms() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	rooms := vm.wsHandler.GetRooms()

	// Convert to ArrayValue
	arr := make([]Value, len(rooms))
	for i, room := range rooms {
		arr[i] = StringValue{Val: room}
	}

	vm.Push(ArrayValue{Val: arr})
	return nil
}

// execWsGetClients gets the clients in a room
func (vm *VM) execWsGetClients() error {
	if vm.wsHandler == nil {
		return fmt.Errorf("WebSocket handler not available")
	}

	roomVal, err := vm.Pop()
	if err != nil {
		return err
	}

	room, ok := roomVal.(StringValue)
	if !ok {
		return fmt.Errorf("room name must be a string, got %T", roomVal)
	}

	clients := vm.wsHandler.GetRoomClients(room.Val)

	// Convert to ArrayValue
	arr := make([]Value, len(clients))
	for i, client := range clients {
		arr[i] = StringValue{Val: client}
	}

	vm.Push(ArrayValue{Val: arr})
	return nil
}

// execWsGetConnCount gets the total connection count
func (vm *VM) execWsGetConnCount() error {
	if vm.wsHandler == nil {
		// Return 0 if no handler (graceful degradation)
		vm.Push(IntValue{Val: 0})
		return nil
	}

	count := vm.wsHandler.GetConnectionCount()
	vm.Push(IntValue{Val: int64(count)})
	return nil
}

// execWsGetUptime gets the server uptime in seconds
func (vm *VM) execWsGetUptime() error {
	if vm.wsHandler == nil {
		// Return 0 if no handler (graceful degradation)
		vm.Push(IntValue{Val: 0})
		return nil
	}

	uptime := vm.wsHandler.GetUptime()
	vm.Push(IntValue{Val: uptime})
	return nil
}

// valueToInterface converts a VM Value to a Go interface{}
func valueToInterface(v Value) interface{} {
	switch val := v.(type) {
	case IntValue:
		return val.Val
	case FloatValue:
		return val.Val
	case StringValue:
		return val.Val
	case BoolValue:
		return val.Val
	case NullValue:
		return nil
	case ArrayValue:
		result := make([]interface{}, len(val.Val))
		for i, elem := range val.Val {
			result[i] = valueToInterface(elem)
		}
		return result
	case ObjectValue:
		result := make(map[string]interface{})
		for k, v := range val.Val {
			result[k] = valueToInterface(v)
		}
		return result
	default:
		return nil
	}
}

// execAsync executes an async block and creates a future
// The async body bytecode follows immediately after the opcode
// Format: OpAsync [bodyLen:4 bytes] [body bytecode]
func (vm *VM) execAsync() error {
	// Read body length
	bodyLen, err := vm.readOperand()
	if err != nil {
		return err
	}

	// Capture the async body bytecode
	if vm.pc+int(bodyLen) > len(vm.code) {
		return fmt.Errorf("async body extends beyond bytecode")
	}
	asyncBody := make([]byte, bodyLen)
	copy(asyncBody, vm.code[vm.pc:vm.pc+int(bodyLen)])

	// Skip past the async body in the main execution
	vm.pc += int(bodyLen)

	// Create a future and execute async body in a goroutine
	future := &FutureValue{
		Done: make(chan struct{}),
	}

	// Copy constants to avoid sharing mutable slice reference
	constantsCopy := make([]Value, len(vm.constants))
	copy(constantsCopy, vm.constants)

	// Snapshot maps before launching goroutine to avoid concurrent reads
	localsCopy := make(map[string]Value, len(vm.env.Values))
	for k, v := range vm.env.Values {
		localsCopy[k] = v
	}
	globalsCopy := make(map[string]Value, len(vm.globals))
	for k, v := range vm.globals {
		globalsCopy[k] = v
	}
	builtinsCopy := make(map[string]BuiltinFunc, len(vm.builtins))
	for k, v := range vm.builtins {
		builtinsCopy[k] = v
	}

	go func() {
		defer close(future.Done)
		defer func() {
			if r := recover(); r != nil {
				future.Error = fmt.Errorf("async panic: %v", r)
			}
		}()

		// Create a new VM for the async execution
		asyncVM := NewVM()
		asyncVM.constants = constantsCopy
		asyncVM.env = &Environment{Values: localsCopy}
		asyncVM.globals = globalsCopy
		asyncVM.builtins = builtinsCopy

		// Execute the async body using raw instructions (no GLYP header)
		result, execErr := asyncVM.executeRaw(asyncBody)
		if execErr != nil {
			future.Error = execErr
		} else {
			future.Result = result
		}
	}()

	// Push the future onto the stack
	vm.Push(future)
	return nil
}

// execAwait waits for a future to resolve and pushes its result
func (vm *VM) execAwait() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}

	future, ok := val.(*FutureValue)
	if !ok {
		// If it's not a future, just return the value as-is
		// This allows await to work with non-future values
		vm.Push(val)
		return nil
	}

	// Wait for the future to resolve
	result, err := future.Await()
	if err != nil {
		return err
	}

	vm.Push(result)
	return nil
}

// execTry handles the try expression - unwraps ResultValue, propagates errors
func (vm *VM) execTry() error {
	val, err := vm.Pop()
	if err != nil {
		return err
	}

	// If the value is a ResultValue, unwrap it or propagate the error
	if rv, ok := val.(*ResultValue); ok {
		if rv.IsErr {
			// Propagate the error with ERROR_VALUE: prefix for server HTTP mapping
			return fmt.Errorf("ERROR_VALUE:%s", rv.ErrMsg)
		}
		// Unwrap the success value
		vm.Push(rv.Val)
		return nil
	}

	// For non-ResultValue types, just pass through
	vm.Push(val)
	return nil
}
