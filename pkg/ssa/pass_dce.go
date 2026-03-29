package ssa

// DCEPass removes values that have no uses and no side effects.
type DCEPass struct{}

func (p *DCEPass) Name() string { return "dce" }

func (p *DCEPass) Run(f *Func) bool {
	// Build use counts
	uses := make(map[*Value]int)
	for _, blk := range f.Blocks {
		for _, v := range blk.Values {
			for _, a := range v.Args {
				uses[a]++
			}
		}
	}

	changed := false
	for _, blk := range f.Blocks {
		live := blk.Values[:0]
		for _, v := range blk.Values {
			if uses[v] > 0 || v.Op.IsSideEffect() || v.Op.IsTerminator() {
				live = append(live, v)
			} else {
				// Decrement use counts for dead value's args
				for _, a := range v.Args {
					uses[a]--
				}
				changed = true
			}
		}
		blk.Values = live
	}
	return changed
}
