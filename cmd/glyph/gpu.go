package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/spf13/cobra"
)

func runGPU(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: glyph gpu <bytecode-file> [--vms N]")
	}

	numVMs, _ := cmd.Flags().GetInt("vms")
	showShader, _ := cmd.Flags().GetBool("shader")

	showSpatial, _ := cmd.Flags().GetBool("spatial")

	if showShader {
		d := gpu.NewDispatcher()
		fmt.Println(d.ShaderSource())
		return nil
	}

	if showSpatial && len(args) > 0 {
		bytecode, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading bytecode file: %w", err)
		}
		config, err := gpu.ParseBytecodeLayoutPublic(bytecode)
		if err != nil {
			return fmt.Errorf("parsing bytecode: %w", err)
		}
		grid := gpu.NewSpatialGrid(bytecode, int(config.CodeOffset))
		bold := color.New(color.Bold)
		bold.Printf("=== Hilbert Spatial Grid (%dx%d) ===\n", grid.Width, grid.Height)
		fmt.Printf("Bytecode: %d bytes, Code: %d bytes\n\n", len(bytecode), len(bytecode)-int(config.CodeOffset))
		fmt.Print(grid.RenderText())
		return nil
	}

	// Read bytecode file
	bytecode, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading bytecode file: %w", err)
	}

	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return fmt.Errorf("invalid bytecode file: missing GLYP header")
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	bold.Println("=== GlyphLang GPU Execution ===")
	fmt.Printf("Bytecode: %d bytes\n", len(bytecode))
	fmt.Printf("VMs:      %d\n", numVMs)

	// Count constants
	numConst := countConstants(bytecode)
	fmt.Printf("Constants: %d\n", numConst)
	fmt.Println()

	dispatcher := gpu.NewDispatcher()
	start := time.Now()
	results, err := dispatcher.Execute(bytecode, numVMs)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("GPU execution failed: %w", err)
	}

	bold.Println("Results:")
	for i, r := range results {
		if r.Error != nil {
			color.Red("  VM %d: ERROR — %v (steps: %d)", i, r.Error, r.Steps)
			continue
		}
		switch r.Tag {
		case gpu.TagInt:
			green.Printf("  VM %d: %d", i, r.IntVal)
		case gpu.TagFloat:
			green.Printf("  VM %d: %f", i, r.FloatVal)
		case gpu.TagBool:
			green.Printf("  VM %d: %v", i, r.BoolVal)
		default:
			green.Printf("  VM %d: null", i)
		}
		cyan.Printf(" (steps: %d)\n", r.Steps)
	}

	fmt.Println()
	bold.Printf("Executed %d VMs in %v\n", numVMs, elapsed)
	if numVMs > 1 {
		fmt.Printf("Avg per VM: %v\n", elapsed/time.Duration(numVMs))
	}

	return nil
}

func countConstants(bytecode []byte) int {
	if len(bytecode) < 12 {
		return 0
	}
	// Format: GLYP(4) + version(4) + constCount(4 LE)
	return int(binary.LittleEndian.Uint32(bytecode[8:12]))
}
