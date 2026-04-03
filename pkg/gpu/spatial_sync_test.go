package gpu

import (
	"testing"
)

// TestSpatialOpcodeNames verifies spatial opcodes appear in the spatial grid
// visualization labels. Issue #19 requires MITOSIS and MUTATOR to be
// recognizable in the geometry-os-gpu substrate.
func TestSpatialOpcodeNames(t *testing.T) {
	tests := map[byte]string{
		0xC0: "MITOSIS",
		0xC1: "MUTATOR",
		0xC2: "TELEMETRY",
	}
	for op, want := range tests {
		got := opcodeName(op)
		if got != want {
			t.Errorf("opcodeName(0x%02x) = %q, want %q", op, got, want)
		}
	}
}

// TestErrorStringMutatorOOB verifies ErrorString handles ErrMutatorOOB.
func TestErrorStringMutatorOOB(t *testing.T) {
	got := ErrorString(ErrMutatorOOB)
	if got == "" || got == "unknown error 5" {
		t.Fatalf("ErrorString(ErrMutatorOOB) = %q, want specific mutator OOB message", got)
	}
}

// TestSpatialOpcodesInGrid verifies that spatial opcodes are rendered in the
// Hilbert spatial grid visualization, proving synchronization between Go VM
// and the WGSL substrate (issue #19).
func TestSpatialOpcodesInGrid(t *testing.T) {
	// Build a program with MITOSIS and MUTATOR opcodes
	constants := []interface{}{42, 0}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push value
	code = append(code, pushConst(1)...) // 5-9: push offset
	code = append(code, 0xC1)            // 10: MUTATOR
	code = append(code, 0xC0)            // 11: MITOSIS
	code = append(code, 0xFF)            // 12: HALT

	bytecode := buildBytecode(constants, code)
	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		t.Fatal(err)
	}

	grid := NewSpatialGrid(bytecode, int(config.CodeOffset))
	if grid.Width == 0 {
		t.Fatal("grid has zero width")
	}

	// Verify that the spatial opcodes are visible in the rendered grid
	text := grid.RenderText()
	t.Logf("Spatial grid:\n%s", text)

	// The grid should contain the hex representation of our spatial opcodes
	if !containsStr(text, "c0") && !containsStr(text, "C0") {
		// Check if MITOSIS appears as label via opcodeName
		found := false
		for _, cell := range grid.Cells {
			if cell.Opcode == 0xC0 && cell.Label == "MITOSIS" {
				found = true
				break
			}
		}
		if !found {
			t.Error("MITOSIS opcode not visible in spatial grid")
		}
	}

	if !containsStr(text, "c1") && !containsStr(text, "C1") {
		found := false
		for _, cell := range grid.Cells {
			if cell.Opcode == 0xC1 && cell.Label == "MUTATOR" {
				found = true
				break
			}
		}
		if !found {
			t.Error("MUTATOR opcode not visible in spatial grid")
		}
	}
}

// TestTelemetryCPU verifies the TELEMETRY opcode (0xC2) in the CPU fallback.
// TELEMETRY pops a value from the stack and discards it (write to telemetry buffer).
// After telemetry pops, only the remaining stack values should be visible.
func TestTelemetryCPU(t *testing.T) {
	// push 42, push 99, TELEMETRY (pops 99, discards), HALT → result = 42
	constants := []interface{}{42, 99}
	var code []byte
	code = append(code, pushConst(0)...) // push 42
	code = append(code, pushConst(1)...) // push 99
	code = append(code, 0xC2)            // TELEMETRY — pops 99, discards
	code = append(code, 0xFF)            // HALT → top of stack is 42

	d := NewDispatcher()
	d.SetCPUFallback() // Force CPU path for deterministic telemetry behavior
	result, err := d.ExecuteOne(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("VM error: %v", result.Error)
	}
	// TELEMETRY should pop 99, leaving 42 as the top of stack
	if result.IntVal != 42 {
		t.Fatalf("expected 42 (telemetry should have popped 99), got %d", result.IntVal)
	}
}

// TestTelemetryGPUCompatible verifies IsGPUCompatible accepts 0xC2.
func TestTelemetryGPUCompatible(t *testing.T) {
	constants := []interface{}{0}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, 0xC2) // TELEMETRY
	code = append(code, 0xFF)

	if !IsGPUCompatible(buildBytecode(constants, code)) {
		t.Fatal("bytecode with TELEMETRY should be GPU compatible")
	}
}

// TestWGSLTelemetryOpcode verifies the WGSL shader declares OP_TELEMETRY.
func TestWGSLTelemetryOpcode(t *testing.T) {
	d := NewDispatcher()
	src := d.ShaderSource()
	if !containsStr(src, "OP_TELEMETRY") {
		t.Fatal("WGSL shader missing OP_TELEMETRY declaration")
	}
	if !containsStr(src, "case OP_TELEMETRY") {
		t.Fatal("WGSL shader missing OP_TELEMETRY handler")
	}
}
