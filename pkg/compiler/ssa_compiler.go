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

// CompileRouteToWGSL compiles a route directly to a WGSL compute shader via SSA.
func (c *SSACompiler) CompileRouteToWGSL(route *ast.Route) (string, error) {
	// 1. AST -> SSA
	builder := ssa.NewBuilder()
	f, err := builder.BuildRoute(route)
	if err != nil {
		return "", fmt.Errorf("SSA build failed: %w", err)
	}

	// 2. Optimize SSA
	c.passMgr.Run(f)

	// 3. Lower SSA -> WGSL
	lowering := ssa.NewWGSLLowering()
	return lowering.LowerFunc(f)
}

// CompileModuleToWGSL compiles all routes in a module to a single multi-route WGSL shader
// with adaptive workgroup sizing.
func (c *SSACompiler) CompileModuleToWGSL(module *ast.Module, workgroupSize int) (string, error) {
	var funcs []*ssa.Func
	builder := ssa.NewBuilder()

	for _, item := range module.Items {
		if route, ok := item.(*ast.Route); ok {
			f, err := builder.BuildRoute(route)
			if err != nil {
				return "", fmt.Errorf("SSA build failed for %s: %w", route.Path, err)
			}
			c.passMgr.Run(f)
			funcs = append(funcs, f)
		}
	}

	if len(funcs) == 0 {
		return "", fmt.Errorf("no routes found in module")
	}

	lowering := ssa.NewWGSLLowering()
	if workgroupSize > 0 {
		lowering.SetWorkgroupSize(workgroupSize)
	}
	return lowering.LowerMultiFunc(funcs)
}

// CompileFunction compiles an AST function to bytecode via SSA.
func (c *SSACompiler) CompileFunction(fn *ast.Function) ([]byte, error) {
	// 1. AST -> SSA
	builder := ssa.NewBuilder()
	f, err := builder.BuildFunction(fn)
	if err != nil {
		return nil, fmt.Errorf("SSA build failed: %w", err)
	}

	// 2. Optimize SSA
	c.passMgr.Run(f)

	// 3. Lower SSA -> Bytecode (CPU)
	lowering := ssa.NewCPULowering()
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

// IsWGSLCompatible checks if a route can be lowered to a WGSL compute shader.
func (c *SSACompiler) IsWGSLCompatible(route *ast.Route) bool {
	builder := ssa.NewBuilder()
	f, err := builder.BuildRoute(route)
	if err != nil {
		return false
	}

	// Run passes first (might simplify unsupported code)
	c.passMgr.Run(f)

	lowering := ssa.NewWGSLLowering()
	_, err = lowering.LowerFunc(f)
	return err == nil
}
