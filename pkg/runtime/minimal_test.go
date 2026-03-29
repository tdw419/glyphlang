package runtime

import (
	"testing"
)

func TestVM_PushPop(t *testing.T) {
	vm := NewVM()
	val := Value{Type: TypeInt, Int: 42}
	vm.push(val)

	if vm.sp != 1 {
		t.Fatalf("expected stack size 1, got %d", vm.sp)
	}

	popped := vm.pop()
	if popped.Type != TypeInt || popped.Int != 42 {
		t.Errorf("expected 42, got %v", popped.Int)
	}

	if vm.sp != 0 {
		t.Errorf("expected stack size 0, got %d", vm.sp)
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Errorf("expected 1 for true")
	}
	if boolToInt(false) != 0 {
		t.Errorf("expected 0 for false")
	}
}
