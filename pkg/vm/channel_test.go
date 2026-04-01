package vm

import (
	"encoding/binary"
	"testing"
)

// --- Channel Table Tests (Step 1.1) ---

func TestChannelTableCreateChannel(t *testing.T) {
	ct := NewChannelTable()

	ch := ct.CreateChannel(8)
	if ch == nil {
		t.Fatal("CreateChannel returned nil")
	}
	if ch.ID != 1 {
		t.Errorf("expected first channel ID to be 1, got %d", ch.ID)
	}
	if ch.Capacity != 8 {
		t.Errorf("expected capacity 8, got %d", ch.Capacity)
	}

	ch2 := ct.CreateChannel(16)
	if ch2.ID != 2 {
		t.Errorf("expected second channel ID to be 2, got %d", ch2.ID)
	}
}

func TestChannelTableGet(t *testing.T) {
	ct := NewChannelTable()
	ct.CreateChannel(4)

	ch, ok := ct.Get(1)
	if !ok {
		t.Fatal("expected to find channel 1")
	}
	if ch.Capacity != 4 {
		t.Errorf("expected capacity 4, got %d", ch.Capacity)
	}

	_, ok = ct.Get(999)
	if ok {
		t.Error("expected Get(999) to return false")
	}
}

func TestChannelTableRemove(t *testing.T) {
	ct := NewChannelTable()
	ct.CreateChannel(4)

	ct.Remove(1)
	_, ok := ct.Get(1)
	if ok {
		t.Error("expected channel to be removed")
	}
}

// --- Circular Buffer Tests ---

func TestChannelBufferedSendRecv(t *testing.T) {
	ch := newChannel(1, 4)

	// Send values
	blocked, woken, err := ch.Send(IntValue{Val: 42}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if blocked {
		t.Error("expected send not to block (buffer has space)")
	}
	if woken != 0 {
		t.Error("expected no woken receiver")
	}

	blocked, woken, err = ch.Send(IntValue{Val: 99}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if blocked {
		t.Error("expected send not to block")
	}

	// Recv values
	val, blocked, woken, err := ch.Recv(2)
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if blocked {
		t.Error("expected recv not to block (buffer has data)")
	}
	if val.(IntValue).Val != 42 {
		t.Errorf("expected 42, got %d", val.(IntValue).Val)
	}
	if woken != 0 {
		t.Error("expected no woken sender (buffer still has space)")
	}

	val, blocked, woken, err = ch.Recv(2)
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if val.(IntValue).Val != 99 {
		t.Errorf("expected 99, got %d", val.(IntValue).Val)
	}
}

func TestChannelBufferedFullBlocksSender(t *testing.T) {
	ch := newChannel(1, 1)

	blocked, _, err := ch.Send(IntValue{Val: 1}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if blocked {
		t.Error("first send should not block")
	}

	// Buffer is now full
	blocked, _, err = ch.Send(IntValue{Val: 2}, 10)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if !blocked {
		t.Error("expected send to block when buffer is full")
	}

	// Sender should be in the wait queue
	if len(ch.senderWaitQueue) != 1 {
		t.Fatalf("expected 1 sender in wait queue, got %d", len(ch.senderWaitQueue))
	}
	if ch.senderWaitQueue[0] != 10 {
		t.Errorf("expected sender PID 10 in wait queue, got %d", ch.senderWaitQueue[0])
	}
}

func TestChannelBufferedEmptyBlocksReceiver(t *testing.T) {
	ch := newChannel(1, 4)

	_, blocked, _, err := ch.Recv(5)
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if !blocked {
		t.Error("expected recv to block when buffer is empty")
	}

	if len(ch.recvWaitQueue) != 1 {
		t.Fatalf("expected 1 receiver in wait queue, got %d", len(ch.recvWaitQueue))
	}
	if ch.recvWaitQueue[0] != 5 {
		t.Errorf("expected receiver PID 5 in wait queue, got %d", ch.recvWaitQueue[0])
	}
}

func TestChannelSendWakesBlockedReceiver(t *testing.T) {
	ch := newChannel(1, 4)

	// Receiver blocks first (empty buffer)
	_, blocked, _, _ := ch.Recv(5)
	if !blocked {
		t.Fatal("expected recv to block")
	}

	// Send should wake the blocked receiver
	blocked, woken, err := ch.Send(IntValue{Val: 77}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if blocked {
		t.Error("send should not block (buffer has space)")
	}
	if woken != 5 {
		t.Errorf("expected woken receiver PID 5, got %d", woken)
	}
}

func TestChannelRecvWakesBlockedSender(t *testing.T) {
	ch := newChannel(1, 1)

	// Fill the buffer
	ch.Send(IntValue{Val: 1}, 1)

	// Sender blocks (full)
	blocked, _, _ := ch.Send(IntValue{Val: 2}, 10)
	if !blocked {
		t.Fatal("expected send to block")
	}

	// Recv should wake the blocked sender
	val, blocked, woken, err := ch.Recv(5)
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if blocked {
		t.Error("recv should not block (buffer has data)")
	}
	if val.(IntValue).Val != 1 {
		t.Errorf("expected 1, got %d", val.(IntValue).Val)
	}
	if woken != 10 {
		t.Errorf("expected woken sender PID 10, got %d", woken)
	}
}

func TestChannelRendezvousSendBlocksNoReceiver(t *testing.T) {
	ch := newChannel(1, 0) // capacity 0 = rendezvous

	blocked, _, err := ch.Send(IntValue{Val: 42}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if !blocked {
		t.Error("expected send to block on rendezvous channel with no receiver")
	}
}

func TestChannelRendezvousRecvPicksUp(t *testing.T) {
	ch := newChannel(1, 0)

	// Sender blocks but leaves value
	ch.Send(IntValue{Val: 42}, 1)

	// Receiver should get the value and wake the sender
	val, blocked, woken, err := ch.Recv(2)
	if err != nil {
		t.Fatalf("Recv error: %v", err)
	}
	if blocked {
		t.Error("recv should not block (sender left value)")
	}
	if val.(IntValue).Val != 42 {
		t.Errorf("expected 42, got %d", val.(IntValue).Val)
	}
	if woken != 1 {
		t.Errorf("expected woken sender PID 1, got %d", woken)
	}
}

func TestChannelRendezvousRecvBlocksNoSender(t *testing.T) {
	ch := newChannel(1, 0)

	_, blocked, _, _ := ch.Recv(2)
	if !blocked {
		t.Error("expected recv to block on empty rendezvous channel")
	}
}

func TestChannelRendezvousSendWakesBlockedReceiver(t *testing.T) {
	ch := newChannel(1, 0)

	// Receiver blocks (no sender yet)
	ch.Recv(2)

	// Send should wake the blocked receiver
	blocked, woken, err := ch.Send(IntValue{Val: 55}, 1)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}
	if blocked {
		t.Error("send should not block when receiver is waiting")
	}
	if woken != 2 {
		t.Errorf("expected woken receiver PID 2, got %d", woken)
	}
}

// --- OpChannelCreate Tests (Step 1.1) ---

func TestOpChannelCreate(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	// Set up code with capacity operand at the start (executeInstruction is called
	// directly, so readOperand reads from vm.pc which we set to 0)
	capacity := uint32(8)
	opBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(opBytes, capacity)
	testVM.code = opBytes
	testVM.pc = 0

	err := testVM.executeInstruction(OpChannelCreate)
	if err != nil {
		t.Fatalf("executeInstruction(OpChannelCreate) error: %v", err)
	}

	// Channel ID should be on stack
	chID := mustPopInt(t, testVM)
	if chID != 1 {
		t.Errorf("expected channel ID 1, got %d", chID)
	}

	// Channel should exist in table
	ch, ok := pt.channelTable.Get(1)
	if !ok {
		t.Fatal("expected channel 1 in channel table")
	}
	if ch.Capacity != 8 {
		t.Errorf("expected capacity 8, got %d", ch.Capacity)
	}
}

func TestOpChannelCreateNoProcessTable(t *testing.T) {
	testVM := NewVM()
	testVM.code = []byte{byte(OpChannelCreate), 0, 0, 0, 0}
	testVM.pc = 0

	err := testVM.executeInstruction(OpChannelCreate)
	if err == nil {
		t.Fatal("expected error without process table")
	}
}

// --- OpSend Tests (Step 1.2) ---

func TestOpSendBufferedNoBlock(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	// Create a channel with capacity 4
	pt.channelTable = NewChannelTable()
	ch := pt.channelTable.CreateChannel(4)

	// Push channel_id and value onto stack, then exec OpSend
	testVM.Push(IntValue{Val: int64(ch.ID)})
	testVM.Push(IntValue{Val: 42})

	err := testVM.executeInstruction(OpSend)
	if err != nil {
		t.Fatalf("OpSend error: %v", err)
	}

	// Process should still be running (not blocked)
	proc, _ := pt.Get(1)
	if proc.State != ProcessRunning {
		t.Errorf("expected process to remain running, got %v", proc.State)
	}

	// Verify the value is in the channel buffer
	val, blocked, _, err := ch.Recv(0)
	if err != nil || blocked {
		t.Fatalf("Recv error or blocked: err=%v blocked=%v", err, blocked)
	}
	if val.(IntValue).Val != 42 {
		t.Errorf("expected 42 from channel, got %d", val.(IntValue).Val)
	}
}

func TestOpSendBufferFullBlocksSender(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	// Create a channel with capacity 1
	pt.channelTable = NewChannelTable()
	ch := pt.channelTable.CreateChannel(1)

	// Fill the buffer
	ch.Send(IntValue{Val: 1}, 0)

	// Try to send again — should block
	testVM.Push(IntValue{Val: int64(ch.ID)})
	testVM.Push(IntValue{Val: 2})

	err := testVM.executeInstruction(OpSend)
	if err != nil {
		t.Fatalf("OpSend error: %v", err)
	}

	proc, _ := pt.Get(1)
	if proc.State != ProcessBlocked {
		t.Errorf("expected process to be blocked, got %v", proc.State)
	}
}

func TestOpSendInvalidChannelID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)
	pt.channelTable = NewChannelTable()

	testVM.Push(IntValue{Val: 999}) // invalid channel ID
	testVM.Push(IntValue{Val: 42})

	err := testVM.executeInstruction(OpSend)
	if err == nil {
		t.Fatal("expected error for invalid channel ID")
	}
}

// --- OpRecv Tests (Step 1.3) ---

func TestOpRecvBufferedWithData(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.channelTable = NewChannelTable()
	ch := pt.channelTable.CreateChannel(4)

	// Put a value in the channel
	ch.Send(StringValue{Val: "hello"}, 0)

	// Recv it
	testVM.Push(IntValue{Val: int64(ch.ID)})

	err := testVM.executeInstruction(OpRecv)
	if err != nil {
		t.Fatalf("OpRecv error: %v", err)
	}

	proc, _ := pt.Get(1)
	if proc.State != ProcessRunning {
		t.Errorf("expected process to remain running, got %v", proc.State)
	}

	val, err := testVM.Pop()
	if err != nil {
		t.Fatalf("Pop error: %v", err)
	}
	sv, ok := val.(StringValue)
	if !ok || sv.Val != "hello" {
		t.Errorf("expected StringValue{hello}, got %v", val)
	}
}

func TestOpRecvBufferEmptyBlocksReceiver(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)

	pt.channelTable = NewChannelTable()
	ch := pt.channelTable.CreateChannel(4)

	// Recv from empty channel — should block
	testVM.Push(IntValue{Val: int64(ch.ID)})

	err := testVM.executeInstruction(OpRecv)
	if err != nil {
		t.Fatalf("OpRecv error: %v", err)
	}

	proc, _ := pt.Get(1)
	if proc.State != ProcessBlocked {
		t.Errorf("expected process to be blocked, got %v", proc.State)
	}
}

func TestOpRecvInvalidChannelID(t *testing.T) {
	testVM := NewVM()
	pt := NewProcessTable()
	testVM.processTable = pt
	testVM.pid = 1

	parent := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: testVM}
	pt.Register(parent)
	pt.channelTable = NewChannelTable()

	testVM.Push(IntValue{Val: 999})

	err := testVM.executeInstruction(OpRecv)
	if err == nil {
		t.Fatal("expected error for invalid channel ID")
	}
}

// --- Integration: Create -> Send -> Recv ---

func TestChannelCreateSendRecvIntegration(t *testing.T) {
	senderVM := NewVM()
	pt := NewProcessTable()
	senderVM.processTable = pt
	senderVM.pid = 1

	sender := &Process{PID: 1, PPID: 0, State: ProcessRunning, VM: senderVM}
	pt.Register(sender)

	// Step 1: Create channel with capacity 4
	pt.channelTable = NewChannelTable()
	ch := pt.channelTable.CreateChannel(4)

	// Step 2: Send a value to the channel
	senderVM.Push(IntValue{Val: int64(ch.ID)})
	senderVM.Push(IntValue{Val: 100})
	senderVM.executeInstruction(OpSend)

	// Step 3: Create a receiver process and recv from the channel
	receiverVM := NewVM()
	receiverVM.processTable = pt
	receiverVM.pid = 2
	receiver := &Process{PID: 2, PPID: 1, State: ProcessRunning, VM: receiverVM}
	pt.Register(receiver)

	receiverVM.Push(IntValue{Val: int64(ch.ID)})
	receiverVM.executeInstruction(OpRecv)

	// Receiver should get 100
	val, err := receiverVM.Pop()
	if err != nil {
		t.Fatalf("Pop error: %v", err)
	}
	if val.(IntValue).Val != 100 {
		t.Errorf("expected 100, got %d", val.(IntValue).Val)
	}
}
