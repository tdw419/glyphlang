package vm

import (
	"testing"
)

// SEC-3: OP_ADD type dispatch for strings

// TestStringConcatBothStrings tests "hello" + " " + "world" = "hello world"
func TestStringConcatBothStrings(t *testing.T) {
	vm := NewVM()
	vm.Push(StringValue{Val: "hello"})
	vm.Push(StringValue{Val: " "})

	if err := vm.execAdd(); err != nil {
		t.Fatalf("execAdd(string, string) error: %v", err)
	}

	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}

	sv, ok := result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if sv.Val != "hello " {
		t.Fatalf("expected 'hello ', got %q", sv.Val)
	}

	// Now add "world"
	vm.Push(sv)
	vm.Push(StringValue{Val: "world"})

	if err := vm.execAdd(); err != nil {
		t.Fatalf("execAdd(string, string) error: %v", err)
	}

	result, err = vm.Pop()
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}

	sv, ok = result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if sv.Val != "hello world" {
		t.Fatalf("expected 'hello world', got %q", sv.Val)
	}
}

// TestStringConcatMixedTypes tests string + int coercion: "count: " + 42
func TestStringConcatMixedTypes(t *testing.T) {
	tests := []struct {
		name     string
		a        Value
		b        Value
		expected string
	}{
		{"string + int", StringValue{Val: "count: "}, IntValue{Val: 42}, "count: 42"},
		{"int + string", IntValue{Val: 42}, StringValue{Val: " items"}, "42 items"},
		{"string + float", StringValue{Val: "pi="}, FloatValue{Val: 3.14}, "pi=3.14"},
		{"float + string", FloatValue{Val: 2.5}, StringValue{Val: "x"}, "2.5x"},
		{"string + bool", StringValue{Val: "flag="}, BoolValue{Val: true}, "flag=true"},
		{"string + negative int", StringValue{Val: "val="}, IntValue{Val: -7}, "val=-7"},
		{"string + zero", StringValue{Val: "zero="}, IntValue{Val: 0}, "zero=0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			vm.Push(tt.a)
			vm.Push(tt.b)

			if err := vm.execAdd(); err != nil {
				t.Fatalf("execAdd(%s, %s) error: %v", tt.a.Type(), tt.b.Type(), err)
			}

			result, err := vm.Pop()
			if err != nil {
				t.Fatalf("pop error: %v", err)
			}

			sv, ok := result.(StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T (%v)", result, result)
			}
			if sv.Val != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sv.Val)
			}
		})
	}
}

// TestStrBuiltin tests the str() builtin function for int-to-string coercion
func TestStrBuiltin(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{"int 42", IntValue{Val: 42}, "42"},
		{"int 0", IntValue{Val: 0}, "0"},
		{"int negative", IntValue{Val: -99}, "-99"},
		{"string passthrough", StringValue{Val: "hello"}, "hello"},
		{"float", FloatValue{Val: 3.14}, "3.14"},
		{"bool true", BoolValue{Val: true}, "true"},
		{"bool false", BoolValue{Val: false}, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			fn, ok := vm.builtins["str"]
			if !ok {
				t.Fatal("str builtin not registered")
			}

			result, err := fn([]Value{tt.input})
			if err != nil {
				t.Fatalf("str(%v) error: %v", tt.input, err)
			}

			sv, ok := result.(StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T", result)
			}
			if sv.Val != tt.expected {
				t.Errorf("str(%v) = %q, want %q", tt.input, sv.Val, tt.expected)
			}
		})
	}
}

// TestStrBuiltinWithConcat tests "count: " + str(42) end-to-end
func TestStrBuiltinWithConcat(t *testing.T) {
	vm := NewVM()

	// Call str(42)
	strFn := vm.builtins["str"]
	converted, err := strFn([]Value{IntValue{Val: 42}})
	if err != nil {
		t.Fatalf("str(42) error: %v", err)
	}

	// Push "count: " and str(42) result, then add
	vm.Push(StringValue{Val: "count: "})
	vm.Push(converted)

	if err := vm.execAdd(); err != nil {
		t.Fatalf("execAdd error: %v", err)
	}

	result, err := vm.Pop()
	if err != nil {
		t.Fatalf("pop error: %v", err)
	}

	sv, ok := result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if sv.Val != "count: 42" {
		t.Fatalf("expected 'count: 42', got %q", sv.Val)
	}
}
