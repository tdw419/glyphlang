package main
import (
	"fmt"
	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/parser"
)
func main() {
	src := "! main() {\n  $ i = 0\n  $ sum = 0\n  while i < 5 {\n    $ sum = sum + i\n    $ i = i + 1\n  }\n  > sum\n}"
	lexer := parser.NewLexer(src)
	tokens, err := lexer.Tokenize()
	if err != nil { panic(err) }
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil { panic(err) }
	
	c := compiler.NewSSACompiler()
	var fn *ast.Function
	for _, item := range mod.Items {
		if f, ok := item.(*ast.Function); ok {
			fn = f
			break
		}
	}
	bc, err := c.CompileFunction(fn)
	if err != nil { panic(err) }
	fmt.Printf("%x\n", bc)
}