package repl

import (
	"os"
	"bytes"
	"strings"
	"testing"
)

// TestGPUModeArithmetic verifies that the REPL in GPU mode (useGPU=true)
// correctly evaluates arithmetic expressions through the SSA compiler and
// GPU dispatcher (CPU fallback when no physical GPU is present).
func TestGPUModeArithmetic(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"addition", "3 + 2\n", "=> 5"},
		{"subtraction", "10 - 4\n", "=> 6"},
		{"multiplication", "6 * 7\n", "=> 42"},
		{"division", "15 / 3\n", "=> 5"},
		{"negative result", "3 - 10\n", "=> -7"},
		{"precedence: mul before add", "2 + 3 * 4\n", "=> 14"},
		{"precedence: parens", "(2 + 3) * 4\n", "=> 20"},
		{"modulo", "17 % 5\n", "=> 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input + ":quit\n")
			output := &bytes.Buffer{}

			r := New(input, output, "test", true) // useGPU = true
			if err := r.Start(); err != nil {
				t.Fatalf("REPL Start error: %v", err)
			}

			got := output.String()
			if !strings.Contains(got, tt.expected) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.expected, got)
			}
		})
	}
}

// TestGPUModeSingleExpression verifies a single arithmetic expression end-to-end
// through the GPU path using processLine directly (no full REPL loop).
func TestGPUModeSingleExpression(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"addition", "1 + 1", "=> 2"},
		{"multiply", "5 * 5", "=> 25"},
		{"divide", "20 / 4", "=> 5"},
		{"subtract", "100 - 1", "=> 99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			r := New(strings.NewReader(""), output, "test", true)

			if err := r.processLine(tt.input); err != nil {
				t.Fatalf("processLine(%q) error: %v", tt.input, err)
			}

			got := output.String()
			if !strings.Contains(got, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestGPUModeCreatesDispatcher verifies that GPU mode initializes a dispatcher.
func TestGPUModeCreatesDispatcher(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	r := New(strings.NewReader(""), &bytes.Buffer{}, "test", true)
	if r.dispatcher == nil {
		t.Error("Expected dispatcher to be initialized when useGPU=true")
	}
	if r.useGPU != true {
		t.Error("Expected useGPU=true")
	}
}

// TestCPUModeNoDispatcher verifies that CPU mode does not create a dispatcher.
func TestCPUModeNoDispatcher(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	r := New(strings.NewReader(""), &bytes.Buffer{}, "test", false)
	if r.dispatcher != nil {
		t.Error("Expected nil dispatcher when useGPU=false")
	}
}

// TestGPUModeDivisionByZero verifies that GPU mode handles division by zero.
func TestGPUModeDivisionByZero(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	output := &bytes.Buffer{}
	r := New(strings.NewReader(""), output, "test", true)

	err := r.processLine("10 / 0")
	if err == nil {
		t.Error("Expected error for division by zero in GPU mode")
	}
}

// TestGPUModeComplexExpression tests a more complex arithmetic expression.
func TestGPUModeComplexExpression(t *testing.T) {
	if os.Getenv("GLYPH_TEST_GPU") == "" { t.Skip("requires GLYPH_TEST_GPU=1") }
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"chained add", "1 + 2 + 3", "=> 6"},
		{"mixed ops", "2 * 3 + 4", "=> 10"},
		{"nested parens", "((2 + 3))", "=> 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &bytes.Buffer{}
			r := New(strings.NewReader(""), output, "test", true)

			if err := r.processLine(tt.input); err != nil {
				t.Fatalf("processLine(%q) error: %v", tt.input, err)
			}

			got := output.String()
			if !strings.Contains(got, tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, got)
			}
		})
	}
}
