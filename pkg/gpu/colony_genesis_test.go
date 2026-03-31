//go:build gpu
package gpu

import (
	"fmt"
	"os"
	"testing"
)

func TestColonyGenesis(t *testing.T) {
	if os.Getenv("GLYPH_GPU_TEST") == "" {
		t.Skip("Skipping GPU test. Set GLYPH_GPU_TEST=1 to run.")
	}

	// Constants: [0, 4096]
	constants := []interface{}{0, 4096}
	var code []byte
	
	// Start:
	// PUSH 0
	// S (Mitosis)
	// S (Mitosis)
	// S (Mitosis)
	// S (Mitosis)
	// HALT
	
	for i := 0; i < 12; i++ {
		code = append(code, pushConst(0)...)
		code = append(code, OpMitosis)
	}
	code = append(code, 0xFF)

	bytecode := buildBytecode(constants, code)
	
	results, err := ExecuteMultiWGSL(bytecode, 1, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Printf("Exponential Swarm Complete. Total VMs: %d\n", len(results))
}
