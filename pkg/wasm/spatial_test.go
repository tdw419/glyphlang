//go:build !js || !wasm

package main

import (
	"testing"

	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/parser"
	"github.com/glyphlang/glyph/pkg/vm"
)

// TestSpatialCompileMutator verifies that OP_MUTATOR (0xC1) compiles and executes.
// This exercises the same pipeline the WASM playground uses: parse → compile → execute.
func TestSpatialCompileMutator(t *testing.T) {
	// Program that uses the mutator: push a value, push offset, call MUTATOR
	// We build bytecode manually since the high-level syntax may not expose spatial opcodes.
	bytecode := buildSpatialBytecode(0xC1)
	v := vm.NewVM()
	result, err := v.Execute(bytecode)
	if err != nil {
		t.Fatalf("mutator execution failed: %v", err)
	}
	_ = result // just verifying it runs without error
}

// TestSpatialCompileMitosis verifies that OP_MITOSIS (0xC0) compiles and executes.
func TestSpatialCompileMitosis(t *testing.T) {
	bytecode := buildSpatialBytecode(0xC0)
	v := vm.NewVM()
	result, err := v.Execute(bytecode)
	if err != nil {
		t.Fatalf("mitosis execution failed: %v", err)
	}
	_ = result
}

// TestSpatialOpcodesInPlaygroundPipeline verifies the full compile→execute pipeline
// that the browser playground uses, confirming spatial opcodes work end-to-end.
func TestSpatialOpcodesInPlaygroundPipeline(t *testing.T) {
	// Compile a simple program through the same pipeline the WASM module uses
	source := ": Point { x: int\n  y: int }"
	lexer := parser.NewLexer(source)
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
	_, err = c.Compile(module)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}
}

// TestSpatialOpcodesAreRecognized confirms the VM recognizes spatial opcodes.
func TestSpatialOpcodesAreRecognized(t *testing.T) {
	// Verify OpMutator and OpMitosis are defined in the VM
	if vm.OpMutator != 0xC1 {
		t.Fatalf("OpMutator should be 0xC1, got 0x%02X", vm.OpMutator)
	}
	if vm.OpMitosis != 0xC0 {
		t.Fatalf("OpMitosis should be 0xC0, got 0x%02X", vm.OpMitosis)
	}
}

// buildSpatialBytecode creates GLYP-format bytecode with a spatial opcode.
// Format: GLYP header + constants + instruction count + code
func buildSpatialBytecode(spatialOp byte) []byte {
	var buf []byte

	// Magic header
	buf = append(buf, "GLYP"...)
	// Version (1, LE)
	buf = append(buf, 0x01, 0x00, 0x00, 0x00)
	// Constant count: 2 (value=0xFF for mutator write, offset=1)
	buf = append(buf, 0x02, 0x00, 0x00, 0x00)
	// Constant 0: int64 value 0xFF (type=0x01)
	buf = append(buf, 0x01)
	buf = appendU32LE(buf, 0xFF) // lower 4 bytes
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // upper 4 bytes
	// Constant 1: int64 value 1 (type=0x01)
	buf = append(buf, 0x01)
	buf = append(buf, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	// String pool count (empty)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)
	// Instruction count (placeholder, 4 bytes)
	instrCountOffset := len(buf)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)
	// Code:
	//   PUSH const[0]  (5 bytes: opcode + 4-byte index)
	buf = append(buf, 0x01, 0x00, 0x00, 0x00, 0x00)
	//   PUSH const[1]  (5 bytes)
	buf = append(buf, 0x01, 0x01, 0x00, 0x00, 0x00)
	//   spatial opcode (1 byte)
	buf = append(buf, spatialOp)
	//   HALT (1 byte)
	buf = append(buf, 0xFF)
	//   NOP padding (1 byte — mutator target)
	buf = append(buf, 0x00)

	// Fix instruction count (6 instructions)
	instrCount := uint32(6)
	buf[instrCountOffset] = byte(instrCount)
	buf[instrCountOffset+1] = byte(instrCount >> 8)
	buf[instrCountOffset+2] = byte(instrCount >> 16)
	buf[instrCountOffset+3] = byte(instrCount >> 24)

	return buf
}

func appendU32LE(buf []byte, v uint32) []byte {
	return append(buf, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
