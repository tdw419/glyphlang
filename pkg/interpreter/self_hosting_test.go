package interpreter

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/parser"
)

// ─── SEC-2: Self-hosting — interpreter.glyph runs a simple test program ───

// sec2Setup loads interpreter.glyph (with transitive imports: parser → lexer,
// token, ast) into a Go interpreter instance and returns it.
// This proves that module resolution works across the full bootstrap chain.
func sec2Setup(t *testing.T) *Interpreter {
	t.Helper()

	rootDir := "../../bootstrap"
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatalf("resolving bootstrap dir: %v", err)
	}

	interpSrc, err := os.ReadFile(filepath.Join(absRoot, "interpreter.glyph"))
	if err != nil {
		t.Fatalf("reading interpreter.glyph: %v", err)
	}

	lex := parser.NewLexer(string(interpSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenizing interpreter.glyph: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parsing interpreter.glyph: %v", err)
	}

	interp := NewInterpreter()
	interp.moduleResolver.ParseFunc = func(source string) (*Module, error) {
		lex := parser.NewLexer(source)
		tokens, err := lex.Tokenize()
		if err != nil {
			return nil, err
		}
		p := parser.NewParser(tokens)
		return p.Parse()
	}

	// Load interpreter.glyph from the bootstrap directory so that
	// import "./parser" resolves to bootstrap/parser.glyph, which
	// transitively imports bootstrap/lexer.glyph, bootstrap/token.glyph,
	// and bootstrap/ast.glyph.
	if err := interp.LoadModuleWithPath(*mod, absRoot+"/"); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	return interp
}

// TestSEC2_ImportChain verifies that interpreter.glyph loads successfully
// with the full import chain: interpreter → parser → lexer, token, ast.
// This confirms module resolution works for the self-hosting path.
func TestSEC2_ImportChain(t *testing.T) {
	interp := sec2Setup(t)

	// Verify key functions were loaded from interpreter.glyph
	for _, name := range []string{"eval_source", "eval_expr", "exec_stmt", "exec_module"} {
		if _, ok := interp.GetFunction(name); !ok {
			t.Errorf("expected function %q to be loaded from interpreter.glyph", name)
		}
	}

	// Verify the parser module was imported (as a namespace in global env)
	parserVal, err := interp.globalEnv.Get("parser")
	if err != nil {
		t.Fatalf("parser module not found in global env: %v", err)
	}
	parserMap, ok := parserVal.(map[string]interface{})
	if !ok {
		t.Fatalf("expected parser to be a map, got %T", parserVal)
	}

	// Verify key parser functions are exported
	for _, name := range []string{"new_parser", "parse"} {
		if _, exists := parserMap[name]; !exists {
			t.Errorf("parser module missing export %q", name)
		}
	}
}

// TestSEC2_EvalSource_SimpleArithmetic verifies that the bootstrap
// interpreter (loaded from interpreter.glyph) can execute a simple
// arithmetic program via its eval_source function and return 5.
//
// Chain: Go → interpreter.glyph → parser.glyph → eval → result
func TestSEC2_EvalSource_SimpleArithmetic(t *testing.T) {
	interp := sec2Setup(t)

	// Use the bootstrap interpreter's eval_source to run "$ result = 2 + 3"
	program := "$ result = 2 + 3"

	evalCallExpr := FunctionCallExpr{
		Name: "eval_source",
		Args: []Expr{LiteralExpr{Value: StringLiteral{Value: program}}},
	}

	evalResult, err := interp.evaluateFunctionCall(evalCallExpr, interp.globalEnv)
	if err != nil {
		t.Fatalf("eval_source call failed: %v", err)
	}

	// eval_source returns { value, error }
	resMap, ok := evalResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", evalResult, evalResult)
	}

	if errStr, _ := resMap["error"].(string); errStr != "" {
		t.Fatalf("eval_source returned error: %s", errStr)
	}

	value := resMap["value"]
	if value == nil {
		t.Fatal("eval_source returned nil value")
	}

	valInt, ok := value.(int64)
	if !ok {
		t.Fatalf("expected int64 value, got %T: %v", value, value)
	}

	if valInt != 5 {
		t.Errorf("expected 5, got %d", valInt)
	}
}

// TestSEC2_EvalSource_PrintOutput verifies that the bootstrap interpreter
// can execute a program that uses print(), capturing stdout to confirm
// the output is "5".
//
// This uses a program with a function call and print, since bare print()
// at the top level isn't supported by the bootstrap parser.
func TestSEC2_EvalSource_PrintOutput(t *testing.T) {
	interp := sec2Setup(t)

	// A program that computes 2+3 and prints it
	program := "! main() {\n  $ x = 2 + 3\n  print(x)\n}\nmain()"

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	evalCallExpr := FunctionCallExpr{
		Name: "eval_source",
		Args: []Expr{LiteralExpr{Value: StringLiteral{Value: program}}},
	}

	evalResult, err := interp.evaluateFunctionCall(evalCallExpr, interp.globalEnv)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if err != nil {
		t.Fatalf("eval_source call failed: %v", err)
	}

	output := buf.String()

	// Check eval_source didn't return an error
	resMap, ok := evalResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", evalResult, evalResult)
	}
	if errStr, _ := resMap["error"].(string); errStr != "" {
		t.Fatalf("eval_source returned error: %s", errStr)
	}

	if !strings.Contains(output, "5") {
		t.Errorf("expected output to contain '5', got: %q", output)
	}
}

// TestSEC2_RunTest verifies that the bootstrap interpreter's built-in
// run_test function can parse and execute a program, confirming the
// full pipeline: lexer → parser → eval.
func TestSEC2_RunTest(t *testing.T) {
	interp := sec2Setup(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call run_test("simple add", "$ result = 2 + 3", 5)
	runTestCall := FunctionCallExpr{
		Name: "run_test",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: "simple add"}},
			LiteralExpr{Value: StringLiteral{Value: "$ result = 2 + 3"}},
			LiteralExpr{Value: IntLiteral{Value: 5}},
		},
	}

	result, err := interp.evaluateFunctionCall(runTestCall, interp.globalEnv)

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if err != nil {
		t.Fatalf("run_test call failed: %v", err)
	}

	output := buf.String()

	// run_test returns true on success
	passed, ok := result.(bool)
	if !ok {
		t.Fatalf("expected bool result from run_test, got %T: %v", result, result)
	}

	if !passed {
		t.Errorf("run_test returned false. Output:\n%s", output)
	}

	if !strings.Contains(output, "PASS") {
		t.Errorf("expected PASS in output, got: %s", output)
	}

	if !strings.Contains(output, "5") {
		t.Errorf("expected result 5 in output, got: %s", output)
	}
}
