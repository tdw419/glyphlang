package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// WGSLExecutionResult represents the result from the Rust WGSL runner.
type WGSLExecutionResult struct {
	PC         uint32 `json:"pc"`
	SP         uint32 `json:"sp"`
	Halted     uint32 `json:"halted"`
	Error      uint32 `json:"error"`
	Steps      uint32 `json:"steps"`
	ResultTag  uint32 `json:"result_tag"`
	ResultData int32  `json:"result_data"`
}

// ExecuteWGSL runs a WGSL shader using the native wgpu runner.
func ExecuteWGSL(wgslSource string) (*Result, error) {
	// 1. Find the runner binary
	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s. Run 'cd %s && cargo build --release' first", runnerBin, runnerDir)
	}

	// 2. Execute the runner with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, runnerBin, wgslSource)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("WGSL runner timed out (likely no compatible GPU)")
		}
		return nil, fmt.Errorf("WGSL runner failed: %w (output: %s)", err, string(output))
	}

	// 3. Parse JSON result
	var res WGSLExecutionResult
	if err := json.Unmarshal(output, &res); err != nil {
		return nil, fmt.Errorf("failed to parse WGSL runner output: %w (output: %s)", err, string(output))
	}

	// 4. Convert to common Result type
	result := &Result{
		Tag:    res.ResultTag,
		IntVal: int64(res.ResultData),
		Steps:  res.Steps,
	}

	if res.Error != 0 {
		result.Error = fmt.Errorf("GPU error code: %d", res.Error)
	}

	return result, nil
}
