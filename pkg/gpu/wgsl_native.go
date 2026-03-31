package gpu

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type WGSLExecutionResult struct {
	PC         uint32 `json:"pc"`
	SP         uint32 `json:"sp"`
	Halted     uint32 `json:"halted"`
	Error      uint32 `json:"error"`
	Steps      uint32 `json:"steps"`
	ResultTag  uint32 `json:"result_tag"`
	ResultData int32  `json:"result_data"`
}

type JobTimings struct {
	InitMS     float64 `json:"init_ms"`
	ComputeMS  float64 `json:"compute_ms"`
	ReadbackMS float64 `json:"readback_ms"`
	TotalMS    float64 `json:"total_ms"`
}

type JobResponse struct {
	Results []WGSLExecutionResult `json:"results"`
	Timings JobTimings            `json:"timings_ms"`
}

type GlyphJob struct {
	BytecodeHex    string `json:"bytecode_hex"`
	NumVMs         uint32 `json:"num_vms"`
	WorkgroupCount uint32 `json:"workgroup_count"`
	CodeOffset     uint32 `json:"code_offset"`
	ConstOffset    uint32 `json:"const_offset"`
	NumConstants   uint32 `json:"num_constants"`
}

type PersistentRunner struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

var (
	globalRunner *PersistentRunner
	runnerMu     sync.Mutex
	serverOnce   sync.Once
)

func startVCCServer() {
	serverOnce.Do(func() {
		http.HandleFunc("/vcc/colony.rgba", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.ServeFile(w, r, "/tmp/vcc_colony.rgba")
		})
		go http.ListenAndServe(":8080", nil)
		fmt.Println("[VCC] Texture server started at http://localhost:8080/vcc/colony.rgba")
	})
}

func getPersistentRunner() (*PersistentRunner, error) {
	runnerMu.Lock()
	defer runnerMu.Unlock()

	if globalRunner != nil {
		return globalRunner, nil
	}

	startVCCServer()

	tmpShader, err := os.CreateTemp("", "glyph_substrate_*.wgsl")
	if err != nil {
		return nil, err
	}
	if _, err := tmpShader.WriteString(shaderSource); err != nil {
		return nil, err
	}
	tmpShader.Close()

	_, filename, _, _ := runtime.Caller(0)
	runnerDir := filepath.Join(filepath.Dir(filename), "wgsl_runner")
	runnerBin := filepath.Join(runnerDir, "target", "release", "glyphlang-wgsl-runner")

	if _, err := os.Stat(runnerBin); os.IsNotExist(err) {
		return nil, fmt.Errorf("WGSL runner binary not found at %s", runnerBin)
	}

	cmd := exec.Command(runnerBin, tmpShader.Name(), "1", "1")
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

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	globalRunner = &PersistentRunner{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
	}

	return globalRunner, nil
}

func ExecuteMultiWGSL(bytecode []byte, numVMs int, workgroupCount int) ([]Result, error) {
	runner, err := getPersistentRunner()
	if err == nil {
		results, err := runner.Submit(bytecode, numVMs, workgroupCount)
		if err == nil {
			return results, nil
		}
		fmt.Fprintf(os.Stderr, "[GPU] Persistent runner failed: %v\n", err)
		return nil, err
	}
	return nil, fmt.Errorf("GPU daemon unavailable: %v", err)
}

func (r *PersistentRunner) Submit(bytecode []byte, numVMs, workgroupCount int) ([]Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		return nil, err
	}

	job := GlyphJob{
		BytecodeHex:    hex.EncodeToString(bytecode),
		NumVMs:         uint32(numVMs),
		WorkgroupCount: uint32(workgroupCount),
		CodeOffset:     config.CodeOffset,
		ConstOffset:    config.ConstantsOffset,
		NumConstants:   config.NumConstants,
	}

	data, _ := json.Marshal(job)
	if _, err := fmt.Fprintln(r.stdin, string(data)); err != nil {
		return nil, err
	}

	if !r.stdout.Scan() {
		return nil, fmt.Errorf("GPU runner scan failed: %v", r.stdout.Err())
	}

	var resp JobResponse
	if err := json.Unmarshal(r.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse job response: %w", err)
	}

	// Print timings if in debug mode or for benchmark
	fmt.Fprintf(os.Stderr, "[GPU] Timings: Init: %.2fms, Compute: %.2fms, Readback: %.2fms, Total: %.2fms\n",
		resp.Timings.InitMS, resp.Timings.ComputeMS, resp.Timings.ReadbackMS, resp.Timings.TotalMS)

	results := make([]Result, len(resp.Results))
	for i, res := range resp.Results {
		results[i] = Result{
			Tag:    res.ResultTag,
			IntVal: int64(res.ResultData),
			Steps:  res.Steps,
		}
		if res.Error != 0 {
			results[i].Error = fmt.Errorf("GPU error code: %d", res.Error)
		}
	}

	return results, nil
}

func parseResults(output []byte, numVMs int) ([]Result, error) {
	return nil, fmt.Errorf("parseResults is legacy")
}

func ExecuteWGSL(wgslSource string) (*Result, error) {
	return nil, fmt.Errorf("ExecuteWGSL is legacy")
}

func ExecuteWGSLDaemon(shaderPath string, numVMs int, workgroupCount int) (*exec.Cmd, error) {
	return nil, fmt.Errorf("ExecuteWGSLDaemon is legacy")
}
