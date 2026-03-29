package ssa

// SimplifyPass performs algebraic simplifications:
//   x + 0  → x       x - 0  → x
//   x * 1  → x       x * 0  → 0
//   x / 1  → x       --x    → x
//   !!x    → x
type SimplifyPass struct{}

func (p *SimplifyPass) Name() string { return "simplify" }

func (p *SimplifyPass) Run(f *Func) bool {
	changed := false
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			if r := simplify(v); r != nil {
				replaceUses(f, v, r)
				changed = true
			}
		}
	}
	return changed
}

func simplify(v *Value) *Value {
	switch v.Op {
	case OpAddInt:
		if isIntConst(v.Args[1], 0) {
			return v.Args[0]
		}
		if isIntConst(v.Args[0], 0) {
			return v.Args[1]
		}
	case OpSubInt:
		if isIntConst(v.Args[1], 0) {
			return v.Args[0]
		}
	case OpMulInt:
		if isIntConst(v.Args[1], 1) {
			return v.Args[0]
		}
		if isIntConst(v.Args[0], 1) {
			return v.Args[1]
		}
		if isIntConst(v.Args[1], 0) {
			return v.Args[1] // the zero constant
		}
		if isIntConst(v.Args[0], 0) {
			return v.Args[0]
		}
	case OpDivInt:
		if isIntConst(v.Args[1], 1) {
			return v.Args[0]
		}
	case OpNegInt:
		if v.Args[0].Op == OpNegInt {
			return v.Args[0].Args[0] // --x → x
		}
	case OpNegFloat:
		if v.Args[0].Op == OpNegFloat {
			return v.Args[0].Args[0]
		}
	case OpNot:
		if v.Args[0].Op == OpNot {
			return v.Args[0].Args[0] // !!x → x
		}
	}
	return nil
}

func isIntConst(v *Value, n int64) bool {
	return v.Op == OpConst && v.Type == TypeInt && v.AuxInt == n
}
