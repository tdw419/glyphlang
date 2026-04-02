package vm

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestFutureValue_Type(t *testing.T) {
	f := &FutureValue{
		Done: make(chan struct{}),
	}
	if f.Type() != "future" {
		t.Errorf("FutureValue.Type() = %q, want %q", f.Type(), "future")
	}
}

func TestFutureValue_MarshalJSON_Pending(t *testing.T) {
	f := &FutureValue{
		Done: make(chan struct{}),
	}

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := `{"pending":true}`
	if string(b) != expected {
		t.Errorf("got %s, want %s", string(b), expected)
	}
}

func TestFutureValue_MarshalJSON_Resolved(t *testing.T) {
	f := &FutureValue{
		Result: IntValue{Val: 42},
		Done:   make(chan struct{}),
	}
	close(f.Done)

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := "42"
	if string(b) != expected {
		t.Errorf("got %s, want %s", string(b), expected)
	}
}

func TestFutureValue_MarshalJSON_ResolvedString(t *testing.T) {
	f := &FutureValue{
		Result: StringValue{Val: "hello"},
		Done:   make(chan struct{}),
	}
	close(f.Done)

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := `"hello"`
	if string(b) != expected {
		t.Errorf("got %s, want %s", string(b), expected)
	}
}

func TestFutureValue_MarshalJSON_ResolvedNull(t *testing.T) {
	f := &FutureValue{
		Result: nil,
		Done:   make(chan struct{}),
	}
	close(f.Done)

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	expected := "null"
	if string(b) != expected {
		t.Errorf("got %s, want %s", string(b), expected)
	}
}

func TestFutureValue_Await_Success(t *testing.T) {
	f := &FutureValue{
		Result: IntValue{Val: 42},
		Done:   make(chan struct{}),
	}
	close(f.Done)

	val, err := f.Await()
	if err != nil {
		t.Fatalf("Await() returned error: %v", err)
	}

	intVal, ok := val.(IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T", val)
	}
	if intVal.Val != 42 {
		t.Errorf("Expected 42, got %d", intVal.Val)
	}
}

func TestFutureValue_Await_Error(t *testing.T) {
	expectedErr := errors.New("async operation failed")
	f := &FutureValue{
		Error: expectedErr,
		Done:  make(chan struct{}),
	}
	close(f.Done)

	val, err := f.Await()
	if err == nil {
		t.Fatal("Await() should have returned an error")
	}
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if val != nil {
		t.Errorf("Expected nil value, got %v", val)
	}
}

func TestFutureValue_Await_NilDone(t *testing.T) {
	f := &FutureValue{
		Result: IntValue{Val: 99},
		Done:   nil,
	}

	val, err := f.Await()
	if err != nil {
		t.Fatalf("Await() returned error: %v", err)
	}

	intVal, ok := val.(IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T", val)
	}
	if intVal.Val != 99 {
		t.Errorf("Expected 99, got %d", intVal.Val)
	}
}

func TestFutureValue_Await_AsyncResolve(t *testing.T) {
	f := &FutureValue{
		Done: make(chan struct{}),
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		f.Result = StringValue{Val: "async result"}
		close(f.Done)
	}()

	val, err := f.Await()
	if err != nil {
		t.Fatalf("Await() returned error: %v", err)
	}

	strVal, ok := val.(StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T", val)
	}
	if strVal.Val != "async result" {
		t.Errorf("Expected 'async result', got %q", strVal.Val)
	}
}

func TestFutureValue_Await_AsyncError(t *testing.T) {
	f := &FutureValue{
		Done: make(chan struct{}),
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		f.Error = errors.New("async error")
		close(f.Done)
	}()

	val, err := f.Await()
	if err == nil {
		t.Fatal("Await() should have returned an error")
	}
	if err.Error() != "async error" {
		t.Errorf("Expected 'async error', got %q", err.Error())
	}
	if val != nil {
		t.Errorf("Expected nil value, got %v", val)
	}
}

func TestDefaultAsyncTimeout(t *testing.T) {
	if DefaultAsyncTimeout != 30*time.Second {
		t.Errorf("DefaultAsyncTimeout = %v, want 30s", DefaultAsyncTimeout)
	}
}

func TestValueTagConstants(t *testing.T) {
	// Verify tag constants are distinct
	tags := map[string]int{
		"NullTag":   NullTag,
		"IntTag":    IntTag,
		"FloatTag":  FloatTag,
		"StringTag": StringTag,
		"BoolTag":   BoolTag,
		"ArrayTag":  ArrayTag,
		"ObjectTag": ObjectTag,
		"FuncTag":   FuncTag,
		"FutureTag": FutureTag,
		"ResultTag": ResultTag,
		"RefPtrTag": RefPtrTag,
	}
	seen := map[int]string{}
	for name, tag := range tags {
		if prev, dup := seen[tag]; dup {
			t.Errorf("duplicate tag value: %s and %s both = %d", prev, name, tag)
		}
		seen[tag] = name
	}
}

func TestValueTagMethod(t *testing.T) {
	cases := []struct {
		val  Value
		want int
	}{
		{NullValue{}, NullTag},
		{IntValue{Val: 42}, IntTag},
		{FloatValue{Val: 3.14}, FloatTag},
		{StringValue{Val: "hello"}, StringTag},
		{BoolValue{Val: true}, BoolTag},
		{ArrayValue{Val: []Value{}}, ArrayTag},
		{ObjectValue{Val: map[string]Value{}}, ObjectTag},
		{FunctionValue{Name: "f"}, FuncTag},
		{&FutureValue{Done: make(chan struct{})}, FutureTag},
		{&ResultValue{Val: IntValue{Val: 1}}, ResultTag},
		{PointerValue{Address: 0x10000}, RefPtrTag},
	}

	for _, tc := range cases {
		got := tc.val.Tag()
		if got != tc.want {
			t.Errorf("%T.Tag() = %d, want %d", tc.val, got, tc.want)
		}
	}
}

func TestValueTagConsistentWithType(t *testing.T) {
	// Verify Tag() and Type() are consistent for every value type
	cases := []struct {
		val      Value
		typeStr  string
	}{
		{NullValue{}, "null"},
		{IntValue{Val: 0}, "int"},
		{FloatValue{Val: 0}, "float"},
		{StringValue{Val: ""}, "str"},
		{BoolValue{Val: false}, "bool"},
		{ArrayValue{Val: nil}, "array"},
		{ObjectValue{Val: nil}, "object"},
		{FunctionValue{}, "function"},
		{&FutureValue{Done: make(chan struct{})}, "future"},
		{&ResultValue{}, "result"},
		{PointerValue{}, "ptr"},
	}
	for _, tc := range cases {
		if tc.val.Type() != tc.typeStr {
			t.Errorf("%T.Type() = %q, want %q", tc.val, tc.val.Type(), tc.typeStr)
		}
	}
}
