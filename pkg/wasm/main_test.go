//go:build js && wasm

package main

import (
	"testing"

	"github.com/glyphlang/glyph/pkg/vm"
)

func TestValueToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    vm.Value
		expected interface{}
	}{
		{"int", vm.IntValue{Val: 42}, int64(42)},
		{"float", vm.FloatValue{Val: 3.14}, float64(3.14)},
		{"string", vm.StringValue{Val: "test"}, "test"},
		{"bool", vm.BoolValue{Val: true}, true},
		{"null", vm.NullValue{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueToJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMustMarshal(t *testing.T) {
	res := CompileResult{
		Success: true,
		Error:   "",
	}
	
	jsonStr := mustMarshal(res)
	expected := `{"success":true}`
	if jsonStr != expected {
		t.Errorf("Expected %s, got %s", expected, jsonStr)
	}
}
