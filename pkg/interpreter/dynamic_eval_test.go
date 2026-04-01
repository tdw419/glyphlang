package interpreter

import (
	"strings"
	"testing"

	. "github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/parser"
)

// TestEvalSource_SingleLevel tests that eval_source("print(1 + 2)") works
// when called from within interpreted .glyph code -- the interpreter calls
// its own eval function recursively.
//
// SEC-1 step 1.1: single-level dynamic eval.
func TestEvalSource_SingleLevel(t *testing.T) {
	// Outer .glyph program defines a command that calls eval_source with
	// a source string. eval_source is a Go builtin that parses and executes
	// the given source in a child interpreter.
	source := `@ command run {
  $ r = eval_source("print(1 + 2)")
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "3") {
		t.Errorf("expected output to contain '3', got: %q", output)
	}
}

// TestEvalSource_Nested tests nested eval:
//
//	eval_source("eval_source(\"print(42)\")")
//
// Two levels of interpretation -- the outer program evals a string that
// itself contains an eval_source call.
//
// SEC-1 step 1.2: two levels of nested interpretation.
func TestEvalSource_Nested(t *testing.T) {
	source := `@ command run {
  $ r = eval_source("eval_source(\"print(42)\")")
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "42") {
		t.Errorf("expected output to contain '42', got: %q", output)
	}
}

// TestEvalSource_ReturnsValue verifies that eval_source returns the value
// of the evaluated expression via the result map's "value" field.
func TestEvalSource_ReturnsValue(t *testing.T) {
	source := `@ command run {
  $ r = eval_source("1 + 2")
  $ v = r.value
  print("value:", v)
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "value: 3") {
		t.Errorf("expected output to contain 'value: 3', got: %q", output)
	}
}

// TestEvalSource_FromFunction verifies that eval_source works when called
// from within a user-defined function (not just from a command body).
func TestEvalSource_FromFunction(t *testing.T) {
	source := `! do_eval() {
  $ r = eval_source("10 * 4")
  > r.value
}
@ command run {
  $ v = do_eval()
  print("do_eval returned:", v)
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "do_eval returned: 40") {
		t.Errorf("expected output to contain 'do_eval returned: 40', got: %q", output)
	}
}

// TestEvalSource_WithArithmetic verifies eval_source correctly evaluates
// more complex arithmetic expressions.
func TestEvalSource_WithArithmetic(t *testing.T) {
	source := `@ command run {
  $ r = eval_source("(10 + 5) * 3")
  print("computed:", r.value)
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "45") {
		t.Errorf("expected output to contain '45', got: %q", output)
	}
}

// TestEvalSource_ParseError verifies that eval_source returns a useful error
// when given invalid source, rather than panicking.
func TestEvalSource_ParseError(t *testing.T) {
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
	env := NewEnvironment()

	// Call eval_source with invalid source directly via the builtin
	invalidSrc := "}}}}" + "INVALID" + "}}}"
	result, err := interp.evaluateFunctionCall(FunctionCallExpr{
		Name: "eval_source",
		Args: []Expr{LiteralExpr{Value: StringLiteral{Value: invalidSrc}}},
	}, env)

	if err != nil {
		t.Fatalf("eval_source should not return a Go error for parse failures, got: %v", err)
	}

	// Should return a map with an "error" key
	resMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	errStr, _ := resMap["error"].(string)
	if errStr == "" {
		t.Error("expected non-empty error string for invalid source")
	}
}
