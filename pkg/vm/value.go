package vm

import (
	"encoding/json"
	"fmt"
	"time"
)

// Value represents a runtime value

// Environment represents a scope for variables
type Environment struct {
        Values map[string]Value
        Parent *Environment
}


func (e *Environment) Clone() *Environment {
	if e == nil {
		return nil
	}
	newEnv := &Environment{
		Values: make(map[string]Value),
		Parent: e.Parent.Clone(),
	}
	for k, v := range e.Values {
		newEnv.Values[k] = v
	}
	return newEnv
}

func NewEnvironment(parent *Environment) *Environment {
        return &Environment{
                Values: make(map[string]Value),
                Parent: parent,
        }
}

type Value interface {
	Type() string
}

// NullValue represents a null value
type NullValue struct{}

func (v NullValue) Type() string { return "null" }

func (v NullValue) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// IntValue represents an integer
type IntValue struct {
	Val int64
}

func (v IntValue) Type() string { return "int" }

func (v IntValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// FloatValue represents a floating-point number
type FloatValue struct {
	Val float64
}

func (v FloatValue) Type() string { return "float" }

func (v FloatValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// StringValue represents a string
type StringValue struct {
	Val string
}

func (v StringValue) Type() string { return "str" }

func (v StringValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// BoolValue represents a boolean
type BoolValue struct {
	Val bool
}

func (v BoolValue) Type() string { return "bool" }

func (v BoolValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// ArrayValue represents an array
type ArrayValue struct {
	Val []Value
}

func (v ArrayValue) Type() string { return "array" }

func (v ArrayValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// ObjectValue represents an object (map)
type ObjectValue struct {
	Val map[string]Value
}

func (v ObjectValue) Type() string { return "object" }

func (v ObjectValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Val)
}

// FunctionValue represents a compiled function
type FunctionValue struct {
	Name       string   // Function name
	ParamNames []string // Parameter names in order
	CodeOffset int      // Start offset in the bytecode (if Bytecode is nil)
	CodeLength int      // Length of the function body bytecode
	Bytecode   []byte   // Optional: independent bytecode blob for this function
	Constants  []Value  // Optional: constant pool for the independent bytecode
	Closure    *Environment // Captured environment
}

func (v FunctionValue) Type() string { return "function" }

func (v FunctionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   "function",
		"name":   v.Name,
		"params": v.ParamNames,
	})
}

// FutureValue represents an async future
type FutureValue struct {
	Result Value
	Error  error
	Done   chan struct{}
}

func (v *FutureValue) Type() string { return "future" }

func (v *FutureValue) MarshalJSON() ([]byte, error) {
	select {
	case <-v.Done:
		return json.Marshal(v.Result)
	default:
		return []byte(`{"pending":true}`), nil
	}
}

// DefaultAsyncTimeout is the maximum time to wait for an async future.
const DefaultAsyncTimeout = 30 * time.Second

// Await blocks until the future is resolved and returns the result.
// Times out after DefaultAsyncTimeout to prevent goroutine leaks.
func (v *FutureValue) Await() (Value, error) {
	if v.Done != nil {
		select {
		case <-v.Done:
			// Resolved normally
		case <-time.After(DefaultAsyncTimeout):
			return nil, fmt.Errorf("async operation timed out after %v", DefaultAsyncTimeout)
		}
	}
	if v.Error != nil {
		return nil, v.Error
	}
	return v.Result, nil
}

// ResultValue represents a Result type for error handling (Ok or Err)
// Used by the try operator for unwrapping success or propagating errors
type ResultValue struct {
	Val    Value  // The success value (when IsErr is false)
	ErrMsg string // Error message (when IsErr is true)
	IsErr  bool   // True if this is an error result
}

func (v *ResultValue) Type() string { return "result" }

func (v *ResultValue) MarshalJSON() ([]byte, error) {
	if v.IsErr {
		return json.Marshal(map[string]interface{}{"error": v.ErrMsg})
	}
	return json.Marshal(v.Val)
}
