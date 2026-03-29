package vm

import (
	"time"

	"github.com/glyphlang/glyph/pkg/profile"
)

// ProfileRecorder is a simple interface for profiling execution.
type ProfileRecorder = profile.ProfileRecorder

// ProfileData represents execution statistics for a named profile.
type ProfileData = profile.ProfileData

// SimpleProfiler wraps a profile.ProfileRecorder.
type SimpleProfiler struct {
	recorder profile.ProfileRecorder
}

// NewSimpleProfiler creates a new SimpleProfiler.
func NewSimpleProfiler() *SimpleProfiler {
	return &SimpleProfiler{
		recorder: profile.NewNullProfiler(),
	}
}

// Record records an execution.
func (p *SimpleProfiler) Record(name string, executionTime time.Duration) {
	p.recorder.Record(name, executionTime)
}
