package ssa

import (
	"encoding/binary"
	"fmt"
	"math"
)

// VM opcodes — mirrors pkg/vm/vm.go constants.
const (
	vmOpPush    byte = 0x01
	vmOpPop     byte = 0x02
	vmOpAdd     byte = 0x10
	vmOpSub     byte = 0x11
	vmOpMul     byte = 0x12
	vmOpDiv     byte = 0x13
	vmOpMod     byte = 0x14
	vmOpEq      byte = 0x20
	vmOpNe      byte = 0x21
	vmOpLt      byte = 0x22
	vmOpGt      byte = 0x23
	vmOpGe      byte = 0x24
	vmOpLe      byte = 0x25
	vmOpAnd     byte = 0x26
	vmOpOr      byte = 0x27
	vmOpNot     byte = 0x28
	vmOpNeg     byte = 0x29
	vmOpLoadVar byte = 0x40
	vmOpStore   byte = 0x41
	vmOpJump    byte = 0x50
	vmOpJumpF   byte = 0x51
	vmOpGetIter byte = 0x53
	vmOpIterN   byte = 0x54
	vmOpIterH   byte = 0x55
	vmOpGetIdx  byte = 0x56
	vmOpSetIdx  byte = 0x57
	vmOpReturn  byte = 0x61
	vmOpCall    byte = 0x62
	vmOpBldObj  byte = 0x70
	vmOpGetFld  byte = 0x71
	vmOpSetFld  byte = 0x72
	vmOpDefFunc byte = 0x73
	vmOpBldArr  byte = 0x80
	vmOpHTTP    byte = 0x90
	vmOpWsSend  byte = 0xA0
	vmOpWsBcast byte = 0xA1
	vmOpWsClose byte = 0xA5
	vmOpTry     byte = 0xB2
	vmOpMitosis byte = 0xC0
	vmOpMutator byte = 0xC1
	vmOpTelem   byte = 0xC2
	vmOpHalt    byte = 0xFF
)

// Constant pool type tags.
const (
	tagConstNull   byte = 0x00
	tagConstInt    byte = 0x01
	tagConstFloat  byte = 0x02
	tagConstBool   byte = 0x03
	tagConstString byte = 0x04
)

// CPULowering lowers an SSA program to GLYP bytecode for the CPU VM.
type CPULowering struct {
	code     []byte
	consts   []byte // serialized constant pool
	constN   int    // number of constants
	constMap map[string]uint32 // dedup: key → index

	// Block offset mapping for jump patching
	blockOffset map[int]int // block ID → code offset
	patches     []jumpPatch // jumps to patch after layout
}

type jumpPatch struct {
	codeOffset int // position of the 4-byte operand in code
	blockID    int // target block ID
}

// NewCPULowering creates a new CPU lowering context.
func NewCPULowering() *CPULowering {
	return &CPULowering{
		constMap:    make(map[string]uint32),
		blockOffset: make(map[int]int),
	}
}

// LowerFunc lowers a single SSA function to bytecode.
func (l *CPULowering) LowerFunc(f *Func) ([]byte, error) {
	// Schedule blocks: entry first, then topological order
	order := l.scheduleBlocks(f)

	// First pass: emit code and record block offsets
	for _, blk := range order {
		l.blockOffset[blk.ID] = len(l.code)
		for _, v := range blk.Values {
			if err := l.lowerValue(v); err != nil {
				return nil, fmt.Errorf("lowering v%d in %s: %w", v.ID, blk, err)
			}
		}
	}

	// Second pass: patch jump targets
	headerSize := 4 + 4 + 4 + len(l.consts) + 4 // magic + version + constCount + consts + instrCount
	for _, p := range l.patches {
		target, ok := l.blockOffset[p.blockID]
		if !ok {
			return nil, fmt.Errorf("unknown jump target block b%d", p.blockID)
		}
		adjusted := uint32(target + headerSize)
		binary.LittleEndian.PutUint32(l.code[p.codeOffset:], adjusted)
	}

	return l.buildBytecode(), nil
}

// LowerProgram lowers all functions. Currently emits the first function
// (main route) as a single bytecode blob.
func (l *CPULowering) LowerProgram(prog *Program) ([]byte, error) {
	if len(prog.Funcs) == 0 {
		return nil, fmt.Errorf("empty program")
	}
	// Lower the first function as the entry point
	return l.LowerFunc(prog.Funcs[0])
}

func (l *CPULowering) scheduleBlocks(f *Func) []*Block {
	visited := make(map[int]bool)
	var order []*Block

	var visit func(blk *Block)
	visit = func(blk *Block) {
		if visited[blk.ID] {
			return
		}
		visited[blk.ID] = true
		order = append(order, blk)
		for _, s := range blk.Succs {
			visit(s)
		}
	}
	visit(f.Entry)
	return order
}

func (l *CPULowering) lowerValue(v *Value) error {
	switch v.Op {
	case OpConst:
		idx := l.addConst(v)
		l.emitOp(vmOpPush, idx)

	case OpAddInt, OpAddFloat, OpAddStr:
		l.emitArgs(v)
		l.emitBare(vmOpAdd)
	case OpSubInt, OpSubFloat:
		l.emitArgs(v)
		l.emitBare(vmOpSub)
	case OpMulInt, OpMulFloat:
		l.emitArgs(v)
		l.emitBare(vmOpMul)
	case OpDivInt, OpDivFloat:
		l.emitArgs(v)
		l.emitBare(vmOpDiv)
	case OpModInt:
		l.emitArgs(v)
		l.emitBare(vmOpMod)
	case OpNegInt, OpNegFloat:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpNeg)
	case OpNot:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpNot)

	case OpEqInt, OpEqFloat, OpEqStr, OpEqBool:
		l.emitArgs(v)
		l.emitBare(vmOpEq)
	case OpNeInt, OpNeFloat, OpNeStr, OpNeBool:
		l.emitArgs(v)
		l.emitBare(vmOpNe)
	case OpLtInt, OpLtFloat:
		l.emitArgs(v)
		l.emitBare(vmOpLt)
	case OpLeInt, OpLeFloat:
		l.emitArgs(v)
		l.emitBare(vmOpLe)
	case OpGtInt, OpGtFloat:
		l.emitArgs(v)
		l.emitBare(vmOpGt)
	case OpGeInt, OpGeFloat:
		l.emitArgs(v)
		l.emitBare(vmOpGe)

	case OpAnd:
		l.emitArgs(v)
		l.emitBare(vmOpAnd)
	case OpOr:
		l.emitArgs(v)
		l.emitBare(vmOpOr)

	case OpLoadVar:
		idx := l.addStringConst(v.AuxStr)
		l.emitOp(vmOpLoadVar, idx)
	case OpStoreVar:
		l.pushArg(v.Args[0])
		idx := l.addStringConst(v.AuxStr)
		l.emitOp(vmOpStore, idx)

	case OpCall:
		// Push args in order, then push function name, then call
		for _, a := range v.Args {
			l.pushArg(a)
		}
		fnIdx := l.addStringConst(v.AuxStr)
		l.emitOp(vmOpLoadVar, fnIdx)
		l.emitOp(vmOpCall, uint32(len(v.Args)))

	case OpBuildObj:
		// Args are values, AuxStr has comma-sep field names
		for _, a := range v.Args {
			l.pushArg(a)
		}
		l.emitOp(vmOpBldObj, uint32(len(v.Args)))

	case OpBuildArray:
		for _, a := range v.Args {
			l.pushArg(a)
		}
		l.emitOp(vmOpBldArr, uint32(len(v.Args)))

	case OpGetField:
		l.pushArg(v.Args[0])
		idx := l.addStringConst(v.AuxStr)
		l.emitOp(vmOpPush, idx) // push field name
		l.emitBare(vmOpGetFld)

	case OpSetField:
		l.pushArg(v.Args[0]) // object
		idx := l.addStringConst(v.AuxStr)
		l.emitOp(vmOpPush, idx) // field name
		l.pushArg(v.Args[1])    // value
		l.emitBare(vmOpSetFld)

	case OpGetIndex:
		l.pushArg(v.Args[0])
		l.pushArg(v.Args[1])
		l.emitBare(vmOpGetIdx)

	case OpSetIndex:
		l.pushArg(v.Args[0])
		l.pushArg(v.Args[1])
		l.pushArg(v.Args[2])
		l.emitBare(vmOpSetIdx)

	case OpGetIter:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpGetIter)
	case OpIterNext:
		l.pushArg(v.Args[0])
		l.emitOp(vmOpIterN, 0)
	case OpIterHasNext:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpIterH)

	case OpTry:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpTry)

	case OpWsSend:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpWsSend)
	case OpWsBroadcast:
		l.pushArg(v.Args[0])
		l.emitBare(vmOpWsBcast)
	case OpWsClose:
		l.emitBare(vmOpWsClose)

	case OpMitosis:
		l.emitBare(vmOpMitosis)
	case OpMutator:
		l.emitBare(vmOpMutator)

	case OpIf:
		l.pushArg(v.Args[0])
		// JumpIfFalse to the false successor (Succs[1])
		if len(v.Block.Succs) < 2 {
			return fmt.Errorf("OpIf block %s has < 2 successors", v.Block)
		}
		l.emitJump(vmOpJumpF, v.Block.Succs[1].ID)
		// Fall through to true successor (Succs[0])
		// If true successor is not the next block in layout, emit unconditional jump
		l.emitJump(vmOpJump, v.Block.Succs[0].ID)

	case OpJump:
		if len(v.Block.Succs) < 1 {
			return fmt.Errorf("OpJump block %s has no successors", v.Block)
		}
		l.emitJump(vmOpJump, v.Block.Succs[0].ID)

	case OpReturn:
		if len(v.Args) > 0 {
			l.pushArg(v.Args[0])
		}
		l.emitBare(vmOpReturn)

	case OpHalt:
		if len(v.Args) > 0 {
			l.pushArg(v.Args[0])
		}
		l.emitBare(vmOpHalt)

	case OpPhi:
		// Phi nodes are resolved during register allocation / block ordering.
		// For a stack-based VM, we rely on store/load of the variable.
		// Phi is a no-op here; the builder should have emitted stores.

	case OpTelemetry:
		l.pushArg(v.Args[0]) // slot
		l.pushArg(v.Args[1]) // val
		l.emitBare(vmOpTelem)

	default:
		return fmt.Errorf("unsupported SSA op: %s", v.Op)
	}
	return nil
}

// pushArg pushes a value onto the VM stack. If it's a constant, emit a Push.
// For non-constants, we assume values are already on the stack from prior emissions.
// This is a simplification for the stack-based VM — a real register allocator
// would track value locations.
func (l *CPULowering) pushArg(v *Value) {
	if v.Op == OpConst {
		idx := l.addConst(v)
		l.emitOp(vmOpPush, idx)
	}
	// Non-const values: their producing instruction already left the result on stack.
	// The stack-based lowering relies on instruction ordering matching use order.
}

// emitArgs pushes all args of a binary op.
func (l *CPULowering) emitArgs(v *Value) {
	for _, a := range v.Args {
		l.pushArg(a)
	}
}

func (l *CPULowering) emitBare(op byte) {
	l.code = append(l.code, op)
}

func (l *CPULowering) emitOp(op byte, operand uint32) {
	l.code = append(l.code, op)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], operand)
	l.code = append(l.code, buf[:]...)
}

func (l *CPULowering) emitJump(op byte, blockID int) {
	l.code = append(l.code, op)
	patchPos := len(l.code)
	l.code = append(l.code, 0, 0, 0, 0) // placeholder
	l.patches = append(l.patches, jumpPatch{codeOffset: patchPos, blockID: blockID})
}

// addConst adds an SSA constant value to the constant pool and returns its index.
func (l *CPULowering) addConst(v *Value) uint32 {
	var key string
	switch v.Type {
	case TypeInt:
		key = fmt.Sprintf("i%d", v.AuxInt)
	case TypeFloat:
		key = fmt.Sprintf("f%d", v.AuxInt)
	case TypeBool:
		key = fmt.Sprintf("b%d", v.AuxInt)
	case TypeString:
		key = fmt.Sprintf("s%s", v.AuxStr)
	case TypeNull:
		key = "null"
	default:
		key = fmt.Sprintf("?%d_%s", v.AuxInt, v.AuxStr)
	}

	if idx, ok := l.constMap[key]; ok {
		return idx
	}

	idx := uint32(l.constN)
	l.constMap[key] = idx
	l.constN++

	switch v.Type {
	case TypeNull:
		l.consts = append(l.consts, tagConstNull)
	case TypeInt:
		l.consts = append(l.consts, tagConstInt)
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(v.AuxInt))
		l.consts = append(l.consts, buf[:]...)
	case TypeFloat:
		l.consts = append(l.consts, tagConstFloat)
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(v.AuxInt))
		l.consts = append(l.consts, buf[:]...)
	case TypeBool:
		l.consts = append(l.consts, tagConstBool)
		if v.AuxInt != 0 {
			l.consts = append(l.consts, 1)
		} else {
			l.consts = append(l.consts, 0)
		}
	case TypeString:
		l.consts = append(l.consts, tagConstString)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(len(v.AuxStr)))
		l.consts = append(l.consts, buf[:]...)
		l.consts = append(l.consts, []byte(v.AuxStr)...)
	}

	return idx
}

// addStringConst adds a string constant and returns its pool index.
func (l *CPULowering) addStringConst(s string) uint32 {
	v := &Value{Op: OpConst, Type: TypeString, AuxStr: s}
	return l.addConst(v)
}

// buildBytecode assembles the final GLYP binary.
func (l *CPULowering) buildBytecode() []byte {
	// Header: "GLYP" + version(4) + constCount(4) + constants + instrCount(4) + instructions
	size := 4 + 4 + 4 + len(l.consts) + 4 + len(l.code)
	out := make([]byte, 0, size)

	// Magic
	out = append(out, 'G', 'L', 'Y', 'P')

	// Version
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], 1)
	out = append(out, buf[:]...)

	// Constant count
	binary.LittleEndian.PutUint32(buf[:], uint32(l.constN))
	out = append(out, buf[:]...)

	// Constants
	out = append(out, l.consts...)

	// Instruction count
	binary.LittleEndian.PutUint32(buf[:], uint32(len(l.code)))
	out = append(out, buf[:]...)

	// Instructions
	out = append(out, l.code...)

	return out
}

// floatConst is a helper to create a float constant value for the pool.
func floatConst(f float64) *Value {
	return &Value{
		Op:     OpConst,
		Type:   TypeFloat,
		AuxInt: int64(math.Float64bits(f)),
	}
}
