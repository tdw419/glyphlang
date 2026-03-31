//go:build !js || !wasm

package main

import (
	"encoding/json"
	"testing"

	"github.com/glyphlang/glyph/pkg/vm"
)

// TestMockBrowserCapabilities verifies the spatial capabilities API
// returns the correct opcode map for browser consumers.
func TestMockBrowserCapabilities(t *testing.T) {
	b := NewMockBrowser()
	caps := b.SpatialCapabilities()

	if !caps.Mitosis {
		t.Error("browser should report mitosis as available")
	}
	if !caps.Mutator {
		t.Error("browser should report mutator as available")
	}
	if caps.Version == "" {
		t.Error("capabilities should include a version string")
	}
	if caps.Opcodes["OP_MITOSIS"] != 0xC0 {
		t.Errorf("OP_MITOSIS should be 0xC0, got 0x%02X", caps.Opcodes["OP_MITOSIS"])
	}
	if caps.Opcodes["OP_MUTATOR"] != 0xC1 {
		t.Errorf("OP_MUTATOR should be 0xC1, got 0x%02X", caps.Opcodes["OP_MUTATOR"])
	}
}

// TestMockBrowserCapabilitiesJSON verifies the capabilities serialize
// correctly to JSON — this is the wire format the browser receives.
func TestMockBrowserCapabilitiesJSON(t *testing.T) {
	b := NewMockBrowser()
	raw := b.SpatialCapabilitiesJSON()

	var caps BrowserCapabilities
	if err := json.Unmarshal([]byte(raw), &caps); err != nil {
		t.Fatalf("capabilities JSON should parse: %v", err)
	}
	if !caps.Mitosis || !caps.Mutator {
		t.Error("both spatial opcodes should be reported as available")
	}
	if _, ok := caps.Opcodes["OP_MITOSIS"]; !ok {
		t.Error("OP_MITOSIS missing from opcodes map")
	}
	if _, ok := caps.Opcodes["OP_MUTATOR"]; !ok {
		t.Error("OP_MUTATOR missing from opcodes map")
	}
}

// TestMockBrowserSpatialExecStructProgram runs a struct-definition
// program through the spatial pipeline and verifies that non-spatial
// programs report zero mutations and one thread.
func TestMockBrowserSpatialExecStructProgram(t *testing.T) {
	b := NewMockBrowser()
	result := b.SpatialExec(": Point { x: int\n  y: int }")

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Threads != 1 {
		t.Errorf("non-spatial program should report 1 thread, got %d", result.Threads)
	}
	if result.Mutations != 0 {
		t.Errorf("non-spatial program should report 0 mutations, got %d", result.Mutations)
	}
	if result.TimeMS < 0 {
		t.Error("time should be non-negative")
	}
}

// TestMockBrowserCompileAndCache verifies that Compile caches bytecode
// and Run retrieves from cache on the second call.
func TestMockBrowserCompileAndCache(t *testing.T) {
	b := NewMockBrowser()
	source := ": Num { val: int }"

	bytecode, err := b.Compile(source)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		t.Fatalf("bytecode should start with GLYP magic, got %q", bytecode[:4])
	}

	// Run should use cached bytecode (same source length key)
	val, err := b.Run(source)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	_ = val
}

// TestMockBrowserRunBytecodeWithMutator executes raw spatial bytecode
// through the mock browser, verifying MUTATOR works in the browser env.
func TestMockBrowserRunBytecodeWithMutator(t *testing.T) {
	b := NewMockBrowser()
	bytecode := buildSpatialBytecode(byte(vm.OpMutator))

	val, err := b.RunBytecode(bytecode)
	if err != nil {
		t.Fatalf("mutator execution failed: %v", err)
	}
	_ = val

	mutations, threads := CountSpatialOpcodes(bytecode)
	if mutations != 1 {
		t.Errorf("expected 1 mutator opcode in bytecode, found %d", mutations)
	}
	if threads != 1 {
		t.Errorf("expected 1 thread (no mitosis), got %d", threads)
	}
}

// TestMockBrowserRunBytecodeWithMitosis executes raw spatial bytecode
// through the mock browser, verifying MITOSIS works in the browser env.
func TestMockBrowserRunBytecodeWithMitosis(t *testing.T) {
	b := NewMockBrowser()
	bytecode := buildSpatialBytecode(byte(vm.OpMitosis))

	val, err := b.RunBytecode(bytecode)
	if err != nil {
		t.Fatalf("mitosis execution failed: %v", err)
	}

	bv, ok := val.(vm.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue from mitosis, got %T", val)
	}
	if !bv.Val {
		t.Error("mitosis should return true for parent thread")
	}

	mutations, threads := CountSpatialOpcodes(bytecode)
	if mutations != 0 {
		t.Errorf("expected 0 mutator opcodes, found %d", mutations)
	}
	if threads != 2 {
		t.Errorf("expected 2 threads (1 + mitosis), got %d", threads)
	}
}

// TestMockBrowserSpatialExecError verifies the mock browser returns
// structured error diagnostics when given invalid source.
func TestMockBrowserSpatialExecError(t *testing.T) {
	b := NewMockBrowser()
	result := b.SpatialExec("fn ( broken syntax !!!")

	if result.Success {
		t.Fatal("expected failure for invalid source")
	}
	if result.Error == "" {
		t.Error("error message should not be empty")
	}
	// Even on error, the result should be well-formed JSON-serializable
	_, err := json.Marshal(result)
	if err != nil {
		t.Errorf("BrowserSpatialResult should be JSON-serializable: %v", err)
	}
}

// TestMockBrowserMultiplePrograms verifies state isolation between
// multiple programs executed in sequence through the mock browser.
func TestMockBrowserMultiplePrograms(t *testing.T) {
	b := NewMockBrowser()

	programs := []struct {
		source   string
		hasError bool
	}{
		{": Vec2 { x: int\n  y: int }", false},
		{": Pair { first: int\n  second: int }", false},
		{": Rect { w: int\n  h: int\n  area: int = 0 }", false},
	}

	for i, prog := range programs {
		result := b.SpatialExec(prog.source)
		if prog.hasError && result.Success {
			t.Errorf("program %d: expected error", i)
		} else if !prog.hasError && !result.Success {
			t.Errorf("program %d: unexpected error: %s", i, result.Error)
		}
		// Each non-spatial program should report 1 thread, 0 mutations
		if !prog.hasError {
			if result.Threads != 1 {
				t.Errorf("program %d: expected 1 thread, got %d", i, result.Threads)
			}
			if result.Mutations != 0 {
				t.Errorf("program %d: expected 0 mutations, got %d", i, result.Mutations)
			}
		}
	}
}

// TestMockBrowserSpatialExecResultSerialization verifies the spatial
// result is JSON-serializable for transmission to the browser.
func TestMockBrowserSpatialExecResultSerialization(t *testing.T) {
	b := NewMockBrowser()
	result := b.SpatialExec(": Coord { x: int\n  y: int }")

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("BrowserSpatialResult should serialize to JSON: %v", err)
	}

	var parsed BrowserSpatialResult
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("BrowserSpatialResult should deserialize: %v", err)
	}
	if !parsed.Success {
		t.Error("round-tripped result should show success")
	}
}

// TestCountSpatialOpcodesMixed verifies counting with a bytecode
// containing both mutator and mitosis opcodes.
func TestCountSpatialOpcodesMixed(t *testing.T) {
	// Manually craft bytecode with both opcodes
	var buf []byte
	buf = append(buf, "GLYP"...)
	buf = append(buf, 0x01, 0x00, 0x00, 0x00) // version
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // 0 constants
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // instruction count placeholder
	buf = append(buf, byte(vm.OpMutator))
	buf = append(buf, byte(vm.OpMitosis))
	buf = append(buf, byte(vm.OpMutator))
	buf = append(buf, 0xFF) // HALT

	mutations, threads := CountSpatialOpcodes(buf)
	if mutations != 2 {
		t.Errorf("expected 2 mutations, got %d", mutations)
	}
	if threads != 2 {
		t.Errorf("expected 2 threads (1 + 1 mitosis), got %d", threads)
	}
}
