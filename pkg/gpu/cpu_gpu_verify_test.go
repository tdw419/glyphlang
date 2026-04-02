package gpu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/glyphlang/glyph/pkg/vm"
)

// --------------------------------------------------------------------------
// cpuGPUTestCase describes a single cross-verification test.
//
// Each test provides bytecode that exercises a particular opcode or
// combination.  The harness runs the same bytecode on:
//   - The Go CPU VM (pkg/vm)
//   - The GPU CPU-fallback VM (pkg/gpu dispatcher, hasGPU=false)
//
// and verifies that both agree on the final result value.
// --------------------------------------------------------------------------

type cpuGPUTestCase struct {
	name      string
	constants []interface{}
	code      []byte
	// Optional expected values — if set, checked against BOTH VMs.
	wantTag   uint32
	wantInt   int64
	wantFloat float64
	wantBool  bool
	wantError bool // if true, both paths should produce an error
}

// runCPUVM executes bytecode on the Go CPU VM and returns the top-of-stack value.
func runCPUVM(bytecode []byte) (vm.Value, error) {
	gvm := vm.NewVM()
	return gvm.Execute(bytecode)
}

// runGPUVM executes bytecode on the GPU dispatcher (CPU fallback).
func runGPUVM(bytecode []byte) (*Result, error) {
	d := NewDispatcher()
	d.hasGPU = false // force CPU fallback
	return d.ExecuteOne(bytecode)
}

// valuesMatch checks whether the Go CPU VM Value matches the GPU Result.
func valuesMatch(cpuVal vm.Value, gpuRes *Result) (bool, string) {
	if cpuVal == nil && gpuRes == nil {
		return true, ""
	}
	if cpuVal == nil {
		return false, "CPU nil, GPU non-nil"
	}
	if gpuRes == nil {
		return false, "CPU non-nil, GPU nil"
	}

	// Check for GPU error
	if gpuRes.Error != nil {
		return false, fmt.Sprintf("GPU error: %v", gpuRes.Error)
	}

	switch v := cpuVal.(type) {
	case vm.IntValue:
		if gpuRes.Tag != TagInt {
			return false, fmt.Sprintf("tag mismatch: CPU=Int(%d), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		if gpuRes.IntVal != v.Val {
			return false, fmt.Sprintf("int mismatch: CPU=%d, GPU=%d", v.Val, gpuRes.IntVal)
		}
		return true, ""

	case vm.BoolValue:
		if gpuRes.Tag != TagBool {
			return false, fmt.Sprintf("tag mismatch: CPU=Bool(%v), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		if gpuRes.BoolVal != v.Val {
			return false, fmt.Sprintf("bool mismatch: CPU=%v, GPU=%v", v.Val, gpuRes.BoolVal)
		}
		return true, ""

	case vm.FloatValue:
		if gpuRes.Tag != TagFloat {
			return false, fmt.Sprintf("tag mismatch: CPU=Float(%v), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		// Float precision: GPU uses f32 internally, CPU uses f64. Compare at f32 precision.
		cpuF := float32(v.Val)
		gpuF := float32(gpuRes.FloatVal)
		if math.Float32bits(cpuF) != math.Float32bits(gpuF) {
			return false, fmt.Sprintf("float mismatch: CPU=%v (f32=%v), GPU=%v", v.Val, cpuF, gpuRes.FloatVal)
		}
		return true, ""

	case vm.NullValue:
		if gpuRes.Tag != TagNull {
			return false, fmt.Sprintf("tag mismatch: CPU=Null, GPU=Tag(%d)", gpuRes.Tag)
		}
		return true, ""

	default:
		return false, fmt.Sprintf("unsupported CPU value type for GPU comparison: %T", cpuVal)
	}
}

// runCrossVerify runs a single test case on both VMs and checks agreement.
func bytesWriter(b *[]byte) *bytes.Buffer {
	return bytes.NewBuffer(*b)
}

// emitU32 appends a little-endian uint32 to the code slice.
func emitU32(code []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(code, b...)
}

// headerSize computes the bytecode header size for a given set of constants.
func headerSize(constants []interface{}) uint32 {
	h := uint32(4 + 4 + 4) // GLYP + version + constCount
	for _, c := range constants {
		switch c.(type) {
		case nil:
			h += 1
		case int, int64, float64:
			h += 1 + 8
		case bool:
			h += 1 + 1
		case string:
			h += 1 + 4 + uint32(len(c.(string)))
		}
	}
	h += 4 + 4 // stringPoolCount + instrCount
	return h
}

func runCrossVerify(t *testing.T, tc cpuGPUTestCase) {
	t.Helper()
	bytecode := buildBytecode(tc.constants, tc.code)

	// Run on CPU VM
	cpuResult, cpuErr := runCPUVM(bytecode)

	// Run on GPU dispatcher
	gpuResult, gpuErr := runGPUVM(bytecode)

	if tc.wantError {
		if cpuErr == nil {
			t.Fatalf("CPU VM: expected error but got nil (result=%v)", cpuResult)
		}
		if gpuErr != nil {
			return
		}
		if gpuResult != nil && gpuResult.Error != nil {
			return
		}
		t.Fatalf("GPU VM: expected error but got none (result=%v)", gpuResult)
		return
	}

	if cpuErr != nil {
		t.Fatalf("CPU VM unexpected error: %v", cpuErr)
	}
	if gpuErr != nil {
		t.Fatalf("GPU VM unexpected error: %v", gpuErr)
	}

	// Verify against expected values if provided
	switch tc.wantTag {
	case TagInt:
		if v, ok := cpuResult.(vm.IntValue); !ok || v.Val != tc.wantInt {
			t.Fatalf("CPU VM value mismatch: want Int(%d), got %v", tc.wantInt, cpuResult)
		}
	case TagBool:
		if v, ok := cpuResult.(vm.BoolValue); !ok || v.Val != tc.wantBool {
			t.Fatalf("CPU VM value mismatch: want Bool(%v), got %v", tc.wantBool, cpuResult)
		}
	case TagFloat:
		if _, ok := cpuResult.(vm.FloatValue); !ok {
			t.Fatalf("CPU VM value mismatch: want Float(%v), got %v", tc.wantFloat, cpuResult)
		}
	case TagNull:
		if _, ok := cpuResult.(vm.NullValue); !ok {
			t.Fatalf("CPU VM value mismatch: want Null, got %v", cpuResult)
		}
	}

	// Cross-verify CPU vs GPU
	ok, reason := valuesMatch(cpuResult, gpuResult)
	if !ok {
		t.Fatalf("CPU/GPU mismatch: %s\n  CPU result: %v\n  GPU result: Tag=%d IntVal=%d FloatVal=%v BoolVal=%v",
			reason, cpuResult, gpuResult.Tag, gpuResult.IntVal, gpuResult.FloatVal, gpuResult.BoolVal)
	}
}

// ==========================================================================
// Actual test cases
// ==========================================================================

func TestCrossVerify_Arithmetic(t *testing.T) {
	tests := []cpuGPUTestCase{
		{
			name:      "add 10+5",
			constants: []interface{}{10, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH const 0 (10)
					0x01, 0x01, 0x00, 0x00, 0x00, // PUSH const 1 (5)
					0x10, // ADD
					0xFF, // HALT
				}
			}(),
			wantTag: TagInt,
			wantInt: 15,
		},
		{
			name:      "sub 50-8",
			constants: []interface{}{50, 8},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH const 0 (50)
					0x01, 0x01, 0x00, 0x00, 0x00, // PUSH const 1 (8)
					0x11, // SUB
					0xFF, // HALT
				}
			}(),
			wantTag: TagInt,
			wantInt: 42,
		},
		{
			name:      "mul 6*7",
			constants: []interface{}{6, 7},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x12, // MUL
					0xFF,
				}
			}(),
			wantTag: TagInt,
			wantInt: 42,
		},
		{
			name:      "div 84/2",
			constants: []interface{}{84, 2},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x13, // DIV
					0xFF,
				}
			}(),
			wantTag: TagInt,
			wantInt: 42,
		},
		{
			name:      "mod 17%5",
			constants: []interface{}{17, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x14, // MOD
					0xFF,
				}
			}(),
			wantTag: TagInt,
			wantInt: 2,
		},
		{
			name:      "negation -42",
			constants: []interface{}{42},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 42
					0x29, // NEG
					0xFF, // HALT
				}
			}(),
			wantTag: TagInt,
			wantInt: -42,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCrossVerify(t, tc)
		})
	}
}

func TestCrossVerify_Comparisons(t *testing.T) {
	tests := []cpuGPUTestCase{
		{
			name:      "eq 5==5 true",
			constants: []interface{}{5, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x20, // EQ
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "eq 5==3 false",
			constants: []interface{}{5, 3},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x20, // EQ
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: false,
		},
		{
			name:      "ne 5!=3 true",
			constants: []interface{}{5, 3},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x21, // NE
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "lt 3<5 true",
			constants: []interface{}{3, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x22, // LT
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "gt 5>3 true",
			constants: []interface{}{5, 3},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x23, // GT
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "ge 5>=5 true",
			constants: []interface{}{5, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x24, // GE
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "le 3<=5 true",
			constants: []interface{}{3, 5},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x25, // LE
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCrossVerify(t, tc)
		})
	}
}

func TestCrossVerify_LogicalOps(t *testing.T) {
	tests := []cpuGPUTestCase{
		{
			name:      "AND true,true",
			constants: []interface{}{true, true},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x26, // AND
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "AND true,false",
			constants: []interface{}{true, false},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x26, // AND
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: false,
		},
		{
			name:      "OR true,false",
			constants: []interface{}{true, false},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x27, // OR
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "NOT true",
			constants: []interface{}{true},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x28, // NOT
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: false,
		},
		{
			name:      "NOT false",
			constants: []interface{}{false},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x28, // NOT
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCrossVerify(t, tc)
		})
	}
}

func TestCrossVerify_Variables(t *testing.T) {
	// x = 42; load x; halt
	// Constants: 42 (val), "x" (name)
	tc := cpuGPUTestCase{
		name:      "store and load variable",
		constants: []interface{}{42, "x"},
		code: func() []byte {
			return []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 42
				0x41, 0x01, 0x00, 0x00, 0x00, // STORE_VAR[1] ("x")
				0x40, 0x01, 0x00, 0x00, 0x00, // LOAD_VAR[1] ("x")
				0xFF, // HALT
			}
		}(),
		wantTag: TagInt,
		wantInt: 42,
	}
	runCrossVerify(t, tc)
}

func TestCrossVerify_ConditionalJump(t *testing.T) {
	// if (false) { x = 1 } else { x = 2 } → result should be 2
	// STORE_VAR/LOAD_VAR take a string constant index as operand (CPU VM uses name lookup)
	constants := []interface{}{false, 1, 2, "x"}
	h := headerSize(constants)
	tc := cpuGPUTestCase{
		name:      "if-false-else",
		constants: constants,
		code: func() []byte {
			var code []byte
			code = append(code, 0x01, 0x00, 0x00, 0x00, 0x00) // 0: PUSH false
			code = append(code, 0x51)                           // 5: JUMP_IF_FALSE
			code = emitU32(code, h+25)                          // target: h+25
			code = append(code, 0x01, 0x01, 0x00, 0x00, 0x00)  // 10: PUSH 1
			code = append(code, 0x41, 0x03, 0x00, 0x00, 0x00)  // 15: STORE_VAR[3] ("x")
			code = append(code, 0x50)                           // 20: JUMP
			code = emitU32(code, h+35)                          // target: h+35
			code = append(code, 0x01, 0x02, 0x00, 0x00, 0x00)  // 25: PUSH 2
			code = append(code, 0x41, 0x03, 0x00, 0x00, 0x00)  // 30: STORE_VAR[3] ("x")
			code = append(code, 0x40, 0x03, 0x00, 0x00, 0x00)  // 35: LOAD_VAR[3] ("x")
			code = append(code, 0xFF)                           // 40: HALT
			return code
		}(),
		wantTag: TagInt,
		wantInt: 2,
	}
	runCrossVerify(t, tc)
}

func TestCrossVerify_Loop(t *testing.T) {
	// x = 0; while (x < 10) { x = x + 1 } ; result = x
	// STORE_VAR/LOAD_VAR take a string constant index as operand (CPU VM uses name lookup)
	constants := []interface{}{0, 1, 10, "x"}
	h := headerSize(constants)
	tc := cpuGPUTestCase{
		name:      "count to 10",
		constants: constants,
		code: func() []byte {
			var code []byte
			code = append(code, 0x01, 0x00, 0x00, 0x00, 0x00) // 0: PUSH 0
			code = append(code, 0x41, 0x03, 0x00, 0x00, 0x00) // 5: STORE_VAR[3] ("x")
			// loop start at 10:
			code = append(code, 0x40, 0x03, 0x00, 0x00, 0x00) // 10: LOAD_VAR[3] ("x")
			code = append(code, 0x01, 0x02, 0x00, 0x00, 0x00) // 15: PUSH 10
			code = append(code, 0x22)                           // 20: LT
			code = append(code, 0x51)                           // 21: JUMP_IF_FALSE
			code = emitU32(code, h+47)                          // target: h+47
			code = append(code, 0x40, 0x03, 0x00, 0x00, 0x00) // 26: LOAD_VAR[3] ("x")
			code = append(code, 0x01, 0x01, 0x00, 0x00, 0x00) // 31: PUSH 1
			code = append(code, 0x10)                           // 36: ADD
			code = append(code, 0x41, 0x03, 0x00, 0x00, 0x00) // 37: STORE_VAR[3] ("x")
			code = append(code, 0x50)                           // 42: JUMP
			code = emitU32(code, h+10)                          // target: h+10 (loop start)
			code = append(code, 0x40, 0x03, 0x00, 0x00, 0x00) // 47: LOAD_VAR[3] ("x")
			code = append(code, 0xFF)                           // 52: HALT
			return code
		}(),
		wantTag: TagInt,
		wantInt: 10,
	}
	runCrossVerify(t, tc)
}

func TestCrossVerify_ConstantTypes(t *testing.T) {
	tests := []cpuGPUTestCase{
		{
			name:      "int constant 99",
			constants: []interface{}{99},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH const 0
					0x61, // RETURN
				}
			}(),
			wantTag: TagInt,
			wantInt: 99,
		},
		{
			name:      "bool true constant",
			constants: []interface{}{true},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH const 0
					0xFF, // HALT
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
		{
			name:      "bool false constant",
			constants: []interface{}{false},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: false,
		},
		{
			name:      "null constant",
			constants: []interface{}{nil},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0xFF,
				}
			}(),
			wantTag: TagNull,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCrossVerify(t, tc)
		})
	}
}

func TestCrossVerify_DivByZero(t *testing.T) {
	tc := cpuGPUTestCase{
		name:      "division by zero",
		constants: []interface{}{10, 0},
		code: func() []byte {
			return []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 10
				0x01, 0x01, 0x00, 0x00, 0x00, // PUSH 0
				0x13, // DIV
				0xFF,
			}
		}(),
		wantError: true,
	}
	runCrossVerify(t, tc)
}

func TestCrossVerify_Expressions(t *testing.T) {
	tests := []cpuGPUTestCase{
		{
			// (2 + 3) * 4 = 20
			name:      "compound (2+3)*4",
			constants: []interface{}{2, 3, 4},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 2
					0x01, 0x01, 0x00, 0x00, 0x00, // PUSH 3
					0x10,                           // ADD → 5
					0x01, 0x02, 0x00, 0x00, 0x00, // PUSH 4
					0x12, // MUL → 20
					0xFF,
				}
			}(),
			wantTag: TagInt,
			wantInt: 20,
		},
		{
			// 100 - (10 + 20) = 70
			name:      "compound 100-(10+20)",
			constants: []interface{}{100, 10, 20},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 100
					0x01, 0x01, 0x00, 0x00, 0x00, // PUSH 10
					0x01, 0x02, 0x00, 0x00, 0x00, // PUSH 20
					0x10, // ADD → 30
					0x11, // SUB → 70
					0xFF,
				}
			}(),
			wantTag: TagInt,
			wantInt: 70,
		},
		{
			// Nested condition: (5 > 3) AND (2 < 4) → true
			name:      "nested comparison AND",
			constants: []interface{}{5, 3, 2, 4},
			code: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 5
					0x01, 0x01, 0x00, 0x00, 0x00, // PUSH 3
					0x23,                           // GT → true
					0x01, 0x02, 0x00, 0x00, 0x00, // PUSH 2
					0x01, 0x03, 0x00, 0x00, 0x00, // PUSH 4
					0x22,  // LT → true
					0x26,  // AND → true
					0xFF,
				}
			}(),
			wantTag:  TagBool,
			wantBool: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCrossVerify(t, tc)
		})
	}
}

func TestCrossVerify_ParallelVMs(t *testing.T) {
	// Run the same bytecode (3 + 4 = 7) on 50 parallel VMs via GPU dispatcher
	// and a single CPU VM. All should produce 7.
	constants := []interface{}{3, 4}
	code := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 3
		0x01, 0x01, 0x00, 0x00, 0x00, // PUSH 4
		0x10, // ADD
		0xFF, // HALT
	}
	bytecode := buildBytecode(constants, code)

	// CPU VM reference
	cpuResult, cpuErr := runCPUVM(bytecode)
	if cpuErr != nil {
		t.Fatalf("CPU VM error: %v", cpuErr)
	}
	cpuInt, ok := cpuResult.(vm.IntValue)
	if !ok || cpuInt.Val != 7 {
		t.Fatalf("CPU VM: expected Int(7), got %v", cpuResult)
	}

	// GPU parallel VMs
	d := NewDispatcher()
	d.hasGPU = false
	results, err := d.Execute(bytecode, 50)
	if err != nil {
		t.Fatalf("GPU Execute error: %v", err)
	}
	if len(results) != 50 {
		t.Fatalf("expected 50 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Error != nil {
			t.Fatalf("GPU VM %d error: %v", i, r.Error)
		}
		if r.IntVal != cpuInt.Val {
			t.Fatalf("GPU VM %d: expected %d, got %d (CPU/GPU mismatch)", i, cpuInt.Val, r.IntVal)
		}
	}
}

func TestCrossVerify_Mitosis(t *testing.T) {
	// push offset 0, MITOSIS, halt
	// The S opcode pushes a spawn-success indicator (true).
	tc := cpuGPUTestCase{
		name:      "mitosis basic",
		constants: []interface{}{0},
		code: func() []byte {
			return []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, // PUSH 0 (offset)
				0xC0,                           // MITOSIS — pops offset, pushes true
				0xFF,
			}
		}(),
		wantTag: TagBool,
		wantBool: true,
	}
	runCrossVerify(t, tc)
}

// ==========================================================================
// Bytecode builder helpers for cross-verify tests
// ==========================================================================

// buildTestBytecode is an alias for buildBytecode (defined in gpu_test.go).
// We use it here to create test programs programmatically.
func buildTestBytecode(constants []interface{}, code []byte) []byte {
	return buildBytecode(constants, code)
}

// emitPush emits an OP_PUSH instruction with the given constant index.
func emitPush(constIdx uint32) []byte {
	b := []byte{0x01}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, constIdx)
	return append(b, operand...)
}

// emitStoreVar emits an OP_STORE_VAR instruction.
func emitStoreVar(varIdx uint32) []byte {
	b := []byte{0x41}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, varIdx)
	return append(b, operand...)
}

// emitLoadVar emits an OP_LOAD_VAR instruction.
func emitLoadVar(varIdx uint32) []byte {
	b := []byte{0x40}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, varIdx)
	return append(b, operand...)
}

// emitJump emits an OP_JUMP instruction.
func emitJump(target uint32) []byte {
	b := []byte{0x50}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, target)
	return append(b, operand...)
}

// emitJumpIfFalse emits an OP_JUMP_IF_FALSE instruction.
func emitJumpIfFalse(target uint32) []byte {
	b := []byte{0x51}
	operand := make([]byte, 4)
	binary.LittleEndian.PutUint32(operand, target)
	return append(b, operand...)
}

// TestCrossVerify_UsingBuilder tests the same scenarios but using the builder
// helpers to construct bytecode more readably.
func TestCrossVerify_UsingBuilder(t *testing.T) {
	tests := []struct {
		name      string
		constants []interface{}
		buildCode func() []byte
		wantTag   uint32
		wantInt   int64
		wantBool  bool
	}{
		{
			name:      "builder: add",
			constants: []interface{}{10, 20, "x"},
			buildCode: func() []byte {
				return []byte{
					0x01, 0x00, 0x00, 0x00, 0x00,
					0x01, 0x01, 0x00, 0x00, 0x00,
					0x10,
					0xFF,
				}
			},
			wantTag: TagInt,
			wantInt: 30,
		},
		{
			name:      "builder: store-load-mul",
			constants: []interface{}{7, 6, "x", "y"},
			buildCode: func() []byte {
				var code []byte
				code = append(code, emitPush(0)...)
				code = append(code, emitStoreVar(2)...)
				code = append(code, emitPush(1)...)
				code = append(code, emitStoreVar(3)...)
				code = append(code, emitLoadVar(2)...)
				code = append(code, emitLoadVar(3)...)
				code = append(code, 0x12) // MUL
				code = append(code, 0xFF) // HALT
				return code
			},
			wantTag: TagInt,
			wantInt: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := cpuGPUTestCase{
				name:      tt.name,
				constants: tt.constants,
				code:      tt.buildCode(),
				wantTag:   tt.wantTag,
				wantInt:   tt.wantInt,
				wantBool:  tt.wantBool,
			}
			runCrossVerify(t, tc)
		})
	}
}
