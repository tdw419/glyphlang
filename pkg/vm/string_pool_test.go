package vm

import (
	"testing"
)

func TestOpLoadString(t *testing.T) {
	vm := NewVM()
	vm.stringPool = []string{"hello", "world", ""}

	// Push string from pool index 0
	vm.execLoadString()
	// Should fail because we haven't set up the instruction stream
	// Let's test via executeRaw instead
}

func TestLoadStringViaExecuteRaw(t *testing.T) {
	// Build instructions: OpLoadString(0), OpHalt
	code := []byte{byte(OpLoadString), 0x00, 0x00, 0x00, 0x00, byte(OpHalt)}

	vm := NewVM()
	vm.stringPool = []string{"hello world"}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("executeRaw error: %v", err)
	}

	s, ok := result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T: %v", result, result)
	}
	if s.Val != "hello world" {
		t.Errorf("expected 'hello world', got %q", s.Val)
	}
}

func TestLoadStringMultiple(t *testing.T) {
	// Load two strings and concatenate them via OP_ADD
	code := []byte{
		byte(OpLoadString), 0x00, 0x00, 0x00, 0x00, // "hello, "
		byte(OpLoadString), 0x01, 0x00, 0x00, 0x00, // "world"
		byte(OpAdd),
		byte(OpHalt),
	}

	vm := NewVM()
	vm.stringPool = []string{"hello, ", "world"}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("executeRaw error: %v", err)
	}

	s, ok := result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T: %v", result, result)
	}
	if s.Val != "hello, world" {
		t.Errorf("expected 'hello, world', got %q", s.Val)
	}
}

func TestLoadStringOutOfBounds(t *testing.T) {
	code := []byte{byte(OpLoadString), 0x05, 0x00, 0x00, 0x00, byte(OpHalt)}

	vm := NewVM()
	vm.stringPool = []string{"hello"}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Fatal("expected error for out-of-bounds string pool index")
	}
}

func TestLoadStringEmptyPool(t *testing.T) {
	code := []byte{byte(OpLoadString), 0x00, 0x00, 0x00, 0x00, byte(OpHalt)}

	vm := NewVM()
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Fatal("expected error for empty string pool")
	}
}

func TestStringPoolInClone(t *testing.T) {
	vm := NewVM()
	vm.stringPool = []string{"alpha", "beta"}

	cloned := vm.Clone()
	if len(cloned.stringPool) != 2 {
		t.Fatalf("expected 2 strings in cloned pool, got %d", len(cloned.stringPool))
	}
	if cloned.stringPool[0] != "alpha" || cloned.stringPool[1] != "beta" {
		t.Errorf("cloned pool content mismatch: %v", cloned.stringPool)
	}
}

func TestStringPoolInBytecode(t *testing.T) {
	// Test that string pool is embedded in bytecode and parsed correctly
	// Build a full bytecode blob with string pool
	buf := []byte("GLYP")                        // magic
	buf = append(buf, 0x01, 0x00, 0x00, 0x00)    // version 1
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)    // constant count = 0

	// String pool: count = 2
	pool := []byte{0x02, 0x00, 0x00, 0x00}
	// String 0: "foo" (length 3)
	pool = append(pool, 0x03, 0x00, 0x00, 0x00)
	pool = append(pool, "foo"...)
	// String 1: "bar" (length 3)
	pool = append(pool, 0x03, 0x00, 0x00, 0x00)
	pool = append(pool, "bar"...)
	buf = append(buf, pool...)

	// Instruction count
	buf = append(buf, 0x06, 0x00, 0x00, 0x00)

	// Instructions: OpLoadString(0), OpLoadString(1), OpAdd, OpHalt
	buf = append(buf, byte(OpLoadString), 0x00, 0x00, 0x00, 0x00)
	buf = append(buf, byte(OpLoadString), 0x01, 0x00, 0x00, 0x00)
	buf = append(buf, byte(OpAdd))
	buf = append(buf, byte(OpHalt))

	vm := NewVM()
	result, err := vm.Execute(buf)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	s, ok := result.(StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if s.Val != "foobar" {
		t.Errorf("expected 'foobar', got %q", s.Val)
	}
}
