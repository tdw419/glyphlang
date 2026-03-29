package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/glyphlang/glyph/pkg/runtime"
	"github.com/spf13/cobra"
)

func runProfile(cmd *cobra.Command, args []string) error {
	bytecodeFile := args[0]
	runs, _ := cmd.Flags().GetInt("runs")
	threshold, _ := cmd.Flags().GetInt("threshold")

	// Read bytecode
	bytecode, err := os.ReadFile(bytecodeFile)
	if err != nil {
		return fmt.Errorf("failed to read bytecode: %w", err)
	}

	// Verify it's valid bytecode
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return fmt.Errorf("invalid bytecode file: missing GLYP header")
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)

	bold.Println("=== GlyphLang Profiler ===")
	fmt.Printf("Bytecode: %s (%d bytes)\n", bytecodeFile, len(bytecode))
	fmt.Printf("Runs: %d, Threshold: %d\n\n", runs, threshold)

	// Create JIT-enabled VM
	jitVM := runtime.NewJITVM()

	// Execute multiple runs
	totalStart := time.Now()
	for i := 0; i < runs; i++ {
		_, err := jitVM.Execute(bytecode)
		if err != nil {
			return fmt.Errorf("execution failed at run %d: %w", i+1, err)
		}
	}
	totalElapsed := time.Since(totalStart)

	// Get profiling data
	topProfiles := jitVM.TopProfiles(10)
	hotPaths := jitVM.HotPaths(threshold)
	stats := jitVM.Stats()

	// Display results
	bold.Println("Top Profiles:")
	if len(topProfiles) == 0 {
		fmt.Println("  (no profiles)")
	} else {
		for i, p := range topProfiles {
			green.Printf("  %d. %s", i+1, p.Name)
			cyan.Printf(" - %d executions, avg %v, total %v\n",
				p.ExecutionCount, p.AverageTime.Round(time.Microsecond), p.TotalTime.Round(time.Millisecond))
		}
	}

	fmt.Println()
	bold.Println("Hot Paths:")
	if len(hotPaths) == 0 {
		yellow.Printf("  (no paths exceed threshold of %d)\n", threshold)
	} else {
		for _, path := range hotPaths {
			green.Printf("  • %s\n", path)
		}
	}

	fmt.Println()
	bold.Println("Statistics:")
	fmt.Printf("  Total Runs:       %d\n", runs)
	fmt.Printf("  Total Time:       %v\n", totalElapsed)
	fmt.Printf("  Avg per Run:      %v\n", totalElapsed/time.Duration(runs))
	if execs, ok := stats["executions"].(int64); ok {
		fmt.Printf("  Profiled Calls:   %d\n", execs)
	}
	if hotCount, ok := stats["hot_paths"].(int); ok {
		fmt.Printf("  Hot Path Count:   %d\n", hotCount)
	}

	return nil
}
