package profile

import "time"

// ProfileRecorder is a simple interface for profiling execution.
type ProfileRecorder interface {
	Record(name string, executionTime time.Duration)
}

// NullProfiler is a no-op profiler.
type NullProfiler struct{}

// NewNullProfiler creates a new NullProfiler.
func NewNullProfiler() *NullProfiler {
	return &NullProfiler{}
}

func (n *NullProfiler) Record(name string, executionTime time.Duration) {}

// ProfileData represents execution statistics for a named profile.
type ProfileData struct {
	Name           string
	ExecutionCount int64
	AverageTime    time.Duration
	TotalTime      time.Duration
}
