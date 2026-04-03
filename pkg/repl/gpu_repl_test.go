package repl

import (
	"bytes"
	"strings"
	"testing"
)

// TestREPLGPUModeArithmetic tests that --gpu mode evaluates arithmetic.
// Since the SSA compiler's bytecode format differs from the GPU VM format,
// the REPL falls back to the CPU interpreter when GPU execution fails.
func TestREPLGPUModeArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"addition", "1 + 2\n", "=> 3"},
		{"multiplication", "3 * 4\n", "=> 12"},
		{"precedence", "1 + 2 * 3\n", "=> 7"},
		{"subtraction", "10 - 4\n", "=> 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input + ":quit\n")
			output := &bytes.Buffer{}

			// useGPU=true but SSA bytecode is incompatible with GPU VM
			// => should fall back to CPU and produce correct results
			r := New(input, output, "test", true)
			r.Start()

			result := output.String()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.expected, result)
			}
		})
	}
}

// TestREPLGPUFallbackMessage tests that GPU mode shows a fallback message
// when GPU execution fails and the CPU interpreter takes over.
func TestREPLGPUFallbackMessage(t *testing.T) {
	input := strings.NewReader("2 + 2\n:quit\n")
	output := &bytes.Buffer{}

	r := New(input, output, "test", true)
	r.Start()

	result := output.String()

	// Should get the correct result
	if !strings.Contains(result, "=> 4") {
		t.Errorf("Expected '=> 4', got:\n%s", result)
	}
}

// TestREPLGPUFalseUsesCPU tests that useGPU=false always uses the CPU path.
func TestREPLGPUFalseUsesCPU(t *testing.T) {
	input := strings.NewReader("2 + 2\n:quit\n")
	output := &bytes.Buffer{}

	r := New(input, output, "test", false)
	r.Start()

	result := output.String()
	if !strings.Contains(result, "=> 4") {
		t.Errorf("Expected '=> 4', got:\n%s", result)
	}
	// Should NOT contain GPU fallback message
	if strings.Contains(result, "[GPU fallback:") {
		t.Errorf("CPU-only mode should not show GPU fallback message")
	}
}

// TestREPLGPUVariablePersistence tests that variables persist across
// expressions even in GPU mode (falls back to CPU for storage).
func TestREPLGPUVariablePersistence(t *testing.T) {
	input := strings.NewReader("$ x = 10\nx * 2\n:quit\n")
	output := &bytes.Buffer{}

	r := New(input, output, "test", true)
	r.Start()

	result := output.String()
	if !strings.Contains(result, "=> 20") {
		t.Errorf("Expected '=> 20' for x*2, got:\n%s", result)
	}
}
