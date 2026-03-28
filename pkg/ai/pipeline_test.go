package ai

import (
	"testing"
)

func TestExtractCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain code",
			input:    `@ command run { > {x: 42} }`,
			expected: `@ command run { > {x: 42} }`,
		},
		{
			name:     "markdown fenced",
			input:    "```glyph\n@ command run { > {x: 42} }\n```",
			expected: `@ command run { > {x: 42} }`,
		},
		{
			name:     "markdown no lang",
			input:    "```\n@ command run { > {x: 42} }\n```",
			expected: `@ command run { > {x: 42} }`,
		},
		{
			name:     "whitespace trimmed",
			input:    "\n\n  @ command run { > {x: 42} }  \n\n",
			expected: `@ command run { > {x: 42} }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCode(tt.input)
			if got != tt.expected {
				t.Errorf("extractCode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseGlyphLang(t *testing.T) {
	source := `@ command run {
  $ x = 42
  > {result: x}
}`
	module, err := parse(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(module.Items) == 0 {
		t.Fatal("expected at least one item in module")
	}
}

func TestRunCodeDirect(t *testing.T) {
	// Test the RunCode path (no LLM call) with a simple command
	p := &Pipeline{} // No LLM handler needed for RunCode
	result, err := p.RunCode(`@ command run {
  $ x = 10 + 5
  > {result: x}
}`)
	if err != nil {
		t.Fatalf("RunCode failed: %v", err)
	}
	if result.Method != "interpreter" {
		t.Errorf("expected method=interpreter, got %s", result.Method)
	}
	if result.Output == nil {
		t.Fatal("expected non-nil output")
	}
}

func TestRunCodeRoute(t *testing.T) {
	// Test with a simple GET route — should try bytecode path
	p := &Pipeline{}
	result, err := p.RunCode(`@ GET /test {
  > {message: "hello"}
}`)
	if err != nil {
		t.Fatalf("RunCode failed: %v", err)
	}
	if result.Output == nil {
		t.Fatal("expected non-nil output")
	}
}
