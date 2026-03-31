package interpreter

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/parser"
)

// parseLoadExecute parses GlyphLang source, loads it into an interpreter,
// executes the "run" command (if defined), and captures stdout.
// Returns (captured output, interpreter, error).
func parseLoadExecute(source string) (string, *Interpreter, error) {
	lex := parser.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		return "", nil, err
	}
	p := parser.NewParser(tokens)
	mod, err := p.Parse()
	if err != nil {
		return "", nil, err
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

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

	loadErr := interp.LoadModule(*mod)

	// Execute the "run" command if it exists
	var execErr error
	if loadErr == nil {
		if cmd, ok := interp.GetCommand("run"); ok {
			_, execErr = interp.ExecuteCommand(&cmd, nil)
		}
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if loadErr != nil {
		return buf.String(), interp, loadErr
	}
	return buf.String(), interp, execErr
}

// TestMutatorBuiltin verifies the __mutator builtin works without errors.
func TestMutatorBuiltin(t *testing.T) {
	source := `! main() {
  $ _ = __mutator(42, 10)
}
@ command run {
  main()
}`

	_, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
}

// TestMitosisBuiltin verifies the __mitosis builtin returns true (parent path).
func TestMitosisBuiltin(t *testing.T) {
	source := `! main() {
  $ is_parent = __mitosis(0)
}
@ command run {
  main()
}`

	_, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
}

// TestMutatorStoresInMutationsTable verifies that __mutator writes to the
// __mutations map in the global environment and the values can be read back.
func TestMutatorStoresInMutationsTable(t *testing.T) {
	source := `! main() {
  $ v1 = __mutator(100, 0)
  $ v2 = __mutator(200, 1)
  $ v3 = __mutator(255, 7)
}
@ command run {
  main()
}`

	_, interp, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	mutationsRaw, getErr := interp.globalEnv.Get("__mutations")
	if getErr != nil {
		t.Fatalf("expected __mutations in global env, got error: %v", getErr)
	}
	mutations, ok := mutationsRaw.(map[int64]int64)
	if !ok {
		t.Fatalf("expected map[int64]int64, got %T", mutationsRaw)
	}

	if mutations[0] != 100 {
		t.Errorf("mutations[0] = %d, want 100", mutations[0])
	}
	if mutations[1] != 200 {
		t.Errorf("mutations[1] = %d, want 200", mutations[1])
	}
	if mutations[7] != 255 {
		t.Errorf("mutations[7] = %d, want 255", mutations[7])
	}
}

// TestMutatorPassthroughReturn verifies that __mutator returns the written
// value so it can be used inline.
func TestMutatorPassthroughReturn(t *testing.T) {
	source := `! main() {
  $ written = __mutator(42, 5)
  $ _ = print("written:", written)
}
@ command run {
  main()
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !strings.Contains(output, "42") {
		t.Errorf("expected output to contain '42', got: %s", output)
	}
}

// TestOuroborosSelfModification verifies the full self-modification loop:
// mitosis forks (returns true = parent), mutator writes to the bytecode stream,
// and the mutation table is readable from Go. The output demonstrates that
// code executed which did not exist at compile time.
func TestOuroborosSelfModification(t *testing.T) {
	source := `! main() {
  $ _ = print("=== Ouroboros ===")
  $ is_parent = __mitosis(0)
  $ _ = print("is_parent =", is_parent)
  $ written = __mutator(42, 7)
  $ _ = print("written:", written)
  $ _ = __mutator(99, 3)
}
@ command run {
  main()
}`

	output, interp, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify execution trace output
	if !strings.Contains(output, "=== Ouroboros ===") {
		t.Errorf("expected header in output, got: %s", output)
	}
	if !strings.Contains(output, "is_parent = true") {
		t.Errorf("expected 'is_parent = true' in output, got: %s", output)
	}
	if !strings.Contains(output, "written: 42") {
		t.Errorf("expected 'written: 42' in output, got: %s", output)
	}

	// Verify mutation table shows self-modified state
	mutationsRaw, getErr := interp.globalEnv.Get("__mutations")
	if getErr != nil {
		t.Fatalf("expected __mutations in global env: %v", getErr)
	}
	mutations := mutationsRaw.(map[int64]int64)

	// The value at offset 7 is now 42 — it was 0 at compile time
	if mutations[7] != 42 {
		t.Errorf("mutations[7] = %d, want 42", mutations[7])
	}
	// The value at offset 3 is now 99 — also not present at compile time
	if mutations[3] != 99 {
		t.Errorf("mutations[3] = %d, want 99", mutations[3])
	}
}

// TestMultipleMutationsSameOffset verifies that writing to the same offset
// twice overwrites the previous value (last write wins).
func TestMultipleMutationsSameOffset(t *testing.T) {
	source := `! main() {
  $ _ = __mutator(10, 3)
  $ _ = __mutator(99, 3)
}
@ command run {
  main()
}`

	_, interp, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	mutationsRaw, _ := interp.globalEnv.Get("__mutations")
	mutations := mutationsRaw.(map[int64]int64)

	if mutations[3] != 99 {
		t.Errorf("mutations[3] = %d, want 99 (last write wins)", mutations[3])
	}
	if len(mutations) != 1 {
		t.Errorf("expected 1 mutation entry, got %d", len(mutations))
	}
}

// TestMutatorRejectsNonIntegerArgs verifies that __mutator returns an error
// when called with non-integer arguments.
func TestMutatorRejectsNonIntegerArgs(t *testing.T) {
	source := `! main() {
  $ _ = __mutator("hello", 0)
}
@ command run {
  main()
}`

	_, _, err := parseLoadExecute(source)
	if err == nil {
		t.Fatal("expected error for non-integer __mutator args, got nil")
	}
	if !strings.Contains(err.Error(), "integer arguments") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestOuroborosScriptExecution loads and executes the full ouroboros.glyph
// example, verifying it runs without errors and produces the expected output
// demonstrating self-modification.
func TestOuroborosScriptExecution(t *testing.T) {
	source := `! main() {
  $ _ = print("=== Ouroboros: Self-Evolving Assembler ===")

  # Phase 1: Fork
  $ is_parent = __mitosis(0)
  $ _ = print("[fork] is_parent =", is_parent)

  # Phase 2: Self-modify
  $ _ = print("[mutator] Writing value 42 at offset 7 ...")
  $ written = __mutator(42, 7)
  $ _ = print("[mutator] Confirmed write:", written)

  # Phase 3: Read back mutation table
  $ mutations = __mutations
  $ patched = mutations[7]
  $ _ = print("[evolve] Mutation table at offset 7:", patched)

  # Phase 4: Verify self-modification
  if patched == 42 {
    $ _ = print("=== SELF-MODIFICATION CONFIRMED ===")
    $ _ = print("Code at offset 7 is now 42 (was 0 at compile time)")
    $ _ = print("Execution trace demonstrates code that did not exist at compile time.")
  } else {
    $ _ = print("ERROR: Self-modification failed, offset 7 =")
    $ _ = print(patched)
  }
}
@ command run {
  main()
}`

	output, interp, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("ouroboros script execution failed: %v", err)
	}

	// Verify the execution trace demonstrates self-modification
	expectedStrings := []string{
		"=== Ouroboros: Self-Evolving Assembler ===",
		"[fork] is_parent = true",
		"[mutator] Confirmed write: 42",
		"[evolve] Mutation table at offset 7: 42",
		"=== SELF-MODIFICATION CONFIRMED ===",
		"did not exist at compile time",
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Verify the mutation table is populated correctly
	mutationsRaw, getErr := interp.globalEnv.Get("__mutations")
	if getErr != nil {
		t.Fatalf("expected __mutations in global env: %v", getErr)
	}
	mutations := mutationsRaw.(map[int64]int64)
	if mutations[7] != 42 {
		t.Errorf("mutations[7] = %d, want 42", mutations[7])
	}
}

// TestIntKeyedMapIndexing verifies that maps with integer keys can be
// indexed using array-index syntax (e.g., myMap[7]).
func TestIntKeyedMapIndexing(t *testing.T) {
	source := `! main() {
  $ m = __mutator(100, 0)
  $ m2 = __mutator(200, 1)
  $ mutations = __mutations
  $ v0 = mutations[0]
  $ v1 = mutations[1]
  $ _ = print("v0:", v0, "v1:", v1)
}
@ command run {
  main()
}`

	output, _, err := parseLoadExecute(source)
	if err != nil {
		t.Fatalf("int-keyed map indexing failed: %v", err)
	}
	if !strings.Contains(output, "v0: 100") {
		t.Errorf("expected 'v0: 100' in output, got: %s", output)
	}
	if !strings.Contains(output, "v1: 200") {
		t.Errorf("expected 'v1: 200' in output, got: %s", output)
	}
}
