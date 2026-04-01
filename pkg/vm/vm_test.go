package vm

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

func TestNewVM(t *testing.T) {
	vm := NewVM()
	if vm == nil {
		t.Fatal("NewVM() returned nil")
	}
}

func TestVMStackOperations(t *testing.T) {
	vm := NewVM()

	// Test push
	vm.Push(IntValue{Val: 42})
	if len(vm.stack) != 1 {
		t.Errorf("Expected stack length 1, got %d", len(vm.stack))
	}

	// Test pop
	val, err := vm.Pop()
	if err != nil {
		t.Fatalf("Pop() error: %v", err)
	}

	if intVal, ok := val.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", val)
	}

	// Test underflow
	_, err = vm.Pop()
	if err == nil {
		t.Error("Expected error on stack underflow")
	}
}

// Helper function to create bytecode header
func createBytecodeHeader(constants []Value) []byte {
	bytecode := []byte{0x47, 0x4C, 0x59, 0x50}          // Magic "GLYP"
	bytecode = append(bytecode, 0x01, 0x00, 0x00, 0x00) // Version 1

	// Constant count
	constCount := make([]byte, 4)
	binary.LittleEndian.PutUint32(constCount, uint32(len(constants)))
	bytecode = append(bytecode, constCount...)

	// Serialize constants
	for _, c := range constants {
		bytecode = append(bytecode, serializeConstant(c)...)
	}

	// String pool (empty)
	bytecode = append(bytecode, 0x00, 0x00, 0x00, 0x00) // String pool count = 0

	// Instruction count (0 for now, will be added later)
	bytecode = append(bytecode, 0x00, 0x00, 0x00, 0x00)

	return bytecode
}

func serializeConstant(c Value) []byte {
	switch v := c.(type) {
	case NullValue:
		return []byte{0x00}
	case IntValue:
		buf := make([]byte, 9)
		buf[0] = 0x01
		binary.LittleEndian.PutUint64(buf[1:], uint64(v.Val))
		return buf
	case FloatValue:
		buf := make([]byte, 9)
		buf[0] = 0x02
		binary.LittleEndian.PutUint64(buf[1:], math.Float64bits(v.Val))
		return buf
	case BoolValue:
		if v.Val {
			return []byte{0x03, 0x01}
		}
		return []byte{0x03, 0x00}
	case StringValue:
		buf := []byte{0x04}
		length := make([]byte, 4)
		binary.LittleEndian.PutUint32(length, uint32(len(v.Val)))
		buf = append(buf, length...)
		buf = append(buf, []byte(v.Val)...)
		return buf
	}
	return nil
}

func addInstruction(bytecode []byte, opcode Opcode, operand *uint32) []byte {
	bytecode = append(bytecode, byte(opcode))
	if operand != nil {
		op := make([]byte, 4)
		binary.LittleEndian.PutUint32(op, *operand)
		bytecode = append(bytecode, op...)
	}
	return bytecode
}

func TestOpPush(t *testing.T) {
	constants := []Value{IntValue{Val: 42}}
	bytecode := createBytecodeHeader(constants)

	operand := uint32(0)
	bytecode = addInstruction(bytecode, OpPush, &operand)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpAdd(t *testing.T) {
	constants := []Value{IntValue{Val: 10}, IntValue{Val: 32}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 10
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 32
	bytecode = addInstruction(bytecode, OpAdd, nil)        // Add
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpSub(t *testing.T) {
	constants := []Value{IntValue{Val: 50}, IntValue{Val: 8}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 50
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 8
	bytecode = addInstruction(bytecode, OpSub, nil)        // Subtract
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpMul(t *testing.T) {
	constants := []Value{IntValue{Val: 6}, IntValue{Val: 7}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 6
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 7
	bytecode = addInstruction(bytecode, OpMul, nil)        // Multiply
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpDiv(t *testing.T) {
	constants := []Value{IntValue{Val: 84}, IntValue{Val: 2}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 84
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 2
	bytecode = addInstruction(bytecode, OpDiv, nil)        // Divide
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpDivByZero(t *testing.T) {
	constants := []Value{IntValue{Val: 42}, IntValue{Val: 0}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 42
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 0
	bytecode = addInstruction(bytecode, OpDiv, nil)        // Divide
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected division by zero error")
	}
}

func TestOpEq(t *testing.T) {
	constants := []Value{IntValue{Val: 42}, IntValue{Val: 42}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 42
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 42
	bytecode = addInstruction(bytecode, OpEq, nil)         // Equal
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if boolVal, ok := result.(BoolValue); !ok || !boolVal.Val {
		t.Errorf("Expected BoolValue{true}, got %v", result)
	}
}

func TestOpNe(t *testing.T) {
	constants := []Value{IntValue{Val: 42}, IntValue{Val: 24}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 42
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 24
	bytecode = addInstruction(bytecode, OpNe, nil)         // Not Equal
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if boolVal, ok := result.(BoolValue); !ok || !boolVal.Val {
		t.Errorf("Expected BoolValue{true}, got %v", result)
	}
}

func TestOpLt(t *testing.T) {
	constants := []Value{IntValue{Val: 10}, IntValue{Val: 42}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 10
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 42
	bytecode = addInstruction(bytecode, OpLt, nil)         // Less Than
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if boolVal, ok := result.(BoolValue); !ok || !boolVal.Val {
		t.Errorf("Expected BoolValue{true}, got %v", result)
	}
}

func TestOpStoreAndLoadVar(t *testing.T) {
	constants := []Value{IntValue{Val: 42}, StringValue{Val: "x"}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0)     // Push 42
	bytecode = addInstruction(bytecode, OpStoreVar, &operand1) // Store to "x"
	bytecode = addInstruction(bytecode, OpLoadVar, &operand1)  // Load "x"
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 42 {
		t.Errorf("Expected IntValue{42}, got %v", result)
	}
}

func TestOpBuildArray(t *testing.T) {
	constants := []Value{IntValue{Val: 1}, IntValue{Val: 2}, IntValue{Val: 3}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand3 := uint32(3)
	bytecode = addInstruction(bytecode, OpPush, &operand0)       // Push 1
	bytecode = addInstruction(bytecode, OpPush, &operand1)       // Push 2
	bytecode = addInstruction(bytecode, OpPush, &operand2)       // Push 3
	bytecode = addInstruction(bytecode, OpBuildArray, &operand3) // Build array of 3 elements
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	arrVal, ok := result.(ArrayValue)
	if !ok {
		t.Fatalf("Expected ArrayValue, got %T", result)
	}

	if len(arrVal.Val) != 3 {
		t.Errorf("Expected array length 3, got %d", len(arrVal.Val))
	}

	expected := []int64{1, 2, 3}
	for i, exp := range expected {
		if intVal, ok := arrVal.Val[i].(IntValue); !ok || intVal.Val != exp {
			t.Errorf("Expected element %d to be %d, got %v", i, exp, arrVal.Val[i])
		}
	}
}

func TestOpBuildObject(t *testing.T) {
	constants := []Value{
		StringValue{Val: "name"},
		StringValue{Val: "Alice"},
		StringValue{Val: "age"},
		IntValue{Val: 30},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand3 := uint32(3)
	operand2fields := uint32(2)

	bytecode = addInstruction(bytecode, OpPush, &operand0)              // Push "name"
	bytecode = addInstruction(bytecode, OpPush, &operand1)              // Push "Alice"
	bytecode = addInstruction(bytecode, OpPush, &operand2)              // Push "age"
	bytecode = addInstruction(bytecode, OpPush, &operand3)              // Push 30
	bytecode = addInstruction(bytecode, OpBuildObject, &operand2fields) // Build object with 2 fields
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	objVal, ok := result.(ObjectValue)
	if !ok {
		t.Fatalf("Expected ObjectValue, got %T", result)
	}

	if len(objVal.Val) != 2 {
		t.Errorf("Expected object with 2 fields, got %d", len(objVal.Val))
	}

	if nameVal, ok := objVal.Val["name"].(StringValue); !ok || nameVal.Val != "Alice" {
		t.Errorf("Expected name='Alice', got %v", objVal.Val["name"])
	}

	if ageVal, ok := objVal.Val["age"].(IntValue); !ok || ageVal.Val != 30 {
		t.Errorf("Expected age=30, got %v", objVal.Val["age"])
	}
}

func TestOpHttpReturn(t *testing.T) {
	constants := []Value{StringValue{Val: "Hello, World!"}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push "Hello, World!"
	bytecode = addInstruction(bytecode, OpHttpReturn, nil) // HTTP Return

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if strVal, ok := result.(StringValue); !ok || strVal.Val != "Hello, World!" {
		t.Errorf("Expected StringValue{\"Hello, World!\"}, got %v", result)
	}
}

func TestArithmeticProgram(t *testing.T) {
	// Test: (10 + 5) * 2 - 3 = 27
	constants := []Value{
		IntValue{Val: 10},
		IntValue{Val: 5},
		IntValue{Val: 2},
		IntValue{Val: 3},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand3 := uint32(3)

	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 10
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 5
	bytecode = addInstruction(bytecode, OpAdd, nil)        // 10 + 5 = 15
	bytecode = addInstruction(bytecode, OpPush, &operand2) // Push 2
	bytecode = addInstruction(bytecode, OpMul, nil)        // 15 * 2 = 30
	bytecode = addInstruction(bytecode, OpPush, &operand3) // Push 3
	bytecode = addInstruction(bytecode, OpSub, nil)        // 30 - 3 = 27
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if intVal, ok := result.(IntValue); !ok || intVal.Val != 27 {
		t.Errorf("Expected IntValue{27}, got %v", result)
	}
}

func TestStringConcatenation(t *testing.T) {
	constants := []Value{
		StringValue{Val: "Hello, "},
		StringValue{Val: "World!"},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push "Hello, "
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push "World!"
	bytecode = addInstruction(bytecode, OpAdd, nil)        // Concatenate
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if strVal, ok := result.(StringValue); !ok || strVal.Val != "Hello, World!" {
		t.Errorf("Expected StringValue{\"Hello, World!\"}, got %v", result)
	}
}

func TestFloatArithmetic(t *testing.T) {
	constants := []Value{
		FloatValue{Val: 3.14},
		FloatValue{Val: 2.0},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 3.14
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 2.0
	bytecode = addInstruction(bytecode, OpMul, nil)        // 3.14 * 2.0
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if floatVal, ok := result.(FloatValue); !ok || math.Abs(floatVal.Val-6.28) > 0.01 {
		t.Errorf("Expected FloatValue{6.28}, got %v", result)
	}
}

func TestErrorInvalidBytecode(t *testing.T) {
	tests := []struct {
		name     string
		bytecode []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x41, 0x49}},
		{"invalid magic", []byte{0x00, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			_, err := vm.Execute(tt.bytecode)
			if err == nil {
				t.Error("Expected error for invalid bytecode")
			}
		})
	}
}

func TestErrorUndefinedVariable(t *testing.T) {
	constants := []Value{StringValue{Val: "undefined"}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	bytecode = addInstruction(bytecode, OpLoadVar, &operand0) // Load undefined variable
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected error for undefined variable")
	}
}

func TestErrorStackUnderflow(t *testing.T) {
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)

	bytecode = addInstruction(bytecode, OpAdd, nil) // Try to add with empty stack
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected error for stack underflow")
	}
}

func TestRouteExecution(t *testing.T) {
	// Simulate a simple route: return { "message": "Hello" }
	constants := []Value{
		StringValue{Val: "message"},
		StringValue{Val: "Hello"},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operandCount := uint32(1)

	bytecode = addInstruction(bytecode, OpPush, &operand0)            // Push "message"
	bytecode = addInstruction(bytecode, OpPush, &operand1)            // Push "Hello"
	bytecode = addInstruction(bytecode, OpBuildObject, &operandCount) // Build object with 1 field
	bytecode = addInstruction(bytecode, OpHttpReturn, nil)            // HTTP Return

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	objVal, ok := result.(ObjectValue)
	if !ok {
		t.Fatalf("Expected ObjectValue, got %T", result)
	}

	if msgVal, ok := objVal.Val["message"].(StringValue); !ok || msgVal.Val != "Hello" {
		t.Errorf("Expected message='Hello', got %v", objVal.Val["message"])
	}
}

func TestOpGt(t *testing.T) {
	constants := []Value{IntValue{Val: 42}, IntValue{Val: 10}}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 42
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 10
	bytecode = addInstruction(bytecode, OpGt, nil)         // Greater Than
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if boolVal, ok := result.(BoolValue); !ok || !boolVal.Val {
		t.Errorf("Expected BoolValue{true}, got %v", result)
	}
}

func TestOpGe(t *testing.T) {
	tests := []struct {
		name     string
		a        int64
		b        int64
		expected bool
	}{
		{"greater", 42, 10, true},
		{"equal", 42, 42, true},
		{"less", 10, 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []Value{IntValue{Val: tt.a}, IntValue{Val: tt.b}}
			bytecode := createBytecodeHeader(constants)

			operand0 := uint32(0)
			operand1 := uint32(1)
			bytecode = addInstruction(bytecode, OpPush, &operand0) // Push a
			bytecode = addInstruction(bytecode, OpPush, &operand1) // Push b
			bytecode = addInstruction(bytecode, OpGe, nil)         // Greater Than or Equal
			bytecode = addInstruction(bytecode, OpHalt, nil)

			vm := NewVM()
			result, err := vm.Execute(bytecode)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected BoolValue{%v}, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpLe(t *testing.T) {
	tests := []struct {
		name     string
		a        int64
		b        int64
		expected bool
	}{
		{"less", 10, 42, true},
		{"equal", 42, 42, true},
		{"greater", 42, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []Value{IntValue{Val: tt.a}, IntValue{Val: tt.b}}
			bytecode := createBytecodeHeader(constants)

			operand0 := uint32(0)
			operand1 := uint32(1)
			bytecode = addInstruction(bytecode, OpPush, &operand0) // Push a
			bytecode = addInstruction(bytecode, OpPush, &operand1) // Push b
			bytecode = addInstruction(bytecode, OpLe, nil)         // Less Than or Equal
			bytecode = addInstruction(bytecode, OpHalt, nil)

			vm := NewVM()
			result, err := vm.Execute(bytecode)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected BoolValue{%v}, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpAnd(t *testing.T) {
	tests := []struct {
		name     string
		a        bool
		b        bool
		expected bool
	}{
		{"true && true", true, true, true},
		{"true && false", true, false, false},
		{"false && true", false, true, false},
		{"false && false", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []Value{BoolValue{Val: tt.a}, BoolValue{Val: tt.b}}
			bytecode := createBytecodeHeader(constants)

			operand0 := uint32(0)
			operand1 := uint32(1)
			bytecode = addInstruction(bytecode, OpPush, &operand0) // Push a
			bytecode = addInstruction(bytecode, OpPush, &operand1) // Push b
			bytecode = addInstruction(bytecode, OpAnd, nil)        // Logical AND
			bytecode = addInstruction(bytecode, OpHalt, nil)

			vm := NewVM()
			result, err := vm.Execute(bytecode)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected BoolValue{%v}, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpOr(t *testing.T) {
	tests := []struct {
		name     string
		a        bool
		b        bool
		expected bool
	}{
		{"true || true", true, true, true},
		{"true || false", true, false, true},
		{"false || true", false, true, true},
		{"false || false", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []Value{BoolValue{Val: tt.a}, BoolValue{Val: tt.b}}
			bytecode := createBytecodeHeader(constants)

			operand0 := uint32(0)
			operand1 := uint32(1)
			bytecode = addInstruction(bytecode, OpPush, &operand0) // Push a
			bytecode = addInstruction(bytecode, OpPush, &operand1) // Push b
			bytecode = addInstruction(bytecode, OpOr, nil)         // Logical OR
			bytecode = addInstruction(bytecode, OpHalt, nil)

			vm := NewVM()
			result, err := vm.Execute(bytecode)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected BoolValue{%v}, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpNot(t *testing.T) {
	tests := []struct {
		name     string
		a        bool
		expected bool
	}{
		{"!true", true, false},
		{"!false", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []Value{BoolValue{Val: tt.a}}
			bytecode := createBytecodeHeader(constants)

			operand0 := uint32(0)
			bytecode = addInstruction(bytecode, OpPush, &operand0) // Push a
			bytecode = addInstruction(bytecode, OpNot, nil)        // Logical NOT
			bytecode = addInstruction(bytecode, OpHalt, nil)

			vm := NewVM()
			result, err := vm.Execute(bytecode)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected BoolValue{%v}, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpJumpIfFalse(t *testing.T) {
	constants := []Value{
		BoolValue{Val: false},
		IntValue{Val: 1},
		IntValue{Val: 2},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)

	startOffset := len(bytecode)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push false - offset: startOffset
	// After this instruction: startOffset + 5 (1 byte opcode + 4 byte operand)
	jumpOffset := startOffset + 5 + 5 + 5 // Jump to after "Push 1" instruction
	jumpTarget := uint32(jumpOffset)
	bytecode = addInstruction(bytecode, OpJumpIfFalse, &jumpTarget) // offset: startOffset + 5
	bytecode = addInstruction(bytecode, OpPush, &operand1)          // Push 1 (should be skipped) - offset: startOffset + 10
	// jumpTarget points here: startOffset + 15
	bytecode = addInstruction(bytecode, OpPush, &operand2) // Push 2 - offset: startOffset + 15
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Should have pushed 2 (not 1) since condition was false and jump occurred
	if intVal, ok := result.(IntValue); !ok || intVal.Val != 2 {
		t.Errorf("Expected IntValue{2}, got %v", result)
	}
}

func TestOpJumpIfTrue(t *testing.T) {
	constants := []Value{
		BoolValue{Val: true},
		IntValue{Val: 1},
		IntValue{Val: 2},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)

	startOffset := len(bytecode)
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push true - offset: startOffset
	// After this instruction: startOffset + 5 (1 byte opcode + 4 byte operand)
	jumpOffset := startOffset + 5 + 5 + 5 // Jump to after "Push 1" instruction
	jumpTarget := uint32(jumpOffset)
	bytecode = addInstruction(bytecode, OpJumpIfTrue, &jumpTarget) // offset: startOffset + 5
	bytecode = addInstruction(bytecode, OpPush, &operand1)         // Push 1 (should be skipped) - offset: startOffset + 10
	// jumpTarget points here: startOffset + 15
	bytecode = addInstruction(bytecode, OpPush, &operand2) // Push 2 - offset: startOffset + 15
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	// Should have pushed 2 (not 1) since condition was true and jump occurred
	if intVal, ok := result.(IntValue); !ok || intVal.Val != 2 {
		t.Errorf("Expected IntValue{2}, got %v", result)
	}
}

func TestOpGetField(t *testing.T) {
	constants := []Value{
		StringValue{Val: "name"},
		StringValue{Val: "Alice"},
		StringValue{Val: "age"},
		IntValue{Val: 30},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand3 := uint32(3)
	operand2fields := uint32(2)

	// Build object: {name: "Alice", age: 30}
	bytecode = addInstruction(bytecode, OpPush, &operand0)              // Push "name"
	bytecode = addInstruction(bytecode, OpPush, &operand1)              // Push "Alice"
	bytecode = addInstruction(bytecode, OpPush, &operand2)              // Push "age"
	bytecode = addInstruction(bytecode, OpPush, &operand3)              // Push 30
	bytecode = addInstruction(bytecode, OpBuildObject, &operand2fields) // Build object with 2 fields

	// Get field "name"
	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push "name" (field name)
	bytecode = addInstruction(bytecode, OpGetField, nil)   // Get field
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if strVal, ok := result.(StringValue); !ok || strVal.Val != "Alice" {
		t.Errorf("Expected StringValue{\"Alice\"}, got %v", result)
	}
}

func TestOpGetFieldNotFound(t *testing.T) {
	constants := []Value{
		StringValue{Val: "name"},
		StringValue{Val: "Alice"},
		StringValue{Val: "missing"},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand1field := uint32(1)

	// Build object: {name: "Alice"}
	bytecode = addInstruction(bytecode, OpPush, &operand0)             // Push "name"
	bytecode = addInstruction(bytecode, OpPush, &operand1)             // Push "Alice"
	bytecode = addInstruction(bytecode, OpBuildObject, &operand1field) // Build object with 1 field

	// Try to get non-existent field "missing"
	bytecode = addInstruction(bytecode, OpPush, &operand2) // Push "missing" (field name)
	bytecode = addInstruction(bytecode, OpGetField, nil)   // Get field
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected error for non-existent field")
	}
}

func TestConditionalExpression(t *testing.T) {
	// Test: (5 > 3) && (10 < 20)
	constants := []Value{
		IntValue{Val: 5},
		IntValue{Val: 3},
		IntValue{Val: 10},
		IntValue{Val: 20},
	}
	bytecode := createBytecodeHeader(constants)

	operand0 := uint32(0)
	operand1 := uint32(1)
	operand2 := uint32(2)
	operand3 := uint32(3)

	bytecode = addInstruction(bytecode, OpPush, &operand0) // Push 5
	bytecode = addInstruction(bytecode, OpPush, &operand1) // Push 3
	bytecode = addInstruction(bytecode, OpGt, nil)         // 5 > 3 = true
	bytecode = addInstruction(bytecode, OpPush, &operand2) // Push 10
	bytecode = addInstruction(bytecode, OpPush, &operand3) // Push 20
	bytecode = addInstruction(bytecode, OpLt, nil)         // 10 < 20 = true
	bytecode = addInstruction(bytecode, OpAnd, nil)        // true && true = true
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if boolVal, ok := result.(BoolValue); !ok || !boolVal.Val {
		t.Errorf("Expected BoolValue{true}, got %v", result)
	}
}

// Test built-in string manipulation functions

func TestBuiltinString_Upper(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["upper"]([]Value{StringValue{Val: "hello world"}})
	if err != nil {
		t.Fatalf("upper() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got %v", result)
	}
}

func TestBuiltinString_Lower(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["lower"]([]Value{StringValue{Val: "HELLO WORLD"}})
	if err != nil {
		t.Fatalf("lower() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "hello world" {
		t.Errorf("Expected 'hello world', got %v", result)
	}
}

func TestBuiltinString_Trim(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["trim"]([]Value{StringValue{Val: "  hello world  "}})
	if err != nil {
		t.Fatalf("trim() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "hello world" {
		t.Errorf("Expected 'hello world', got %v", result)
	}
}

func TestBuiltinString_Split(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["split"]([]Value{
		StringValue{Val: "a,b,c"},
		StringValue{Val: ","},
	})
	if err != nil {
		t.Fatalf("split() error: %v", err)
	}
	arrVal, ok := result.(ArrayValue)
	if !ok {
		t.Fatalf("Expected ArrayValue, got %T", result)
	}
	if len(arrVal.Val) != 3 {
		t.Errorf("Expected array length 3, got %d", len(arrVal.Val))
	}
	if arrVal.Val[0].(StringValue).Val != "a" {
		t.Errorf("Expected 'a', got %v", arrVal.Val[0])
	}
	if arrVal.Val[1].(StringValue).Val != "b" {
		t.Errorf("Expected 'b', got %v", arrVal.Val[1])
	}
	if arrVal.Val[2].(StringValue).Val != "c" {
		t.Errorf("Expected 'c', got %v", arrVal.Val[2])
	}
}

func TestBuiltinString_Join(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["join"]([]Value{
		ArrayValue{Val: []Value{
			StringValue{Val: "a"},
			StringValue{Val: "b"},
			StringValue{Val: "c"},
		}},
		StringValue{Val: ","},
	})
	if err != nil {
		t.Fatalf("join() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "a,b,c" {
		t.Errorf("Expected 'a,b,c', got %v", result)
	}
}

func TestBuiltinString_Contains(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"contains_yes", "hello world", "world", true},
		{"contains_no", "hello world", "xyz", false},
		{"contains_empty", "hello", "", true},
		{"contains_exact", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			result, err := vm.builtins["contains"]([]Value{
				StringValue{Val: tt.str},
				StringValue{Val: tt.substr},
			})
			if err != nil {
				t.Fatalf("contains() error: %v", err)
			}
			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinString_Replace(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["replace"]([]Value{
		StringValue{Val: "hello world world"},
		StringValue{Val: "world"},
		StringValue{Val: "universe"},
	})
	if err != nil {
		t.Fatalf("replace() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "hello universe universe" {
		t.Errorf("Expected 'hello universe universe', got %v", result)
	}
}

func TestBuiltinString_StartsWith(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		prefix   string
		expected bool
	}{
		{"starts_yes", "hello world", "hello", true},
		{"starts_no", "hello world", "world", false},
		{"starts_empty", "hello", "", true},
		{"starts_exact", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			result, err := vm.builtins["startsWith"]([]Value{
				StringValue{Val: tt.str},
				StringValue{Val: tt.prefix},
			})
			if err != nil {
				t.Fatalf("startsWith() error: %v", err)
			}
			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinString_EndsWith(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		suffix   string
		expected bool
	}{
		{"ends_yes", "hello world", "world", true},
		{"ends_no", "hello world", "hello", false},
		{"ends_empty", "hello", "", true},
		{"ends_exact", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			result, err := vm.builtins["endsWith"]([]Value{
				StringValue{Val: tt.str},
				StringValue{Val: tt.suffix},
			})
			if err != nil {
				t.Fatalf("endsWith() error: %v", err)
			}
			if boolVal, ok := result.(BoolValue); !ok || boolVal.Val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuiltinString_Repeat(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["repeat"]([]Value{
		StringValue{Val: "ab"},
		IntValue{Val: 3},
	})
	if err != nil {
		t.Fatalf("repeat() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "ababab" {
		t.Errorf("Expected 'ababab', got %v", result)
	}
}

func TestBuiltinString_Reverse(t *testing.T) {
	vm := NewVM()
	result, err := vm.builtins["reverse"]([]Value{
		StringValue{Val: "hello"},
	})
	if err != nil {
		t.Fatalf("reverse() error: %v", err)
	}
	if strVal, ok := result.(StringValue); !ok || strVal.Val != "olleh" {
		t.Errorf("Expected 'olleh', got %v", result)
	}
}

func TestBuiltinString_Substring(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		start    int64
		end      int64
		expected string
	}{
		{"normal_range", "hello world", 0, 5, "hello"},
		{"middle_range", "hello world", 6, 11, "world"},
		{"full_string", "test", 0, 4, "test"},
		{"empty_string", "test", 2, 2, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			result, err := vm.builtins["substring"]([]Value{
				StringValue{Val: tt.str},
				IntValue{Val: tt.start},
				IntValue{Val: tt.end},
			})
			if err != nil {
				t.Fatalf("substring() error: %v", err)
			}
			if strVal, ok := result.(StringValue); !ok || strVal.Val != tt.expected {
				t.Errorf("Expected '%s', got %v", tt.expected, result)
			}
		})
	}

	t.Run("beyond_length", func(t *testing.T) {
		vm := NewVM()
		_, err := vm.builtins["substring"]([]Value{
			StringValue{Val: "test"},
			IntValue{Val: 0},
			IntValue{Val: 10},
		})
		if err == nil {
			t.Error("Expected error for out-of-bounds end index")
		}
	})
}

func TestOpSetField(t *testing.T) {
	constants := []Value{
		StringValue{Val: "x"},
		IntValue{Val: 10},
		IntValue{Val: 42},
	}
	bytecode := createBytecodeHeader(constants)

	c0, c1, c2 := uint32(0), uint32(1), uint32(2)
	count := uint32(1)

	// Build object: {x: 10}
	bytecode = addInstruction(bytecode, OpPush, &c0) // "x"
	bytecode = addInstruction(bytecode, OpPush, &c1) // 10
	bytecode = addInstruction(bytecode, OpBuildObject, &count)

	// Set field "x" = 42
	bytecode = addInstruction(bytecode, OpPush, &c0) // "x"
	bytecode = addInstruction(bytecode, OpPush, &c2) // 42
	bytecode = addInstruction(bytecode, OpSetField, nil)

	// Get field "x"
	bytecode = addInstruction(bytecode, OpPush, &c0) // "x"
	bytecode = addInstruction(bytecode, OpGetField, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if val, ok := result.(IntValue); !ok || val.Val != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestOpMod(t *testing.T) {
	constants := []Value{IntValue{Val: 10}, IntValue{Val: 3}}
	bytecode := createBytecodeHeader(constants)
	c0, c1 := uint32(0), uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = addInstruction(bytecode, OpPush, &c1)
	bytecode = addInstruction(bytecode, OpMod, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if intVal, ok := result.(IntValue); !ok || intVal.Val != 1 {
		t.Errorf("Expected 1, got %v", result)
	}
}

func TestOpSetIndex(t *testing.T) {
	constants := []Value{IntValue{Val: 1}, IntValue{Val: 2}, IntValue{Val: 3}, IntValue{Val: 42}, IntValue{Val: 1}}
	bytecode := createBytecodeHeader(constants)
	c0, c1, c2, c3, c4 := uint32(0), uint32(1), uint32(2), uint32(3), uint32(4)
	
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = addInstruction(bytecode, OpPush, &c1)
	bytecode = addInstruction(bytecode, OpPush, &c2)
	count := uint32(3)
	bytecode = addInstruction(bytecode, OpBuildArray, &count)
	
	bytecode = addInstruction(bytecode, OpPush, &c4) // index 1
	bytecode = addInstruction(bytecode, OpPush, &c3) // value 42
	bytecode = addInstruction(bytecode, OpSetIndex, nil)
	
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	arrVal, ok := result.(ArrayValue)
	if !ok || arrVal.Val[1].(IntValue).Val != 42 {
		t.Errorf("Expected element at index 1 to be 42, got %v", result)
	}
}

func TestOpDefFunc(t *testing.T) {
	constants := []Value{
		StringValue{Val: "myFunc"},
		StringValue{Val: "a"},
	}
	bytecode := createBytecodeHeader(constants)

	bytecode = append(bytecode, byte(OpDefFunc))
	
	nameIdx := make([]byte, 4)
	binary.LittleEndian.PutUint32(nameIdx, 0)
	bytecode = append(bytecode, nameIdx...)
	
	paramCount := make([]byte, 4)
	binary.LittleEndian.PutUint32(paramCount, 1)
	bytecode = append(bytecode, paramCount...)
	
	bodyLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(bodyLength, 2)
	bytecode = append(bytecode, bodyLength...)
	
	param1Idx := make([]byte, 4)
	binary.LittleEndian.PutUint32(param1Idx, 1)
	bytecode = append(bytecode, param1Idx...)
	
	bytecode = append(bytecode, byte(OpPop), byte(OpReturn))

	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	fn, exists := vm.functions["myFunc"]
	if !exists {
		t.Errorf("Function 'myFunc' not registered")
	}
	if len(fn.ParamNames) != 1 || fn.ParamNames[0] != "a" {
		t.Errorf("Unexpected param names: %v", fn.ParamNames)
	}
}

func TestOpSetIndexObject(t *testing.T) {
	constants := []Value{StringValue{Val: "x"}, IntValue{Val: 10}, IntValue{Val: 42}}
	bytecode := createBytecodeHeader(constants)
	c0, c1, c2 := uint32(0), uint32(1), uint32(2)
	
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = addInstruction(bytecode, OpPush, &c1)
	count := uint32(1)
	bytecode = addInstruction(bytecode, OpBuildObject, &count)
	
	bytecode = addInstruction(bytecode, OpPush, &c0) // key "x"
	bytecode = addInstruction(bytecode, OpPush, &c2) // value 42
	bytecode = addInstruction(bytecode, OpSetIndex, nil)
	
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	objVal, ok := result.(ObjectValue)
	if !ok || objVal.Val["x"].(IntValue).Val != 42 {
		t.Errorf("Expected field 'x' to be 42, got %v", result)
	}
}

func TestOpDefFuncErrors(t *testing.T) {
	// Test constant index out of bounds for name
	constants := []Value{StringValue{Val: "myFunc"}}
	bytecode := createBytecodeHeader(constants)
	bytecode = append(bytecode, byte(OpDefFunc))
	nameIdx := make([]byte, 4)
	binary.LittleEndian.PutUint32(nameIdx, 99) // Out of bounds
	bytecode = append(bytecode, nameIdx...)
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected error for name index out of bounds")
	}
}

func TestOpDefFuncTypeErrors(t *testing.T) {
	// 1. Name is not a string
	constants := []Value{IntValue{Val: 123}}
	bytecode := createBytecodeHeader(constants)
	bytecode = append(bytecode, byte(OpDefFunc))
	nameIdx := make([]byte, 4)
	binary.LittleEndian.PutUint32(nameIdx, 0) // Index 0 is IntValue
	bytecode = append(bytecode, nameIdx...)
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil || err.Error() != "function name must be a string" {
		t.Errorf("Expected 'function name must be a string' error, got %v", err)
	}
}

func TestOpSetIndexErrors(t *testing.T) {
	// Cannot set index on integer
	constants := []Value{IntValue{Val: 123}, IntValue{Val: 0}, IntValue{Val: 42}}
	bytecode := createBytecodeHeader(constants)
	c0, c1, c2 := uint32(0), uint32(1), uint32(2)
	
	bytecode = addInstruction(bytecode, OpPush, &c0) // obj (int)
	bytecode = addInstruction(bytecode, OpPush, &c1) // index
	bytecode = addInstruction(bytecode, OpPush, &c2) // value
	bytecode = addInstruction(bytecode, OpSetIndex, nil)
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil {
		t.Error("Expected error for setting index on non-collection")
	}
}

func TestOpBuildObjectKeyError(t *testing.T) {
	constants := []Value{IntValue{Val: 123}, IntValue{Val: 456}}
	bytecode := createBytecodeHeader(constants)
	c0, c1 := uint32(0), uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &c0) // key (int)
	bytecode = addInstruction(bytecode, OpPush, &c1) // value
	count := uint32(1)
	bytecode = addInstruction(bytecode, OpBuildObject, &count)
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil || err.Error() != "object key must be a string" {
		t.Errorf("Expected 'object key must be a string' error, got %v", err)
	}
}

func TestOpGetFieldErrors(t *testing.T) {
	// 1. Key is not a string
	constants := []Value{StringValue{Val: "x"}, IntValue{Val: 10}, IntValue{Val: 123}}
	bytecode := createBytecodeHeader(constants)
	c0, c1, c2 := uint32(0), uint32(1), uint32(2)
	
	bytecode = addInstruction(bytecode, OpPush, &c0) // key
	bytecode = addInstruction(bytecode, OpPush, &c1) // value
	count := uint32(1)
	bytecode = addInstruction(bytecode, OpBuildObject, &count)
	
	bytecode = addInstruction(bytecode, OpPush, &c2) // key (int)
	bytecode = addInstruction(bytecode, OpGetField, nil)
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err == nil || err.Error() != "type error: field name must be a string" {
		t.Errorf("Expected 'type error: field name must be a string' error, got %v", err)
	}

	// 2. Not an object
	constants = []Value{StringValue{Val: "x"}, IntValue{Val: 123}}
	bytecode = createBytecodeHeader(constants)
	c0, c1 = uint32(0), uint32(1)
	bytecode = addInstruction(bytecode, OpPush, &c1) // obj (int)
	bytecode = addInstruction(bytecode, OpPush, &c0) // key
	bytecode = addInstruction(bytecode, OpGetField, nil)
	
	vm = NewVM()
	_, err = vm.Execute(bytecode)
	if err == nil || err.Error() != "type error: can only get field from object" {
		t.Errorf("Expected 'type error: can only get field from object' error, got %v", err)
	}
}

func TestIssue7FinalCoverage(t *testing.T) {
	// 1. execSetField: key not string
	{
		constants := []Value{StringValue{Val: "x"}, IntValue{Val: 10}, IntValue{Val: 42}, IntValue{Val: 99}}
		bytecode := createBytecodeHeader(constants)
		c0, c1, c2, c3 := uint32(0), uint32(1), uint32(2), uint32(3)
		bytecode = addInstruction(bytecode, OpPush, &c0) // key
		bytecode = addInstruction(bytecode, OpPush, &c1) // value
		count := uint32(1)
		bytecode = addInstruction(bytecode, OpBuildObject, &count)
		bytecode = addInstruction(bytecode, OpPush, &c3) // key (int)
		bytecode = addInstruction(bytecode, OpPush, &c2) // value
		bytecode = addInstruction(bytecode, OpSetField, nil)
		vm := NewVM()
		_, err := vm.Execute(bytecode)
		if err == nil || !strings.Contains(err.Error(), "field name must be a string") {
			t.Errorf("Expected 'field name must be a string' error, got %v", err)
		}
	}
	// 2. execSetField: not an object
	{
		constants := []Value{StringValue{Val: "x"}, IntValue{Val: 42}, IntValue{Val: 123}}
		bytecode := createBytecodeHeader(constants)
		c0, c1, c2 := uint32(0), uint32(1), uint32(2)
		bytecode = addInstruction(bytecode, OpPush, &c2) // obj (int)
		bytecode = addInstruction(bytecode, OpPush, &c0) // key
		bytecode = addInstruction(bytecode, OpPush, &c1) // value
		bytecode = addInstruction(bytecode, OpSetField, nil)
		vm := NewVM()
		_, err := vm.Execute(bytecode)
		if err == nil || !strings.Contains(err.Error(), "can only set field on object") {
			t.Errorf("Expected 'can only set field on object' error, got %v", err)
		}
	}
}

func TestOpDefFuncParamErrors(t *testing.T) {
	// 1. Param name constant out of bounds
	{
		constants := []Value{StringValue{Val: "myFunc"}}
		bytecode := createBytecodeHeader(constants)
		bytecode = append(bytecode, byte(OpDefFunc))
		nameIdx := make([]byte, 4); binary.LittleEndian.PutUint32(nameIdx, 0); bytecode = append(bytecode, nameIdx...)
		paramCount := make([]byte, 4); binary.LittleEndian.PutUint32(paramCount, 1); bytecode = append(bytecode, paramCount...)
		bodyLength := make([]byte, 4); binary.LittleEndian.PutUint32(bodyLength, 0); bytecode = append(bytecode, bodyLength...)
		param1Idx := make([]byte, 4); binary.LittleEndian.PutUint32(param1Idx, 99); bytecode = append(bytecode, param1Idx...)
		vm := NewVM()
		_, err := vm.Execute(bytecode)
		if err == nil || !strings.Contains(err.Error(), "param name constant out of bounds") {
			t.Errorf("Expected 'param name constant out of bounds' error, got %v", err)
		}
	}
	// 2. Param name is not a string
	{
		constants := []Value{StringValue{Val: "myFunc"}, IntValue{Val: 123}}
		bytecode := createBytecodeHeader(constants)
		bytecode = append(bytecode, byte(OpDefFunc))
		nameIdx := make([]byte, 4); binary.LittleEndian.PutUint32(nameIdx, 0); bytecode = append(bytecode, nameIdx...)
		paramCount := make([]byte, 4); binary.LittleEndian.PutUint32(paramCount, 1); bytecode = append(bytecode, paramCount...)
		bodyLength := make([]byte, 4); binary.LittleEndian.PutUint32(bodyLength, 0); bytecode = append(bytecode, bodyLength...)
		param1Idx := make([]byte, 4); binary.LittleEndian.PutUint32(param1Idx, 1); bytecode = append(bytecode, param1Idx...)
		vm := NewVM()
		_, err := vm.Execute(bytecode)
		if err == nil || !strings.Contains(err.Error(), "param name must be a string") {
			t.Errorf("Expected 'param name must be a string' error, got %v", err)
		}
	}
}

func TestOpMutatorHalt(t *testing.T) {
	// Test: Use MUTATOR to write an OP_HALT over an OP_ADD
	constants := []Value{IntValue{Val: 255}, IntValue{Val: 0}}
	bytecode := createBytecodeHeader(constants)
	c255, c0 := uint32(0), uint32(1)
	
	bytecode = addInstruction(bytecode, OpPush, &c255)
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = append(bytecode, byte(OpMutator))
	// This OP_ADD will be turned into OP_HALT (255) by the previous instruction
	bytecode = append(bytecode, byte(OpAdd))
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestOpMitosis(t *testing.T) {
	// Test: Use MITOSIS to branch parent and child
	constants := []Value{IntValue{Val: 0}}
	bytecode := createBytecodeHeader(constants)
	c0 := uint32(0)
	
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = append(bytecode, byte(OpMitosis))
	
	// Parent path continues, Child branches
	jumpPos := len(bytecode)
	bytecode = append(bytecode, byte(OpJumpIfFalse), 0, 0, 0, 0)
	
	// Parent: Halt
	bytecode = append(bytecode, byte(OpHalt))
	
	// Child path
	childStart := uint32(len(bytecode))
	binary.LittleEndian.PutUint32(bytecode[jumpPos+1:], childStart)
	bytecode = append(bytecode, byte(OpHalt))
	
	vm := NewVM()
	_, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}
