package compiler

import (
	"fmt"
	"testing"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/gpu"
)

func TestE2E_WGSL_Lowering(t *testing.T) {
	// 1. Define GlyphLang Source
	route := &ast.Route{
		Method: ast.Get,
		Path:   "/compute",
		Body: []ast.Statement{
			ast.AssignStatement{
				Target: "x",
				Value:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 100}},
			},
			ast.AssignStatement{
				Target: "y",
				Value:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 200}},
			},
			ast.ReturnStatement{
				Value: ast.BinaryOpExpr{
					Left:  ast.VariableExpr{Name: "x"},
					Op:    ast.Add,
					Right: ast.VariableExpr{Name: "y"},
				},
			},
		},
	}

	// 2. Compile to WGSL
	c := NewSSACompiler()
	wgsl, err := c.CompileRouteToWGSL(route)
	if err != nil {
		t.Fatalf("Failed to compile to WGSL: %v", err)
	}

	fmt.Println("Generated WGSL:")
	fmt.Println(wgsl)

	// 3. Execute on GPU (using the Rust runner)
	result, err := gpu.ExecuteWGSL(wgsl)
	if err != nil {
		t.Skipf("Skipping GPU execution test (likely no compatible GPU in this environment): %v", err)
		return
	}

	// 4. Verify Result
	// 100 + 200 = 300
	if result.IntVal != 300 {
		t.Errorf("Expected result 300, got %d", result.IntVal)
	}
	
	fmt.Printf("GPU execution successful! Result: %d\n", result.IntVal)
}

func TestE2E_WGSL_Loop(t *testing.T) {
	// 1. Define GlyphLang Source (sum 0..9)
	route := &ast.Route{
		Method: ast.Get,
		Path:   "/loop",
		Body: []ast.Statement{
			ast.AssignStatement{
				Target: "sum",
				Value:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 0}},
			},
			ast.AssignStatement{
				Target: "i",
				Value:  ast.LiteralExpr{Value: ast.IntLiteral{Value: 0}},
			},
			ast.WhileStatement{
				Condition: ast.BinaryOpExpr{
					Left:  ast.VariableExpr{Name: "i"},
					Op:    ast.Lt,
					Right: ast.LiteralExpr{Value: ast.IntLiteral{Value: 10}},
				},
				Body: []ast.Statement{
					ast.AssignStatement{
						Target: "sum",
						Value:  ast.BinaryOpExpr{
							Left:  ast.VariableExpr{Name: "sum"},
							Op:    ast.Add,
							Right: ast.VariableExpr{Name: "i"},
						},
					},
					ast.AssignStatement{
						Target: "i",
						Value:  ast.BinaryOpExpr{
							Left:  ast.VariableExpr{Name: "i"},
							Op:    ast.Add,
							Right: ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
						},
					},
				},
			},
			ast.ReturnStatement{
				Value: ast.VariableExpr{Name: "sum"},
			},
		},
	}

	// 2. Compile to WGSL
	c := NewSSACompiler()
	wgsl, err := c.CompileRouteToWGSL(route)
	if err != nil {
		t.Fatalf("Failed to compile to WGSL: %v", err)
	}

	// 3. Execute on GPU
	result, err := gpu.ExecuteWGSL(wgsl)
	if err != nil {
		t.Skipf("Skipping GPU execution test: %v", err)
		return
	}

	// 4. Verify Result
	// sum(0..9) = 45
	if result.IntVal != 45 {
		t.Errorf("Expected result 45, got %d", result.IntVal)
	}
	
	fmt.Printf("GPU Loop execution successful! Result: %d\n", result.IntVal)
}
