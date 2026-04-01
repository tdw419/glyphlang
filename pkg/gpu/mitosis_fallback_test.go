package gpu

import (
	"strings"
	"testing"
)

// TestMitosisGPUFallbackWarning verifies that when GPU execution fails during
// Mitosis, a warning is logged and execution falls back to CPU successfully.
// This tests the infrastructure for CPU fallback (full fallback is #78).
func TestMitosisGPUFallbackWarning(t *testing.T) {
	// Program that spawns 1 child:
	// 0-4:  push const 0 (offset=1)
	// 5:    MITOSIS → child at 5+1+1 = 7
	// 6:    HALT (parent)
	// 7-11: push const 1 (value=99) — child code
	// 12:   HALT (child)
	constants := []interface{}{1, 99}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 1 (offset)
	code = append(code, OpMitosis)       // 5: S opcode
	code = append(code, 0xFF)            // 6: HALT (parent)
	code = append(code, pushConst(1)...) // 7-11: push 99
	code = append(code, 0xFF)            // 12: HALT (child)

	bytecode := buildBytecode(constants, code)

	m := NewMitosisVM(256)
	// Force GPU to fail so the fallback path is exercised
	m.ForceGPUError = true

	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}

	// Should still get results via CPU fallback
	if len(results) < 2 {
		t.Fatalf("expected at least 2 thread results (1 parent + 1 child), got %d", len(results))
	}

	// Verify parent exists
	var root *ThreadResult
	for i := range results {
		if results[i].ParentID == -1 {
			root = &results[i]
			break
		}
	}
	if root == nil {
		t.Fatal("no root thread found")
	}

	// Verify the fallback warning was captured
	warnings := m.FallbackWarnings()
	if len(warnings) == 0 {
		t.Fatal("expected at least one GPU fallback warning")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "GPU mitosis execution failed") &&
			strings.Contains(w, "falling back to CPU") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("warning should mention GPU failure and CPU fallback, got: %v", warnings)
	}
}

// TestMitosisGPUFallbackNoWarningWithoutError verifies that no fallback
// warning is emitted when GPU is not forced to fail (normal CPU execution path).
func TestMitosisGPUFallbackNoWarningWithoutError(t *testing.T) {
	constants := []interface{}{1, 99}
	var code []byte
	code = append(code, pushConst(0)...) // push 1 (offset)
	code = append(code, OpMitosis)       // S opcode
	code = append(code, 0xFF)            // HALT (parent)
	code = append(code, pushConst(1)...) // push 99
	code = append(code, 0xFF)            // HALT (child)

	bytecode := buildBytecode(constants, code)

	m := NewMitosisVM(256)
	// ForceGPUError is false (default) — no GPU error should be simulated
	m.ForceGPUError = false

	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// No fallback warning should have been emitted
	warnings := m.FallbackWarnings()
	if len(warnings) != 0 {
		t.Errorf("expected no warnings when ForceGPUError=false, got: %v", warnings)
	}
}

// TestMitosisGPUFallbackResultsCorrect verifies that the CPU fallback produces
// correct child results even when GPU fails.
func TestMitosisGPUFallbackResultsCorrect(t *testing.T) {
	constants := []interface{}{1, 99}
	var code []byte
	code = append(code, pushConst(0)...) // push 1 (offset)
	code = append(code, OpMitosis)       // S opcode
	code = append(code, 0xFF)            // HALT (parent)
	code = append(code, pushConst(1)...) // push 99
	code = append(code, 0xFF)            // HALT (child)

	bytecode := buildBytecode(constants, code)

	m := NewMitosisVM(256)
	m.ForceGPUError = true

	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}

	// Find the child thread and verify it computed correctly
	var childResult *ThreadResult
	for i := range results {
		if results[i].ParentID != -1 {
			childResult = &results[i]
			break
		}
	}
	if childResult == nil {
		t.Fatal("no child thread found")
	}
	if childResult.Result.IntVal != 99 {
		t.Errorf("child result: expected 99, got %d", childResult.Result.IntVal)
	}
	if childResult.Result.Error != nil {
		t.Errorf("child had unexpected error: %v", childResult.Result.Error)
	}
}
