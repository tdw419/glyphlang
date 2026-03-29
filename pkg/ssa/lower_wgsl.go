package ssa

import (
	"fmt"
	"strings"
)

// WGSLLowering lowers an SSA function directly to a WGSL compute shader.
// This bypasses the bytecode interpreter for maximum performance.
type WGSLLowering struct {
	indent int
}

// NewWGSLLowering creates a new WGSL lowering context.
func NewWGSLLowering() *WGSLLowering {
	return &WGSLLowering{}
}

// LowerFunc generates a complete WGSL compute shader for the given function.
func (l *WGSLLowering) LowerFunc(f *Func) (string, error) {
	var b strings.Builder

	// 1. Headers and Structs
	b.WriteString("// Auto-generated WGSL shader from GlyphLang SSA\n")
	b.WriteString("struct VMState {\n")
	b.WriteString("    pc: u32,\n")
	b.WriteString("    sp: u32,\n")
	b.WriteString("    halted: u32,\n")
	b.WriteString("    error: u32,\n")
	b.WriteString("    steps: u32,\n")
	b.WriteString("    result_tag: u32,\n")
	b.WriteString("    result_data: i32,\n")
	b.WriteString("}\n\n")

	// 2. Bindings
	b.WriteString("@group(0) @binding(0) var<storage, read_write> states: array<VMState>;\n")
	b.WriteString("@group(0) @binding(1) var<storage, read_write> locals: array<i32>; // Simple flat memory\n\n")

	// 3. Entry Point
	b.WriteString("@compute @workgroup_size(64)\n")
	b.WriteString("fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {\n")
	l.indent++
	
	b.WriteString(l.line("let id = global_id.x;"))
	b.WriteString(l.line("var block_id: u32 = 0; // entry block"))
	b.WriteString(l.line("var halted: bool = false;"))
	b.WriteString("\n")

	// 4. Declare all SSA values as variables
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			if v.Type != TypeVoid {
				b.WriteString(l.line(fmt.Sprintf("var v%d: %s;", v.ID, l.wgslType(v.Type))))
			}
		}
	}
	b.WriteString("\n")

	// 5. Main Execution Loop
	b.WriteString(l.line("while (!halted) {"))
	l.indent++
	b.WriteString(l.line("switch (block_id) {"))
	l.indent++

	for _, blk := range f.Blocks {
		b.WriteString(l.line(fmt.Sprintf("case %d: { // %s", blk.ID, blk.Name)))
		l.indent++

		for _, v := range blk.Values {
			if err := l.emitValue(&b, v); err != nil {
				return "", err
			}
		}

		l.indent--
		b.WriteString(l.line("}"))
	}

	l.indent--
	b.WriteString(l.line("}"))
	l.indent--
	b.WriteString(l.line("}"))

	l.indent--
	b.WriteString("}\n")

	return b.String(), nil
}

func (l *WGSLLowering) emitValue(b *strings.Builder, v *Value) error {
	switch v.Op {
	case OpConst:
		switch v.Type {
		case TypeInt:
			b.WriteString(l.line(fmt.Sprintf("v%d = %di;", v.ID, v.AuxInt)))
		case TypeFloat:
			b.WriteString(l.line(fmt.Sprintf("v%d = %f;", v.ID, float64(v.AuxInt)))) // Simplified
		case TypeBool:
			val := "false"
			if v.AuxInt != 0 {
				val = "true"
			}
			b.WriteString(l.line(fmt.Sprintf("v%d = %s;", v.ID, val)))
		}

	case OpAddInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d + v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))
	case OpSubInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d - v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))
	case OpMulInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d * v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))
	case OpDivInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d / v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))

	case OpEqInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d == v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))
	case OpLtInt:
		b.WriteString(l.line(fmt.Sprintf("v%d = v%d < v%d;", v.ID, v.Args[0].ID, v.Args[1].ID)))

	case OpLoadVar:
		// Map variable names to numeric indices for simplicity in this MVP
		// In production, we'd have a proper symbol table mapping
		idx := 0 // placeholder
		b.WriteString(l.line(fmt.Sprintf("v%d = locals[id * 64 + %d];", v.ID, idx)))

	case OpStoreVar:
		idx := 0 // placeholder
		b.WriteString(l.line(fmt.Sprintf("locals[id * 64 + %d] = v%d;", idx, v.Args[0].ID)))

	case OpIf:
		b.WriteString(l.line(fmt.Sprintf("if (v%d) { block_id = %d; } else { block_id = %d; }", v.Args[0].ID, v.Block.Succs[0].ID, v.Block.Succs[1].ID)))
	case OpJump:
		b.WriteString(l.line(fmt.Sprintf("block_id = %d;", v.Block.Succs[0].ID)))
	case OpReturn, OpHalt:
		b.WriteString(l.line("halted = true;"))
		if len(v.Args) > 0 {
			b.WriteString(l.line(fmt.Sprintf("states[id].result_tag = %d;", l.tagForType(v.Args[0].Type))))
			b.WriteString(l.line(fmt.Sprintf("states[id].result_data = i32(v%d);", v.Args[0].ID)))
		}

	case OpPhi:
		// Phis are handled by the predecessor blocks in a structured lowering,
		// but for this switch-based approach, we need to carefully assign them.
		// Simplified: we assume phis are already lowered or handled by variables.
	}

	return nil
}

func (l *WGSLLowering) wgslType(t Type) string {
	switch t {
	case TypeInt:
		return "i32"
	case TypeFloat:
		return "f32"
	case TypeBool:
		return "bool"
	default:
		return "i32"
	}
}

func (l *WGSLLowering) tagForType(t Type) uint32 {
	switch t {
	case TypeInt:
		return 1
	case TypeFloat:
		return 2
	case TypeBool:
		return 3
	default:
		return 0
	}
}

func (l *WGSLLowering) line(s string) string {
	return strings.Repeat("    ", l.indent) + s + "\n"
}
