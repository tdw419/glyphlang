package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestColonyExamples validates that the three Mitosis pattern example
// files parse successfully via glyph validate.
func TestColonyExamples(t *testing.T) {
	examples := []string{
		"colony_linear.glyph",
		"colony_recursive.glyph",
		"colony_conditional.glyph",
	}

	// Find the glyph binary
	glyphBin, err := exec.LookPath("glyph")
	if err != nil {
		// Build it on the fly
		glyphBin = filepath.Join(t.TempDir(), "glyph")
		cmd := exec.Command("go", "build", "-o", glyphBin, "./cmd/glyph")
		cmd.Dir = mustGetProjectRoot(t)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to build glyph binary: %v", err)
		}
	}

	examplesDir := filepath.Join(mustGetProjectRoot(t), "examples")

	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(examplesDir, name)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Fatalf("example file not found: %s", path)
			}

			cmd := exec.Command(glyphBin, "validate", path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("glyph validate failed for %s: %v\n%s", name, err, output)
			}

			if !containsString(string(output), "is valid") {
				t.Errorf("expected 'is valid' in output, got: %s", output)
			}
		})
	}
}

func mustGetProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from test file to find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
