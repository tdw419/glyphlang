package gpu

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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

// Result struct copied from gpu.go for reference/completeness if needed in this file
// type Result struct {
//	Tag      uint32
//	IntVal   int64
//	FloatVal float64
//	BoolVal  bool
//	Error    error
//	Steps    uint32
// }

// GlyphJob matches the Rust runner's job structure.
type GlyphJob struct {
	BytecodeHex    string `json:"bytecode_hex"`
	NumVMs         uint32 `json:"num_vms"`
	WorkgroupCount uint32 `json:"workgroup_count"`
}

// PersistentRunner manages a long-running GPU process.
type PersistentRunner struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

var (
	globalRunner *PersistentRunner
	runnerMu     sync.Mutex
)

// getPersistentRunner initializes or returns the global GPU runner.
func getPersistentRunner(shaderPath string) (*PersistentRunner, error) {
	runnerMu.Lock()
	defer runnerMu.Unlock()

	if globalRunner != nil {
		return globalRunner, nil
	}

	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s", runnerBin)
	}

	cmd := exec.Command(runnerBin, shaderPath, "1", "1")
	cmd.Env = append(os.Environ(), "GLYPH_DAEMON=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	globalRunner = &PersistentRunner{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}

	return globalRunner, nil
}

// ExecuteMultiWGSL runs a WGSL shader with multiple VMs and dynamic dispatch dimensions.
func ExecuteMultiWGSL(wgslSource string, numVMs int, workgroupCount int) ([]Result, error) {
	// 1. Prepare shader substrate
	tmpFile, err := os.CreateTemp("", "glyph_substrate_*.wgsl")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(wgslSource); err != nil {
		return nil, err
	}
	tmpFile.Close()

	// 2. Try persistent runner
	runner, err := getPersistentRunner(tmpFile.Name())
	if err == nil {
		results, err := runner.Submit(numVMs, workgroupCount)
		if err == nil {
			return results, nil
		}
		// If runner fails, we could potentially reset it or fallback
		fmt.Fprintf(os.Stderr, "[GPU] Persistent runner failed, falling back: %v\n", err)
	}

	// 3. Fallback to legacy shell-out
	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s. Run 'cd %s && cargo build --release' first", runnerBin, runnerDir)
	}

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

	return parseResults(output, numVMs)
}

func (r *PersistentRunner) Submit(numVMs, workgroupCount int) ([]Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job := GlyphJob{
		BytecodeHex:    hex.EncodeToString([]byte("stub")),
		NumVMs:         uint32(numVMs),
		WorkgroupCount: uint32(workgroupCount),
	}

	data, _ := json.Marshal(job)
	if _, err := fmt.Fprintln(r.stdin, string(data)); err != nil {
		return nil, err
	}

	if !r.stdout.Scan() {
		return nil, fmt.Errorf("GPU runner disconnected")
	}

	return parseResults(r.stdout.Bytes(), numVMs)
}

func parseResults(output []byte, numVMs int) ([]Result, error) {
	if numVMs == 1 {
		var res WGSLExecutionResult
		if err := json.Unmarshal(output, &res); err != nil {
			return nil, fmt.Errorf("failed to parse WGSL result: %w (output: %s)", err, string(output))
		}
		return []Result{{
			Tag:    res.ResultTag,
			IntVal: int64(res.ResultData),
			Steps:  res.Steps,
		}}, nil
	}

	var res []WGSLExecutionResult
	if err := json.Unmarshal(output, &res); err != nil {
		return nil, fmt.Errorf("failed to parse multi WGSL result: %w (output: %s)", err, string(output))
	}

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

// ExecuteWGSL runs a WGSL shader using the native wgpu runner.
func ExecuteWGSL(wgslSource string) (*Result, error) {
	results, err := ExecuteMultiWGSL(wgslSource, 1, 1)
	if err != nil {
		return nil, err
	}
	return &results[0], nil
}

// ExecuteWGSLDaemon starts a persistent runner that watches a shader file for hot-reload.
func ExecuteWGSLDaemon(shaderPath string, numVMs int, workgroupCount int) (*exec.Cmd, error) {
	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s", runnerBin)
	}

	cmd := exec.Command(runnerBin, shaderPath, fmt.Sprintf("%d", numVMs), fmt.Sprintf("%d", workgroupCount))
	cmd.Env = append(os.Environ(), "GLYPH_WATCH=1")

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start WGSL daemon: %w", err)
	}

	return cmd, nil
}
