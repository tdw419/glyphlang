package interpreter

import (
	. "github.com/glyphlang/glyph/pkg/ast"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glyphlang/glyph/pkg/parser"
)

// TestStandaloneExecution verifies that bootstrap/main.glyph can act as a
// standalone entry point — loading and executing an external .glyph file
// through the bootstrap interpreter pipeline (lexer → parser → eval).
// This is Issue #12: Self-Hosting — Standalone GlyphLang execution without Go binary.
func TestStandaloneExecution(t *testing.T) {
	// Load bootstrap/main.glyph (which imports ./interpreter)
	rootDir := filepath.Join("..", "..", "bootstrap")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatalf("resolving bootstrap dir: %v", err)
	}

	mainSrc, err := os.ReadFile(filepath.Join(absRoot, "main.glyph"))
	if err != nil {
		t.Fatalf("reading bootstrap/main.glyph: %v (does it exist?)", err)
	}

	lex := parser.NewLexer(string(mainSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenizing main.glyph: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parsing main.glyph: %v", err)
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
	t.Logf("Loaded bootstrap/main.glyph in %s", loadDur)

	// The main.glyph should register a "run" command that accepts a file path
	cmd, ok := interp.GetCommand("run")
	if !ok {
		t.Fatal("bootstrap/main.glyph did not register '@ command run'")
	}

	// Create a temporary .glyph file to execute via the standalone path
	tmpDir := t.TempDir()
	helloFile := filepath.Join(tmpDir, "hello.glyph")
	helloSrc := `$ result = 40 + 2`
	if err := os.WriteFile(helloFile, []byte(helloSrc), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	// Execute the "run" command with the file path
	result, err := interp.ExecuteCommand(&cmd, map[string]interface{}{
		"file": helloFile,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand run: %v", err)
	}

	// Verify result
	resMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	if okVal, _ := resMap["ok"].(bool); !okVal {
		t.Fatalf("standalone execution FAILED — result: %+v", resMap)
	}

	t.Logf("Standalone execution result: %+v", resMap)
}

// TestStandaloneExecutionWithFunction verifies that main.glyph can execute
// a .glyph file containing function definitions and calls.
func TestStandaloneExecutionWithFunction(t *testing.T) {
	rootDir := filepath.Join("..", "..", "bootstrap")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatalf("resolving bootstrap dir: %v", err)
	}

	mainSrc, err := os.ReadFile(filepath.Join(absRoot, "main.glyph"))
	if err != nil {
		t.Fatalf("reading bootstrap/main.glyph: %v", err)
	}

	lex := parser.NewLexer(string(mainSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenizing main.glyph: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parsing main.glyph: %v", err)
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
	if err := interp.LoadModuleWithPath(*mod, absRoot+"/"); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	cmd, ok := interp.GetCommand("run")
	if !ok {
		t.Fatal("bootstrap/main.glyph did not register '@ command run'")
	}

	// Create a temp file with a function
	tmpDir := t.TempDir()
	fibFile := filepath.Join(tmpDir, "fib.glyph")
	fibSrc := `! fib(n: int) -> int {
  if n <= 1 { > n }
  > fib(n - 1) + fib(n - 2)
}
$ result = fib(10)`
	if err := os.WriteFile(fibFile, []byte(fibSrc), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	result, err := interp.ExecuteCommand(&cmd, map[string]interface{}{
		"file": fibFile,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand run: %v", err)
	}

	resMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T: %v", result, result)
	}

	if okVal, _ := resMap["ok"].(bool); !okVal {
		t.Fatalf("standalone execution FAILED — result: %+v", resMap)
	}

	t.Logf("Standalone fib execution result: %+v", resMap)
}
