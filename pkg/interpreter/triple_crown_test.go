package interpreter

import (
	. "github.com/glyphlang/glyph/pkg/ast"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glyphlang/glyph/pkg/parser"
)

// tripleCrownSetup loads interpreter.glyph (and its transitive imports) into
// a Go interpreter instance. Returns the interpreter and the elapsed load time.
func tripleCrownSetup(t *testing.T) (*Interpreter, time.Duration) {
	t.Helper()

	rootDir := filepath.Join("..", "..", "bootstrap")
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
	start := time.Now()
	if err := interp.LoadModuleWithPath(*mod, absRoot+"/"); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}
	loadDur := time.Since(start)

	return interp, loadDur
}

// ─── Stage 2: Bootstrap interpreter evaluates a non-trivial program ──────

func TestTripleCrown_Stage2_BootstrapEvaluatesProgram(t *testing.T) {
	interp, loadDur := tripleCrownSetup(t)
	t.Logf("Stage 2 setup: loaded interpreter.glyph in %s", loadDur)

	// The bootstrap interpreter registers an "@ command test" that runs a
	// quick 6-test subset covering: const+arithmetic, function call,
	// while loop, if/else, recursive factorial, object literal + field access.
	cmd, ok := interp.GetCommand("test")
	if !ok {
		t.Fatal("bootstrap interpreter did not register '@ command test'")
	}

	start := time.Now()
	result, err := interp.ExecuteCommand(&cmd, map[string]interface{}{})
	execDur := time.Since(start)
	t.Logf("Stage 2 execution: @ command test ran in %s", execDur)
	t.Logf("Stage 2 total: %s", loadDur+execDur)

	if err != nil {
		t.Fatalf("ExecuteCommand test: %v", err)
	}

	// The command returns { ok: true, passed: N, failed: N }
	resMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	if okVal, _ := resMap["ok"].(bool); !okVal {
		t.Fatalf("Stage 2 FAILED - result: %+v", resMap)
	}

	passed, _ := resMap["passed"].(int64)
	failed, _ := resMap["failed"].(int64)
	t.Logf("Stage 2 result: passed=%d failed=%d", passed, failed)

	if failed != 0 {
		t.Errorf("Stage 2: expected 0 failures, got %d", failed)
	}
}

// ─── Stage 3: Triple nesting — interpreter interprets itself ─────────────

func TestTripleCrown_Stage3_NestedSelfInterpretation(t *testing.T) {
	interp, loadDur := tripleCrownSetup(t)
	t.Logf("Stage 3 setup: loaded interpreter.glyph in %s", loadDur)

	// The bootstrap interpreter has eval_source(src) which parses + executes
	// GlyphLang source. We create a GlyphLang program that calls eval_source
	// with a non-trivial program (our own mini test), effectively:
	//   Go -> interpreter.glyph -> eval_source(program)
	//
	// This is the triple crown: Go interprets GlyphLang (interpreter.glyph)
	// which itself interprets GlyphLang (the eval_source'd program).

	// The program we'll eval_source: a non-trivial computation involving
	// arithmetic, function calls, recursion, and string concatenation.
	nestedProgram := `! fib(n: int) -> int {
  if n <= 1 { > n }
  > fib(n - 1) + fib(n - 2)
}
$ result = fib(10)`

	// Use the bootstrap interpreter's eval_source function to run the nested program.
	// eval_source is defined in interpreter.glyph and returns an EvalResult { value, error }.
	_, ok := interp.GetFunction("eval_source")
	if !ok {
		t.Fatal("bootstrap interpreter did not define eval_source function")
	}

	env := interp.globalEnv
	start := time.Now()

	// Call eval_source with the nested program
	evalCallExpr := FunctionCallExpr{
		Name: "eval_source",
		Args: []Expr{LiteralExpr{Value: StringLiteral{Value: nestedProgram}}},
	}
	evalResult, err := interp.evaluateFunctionCall(evalCallExpr, env)
	evalDur := time.Since(start)
	t.Logf("Stage 3 eval_source: ran in %s", evalDur)
	t.Logf("Stage 3 total: %s", loadDur+evalDur)

	if err != nil {
		t.Fatalf("eval_source call failed: %v", err)
	}

	// eval_source returns an EvalResult object { value, error }
	resMap, ok := evalResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result from eval_source, got %T: %v", evalResult, evalResult)
	}

	// Check for eval error
	if errStr, _ := resMap["error"].(string); errStr != "" {
		t.Fatalf("eval_source returned error: %s", errStr)
	}

	// fib(10) = 55
	value := resMap["value"]
	if value == nil {
		t.Fatal("eval_source returned nil value")
	}

	if valInt, ok := value.(int64); ok {
		if valInt != 55 {
			t.Errorf("Stage 3: fib(10) = %d, want 55", valInt)
		} else {
			t.Logf("Stage 3 PASS: fib(10) = %d", valInt)
		}
	} else {
		t.Errorf("Stage 3: expected int64 value, got %T: %v", value, value)
	}

	// Verify the loaded function was also accessible
	t.Logf("Stage 3: eval_source loaded successfully - triple nesting confirmed")
	t.Logf("  Go -> bootstrap interpreter.glyph -> eval_source(fib program)")
	t.Logf("  Performance: load=%s eval=%s total=%s", loadDur, evalDur, loadDur+evalDur)
}
