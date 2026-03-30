package vm

import (
	"encoding/binary"
	"testing"
)

func TestIssue7BuildObject(t *testing.T) {
	constants := []Value{
		StringValue{Val: "x"}, IntValue{Val: 10},
		StringValue{Val: "y"}, IntValue{Val: 20},
	}
	bytecode := createBytecodeHeader(constants)

	c0, c1, c2, c3 := uint32(0), uint32(1), uint32(2), uint32(3)
	bytecode = addInstruction(bytecode, OpPush, &c0) // "x"
	bytecode = addInstruction(bytecode, OpPush, &c1) // 10
	bytecode = addInstruction(bytecode, OpPush, &c2) // "y"
	bytecode = addInstruction(bytecode, OpPush, &c3) // 20

	count := uint32(2)
	bytecode = addInstruction(bytecode, OpBuildObject, &count)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	objVal, ok := result.(ObjectValue)
	if !ok {
		t.Fatalf("Expected ObjectValue, got %T", result)
	}

	if len(objVal.Val) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(objVal.Val))
	}
	
	if x, ok := objVal.Val["x"].(IntValue); !ok || x.Val != 10 {
		t.Errorf("Expected x: 10, got %v", objVal.Val["x"])
	}
}

func TestIssue7GetSetField(t *testing.T) {
	constants := []Value{
		StringValue{Val: "x"}, IntValue{Val: 10},
		IntValue{Val: 42},
	}
	bytecode := createBytecodeHeader(constants)

	c0, c1, c2 := uint32(0), uint32(1), uint32(2)
	
	// obj = { "x": 10 }
	bytecode = addInstruction(bytecode, OpPush, &c0) // "x"
	bytecode = addInstruction(bytecode, OpPush, &c1) // 10
	count := uint32(1)
	bytecode = addInstruction(bytecode, OpBuildObject, &count)
	
	// obj["x"] = 42
	bytecode = addInstruction(bytecode, OpPush, &c0) // key "x"
	bytecode = addInstruction(bytecode, OpPush, &c2) // val 42
	bytecode = addInstruction(bytecode, OpSetField, nil)
	
	// get obj["x"]
	bytecode = addInstruction(bytecode, OpPush, &c0) // key "x"
	bytecode = addInstruction(bytecode, OpGetField, nil)
	
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if val, ok := result.(IntValue); !ok || val.Val != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

// addDefFuncInstruction appends an OpDefFunc instruction to the bytecode.
// Format: OpDefFunc, name_idx(4), param_count(4), body_length(4), [param_name_idx(4)]..., body...
func addDefFuncInstruction(bytecode []byte, nameIdx, paramCount, bodyLength uint32, paramNameIdxs []uint32, body []byte) []byte {
	bytecode = append(bytecode, byte(OpDefFunc))
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, nameIdx)
	bytecode = append(bytecode, buf...)
	binary.LittleEndian.PutUint32(buf, paramCount)
	bytecode = append(bytecode, buf...)
	binary.LittleEndian.PutUint32(buf, bodyLength)
	bytecode = append(bytecode, buf...)
	for _, idx := range paramNameIdxs {
		binary.LittleEndian.PutUint32(buf, idx)
		bytecode = append(bytecode, buf...)
	}
	bytecode = append(bytecode, body...)
	return bytecode
}

func TestIssue7DefFuncNoArgs(t *testing.T) {
	// Define a function "fortyTwo" with no params that returns 42.
	// Constants: 0="fortyTwo", 1=42
	constants := []Value{
		StringValue{Val: "fortyTwo"}, // 0
		IntValue{Val: 42},            // 1
	}
	bytecode := createBytecodeHeader(constants)

	// Build function body: push 42, return
	var body []byte
	c1 := uint32(1)
	body = addInstruction(body, OpPush, &c1)
	body = append(body, byte(OpReturn))

	// Emit OpDefFunc: name=0("fortyTwo"), params=0, bodyLen, body
	bodyLen := uint32(len(body))
	bytecode = addDefFuncInstruction(bytecode, 0, 0, bodyLen, nil, body)

	// Call the function: push "fortyTwo", call with 0 args
	c0 := uint32(0)
	callArgs := uint32(0)
	bytecode = addInstruction(bytecode, OpPush, &c0)
	bytecode = addInstruction(bytecode, OpCall, &callArgs)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if val, ok := result.(IntValue); !ok || val.Val != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestIssue7DefFuncWithParams(t *testing.T) {
	// Define "add(a, b)" that returns a + b, then call add(3, 4) => 7.
	// Constants: 0="add", 1="a", 2="b", 3=IntValue(3), 4=IntValue(4)
	constants := []Value{
		StringValue{Val: "add"}, // 0
		StringValue{Val: "a"},   // 1
		StringValue{Val: "b"},   // 2
		IntValue{Val: 3},        // 3
		IntValue{Val: 4},        // 4
	}
	bytecode := createBytecodeHeader(constants)

	// Build function body: load "a", load "b", add, return
	var body []byte
	c1 := uint32(1)
	c2 := uint32(2)
	body = addInstruction(body, OpLoadVar, &c1) // load a
	body = addInstruction(body, OpLoadVar, &c2) // load b
	body = append(body, byte(OpAdd))
	body = append(body, byte(OpReturn))

	// Emit OpDefFunc: name=0("add"), params=2, bodyLen, param_names=[1,2], body
	bodyLen := uint32(len(body))
	bytecode = addDefFuncInstruction(bytecode, 0, 2, bodyLen, []uint32{1, 2}, body)

	// Call: push "add", push 3, push 4, call(2)
	c0, c3, c4 := uint32(0), uint32(3), uint32(4)
	callArgs := uint32(2)
	bytecode = addInstruction(bytecode, OpPush, &c0) // function name
	bytecode = addInstruction(bytecode, OpPush, &c3) // arg 3
	bytecode = addInstruction(bytecode, OpPush, &c4) // arg 4
	bytecode = addInstruction(bytecode, OpCall, &callArgs)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	vm := NewVM()
	result, err := vm.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if val, ok := result.(IntValue); !ok || val.Val != 7 {
		t.Errorf("Expected 7, got %v", result)
	}
}
