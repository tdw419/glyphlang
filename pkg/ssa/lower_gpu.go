package ssa

import (
	"fmt"
)

// GPULowering lowers an SSA function to GLYP bytecode for the GPU dispatcher.
// It only supports a subset of operations and primitive types (int, float, bool).
type GPULowering struct {
	cpu CPULowering // reuse bytecode emission logic
}

// NewGPULowering creates a new GPU lowering context.
func NewGPULowering() *GPULowering {
	return &GPULowering{
		cpu: *NewCPULowering(),
	}
}

// LowerFunc lowers a single SSA function to GPU-compatible bytecode.
func (l *GPULowering) LowerFunc(f *Func) ([]byte, error) {
	// 1. Validate compatibility
	if err := l.validate(f); err != nil {
		return nil, fmt.Errorf("GPU compatibility error: %w", err)
	}

	// 2. Reuse CPU lowering logic (bytecode format is the same)
	return l.cpu.LowerFunc(f)
}

func (l *GPULowering) validate(f *Func) error {
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			if !l.isOpSupported(v.Op) {
				return fmt.Errorf("unsupported GPU operation: %s", v.Op)
			}
			if !l.isTypeSupported(v.Type) {
				return fmt.Errorf("unsupported GPU type: %s in v%d", v.Type, v.ID)
			}
			for _, arg := range v.Args {
				if !l.isTypeSupported(arg.Type) {
					return fmt.Errorf("unsupported GPU type: %s in argument to v%d", arg.Type, v.ID)
				}
			}
		}
	}
	return nil
}

func (l *GPULowering) isOpSupported(op Op) bool {
	switch op {
	case OpConst,
		OpAddInt, OpSubInt, OpMulInt, OpDivInt, OpModInt, OpNegInt,
		OpAddFloat, OpSubFloat, OpMulFloat, OpDivFloat, OpNegFloat,
		OpEqInt, OpNeInt, OpLtInt, OpLeInt, OpGtInt, OpGeInt,
		OpEqFloat, OpNeFloat, OpLtFloat, OpLeFloat, OpGtFloat, OpGeFloat,
		OpEqBool, OpNeBool,
		OpAnd, OpOr, OpNot,
		OpLoadVar, OpStoreVar,
		OpIf, OpJump, OpReturn, OpHalt, OpPhi:
		return true
	}
	return false
}

func (l *GPULowering) isTypeSupported(t Type) bool {
	switch t {
	case TypeInt, TypeFloat, TypeBool, TypeNull, TypeVoid:
		return true
	}
	return false
}
