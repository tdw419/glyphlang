package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/glyphlang/glyph/pkg/ast"
	"github.com/glyphlang/glyph/pkg/gpu"
	"github.com/glyphlang/glyph/pkg/jit"
)

// JITRouteManager manages the lifecycle of JIT-optimized routes.
// It tracks execution frequency and upgrades routes to higher tiers.
type JITRouteManager struct {
	compiler *jit.JITCompiler
	routes   map[string]*ast.Route
	gpuReady map[string]bool // routes detected as GPU-compatible and beneficial
	mu       sync.RWMutex
}

// NewJITRouteManager creates a new JIT manager for routes.
func NewJITRouteManager() *JITRouteManager {
	return &JITRouteManager{
		compiler: jit.NewJITCompiler(),
		routes:   make(map[string]*ast.Route),
		gpuReady: make(map[string]bool),
	}
}

// CompileRoute performs the initial baseline compilation.
func (j *JITRouteManager) CompileRoute(route *ast.Route) ([]byte, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	routeName := fmt.Sprintf("%s %s", route.Method, route.Path)
	j.routes[routeName] = route

	// Initial compilation at Tier 1 (Baseline)
	bytecode, err := j.compiler.CompileRoute(routeName, route)
	if err != nil {
		return nil, err
	}

	// Initial check for GPU compatibility
	if gpu.IsGPUCompatible(bytecode) {
		fmt.Printf("[JIT] Route %s is GPU-compatible\n", routeName)
	}

	return bytecode, nil
}

// RecordExecution logs a route execution and checks for tier upgrades or GPU offload.
// Returns upgraded bytecode if an upgrade occurred, nil otherwise.
func (j *JITRouteManager) RecordExecution(routeName string, duration time.Duration) []byte {
	// Record execution in profiler
	j.compiler.RecordExecution(routeName, duration)

	// Get stats from JIT compiler
	unit, exists := j.compiler.GetUnit(routeName)
	if !exists {
		return nil
	}

	// Check if we should upgrade to GPU offload
	// Strategy: If route is GPU-compatible and avg duration > 5ms, enable GPU offload
	profile := j.compiler.GetProfiler().GetProfile(routeName)
	if profile != nil && profile.ExecutionCount > 20 {
		avgMs := float64(profile.TotalTime.Milliseconds()) / float64(profile.ExecutionCount)
		
		j.mu.RLock()
		isGpuReady := j.gpuReady[routeName]
		j.mu.RUnlock()

		if !isGpuReady && avgMs > 5.0 && gpu.IsGPUCompatible(unit.Bytecode) {
			j.mu.Lock()
			j.gpuReady[routeName] = true
			j.mu.Unlock()
			fmt.Printf("[JIT] Enabling auto GPU-offload for %s (avg: %.2fms)\n", routeName, avgMs)
		}
	}

	// Check if we should recompile to a higher tier
	route, _ := j.getRoute(routeName)
	if route == nil {
		return nil
	}

	recompiled, err := j.compiler.CheckAdaptiveRecompilation(routeName, route)
	if err != nil {
		fmt.Printf("[JIT] Warning: Recompilation failed for %s: %v\n", routeName, err)
		return nil
	}

	if recompiled {
		newUnit, _ := j.compiler.GetUnit(routeName)
		return newUnit.Bytecode
	}

	return nil
}

// ShouldUseGPU returns true if the route should be offloaded to GPU.
func (j *JITRouteManager) ShouldUseGPU(routeName string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.gpuReady[routeName]
}

func (j *JITRouteManager) getRoute(name string) (*ast.Route, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	r, ok := j.routes[name]
	return r, ok
}

// DetailedStats returns a summary of JIT performance.
func (j *JITRouteManager) DetailedStats() interface{} {
	j.mu.RLock()
	defer j.mu.RUnlock()

	summary := make(map[string]interface{})
	for name := range j.routes {
		unit, _ := j.compiler.GetUnit(name)
		profile := j.compiler.GetProfiler().GetProfile(name)
		if unit != nil && profile != nil {
			summary[name] = map[string]interface{}{
				"tier":            unit.Tier.String(),
				"executions":      profile.ExecutionCount,
				"avg_duration_ms": float64(profile.TotalTime.Milliseconds()) / float64(profile.ExecutionCount),
				"gpu_offload":     j.gpuReady[name],
			}
		}
	}
	
	// Add global JIT stats
	summary["_global"] = j.compiler.GetDetailedStats()
	
	return summary
}
