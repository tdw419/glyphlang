package gpu

import (
	"fmt"
	"testing"
)

func TestCPUMitosis(t *testing.T) {
	t.Skip("TODO: Mitosis spawn semantics need fixing (Issue #89)")
	// Program: 
	// 0-4:  push 5 (offset to child code)
	// 5:    S opcode (spawn child at PC+1+5 = 11)
	// 6:    HALT (parent)
	// 7-10: NOP/padding
	// 11-15: push 42 (child result)
	// 16:   HALT (child)

	constants := []interface{}{5, 42}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 5
	code = append(code, OpMitosisByte)       // 5: S opcode
	code = append(code, 0xFF)            // 6: HALT (parent)
	for i := 0; i < 4; i++ { code = append(code, 0x00) } // 7-10: NOPs
	code = append(code, pushConst(1)...) // 11-15: push 42
	code = append(code, 0xFF)            // 16: HALT (child)

	bytecode := buildBytecode(constants, code)
	
	d := NewDispatcher()
	// Force CPU mode
	d.hasGPU = false
	
	results, err := d.Execute(bytecode, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	fmt.Printf("CPU results: %+v\n", results)

	if results[0].IntVal != 1 {
		t.Errorf("parent expected true, got %v", results[0].IntVal)
	}

	if results[1].IntVal != 42 {
		t.Errorf("child expected 42, got %d", results[1].IntVal)
	}
}
