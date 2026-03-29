package ssa

import (
	"fmt"
	"strings"
)

// WGSLLowering lowers an SSA function directly to a WGSL compute shader.
// This bypasses the bytecode interpreter for maximum performance.
type WGSLLowering struct {
	indent  int
	vars    map[string]int
	nextVar int
}

// NewWGSLLowering creates a new WGSL lowering context.
func NewWGSLLowering() *WGSLLowering {
	return &WGSLLowering{
		vars: make(map[string]int),
	}
}

// LowerFunc generates a complete WGSL compute shader for the given function.
func (l *WGSLLowering) LowerFunc(f *Func) (string, error) {
	return l.LowerMultiFunc([]*Func{f})
}

// LowerMultiFunc generates a single WGSL shader that can execute multiple routes.
// Each route is dispatched based on the global invocation ID.
func (l *WGSLLowering) LowerMultiFunc(funcs []*Func) (string, error) {
	var b strings.Builder

	// 1. Headers and Structs
	b.WriteString("// Auto-generated WGSL multi-route shader from GlyphLang SSA\n")
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
	b.WriteString("@group(0) @binding(1) var<storage, read_write> locals: array<i32>; // Simple flat memory\n")
	b.WriteString("@group(0) @binding(2) var<storage, read_write> vm_stats: array<atomic<u32>, 11>; // Telemetry (ASCII World HUD)\n\n")

	// 3. Entry Point
	b.WriteString("@compute @workgroup_size(64)\n")
	b.WriteString("fn main(@builtin(global_invocation_id) global_id: vec3<u32>) {\n")
	l.indent++
	
	b.WriteString(l.line("let id = global_id.x;"))
	b.WriteString(l.line("if (id >= 11u) { return; } // Boundary check for vm_stats"))
	b.WriteString(l.line("var block_id: u32 = 0;"))
	b.WriteString(l.line("var halted: bool = false;"))
	b.WriteString("\n")

	// 4. Declare all SSA values for all functions
	// Use unique prefix per function to avoid collisions
	for i, f := range funcs {
		b.WriteString(l.line(fmt.Sprintf("// --- Variables for Route %d (%s) ---", i, f.Name)))
		for _, blk := range f.Blocks {
			for _, v := range blk.Values {
				if v.Type != TypeVoid {
					b.WriteString(l.line(fmt.Sprintf("var f%d_v%d: %s;", i, v.ID, l.wgslType(v.Type))))
				}
			}
		}
	}
	b.WriteString("\n")

	// 5. Multi-Route Dispatch Switch
	b.WriteString(l.line("switch (id) {"))
	l.indent++

	for i, f := range funcs {
		b.WriteString(l.line(fmt.Sprintf("case %du: { // Route: %s", i, f.Name)))
		l.indent++
		
		// Reset state for this route
		b.WriteString(l.line("block_id = 0u;"))
		b.WriteString(l.line("halted = false;"))
		
		b.WriteString(l.line("while (!halted) {"))
		l.indent++
		b.WriteString(l.line("switch (block_id) {"))
		l.indent++

		for _, blk := range f.Blocks {
			b.WriteString(l.line(fmt.Sprintf("case %du: { // %s", blk.ID, blk.Name)))
			l.indent++

			for _, v := range blk.Values {
				if err := l.emitValueWithPrefix(&b, v, i); err != nil {
					return "", err
				}
			}

			l.indent--
			b.WriteString(l.line("}"))
		}

		b.WriteString(l.line("default: { halted = true; }"))
		l.indent--
		b.WriteString(l.line("}"))
		l.indent--
		b.WriteString(l.line("}"))

		l.indent--
		b.WriteString(l.line("}"))
	}

	b.WriteString(l.line("default: { }"))
	l.indent--
	b.WriteString(l.line("}"))

	l.indent--
	b.WriteString("}\n")

	return b.String(), nil
}

func (l *WGSLLowering) emitValueWithPrefix(b *strings.Builder, v *Value, fnIdx int) error {
	prefix := fmt.Sprintf("f%d_", fnIdx)
	
	switch v.Op {
	case OpConst:
		switch v.Type {
		case TypeInt:
			b.WriteString(l.line(fmt.Sprintf("%sv%d = %di;", prefix, v.ID, v.AuxInt)))
		case TypeFloat:
			b.WriteString(l.line(fmt.Sprintf("%sv%d = %f;", prefix, v.ID, float64(v.AuxInt))))
		case TypeBool:
			val := "false"
			if v.AuxInt != 0 {
				val = "true"
			}
			b.WriteString(l.line(fmt.Sprintf("%sv%d = %s;", prefix, v.ID, val)))
		}

	case OpAddInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d + %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))
	case OpSubInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d - %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))
	case OpMulInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d * %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))
	case OpDivInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d / %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))

	case OpEqInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d == %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))
	case OpLtInt:
		b.WriteString(l.line(fmt.Sprintf("%sv%d = %sv%d < %sv%d;", prefix, v.ID, prefix, v.Args[0].ID, prefix, v.Args[1].ID)))

	case OpLoadVar:
		idx, ok := l.vars[v.AuxStr]
		if !ok {
			idx = l.nextVar
			l.vars[v.AuxStr] = idx
			l.nextVar++
		}
		b.WriteString(l.line(fmt.Sprintf("%sv%d = locals[id * 64u + %du];", prefix, v.ID, idx)))

	case OpStoreVar:
		idx, ok := l.vars[v.AuxStr]
		if !ok {
			idx = l.nextVar
			l.vars[v.AuxStr] = idx
			l.nextVar++
		}
		b.WriteString(l.line(fmt.Sprintf("locals[id * 64u + %du] = %sv%d;", idx, prefix, v.Args[0].ID)))

	case OpIf:
		b.WriteString(l.line(fmt.Sprintf("if (%sv%d) { block_id = %du; } else { block_id = %du; }", prefix, v.Args[0].ID, v.Block.Succs[0].ID, v.Block.Succs[1].ID)))
	case OpJump:
		b.WriteString(l.line(fmt.Sprintf("block_id = %du;", v.Block.Succs[0].ID)))
	case OpReturn, OpHalt:
		b.WriteString(l.line("halted = true;"))
		if len(v.Args) > 0 {
			// Write to both VMState and ASCII World HUD (vm_stats)
			b.WriteString(l.line(fmt.Sprintf("atomicStore(&vm_stats[id], u32(%sv%d));", prefix, v.Args[0].ID)))
			b.WriteString(l.line(fmt.Sprintf("states[id].result_tag = %du;", l.tagForType(v.Args[0].Type))))
			b.WriteString(l.line(fmt.Sprintf("states[id].result_data = i32(%sv%d);", prefix, v.Args[0].ID)))
		}

	case OpPhi:
		// Simplified for MVP
	}

	return nil
}

func (l *WGSLLowering) emitValue(b *strings.Builder, v *Value) error {
	return l.emitValueWithPrefix(b, v, 0)
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
