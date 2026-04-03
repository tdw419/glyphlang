package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/glyphlang/glyph/pkg/compiler"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/glyphlang/glyph/pkg/vm"
	"github.com/spf13/cobra"
)

func runVerify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: glyph verify <file.glyph|file.glyphc>")
	}

	verbose, _ := cmd.Flags().GetBool("verbose")

	var bytecode []byte
	var sourcePath string

	if filepath.Ext(args[0]) == ".glyphc" {
		// Direct bytecode file
		var err error
		bytecode, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading bytecode: %w", err)
		}
		sourcePath = args[0]
	} else {
		// Source file: compile first
		sourcePath = args[0]
		source, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading source: %w", err)
		}

		module, err := parseSource(string(source))
		if err != nil {
			return fmt.Errorf("parse failed: %w", err)
		}

		// Resolve imports
		visited := make(map[string]bool)
		absFile, _ := filepath.Abs(args[0])
		visited[absFile] = true
		err = resolveAndFlattenModule(module, filepath.Dir(args[0]), visited)
		if err != nil {
			return fmt.Errorf("import resolution failed: %w", err)
		}

		// Compile
		c := compiler.NewCompiler()
		bytecode, err = c.Compile(module)
		if err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}
		if bytecode == nil {
			return fmt.Errorf("no runnable item found in %s", args[0])
		}
	}

	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	bold.Println("=== GlyphLang Cross-Verification ===")
	fmt.Printf("File:     %s\n", sourcePath)
	fmt.Printf("Bytecode: %d bytes\n", len(bytecode))

	// Parse header for metadata
	if len(bytecode) >= 12 {
		constCount := binary.LittleEndian.Uint32(bytecode[8:12])
		fmt.Printf("Constants: %d\n", constCount)
	}
	fmt.Println()

	// --- CPU VM execution ---
	cpuStart := time.Now()
	cpuVM := vm.NewVM()
	cpuResult, cpuErr := cpuVM.Execute(bytecode)
	cpuElapsed := time.Since(cpuStart)

	if verbose {
		cyan.Println("--- CPU VM ---")
		fmt.Printf("  Time: %s\n", cpuElapsed)
	}

	// --- GPU dispatcher (CPU fallback) execution ---
	gpuStart := time.Now()
	dispatcher := gpu.NewDispatcher()
	dispatcher.SetCPUFallback() // force CPU fallback (no real GPU needed)
	gpuResult, gpuErr := dispatcher.ExecuteOne(bytecode)
	gpuElapsed := time.Since(gpuStart)

	if verbose {
		cyan.Println("--- GPU Dispatcher ---")
		fmt.Printf("  Time: %s\n", gpuElapsed)
	}

	// --- Report ---
	fmt.Println()
	bold.Println("Results:")

	// CPU result
	fmt.Print("  CPU:  ")
	if cpuErr != nil {
		red.Printf("ERROR — %s\n", cpuErr)
	} else {
		printValue(cpuResult, green)
		fmt.Printf(" (%s)\n", cpuElapsed)
	}

	// GPU result
	fmt.Print("  GPU:  ")
	if gpuErr != nil {
		red.Printf("ERROR — %s\n", gpuErr)
	} else if gpuResult != nil && gpuResult.Error != nil {
		red.Printf("ERROR — %s (steps: %d)\n", gpuResult.Error, gpuResult.Steps)
	} else if gpuResult != nil {
		printGPUValue(gpuResult, green)
		fmt.Printf(" (%s, steps: %d)\n", gpuElapsed, gpuResult.Steps)
	} else {
		yellow.Println("null")
	}

	// --- Comparison ---
	fmt.Println()
	if cpuErr != nil || gpuErr != nil {
		red.Println("VERIFY: FAILED (execution error)")
		if verbose && cpuErr != nil {
			fmt.Printf("  CPU error: %v\n", cpuErr)
		}
		if verbose && gpuErr != nil {
			fmt.Printf("  GPU error: %v\n", gpuErr)
		}
		if gpuResult != nil && gpuResult.Error != nil {
			fmt.Printf("  GPU VM error: %v\n", gpuResult.Error)
		}
		return fmt.Errorf("verification failed")
	}

	match, reason := crossVerify(cpuResult, gpuResult)
	if match {
		color.New(color.FgGreen, color.Bold).Println("VERIFY: PASS (CPU and GPU agree)")
	} else {
		color.New(color.FgRed, color.Bold).Println("VERIFY: MISMATCH")
		yellow.Printf("  Reason: %s\n", reason)
		if verbose {
			fmt.Printf("  CPU value: %v\n", cpuResult)
			if gpuResult != nil {
				fmt.Printf("  GPU value: Tag=%d IntVal=%d FloatVal=%v BoolVal=%v\n",
					gpuResult.Tag, gpuResult.IntVal, gpuResult.FloatVal, gpuResult.BoolVal)
			}
		}
		return fmt.Errorf("CPU/GPU mismatch: %s", reason)
	}

	return nil
}

// printValue formats a VM value for display.
func printValue(val vm.Value, color *color.Color) {
	if val == nil {
		color.Print("null")
		return
	}
	switch v := val.(type) {
	case vm.IntValue:
		color.Printf("Int(%d)", v.Val)
	case vm.FloatValue:
		color.Printf("Float(%v)", v.Val)
	case vm.BoolValue:
		color.Printf("Bool(%v)", v.Val)
	case vm.StringValue:
		color.Printf("String(%q)", v.Val)
	case vm.NullValue:
		color.Print("Null")
	default:
		color.Printf("%T(%v)", val, val)
	}
}

// printGPUValue formats a GPU result for display.
func printGPUValue(r *gpu.Result, color *color.Color) {
	switch r.Tag {
	case gpu.TagInt:
		color.Printf("Int(%d)", r.IntVal)
	case gpu.TagFloat:
		color.Printf("Float(%v)", r.FloatVal)
	case gpu.TagBool:
		color.Printf("Bool(%v)", r.BoolVal)
	case gpu.TagNull:
		color.Print("Null")
	default:
		color.Printf("Tag(%d, int=%d, float=%v, bool=%v)", r.Tag, r.IntVal, r.FloatVal, r.BoolVal)
	}
}

// crossVerify checks whether a CPU VM Value and GPU Result agree.
func crossVerify(cpuVal vm.Value, gpuRes *gpu.Result) (bool, string) {
	if cpuVal == nil && gpuRes == nil {
		return true, ""
	}
	if cpuVal == nil {
		return false, "CPU nil, GPU non-nil"
	}
	if gpuRes == nil {
		return false, "CPU non-nil, GPU nil"
	}
	if gpuRes.Error != nil {
		return false, fmt.Sprintf("GPU error: %v", gpuRes.Error)
	}

	switch v := cpuVal.(type) {
	case vm.IntValue:
		if gpuRes.Tag != gpu.TagInt {
			return false, fmt.Sprintf("tag mismatch: CPU=Int(%d), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		if gpuRes.IntVal != v.Val {
			return false, fmt.Sprintf("int mismatch: CPU=%d, GPU=%d", v.Val, gpuRes.IntVal)
		}
		return true, ""

	case vm.BoolValue:
		if gpuRes.Tag != gpu.TagBool {
			return false, fmt.Sprintf("tag mismatch: CPU=Bool(%v), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		if gpuRes.BoolVal != v.Val {
			return false, fmt.Sprintf("bool mismatch: CPU=%v, GPU=%v", v.Val, gpuRes.BoolVal)
		}
		return true, ""

	case vm.FloatValue:
		if gpuRes.Tag != gpu.TagFloat {
			return false, fmt.Sprintf("tag mismatch: CPU=Float(%v), GPU=Tag(%d)", v.Val, gpuRes.Tag)
		}
		// Float precision: GPU uses f32 internally, compare at f32 precision
		cpuF := float32(v.Val)
		gpuF := float32(gpuRes.FloatVal)
		if math.Float32bits(cpuF) != math.Float32bits(gpuF) {
			return false, fmt.Sprintf("float mismatch: CPU=%v (f32=%v), GPU=%v", v.Val, cpuF, gpuRes.FloatVal)
		}
		return true, ""

	case vm.NullValue:
		if gpuRes.Tag != gpu.TagNull {
			return false, fmt.Sprintf("tag mismatch: CPU=Null, GPU=Tag(%d)", gpuRes.Tag)
		}
		return true, ""

	default:
		return false, fmt.Sprintf("unsupported CPU value type: %T", cpuVal)
	}
}
