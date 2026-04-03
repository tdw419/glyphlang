package gpu

import (
	"testing"
)

// TestSpawnOffsetGreaterThanZero verifies SEC-2.1: spawn offset > 0 for compiled
// programs. When MITOSIS pops a positive offset from the stack, the child thread
// must start at parent.PC + 1 + offset, not at parent.PC + 1 (offset=0 behavior).
//
// This test uses offset=10 so the child lands well past the parent's HALT into a
// distinct code region that returns a different value (777) than the parent would
// if offset were ignored.
func TestSpawnOffsetGreaterThanZero(t *testing.T) {
	// Bytecode layout:
	//   0-4:  PUSH_CONST[0]  (value=10 — the spawn offset)
	//   5:    MITOSIS (0xC0)
	//   6:    HALT            ← parent halts (BoolVal=true)
	//   7:    0x00            ← padding (skipped by child)
	//   8:    0x00            ← padding
	//   9:    0x00            ← padding
	//  10:    0x00            ← padding
	//  11-15: PUSH_CONST[1]  (value=777 — child result)
	//  16:    HALT            ← child halts with 777
	//
	// The offset is 10. The MITOSIS opcode is at PC=5.
	// Child PC = 5 + 1 + 10 = 16... that's right at HALT, not what we want.
	//
	// Let me recalculate: child PC = MITOSIS_PC + 1 + offset
	// MITOSIS is at index 5 in the code section.
	// child PC = 5 + 1 + offset
	// We want child to land at the PUSH_CONST[1] at code index 7.
	// So offset = 7 - 5 - 1 = 1. But we want offset > 1 to truly test > 0.
	//
	// Better layout with offset=7:
	//   0-4:  PUSH_CONST[0]  (offset=7)
	//   5:    MITOSIS
	//   6:    HALT (parent)
	//   7-11: NOP padding (child should NOT land here)
	//   12-13: more padding
	//   13:    ... child should land at 5+1+7 = 13
	//
	// Actually, let's keep it simple with a moderate offset and verify:
	// offset=7 → child PC = 5+1+7 = 13

	const spawnOffset = 7
	const childValue = 777

	constants := []interface{}{spawnOffset, childValue}
	var code []byte

	// 0-4: push spawnOffset
	code = append(code, pushConst(0)...)
	// 5: MITOSIS
	code = append(code, OpMitosisByte)
	// 6: HALT (parent path)
	code = append(code, 0xFF)

	// 7-12: padding — child should NOT execute this
	code = append(code, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)

	// 13-17: push childValue — child should land here (PC = 5+1+7 = 13)
	code = append(code, pushConst(1)...)
	// 18: HALT (child)
	code = append(code, 0xFF)

	bytecode := buildBytecode(constants, code)

	m := NewMitosisVM(256)
	results, err := m.ExecuteWithMitosis(bytecode)
	if err != nil {
		t.Fatalf("ExecuteWithMitosis error: %v", err)
	}

	// Must have 2 results: parent + child
	if len(results) < 2 {
		t.Fatalf("expected at least 2 thread results, got %d", len(results))
	}

	// Find parent (ParentID == -1) and child (ParentID != -1)
	var parent *ThreadResult
	var child *ThreadResult
	for i := range results {
		if results[i].ParentID == -1 {
			parent = &results[i]
		} else {
			child = &results[i]
		}
	}

	if parent == nil {
		t.Fatal("no parent thread found")
	}
	if child == nil {
		t.Fatal("no child thread found — spawn offset may not have produced a child")
	}

	// Parent should have BoolVal=true (parent path of mitosis)
	if !parent.Result.BoolVal {
		t.Errorf("parent should have BoolVal=true, got false")
	}

	// The critical assertion: child must have computed childValue (777).
	// If offset were 0 or ignored, the child would land at PC=6 (the HALT)
	// or PC=7 (padding) and would NOT produce 777.
	if child.Result.IntVal != childValue {
		t.Errorf("child result: expected %d (proving offset=%d was applied), got %d",
			childValue, spawnOffset, child.Result.IntVal)
	}

	t.Logf("parent: threadID=%d boolVal=%v", parent.ThreadID, parent.Result.BoolVal)
	t.Logf("child:  threadID=%d parentID=%d intVal=%d (offset=%d correctly applied)",
		child.ThreadID, child.ParentID, child.Result.IntVal, spawnOffset)
}
