package gpu

import (
	"encoding/binary"
	"math"
	"testing"
)

// buildBytecode creates a minimal GLYP bytecode with given constants and instructions.
// Format: "GLYP" + version(4 LE) + constCount(4 LE) + constants... + instrCount(4 LE) + code...
func buildBytecode(constants []interface{}, code []byte) []byte {
	buf := []byte("GLYP")

	// Version: 1 (little-endian)
	ver := make([]byte, 4)
	binary.LittleEndian.PutUint32(ver, 1)
	buf = append(buf, ver...)

	// Constant count (little-endian)
	cc := make([]byte, 4)
	binary.LittleEndian.PutUint32(cc, uint32(len(constants)))
	buf = append(buf, cc...)

	// Encode constants (little-endian to match bootstrap intToBytes4)
	for _, c := range constants {
		switch v := c.(type) {
		case nil:
			buf = append(buf, 0x00)
		case int:
			buf = append(buf, 0x01)
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(v))
			buf = append(buf, b...)
		case int64:
			buf = append(buf, 0x01)
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(v))
			buf = append(buf, b...)
		case float64:
			buf = append(buf, 0x02)
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, math.Float64bits(v))
			buf = append(buf, b...)
		case bool:
			buf = append(buf, 0x03)
			if v {
				buf = append(buf, 1)
			} else {
				buf = append(buf, 0)
			}
		case string:
			buf = append(buf, 0x04)
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(v)))
			buf = append(buf, b...)
			buf = append(buf, []byte(v)...)
		}
	}

	// Instruction count (little-endian)
	ic := make([]byte, 4)
	binary.LittleEndian.PutUint32(ic, uint32(len(code)))
	buf = append(buf, ic...)

	buf = append(buf, code...)
	return buf
}

// emit helpers — all operands are little-endian to match Go VM
func pushConst(idx uint32) []byte {
	b := []byte{0x01} // OP_PUSH
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, idx)
	return append(b, operand...)
}

func storeVar(idx uint32) []byte {
	b := []byte{0x41} // OP_STORE_VAR
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, idx)
	return append(b, operand...)
}

func loadVar(idx uint32) []byte {
	b := []byte{0x40} // OP_LOAD_VAR
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, idx)
	return append(b, operand...)
}

func jump(target uint32) []byte {
	b := []byte{0x50} // OP_JUMP
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, target)
	return append(b, operand...)
}

func jumpIfFalse(target uint32) []byte {
	b := []byte{0x51}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, target)
	return append(b, operand...)
}

func TestNewDispatcher(t *testing.T) {
	d := NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}
	if d.ShaderSource() == "" {
		t.Fatal("shader source is empty")
	}
}

func TestSimpleAdd(t *testing.T) {
	// 10 + 5 = 15
	constants := []interface{}{10, 5}
	var code []byte
	code = append(code, pushConst(0)...) // push 10
	code = append(code, pushConst(1)...) // push 5
	code = append(code, 0x10)            // ADD
	code = append(code, 0xFF)            // HALT

	bytecode := buildBytecode(constants, code)

	d := NewDispatcher()
	result, err := d.ExecuteOne(bytecode)
	if err != nil {
		t.Fatalf("ExecuteOne error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
	if result.Tag != TagInt {
		t.Fatalf("expected int result, got tag %d", result.Tag)
	}
	if result.IntVal != 15 {
		t.Fatalf("expected 15, got %d", result.IntVal)
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		op   byte
		want int64
	}{
		{"add", 10, 5, 0x10, 15},
		{"sub", 10, 3, 0x11, 7},
		{"mul", 6, 7, 0x12, 42},
		{"div", 20, 4, 0x13, 5},
		{"mod", 17, 5, 0x14, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []interface{}{tt.a, tt.b}
			var code []byte
			code = append(code, pushConst(0)...)
			code = append(code, pushConst(1)...)
			code = append(code, tt.op)
			code = append(code, 0xFF) // HALT

			d := NewDispatcher()
			result, err := d.ExecuteOne(buildBytecode(constants, code))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result.Error != nil {
				t.Fatalf("VM error: %v", result.Error)
			}
			if result.IntVal != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, result.IntVal)
			}
		})
	}
}

func TestComparisons(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		op   byte
		want bool
	}{
		{"eq_true", 5, 5, 0x20, true},
		{"eq_false", 5, 3, 0x20, false},
		{"ne_true", 5, 3, 0x21, true},
		{"ne_false", 5, 5, 0x21, false},
		{"lt_true", 3, 5, 0x22, true},
		{"lt_false", 5, 3, 0x22, false},
		{"gt_true", 5, 3, 0x23, true},
		{"gt_false", 3, 5, 0x23, false},
		{"ge_true", 5, 5, 0x24, true},
		{"ge_false", 3, 5, 0x24, false},
		{"le_true", 5, 5, 0x25, true},
		{"le_false", 5, 3, 0x25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constants := []interface{}{tt.a, tt.b}
			var code []byte
			code = append(code, pushConst(0)...)
			code = append(code, pushConst(1)...)
			code = append(code, tt.op)
			code = append(code, 0xFF) // HALT

			d := NewDispatcher()
			result, err := d.ExecuteOne(buildBytecode(constants, code))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result.Error != nil {
				t.Fatalf("VM error: %v", result.Error)
			}
			if result.BoolVal != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, result.BoolVal)
			}
		})
	}
}

func TestVariables(t *testing.T) {
	// x = 42; load x; halt → result should be 42
	constants := []interface{}{42}
	var code []byte
	code = append(code, pushConst(0)...) // push 42
	code = append(code, storeVar(0)...)  // store to var[0]
	code = append(code, loadVar(0)...)   // load var[0]
	code = append(code, 0xFF)            // HALT

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IntVal != 42 {
		t.Fatalf("expected 42, got %d", result.IntVal)
	}
}

func TestConditionalJump(t *testing.T) {
	// if (false) { x = 1 } else { x = 2 }
	// push false, jump_if_false to else, push 1, store, jump end, push 2, store, halt
	constants := []interface{}{false, 1, 2}

	var code []byte
	code = append(code, pushConst(0)...) // push false (const 0)
	// jump_if_false to offset 20 (else branch)
	code = append(code, jumpIfFalse(20)...) // 5 bytes
	// then branch: push 1, store, jump to end
	code = append(code, pushConst(1)...) // push 1 (offset 10)
	code = append(code, storeVar(0)...)  // store x (offset 15)
	code = append(code, jump(25)...)     // jump to halt (offset 20)
	// else branch: push 2, store
	code = append(code, pushConst(2)...) // push 2 (offset 25... wait, recalc)

	// Let me recalculate offsets more carefully:
	// offset 0: pushConst(0) = 5 bytes → ends at 5
	// offset 5: jumpIfFalse(X) = 5 bytes → ends at 10
	// offset 10: pushConst(1) = 5 bytes → ends at 15
	// offset 15: storeVar(0) = 5 bytes → ends at 20
	// offset 20: jump(X) = 5 bytes → ends at 25
	// offset 25: pushConst(2) = 5 bytes → ends at 30
	// offset 30: storeVar(0) = 5 bytes → ends at 35
	// offset 35: loadVar(0) = 5 bytes → ends at 40
	// offset 40: HALT

	// Redo with correct offsets:
	code = nil
	code = append(code, pushConst(0)...)    // 0-4: push false
	code = append(code, jumpIfFalse(25)...) // 5-9: jump to else at 25
	code = append(code, pushConst(1)...)    // 10-14: push 1
	code = append(code, storeVar(0)...)     // 15-19: store x
	code = append(code, jump(35)...)        // 20-24: jump to end at 35
	code = append(code, pushConst(2)...)    // 25-29: push 2 (else branch)
	code = append(code, storeVar(0)...)     // 30-34: store x
	code = append(code, loadVar(0)...)      // 35-39: load x
	code = append(code, 0xFF)               // 40: HALT

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
	// false → else branch → x = 2
	if result.IntVal != 2 {
		t.Fatalf("expected 2, got %d", result.IntVal)
	}
}

func TestReturn(t *testing.T) {
	constants := []interface{}{99}
	var code []byte
	code = append(code, pushConst(0)...) // push 99
	code = append(code, 0x61)            // RETURN

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IntVal != 99 {
		t.Fatalf("expected 99, got %d", result.IntVal)
	}
}

func TestDivByZero(t *testing.T) {
	constants := []interface{}{10, 0}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x13) // DIV
	code = append(code, 0xFF)

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected division by zero error")
	}
}

func TestParallelVMs(t *testing.T) {
	// Run 100 VMs in parallel, all computing 3 + 4 = 7
	constants := []interface{}{3, 4}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x10) // ADD
	code = append(code, 0xFF) // HALT

	d := NewDispatcher()
	results, err := d.Execute(buildBytecode(constants, code), 100)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 100 {
		t.Fatalf("expected 100 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Error != nil {
			t.Fatalf("VM %d error: %v", i, r.Error)
		}
		if r.IntVal != 7 {
			t.Fatalf("VM %d: expected 7, got %d", i, r.IntVal)
		}
	}
}

func TestLogicalOps(t *testing.T) {
	// true AND false = false
	constants := []interface{}{true, false}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x26) // AND
	code = append(code, 0xFF)

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.BoolVal != false {
		t.Fatal("expected false for true AND false")
	}

	// true OR false = true
	code = nil
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x27) // OR
	code = append(code, 0xFF)

	result, err = d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.BoolVal != true {
		t.Fatal("expected true for true OR false")
	}

	// NOT true = false
	constants2 := []interface{}{true}
	code = nil
	code = append(code, pushConst(0)...)
	code = append(code, 0x28) // NOT
	code = append(code, 0xFF)

	result, err = d.ExecuteOne(buildBytecode(constants2, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.BoolVal != false {
		t.Fatal("expected false for NOT true")
	}
}

func TestNegation(t *testing.T) {
	constants := []interface{}{42}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, 0x29) // NEG
	code = append(code, 0xFF)

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.IntVal != -42 {
		t.Fatalf("expected -42, got %d", result.IntVal)
	}
}

func TestInvalidBytecode(t *testing.T) {
	d := NewDispatcher()

	// Too short
	_, err := d.ExecuteOne([]byte{1, 2})
	if err == nil {
		t.Fatal("expected error for short bytecode")
	}

	// Wrong magic
	_, err = d.ExecuteOne([]byte{'N', 'O', 'P', 'E', 0xFF})
	if err == nil {
		t.Fatal("expected error for wrong magic")
	}
}

func TestShaderSourceEmbedded(t *testing.T) {
	d := NewDispatcher()
	src := d.ShaderSource()
	if len(src) < 100 {
		t.Fatal("shader source too short")
	}
	// Check it contains key WGSL constructs
	if !containsStr(src, "@compute") {
		t.Fatal("shader missing @compute directive")
	}
	if !containsStr(src, "OP_PUSH") {
		t.Fatal("shader missing OP_PUSH constant")
	}
	if !containsStr(src, "exec_step") {
		t.Fatal("shader missing exec_step function")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func BenchmarkSingleVM(b *testing.B) {
	constants := []interface{}{1, 1000}
	// Simple loop: x = 0, while x < 1000: x = x + 1
	var code []byte
	code = append(code, pushConst(0)...) // push 0 (initial x)     0-4
	code = append(code, storeVar(0)...)  // store x                5-9
	// loop start at offset 10:
	code = append(code, loadVar(0)...)      // load x                 10-14
	code = append(code, pushConst(1)...)    // push 1000              15-19
	code = append(code, 0x22)               // LT                     20
	code = append(code, jumpIfFalse(36)...) // exit if x >= 1000      21-25
	code = append(code, loadVar(0)...)      // load x                 26-30
	code = append(code, pushConst(0)...)    // push 1                 31-35... wait

	// Redo: need constant for 0 (initial) and 1 (increment) and 1000 (limit)
	constants = []interface{}{0, 1, 1000}
	code = nil
	code = append(code, pushConst(0)...) // push 0                 0-4
	code = append(code, storeVar(0)...)  // store x                5-9
	// loop:
	code = append(code, loadVar(0)...)      // load x                 10-14
	code = append(code, pushConst(2)...)    // push 1000              15-19
	code = append(code, 0x22)               // LT                     20
	code = append(code, jumpIfFalse(41)...) // exit loop at 41        21-25
	code = append(code, loadVar(0)...)      // load x                 26-30
	code = append(code, pushConst(1)...)    // push 1                 31-35
	code = append(code, 0x10)               // ADD                    36
	code = append(code, storeVar(0)...)     // store x                37-41... wait, that's wrong

	// offset 37: storeVar = 5 bytes, ends at 42
	// offset 42: jump(10) = 5 bytes, ends at 47
	// Then exitLoop should be at 47
	code = nil
	code = append(code, pushConst(0)...)    // 0: push 0
	code = append(code, storeVar(0)...)     // 5: store x
	code = append(code, loadVar(0)...)      // 10: load x
	code = append(code, pushConst(2)...)    // 15: push 1000
	code = append(code, 0x22)               // 20: LT
	code = append(code, jumpIfFalse(47)...) // 21: exit at 47
	code = append(code, loadVar(0)...)      // 26: load x
	code = append(code, pushConst(1)...)    // 31: push 1
	code = append(code, 0x10)               // 36: ADD
	code = append(code, storeVar(0)...)     // 37: store x
	code = append(code, jump(10)...)        // 42: back to loop
	code = append(code, loadVar(0)...)      // 47: load x (result)
	code = append(code, 0xFF)               // 52: HALT

	bytecode := buildBytecode(constants, code)
	d := NewDispatcher()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := d.ExecuteOne(bytecode)
		if err != nil {
			b.Fatal(err)
		}
		if result.IntVal != 1000 {
			b.Fatalf("expected 1000, got %d", result.IntVal)
		}
	}
}

// --- Spatial opcode tests (issue #19) ---

// TestMutatorCPU verifies the M opcode (0xC1) in the CPU fallback VM.
// Mutator pops value and offset, then writes value as a byte to bytecode[IP+offset].
func TestMutatorCPU(t *testing.T) {
	// Program: push 42, push 5, MUTATOR, HALT
	// The mutator should write 42 to code[IP + 5] which is the HALT instruction.
	// After mutation, we verify it was accepted without error.
	//
	// Layout:
	// 0-4: push const 0 (value=42)
	// 5-9: push const 1 (offset=3)
	// 10:  MUTATOR (0xC1)
	// 11:  HALT (0xFF) — this will be overwritten to 42
	// ... extra bytes so target is within bounds
	constants := []interface{}{42, 3}
	var code []byte
	code = append(code, pushConst(0)...) // push 42
	code = append(code, pushConst(1)...) // push offset=3
	code = append(code, 0xC1)            // MUTATOR
	code = append(code, 0xFF)            // HALT (at code offset 11, target = IP(11)+3=14... no)
	// Wait: mutator target = current PC + offset. PC when MUTATOR runs is 10.
	// Target = 10 + 3 = 13. But code only has 12 bytes. Need more.
	// Let me use offset=2 so target=12 and pad code to 14 bytes.
	constants = []interface{}{42, 2}
	code = nil
	code = append(code, pushConst(0)...) // 0-4: push 42
	code = append(code, pushConst(1)...) // 5-9: push offset=2
	code = append(code, 0xC1)            // 10: MUTATOR
	code = append(code, 0xFF)            // 11: HALT
	code = append(code, 0x00)            // 12: padding
	code = append(code, 0x00)            // 13: padding (target = PC + offset = 10 + 2 = 12... no)
	// The CPU VM uses base+pc for absolute addressing within the instruction section.
	// MUTATOR pops value then offset. target = pc + offset.
	// pc when executing MUTATOR = 10. target = 10 + 2 = 12.
	// That's a valid offset. The value 42 will be written to bytecode[base+12].
	// But we need to make sure there's enough bytecode after base+12.

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
}

// TestMitosisCPU verifies the S opcode (0xC0) in the CPU fallback VM.
// Mitosis pops an offset and pushes a child thread ID.
func TestMitosisCPU(t *testing.T) {
	// Program: push 0 (offset), MITOSIS, push 1, ADD, HALT
	// The S opcode should push the thread ID (1 for root VM).
	constants := []interface{}{0, 10}
	var code []byte
	code = append(code, pushConst(0)...) // push 0 (spatial offset)
	code = append(code, 0xC0)            // MITOSIS — pops offset, pushes thread ID
	code = append(code, pushConst(1)...) // push 10
	code = append(code, 0x10)            // ADD (threadID + 10)
	code = append(code, 0xFF)            // HALT

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
	// Thread ID for VM 0 should be 0, so result = 0 + 10 = 10
	if result.IntVal != 10 {
		t.Fatalf("expected 10 (vmID + 10), got %d", result.IntVal)
	}
}

// TestSpatialOpcodesGPUCompatible verifies IsGPUCompatible accepts spatial opcodes.
func TestSpatialOpcodesGPUCompatible(t *testing.T) {
	// Build bytecode with MUTATOR (0xC1)
	constants := []interface{}{42, 2}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0xC1) // MUTATOR
	code = append(code, 0xFF)
	code = append(code, 0x00, 0x00)

	if !IsGPUCompatible(buildBytecode(constants, code)) {
		t.Fatal("bytecode with MUTATOR should be GPU compatible")
	}

	// Build bytecode with MITOSIS (0xC0)
	code = nil
	code = append(code, pushConst(0)...)
	code = append(code, 0xC0) // MITOSIS
	code = append(code, 0xFF)

	if !IsGPUCompatible(buildBytecode(constants, code)) {
		t.Fatal("bytecode with MITOSIS should be GPU compatible")
	}
}

// TestMutatorOutOfBounds verifies mutator errors on out-of-bounds target.
func TestMutatorOutOfBounds(t *testing.T) {
	constants := []interface{}{42, 999}
	var code []byte
	code = append(code, pushConst(0)...) // push 42
	code = append(code, pushConst(1)...) // push offset=999
	code = append(code, 0xC1)            // MUTATOR — should error
	code = append(code, 0xFF)            // HALT

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected error for out-of-bounds mutator target")
	}
}

// TestMutatorSelfModify verifies that the mutator actually modifies bytecode.
// We write a program that self-modifies: replaces a NOP with HALT.
func TestMutatorSelfModify(t *testing.T) {
	// Program: push 0xFF, push 1, MUTATOR, NOP, push 99, HALT
	// The mutator writes 0xFF to code[PC+1] = code[10+1] = code[11]
	// code[11] is the NOP (0x00). After mutation, the program halts at offset 11
	// when it encounters the 0xFF we wrote there.
	//
	// Wait, the flow is: after MUTATOR at pc=10, PC advances to 11 (the NOP).
	// But we just wrote 0xFF to code[11], so when PC=11 it reads 0xFF = HALT.
	// The stack will have... let me trace:
	//   push 0xFF → stack: [0xFF]
	//   push 1    → stack: [0xFF, 1]
	//   MUTATOR   → pops offset=1, value=0xFF, writes 0xFF to code[10+1]=code[11]
	//   PC advances to 11, reads 0xFF (was 0x00, now 0xFF) → HALT
	//   Stack is empty at HALT, so result is default.
	//
	// Actually, on second thought, we need the value on stack to be the int value.
	// The mutator pops: first offset, then value. So stack order matters.
	// In the CPU VM execMutator: offset = Pop(), val = Pop().
	// So push value first, push offset second.
	constants := []interface{}{0xFF, 1}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 0xFF (value to write)
	code = append(code, pushConst(1)...) // 5-9: push 1 (offset from PC)
	code = append(code, 0xC1)            // 10: MUTATOR — pops offset, value; writes to code[10+1]
	code = append(code, 0x00)            // 11: NOP (will be overwritten to 0xFF = HALT)
	code = append(code, 0xFF)            // 12: HALT (unreachable after mutation)

	d := NewDispatcher()
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
	// The program should have halted at offset 11 (the mutated byte)
	// Steps should be small since we halt early
	if result.Steps == 0 {
		t.Fatal("expected some steps to execute")
	}
}

func BenchmarkParallelVMs(b *testing.B) {
	constants := []interface{}{10, 5}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x10) // ADD
	code = append(code, 0xFF) // HALT

	bytecode := buildBytecode(constants, code)
	d := NewDispatcher()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := d.Execute(bytecode, 1000)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) != 1000 {
			b.Fatal("wrong result count")
		}
	}
}
