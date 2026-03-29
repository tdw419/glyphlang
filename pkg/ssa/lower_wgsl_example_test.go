// Example test demonstrating WGSL lowering from GlyphLang SSA
package ssa_test

import (
	"fmt"
	"testing"

	"github.com/glyphlang/glyph/pkg/ssa"
)

func TestWGSLLoweringExample(t *testing.T) {
	// Create a simple SSA function: add two numbers
	f := ssa.NewFunc("add_example", ssa.SourceRoute)

	// Entry block
	entry := f.Entry

	// v0 = Const 40
	v0 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v0.AuxInt = 40

	// v1 = Const 2
	v1 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v1.AuxInt = 2

	// v2 = AddInt v0 v1
	v2 := f.NewValue(entry, ssa.OpAddInt, ssa.TypeInt, v0, v1)

	// v3 = Return v2
	f.NewValue(entry, ssa.OpReturn, ssa.TypeVoid, v2)

	// Run optimizations
	passMgr := ssa.DefaultPasses()
	passMgr.Run(f)

	// Lower to WGSL
	lowering := ssa.NewWGSLLowering()
	wgsl, err := lowering.LowerFunc(f)
	if err != nil {
		t.Fatalf("WGSL lowering failed: %v", err)
	}

	fmt.Println("=== Generated WGSL ===")
	fmt.Println(wgsl)
	fmt.Println("======================")

	// Verify key elements
	if !contains(wgsl, "@compute") {
		t.Error("Missing @compute entry point")
	}
	if !contains(wgsl, "vm_stats") {
		t.Error("Missing vm_stats telemetry binding")
	}
	if !contains(wgsl, "v2: i32") {
		t.Error("Missing result variable declaration")
	}
}

func TestWGSLLoweringWithConditionals(t *testing.T) {
	// Create: if (x > 10) { return 1 } else { return 0 }
	f := ssa.NewFunc("conditional_example", ssa.SourceRoute)
	entry := f.Entry

	// v0 = Const 15 (x)
	v0 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v0.AuxInt = 15

	// v1 = Const 10
	v1 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v1.AuxInt = 10

	// v2 = GtInt v0 v1
	v2 := f.NewValue(entry, ssa.OpGtInt, ssa.TypeBool, v0, v1)

	// True block
	trueBlock := f.NewBlock("true")
	// v3 = Const 1
	v3 := f.NewValue(trueBlock, ssa.OpConst, ssa.TypeInt)
	v3.AuxInt = 1
	// Return v3
	f.NewValue(trueBlock, ssa.OpReturn, ssa.TypeVoid, v3)

	// False block
	falseBlock := f.NewBlock("false")
	// v4 = Const 0
	v4 := f.NewValue(falseBlock, ssa.OpConst, ssa.TypeInt)
	v4.AuxInt = 0
	// Return v4
	f.NewValue(falseBlock, ssa.OpReturn, ssa.TypeVoid, v4)

	// If v2 then trueBlock else falseBlock
	ifVal := f.NewValue(entry, ssa.OpIf, ssa.TypeVoid, v2)
	ifVal.Block.Succs = []*ssa.Block{trueBlock, falseBlock}
	trueBlock.Preds = []*ssa.Block{entry}
	falseBlock.Preds = []*ssa.Block{entry}

	// Lower to WGSL
	lowering := ssa.NewWGSLLowering()
	wgsl, err := lowering.LowerFunc(f)
	if err != nil {
		t.Fatalf("WGSL lowering failed: %v", err)
	}

	fmt.Println("=== Conditional WGSL ===")
	fmt.Println(wgsl)
	fmt.Println("=========================")

	// Verify conditional generated
	if !contains(wgsl, "if (") {
		t.Error("Missing if statement")
	}
	if !contains(wgsl, "block_id") {
		t.Error("Missing block dispatch")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
