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
