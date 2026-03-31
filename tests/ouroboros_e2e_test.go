package tests

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/interpreter"
	"github.com/glyphlang/glyph/pkg/parser"
)

// parseAndExecute loads GlyphLang source, creates an interpreter, loads the
// module, executes the "run" command, and captures stdout.
func parseAndExecute(source string) (string, error) {
	lex := parser.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		return "", err
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		return "", err
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	interp := interpreter.NewInterpreter()
	interp.GetModuleResolver().ParseFunc = func(src string) (*ast.Module, error) {
		lex := parser.NewLexer(src)
		tok, err := lex.Tokenize()
		if err != nil {
			return nil, err
		}
		p := parser.NewParser(tok)
		return p.Parse()
	}

	loadErr := interp.LoadModule(*mod)

	var execErr error
	if loadErr == nil {
		if cmd, ok := interp.GetCommand("run"); ok {
			_, execErr = interp.ExecuteCommand(&cmd, nil)
		}
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if loadErr != nil {
		return buf.String(), loadErr
	}
	return buf.String(), execErr
}

// TestOuroborosExampleE2E loads and executes examples/ouroboros.glyph
// through the interpreter pipeline, verifying the self-modification output
// meets the acceptance criteria from issue #21.
func TestOuroborosExampleE2E(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "examples", "ouroboros.glyph"))
	if err != nil {
		t.Fatalf("failed to read ouroboros.glyph: %v", err)
	}

	output, err := parseAndExecute(string(source))
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Acceptance: output demonstrates code executed which did not exist at compile time
	expected := []string{
		"Ouroboros",
		"is_parent = true",
		"SELF-MODIFICATION CONFIRMED",
	}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("output missing %q\ngot:\n%s", exp, output)
		}
	}
}
