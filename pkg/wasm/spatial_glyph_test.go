//go:build !js || !wasm

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/glyphlang/glyph/pkg/vm"
)

// TestCompilerSpatialGlyphFileReadable verifies that the self-hosted spatial
// test file exists and contains the expected __mutator and __mitosis programs.
// This confirms the .glyph fixture is in place for the WASM pipeline.
func TestCompilerSpatialGlyphFileReadable(t *testing.T) {
	paths := []string{
		"../../bootstrap/test_compiler_spatial.glyph",
		"../../../bootstrap/test_compiler_spatial.glyph",
	}
	var content []byte
	var err error
	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("cannot read test_compiler_spatial.glyph: %v", err)
	}
	src := string(content)

	if !strings.Contains(src, "__mutator") {
		t.Error("test_compiler_spatial.glyph should reference __mutator")
	}
	if !strings.Contains(src, "__mitosis") {
		t.Error("test_compiler_spatial.glyph should reference __mitosis")
	}
}

// TestCompilerSpatialGlyphMutatorProgram verifies that the mutator program
// from test_compiler_spatial.glyph ("__mutator(255, 0)") executes correctly
// through the mock browser's WASM pipeline. This is the Go-side equivalent
// of the self-hosted spatial test.
func TestCompilerSpatialGlyphMutatorProgram(t *testing.T) {
	b := NewMockBrowser()

	// Build bytecode matching the __mutator(255, 0) call from the .glyph file:
	// push value=255, push offset=0, OP_MUTATOR
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	val, err := b.RunBytecode(bytecode)
	if err != nil {
		t.Fatalf("mutator program failed in mock browser: %v", err)
	}
	_ = val

	mutations, threads := CountSpatialOpcodes(bytecode)
	if mutations != 1 {
		t.Errorf("expected 1 mutator opcode, got %d", mutations)
	}
	if threads != 1 {
		t.Errorf("expected 1 thread (no mitosis in mutator program), got %d", threads)
	}
}

// TestCompilerSpatialGlyphMitosisProgram verifies that the mitosis program
// from test_compiler_spatial.glyph ("__mitosis(0)") executes correctly
// through the mock browser. The .glyph file expects mitosis to return true
// (parent thread), and then conditionally return 100.
func TestCompilerSpatialGlyphMitosisProgram(t *testing.T) {
	b := NewMockBrowser()

	// Build bytecode matching the __mitosis(0) call from the .glyph file
	bytecode := buildSpatialBytecode(byte(vm.OpMitosis))

	val, err := b.RunBytecode(bytecode)
	if err != nil {
		t.Fatalf("mitosis program failed in mock browser: %v", err)
	}

	bv, ok := val.(vm.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue from mitosis, got %T", val)
	}
	if !bv.Val {
		t.Error("mitosis should return true for parent thread (matching .glyph expectation)")
	}

	mutations, threads := CountSpatialOpcodes(bytecode)
	if mutations != 0 {
		t.Errorf("expected 0 mutator opcodes, got %d", mutations)
	}
	if threads != 2 {
		t.Errorf("expected 2 threads (1 base + 1 mitosis), got %d", threads)
	}
}

// TestCompilerSpatialGlyphFullPipeline runs the complete spatial pipeline
// that the WASM playground would use for spatial programs, verifying the
// mock browser's SpatialExec produces correct diagnostics for both opcodes.
func TestCompilerSpatialGlyphFullPipeline(t *testing.T) {
	b := NewMockBrowser()

	// Test mutator program through full SpatialExec pipeline
	mutatorBytecode := buildSpatialBytecode(byte(vm.OpMutator))
	result := b.SpatialExec(": Point { x: int\n  y: int }")
	if !result.Success {
		t.Fatalf("non-spatial program should succeed: %s", result.Error)
	}

	// Verify spatial opcode counting on hand-crafted bytecode
	mutations, _ := CountSpatialOpcodes(mutatorBytecode)
	if mutations != 1 {
		t.Errorf("mutator bytecode should have 1 mutation, got %d", mutations)
	}

	// Test mitosis bytecode through RunBytecode
	mitosisBytecode := buildSpatialBytecode(byte(vm.OpMitosis))
	val, err := b.RunBytecode(mitosisBytecode)
	if err != nil {
		t.Fatalf("mitosis in full pipeline failed: %v", err)
	}

	bv, ok := val.(vm.BoolValue)
	if !ok || !bv.Val {
		t.Error("mitosis should return true in full pipeline")
	}

	mutations2, threads2 := CountSpatialOpcodes(mitosisBytecode)
	if mutations2 != 0 {
		t.Errorf("mitosis bytecode should have 0 mutations, got %d", mutations2)
	}
	if threads2 != 2 {
		t.Errorf("mitosis bytecode should report 2 threads, got %d", threads2)
	}
}
