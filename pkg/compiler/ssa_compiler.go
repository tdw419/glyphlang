package compiler

import (
	"fmt"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/ssa"
)

// SSACompiler implements the compilation pipeline using SSA intermediate representation.
type SSACompiler struct {
	passMgr *ssa.PassManager
}

// NewSSACompiler creates a new SSA-based compiler.
func NewSSACompiler() *SSACompiler {
	return &SSACompiler{
		passMgr: ssa.DefaultPasses(),
	}
}

// CompileRoute compiles a route to bytecode via SSA.
func (c *SSACompiler) CompileRoute(route *ast.Route) ([]byte, error) {
	// 1. AST -> SSA
	builder := ssa.NewBuilder()
	f, err := builder.BuildRoute(route)
	if err != nil {
		return nil, fmt.Errorf("SSA build failed: %w", err)
	}

	// 2. Optimize SSA
	c.passMgr.Run(f)

	// 3. Lower SSA -> Bytecode (CPU)
	lowering := ssa.NewCPULowering()
	return lowering.LowerFunc(f)
}

// CompileRouteToGPU compiles a route to GPU-compatible bytecode via SSA.
func (c *SSACompiler) CompileRouteToGPU(route *ast.Route) ([]byte, error) {
	// 1. AST -> SSA
	builder := ssa.NewBuilder()
	f, err := builder.BuildRoute(route)
	if err != nil {
		return nil, fmt.Errorf("SSA build failed: %w", err)
	}

	// 2. Optimize SSA
	c.passMgr.Run(f)

	// 3. Lower SSA -> Bytecode (GPU)
	lowering := ssa.NewGPULowering()
	return lowering.LowerFunc(f)
}

// IsGPUCompatible checks if a route can be executed on the GPU using SSA analysis.
func (c *SSACompiler) IsGPUCompatible(route *ast.Route) bool {
	builder := ssa.NewBuilder()
	f, err := builder.BuildRoute(route)
	if err != nil {
		return false
	}

	// Run passes first (might simplify unsupported code into supported code)
	c.passMgr.Run(f)

	lowering := ssa.NewGPULowering()
	_, err = lowering.LowerFunc(f)
	return err == nil
}
