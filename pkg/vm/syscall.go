package vm

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Syscall numbers: 0x00-0x0F are defined. 0x10-0xFF return ENOSYS.
// OpSyscall (0xDD) is defined in vm.go.
const (
	SysRead        = 0x00 // sys_read(fd, buf, count)    - VFS read
	SysWrite       = 0x01 // sys_write(fd, buf, count)   - VFS write
	SysOpen        = 0x02 // sys_open(path, flags)       - VFS open
	SysClose       = 0x03 // sys_close(fd)               - VFS close
	SysSpawn       = 0x04 // sys_spawn(bytecode_ptr)     - Process spawn
	SysKill        = 0x05 // sys_kill(pid)               - Process kill
	SysWait        = 0x06 // sys_wait(pid)               - Process wait
	SysSignal      = 0x07 // sys_signal(pid, sig)        - Process signal
	SysAlloc       = 0x08 // sys_alloc(size)             - Heap allocate
	SysFree        = 0x09 // sys_free(ptr)               - Heap free
	SysSend        = 0x0A // sys_send(channel, value)    - IPC send
	SysRecv        = 0x0B // sys_recv(channel)           - IPC receive
	SysPrint       = 0x0C // sys_print(value)            - Console output
	SysTime        = 0x0D // sys_time()                  - Current timestamp
	SysExit        = 0x0E // sys_exit(code)              - Process exit
	SysGPUDispatch = 0x0F // sys_gpu_dispatch(...)       - GPU kernel dispatch
)

// SyscallHandler is the signature for syscall handler functions.
// It receives the VM (to access stack/environment) and returns a result
// value and an error. The handler is responsible for popping its
// arguments from the stack and pushing the result.
type SyscallHandler func(vm *VM) (Value, error)

// syscallRequiredCap maps each syscall number to the capability it requires.
// SysExit (0x0E) requires no capability — any process may exit.
var syscallRequiredCap [256]uint16

func init() {
	// FS: syscalls 0x00-0x03
	syscallRequiredCap[SysRead] = CAP_FS
	syscallRequiredCap[SysWrite] = CAP_FS
	syscallRequiredCap[SysOpen] = CAP_FS
	syscallRequiredCap[SysClose] = CAP_FS
	// PROC: syscalls 0x04-0x07
	syscallRequiredCap[SysSpawn] = CAP_PROC
	syscallRequiredCap[SysKill] = CAP_PROC
	syscallRequiredCap[SysWait] = CAP_PROC
	syscallRequiredCap[SysSignal] = CAP_PROC
	// MEM: syscalls 0x08-0x09
	syscallRequiredCap[SysAlloc] = CAP_MEM
	syscallRequiredCap[SysFree] = CAP_MEM
	// IPC: syscalls 0x0A-0x0B
	syscallRequiredCap[SysSend] = CAP_IPC
	syscallRequiredCap[SysRecv] = CAP_IPC
	// IO: syscall 0x0C
	syscallRequiredCap[SysPrint] = CAP_IO
	// TIME: syscall 0x0D
	syscallRequiredCap[SysTime] = CAP_TIME
	// SysExit (0x0E) requires no capability — left as 0
	// GPU: syscall 0x0F
	syscallRequiredCap[SysGPUDispatch] = CAP_GPU
}

// syscallTable is the dispatch table indexed by syscall number.
// Entries 0x00-0x0F are registered during init. Unregistered entries
// return ENOSYS.
var syscallTable [256]SyscallHandler

// syscallTableOnce ensures the syscall table is initialized exactly once.
var syscallTableOnce sync.Once

// initSyscallTable registers handlers for syscalls 0x00-0x0F.
func initSyscallTable() {
	// sys_print (0x0C): pops one value, prints it to stdout, pushes null.
	syscallTable[SysPrint] = func(vm *VM) (Value, error) {
		val, err := vm.Pop()
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(os.Stdout, valueToString(val))
		return NullValue{}, nil
	}

	// sys_alloc (0x08): pops size, allocates heap block, pushes pointer.
	syscallTable[SysAlloc] = func(vm *VM) (Value, error) {
		sizeVal, err := vm.Pop()
		if err != nil {
			return nil, err
		}
		size, ok := sizeVal.(IntValue)
		if !ok {
			return nil, fmt.Errorf("sys_alloc: size must be an integer, got %T", sizeVal)
		}
		ptr := vm.heap.Alloc(uint32(size.Val))
		return IntValue{Val: int64(ptr)}, nil
	}

	// sys_free (0x09): pops pointer, frees heap block, pushes null.
	syscallTable[SysFree] = func(vm *VM) (Value, error) {
		ptrVal, err := vm.Pop()
		if err != nil {
			return nil, err
		}
		ptr, ok := ptrVal.(IntValue)
		if !ok {
			return nil, fmt.Errorf("sys_free: pointer must be an integer, got %T", ptrVal)
		}
		if err := vm.heap.Free(uint32(ptr.Val)); err != nil {
			return nil, err
		}
		return NullValue{}, nil
	}

	// sys_time (0x0D): pushes current Unix timestamp.
	syscallTable[SysTime] = func(vm *VM) (Value, error) {
		return IntValue{Val: time.Now().Unix()}, nil
	}

	// sys_exit (0x0E): pops exit code, halts the VM.
	syscallTable[SysExit] = func(vm *VM) (Value, error) {
		_, err := vm.Pop()
		if err != nil {
			return nil, err
		}
		vm.halted = true
		return NullValue{}, nil
	}

	// SysRead (0x00) through SysClose (0x03): VFS stubs - return ENOSYS
	// until VFS integration (depends on change 007).
	for _, nr := range []byte{SysRead, SysWrite, SysOpen, SysClose} {
		nr := nr
		syscallTable[nr] = func(vm *VM) (Value, error) {
			return nil, fmt.Errorf("syscall 0x%02x: not implemented (VFS not integrated)", nr)
		}
	}

	// SysSpawn (0x04) through SysSignal (0x07): Process stubs
	for _, nr := range []byte{SysSpawn, SysKill, SysWait, SysSignal} {
		nr := nr
		syscallTable[nr] = func(vm *VM) (Value, error) {
			return nil, fmt.Errorf("syscall 0x%02x: not implemented (process model not integrated)", nr)
		}
	}

	// SysSend (0x0A), SysRecv (0x0B): IPC stubs
	for _, nr := range []byte{SysSend, SysRecv} {
		nr := nr
		syscallTable[nr] = func(vm *VM) (Value, error) {
			return nil, fmt.Errorf("syscall 0x%02x: not implemented (IPC not integrated)", nr)
		}
	}

	// SysGPUDispatch (0x0F): GPU stub
	syscallTable[SysGPUDispatch] = func(vm *VM) (Value, error) {
		return nil, fmt.Errorf("syscall 0x0f: not implemented (GPU dispatch not integrated)")
	}
}

// execSyscall reads the next byte as the syscall number, looks up the handler
// in the dispatch table, and calls it. Unregistered syscalls return ENOSYS.
// The encoding is: [0xDD, syscall_number] — a 2-byte sequence.
// Capability checking: if the VM's capability bitmask does not include the
// required capability for this syscall, the call is rejected with EPERM.
func (vm *VM) execSyscall() error {
	if vm.pc >= len(vm.code) {
		return fmt.Errorf("truncated syscall: missing syscall number at pc=%d", vm.pc)
	}

	syscallNum := vm.code[vm.pc]
	vm.pc++

	// Capability check: reject if the VM lacks the required capability.
	if required := syscallRequiredCap[syscallNum]; required != 0 {
		if vm.capabilities&required == 0 {
			return fmt.Errorf("EPERM: syscall 0x%02x requires capability 0x%04X, process has 0x%04X", syscallNum, required, vm.capabilities)
		}
	}

	handler := syscallTable[syscallNum]
	if handler == nil {
		return fmt.Errorf("ENOSYS: syscall 0x%02x not registered", syscallNum)
	}

	result, err := handler(vm)
	if err != nil {
		return err
	}
	vm.Push(result)
	return nil
}

// SetCapabilities sets the capability bitmask for the VM.
func (vm *VM) SetCapabilities(caps uint16) {
	vm.capabilities = caps
}

// Capabilities returns the current capability bitmask.
func (vm *VM) Capabilities() uint16 {
	return vm.capabilities
}
