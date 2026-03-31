package profile

import (
	"testing"
	"time"
)

func TestNewNullProfiler(t *testing.T) {
	p := NewNullProfiler()
	if p == nil {
		t.Fatal("NewNullProfiler should return a non-nil profiler")
	}
}

func TestNullProfilerRecordDoesNotPanic(t *testing.T) {
	p := NewNullProfiler()
	// Should not panic
	p.Record("test-op", 100*time.Millisecond)
}

func TestProfileDataFields(t *testing.T) {
	d := ProfileData{
		Name:           "query-users",
		ExecutionCount: 42,
		AverageTime:    5 * time.Millisecond,
		TotalTime:      210 * time.Millisecond,
	}
	if d.Name != "query-users" {
		t.Errorf("expected Name 'query-users', got %q", d.Name)
	}
	if d.ExecutionCount != 42 {
		t.Errorf("expected ExecutionCount 42, got %d", d.ExecutionCount)
	}
	if d.AverageTime != 5*time.Millisecond {
		t.Errorf("expected AverageTime 5ms, got %v", d.AverageTime)
	}
	if d.TotalTime != 210*time.Millisecond {
		t.Errorf("expected TotalTime 210ms, got %v", d.TotalTime)
	}
}
