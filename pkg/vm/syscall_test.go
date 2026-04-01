package vm

import (
	"strings"
	"testing"
)

// TestSyscallOpCode verifies OpSyscall is 0xDD.
func TestSyscallOpCode(t *testing.T) {
	if OpSyscall != 0xDD {
		t.Errorf("OpSyscall = 0x%02X, want 0xDD", OpSyscall)
	}
}

// TestSyscallNumbers verifies the syscall number constants.
func TestSyscallNumbers(t *testing.T) {
	tests := []struct {
		name     string
		number   byte
		expected byte
	}{
		{"SysRead", SysRead, 0x00},
		{"SysWrite", SysWrite, 0x01},
		{"SysOpen", SysOpen, 0x02},
		{"SysClose", SysClose, 0x03},
		{"SysSpawn", SysSpawn, 0x04},
		{"SysKill", SysKill, 0x05},
		{"SysWait", SysWait, 0x06},
		{"SysSignal", SysSignal, 0x07},
		{"SysAlloc", SysAlloc, 0x08},
		{"SysFree", SysFree, 0x09},
		{"SysSend", SysSend, 0x0A},
		{"SysRecv", SysRecv, 0x0B},
		{"SysPrint", SysPrint, 0x0C},
		{"SysTime", SysTime, 0x0D},
		{"SysExit", SysExit, 0x0E},
		{"SysGPUDispatch", SysGPUDispatch, 0x0F},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.number != tt.expected {
				t.Errorf("%s = 0x%02X, want 0x%02X", tt.name, tt.number, tt.expected)
			}
		})
	}
}

// TestSyscallTime tests sys_time (0x0D) through the syscall dispatch.
func TestSyscallTime(t *testing.T) {
	vm := NewVM()
	// Raw bytecode: [OpSyscall, SysTime, OpHalt]
	code := []byte{byte(OpSyscall), SysTime, byte(OpHalt)}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("executeRaw error: %v", err)
	}
	intVal, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Val <= 0 {
		t.Errorf("expected positive timestamp, got %d", intVal.Val)
	}
}

// TestSyscallPrint tests sys_print (0x0C) pops one value and pushes null.
func TestSyscallPrint(t *testing.T) {
	vm := NewVM()
	vm.Push(StringValue{Val: "hello syscall"})

	// Raw bytecode: [OpSyscall, SysPrint, OpHalt]
	code := []byte{byte(OpSyscall), SysPrint, byte(OpHalt)}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("executeRaw error: %v", err)
	}
	if _, ok := result.(NullValue); !ok {
		t.Errorf("expected NullValue, got %T", result)
	}
}

// TestSyscallExit tests sys_exit (0x0E) halts the VM.
func TestSyscallExit(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: 0}) // exit code

	// Raw bytecode: [OpSyscall, SysExit]
	// No OpHalt needed — sys_exit halts the VM.
	code := []byte{byte(OpSyscall), SysExit}
	_, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("executeRaw error: %v", err)
	}
	if !vm.halted {
		t.Error("expected VM to be halted after sys_exit")
	}
}

// TestSyscallENOSYS tests that unregistered syscall numbers return ENOSYS.
func TestSyscallENOSYS(t *testing.T) {
	vm := NewVM()

	// Use an unregistered syscall number (0x10-0xFF are unregistered)
	for _, num := range []byte{0x10, 0x20, 0x50, 0xFF} {
		code := []byte{byte(OpSyscall), num, byte(OpHalt)}
		_, err := vm.executeRaw(code)
		if err == nil {
			t.Errorf("expected ENOSYS error for syscall 0x%02x, got nil", num)
		} else if !strings.Contains(err.Error(), "ENOSYS") {
			t.Errorf("expected ENOSYS error for syscall 0x%02x, got: %v", num, err)
		}
	}
}

// TestSyscallTruncated tests that truncated syscall bytecode returns error.
func TestSyscallTruncated(t *testing.T) {
	vm := NewVM()
	// Only OpSyscall byte, no syscall number following it
	code := []byte{byte(OpSyscall)}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Error("expected error for truncated syscall, got nil")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("expected truncated error, got: %v", err)
	}
}

// TestSyscallAllocFree tests sys_alloc (0x08) and sys_free (0x09).
func TestSyscallAllocFree(t *testing.T) {
	vm := NewVM()

	// Allocate: push size, then sys_alloc
	vm.Push(IntValue{Val: 64})
	code := []byte{byte(OpSyscall), SysAlloc}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("sys_alloc error: %v", err)
	}
	ptr, ok := result.(IntValue)
	if !ok {
		t.Fatalf("expected IntValue pointer, got %T", result)
	}
	if ptr.Val <= 0 {
		t.Errorf("expected positive heap pointer, got %d", ptr.Val)
	}

	// Free: push the pointer back, then sys_free
	vm.Push(ptr)
	code2 := []byte{byte(OpSyscall), SysFree, byte(OpHalt)}
	_, err = vm.executeRaw(code2)
	if err != nil {
		t.Fatalf("sys_free error: %v", err)
	}
}

// TestSyscallVFStubs tests that VFS syscalls return "not implemented".
func TestSyscallVFStubs(t *testing.T) {
	vm := NewVM()
	for _, num := range []byte{SysRead, SysWrite, SysOpen, SysClose} {
		code := []byte{byte(OpSyscall), num, byte(OpHalt)}
		_, err := vm.executeRaw(code)
		if err == nil {
			t.Errorf("expected error for VFS syscall 0x%02x", num)
		} else if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("expected 'not implemented' for syscall 0x%02x, got: %v", num, err)
		}
	}
}

// TestSyscallProcessStubs tests that process syscalls return "not implemented".
func TestSyscallProcessStubs(t *testing.T) {
	vm := NewVM()
	for _, num := range []byte{SysSpawn, SysKill, SysWait, SysSignal} {
		code := []byte{byte(OpSyscall), num, byte(OpHalt)}
		_, err := vm.executeRaw(code)
		if err == nil {
			t.Errorf("expected error for process syscall 0x%02x", num)
		} else if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("expected 'not implemented' for syscall 0x%02x, got: %v", num, err)
		}
	}
}

// TestSyscallIPCStubs tests that IPC syscalls return "not implemented".
func TestSyscallIPCStubs(t *testing.T) {
	vm := NewVM()
	for _, num := range []byte{SysSend, SysRecv} {
		code := []byte{byte(OpSyscall), num, byte(OpHalt)}
		_, err := vm.executeRaw(code)
		if err == nil {
			t.Errorf("expected error for IPC syscall 0x%02x", num)
		} else if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("expected 'not implemented' for syscall 0x%02x, got: %v", num, err)
		}
	}
}

// TestSyscallGPUStub tests that GPU dispatch syscall returns "not implemented".
func TestSyscallGPUStub(t *testing.T) {
	vm := NewVM()
	code := []byte{byte(OpSyscall), SysGPUDispatch, byte(OpHalt)}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Error("expected error for GPU syscall")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' for GPU syscall, got: %v", err)
	}
}
