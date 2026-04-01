package gpu

import (
	"testing"
)

func TestMitosisBasic(t *testing.T) {
	// Simple program without S opcode — should run as single thread
	constants := []interface{}{10, 5}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x10) // ADD
	code = append(code, 0xFF) // HALT

	m := NewMitosisVM(256)
	results, err := m.ExecuteWithMitosis(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 thread result, got %d", len(results))
	}
	if results[0].ParentID != -1 {
		t.Fatal("root thread should have parentID -1")
	}
	if results[0].Result.IntVal != 15 {
		t.Fatalf("expected 15, got %d", results[0].Result.IntVal)
	}
}

func TestMitosisSpawn(t *testing.T) {
	// Program that spawns a child thread via S opcode:
	// push 1 (offset), S (spawn child at PC+1+1 = 7), halt
	// Child starts at: offset 7 — push 99, halt
	//
	// Layout:
	// 0-4:  push const 0 (offset=1)
	// 5:    S opcode (0xC0)
	// 6:    HALT (parent halts with child ID on stack)
	// 7-11: push const 1 (value=99) — child code
	// 12:   HALT (child halts with 99)

	constants := []interface{}{1, 99}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 1 (offset)
	code = append(code, OpMitosisByte)       // 5: S opcode
	code = append(code, 0xFF)            // 6: HALT (parent)
	code = append(code, pushConst(1)...) // 7-11: push 99
	code = append(code, 0xFF)            // 12: HALT (child)

	m := NewMitosisVM(256)
	results, err := m.ExecuteWithMitosis(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(results) < 1 {
		t.Fatal("expected at least 1 thread result")
	}

	// Find root thread
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
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}

	t.Logf("Root thread: ID=%d, result=%d, children=%v",
		root.ThreadID, root.Result.IntVal, root.Children)

	// Check child exists
	for _, r := range results {
		if r.ParentID == root.ThreadID {
			t.Logf("Child thread: ID=%d, result=%d", r.ThreadID, r.Result.IntVal)
		}
	}
}

func TestMitosisMaxThreads(t *testing.T) {
	// Limit to 2 threads — only 1 spawn should succeed
	constants := []interface{}{5, 5}
	var code []byte
	code = append(code, pushConst(0)...) // push offset
	code = append(code, OpMitosisByte)       // spawn 1
	code = append(code, pushConst(1)...) // push offset
	code = append(code, OpMitosisByte)       // spawn 2 (should be ignored due to limit)
	code = append(code, 0xFF)            // HALT

	m := NewMitosisVM(2) // max 2 threads total
	results, err := m.ExecuteWithMitosis(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	t.Logf("Total threads: %d", len(results))
	for _, r := range results {
		t.Logf("  Thread %d (parent=%d): result=%d, children=%v",
			r.ThreadID, r.ParentID, r.Result.IntVal, r.Children)
	}
}

func TestNewMitosisVM(t *testing.T) {
	m := NewMitosisVM(0)
	if m.maxThreads != 256 {
		t.Fatalf("expected default maxThreads=256, got %d", m.maxThreads)
	}

	m = NewMitosisVM(100)
	if m.maxThreads != 100 {
		t.Fatalf("expected maxThreads=100, got %d", m.maxThreads)
	}
}
