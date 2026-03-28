package mod

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected Version
		wantErr  bool
	}{
		{"v0.1.0", Version{Major: 0, Minor: 1, Patch: 0}, false},
		{"0.1.0", Version{Major: 0, Minor: 1, Patch: 0}, false},
		{"v1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, false},
		{"invalid", Version{}, true},
		{"1.2", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	if got := v.String(); got != "v1.2.3" {
		t.Errorf("Version.String() = %q, want %q", got, "v1.2.3")
	}
}

func TestParseModFile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "simple module",
			input: `module github.com/user/my-api

glyph v0.6.0
`,
			wantErr: false,
		},
		{
			name: "module with dependencies",
			input: `module github.com/user/my-api

glyph v0.6.0

require (
    github.com/glyphlang/stdlib v0.1.0
    github.com/glyphlang/auth v0.2.0
)
`,
			wantErr: false,
		},
		{
			name: "missing module",
			input: `glyph v0.6.0
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod, err := ParseModFile(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseModFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if mod == nil {
					t.Error("ParseModFile() returned nil")
					return
				}
				if mod.Module == "" {
					t.Error("ParseModFile() module is empty")
				}
			}
		})
	}
}

func TestModFileString(t *testing.T) {
	mod := &ModFile{
		Module: "github.com/user/my-api",
		Glyph:  Version{Major: 0, Minor: 6, Patch: 0},
		Require: []Require{
			{Path: "github.com/glyphlang/stdlib", Version: Version{Major: 0, Minor: 1, Patch: 0}},
		},
	}

	output := mod.String()
	if !contains(output, "module github.com/user/my-api") {
		t.Errorf("ModFile.String() missing module line")
	}
	if !contains(output, "glyph v0.6.0") {
		t.Errorf("ModFile.String() missing glyph line")
	}
	if !contains(output, "github.com/glyphlang/stdlib") {
		t.Errorf("ModFile.String() missing require")
	}
}

func TestAddRequire(t *testing.T) {
	mod := &ModFile{
		Module:  "test",
		Glyph:   Version{Major: 0, Minor: 6, Patch: 0},
		Require: []Require{},
	}

	// Add new dependency
	mod.AddRequire("github.com/user/pkg1", Version{Major: 1, Minor: 0, Patch: 0})
	if len(mod.Require) != 1 {
		t.Errorf("AddRequire() didn't add dependency")
	}

	// Add same dependency with different version (should update)
	mod.AddRequire("github.com/user/pkg1", Version{Major: 2, Minor: 0, Patch: 0})
	if len(mod.Require) != 1 {
		t.Errorf("AddRequire() added duplicate instead of updating")
	}
	if mod.Require[0].Version.Major != 2 {
		t.Errorf("AddRequire() didn't update version")
	}

	// Add different dependency
	mod.AddRequire("github.com/user/pkg2", Version{Major: 1, Minor: 0, Patch: 0})
	if len(mod.Require) != 2 {
		t.Errorf("AddRequire() didn't add second dependency")
	}
}

func TestRemoveRequire(t *testing.T) {
	mod := &ModFile{
		Module: "test",
		Glyph:  Version{Major: 0, Minor: 6, Patch: 0},
		Require: []Require{
			{Path: "github.com/user/pkg1", Version: Version{Major: 1, Minor: 0, Patch: 0}},
			{Path: "github.com/user/pkg2", Version: Version{Major: 1, Minor: 0, Patch: 0}},
		},
	}

	// Remove existing
	removed := mod.RemoveRequire("github.com/user/pkg1")
	if !removed {
		t.Errorf("RemoveRequire() returned false for existing dependency")
	}
	if len(mod.Require) != 1 {
		t.Errorf("RemoveRequire() didn't remove dependency")
	}

	// Remove non-existing
	removed = mod.RemoveRequire("github.com/user/nonexistent")
	if removed {
		t.Errorf("RemoveRequire() returned true for non-existent dependency")
	}
}

func TestHasRequire(t *testing.T) {
	mod := &ModFile{
		Module: "test",
		Glyph:  Version{Major: 0, Minor: 6, Patch: 0},
		Require: []Require{
			{Path: "github.com/user/pkg1", Version: Version{Major: 1, Minor: 0, Patch: 0}},
		},
	}

	if !mod.HasRequire("github.com/user/pkg1") {
		t.Errorf("HasRequire() returned false for existing dependency")
	}
	if mod.HasRequire("github.com/user/nonexistent") {
		t.Errorf("HasRequire() returned true for non-existent dependency")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
