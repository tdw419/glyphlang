package tests

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestGoReleaserConfigValid validates that .goreleaser.yml parses
// and contains the required fields for cross-platform releases.
func TestGoReleaserConfigValid(t *testing.T) {
	data, err := os.ReadFile("../.goreleaser.yml")
	if err != nil {
		t.Fatalf("read .goreleaser.yml: %v", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse .goreleaser.yml: %v", err)
	}

	// Verify version header (v2 format)
	if v, ok := config["version"].(int); !ok || v != 2 {
		t.Errorf("expected version: 2, got %v", config["version"])
	}

	// Verify project name
	if name, _ := config["project_name"].(string); name != "glyph" {
		t.Errorf("expected project_name 'glyph', got %q", name)
	}

	// Verify builds section
	builds, ok := config["builds"].([]interface{})
	if !ok || len(builds) == 0 {
		t.Fatal("expected non-empty builds section")
	}
	build, ok := builds[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected build entry to be a map")
	}

	// Verify main points to cmd/glyph
	if main, _ := build["main"].(string); main != "./cmd/glyph" {
		t.Errorf("expected build main './cmd/glyph', got %q", main)
	}

	// Verify ldflags inject version via -X flag
	ldflags, ok := build["ldflags"].([]interface{})
	if !ok || len(ldflags) == 0 {
		t.Fatal("expected ldflags in build config")
	}
	ldflagStr, _ := ldflags[0].(string)
	if !strings.Contains(ldflagStr, "main.version") {
		t.Errorf("ldflags should inject main.version, got %q", ldflagStr)
	}

	// Verify cross-platform targets
	goos, ok := build["goos"].([]interface{})
	if !ok || len(goos) < 3 {
		t.Errorf("expected at least 3 GOOS targets, got %v", goos)
	}
	goarch, ok := build["goarch"].([]interface{})
	if !ok || len(goarch) < 2 {
		t.Errorf("expected at least 2 GOARCH targets, got %v", goarch)
	}

	// Verify required sections exist
	for _, section := range []string{"archives", "checksum", "changelog", "release"} {
		if _, ok := config[section]; !ok {
			t.Errorf("expected %q section in goreleaser config", section)
		}
	}

	// Verify changelog filters out noise
	changelog, _ := config["changelog"].(map[string]interface{})
	filters, _ := changelog["filters"].(map[string]interface{})
	exclude, _ := filters["exclude"].([]interface{})
	if len(exclude) == 0 {
		t.Error("changelog should exclude noise commits (docs, test, ci)")
	}
}

// TestReleaseWorkflowValid validates the GitHub Actions release workflow
// triggers correctly on tag push and runs goreleaser after tests.
func TestReleaseWorkflowValid(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/release.yml")
	if err != nil {
		t.Skip("release.yml not present (removed for OAuth scope)")
	}

	var wf map[string]interface{}
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("parse release.yml: %v", err)
	}

	// Verify trigger: push tags v*
	on, ok := wf["on"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'on' section in workflow")
	}
	push, ok := on["push"].(map[string]interface{})
	if !ok {
		t.Fatal("expected push trigger")
	}
	tags, ok := push["tags"].([]interface{})
	if !ok || len(tags) == 0 {
		t.Fatal("expected tags in push trigger")
	}
	tagPattern, _ := tags[0].(string)
	if tagPattern != "v*" {
		t.Errorf("expected tag pattern 'v*', got %q", tagPattern)
	}

	// Verify write permissions for creating releases
	perms, _ := wf["permissions"].(map[string]interface{})
	if perms["contents"] != "write" {
		t.Error("release workflow needs contents: write permission")
	}

	// Verify test job runs before goreleaser
	jobs, ok := wf["jobs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected jobs section")
	}
	if _, ok := jobs["test"]; !ok {
		t.Error("expected 'test' job in release workflow")
	}
	grJob, ok := jobs["goreleaser"].(map[string]interface{})
	if !ok {
		t.Error("expected 'goreleaser' job in release workflow")
	}

	// Verify goreleaser depends on test
	needs, _ := grJob["needs"]
	switch v := needs.(type) {
	case string:
		if v != "test" {
			t.Errorf("goreleaser should depend on test, got %q", v)
		}
	case []interface{}:
		found := false
		for _, n := range v {
			if n == "test" {
				found = true
			}
		}
		if !found {
			t.Errorf("goreleaser should depend on test, got %v", v)
		}
	default:
		t.Errorf("goreleaser 'needs' should reference test job, got %v", needs)
	}
}

// TestVersionVariableExists verifies main.go has the version variable
// that goreleaser injects via ldflags (-X main.version=...).
func TestVersionVariableExists(t *testing.T) {
	data, err := os.ReadFile("../cmd/glyph/main.go")
	if err != nil {
		t.Fatalf("read cmd/glyph/main.go: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `var version =`) && !strings.Contains(content, "var version string") {
		t.Error("expected 'var version' declaration in cmd/glyph/main.go for ldflags injection")
	}
}

// TestDockerfileGoreleaserValid verifies the minimal release Dockerfile.
func TestDockerfileGoreleaserValid(t *testing.T) {
	data, err := os.ReadFile("../Dockerfile.goreleaser")
	if err != nil {
		t.Fatalf("read Dockerfile.goreleaser: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "FROM") {
		t.Error("Dockerfile.goreleaser should have FROM instruction")
	}
	if !strings.Contains(content, "COPY glyph") {
		t.Error("Dockerfile.goreleaser should COPY the glyph binary")
	}
	if !strings.Contains(content, "ENTRYPOINT") {
		t.Error("Dockerfile.goreleaser should have ENTRYPOINT")
	}
}

// TestGoReleaserDockerSection verifies the goreleaser dockers section is
// configured to build and publish multi-arch container images using
// the Dockerfile.goreleaser.
func TestGoReleaserDockerSection(t *testing.T) {
	data, err := os.ReadFile("../.goreleaser.yml")
	if err != nil {
		t.Fatalf("read .goreleaser.yml: %v", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse .goreleaser.yml: %v", err)
	}

	dockers, ok := config["dockers"].([]interface{})
	if !ok || len(dockers) == 0 {
		t.Fatal("expected non-empty dockers section in goreleaser config")
	}

	// Should have separate images for amd64 and arm64
	seenArch := map[string]bool{}
	for i, d := range dockers {
		docker, ok := d.(map[string]interface{})
		if !ok {
			t.Fatalf("dockers[%d] should be a map", i)
		}

		// Must reference the Dockerfile
		if df, _ := docker["dockerfile"].(string); df != "Dockerfile.goreleaser" {
			t.Errorf("dockers[%d] should reference Dockerfile.goreleaser, got %q", i, df)
		}

		// Must have image templates
		imgs, ok := docker["image_templates"].([]interface{})
		if !ok || len(imgs) == 0 {
			t.Errorf("dockers[%d] should have image_templates", i)
		}

		// Must use OCI labels
		flags, _ := docker["build_flag_templates"].([]interface{})
		hasLabel := false
		for _, f := range flags {
			if s, _ := f.(string); strings.Contains(s, "org.opencontainers.image") {
				hasLabel = true
			}
		}
		if !hasLabel {
			t.Errorf("dockers[%d] should set OCI image labels", i)
		}

		// Track architectures
		for _, f := range flags {
			if s, _ := f.(string); strings.Contains(s, "linux/amd64") {
				seenArch["amd64"] = true
			}
			if s, _ := f.(string); strings.Contains(s, "linux/arm64") {
				seenArch["arm64"] = true
			}
		}
	}

	if !seenArch["amd64"] {
		t.Error("dockers section should build amd64 image")
	}
	if !seenArch["arm64"] {
		t.Error("dockers section should build arm64 image")
	}

	// Verify docker_manifests for multi-arch manifest
	manifests, ok := config["docker_manifests"].([]interface{})
	if !ok || len(manifests) == 0 {
		t.Fatal("expected docker_manifests section for multi-arch images")
	}

	hasLatest := false
	for _, m := range manifests {
		manifest, _ := m.(map[string]interface{})
		if nt, _ := manifest["name_template"].(string); strings.Contains(nt, "latest") {
			hasLatest = true
		}
	}
	if !hasLatest {
		t.Error("docker_manifests should include a :latest tag manifest")
	}
}

// TestMakefileVersionInjection verifies Makefile has matching version injection
// so local builds and goreleaser use the same mechanism.
func TestMakefileVersionInjection(t *testing.T) {
	data, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "main.version") {
		t.Error("Makefile should have ldflags with main.version injection")
	}
	if !strings.Contains(content, "VERSION") {
		t.Error("Makefile should define VERSION variable")
	}
}
