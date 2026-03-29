package ssa

// CopyPropPass propagates copies and simplifies trivial phi nodes.
// A phi with all identical arguments (or self-references) is replaced
// by that single value.
type CopyPropPass struct{}

func (p *CopyPropPass) Name() string { return "copyprop" }

func (p *CopyPropPass) Run(f *Func) bool {
	changed := false
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			if v.Op != OpPhi {
				continue
			}
			// Find the single non-self value among phi args
			var single *Value
			trivial := true
			for _, a := range v.Args {
				if a == v {
					continue // self-reference
				}
				if single == nil {
					single = a
				} else if a != single {
					trivial = false
					break
				}
			}
			if trivial && single != nil {
				replaceUses(f, v, single)
				v.Op = single.Op
				v.Type = single.Type
				v.Args = single.Args
				v.AuxInt = single.AuxInt
				v.AuxStr = single.AuxStr
				changed = true
			}
		}
	}
	return changed
}
