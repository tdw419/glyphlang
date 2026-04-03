package ssa_test

import (
	"strings"
	"testing"

	"github.com/glyphlang/glyph/pkg/ssa"
)

// testWGSLArithOp builds a minimal SSA function with a single arithmetic op
// and returns the generated WGSL source.
func testWGSLArithOp(name string, op ssa.Op, typ ssa.Type, auxInts []int64) string {
	f := ssa.NewFunc(name, ssa.SourceRoute)
	entry := f.Entry

	// Create constant operands
	vars := make([]*ssa.Value, len(auxInts))
	for i, val := range auxInts {
		v := f.NewValue(entry, ssa.OpConst, typ)
		v.AuxInt = val
		vars[i] = v
	}

	// Create the arithmetic op
	result := f.NewValue(entry, op, typ, vars...)

	// Return the result
	f.NewValue(entry, ssa.OpReturn, ssa.TypeVoid, result)

	lowering := ssa.NewWGSLLowering()
	wgsl, err := lowering.LowerFunc(f)
	if err != nil {
		return ""
	}
	return wgsl
}

func TestWGSLAddInt(t *testing.T) {
	wgsl := testWGSLArithOp("add_int", ssa.OpAddInt, ssa.TypeInt, []int64{10, 3})
	if !strings.Contains(wgsl, "+") {
		t.Error("WGSL missing addition operator '+'")
	}
	if !strings.Contains(wgsl, "i32") {
		t.Error("WGSL missing i32 type annotation for int add")
	}
}

func TestWGSLSubInt(t *testing.T) {
	wgsl := testWGSLArithOp("sub_int", ssa.OpSubInt, ssa.TypeInt, []int64{10, 3})
	if !strings.Contains(wgsl, "-") {
		t.Error("WGSL missing subtraction operator '-'")
	}
	if !strings.Contains(wgsl, "i32") {
		t.Error("WGSL missing i32 type annotation for int sub")
	}
}

func TestWGSLMulInt(t *testing.T) {
	wgsl := testWGSLArithOp("mul_int", ssa.OpMulInt, ssa.TypeInt, []int64{10, 3})
	if !strings.Contains(wgsl, "*") {
		t.Error("WGSL missing multiplication operator '*'")
	}
	if !strings.Contains(wgsl, "i32") {
		t.Error("WGSL missing i32 type annotation for int mul")
	}
}

func TestWGSLDivInt(t *testing.T) {
	wgsl := testWGSLArithOp("div_int", ssa.OpDivInt, ssa.TypeInt, []int64{10, 3})
	if !strings.Contains(wgsl, "/") {
		t.Error("WGSL missing division operator '/'")
	}
	if !strings.Contains(wgsl, "select(") {
		t.Error("WGSL missing division-by-zero guard (select)")
	}
	if !strings.Contains(wgsl, "i32") {
		t.Error("WGSL missing i32 type annotation for int div")
	}
}

func TestWGSLDivIntByZero(t *testing.T) {
	// Division by zero should emit select guard
	wgsl := testWGSLArithOp("div_zero", ssa.OpDivInt, ssa.TypeInt, []int64{10, 0})
	if !strings.Contains(wgsl, "select(") {
		t.Error("WGSL missing select guard for div-by-zero case")
	}
	if !strings.Contains(wgsl, "0i") {
		t.Error("WGSL missing zero fallback value in select guard")
	}
}

func TestWGSLModInt(t *testing.T) {
	wgsl := testWGSLArithOp("mod_int", ssa.OpModInt, ssa.TypeInt, []int64{10, 3})
	if !strings.Contains(wgsl, "%") {
		t.Error("WGSL missing modulo operator '%%'")
	}
	if !strings.Contains(wgsl, "select(") {
		t.Error("WGSL missing mod-by-zero guard (select)")
	}
}

func TestWGSLAddFloat(t *testing.T) {
	wgsl := testWGSLArithOp("add_float", ssa.OpAddFloat, ssa.TypeFloat, []int64{0, 0})
	if !strings.Contains(wgsl, "+") {
		t.Error("WGSL missing float addition operator '+'")
	}
	if !strings.Contains(wgsl, "f32") {
		t.Error("WGSL missing f32 type annotation for float add")
	}
}

func TestWGSLSubFloat(t *testing.T) {
	wgsl := testWGSLArithOp("sub_float", ssa.OpSubFloat, ssa.TypeFloat, []int64{0, 0})
	if !strings.Contains(wgsl, "-") {
		t.Error("WGSL missing float subtraction operator '-'")
	}
	if !strings.Contains(wgsl, "f32") {
		t.Error("WGSL missing f32 type annotation for float sub")
	}
}

func TestWGSLMulFloat(t *testing.T) {
	wgsl := testWGSLArithOp("mul_float", ssa.OpMulFloat, ssa.TypeFloat, []int64{0, 0})
	if !strings.Contains(wgsl, "*") {
		t.Error("WGSL missing float multiplication operator '*'")
	}
	if !strings.Contains(wgsl, "f32") {
		t.Error("WGSL missing f32 type annotation for float mul")
	}
}

func TestWGSLDivFloat(t *testing.T) {
	wgsl := testWGSLArithOp("div_float", ssa.OpDivFloat, ssa.TypeFloat, []int64{0, 0})
	if !strings.Contains(wgsl, "/") {
		t.Error("WGSL missing float division operator '/'")
	}
	if !strings.Contains(wgsl, "f32") {
		t.Error("WGSL missing f32 type annotation for float div")
	}
	// Float division by zero is well-defined in WGSL (returns inf/nan), no select guard needed
	if strings.Contains(wgsl, "select(") {
		t.Error("WGSL should NOT have select guard for float division")
	}
}

func TestWGSLLoweringAllOps(t *testing.T) {
	// Test that the WGSL output is a valid shader structure
	ops := []struct {
		name string
		op   ssa.Op
		typ  ssa.Type
	}{
		{"add_i", ssa.OpAddInt, ssa.TypeInt},
		{"sub_i", ssa.OpSubInt, ssa.TypeInt},
		{"mul_i", ssa.OpMulInt, ssa.TypeInt},
		{"div_i", ssa.OpDivInt, ssa.TypeInt},
		{"mod_i", ssa.OpModInt, ssa.TypeInt},
		{"add_f", ssa.OpAddFloat, ssa.TypeFloat},
		{"sub_f", ssa.OpSubFloat, ssa.TypeFloat},
		{"mul_f", ssa.OpMulFloat, ssa.TypeFloat},
		{"div_f", ssa.OpDivFloat, ssa.TypeFloat},
	}

	for _, tc := range ops {
		t.Run(tc.name, func(t *testing.T) {
			wgsl := testWGSLArithOp(tc.name, tc.op, tc.typ, []int64{5, 2})
			if !strings.Contains(wgsl, "@compute") {
				t.Errorf("Missing @compute entry point for %s", tc.name)
			}
			if !strings.Contains(wgsl, "states[id]") {
				t.Errorf("Missing state write for %s", tc.name)
			}
		})
	}
}
