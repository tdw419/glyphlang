package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/parser"
	"github.com/glyphlang/glyph/pkg/vm"
)

// CompileResult is returned by the compile function.
type CompileResult struct {
	Success  bool   `json:"success"`
	Bytecode []byte `json:"bytecode,omitempty"`
	Error    string `json:"error,omitempty"`
	Size     int    `json:"size,omitempty"`
	TimeMS   int64  `json:"time_ms,omitempty"`
}

// RunResult is returned by the run function.
type RunResult struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	TimeMS  int64       `json:"time_ms,omitempty"`
}

// ParseResult is returned by the parse function.
type ParseResult struct {
	Success bool        `json:"success"`
	AST     interface{} `json:"ast,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// bytecodeCache stores compiled bytecode for reuse.
var bytecodeCache = make(map[string][]byte)

// compileGlyph compiles source to bytecode.
func compileGlyph(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return mustMarshal(CompileResult{Success: false, Error: "source code required"})
	}

	source := args[0].String()
	start := time.Now()

	// Parse
	lexer := parser.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return mustMarshal(CompileResult{Success: false, Error: fmt.Sprintf("lexer: %v", err)})
	}

	p := parser.NewParser(tokens)
	module, err := p.Parse()
	if err != nil {
		return mustMarshal(CompileResult{Success: false, Error: fmt.Sprintf("parser: %v", err)})
	}

	// Compile
	c := compiler.NewCompiler()
	bytecode, err := c.Compile(module)
	if err != nil {
		return mustMarshal(CompileResult{Success: false, Error: fmt.Sprintf("compiler: %v", err)})
	}

	// Cache it
	cacheKey := fmt.Sprintf("%x", len(source))
	bytecodeCache[cacheKey] = bytecode

	return mustMarshal(CompileResult{
		Success:  true,
		Bytecode: bytecode,
		Size:     len(bytecode),
		TimeMS:   time.Since(start).Milliseconds(),
	})
}

// runGlyph executes source code and returns the result.
func runGlyph(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return mustMarshal(RunResult{Success: false, Error: "source code required"})
	}

	source := args[0].String()
	start := time.Now()

	// Check cache first
	cacheKey := fmt.Sprintf("%x", len(source))
	bytecode, cached := bytecodeCache[cacheKey]

	if !cached {
		// Compile fresh
		lexer := parser.NewLexer(source)
		tokens, err := lexer.Tokenize()
		if err != nil {
			return mustMarshal(RunResult{Success: false, Error: fmt.Sprintf("lexer: %v", err)})
		}

		p := parser.NewParser(tokens)
		module, err := p.Parse()
		if err != nil {
			return mustMarshal(RunResult{Success: false, Error: fmt.Sprintf("parser: %v", err)})
		}

		c := compiler.NewCompiler()
		bytecode, err = c.Compile(module)
		if err != nil {
			return mustMarshal(RunResult{Success: false, Error: fmt.Sprintf("compiler: %v", err)})
		}
	}

	// Execute
	vmInstance := vm.NewVM()
	result, err := vmInstance.Execute(bytecode)
	if err != nil {
		return mustMarshal(RunResult{Success: false, Error: fmt.Sprintf("execution: %v", err)})
	}

	return mustMarshal(RunResult{
		Success: true,
		Result:  valueToJSON(result),
		TimeMS:  time.Since(start).Milliseconds(),
	})
}

// runBytecode executes pre-compiled bytecode.
func runBytecode(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return mustMarshal(RunResult{Success: false, Error: "bytecode required"})
	}

	// Get bytecode from Uint8Array
	bytecodeArray := args[0]
	bytecode := make([]byte, bytecodeArray.Get("length").Int())
	for i := 0; i < len(bytecode); i++ {
		bytecode[i] = byte(bytecodeArray.Call("at", i).Int())
	}

	start := time.Now()

	vmInstance := vm.NewVM()
	result, err := vmInstance.Execute(bytecode)
	if err != nil {
		return mustMarshal(RunResult{Success: false, Error: fmt.Sprintf("execution: %v", err)})
	}

	return mustMarshal(RunResult{
		Success: true,
		Result:  valueToJSON(result),
		TimeMS:  time.Since(start).Milliseconds(),
	})
}

// parseGlyph parses source and returns the AST.
func parseGlyph(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return mustMarshal(ParseResult{Success: false, Error: "source code required"})
	}

	source := args[0].String()

	lexer := parser.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return mustMarshal(ParseResult{Success: false, Error: fmt.Sprintf("lexer: %v", err)})
	}

	p := parser.NewParser(tokens)
	module, err := p.Parse()
	if err != nil {
		return mustMarshal(ParseResult{Success: false, Error: fmt.Sprintf("parser: %v", err)})
	}

	return mustMarshal(ParseResult{Success: true, AST: moduleToString(module)})
}

// tokenize returns tokens for source.
func tokenize(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "[]"
	}

	source := args[0].String()

	lexer := parser.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return mustMarshal(map[string]interface{}{"success": false, "error": err.Error()})
	}

	// Convert tokens to JSON-friendly format
	result := make([]map[string]interface{}, len(tokens))
	for i, t := range tokens {
		result[i] = map[string]interface{}{
			"type":    t.Type.String(),
			"literal": t.Literal,
			"line":    t.Line,
			"column":  t.Column,
		}
	}

	return mustMarshal(map[string]interface{}{"success": true, "tokens": result})
}

// version returns the GlyphLang version.
func version(this js.Value, args []js.Value) interface{} {
	return "0.6.0-alpha"
}

// mustMarshal converts a value to JSON string.
func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"success":false,"error":"marshal: %v"}`, err)
	}
	return string(b)
}

// valueToJSON converts vm.Value to JSON-friendly format.
func valueToJSON(v vm.Value) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case vm.NullValue:
		return nil
	case vm.IntValue:
		return val.Val
	case vm.FloatValue:
		return val.Val
	case vm.StringValue:
		return val.Val
	case vm.BoolValue:
		return val.Val
	case vm.ArrayValue:
		arr := make([]interface{}, len(val.Val))
		for i, e := range val.Val {
			arr[i] = valueToJSON(e)
		}
		return arr
	case vm.ObjectValue:
		obj := make(map[string]interface{})
		for k, e := range val.Val {
			obj[k] = valueToJSON(e)
		}
		return obj
	default:
		return fmt.Sprintf("<%T>", v)
	}
}

// moduleToString creates a simple string representation of the AST.
func moduleToString(m *ast.Module) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  \"items\": [\n")
	for i, item := range m.Items {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(fmt.Sprintf("    %q", item))
	}
	b.WriteString("\n  ]\n")
	b.WriteString("}")
	return b.String()
}

func main() {
	fmt.Println("GlyphLang WASM v0.6.0-alpha initialized")

	// Register functions
	js.Global().Set("glyphCompile", js.FuncOf(compileGlyph))
	js.Global().Set("glyphRun", js.FuncOf(runGlyph))
	js.Global().Set("glyphRunBytecode", js.FuncOf(runBytecode))
	js.Global().Set("glyphParse", js.FuncOf(parseGlyph))
	js.Global().Set("glyphTokenize", js.FuncOf(tokenize))
	js.Global().Set("glyphVersion", js.FuncOf(version))

	// Keep running
	select {}
}
