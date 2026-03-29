package config

import (
	"testing"
)

func TestDefaultPort(t *testing.T) {
	if DefaultPort != 3000 {
		t.Errorf("Expected DefaultPort to be 3000, got %d", DefaultPort)
	}
}