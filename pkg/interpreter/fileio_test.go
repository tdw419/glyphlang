package interpreter

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── File I/O Edge Cases (Issue #10) ──────────────────────────────

func TestFileIO_WriteByteArray_ReadBack(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "bytes.bin")

	// Write byte array [72, 101, 108, 108, 111] = "Hello"
	writeExpr := FunctionCallExpr{
		Name: "writeFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
			ArrayExpr{Elements: []Expr{
				LiteralExpr{Value: IntLiteral{Value: 72}},
				LiteralExpr{Value: IntLiteral{Value: 101}},
				LiteralExpr{Value: IntLiteral{Value: 108}},
				LiteralExpr{Value: IntLiteral{Value: 108}},
				LiteralExpr{Value: IntLiteral{Value: 111}},
			}},
		},
	}
	result, err := interp.evaluateFunctionCall(writeExpr, env)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Read back — should get "Hello" as a string
	readExpr := FunctionCallExpr{
		Name: "readFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
		},
	}
	result, err = interp.evaluateFunctionCall(readExpr, env)
	require.NoError(t, err)
	assert.Equal(t, "Hello", result)
}

func TestFileIO_WriteFileOverwrite(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "overwrite.txt")

	// Write first content
	write1 := FunctionCallExpr{
		Name: "writeFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
			LiteralExpr{Value: StringLiteral{Value: "first"}},
		},
	}
	_, err := interp.evaluateFunctionCall(write1, env)
	require.NoError(t, err)

	// Overwrite with second content
	write2 := FunctionCallExpr{
		Name: "writeFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
			LiteralExpr{Value: StringLiteral{Value: "second"}},
		},
	}
	_, err = interp.evaluateFunctionCall(write2, env)
	require.NoError(t, err)

	// Read back — should be "second"
	readExpr := FunctionCallExpr{
		Name: "readFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
		},
	}
	result, err := interp.evaluateFunctionCall(readExpr, env)
	require.NoError(t, err)
	assert.Equal(t, "second", result)
}

func TestFileIO_FullPipeline(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "pipeline.txt")
	content := "line1\nline2\nline3"

	// Step 1: File should not exist yet
	existsExpr := FunctionCallExpr{
		Name: "exists",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
		},
	}
	result, err := interp.evaluateFunctionCall(existsExpr, env)
	require.NoError(t, err)
	assert.Equal(t, false, result)

	// Step 2: Write file
	writeExpr := FunctionCallExpr{
		Name: "writeFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
			LiteralExpr{Value: StringLiteral{Value: content}},
		},
	}
	_, err = interp.evaluateFunctionCall(writeExpr, env)
	require.NoError(t, err)

	// Step 3: File should now exist
	result, err = interp.evaluateFunctionCall(existsExpr, env)
	require.NoError(t, err)
	assert.Equal(t, true, result)

	// Step 4: Read back and verify
	readExpr := FunctionCallExpr{
		Name: "readFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
		},
	}
	result, err = interp.evaluateFunctionCall(readExpr, env)
	require.NoError(t, err)
	assert.Equal(t, content, result)

	// Step 5: Delete file externally, verify exists returns false
	os.Remove(filePath)
	result, err = interp.evaluateFunctionCall(existsExpr, env)
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestFileIO_ExistsDirectory(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	tmpDir := t.TempDir()

	// A directory should also return true for exists()
	expr := FunctionCallExpr{
		Name: "exists",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: tmpDir}},
		},
	}
	result, err := interp.evaluateFunctionCall(expr, env)
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestFileIO_WriteFileMultilineContent(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "multiline.txt")
	content := "first line\nsecond line\nthird line\n"

	writeExpr := FunctionCallExpr{
		Name: "writeFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
			LiteralExpr{Value: StringLiteral{Value: content}},
		},
	}
	_, err := interp.evaluateFunctionCall(writeExpr, env)
	require.NoError(t, err)

	readExpr := FunctionCallExpr{
		Name: "readFile",
		Args: []Expr{
			LiteralExpr{Value: StringLiteral{Value: filePath}},
		},
	}
	result, err := interp.evaluateFunctionCall(readExpr, env)
	require.NoError(t, err)
	assert.Equal(t, content, result)
}

// ─── Error Cases ──────────────────────────────────────────────────

func TestFileIO_WriteFileErrors(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	t.Run("wrong_arg_count_zero", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "writeFile",
			Args: []Expr{},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects 2 arguments")
	})

	t.Run("wrong_arg_count_one", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "writeFile",
			Args: []Expr{
				LiteralExpr{Value: StringLiteral{Value: "path.txt"}},
			},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects 2 arguments")
	})

	t.Run("non_string_path", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "writeFile",
			Args: []Expr{
				LiteralExpr{Value: IntLiteral{Value: 42}},
				LiteralExpr{Value: StringLiteral{Value: "data"}},
			},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects first argument to be a string path")
	})

	t.Run("non_string_non_array_data", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "writeFile",
			Args: []Expr{
				LiteralExpr{Value: StringLiteral{Value: filepath.Join(t.TempDir(), "test.txt")}},
				LiteralExpr{Value: IntLiteral{Value: 99}},
			},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects second argument to be a string or array of bytes")
	})
}

func TestFileIO_ReadFileErrors(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	t.Run("wrong_arg_count_zero", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "readFile",
			Args: []Expr{},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects 1 argument")
	})

	t.Run("non_string_path", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "readFile",
			Args: []Expr{
				LiteralExpr{Value: IntLiteral{Value: 42}},
			},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects argument to be a string path")
	})
}

func TestFileIO_ExistsErrors(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	t.Run("wrong_arg_count", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "exists",
			Args: []Expr{},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects 1 argument")
	})

	t.Run("non_string_path", func(t *testing.T) {
		expr := FunctionCallExpr{
			Name: "exists",
			Args: []Expr{
				LiteralExpr{Value: IntLiteral{Value: 42}},
			},
		}
		_, err := interp.evaluateFunctionCall(expr, env)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects argument to be a string path")
	})
}

func TestFileIO_ArgsLength(t *testing.T) {
	interp := NewInterpreter()
	env := NewEnvironment()

	origArgs := os.Args
	os.Args = []string{"glyph", "run", "script.glyph", "hello", "world"}
	defer func() { os.Args = origArgs }()

	expr := FunctionCallExpr{
		Name: "args",
		Args: []Expr{},
	}
	result, err := interp.evaluateFunctionCall(expr, env)
	require.NoError(t, err)

	arr, ok := result.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 5)
	assert.Equal(t, "glyph", arr[0])
	assert.Equal(t, "run", arr[1])
	assert.Equal(t, "script.glyph", arr[2])
	assert.Equal(t, "hello", arr[3])
	assert.Equal(t, "world", arr[4])
}

// ─── SEC-1: readFile from .glyph source (full parser pipeline) ─────

// parseGlyph is a test helper that lexes, tokenizes, and parses .glyph source
// into a Module AST.
func parseGlyph(t *testing.T, src string) *Module {
	t.Helper()
	lex := parser.NewLexer(src)
	tokens, err := lex.Tokenize()
	require.NoError(t, err, "lexer failed")
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	require.NoError(t, err, "parser failed")
	return mod
}

// TestReadFile_FromGlyphSource verifies that readFile("test.glyph") called
// from parsed .glyph source code returns the file contents as a string.
// This exercises the full lexer -> parser -> evaluator pipeline (SEC-1 step 1.2).
func TestReadFile_FromGlyphSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .glyph file with known content to be read
	targetFile := filepath.Join(tmpDir, "test.glyph")
	targetContent := "// hello from glyph\n$ x = 42"
	require.NoError(t, os.WriteFile(targetFile, []byte(targetContent), 0644))

	// Define the file path in the environment so the .glyph function can reference it
	interp := NewInterpreter()
	env := interp.globalEnv
	env.Define("__targetPath", targetFile)

	// Parse a .glyph program that defines a function calling readFile
	src := `! loadFile() -> string {
  > readFile(__targetPath)
}`
	mod := parseGlyph(t, src)
	require.NoError(t, interp.LoadModule(*mod))

	// Call the parsed function through the evaluator
	result, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "loadFile",
		Args: []Expr{},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, targetContent, result, "readFile should return exact file contents")
}

// TestReadFile_NonexistentFromGlyphSource verifies that readFile on a
// nonexistent file returns an empty string when called from .glyph source.
func TestReadFile_NonexistentFromGlyphSource(t *testing.T) {
	interp := NewInterpreter()
	env := interp.globalEnv
	env.Define("__badPath", "/tmp/glyphlang_no_such_file_xyz.txt")

	src := `! loadMissing() -> string {
  > readFile(__badPath)
}`
	mod := parseGlyph(t, src)
	require.NoError(t, interp.LoadModule(*mod))

	result, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "loadMissing",
		Args: []Expr{},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, "", result, "readFile of nonexistent file should return empty string")
}

// TestReadFile_WriteThenReadFromGlyphSource verifies the writeFile -> readFile
// round-trip through parsed .glyph source.
func TestReadFile_WriteThenReadFromGlyphSource(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "data.txt")
	content := "glyphlang file I/O test content"

	interp := NewInterpreter()
	env := interp.globalEnv
	env.Define("__filePath", filePath)
	env.Define("__content", content)

	src := `! writeData() {
  $ _ = writeFile(__filePath, __content)
}
! readData() -> string {
  > readFile(__filePath)
}`
	mod := parseGlyph(t, src)
	require.NoError(t, interp.LoadModule(*mod))

	// Write the file via the parsed function
	_, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "writeData",
		Args: []Expr{},
	}, env)
	require.NoError(t, err)

	// Read it back via the parsed function
	result, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "readData",
		Args: []Expr{},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, content, result)
}

// ─── SEC-2: Import path resolution and module loading ────────────────

// TestImport_ModuleResolution verifies the full import pipeline:
// main.glyph imports "./helper", calls helper.add(1, 2), result is 3.
// This exercises: parse import statement → resolve path → readFile →
// parse imported source → execute in module scope → bind exports as
// namespace → call namespace.method().
func TestImport_ModuleResolution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create helper.glyph with an exported add function
	helperSrc := `! add(a: int, b: int) -> int {
  > a + b
}`
	helperFile := filepath.Join(tmpDir, "helper.glyph")
	require.NoError(t, os.WriteFile(helperFile, []byte(helperSrc), 0644))

	// Create main.glyph that imports helper and defines a run function
	mainSrc := `import "./helper"
! run() -> int {
  > helper.add(1, 2)
}`
	mainFile := filepath.Join(tmpDir, "main.glyph")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainSrc), 0644))

	// Parse main.glyph
	mainMod := parseGlyph(t, mainSrc)

	// Set up interpreter with ParseFunc wired to the parser
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

	// Load main module with the temp dir as base path for imports
	require.NoError(t, interp.LoadModuleWithPath(*mainMod, tmpDir))

	// Call the run function and verify result
	env := interp.globalEnv
	result, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "run",
		Args: []Expr{},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result, "helper.add(1, 2) should return 3")
}

// ─── SEC-3: Circular import guard and module cache ─────────────────

// TestCircularImport_NoInfiniteLoop creates two .glyph files that import
// each other (A imports B, B imports A) and verifies the interpreter
// terminates without infinite recursion. The second (circular) import
// should return the partially-initialised module from cache rather than
// recursing. Both modules should load successfully and their own
// functions must remain callable.
func TestCircularImport_NoInfiniteLoop(t *testing.T) {
	tmpDir := t.TempDir()

	// alpha.glyph imports ./beta and exports an identity function
	alphaSrc := `import "./beta"
! alphaFn(x: int) -> int {
  > x
}`
	alphaFile := filepath.Join(tmpDir, "alpha.glyph")
	require.NoError(t, os.WriteFile(alphaFile, []byte(alphaSrc), 0644))

	// beta.glyph imports ./alpha (circular) and exports its own function
	betaSrc := `import "./alpha"
! betaFn(x: int) -> int {
  > x + 1
}`
	betaFile := filepath.Join(tmpDir, "beta.glyph")
	require.NoError(t, os.WriteFile(betaFile, []byte(betaSrc), 0644))

	// Wire interpreter with real parser
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

	// Loading alpha must not hang or error
	require.NoError(t, interp.LoadModuleWithPath(*parseGlyph(t, alphaSrc), tmpDir))

	// alphaFn should be callable
	env := interp.globalEnv
	result, err := interp.EvaluateExpression(FunctionCallExpr{
		Name: "alphaFn",
		Args: []Expr{LiteralExpr{Value: IntLiteral{Value: 42}}},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result, "alphaFn(42) should return 42")

	// betaFn should also be loaded and callable
	result, err = interp.EvaluateExpression(FunctionCallExpr{
		Name: "betaFn",
		Args: []Expr{LiteralExpr{Value: IntLiteral{Value: 7}}},
	}, env)
	require.NoError(t, err)
	assert.Equal(t, int64(8), result, "betaFn(7) should return 8")
}

// TestCircularImport_CachePreventsReload verifies that importing the same
// module twice returns the cached entry on the second import (no re-parse).
func TestCircularImport_CachePreventsReload(t *testing.T) {
	tmpDir := t.TempDir()

	utilSrc := `! utilFn() -> int { > 99 }`
	utilFile := filepath.Join(tmpDir, "util.glyph")
	require.NoError(t, os.WriteFile(utilFile, []byte(utilSrc), 0644))

	mainSrc := `import "./util"
import "./util"
! run() -> int { > util.utilFn() }`
	mainFile := filepath.Join(tmpDir, "main.glyph")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainSrc), 0644))

	interp := NewInterpreter()
	parseCount := 0
	interp.moduleResolver.ParseFunc = func(source string) (*Module, error) {
		parseCount++
		lex := parser.NewLexer(source)
		tokens, err := lex.Tokenize()
		if err != nil {
			return nil, err
		}
		p := parser.NewParser(tokens)
		return p.Parse()
	}

	require.NoError(t, interp.LoadModuleWithPath(*parseGlyph(t, mainSrc), tmpDir))

	// util.glyph should be parsed exactly once (cached on second import)
	assert.Equal(t, 1, parseCount, "module should only be parsed once due to cache")
}
