package ssa

import "math"

// ConstFoldPass evaluates operations on constants at compile time.
type ConstFoldPass struct{}

func (p *ConstFoldPass) Name() string { return "constfold" }

func (p *ConstFoldPass) Run(f *Func) bool {
	changed := false
	for _, blk := range f.Blocks {
		for i, v := range blk.Values {
			if r := foldValue(v); r != nil {
				blk.Values[i] = r
				r.ID = v.ID
				r.Block = blk
				replaceUses(f, v, r)
				changed = true
			}
		}
	}
	return changed
}

func foldValue(v *Value) *Value {
	// Need all args to be constants
	for _, a := range v.Args {
		if a.Op != OpConst {
			return nil
		}
	}

	switch v.Op {
	// Integer binary
	case OpAddInt:
		return constInt(v.Args[0].AuxInt + v.Args[1].AuxInt)
	case OpSubInt:
		return constInt(v.Args[0].AuxInt - v.Args[1].AuxInt)
	case OpMulInt:
		return constInt(v.Args[0].AuxInt * v.Args[1].AuxInt)
	case OpDivInt:
		if v.Args[1].AuxInt == 0 {
			return nil
		}
		return constInt(v.Args[0].AuxInt / v.Args[1].AuxInt)
	case OpModInt:
		if v.Args[1].AuxInt == 0 {
			return nil
		}
		return constInt(v.Args[0].AuxInt % v.Args[1].AuxInt)
	case OpNegInt:
		return constInt(-v.Args[0].AuxInt)

	// Float binary
	case OpAddFloat:
		return constFloat(getFloat(v.Args[0]) + getFloat(v.Args[1]))
	case OpSubFloat:
		return constFloat(getFloat(v.Args[0]) - getFloat(v.Args[1]))
	case OpMulFloat:
		return constFloat(getFloat(v.Args[0]) * getFloat(v.Args[1]))
	case OpDivFloat:
		b := getFloat(v.Args[1])
		if b == 0 {
			return nil
		}
		return constFloat(getFloat(v.Args[0]) / b)
	case OpNegFloat:
		return constFloat(-getFloat(v.Args[0]))

	// String concat
	case OpAddStr:
		return constStr(v.Args[0].AuxStr + v.Args[1].AuxStr)

	// Integer comparisons
	case OpEqInt:
		return constBool(v.Args[0].AuxInt == v.Args[1].AuxInt)
	case OpNeInt:
		return constBool(v.Args[0].AuxInt != v.Args[1].AuxInt)
	case OpLtInt:
		return constBool(v.Args[0].AuxInt < v.Args[1].AuxInt)
	case OpLeInt:
		return constBool(v.Args[0].AuxInt <= v.Args[1].AuxInt)
	case OpGtInt:
		return constBool(v.Args[0].AuxInt > v.Args[1].AuxInt)
	case OpGeInt:
		return constBool(v.Args[0].AuxInt >= v.Args[1].AuxInt)

	// Float comparisons
	case OpEqFloat:
		return constBool(getFloat(v.Args[0]) == getFloat(v.Args[1]))
	case OpNeFloat:
		return constBool(getFloat(v.Args[0]) != getFloat(v.Args[1]))
	case OpLtFloat:
		return constBool(getFloat(v.Args[0]) < getFloat(v.Args[1]))
	case OpLeFloat:
		return constBool(getFloat(v.Args[0]) <= getFloat(v.Args[1]))
	case OpGtFloat:
		return constBool(getFloat(v.Args[0]) > getFloat(v.Args[1]))
	case OpGeFloat:
		return constBool(getFloat(v.Args[0]) >= getFloat(v.Args[1]))

	// Boolean logic
	case OpNot:
		return constBool(v.Args[0].AuxInt == 0)
	case OpAnd:
		return constBool(v.Args[0].AuxInt != 0 && v.Args[1].AuxInt != 0)
	case OpOr:
		return constBool(v.Args[0].AuxInt != 0 || v.Args[1].AuxInt != 0)
	}
	return nil
}

func constInt(n int64) *Value {
	return &Value{Op: OpConst, Type: TypeInt, AuxInt: n}
}

func constFloat(f float64) *Value {
	return &Value{Op: OpConst, Type: TypeFloat, AuxInt: int64(math.Float64bits(f))}
}

func constBool(b bool) *Value {
	v := &Value{Op: OpConst, Type: TypeBool}
	if b {
		v.AuxInt = 1
	}
	return v
}

func constStr(s string) *Value {
	return &Value{Op: OpConst, Type: TypeString, AuxStr: s}
}

func getFloat(v *Value) float64 {
	return math.Float64frombits(uint64(v.AuxInt))
}

// replaceUses replaces all uses of old with new throughout the function.
func replaceUses(f *Func, old, new *Value) {
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			for i, a := range v.Args {
				if a == old {
					v.Args[i] = new
				}
			}
		}
	}
}
