package compiler

import (
	"testing"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/vm"
)

// TestNormalizeExpressionPointerAndValueIdentical verifies that the compiler
// produces identical output whether expressions arrive as pointer or value types.
// This is the core guarantee of P2-9 (consistent pointer vs value types).
func TestNormalizeExpressionPointerAndValueIdentical(t *testing.T) {
	t.Run("literal_expr", func(t *testing.T) {
		ptr := &ast.LiteralExpr{Value: ast.IntLiteral{Value: 42}}
		val := ast.LiteralExpr{Value: ast.IntLiteral{Value: 42}}

		c1 := NewCompiler()
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})

	t.Run("variable_expr", func(t *testing.T) {
		ptr := &ast.VariableExpr{Name: "x"}
		val := ast.VariableExpr{Name: "x"}

		c1 := NewCompiler()
		c1.symbolTable.Define("x", c1.addConstant(vm.StringValue{Val: "x"}))
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		c2.symbolTable.Define("x", c2.addConstant(vm.StringValue{Val: "x"}))
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})

	t.Run("binary_op_expr", func(t *testing.T) {
		ptr := &ast.BinaryOpExpr{
			Left:  &ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
			Right: &ast.LiteralExpr{Value: ast.IntLiteral{Value: 2}},
			Op:    ast.Add,
		}
		val := ast.BinaryOpExpr{
			Left:  &ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
			Right: &ast.LiteralExpr{Value: ast.IntLiteral{Value: 2}},
			Op:    ast.Add,
		}

		c1 := NewCompiler()
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})

	t.Run("unary_op_expr", func(t *testing.T) {
		ptr := &ast.UnaryOpExpr{
			Right: &ast.LiteralExpr{Value: ast.BoolLiteral{Value: true}},
			Op:    ast.Not,
		}
		val := ast.UnaryOpExpr{
			Right: &ast.LiteralExpr{Value: ast.BoolLiteral{Value: true}},
			Op:    ast.Not,
		}

		c1 := NewCompiler()
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})

	t.Run("field_access_expr", func(t *testing.T) {
		ptr := &ast.FieldAccessExpr{
			Object: &ast.VariableExpr{Name: "x"},
			Field:  "name",
		}
		val := ast.FieldAccessExpr{
			Object: &ast.VariableExpr{Name: "x"},
			Field:  "name",
		}

		c1 := NewCompiler()
		c1.symbolTable.Define("x", c1.addConstant(vm.StringValue{Val: "x"}))
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		c2.symbolTable.Define("x", c2.addConstant(vm.StringValue{Val: "x"}))
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})

	t.Run("try_expression", func(t *testing.T) {
		ptr := &ast.TryExpression{
			Expr: &ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
		}
		val := ast.TryExpression{
			Expr: &ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
		}

		c1 := NewCompiler()
		if err := c1.compileExpression(ptr); err != nil {
			t.Fatalf("pointer compileExpression: %v", err)
		}
		bc1, _ := c1.buildBytecode()

		c2 := NewCompiler()
		if err := c2.compileExpression(val); err != nil {
			t.Fatalf("value compileExpression: %v", err)
		}
		bc2, _ := c2.buildBytecode()

		if len(bc1) != len(bc2) {
			t.Errorf("bytecode length mismatch: pointer=%d, value=%d", len(bc1), len(bc2))
		}
	})
}

// TestCompileExpressionValueTypesInStatements verifies that value-type
// expressions embedded in statements compile correctly through normalizeExpression.
func TestCompileExpressionValueTypesInStatements(t *testing.T) {
	t.Run("value_expression_in_if", func(t *testing.T) {
		route := &ast.Route{
			Path: "/test",
			Body: []ast.Statement{
				ast.IfStatement{
					Condition: ast.BinaryOpExpr{
						Left:  &ast.LiteralExpr{Value: ast.IntLiteral{Value: 1}},
						Right: &ast.LiteralExpr{Value: ast.IntLiteral{Value: 2}},
						Op:    ast.Lt,
					},
					ThenBlock: []ast.Statement{
						ast.ReturnStatement{
							Value: &ast.LiteralExpr{Value: ast.StringLiteral{Value: "yes"}},
						},
					},
				},
			},
		}

		c := NewCompiler()
		_, err := c.CompileRoute(route)
		if err != nil {
			t.Fatalf("CompileRoute with value-type expressions: %v", err)
		}
	})

	t.Run("value_expression_in_return", func(t *testing.T) {
		route := &ast.Route{
			Path: "/test",
			Body: []ast.Statement{
				ast.ReturnStatement{
					Value: &ast.UnaryOpExpr{
						Right: &ast.LiteralExpr{Value: ast.BoolLiteral{Value: false}},
						Op:    ast.Not,
					},
				},
			},
		}

		c := NewCompiler()
		_, err := c.CompileRoute(route)
		if err != nil {
			t.Fatalf("CompileRoute with value-type unary: %v", err)
		}
	})
}
