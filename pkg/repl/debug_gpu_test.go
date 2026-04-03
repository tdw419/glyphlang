package repl

import (
	"os"
	"bytes"
	"strings"
	"testing"

	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/parser"
)

func TestDebugGPUPath(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	input := "3 + 2"
	
	// 1. Parse expression
	lexer := parser.NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	p := parser.NewParser(tokens)
	expr, err := p.ParseExpression()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	t.Logf("Parsed expression: %T %+v", expr, expr)

	// 2. Compile via SSA
	c := compiler.NewSSACompiler()
	fn := &ast.Function{Name: "repl", Body: []ast.Statement{&ast.ReturnStatement{Value: expr}}}
	bytecode, err := c.CompileFunction(fn)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	t.Logf("Bytecode length: %d", len(bytecode))
	t.Logf("Bytecode hex: %x", bytecode)

	// 3. Execute via dispatcher
	d := gpu.NewDispatcher()
	d.SetCPUFallback()
	results, err := d.Execute(bytecode, 1)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	t.Logf("Result[0]: Tag=%d IntVal=%d FloatVal=%f BoolVal=%v Error=%v Steps=%d",
		results[0].Tag, results[0].IntVal, results[0].FloatVal, results[0].BoolVal, results[0].Error, results[0].Steps)

	// 4. Full REPL test
	output := &bytes.Buffer{}
	r := New(strings.NewReader(""), output, "test", true)
	if err := r.processLine("3 + 2"); err != nil {
		t.Fatalf("processLine error: %v", err)
	}
	t.Logf("REPL output: %q", output.String())
}
