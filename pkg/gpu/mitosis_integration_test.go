package gpu

import (
	"testing"
)

// TestMitosisIntegration4Children verifies that 4 Mitosis children each compute
// result = pid * 2 correctly after spawning and synchronization.
//
// This test validates the full Mitosis spawn pipeline:
//   - Parent spawns 4 children, each receiving a different pid value
//   - Each child executes: result = pid * 2
//   - All 4 children must complete and return correct values
//
// Bytecode layout (parent code + child code):
//
//	0-4:   push const 0 (pid=0)
//	5-9:   push const 4 (offset=38)
//	10:    MITOSIS → child at 10+1+38 = 49
//	11:    POP (discard child thread ID from parent stack)
//	12-16: push const 1 (pid=1)
//	17-21: push const 5 (offset=26)
//	22:    MITOSIS → child at 22+1+26 = 49
//	23:    POP
//	24-28: push const 2 (pid=2)
//	29-33: push const 6 (offset=14)
//	34:    MITOSIS → child at 34+1+14 = 49
//	35:    POP
//	36-40: push const 3 (pid=3)
//	41-45: push const 2 (offset=2 — reuse const[2])
//	46:    MITOSIS → child at 46+1+2 = 49
//	47:    POP
//	48:    HALT (parent stops before child code)
//	--- child code at offset 49 ---
//	49-53: push const 2 (multiplier=2)
//	54:    MUL    → pops multiplier + pid, pushes pid*2
//	55:    HALT
//
// Each child inherits a stack with only [pid] on it.
// Expected child results: 0*2=0, 1*2=2, 2*2=4, 3*2=6
func TestMitosisIntegration4Children(t *testing.T) {
	// Constants: pid0=0, pid1=1, pid2=2 (also multiplier), pid3=3,
	// offset0=38, offset1=26, offset2=14
	constants := []interface{}{0, 1, 2, 3, 38, 26, 14}

	var code []byte

	// Spawn 0: push pid(0), push offset(38), MITOSIS, POP
	code = append(code, pushConst(0)...) // 0-4: push 0 (pid)
	code = append(code, pushConst(4)...) // 5-9: push 38 (offset)
	code = append(code, OpMitosisByte)       // 10: MITOSIS → child at 49
	code = append(code, 0x02)            // 11: POP (discard child ID)

	// Spawn 1: push pid(1), push offset(26), MITOSIS, POP
	code = append(code, pushConst(1)...) // 12-16: push 1 (pid)
	code = append(code, pushConst(5)...) // 17-21: push 26 (offset)
	code = append(code, OpMitosisByte)       // 22: MITOSIS → child at 49
	code = append(code, 0x02)            // 23: POP

	// Spawn 2: push pid(2), push offset(14), MITOSIS, POP
	code = append(code, pushConst(2)...) // 24-28: push 2 (pid)
	code = append(code, pushConst(6)...) // 29-33: push 14 (offset)
	code = append(code, OpMitosisByte)       // 34: MITOSIS → child at 49
	code = append(code, 0x02)            // 35: POP

	// Spawn 3: push pid(3), push offset(2), MITOSIS, POP
	code = append(code, pushConst(3)...) // 36-40: push 3 (pid)
	code = append(code, pushConst(2)...) // 41-45: push 2 (offset — reuse const[2])
	code = append(code, OpMitosisByte)       // 46: MITOSIS → child at 49
	code = append(code, 0x02)            // 47: POP

	// Parent HALT
	code = append(code, 0xFF) // 48: HALT

	// Child code (all children land here at offset 49)
	// Stack has [pid] from parent. Push multiplier, MUL, HALT.
	code = append(code, pushConst(2)...) // 49-53: push 2 (multiplier — reuse const[2])
	code = append(code, 0x12)            // 54: MUL → pid * 2
	code = append(code, 0xFF)            // 55: HALT

	m := NewMitosisVM(256)
	results, err := m.ExecuteWithMitosis(buildBytecode(constants, code))
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}

	// Expect 5 results: 1 root + 4 children
	if len(results) < 5 {
		t.Fatalf("expected 5 thread results (1 parent + 4 children), got %d", len(results))
	}

	// Separate root from children
	var root *ThreadResult
	childrenByID := make(map[int]*ThreadResult)
	for i := range results {
		r := &results[i]
		if r.ParentID == -1 {
			root = r
		} else {
			childrenByID[r.ThreadID] = r
		}
	}

	if root == nil {
		t.Fatal("no root thread found")
	}

	if len(root.Children) != 4 {
		t.Fatalf("expected root to have 4 children, got %d", len(root.Children))
	}

	t.Logf("Root: threadID=%d, result=%d, children=%v",
		root.ThreadID, root.Result.IntVal, root.Children)

	// Verify each child computed pid * 2 correctly.
	// Children are spawned in order with pid values 0, 1, 2, 3.
	expectedResults := []int64{0, 2, 4, 6}

	for i, childID := range root.Children {
		child, ok := childrenByID[childID]
		if !ok {
			t.Fatalf("child thread %d (spawn index %d) not found in results", childID, i)
		}

		pid := i
		expected := expectedResults[pid]

		t.Logf("Child %d: threadID=%d, pid=%d, result=%d, expected=%d",
			i, child.ThreadID, pid, child.Result.IntVal, expected)

		if child.Result.Error != nil {
			t.Errorf("child %d (pid=%d) had error: %v", i, pid, child.Result.Error)
		}
		if child.Result.IntVal != expected {
			t.Errorf("child %d (pid=%d): expected %d (pid*2), got %d",
				i, pid, expected, child.Result.IntVal)
		}
	}
}
