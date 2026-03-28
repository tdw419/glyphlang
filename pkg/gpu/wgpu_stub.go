//go:build !gpu
// +build !gpu

// Package gpu provides GPU execution fallback for non-GPU builds.
// Default build falls back to CPU execution.
package gpu

import (
	"fmt"
)

// hasGPUDevice always returns false for non-GPU builds.
func hasGPUDevice() bool {
	return false
}

// initGPU is a no-op for non-GPU builds.
func initGPU() error {
	return fmt.Errorf("GPU support not compiled in (rebuild with -tags gpu)")
}

// executeGPU falls back to CPU for non-GPU builds.
func (d *Dispatcher) executeGPU(bytecode []byte, config *Config) ([]Result, error) {
	d.hasGPU = false
	return d.executeCPU(bytecode, config)
}

// CloseGPU is a no-op for non-GPU builds.
func CloseGPU() {}
