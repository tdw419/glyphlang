package vm

import (
	"fmt"
	"sync"
)

// ChannelID is a unique identifier for a channel, allocated from a global namespace.
type ChannelID uint32

// Channel is a typed circular buffer for IPC message passing between processes.
// It supports configurable capacity (0 for synchronous rendezvous, N for buffered async).
// Sender and receiver wait queues track blocked processes when the buffer is full/empty.
type Channel struct {
	ID       ChannelID
	Capacity int           // 0 = synchronous (rendezvous), N = buffered
	ElemType string        // type name for type checking (empty = untyped)
	buf      []Value       // circular buffer
	head     int           // read position
	tail     int           // write position
	count    int           // number of items in buffer
	mu       sync.Mutex    // protects all mutable state

	// Wait queues: PIDs of processes blocked on send/recv.
	// The scheduler is responsible for actually blocking/unblocking processes;
	// these are advisory lists that the channel subsystem maintains.
	senderWaitQueue []uint32 // PIDs blocked because buffer was full
	recvWaitQueue   []uint32 // PIDs blocked because buffer was empty
}

// newChannel creates a channel with the given ID and capacity.
func newChannel(id ChannelID, capacity int) *Channel {
	return &Channel{
		ID:       id,
		Capacity: capacity,
		buf:      make([]Value, capacity), // zero-size slice for capacity=0 (rendezvous)
	}
}

// Send enqueues a value. Returns (blocked, error).
// blocked=true means the sender should be blocked (buffer full).
// If a receiver was waiting, its PID is returned as wokenReceiver.
func (ch *Channel) Send(val Value, senderPID uint32) (blocked bool, wokenReceiver uint32, err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Rendezvous channel (capacity 0): block sender until a receiver arrives.
	// For now, we treat capacity-0 channels as unbuffered — send always blocks
	// unless there's already a waiting receiver.
	if ch.Capacity == 0 {
		if len(ch.recvWaitQueue) > 0 {
			// Hand off directly: wake the first waiting receiver
			wokenReceiver = ch.recvWaitQueue[0]
			ch.recvWaitQueue = ch.recvWaitQueue[1:]
			// Store value for the receiver to pick up
			ch.buf = append(ch.buf, val)
			ch.count = 1
			return false, wokenReceiver, nil
		}
		// No receiver waiting: block the sender
		ch.senderWaitQueue = append(ch.senderWaitQueue, senderPID)
		// We still need to store the value so a future receiver can grab it
		ch.buf = append(ch.buf, val)
		ch.count = 1
		return true, 0, nil
	}

	// Buffered channel
	if ch.count >= ch.Capacity {
		// Buffer full: block the sender
		ch.senderWaitQueue = append(ch.senderWaitQueue, senderPID)
		return true, 0, nil
	}

	// Enqueue into circular buffer
	ch.buf[ch.tail] = val
	ch.tail = (ch.tail + 1) % ch.Capacity
	ch.count++

	// Wake a blocked receiver if any
	if len(ch.recvWaitQueue) > 0 {
		wokenReceiver = ch.recvWaitQueue[0]
		ch.recvWaitQueue = ch.recvWaitQueue[1:]
	}

	return false, wokenReceiver, nil
}

// Recv dequeues a value. Returns (val, blocked, error).
// blocked=true means the receiver should be blocked (buffer empty).
// If a sender was waiting, its PID is returned as wokenSender.
func (ch *Channel) Recv(receiverPID uint32) (val Value, blocked bool, wokenSender uint32, err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Rendezvous channel (capacity 0): check if a sender left a value
	if ch.Capacity == 0 {
		if ch.count > 0 {
			// A sender left a value for us
			val = ch.buf[0]
			ch.buf = ch.buf[:0]
			ch.count = 0
			// Wake the sender who left this value
			if len(ch.senderWaitQueue) > 0 {
				wokenSender = ch.senderWaitQueue[0]
				ch.senderWaitQueue = ch.senderWaitQueue[1:]
			}
			return val, false, wokenSender, nil
		}
		// Nothing available: block the receiver
		ch.recvWaitQueue = append(ch.recvWaitQueue, receiverPID)
		return nil, true, 0, nil
	}

	// Buffered channel
	if ch.count == 0 {
		// Buffer empty: block the receiver
		ch.recvWaitQueue = append(ch.recvWaitQueue, receiverPID)
		return nil, true, 0, nil
	}

	// Dequeue from circular buffer
	val = ch.buf[ch.head]
	ch.buf[ch.head] = nil // allow GC
	ch.head = (ch.head + 1) % ch.Capacity
	ch.count--

	// Wake a blocked sender if any
	if len(ch.senderWaitQueue) > 0 {
		wokenSender = ch.senderWaitQueue[0]
		ch.senderWaitQueue = ch.senderWaitQueue[1:]
	}

	return val, false, wokenSender, nil
}

// ChannelTable is a concurrent-safe global namespace for channels.
// It handles channel ID allocation, creation, lookup, and cleanup.
type ChannelTable struct {
	mu       sync.RWMutex
	channels map[ChannelID]*Channel
	nextID   ChannelID
}

// NewChannelTable creates an empty channel table with ID allocation starting at 1.
func NewChannelTable() *ChannelTable {
	return &ChannelTable{
		channels: make(map[ChannelID]*Channel),
		nextID:   1,
	}
}

// CreateChannel allocates and registers a new channel with the given capacity.
func (ct *ChannelTable) CreateChannel(capacity int) *Channel {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	id := ct.nextID
	ct.nextID++

	ch := newChannel(id, capacity)
	ct.channels[id] = ch
	return ch
}

// Get retrieves a channel by ID. Returns nil, false if not found.
func (ct *ChannelTable) Get(id ChannelID) (*Channel, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	ch, ok := ct.channels[id]
	return ch, ok
}

// Remove deletes a channel from the table.
func (ct *ChannelTable) Remove(id ChannelID) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	delete(ct.channels, id)
}

// OpChannelCreate (0xD0): create a channel. Operand = capacity (uint32).
// Pushes the channel ID (as IntValue) onto the stack.
const OpChannelCreate Opcode = 0xD0

// OpSend (0xD1): send value to channel.
// Stack: [channel_id, value] -> pops both.
// If buffer full, blocks sender (sets process state to blocked).
const OpSend Opcode = 0xD1

// OpRecv (0xD2): receive value from channel.
// Stack: [channel_id] -> pops channel_id, pushes received value.
// If buffer empty, blocks receiver.
const OpRecv Opcode = 0xD2

// execChannelCreate creates a new channel with the given capacity.
func (vm *VM) execChannelCreate() error {
	if vm.processTable == nil {
		return fmt.Errorf("channel_create: no process table attached to VM")
	}

	operand, err := vm.readOperand()
	if err != nil {
		return err
	}
	capacity := int(operand)

	// Lazily initialize channel table on the process table
	if vm.processTable.channelTable == nil {
		vm.processTable.channelTable = NewChannelTable()
	}

	ch := vm.processTable.channelTable.CreateChannel(capacity)
	vm.Push(IntValue{Val: int64(ch.ID)})
	return nil
}

// execSend sends a value to a channel.
// Stack layout: value (top), channel_id (below)
func (vm *VM) execSend() error {
	if vm.processTable == nil {
		return fmt.Errorf("send: no process table attached to VM")
	}

	// Pop value and channel ID from stack
	val, err := vm.Pop()
	if err != nil {
		return err
	}

	chIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	chIDInt, ok := chIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("send: channel ID must be an integer")
	}
	chID := ChannelID(chIDInt.Val)

	if vm.processTable.channelTable == nil {
		return fmt.Errorf("send: no channel table")
	}

	ch, exists := vm.processTable.channelTable.Get(chID)
	if !exists {
		return fmt.Errorf("send: channel %d not found", chID)
	}

	blocked, wokenReceiver, err := ch.Send(val, vm.pid)
	if err != nil {
		return err
	}

	if blocked {
		// Block the sending process
		if proc, ok := vm.processTable.Get(vm.pid); ok {
			proc.State = ProcessBlocked
		}
		return nil
	}

	// Wake the blocked receiver if one was waiting
	if wokenReceiver != 0 {
		if proc, ok := vm.processTable.Get(wokenReceiver); ok {
			proc.State = ProcessReady
		}
	}

	return nil
}

// execRecv receives a value from a channel.
// Stack layout: channel_id (top) -> pops it, pushes received value.
func (vm *VM) execRecv() error {
	if vm.processTable == nil {
		return fmt.Errorf("recv: no process table attached to VM")
	}

	chIDVal, err := vm.Pop()
	if err != nil {
		return err
	}

	chIDInt, ok := chIDVal.(IntValue)
	if !ok {
		return fmt.Errorf("recv: channel ID must be an integer")
	}
	chID := ChannelID(chIDInt.Val)

	if vm.processTable.channelTable == nil {
		return fmt.Errorf("recv: no channel table")
	}

	ch, exists := vm.processTable.channelTable.Get(chID)
	if !exists {
		return fmt.Errorf("recv: channel %d not found", chID)
	}

	val, blocked, wokenSender, err := ch.Recv(vm.pid)
	if err != nil {
		return err
	}

	if blocked {
		// Block the receiving process
		if proc, ok := vm.processTable.Get(vm.pid); ok {
			proc.State = ProcessBlocked
		}
		return nil
	}

	// Wake the blocked sender if one was waiting
	if wokenSender != 0 {
		if proc, ok := vm.processTable.Get(wokenSender); ok {
			proc.State = ProcessReady
		}
	}

	vm.Push(val)
	return nil
}
