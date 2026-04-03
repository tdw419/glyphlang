package gpu

import (
	"runtime"
	"strings"
	"testing"
)

// TestCPUOpMitosis verifies that OP_MITOSIS on CPU either produces a clear
// error or spawns goroutines — never silently ignores the instruction.
//
// This is the SEC-1.1 acceptance test from the GPU Substrate Correctness roadmap.
// Two CPU execution paths exist:
//   1. pkg/vm VM: returns error "OP_MITOSIS requires GPU execution"
//   2. pkg/gpu MitosisVM dispatcher: spawns goroutine children via CpuSpawnRequest
//
// This test covers path #2 — the MitosisVM CPU fallback should spawn real goroutines
// when it encounters OP_MITOSIS (0xC0) and the child should produce correct results.
func TestCPUOpMitosis(t *testing.T) {
	// Bytecode: push offset(1), MITOSIS, HALT(parent), push 42, HALT(child)
	//
	// Layout:
	//   0-4:  push const 0 (offset=1)
	//   5:    MITOSIS (0xC0) → child at PC+1+1 = 7
	//   6:    HALT (parent)
	//   7-11: push const 1 (value=42) — child code
	//   12:   HALT (child)
	constants := []interface{}{1, 42}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 1 (offset)
	code = append(code, OpMitosisByte)   // 5: S opcode
	code = append(code, 0xFF)            // 6: HALT (parent)
	code = append(code, pushConst(1)...) // 7-11: push 42 (child result)
	code = append(code, 0xFF)            // 12: HALT (child)

	bytecode := buildBytecode(constants, code)

	// Use MitosisVM (CPU path, no GPU available)
	m := NewMitosisVM(256)

	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis returned unexpected error: %v", err)
	}

	// Must have at least 2 thread results: 1 parent + 1 child
	if len(results) < 2 {
		t.Fatalf("expected at least 2 thread results (parent + child), got %d", len(results))
	}

	// Identify root and children
	var root *ThreadResult
	var children []ThreadResult
	for i := range results {
		if results[i].ParentID == -1 {
			root = &results[i]
		} else {
			children = append(children, results[i])
		}
	}

	if root == nil {
		t.Fatal("no root thread found in results")
	}

	// Root must report at least 1 child (the mitosis spawn)
	if len(root.Children) < 1 {
		t.Fatalf("root thread should have at least 1 child from MITOSIS, got %d", len(root.Children))
	}

	t.Logf("Root: threadID=%d, children=%v, result.BoolVal=%v",
		root.ThreadID, root.Children, root.Result.BoolVal)

	// Root should have BoolVal=true (parent path)
	if !root.Result.BoolVal {
		t.Errorf("parent thread should have BoolVal=true (parent path), got %v", root.Result.BoolVal)
	}

	// Verify child exists and computed correctly
	if len(children) == 0 {
		t.Fatal("no child threads found")
	}

	child := children[0]
	t.Logf("Child: threadID=%d, parentID=%d, result.IntVal=%d",
		child.ThreadID, child.ParentID, child.Result.IntVal)

	if child.ParentID != root.ThreadID {
		t.Errorf("child parentID should be %d (root), got %d", root.ThreadID, child.ParentID)
	}

	if child.Result.Error != nil {
		t.Errorf("child thread had error: %v", child.Result.Error)
	}

	if child.Result.IntVal != 42 {
		t.Errorf("child result: expected 42, got %d", child.Result.IntVal)
	}

	// Verify a GPU fallback warning was emitted (since no GPU is available)
	warnings := m.FallbackWarnings()
	if len(warnings) == 0 {
		t.Error("expected at least one GPU fallback warning since no GPU is available")
	} else {
		// Check that the warning mentions the GPU failure reason
		found := false
		for _, w := range warnings {
			if strings.Contains(w, "GPU") && strings.Contains(w, "falling back to CPU") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning about GPU fallback, got: %v", warnings)
		}
	}
}

// TestCPUOpMitosisSpawnsGoroutines verifies that OP_MITOSIS on the CPU
// dispatcher path actually spawns goroutines, not just returns a spawn
// request that gets ignored.
func TestCPUOpMitosisSpawnsGoroutines(t *testing.T) {
	// Count goroutines before execution
	before := runtime.NumGoroutine()

	// Bytecode: push offset(1), MITOSIS, HALT(parent), push 99, HALT(child)
	constants := []interface{}{1, 99}
	var code []byte
	code = append(code, pushConst(0)...) // push 1 (offset)
	code = append(code, OpMitosisByte)   // MITOSIS
	code = append(code, 0xFF)            // HALT (parent)
	code = append(code, pushConst(1)...) // push 99 (child)
	code = append(code, 0xFF)            // HALT (child)

	bytecode := buildBytecode(constants, code)

	m := NewMitosisVM(256)
	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}

	// Children must have been spawned — we got more than 1 result
	if len(results) < 2 {
		t.Fatalf("expected child goroutines to be spawned (at least 2 results), got %d", len(results))
	}

	// After execution, goroutines should have settled back down
	// (children are joined via WaitGroup). Just verify we got results.
	after := runtime.NumGoroutine()

	t.Logf("Goroutines before=%d after=%d results=%d", before, after, len(results))

	// The child result must be correct, proving the goroutine actually ran
	var child *ThreadResult
	for i := range results {
		if results[i].ParentID != -1 {
			child = &results[i]
			break
		}
	}
	if child == nil {
		t.Fatal("no child thread found — goroutine was not spawned")
	}
	if child.Result.IntVal != 99 {
		t.Errorf("child goroutine result: expected 99, got %d", child.Result.IntVal)
	}
}
