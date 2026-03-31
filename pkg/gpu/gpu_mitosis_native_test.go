//go:build gpu
package gpu

import (
	"fmt"
	"os"
	"testing"
)

func TestGPUMitosisMultiPass(t *testing.T) {
	if os.Getenv("GLYPH_GPU_TEST") == "" {
		t.Skip("Skipping GPU test. Set GLYPH_GPU_TEST=1 to run.")
	}

	constants := []interface{}{9, 42}
	var code []byte
	code = append(code, pushConst(0)...) // 0-4: push 9 (offset)
	code = append(code, OpMitosis)       // 5: S opcode
	code = append(code, 0xFF)            // 6: HALT (parent)
	for i := 0; i < 8; i++ { code = append(code, 0x00) } // 7-14: NOPs
	code = append(code, pushConst(1)...) // 15-19: push 42
	code = append(code, 0xFF)            // 20: HALT (child)

	bytecode := buildBytecode(constants, code)
	
	results, err := ExecuteMultiWGSL(bytecode, 1, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	fmt.Printf("GPU results: %+v\n", results)

	if results[0].IntVal != 0 {
		t.Errorf("parent expected 0, got %d", results[0].IntVal)
	}

	if results[1].IntVal != 42 {
		t.Errorf("child expected 42, got %d", results[1].IntVal)
	}
}
