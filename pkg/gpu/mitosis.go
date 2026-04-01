package gpu

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

// MitosisVM extends the basic Dispatcher with S opcode support.
type MitosisVM struct {
	dispatcher *Dispatcher
	maxThreads int
	nextID     atomic.Int64

	// ForceGPUError forces the GPU execution path to fail for testing.
	ForceGPUError bool

	// fallbackWarnings collects warning messages from GPU fallback events.
	fallbackWarnings []string
	fallbackMu       sync.Mutex

	// logger is used for fallback warning output. Defaults to stdlib log.
	logger *log.Logger
}

// NewMitosisVM creates a VM dispatcher with mitosis (S opcode) support.
func NewMitosisVM(maxThreads int) *MitosisVM {
	if maxThreads <= 0 {
		maxThreads = 256
	}
	return &MitosisVM{
		dispatcher: NewDispatcher(),
		maxThreads: maxThreads,
		logger:     log.New(os.Stderr, "[mitosis] ", log.LstdFlags),
	}
}

// ThreadResult contains the result from one VM thread including spawn info.
type ThreadResult struct {
	ThreadID int
	ParentID int // -1 for root thread
	Result   Result
	Children []int // IDs of spawned child threads
}

// FallbackWarnings returns the list of GPU fallback warning messages collected
// during execution. Thread-safe.
func (m *MitosisVM) FallbackWarnings() []string {
	m.fallbackMu.Lock()
	defer m.fallbackMu.Unlock()
	out := make([]string, len(m.fallbackWarnings))
	copy(out, m.fallbackWarnings)
	return out
}

// logFallbackWarning records a fallback warning message and logs it.
func (m *MitosisVM) logFallbackWarning(msg string) {
	m.fallbackMu.Lock()
	m.fallbackWarnings = append(m.fallbackWarnings, msg)
	m.fallbackMu.Unlock()
	if m.logger != nil {
		m.logger.Print(msg)
	}
}

// ExecuteWithMitosis runs bytecode with S opcode support.
// Returns results from all threads (root + spawned children).
func (m *MitosisVM) ExecuteWithMitosis(bytecode []byte) ([]ThreadResult, error) {
	if len(bytecode) < 4 || string(bytecode[:4]) != "GLYP" {
		return nil, fmt.Errorf("invalid bytecode: missing GLYP header")
	}

	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		return nil, err
	}

	// Attempt GPU execution first, fall back to CPU on failure.
	if err := m.attemptGPUExecution(bytecode, config); err != nil {
		warning := fmt.Sprintf("GPU mitosis execution failed: %v; falling back to CPU", err)
		m.logFallbackWarning(warning)
	}

	return m.executeCPUFallback(bytecode, config), nil
}

// attemptGPUExecution tries to run the bytecode on GPU via the dispatcher.
func (m *MitosisVM) attemptGPUExecution(bytecode []byte, config *Config) error {
	if m.ForceGPUError {
		return fmt.Errorf("simulated GPU failure (ForceGPUError=true)")
	}
	if !IsGPUCompatible(bytecode) {
		return fmt.Errorf("bytecode contains opcodes not supported by GPU")
	}
	if !m.dispatcher.hasGPU {
		return fmt.Errorf("GPU not available (dispatcher.hasGPU=false)")
	}
	// Full GPU mitosis execution is not yet implemented (#78).
	return fmt.Errorf("GPU mitosis not yet implemented")
}

// executeCPUFallback runs the full Mitosis execution on CPU.
func (m *MitosisVM) executeCPUFallback(bytecode []byte, config *Config) []ThreadResult {
	var (
		mu          sync.Mutex
		results     []ThreadResult
	)

	// Collect spawn requests from the root thread
	// Parent ID is -1, initial PC is 0 (runOneVM handles absolute start)
	res, spawns := m.dispatcher.runOneVM(bytecode, config, 0, 0, nil, nil)

	mu.Lock()
	results = append(results, ThreadResult{
		ThreadID: 0,
		ParentID: -1,
		Result:   res,
	})
	mu.Unlock()

	// Execute all children in parallel using a WaitGroup.
	var pending sync.WaitGroup
	pending.Add(len(spawns))

	for i, s := range spawns {
		go func(idx int, work CpuSpawnRequest) {
			defer pending.Done()
			// Child starts at s.PC + 1 + s.Offset
			// Stack and variables were already cloned and result 0 pushed in runOneVM
			childRes, _ := m.dispatcher.runOneVM(bytecode, config, idx+1, work.PC + 1 + uint32(work.Offset), work.Stack, work.Vars)
			mu.Lock()
			results = append(results, ThreadResult{
				ThreadID: idx+1,
				ParentID: 0,
				Result:   childRes,
			})
			mu.Unlock()
		}(i, s)
	}

	pending.Wait()

	return results
}
