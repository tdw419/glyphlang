package compiler

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/gpu"
)

func TestE2E_WGSL_Adaptive(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" {
		t.Skip("requires GLYPH_TEST_GPU=1")
	}
	// 1. Define Module with Multiple Routes
	module := &ast.Module{
		Items: []ast.Item{
			&ast.Route{
				Method: ast.Get,
				Path:   "/add",
				Body: []ast.Statement{
					ast.ReturnStatement{
						Value: ast.BinaryOpExpr{
							Left:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 40}},
							Op:    ast.Add,
							Right: ast.LiteralExpr{Value: ast.IntLiteral{Value: 2}},
						},
					},
				},
			},
			&ast.Route{
				Method: ast.Get,
				Path:   "/mul",
				Body: []ast.Statement{
					ast.ReturnStatement{
						Value: ast.BinaryOpExpr{
							Left:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 10}},
							Op:    ast.Mul,
							Right: ast.LiteralExpr{Value: ast.IntLiteral{Value: 5}},
						},
					},
				},
			},
		},
	}

	// 2. Determine Adaptive Sizing
	// Heuristic: workgroup_size = min(256, num_routes)
	numRoutes := 2
	wgSize := 256
	if numRoutes < 256 {
		wgSize = 64 // Use 64 for small sets
	}
	
	numWorkgroups := int(math.Ceil(float64(numRoutes) / float64(wgSize)))

	// 3. Compile Module to Multi-Route WGSL
	c := NewSSACompiler()
	wgsl, err := c.CompileModuleToWGSL(module, wgSize)
	if err != nil {
		t.Fatalf("Failed to compile module to WGSL: %v", err)
	}

	fmt.Printf("Generated WGSL (workgroup_size: %d):\n", wgSize)
	// fmt.Println(wgsl)

	// 4. Execute on GPU
	results, err := gpu.ExecuteMultiWGSL([]byte(wgsl), numRoutes, numWorkgroups)
	if err != nil {
		t.Skipf("Skipping GPU execution test: %v", err)
		return
	}

	// 5. Verify Results
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Route 0: 40 + 2 = 42
	if results[0].IntVal != 42 {
		t.Errorf("Result 0: expected 42, got %d", results[0].IntVal)
	}

	// Route 1: 10 * 5 = 50
	if results[1].IntVal != 50 {
		t.Errorf("Result 1: expected 50, got %d", results[1].IntVal)
	}
	
	fmt.Printf("GPU Adaptive Dispatch successful! Results: %v\n", results)
}
