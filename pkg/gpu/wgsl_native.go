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
	results, err := ExecuteMultiWGSL(wgslSource, 1, 1)
	if err != nil {
		return nil, err
	}
	return &results[0], nil
}

// ExecuteMultiWGSL runs a WGSL shader with multiple VMs and dynamic dispatch dimensions.
func ExecuteMultiWGSL(wgslSource string, numVMs int, workgroupCount int) ([]Result, error) {
	// For one-shot execution, we write to a temporary file
	tmpFile, err := os.CreateTemp("", "glyph_shader_*.wgsl")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(wgslSource); err != nil {
		return nil, err
	}
	tmpFile.Close()

	// 1. Find the runner binary
	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s. Run 'cd %s && cargo build --release' first", runnerBin, runnerDir)
	}

	// 2. Execute the runner with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, runnerBin, tmpFile.Name(), fmt.Sprintf("%d", numVMs), fmt.Sprintf("%d", workgroupCount))
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("WGSL runner timed out (likely no compatible GPU)")
		}
		return nil, fmt.Errorf("WGSL runner failed: %w (output: %s)", err, string(output))
	}

	// 3. Parse JSON result
	if numVMs == 1 {
		var res WGSLExecutionResult
		if err := json.Unmarshal(output, &res); err != nil {
			return nil, fmt.Errorf("failed to parse WGSL runner output: %w (output: %s)", err, string(output))
		}
		return []Result{{
			Tag:    res.ResultTag,
			IntVal: int64(res.ResultData),
			Steps:  res.Steps,
		}}, nil
	}

	var res []WGSLExecutionResult
	if err := json.Unmarshal(output, &res); err != nil {
		return nil, fmt.Errorf("failed to parse WGSL runner output: %w (output: %s)", err, string(output))
	}

	// 4. Convert to common Result type
	results := make([]Result, len(res))
	for i, r := range res {
		results[i] = Result{
			Tag:    r.ResultTag,
			IntVal: int64(r.ResultData),
			Steps:  r.Steps,
		}
		if r.Error != 0 {
			results[i].Error = fmt.Errorf("GPU error code: %d", r.Error)
		}
	}

	return results, nil
}

// ExecuteWGSLDaemon starts a persistent runner that watches a shader file for hot-reload.
func ExecuteWGSLDaemon(shaderPath string, numVMs int, workgroupCount int) (*exec.Cmd, error) {
	// 1. Find the runner binary
	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s", runnerBin)
	}

	// 2. Execute the runner in watch mode
	cmd := exec.Command(runnerBin, shaderPath, fmt.Sprintf("%d", numVMs), fmt.Sprintf("%d", workgroupCount))
	cmd.Env = append(os.Environ(), "GLYPH_WATCH=1")
	
	// Direct output to stderr for live telemetry logs
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start WGSL daemon: %w", err)
	}

	return cmd, nil
}
