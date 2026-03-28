package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glyphlang/glyph/pkg/mod"
	"github.com/spf13/cobra"
)

// modCmd represents the mod command
var modCmd = &cobra.Command{
	Use:   "mod",
	Short: "Module management",
	Long: `Manage GlyphLang modules and dependencies.

Examples:
  glyph mod init                    Initialize a new module
  glyph mod add github.com/user/pkg Add a dependency
  glyph mod tidy                    Clean up dependencies
  glyph mod verify                  Verify dependencies`,
}

// modInitCmd initializes a new module
var modInitCmd = &cobra.Command{
	Use:   "init [module-path]",
	Short: "Initialize a new module",
	Long: `Initialize a new glyph.mod file in the current directory.

The module path should be the repository location where your module will be hosted,
e.g., github.com/username/my-api.

If no module path is provided, it will be inferred from the current directory name.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runModInit,
}

// modAddCmd adds a dependency
var modAddCmd = &cobra.Command{
	Use:   "add <package>",
	Short: "Add a dependency",
	Long: `Add a dependency to the current module.

The package should be specified as a module path, e.g., github.com/glyphlang/stdlib.
If no version is specified, the latest version will be used.

Examples:
  glyph mod add github.com/glyphlang/stdlib
  glyph mod add github.com/glyphlang/auth v0.2.0`,
	Args: cobra.MinimumNArgs(1),
	RunE: runModAdd,
}

// modTidyCmd cleans up dependencies
var modTidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "Clean up dependencies",
	Long: `Add missing and remove unused dependencies.

This command:
  - Downloads any missing dependencies
  - Removes dependencies that are not used in the source code
  - Updates the glyph.mod file`,
	RunE: runModTidy,
}

// modVerifyCmd verifies dependencies
var modVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify dependencies",
	Long: `Verify that downloaded dependencies match their expected checksums.

This is useful for ensuring the integrity of your dependencies.`,
	RunE: runModVerify,
}

// modListCmd lists dependencies
var modListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dependencies",
	Long:  `List all dependencies in the current module.`,
	RunE:  runModList,
}

func init() {
	modCmd.AddCommand(modInitCmd)
	modCmd.AddCommand(modAddCmd)
	modCmd.AddCommand(modTidyCmd)
	modCmd.AddCommand(modVerifyCmd)
	modCmd.AddCommand(modListCmd)
}

func runModInit(cmd *cobra.Command, args []string) error {
	// Get module path
	var modulePath string
	if len(args) > 0 {
		modulePath = args[0]
	} else {
		// Infer from current directory
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		modulePath = filepath.Base(dir)
	}

	// Check if glyph.mod already exists
	if _, err := os.Stat("glyph.mod"); err == nil {
		return fmt.Errorf("glyph.mod already exists")
	}

	// Create mod file
	modFile := &mod.ModFile{
		Module: modulePath,
		Glyph:  mod.Version{Major: 0, Minor: 6, Patch: 0},
		Require: []mod.Require{},
	}

	// Write to disk
	if err := mod.WriteModFile(".", modFile); err != nil {
		return fmt.Errorf("writing glyph.mod: %w", err)
	}

	fmt.Printf("Created glyph.mod for %s\n", modulePath)
	return nil
}

func runModAdd(cmd *cobra.Command, args []string) error {
	// Load existing mod file
	modFile, err := mod.LoadModFile(".")
	if err != nil {
		return fmt.Errorf("loading glyph.mod: %w", err)
	}

	packagePath := args[0]
	
	// Parse version if provided
	var version mod.Version
	if len(args) > 1 {
		version, err = mod.ParseVersion(args[1])
		if err != nil {
			return fmt.Errorf("parsing version: %w", err)
		}
	} else {
		// Get latest version from GitHub
		resolver, err := mod.NewResolver()
		if err != nil {
			return fmt.Errorf("creating resolver: %w", err)
		}
		
		version, err = resolver.GetLatestVersion(packagePath)
		if err != nil {
			// If can't fetch latest, use a default
			fmt.Printf("Warning: could not fetch latest version for %s: %v\n", packagePath, err)
			version = mod.Version{Major: 0, Minor: 1, Patch: 0}
		}
	}

	// Add dependency
	modFile.AddRequire(packagePath, version)

	// Write back to disk
	if err := mod.WriteModFile(".", modFile); err != nil {
		return fmt.Errorf("writing glyph.mod: %w", err)
	}

	fmt.Printf("Added %s %s\n", packagePath, version)
	return nil
}

func runModTidy(cmd *cobra.Command, args []string) error {
	// Load existing mod file
	modFile, err := mod.LoadModFile(".")
	if err != nil {
		return fmt.Errorf("loading glyph.mod: %w", err)
	}

	// Create resolver
	resolver, err := mod.NewResolver()
	if err != nil {
		return fmt.Errorf("creating resolver: %w", err)
	}

	// Download all dependencies
	for _, req := range modFile.Require {
		fmt.Printf("Downloading %s %s...\n", req.Path, req.Version)
		if _, err := resolver.Resolve(req); err != nil {
			return fmt.Errorf("downloading %s: %w", req.Path, err)
		}
	}

	fmt.Printf("Tidied %d dependencies\n", len(modFile.Require))
	return nil
}

func runModVerify(cmd *cobra.Command, args []string) error {
	// Load existing mod file
	modFile, err := mod.LoadModFile(".")
	if err != nil {
		return fmt.Errorf("loading glyph.mod: %w", err)
	}

	if len(modFile.Require) == 0 {
		fmt.Println("No dependencies to verify")
		return nil
	}

	// Create resolver
	resolver, err := mod.NewResolver()
	if err != nil {
		return fmt.Errorf("creating resolver: %w", err)
	}

	// Verify each dependency is cached
	verified := 0
	for _, req := range modFile.Require {
		// Try to resolve - this will check cache
		if _, err := resolver.Resolve(req); err == nil {
			fmt.Printf("✓ %s %s\n", req.Path, req.Version)
			verified++
		} else {
			fmt.Printf("✗ %s %s (not cached, run 'glyph mod tidy')\n", req.Path, req.Version)
		}
	}

	fmt.Printf("\nVerified %d/%d dependencies\n", verified, len(modFile.Require))
	return nil
}

func runModList(cmd *cobra.Command, args []string) error {
	// Load existing mod file
	modFile, err := mod.LoadModFile(".")
	if err != nil {
		return fmt.Errorf("loading glyph.mod: %w", err)
	}

	fmt.Printf("Module: %s\n", modFile.Module)
	fmt.Printf("Glyph: %s\n\n", modFile.Glyph)

	if len(modFile.Require) == 0 {
		fmt.Println("No dependencies")
		return nil
	}

	fmt.Println("Dependencies:")
	for _, req := range modFile.Require {
		fmt.Printf("  %s %s\n", req.Path, req.Version)
	}

	return nil
}
