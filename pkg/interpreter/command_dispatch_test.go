package interpreter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glyphlang/glyph/pkg/parser"
)

// TestCommandDispatch tests that @ command definitions are discovered and can
// be dispatched via GetCommand / ExecuteCommand — the core of Issue #9.
func TestCommandDispatch(t *testing.T) {
	// Create a temp .glyph file with two commands
	src := `
@ command greet name: str! {
  > "Hello, " + name + "!"
}

@ command echo msg: str! {
  > msg
}
`
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dispatch_test.glyph")
	if err := os.WriteFile(filePath, []byte(src), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Parse the source
	lex := parser.NewLexer(src)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Load module into interpreter
	interp := NewInterpreter()
	if err := interp.LoadModuleWithPath(*mod, tmpDir); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	// Verify both commands registered
	commands := interp.GetCommands()
	if len(commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(commands))
	}

	// Dispatch "greet" command
	greetCmd, ok := interp.GetCommand("greet")
	if !ok {
		t.Fatal("greet command not found")
	}
	result, err := interp.ExecuteCommand(&greetCmd, map[string]interface{}{"name": "World"})
	if err != nil {
		t.Fatalf("ExecuteCommand greet: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("greet result = %v, want 'Hello, World!'", result)
	}

	// Dispatch "echo" command — confirms multi-command dispatch
	echoCmd, ok := interp.GetCommand("echo")
	if !ok {
		t.Fatal("echo command not found")
	}
	result2, err := interp.ExecuteCommand(&echoCmd, map[string]interface{}{"msg": "dispatched"})
	if err != nil {
		t.Fatalf("ExecuteCommand echo: %v", err)
	}
	if result2 != "dispatched" {
		t.Errorf("echo result = %v, want 'dispatched'", result2)
	}
}

// TestCommandDispatch_ListCommands verifies that GetCommands lists all
// registered command names.
func TestCommandDispatch_ListCommands(t *testing.T) {
	src := `
@ command alpha {
  > "alpha"
}

@ command beta {
  > "beta"
}

@ command gamma {
  > "gamma"
}
`
	lex := parser.NewLexer(src)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	interp := NewInterpreter()
	if err := interp.LoadModuleWithPath(*mod, "."); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	cmds := interp.GetCommands()
	wantNames := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for name := range wantNames {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command: %s", name)
		}
	}
	if len(cmds) < len(wantNames) {
		t.Errorf("GetCommands returned %d commands, want at least %d", len(cmds), len(wantNames))
	}
}

// TestCommandDispatch_UnknownCommand verifies that requesting an unknown
// command returns ok=false.
func TestCommandDispatch_UnknownCommand(t *testing.T) {
	src := `
@ command exists {
  > "yes"
}
`
	lex := parser.NewLexer(src)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	interp := NewInterpreter()
	if err := interp.LoadModuleWithPath(*mod, "."); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	_, ok := interp.GetCommand("nonexistent")
	if ok {
		t.Error("GetCommand should return false for unknown command")
	}
}

// toInt converts a numeric value to int for comparison
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
