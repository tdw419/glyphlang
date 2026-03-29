package ssa

// Pass is an optimization pass that transforms an SSA function.
type Pass interface {
	Name() string
	Run(f *Func) bool // returns true if anything changed
}

// PassManager runs a sequence of optimization passes.
type PassManager struct {
	passes []Pass
}

// NewPassManager creates a new pass manager.
func NewPassManager() *PassManager {
	return &PassManager{}
}

// Add registers a pass.
func (pm *PassManager) Add(p Pass) {
	pm.passes = append(pm.passes, p)
}

// Run executes all passes on the function, repeating until no pass
// makes any changes (fixed-point iteration).
func (pm *PassManager) Run(f *Func) {
	for {
		changed := false
		for _, p := range pm.passes {
			if p.Run(f) {
				changed = true
			}
		}
		if !changed {
			break
		}
	}
}

// RunOnProgram executes all passes on every function in the program.
func (pm *PassManager) RunOnProgram(prog *Program) {
	for _, f := range prog.Funcs {
		pm.Run(f)
	}
}

// DefaultPasses returns a PassManager with the standard optimization pipeline.
func DefaultPasses() *PassManager {
	pm := NewPassManager()
	pm.Add(&ConstFoldPass{})
	pm.Add(&CopyPropPass{})
	pm.Add(&DCEPass{})
	pm.Add(&SimplifyPass{})
	return pm
}
