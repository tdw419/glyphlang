package vm

import (
	"fmt"
	"sync"
)

// ProcessState represents the lifecycle state of a process.
type ProcessState int

const (
	ProcessReady   ProcessState = iota // Created, not yet running
	ProcessRunning                      // Currently executing
	ProcessBlocked                      // Waiting (e.g., OpWait)
	ProcessZombie                       // Exited, awaiting reaping
)

func (s ProcessState) String() string {
	switch s {
	case ProcessReady:
		return "ready"
	case ProcessRunning:
		return "running"
	case ProcessBlocked:
		return "blocked"
	case ProcessZombie:
		return "zombie"
	default:
		return "unknown"
	}
}

// Process represents a single VM process with its own execution context.
type Process struct {
	PID       uint32       // Process identifier
	PPID      uint32       // Parent process identifier
	State     ProcessState // Current lifecycle state
	ExitCode  int64        // Exit code (valid when State == Zombie)
	VM        *VM          // The VM instance running this process
	ChildPIDs []uint32     // List of child process PIDs
}

// ProcessTable is a concurrent-safe map of PID -> Process.
// It handles PID allocation, registration, lookup, reparenting, and cleanup.
type ProcessTable struct {
	mu           sync.RWMutex
	processes    map[uint32]*Process
	nextPID      uint32
	waitChans    map[uint32]chan struct{} // PID -> channel signaled when process becomes zombie
	channelTable *ChannelTable           // lazily initialized on first channel creation
	shmTable     *SharedMemoryTable      // lazily initialized on first shm_create
	syncTable    *SyncTable              // lazily initialized on first mutex/sem create
}

// NewProcessTable creates an empty process table with PID allocation starting at 1.
func NewProcessTable() *ProcessTable {
	return &ProcessTable{
		processes: make(map[uint32]*Process),
		nextPID:   1,
		waitChans: make(map[uint32]chan struct{}),
	}
}

// AllocatePID returns the next available PID. PID 0 is reserved (kernel/init parent).
func (pt *ProcessTable) AllocatePID() uint32 {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pid := pt.nextPID
	pt.nextPID++
	return pid
}

// Register adds a process to the table.
// It also advances nextPID past the registered PID to avoid collisions.
func (pt *ProcessTable) Register(proc *Process) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processes[proc.PID] = proc
	if proc.PID >= pt.nextPID {
		pt.nextPID = proc.PID + 1
	}
}

// Get retrieves a process by PID. Returns nil, false if not found.
func (pt *ProcessTable) Get(pid uint32) (*Process, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	proc, ok := pt.processes[pid]
	return proc, ok
}

// Remove removes a process from the table (used when reaping zombies).
func (pt *ProcessTable) Remove(pid uint32) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	delete(pt.processes, pid)
	delete(pt.waitChans, pid)
}

// Children returns all processes whose PPID matches the given parent PID.
func (pt *ProcessTable) Children(ppid uint32) []*Process {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var children []*Process
	for _, proc := range pt.processes {
		if proc.PPID == ppid {
			children = append(children, proc)
		}
	}
	return children
}

// Reparent moves all children of oldParent to newParent.
// Updates each child's PPID to newParent and transfers ChildPIDs entries.
func (pt *ProcessTable) Reparent(oldParent, newParent uint32) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var orphanPIDs []uint32
	for _, proc := range pt.processes {
		if proc.PPID == oldParent {
			proc.PPID = newParent
			orphanPIDs = append(orphanPIDs, proc.PID)
		}
	}

	// Clear old parent's ChildPIDs (it's dying)
	if oldProc, ok := pt.processes[oldParent]; ok {
		oldProc.ChildPIDs = nil
	}

	// Add orphans to new parent's ChildPIDs
	if newProc, ok := pt.processes[newParent]; ok {
		newProc.ChildPIDs = append(newProc.ChildPIDs, orphanPIDs...)
	}
}

// SetZombie transitions a process to zombie state with the given exit code.
// It also clears the process's VM resources, removes it from its parent's
// ChildPIDs, and notifies any waiters.
func (pt *ProcessTable) SetZombie(pid uint32, exitCode int64) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	proc, ok := pt.processes[pid]
	if !ok {
		return fmt.Errorf("process %d not found", pid)
	}

	proc.State = ProcessZombie
	proc.ExitCode = exitCode

	// Clean up VM resources
	if proc.VM != nil {
		proc.VM.stack = proc.VM.stack[:0]
		proc.VM.halted = true
	}

	// Remove this process from its parent's ChildPIDs
	if proc.PPID != 0 {
		if parent, parentOK := pt.processes[proc.PPID]; parentOK {
			newChildren := make([]uint32, 0, len(parent.ChildPIDs))
			for _, childPID := range parent.ChildPIDs {
				if childPID != pid {
					newChildren = append(newChildren, childPID)
				}
			}
			parent.ChildPIDs = newChildren
		}
	}

	// Notify any process waiting on this PID
	if ch, exists := pt.waitChans[pid]; exists {
		close(ch)
		delete(pt.waitChans, pid)
	}

	return nil
}

// RegisterWaiter creates or returns a channel that will be closed when
// the given PID transitions to zombie state.
func (pt *ProcessTable) RegisterWaiter(pid uint32) <-chan struct{} {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Check if already zombie
	if proc, ok := pt.processes[pid]; ok && proc.State == ProcessZombie {
		ch := make(chan struct{})
		close(ch)
		return ch
	}

	ch, exists := pt.waitChans[pid]
	if !exists {
		ch = make(chan struct{})
		pt.waitChans[pid] = ch
	}
	return ch
}

// notifyWaiter closes the wait channel for the given PID.
// This is used by tests to simulate async process completion.
func (pt *ProcessTable) notifyWaiter(pid uint32) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if ch, exists := pt.waitChans[pid]; exists {
		close(ch)
		delete(pt.waitChans, pid)
	}
}
