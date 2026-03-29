// Package mod implements GlyphLang's module and dependency management.
//
// The module system is inspired by Go modules but simplified for AI-friendly syntax.
// Packages are hosted on GitHub and versioned via git tags.
package mod

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func ParseVersion(s string) (Version, error) {
	// Remove 'v' prefix if present
	s = strings.TrimPrefix(s, "v")

	var major, minor, patch int
	_, err := fmt.Sscanf(s, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		return Version{}, fmt.Errorf("invalid version format: %s", s)
	}
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// Require represents a single dependency
type Require struct {
	Path    string  // e.g., "github.com/glyphlang/stdlib"
	Version Version // e.g., v0.1.0
}

func (r Require) String() string {
	return fmt.Sprintf("%s %s", r.Path, r.Version)
}

// ModFile represents the contents of a glyph.mod file
type ModFile struct {
	Module  string    // Module path (e.g., "github.com/user/my-api")
	Glyph   Version   // GlyphLang version requirement
	Require []Require // Dependencies
}

// String returns the glyph.mod file contents
func (m *ModFile) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("module %s\n\n", m.Module))
	sb.WriteString(fmt.Sprintf("glyph %s\n\n", m.Glyph))

	if len(m.Require) > 0 {
		sb.WriteString("require (\n")
		for _, req := range m.Require {
			sb.WriteString(fmt.Sprintf("    %s\n", req))
		}
		sb.WriteString(")\n")
	}

	return sb.String()
}

// ParseModFile parses a glyph.mod file
func ParseModFile(content string) (*ModFile, error) {
	mod := &ModFile{
		Require: []Require{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	inRequire := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Module directive
		if strings.HasPrefix(line, "module ") {
			mod.Module = strings.TrimPrefix(line, "module ")
			continue
		}

		// Glyph version directive
		if strings.HasPrefix(line, "glyph ") {
			vStr := strings.TrimPrefix(line, "glyph ")
			v, err := ParseVersion(vStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			mod.Glyph = v
			continue
		}

		// Require block
		if line == "require (" {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			// Parse: path version
			parts := strings.Fields(line)
			if len(parts) != 2 {
				return nil, fmt.Errorf("line %d: invalid require syntax, expected 'path version'", lineNum)
			}

			v, err := ParseVersion(parts[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}

			mod.Require = append(mod.Require, Require{
				Path:    parts[0],
				Version: v,
			})
		}
	}

	if mod.Module == "" {
		return nil, fmt.Errorf("missing module directive")
	}

	return mod, nil
}

// LoadModFile loads and parses a glyph.mod file from the given directory
func LoadModFile(dir string) (*ModFile, error) {
	modPath := filepath.Join(dir, "glyph.mod")

	content, err := os.ReadFile(modPath)
	if err != nil {
		return nil, fmt.Errorf("reading glyph.mod: %w", err)
	}

	return ParseModFile(string(content))
}

// WriteModFile writes a ModFile to disk
func WriteModFile(dir string, mod *ModFile) error {
	modPath := filepath.Join(dir, "glyph.mod")

	content := mod.String()
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(modPath, []byte(content), 0644)
}

// AddRequire adds a new dependency to the mod file
func (m *ModFile) AddRequire(path string, version Version) {
	// Check if already exists
	for i, req := range m.Require {
		if req.Path == path {
			m.Require[i].Version = version
			return
		}
	}

	m.Require = append(m.Require, Require{
		Path:    path,
		Version: version,
	})
}

// RemoveRequire removes a dependency from the mod file
func (m *ModFile) RemoveRequire(path string) bool {
	for i, req := range m.Require {
		if req.Path == path {
			m.Require = append(m.Require[:i], m.Require[i+1:]...)
			return true
		}
	}
	return false
}

// HasRequire checks if a dependency exists
func (m *ModFile) HasRequire(path string) bool {
	for _, req := range m.Require {
		if req.Path == path {
			return true
		}
	}
	return false
}

// DefaultCacheDir returns the default directory for cached packages
func DefaultCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".glyph/cache" // fallback
	}
	return filepath.Join(homeDir, ".glyph", "cache")
}

// IsValidModulePath checks if a path looks like a valid module path
func IsValidModulePath(path string) bool {
	// Basic validation: should look like github.com/user/repo or similar
	pattern := `^[a-z0-9.-]+/[a-z0-9_-]+/[a-z0-9_-]+$`
	matched, _ := regexp.MatchString(pattern, path)
	return matched || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}
