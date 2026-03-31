//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/parser"
	"github.com/glyphlang/glyph/pkg/vm"
)

// BrowserSpatialResult is the JSON contract returned to the browser
// by glyphSpatialExec. Duplicated here (without js.Value dependency)
// so the mock browser can return typed results for testing.
type BrowserSpatialResult struct {
	Success   bool        `json:"success"`
	Result    interface{} `json:"result,omitempty"`
	Threads   int         `json:"threads,omitempty"`
	Mutations int         `json:"mutations,omitempty"`
	Error     string      `json:"error,omitempty"`
	TimeMS    int64       `json:"time_ms,omitempty"`
}

// BrowserCapabilities mirrors what glyphSpatialCapabilities returns to JS.
type BrowserCapabilities struct {
	Mitosis bool           `json:"mitosis"`
	Mutator bool           `json:"mutator"`
	Version string         `json:"version"`
	Opcodes map[string]int `json:"opcodes"`
}

// MockBrowser simulates the JavaScript↔WASM bridge for spatial opcode
// operations. It wraps the same parse→compile→scan→execute pipeline
// that main.go's glyphSpatialExec uses, but without syscall/js,
// so it can run in ordinary Go tests.
type MockBrowser struct {
	bytecodeCache map[string][]byte
}

// NewMockBrowser creates a fresh browser simulation environment.
func NewMockBrowser() *MockBrowser {
	return &MockBrowser{bytecodeCache: make(map[string][]byte)}
}

// Compile parses and compiles Glyph source, caching the result.
// Mirrors the glyphCompile JS binding.
func (mb *MockBrowser) Compile(source string) ([]byte, error) {
	lexer := parser.NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer: %v", err)
	}
	p := parser.NewParser(tokens)
	module, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parser: %v", err)
	}
	c := compiler.NewCompiler()
	bytecode, err := c.Compile(module)
	if err != nil {
		return nil, fmt.Errorf("compiler: %v", err)
	}
	key := fmt.Sprintf("%x", len(source))
	mb.bytecodeCache[key] = bytecode
	return bytecode, nil
}

// Run executes cached or freshly-compiled bytecode.
// Mirrors the glyphRun JS binding.
func (mb *MockBrowser) Run(source string) (vm.Value, error) {
	key := fmt.Sprintf("%x", len(source))
	bytecode, cached := mb.bytecodeCache[key]
	if !cached {
		var err error
		bytecode, err = mb.Compile(source)
		if err != nil {
			return nil, err
		}
	}
	v := vm.NewVM()
	return v.Execute(bytecode)
}

// RunBytecode executes raw bytecode directly.
// Mirrors the glyphRunBytecode JS binding.
func (mb *MockBrowser) RunBytecode(bytecode []byte) (vm.Value, error) {
	v := vm.NewVM()
	return v.Execute(bytecode)
}

// SpatialExec runs source through the full spatial pipeline:
// parse → compile → scan for spatial opcodes → execute → diagnostics.
// Mirrors the glyphSpatialExec JS binding.
func (mb *MockBrowser) SpatialExec(source string) BrowserSpatialResult {
	start := time.Now()

	bytecode, err := mb.Compile(source)
	if err != nil {
		return BrowserSpatialResult{
			Success: false,
			Error:   err.Error(),
			TimeMS:  time.Since(start).Milliseconds(),
		}
	}

	mutations, threads := CountSpatialOpcodes(bytecode)

	v := vm.NewVM()
	result, execErr := v.Execute(bytecode)

	if execErr != nil {
		return BrowserSpatialResult{
			Success:   false,
			Threads:   threads,
			Mutations: mutations,
			Error:     fmt.Sprintf("execution: %v", execErr),
			TimeMS:    time.Since(start).Milliseconds(),
		}
	}

	// Cache with spatial key
	spKey := fmt.Sprintf("spatial_%x", len(source))
	mb.bytecodeCache[spKey] = bytecode

	return BrowserSpatialResult{
		Success:   true,
		Result:    result,
		Threads:   threads,
		Mutations: mutations,
		TimeMS:    time.Since(start).Milliseconds(),
	}
}

// SpatialCapabilities reports which spatial opcodes the environment supports.
// Mirrors the glyphSpatialCapabilities JS binding.
func (mb *MockBrowser) SpatialCapabilities() BrowserCapabilities {
	return BrowserCapabilities{
		Mitosis: true,
		Mutator: true,
		Version: "0.6.0-alpha",
		Opcodes: map[string]int{
			"OP_MITOSIS": int(vm.OpMitosis),
			"OP_MUTATOR": int(vm.OpMutator),
		},
	}
}

// SpatialCapabilitiesJSON returns capabilities as JSON (wire format).
func (mb *MockBrowser) SpatialCapabilitiesJSON() string {
	caps := mb.SpatialCapabilities()
	b, _ := json.Marshal(caps)
	return string(b)
}

// CountSpatialOpcodes scans bytecode for spatial opcode bytes and
// returns the mutation count and total thread count (starts at 1).
func CountSpatialOpcodes(bytecode []byte) (mutations, threads int) {
	threads = 1
	for _, b := range bytecode {
		switch b {
		case byte(vm.OpMutator):
			mutations++
		case byte(vm.OpMitosis):
			threads++
		}
	}
	return
}
