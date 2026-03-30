package interpreter

import (
	. "github.com/glyphlang/glyph/pkg/ast"
	"os"
	"testing"

	"github.com/glyphlang/glyph/pkg/parser"
)

func TestStandaloneImportDebug(t *testing.T) {
	rootDir := "../../bootstrap"
	mainSrc, err := os.ReadFile(rootDir + "/main.glyph")
	if err != nil {
		t.Fatalf("read main.glyph: %v", err)
	}

	lex := parser.NewLexer(string(mainSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	t.Logf("main.glyph has %d items", len(mod.Items))
	for i, item := range mod.Items {
		t.Logf("  [%d] %T", i, item)
		if c, ok := item.(*Command); ok {
			t.Logf("      Command: %s", c.Name)
		}
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
	if err := interp.LoadModuleWithPath(*mod, rootDir+"/"); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	t.Logf("Commands: %v", func() []string {
		var names []string
		for k := range interp.commands {
			names = append(names, k)
		}
		return names
	}())
}
