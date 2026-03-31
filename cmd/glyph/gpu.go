package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/spf13/cobra"
)

func runGPU(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: glyph gpu <file.glyph|file.glyphc> [--vms N]")
	}

	numVMs, _ := cmd.Flags().GetInt("vms")
	if numVMs == 0 { numVMs = 1 }
	showShader, _ := cmd.Flags().GetBool("shader")
	showSpatial, _ := cmd.Flags().GetBool("spatial")

	if showShader {
		d := gpu.NewDispatcher()
		fmt.Println(d.ShaderSource())
		return nil
	}

	var bytecode []byte
	var err error

	// Handle .glyph source files by compiling on-the-fly
	if filepath.Ext(args[0]) == ".glyph" {
		source, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		module, err := parseSource(string(source))
		if err != nil {
			return err
		}
		c := compiler.NewSSACompiler()
		// Compile the first function or route to GPU bytecode
		var targetBytecode []byte
		for _, item := range module.Items {
			if fn, ok := item.(*ast.Function); ok {
				targetBytecode, err = c.CompileFunction(fn)
				if err != nil {
					return fmt.Errorf("function compilation failed: %w", err)
				}
				break
			} else if route, ok := item.(*ast.Route); ok {
				targetBytecode, err = c.CompileRoute(route)
				if err != nil {
					return fmt.Errorf("route compilation failed: %w", err)
				}
				break
			}
		}
		if targetBytecode == nil {
			return fmt.Errorf("no runnable function or route found in %s", args[0])
		}
		bytecode = targetBytecode
	} else {
		// Read bytecode file
		bytecode, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading bytecode file: %w", err)
		}
	}

	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	if showSpatial {
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

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	bold.Println("=== GlyphLang GPU Execution ===")
	fmt.Printf("File:      %s\n", args[0])
	fmt.Printf("Bytecode:  %d bytes\n", len(bytecode))
	fmt.Printf("VMs:       %d\n", numVMs)

	// Count constants
	numConst := countConstants(bytecode)
	fmt.Printf("Constants: %d\n", numConst)
	fmt.Println()

	dispatcher := gpu.NewDispatcher()
	useLive, _ := cmd.Flags().GetBool("live")
	if useLive {
		bold.Println("Live execution mode started. Press Ctrl+C to stop.")
		for {
			start := time.Now()
			_, err := dispatcher.Execute(bytecode, numVMs)
			if err != nil { return err }
			fmt.Printf("\rFrame: %v (VMs: %d)    ", time.Since(start), numVMs)
			time.Sleep(16 * time.Millisecond)
		}
	}
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
