// Package runtime provides JIT integration without import cycles.
// It sits between vm and jit, avoiding the compiler-server dependency chain.
package runtime

import (
	"time"

	"github.com/glyphlang/glyph/pkg/jit"
	"github.com/glyphlang/glyph/pkg/vm"
)

// JITProfiler adapts jit.Profiler to vm.ProfileRecorder.
type JITProfiler struct {
	profiler *jit.Profiler
}

// NewJITProfiler creates a new JIT-backed profiler.
func NewJITProfiler() *JITProfiler {
	return &JITProfiler{
		profiler: jit.NewProfiler(),
	}
}

// Record implements vm.ProfileRecorder.
func (j *JITProfiler) Record(name string, executionTime time.Duration) {
	j.profiler.RecordExecution(name, executionTime)
}

// HotPaths returns routes exceeding the threshold.
func (j *JITProfiler) HotPaths(threshold int) []string {
	return j.profiler.GetHotPaths(threshold)
}

// TopProfiles returns top N profiles by execution count.
func (j *JITProfiler) TopProfiles(n int) []vm.ProfileData {
	top := j.profiler.GetTopNByExecutionCount(n)
	result := make([]vm.ProfileData, len(top))
	for i := range top {
		result[i] = vm.ProfileData{
			Name:           top[i].Name,
			ExecutionCount: top[i].ExecutionCount,
			AverageTime:    top[i].AverageTime,
			TotalTime:      top[i].TotalTime,
		}
	}
	return result
}

// Stats returns profiler statistics.
func (j *JITProfiler) Stats() map[string]interface{} {
	stats := j.profiler.GetStats()
	return map[string]interface{}{
		"total_profiles":  stats.TotalProfiles,
		"hot_paths":       stats.HotPathCount,
		"executions":      stats.TotalExecutions,
		"total_time_ms":   stats.TotalExecutionTime.Milliseconds(),
		"call_graph_size": stats.CallGraphSize,
	}
}

// JITVM wraps a VM with JIT profiling.
type JITVM struct {
	*vm.VM
	profiler *JITProfiler
}

// NewJITVM creates a VM with JIT profiling enabled.
func NewJITVM() *JITVM {
	profiler := NewJITProfiler()
	v := vm.NewVM()
	v.SetProfiler(profiler)
	return &JITVM{
		VM:       v,
		profiler: profiler,
	}
}

// HotPaths returns hot paths from profiling data.
func (j *JITVM) HotPaths(threshold int) []string {
	return j.profiler.HotPaths(threshold)
}

// TopProfiles returns top profiles.
func (j *JITVM) TopProfiles(n int) []vm.ProfileData {
	return j.profiler.TopProfiles(n)
}

// Stats returns profiler statistics.
func (j *JITVM) Stats() map[string]interface{} {
	return j.profiler.Stats()
}
