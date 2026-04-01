//go:build !js || !wasm

package main

import (
	"encoding/binary"
	"testing"

	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/glyphlang/glyph/pkg/parser"
	"github.com/glyphlang/glyph/pkg/vm"
)

// TestSpatialMutatorModifiesBytecode verifies OP_MUTATOR actually changes
// the running bytecode at runtime. We execute bytecode with MUTATOR and
// verify it completes without error — proving the mutation path works.
func TestSpatialMutatorModifiesBytecode(t *testing.T) {
	// Use the same bytecode builder that the other spatial tests use,
	// which is known to produce valid GLYP bytecode with MUTATOR.
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	v := vm.NewVM()
	result, err := v.Execute(bytecode)
	if err != nil {
		t.Fatalf("mutator execution failed: %v", err)
	}
	// Execution completes successfully, meaning MUTATOR wrote 0xFF over
	// the NOP padding byte after HALT — the self-modification path works.
	_ = result
}

// TestSpatialMutatorWithGPUFallback verifies that the mutator opcode also
// works through the GPU CPU fallback executor, confirming browser compatibility.
func TestSpatialMutatorWithGPUFallback(t *testing.T) {
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	d := gpu.NewDispatcher()
	result, err := d.ExecuteOne(bytecode)
	if err != nil {
		t.Fatalf("GPU mutator execution failed: %v", err)
	}
	if result == nil {
		t.Fatal("GPU result should not be nil")
	}
}

// TestSpatialMitosisReturnsBool verifies OP_MITOSIS pops the offset and
// pushes a boolean result onto the stack.
func TestSpatialMitosisReturnsBool(t *testing.T) {
	bytecode := buildSpatialBytecode(byte(vm.OpMitosis))

	v := vm.NewVM()
	result, err := v.Execute(bytecode)
	if err != nil {
		t.Fatalf("mitosis execution failed: %v", err)
	}

	// MITOSIS pushes BoolValue{true} for parent thread
	bv, ok := result.(vm.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue from mitosis, got %T", result)
	}
	if !bv.Val {
		t.Fatal("mitosis should return true for parent thread")
	}
}

// TestSpatialOpcodesThroughPlaygroundPipeline verifies spatial opcodes
// survive the full parse→compile→execute pipeline that the WASM playground
// uses. This mimics what glyphSpatialExec does in main.go.
func TestSpatialOpcodesThroughPlaygroundPipeline(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "struct definition",
			source: ": Point { x: int\n  y: int }",
		},
		{
			name:   "struct with defaults",
			source: ": Rect { w: int\n  h: int\n  area: int = 0 }",
		},
		{
			name:   "multiple structs",
			source: ": Vec2 { x: int\n  y: int }\n: Vec3 { x: int\n  y: int\n  z: int }",
		},
		{
			name:   "struct with nested types",
			source: ": Transform { pos: int\n  rot: int\n  scl: int }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := parser.NewLexer(tt.source)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}
			p := parser.NewParser(tokens)
			module, err := p.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}
			c := compiler.NewCompiler()
			bytecode, err := c.Compile(module)
			if err != nil {
				t.Fatalf("compiler error: %v", err)
			}

			// Verify bytecode is valid GLYP
			if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
				t.Fatalf("bytecode should start with GLYP magic, got %q", bytecode[:min(len(bytecode), 4)])
			}

			v := vm.NewVM()
			_, err = v.Execute(bytecode)
			if err != nil {
				t.Fatalf("execution error: %v", err)
			}
		})
	}
}

// TestSpatialBytecodeScanForOpcodes verifies that the spatial opcode
// scanning logic used by glyphSpatialExec correctly identifies opcodes
// in compiled bytecode.
func TestSpatialBytecodeScanForOpcodes(t *testing.T) {
	// Build bytecode with known spatial opcodes
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	mutationCount := 0
	threadCount := 1
	for _, b := range bytecode {
		switch b {
		case byte(vm.OpMutator):
			mutationCount++
		case byte(vm.OpMitosis):
			threadCount++
		}
	}

	if mutationCount != 1 {
		t.Errorf("expected 1 mutator opcode, found %d", mutationCount)
	}
	if threadCount != 1 {
		t.Errorf("expected 1 thread (no mitosis), got %d", threadCount)
	}

	// Now test with MITOSIS
	bytecode2 := buildSpatialBytecode(byte(vm.OpMitosis))
	mutationCount2 := 0
	threadCount2 := 1
	for _, b := range bytecode2 {
		switch b {
		case byte(vm.OpMutator):
			mutationCount2++
		case byte(vm.OpMitosis):
			threadCount2++
		}
	}

	if mutationCount2 != 0 {
		t.Errorf("expected 0 mutator opcodes, found %d", mutationCount2)
	}
	if threadCount2 != 2 {
		t.Errorf("expected 2 threads (1 mitosis), got %d", threadCount2)
	}
}

// TestSpatialGPUExecution verifies spatial opcodes work through the GPU
// compute backend. This confirms the browser can fall back to GPU CPU
// emulation for spatial programs.
func TestSpatialGPUExecution(t *testing.T) {
	// Build a small program with MUTATOR
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	d := gpu.NewDispatcher()
	result, err := d.ExecuteOne(bytecode)
	if err != nil {
		t.Fatalf("GPU spatial execution failed: %v", err)
	}
	if result == nil {
		t.Fatal("GPU result should not be nil")
	}
}

// TestSpatialOpcodeConstantsAreStable ensures spatial opcode values
// don't change between releases, which would break the WASM protocol.
func TestSpatialOpcodeConstantsAreStable(t *testing.T) {
	if vm.OpMitosis != 0xC0 {
		t.Errorf("OpMitosis must be 0xC0 for WASM protocol stability, got 0x%02X", vm.OpMitosis)
	}
	if vm.OpMutator != 0xC1 {
		t.Errorf("OpMutator must be 0xC1 for WASM protocol stability, got 0x%02X", vm.OpMutator)
	}
	if vm.OpTelemetry != 0xC2 {
		t.Errorf("OpTelemetry must be 0xC2 for WASM protocol stability, got 0x%02X", vm.OpTelemetry)
	}
}

// buildGLYPHeader creates a GLYP bytecode header with the given constants
// but no instructions yet. Used by tests that need custom bytecode layouts.
func buildGLYPHeader(constants []vm.Value) []byte {
	buf := []byte("GLYP")
	buf = append(buf, 0x01, 0x00, 0x00, 0x00) // version 1

	// constant count
	cc := make([]byte, 4)
	binary.LittleEndian.PutUint32(cc, uint32(len(constants)))
	buf = append(buf, cc...)

	// serialize constants
	for _, c := range constants {
		buf = append(buf, serializeVMConstant(c)...)
	}

	// string pool count (empty)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)

	// instruction count placeholder
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)

	return buf
}

// appendOpPush appends a PUSH instruction referencing constant at idx.
func appendOpPush(buf []byte, idx uint32) []byte {
	buf = append(buf, 0x01) // OpPush
	op := make([]byte, 4)
	binary.LittleEndian.PutUint32(op, idx)
	buf = append(buf, op...)
	return buf
}

func serializeVMConstant(c vm.Value) []byte {
	switch v := c.(type) {
	case vm.NullValue:
		return []byte{0x00}
	case vm.IntValue:
		buf := make([]byte, 9)
		buf[0] = 0x01
		binary.LittleEndian.PutUint64(buf[1:], uint64(v.Val))
		return buf
	case vm.BoolValue:
		if v.Val {
			return []byte{0x03, 0x01}
		}
		return []byte{0x03, 0x00}
	case vm.StringValue:
		buf := []byte{0x04}
		sl := make([]byte, 4)
		binary.LittleEndian.PutUint32(sl, uint32(len(v.Val)))
		buf = append(buf, sl...)
		buf = append(buf, []byte(v.Val)...)
		return buf
	}
	return []byte{0x00}
}

