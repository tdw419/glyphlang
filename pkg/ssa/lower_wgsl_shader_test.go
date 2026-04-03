package ssa_test

import (
	"strings"
	"testing"

	"github.com/glyphlang/glyph/pkg/ssa"
)

// TestWGSLLowering_OnePlusTwo verifies that SSA IR for the expression `1+2`
// produces valid WGSL with correct structure and arithmetic.
func TestWGSLLowering_OnePlusTwo(t *testing.T) {
	// Build SSA for: return 1 + 2
	f := ssa.NewFunc("one_plus_two", ssa.SourceRoute)
	entry := f.Entry

	// v0 = Const 1
	v0 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v0.AuxInt = 1

	// v1 = Const 2
	v1 := f.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v1.AuxInt = 2

	// v2 = AddInt v0, v1
	v2 := f.NewValue(entry, ssa.OpAddInt, ssa.TypeInt, v0, v1)

	// v3 = Return v2
	f.NewValue(entry, ssa.OpReturn, ssa.TypeVoid, v2)

	lowering := ssa.NewWGSLLowering()
	wgsl, err := lowering.LowerFunc(f)
	if err != nil {
		t.Fatalf("LowerFunc failed: %v", err)
	}

	// Verify WGSL structural elements
	if !strings.Contains(wgsl, "@compute") {
		t.Error("WGSL missing @compute entry point")
	}
	if !strings.Contains(wgsl, "@workgroup_size") {
		t.Error("WGSL missing @workgroup_size")
	}
	if !strings.Contains(wgsl, "fn main") {
		t.Error("WGSL missing fn main entry")
	}
	if !strings.Contains(wgsl, "VMState") {
		t.Error("WGSL missing VMState struct")
	}
	if !strings.Contains(wgsl, "1i") {
		t.Error("WGSL missing constant '1i' for first operand")
	}
	if !strings.Contains(wgsl, "2i") {
		t.Error("WGSL missing constant '2i' for second operand")
	}
	if !strings.Contains(wgsl, "+") {
		t.Error("WGSL missing addition operator")
	}
	if !strings.Contains(wgsl, "i32") {
		t.Error("WGSL missing i32 type")
	}
	// Verify result is written to state
	if !strings.Contains(wgsl, "states[id]") {
		t.Error("WGSL missing result write to states[id]")
	}
}

// TestWGSLLowering_AddFunction verifies that SSA IR for `fn add(a,b) { return a+b }`
// emits a valid WGSL helper function with correct parameter passing and arithmetic.
// The add function is emitted as a helper callable from a compute entry point.
func TestWGSLLowering_AddFunction(t *testing.T) {
	// Build SSA for caller: fn main() { return add(3, 4) }
	caller := ssa.NewFunc("main_route", ssa.SourceRoute)
	entry := caller.Entry

	// v0 = Const 3
	v0 := caller.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v0.AuxInt = 3

	// v1 = Const 4
	v1 := caller.NewValue(entry, ssa.OpConst, ssa.TypeInt)
	v1.AuxInt = 4

	// v2 = Call "add"(v0, v1)
	v2 := caller.NewValue(entry, ssa.OpCall, ssa.TypeInt, v0, v1)
	v2.AuxStr = "add"

	// v3 = Return v2
	caller.NewValue(entry, ssa.OpReturn, ssa.TypeVoid, v2)

	// Build SSA for helper: fn add(a, b) { return a + b }
	addFn := ssa.NewFunc("add", ssa.SourceFunction)
	addFn.Params = []string{"a", "b"}
	addEntry := addFn.Entry

	// v0 = LoadVar "a"
	va := addFn.NewValue(addEntry, ssa.OpLoadVar, ssa.TypeInt)
	va.AuxStr = "a"

	// v1 = LoadVar "b"
	vb := addFn.NewValue(addEntry, ssa.OpLoadVar, ssa.TypeInt)
	vb.AuxStr = "b"

	// v2 = AddInt va, vb
	vsum := addFn.NewValue(addEntry, ssa.OpAddInt, ssa.TypeInt, va, vb)

	// v3 = Return vsum
	addFn.NewValue(addEntry, ssa.OpReturn, ssa.TypeVoid, vsum)

	// Emit the caller as a compute shader with add as a helper
	lowering := ssa.NewWGSLLowering()
	helpers := map[string]*ssa.Func{"add": addFn}
	wgsl, err := lowering.EmitComputeFunc(caller, helpers)
	if err != nil {
		t.Fatalf("EmitComputeFunc failed: %v", err)
	}

	// Verify helper function signature: fn add(a: i32, b: i32) -> i32
	if !strings.Contains(wgsl, "fn add(a: i32, b: i32)") {
		t.Errorf("WGSL missing helper function signature 'fn add(a: i32, b: i32)'\nGot:\n%s", wgsl)
	}
	if !strings.Contains(wgsl, "-> i32") {
		t.Error("WGSL missing return type '-> i32' on helper function")
	}
	// Verify addition in helper body
	if !strings.Contains(wgsl, "+") {
		t.Error("WGSL missing addition operator in add function body")
	}
	// Verify params loaded from function arguments
	if !strings.Contains(wgsl, "= a;") {
		t.Error("WGSL missing param 'a' load (expected '= a;')")
	}
	if !strings.Contains(wgsl, "= b;") {
		t.Error("WGSL missing param 'b' load (expected '= b;')")
	}
	// Verify return statement in helper
	if !strings.Contains(wgsl, "return v") {
		t.Error("WGSL missing return statement in add helper")
	}
	// Verify compute entry point still present
	if !strings.Contains(wgsl, "@compute") {
		t.Error("WGSL missing @compute entry point")
	}
	if !strings.Contains(wgsl, "VMState") {
		t.Error("WGSL missing VMState struct")
	}
	// Verify the caller invokes the helper
	if !strings.Contains(wgsl, "add(") {
		t.Error("WGSL missing call to add() helper from entry point")
	}
}
