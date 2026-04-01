package vm

import (
	"fmt"
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
	ptr, ok := result.(PointerValue)
	if !ok {
		t.Fatalf("expected PointerValue, got %T", result)
	}
	if ptr.Address == 0 {
		t.Errorf("expected non-zero heap pointer")
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

// --- Capability-based permission tests (SEC-2) ---

// TestCapabilityConstants verifies the capability bit values.
func TestCapabilityConstants(t *testing.T) {
	tests := []struct {
		name string
		cap  uint16
		bit  int
	}{
		{"CAP_FS", CAP_FS, 0},
		{"CAP_PROC", CAP_PROC, 1},
		{"CAP_MEM", CAP_MEM, 2},
		{"CAP_IPC", CAP_IPC, 3},
		{"CAP_IO", CAP_IO, 4},
		{"CAP_TIME", CAP_TIME, 5},
		{"CAP_GPU", CAP_GPU, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := uint16(1) << tt.bit
			if tt.cap != expected {
				t.Errorf("%s = 0x%04X, want 0x%04X (bit %d)", tt.name, tt.cap, expected, tt.bit)
			}
		})
	}
	// Verify CAP_ALL includes all capabilities
	if CAP_ALL != (CAP_FS | CAP_PROC | CAP_MEM | CAP_IPC | CAP_IO | CAP_TIME | CAP_GPU) {
		t.Errorf("CAP_ALL = 0x%04X, expected all 7 bits set", CAP_ALL)
	}
}

// TestCapabilityDefaults verifies that NewVM grants all capabilities by default.
func TestCapabilityDefaults(t *testing.T) {
	vm := NewVM()
	if vm.Capabilities() != CAP_ALL {
		t.Errorf("new VM capabilities = 0x%04X, want CAP_ALL 0x%04X", vm.Capabilities(), CAP_ALL)
	}
}

// TestSetCapabilities verifies SetCapabilities and Capabilities round-trip.
func TestSetCapabilities(t *testing.T) {
	vm := NewVM()
	vm.SetCapabilities(CAP_FS | CAP_MEM)
	if got := vm.Capabilities(); got != CAP_FS|CAP_MEM {
		t.Errorf("Capabilities() = 0x%04X, want 0x%04X", got, CAP_FS|CAP_MEM)
	}
}

// TestEPERMRejectedWhenCapMissing tests that syscalls without the required
// capability are rejected with EPERM.
func TestEPERMRejectedWhenCapMissing(t *testing.T) {
	// Test that sys_time requires CAP_TIME
	vm := NewVM()
	vm.SetCapabilities(0) // no capabilities
	code := []byte{byte(OpSyscall), SysTime, byte(OpHalt)}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Fatal("expected EPERM error, got nil")
	}
	if !strings.Contains(err.Error(), "EPERM") {
		t.Errorf("expected EPERM error, got: %v", err)
	}
}

// TestEPERMAllowedWhenCapPresent tests that syscalls succeed when the
// required capability is present.
func TestEPERMAllowedWhenCapPresent(t *testing.T) {
	// sys_time requires CAP_TIME
	vm := NewVM()
	vm.SetCapabilities(CAP_TIME)
	code := []byte{byte(OpSyscall), SysTime, byte(OpHalt)}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("expected success with CAP_TIME, got: %v", err)
	}
	if _, ok := result.(IntValue); !ok {
		t.Errorf("expected IntValue, got %T", result)
	}
}

// TestEPERMForEachCapability tests EPERM rejection for each capability group.
func TestEPERMForEachCapability(t *testing.T) {
	tests := []struct {
		name      string
		syscallNr byte
		required  uint16
	}{
		{"SysRead_requires_CAP_FS", SysRead, CAP_FS},
		{"SysWrite_requires_CAP_FS", SysWrite, CAP_FS},
		{"SysOpen_requires_CAP_FS", SysOpen, CAP_FS},
		{"SysClose_requires_CAP_FS", SysClose, CAP_FS},
		{"SysSpawn_requires_CAP_PROC", SysSpawn, CAP_PROC},
		{"SysKill_requires_CAP_PROC", SysKill, CAP_PROC},
		{"SysWait_requires_CAP_PROC", SysWait, CAP_PROC},
		{"SysSignal_requires_CAP_PROC", SysSignal, CAP_PROC},
		{"SysAlloc_requires_CAP_MEM", SysAlloc, CAP_MEM},
		{"SysFree_requires_CAP_MEM", SysFree, CAP_MEM},
		{"SysSend_requires_CAP_IPC", SysSend, CAP_IPC},
		{"SysRecv_requires_CAP_IPC", SysRecv, CAP_IPC},
		{"SysPrint_requires_CAP_IO", SysPrint, CAP_IO},
		{"SysTime_requires_CAP_TIME", SysTime, CAP_TIME},
		{"SysGPUDispatch_requires_CAP_GPU", SysGPUDispatch, CAP_GPU},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVM()
			// Grant all capabilities EXCEPT the required one
			vm.SetCapabilities(CAP_ALL & ^tt.required)
			code := []byte{byte(OpSyscall), tt.syscallNr, byte(OpHalt)}
			_, err := vm.executeRaw(code)
			if err == nil {
				t.Errorf("expected EPERM for syscall 0x%02x without cap 0x%04X", tt.syscallNr, tt.required)
			} else if !strings.Contains(err.Error(), "EPERM") {
				t.Errorf("expected EPERM, got: %v", err)
			}
		})
	}
}

// TestSysExitNoCapabilityRequired tests that sys_exit (0x0E) requires no
// capability — any process may exit.
func TestSysExitNoCapabilityRequired(t *testing.T) {
	vm := NewVM()
	vm.SetCapabilities(0) // no capabilities at all
	vm.Push(IntValue{Val: 0})
	code := []byte{byte(OpSyscall), SysExit}
	_, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("sys_exit should work without any capability, got: %v", err)
	}
	if !vm.halted {
		t.Error("expected VM to be halted after sys_exit")
	}
}

// TestCapabilityInheritanceOnSpawn tests that a spawned child inherits
// the parent's capabilities.
func TestCapabilityInheritanceOnSpawn(t *testing.T) {
	parentVM := NewVM()
	pt := NewProcessTable()
	parentVM.processTable = pt

	// Register parent process
	parentProc := &Process{
		PID:   1,
		PPID:  0,
		State: ProcessRunning,
		VM:    parentVM,
		Caps:  CAP_FS | CAP_MEM, // restricted capabilities
	}
	pt.Register(parentProc)
	parentVM.pid = 1
	parentVM.SetCapabilities(CAP_FS | CAP_MEM)

	// Set up bytecode for parent: OpSpawn then Halt
	constants := []Value{}
	bytecode := createBytecodeHeader(constants)
	bytecode = addInstruction(bytecode, OpSpawn, nil)
	bytecode = addInstruction(bytecode, OpHalt, nil)

	parentVM.code = bytecode
	layout, _ := parseBytecodeLayout(bytecode)
	parentVM.pc = int(layout.CodeOffset)

	// Execute spawn
	err := parentVM.executeInstruction(OpSpawn)
	if err != nil {
		t.Fatalf("OpSpawn error: %v", err)
	}

	// Get child PID from parent stack
	childPIDVal, _ := parentVM.Pop()
	childPID := uint32(childPIDVal.(IntValue).Val)

	// Verify child process exists and inherited capabilities
	childProc, ok := pt.Get(childPID)
	if !ok {
		t.Fatalf("child process %d not found in process table", childPID)
	}

	if childProc.Caps != CAP_FS|CAP_MEM {
		t.Errorf("child capabilities = 0x%04X, want 0x%04X (inherited from parent)", childProc.Caps, CAP_FS|CAP_MEM)
	}
	if childProc.VM.Capabilities() != CAP_FS|CAP_MEM {
		t.Errorf("child VM capabilities = 0x%04X, want 0x%04X", childProc.VM.Capabilities(), CAP_FS|CAP_MEM)
	}
}

// --- SEC-3: Comprehensive syscall test suite ---

// TestSyscallDispatchEachNumber verifies that each registered syscall number
// (0x00-0x0F) has a handler in the dispatch table.
func TestSyscallDispatchEachNumber(t *testing.T) {
	// Force initialization of the syscall table
	syscallTableOnce.Do(initSyscallTable)

	for nr := byte(0x00); nr <= 0x0F; nr++ {
		t.Run(fmt.Sprintf("syscall_0x%02x", nr), func(t *testing.T) {
			handler := syscallTable[nr]
			if handler == nil {
				t.Errorf("syscall 0x%02x has no handler registered", nr)
			}
		})
	}
}

// TestSyscallENOSYSAllUnregistered verifies that syscall numbers 0x10-0xFF
// all return ENOSYS.
func TestSyscallENOSYSAllUnregistered(t *testing.T) {
	vm := NewVM()
	// Test a representative sample of the unregistered range
	for _, num := range []byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xA0, 0xB0, 0xC0, 0xD0, 0xE0, 0xF0, 0xFE, 0xFF} {
		t.Run(fmt.Sprintf("0x%02x", num), func(t *testing.T) {
			code := []byte{byte(OpSyscall), num, byte(OpHalt)}
			_, err := vm.executeRaw(code)
			if err == nil {
				t.Errorf("expected ENOSYS error for syscall 0x%02x, got nil", num)
			} else if !strings.Contains(err.Error(), "ENOSYS") {
				t.Errorf("expected ENOSYS for syscall 0x%02x, got: %v", num, err)
			}
		})
	}
}

// TestSyscallCapabilityRejectionSweep tests that every capability-requiring
// syscall is rejected when the corresponding capability is missing, and
// succeeds when the capability is present.
func TestSyscallCapabilityRejectionSweep(t *testing.T) {
	tests := []struct {
		name      string
		syscallNr byte
		required  uint16
		prePush   []Value // values to push before the syscall
	}{
		{"SysPrint_0x0C_requires_CAP_IO", SysPrint, CAP_IO, []Value{StringValue{Val: "x"}}},
		{"SysAlloc_0x08_requires_CAP_MEM", SysAlloc, CAP_MEM, []Value{IntValue{Val: 16}}},
		{"SysFree_0x09_requires_CAP_MEM", SysFree, CAP_MEM, nil}, // pointer validity checked after cap check
		{"SysTime_0x0D_requires_CAP_TIME", SysTime, CAP_TIME, nil},
		{"SysExit_0x0E_no_cap_required", SysExit, 0, []Value{IntValue{Val: 0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name+"_rejected", func(t *testing.T) {
			vm := NewVM()
			// Grant all capabilities EXCEPT the required one
			if tt.required != 0 {
				vm.SetCapabilities(CAP_ALL & ^tt.required)
			} else {
				vm.SetCapabilities(0) // test with no caps for exit
			}
			for _, v := range tt.prePush {
				vm.Push(v)
			}
			code := []byte{byte(OpSyscall), tt.syscallNr, byte(OpHalt)}
			_, err := vm.executeRaw(code)
			if tt.required == 0 {
				// sys_exit should work even with no capabilities
				if err != nil && !vm.halted {
					t.Errorf("expected success for cap-free syscall, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected EPERM for syscall 0x%02x without cap 0x%04X", tt.syscallNr, tt.required)
				} else if !strings.Contains(err.Error(), "EPERM") {
					t.Errorf("expected EPERM, got: %v", err)
				}
			}
		})
		t.Run(tt.name+"_allowed", func(t *testing.T) {
			vm := NewVM()
			if tt.required != 0 {
				vm.SetCapabilities(tt.required)
			}
			for _, v := range tt.prePush {
				vm.Push(v)
			}
			code := []byte{byte(OpSyscall), tt.syscallNr, byte(OpHalt)}
			_, err := vm.executeRaw(code)
			// Should not fail with EPERM (may fail with "not implemented" for stubs)
			if err != nil && strings.Contains(err.Error(), "EPERM") {
				t.Errorf("expected no EPERM with cap 0x%04X, got: %v", tt.required, err)
			}
		})
	}
}

// TestSyscallPrintFull verifies sys_print output behavior in detail.
func TestSyscallPrintFull(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: 42})
	code := []byte{byte(OpSyscall), SysPrint, byte(OpHalt)}
	result, err := vm.executeRaw(code)
	if err != nil {
		t.Fatalf("sys_print error: %v", err)
	}
	if _, ok := result.(NullValue); !ok {
		t.Errorf("sys_print should push NullValue, got %T", result)
	}
}

// TestSyscallAllocSizeValidation tests that sys_alloc rejects invalid sizes.
func TestSyscallAllocSizeValidation(t *testing.T) {
	vm := NewVM()
	vm.Push(StringValue{Val: "not a number"})
	code := []byte{byte(OpSyscall), SysAlloc, byte(OpHalt)}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Error("expected error for non-integer size")
	}
}

// TestSyscallFreeInvalidPointer tests that sys_free rejects non-pointer values.
func TestSyscallFreeInvalidPointer(t *testing.T) {
	vm := NewVM()
	vm.Push(IntValue{Val: 999})
	code := []byte{byte(OpSyscall), SysFree, byte(OpHalt)}
	_, err := vm.executeRaw(code)
	if err == nil {
		t.Error("expected error for non-pointer free")
	}
}

// TestSyscallRequiredCapMappingComplete verifies that the syscallRequiredCap
// array is populated correctly for all 16 defined syscalls.
func TestSyscallRequiredCapMappingComplete(t *testing.T) {
	expected := [256]uint16{}
	expected[SysRead] = CAP_FS
	expected[SysWrite] = CAP_FS
	expected[SysOpen] = CAP_FS
	expected[SysClose] = CAP_FS
	expected[SysSpawn] = CAP_PROC
	expected[SysKill] = CAP_PROC
	expected[SysWait] = CAP_PROC
	expected[SysSignal] = CAP_PROC
	expected[SysAlloc] = CAP_MEM
	expected[SysFree] = CAP_MEM
	expected[SysSend] = CAP_IPC
	expected[SysRecv] = CAP_IPC
	expected[SysPrint] = CAP_IO
	expected[SysTime] = CAP_TIME
	// SysExit = 0 (no cap required)
	expected[SysGPUDispatch] = CAP_GPU

	for i := 0; i < 256; i++ {
		if syscallRequiredCap[i] != expected[i] {
			t.Errorf("syscallRequiredCap[0x%02x] = 0x%04X, want 0x%04X", i, syscallRequiredCap[i], expected[i])
		}
	}
}
