package main

import (
	"fmt"
	"syscall/js"
	"encoding/json"

	"github.com/glyphlang/glyph/pkg/parser"
	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/vm"
)

func runGlyph(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Error: Source code required"
	}
	source := args[0].String()

	// 1. Parse
	lexer := parser.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Sprintf("Lexer Error: %v", err)
	}

	p := parser.NewParser(tokens)
	module, err := p.Parse()
	if err != nil {
		return fmt.Sprintf("Parser Error: %v", err)
	}

	// 2. Compile
	c := compiler.NewCompiler()
	bytecode, err := c.Compile(module)
	if err != nil {
		return fmt.Sprintf("Compiler Error: %v", err)
	}

	// 3. Execute
	vmInstance := vm.NewVM()
	result, err := vmInstance.Execute(bytecode)
	if err != nil {
		return fmt.Sprintf("Execution Error: %v", err)
	}

	// 4. Format Result
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("Marshal Error: %v", err)
	}

	return string(jsonBytes)
}

func main() {
	fmt.Println("GlyphLang WASM initialized")
	js.Global().Set("runGlyph", js.FuncOf(runGlyph))
	
	// Keep the Go program running
	select {}
}
