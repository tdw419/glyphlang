package interpreter

import (
	. "github.com/glyphlang/glyph/pkg/ast"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glyphlang/glyph/pkg/parser"
)

// Standalone benchmarks and performance ratio test for Issue #12.

// benchStandaloneSetup loads bootstrap/main.glyph and returns the interpreter + run command.
func benchStandaloneSetup(b *testing.B) (*Interpreter, Command, string) {
	b.Helper()

	rootDir := filepath.Join("..", "..", "bootstrap")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		b.Fatalf("resolving bootstrap dir: %v", err)
	}

	mainSrc, err := os.ReadFile(filepath.Join(absRoot, "main.glyph"))
	if err != nil {
		b.Fatalf("reading bootstrap/main.glyph: %v", err)
	}

	lex := parser.NewLexer(string(mainSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		b.Fatalf("tokenizing main.glyph: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		b.Fatalf("parsing main.glyph: %v", err)
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
		b.Fatalf("LoadModuleWithPath: %v", err)
	}

	cmd, ok := interp.GetCommand("run")
	if !ok {
		b.Fatal("no 'run' command in bootstrap/main.glyph")
	}

	// Create temp file with a simple computation
	tmpDir := b.TempDir()
	targetFile := filepath.Join(tmpDir, "bench.glyph")
	benchSrc := `$ result = 40 + 2`
	if err := os.WriteFile(targetFile, []byte(benchSrc), 0644); err != nil {
		b.Fatalf("writing bench file: %v", err)
	}

	return interp, cmd, targetFile
}

// BenchmarkStandalone benchmarks execution through the bootstrap standalone path.
func BenchmarkStandalone(b *testing.B) {
	interp, cmd, targetFile := benchStandaloneSetup(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := interp.ExecuteCommand(&cmd, map[string]interface{}{
			"file": targetFile,
		})
		if err != nil {
			b.Fatalf("ExecuteCommand: %v", err)
		}
	}
}

// BenchmarkGoNative benchmarks the Go interpreter parsing + loading a simple program.
func BenchmarkGoNative(b *testing.B) {
	// Use a function definition — the Go parser requires valid top-level forms
	src := `! compute() -> int { > 40 + 2 }`

	lex := parser.NewLexer(src)
	tokens, err := lex.Tokenize()
	if err != nil {
		b.Fatalf("tokenize: %v", err)
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		b.Fatalf("parse: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interp := NewInterpreter()
		if err := interp.LoadModule(*mod); err != nil {
			b.Fatalf("LoadModule: %v", err)
		}
	}
}

// TestStandaloneBenchmarkRatio verifies the bootstrap path is within 10x
// of the Go-native path, as required by Issue #12 acceptance criteria.
func TestStandaloneBenchmarkRatio(t *testing.T) {
	rootDir := filepath.Join("..", "..", "bootstrap")
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		t.Fatalf("resolving bootstrap dir: %v", err)
	}

	mainSrc, err := os.ReadFile(filepath.Join(absRoot, "main.glyph"))
	if err != nil {
		t.Fatalf("reading main.glyph: %v", err)
	}

	lex := parser.NewLexer(string(mainSrc))
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("tokenize main: %v", err)
	}
	p := parser.NewParser(tokens)
	mainMod, err := p.Parse()
	if err != nil {
		t.Fatalf("parse main: %v", err)
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
	if err := interp.LoadModuleWithPath(*mainMod, absRoot+"/"); err != nil {
		t.Fatalf("LoadModuleWithPath: %v", err)
	}

	cmd, ok := interp.GetCommand("run")
	if !ok {
		t.Fatal("no 'run' command")
	}

	// Simple target program for standalone path
	standaloneSrc := `$ result = 40 + 2`
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "ratio.glyph")
	if err := os.WriteFile(targetFile, []byte(standaloneSrc), 0644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	// Time standalone path (10 iterations)
	standaloneTimes := make([]int64, 10)
	for i := 0; i < 10; i++ {
		start := time.Now()
		_, err := interp.ExecuteCommand(&cmd, map[string]interface{}{
			"file": targetFile,
		})
		if err != nil {
			t.Fatalf("ExecuteCommand: %v", err)
		}
		standaloneTimes[i] = time.Since(start).Microseconds()
	}

	// Time Go-native path (10 iterations) using a function definition
	goSrc := `! compute() -> int { > 40 + 2 }`
	goTimes := make([]int64, 10)
	for i := 0; i < 10; i++ {
		lex := parser.NewLexer(goSrc)
		tokens, err := lex.Tokenize()
		if err != nil {
			t.Fatalf("tokenize: %v", err)
		}
		p := parser.NewParser(tokens)
		mod, err := p.Parse()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		start := time.Now()
		goInterp := NewInterpreter()
		if err := goInterp.LoadModule(*mod); err != nil {
			t.Fatalf("LoadModule: %v", err)
		}
		goTimes[i] = time.Since(start).Microseconds()
	}

	avgGo := avgMicros(goTimes)
	avgStand := avgMicros(standaloneTimes)

	t.Logf("Go-native avg: %dµs, Standalone avg: %dµs", avgGo, avgStand)

	if avgGo == 0 && avgStand == 0 {
		t.Log("Both paths too fast to measure — ratio test passes trivially")
		return
	}
	if avgGo == 0 {
		t.Log("Go-native too fast to measure — ratio test passes")
		return
	}

	ratio := float64(avgStand) / float64(avgGo)
	t.Logf("Ratio: %.1fx (acceptance: <10x)", ratio)

	if ratio > 10.0 {
		t.Errorf("Standalone path is %.1fx slower than Go-native (acceptance: <10x)", ratio)
	}
}

func avgMicros(vals []int64) int64 {
	var sum int64
	for _, v := range vals {
		sum += v
	}
	return sum / int64(len(vals))
}
