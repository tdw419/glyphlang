// Package ssa provides a Static Single Assignment intermediate representation
// for GlyphLang. It sits between the AST and bytecode emission, enabling
// typed optimization passes and separate lowering for CPU VM and GPU shader.
package ssa

import (
	"fmt"
	"math"
	"strings"
)

// Type represents the type of an SSA value.
type Type int

const (
	TypeVoid Type = iota
	TypeInt
	TypeFloat
	TypeBool
	TypeString
	TypeNull
	TypeArray
	TypeObject
	TypeFunction
	TypeAny // unresolved/polymorphic
)

func (t Type) String() string {
	switch t {
	case TypeVoid:
		return "void"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeBool:
		return "bool"
	case TypeString:
		return "string"
	case TypeNull:
		return "null"
	case TypeArray:
		return "array"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "function"
	case TypeAny:
		return "any"
	default:
		return fmt.Sprintf("type(%d)", t)
	}
}

// Op identifies the SSA operation kind.
type Op int

const (
	// Constants
	OpConst Op = iota // AuxInt or AuxStr depending on Type

	// Integer arithmetic
	OpAddInt
	OpSubInt
	OpMulInt
	OpDivInt
	OpModInt
	OpNegInt

	// Float arithmetic
	OpAddFloat
	OpSubFloat
	OpMulFloat
	OpDivFloat
	OpNegFloat

	// String operations
	OpAddStr // concatenation

	// Integer comparisons -> TypeBool
	OpEqInt
	OpNeInt
	OpLtInt
	OpLeInt
	OpGtInt
	OpGeInt

	// Float comparisons -> TypeBool
	OpEqFloat
	OpNeFloat
	OpLtFloat
	OpLeFloat
	OpGtFloat
	OpGeFloat

	// String/Bool comparisons
	OpEqStr
	OpNeStr
	OpEqBool
	OpNeBool

	// Logical -> TypeBool
	OpAnd
	OpOr
	OpNot

	// Variables
	OpLoadVar  // AuxStr = name
	OpStoreVar // AuxStr = name, Args[0] = value

	// Phi node (SSA join)
	OpPhi // Args = values from predecessor blocks

	// Control flow (block terminators)
	OpIf     // Args[0] = condition; Block.Succs = [true, false]
	OpJump   // Block.Succs = [target]
	OpReturn // Args[0] = return value
	OpHalt   // top-of-stack result

	// Complex (CPU-only)
	OpCall       // AuxStr = function name, Args = arguments
	OpBuildObj   // AuxStr = comma-sep field names, Args = values
	OpBuildArray // Args = elements
	OpGetField   // AuxStr = field, Args[0] = object
	OpSetField   // AuxStr = field, Args[0] = obj, Args[1] = value
	OpGetIndex   // Args[0] = array, Args[1] = index
	OpSetIndex   // Args[0] = array, Args[1] = idx, Args[2] = value

	// Iteration (CPU-only)
	OpGetIter
	OpIterNext
	OpIterHasNext

	// Async (CPU-only)
	OpTry

	// WebSocket (CPU-only)
	OpWsSend
	OpWsBroadcast
	OpWsClose

	// GPU-specific
	OpMitosis // S opcode: clone VM
	OpMutator // M opcode: self-modify
	OpTelemetry // telemetry(slot, val) - write to vm_stats
)

// opNames maps Op to string for printing.
var opNames = map[Op]string{
	OpConst: "Const", OpAddInt: "AddInt", OpSubInt: "SubInt", OpMulInt: "MulInt",
	OpDivInt: "DivInt", OpModInt: "ModInt", OpNegInt: "NegInt",
	OpAddFloat: "AddFloat", OpSubFloat: "SubFloat", OpMulFloat: "MulFloat",
	OpDivFloat: "DivFloat", OpNegFloat: "NegFloat", OpAddStr: "AddStr",
	OpEqInt: "EqInt", OpNeInt: "NeInt", OpLtInt: "LtInt", OpLeInt: "LeInt",
	OpGtInt: "GtInt", OpGeInt: "GeInt", OpEqFloat: "EqFloat", OpNeFloat: "NeFloat",
	OpLtFloat: "LtFloat", OpLeFloat: "LeFloat", OpGtFloat: "GtFloat", OpGeFloat: "GeFloat",
	OpEqStr: "EqStr", OpNeStr: "NeStr", OpEqBool: "EqBool", OpNeBool: "NeBool",
	OpAnd: "And", OpOr: "Or", OpNot: "Not",
	OpLoadVar: "LoadVar", OpStoreVar: "StoreVar", OpPhi: "Phi",
	OpIf: "If", OpJump: "Jump", OpReturn: "Return", OpHalt: "Halt",
	OpCall: "Call", OpBuildObj: "BuildObj", OpBuildArray: "BuildArray",
	OpGetField: "GetField", OpSetField: "SetField",
	OpGetIndex: "GetIndex", OpSetIndex: "SetIndex",
	OpGetIter: "GetIter", OpIterNext: "IterNext", OpIterHasNext: "IterHasNext",
	OpTry: "Try", OpWsSend: "WsSend", OpWsBroadcast: "WsBroadcast", OpWsClose: "WsClose",
	OpMitosis: "Mitosis", OpMutator: "Mutator",
	OpTelemetry: "Telemetry",
}

func (op Op) String() string {
	if s, ok := opNames[op]; ok {
		return s
	}
	return fmt.Sprintf("Op(%d)", op)
}

// IsSideEffect returns true if the op has side effects and must not be eliminated.
func (op Op) IsSideEffect() bool {
	switch op {
	case OpStoreVar, OpCall, OpReturn, OpHalt, OpIf, OpJump,
		OpSetField, OpSetIndex,
		OpWsSend, OpWsBroadcast, OpWsClose,
		OpMitosis, OpMutator, OpTelemetry:
		return true
	}
	return false
}

// IsTerminator returns true if the op is a block terminator.
func (op Op) IsTerminator() bool {
	return op == OpIf || op == OpJump || op == OpReturn || op == OpHalt
}

// Value is an SSA value — each assigned exactly once.
type Value struct {
	ID     int
	Op     Op
	Type   Type
	Args   []*Value
	Block  *Block
	AuxInt int64  // auxiliary integer (constants, indices)
	AuxStr string // auxiliary string (names, field keys)
}

// String returns a human-readable representation: "v3 = AddInt v1 v2"
func (v *Value) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "v%d = %s", v.ID, v.Op)
	if v.Op == OpConst {
		switch v.Type {
		case TypeInt:
			fmt.Fprintf(&b, " %d", v.AuxInt)
		case TypeFloat:
			fmt.Fprintf(&b, " %g", math.Float64frombits(uint64(v.AuxInt)))
		case TypeBool:
			if v.AuxInt != 0 {
				b.WriteString(" true")
			} else {
				b.WriteString(" false")
			}
		case TypeString:
			fmt.Fprintf(&b, " %q", v.AuxStr)
		case TypeNull:
			b.WriteString(" null")
		}
	} else {
		if v.AuxStr != "" {
			fmt.Fprintf(&b, " [%s]", v.AuxStr)
		}
		for _, a := range v.Args {
			fmt.Fprintf(&b, " v%d", a.ID)
		}
	}
	return b.String()
}

// Block is a basic block in the control flow graph.
type Block struct {
	ID     int
	Name   string   // optional label for debugging
	Values []*Value // instructions in order
	Preds  []*Block // predecessor blocks
	Succs  []*Block // successor blocks
	Func   *Func    // parent function
}

// String returns a label like "b2".
func (blk *Block) String() string {
	if blk.Name != "" {
		return fmt.Sprintf("b%d(%s)", blk.ID, blk.Name)
	}
	return fmt.Sprintf("b%d", blk.ID)
}

// Terminator returns the last value in the block (should be a terminator op).
func (blk *Block) Terminator() *Value {
	if len(blk.Values) == 0 {
		return nil
	}
	last := blk.Values[len(blk.Values)-1]
	if last.Op.IsTerminator() {
		return last
	}
	return nil
}

// Func is an SSA function (route, function, handler).
type Func struct {
	Name      string
	Params    []string // parameter names
	Entry     *Block
	Blocks    []*Block
	nextValID int
	nextBlkID int
	Source    SourceKind
}

// SourceKind identifies the AST origin.
type SourceKind int

const (
	SourceRoute SourceKind = iota
	SourceFunction
	SourceCommand
)

// NewFunc creates a new SSA function with an entry block.
func NewFunc(name string, source SourceKind) *Func {
	f := &Func{Name: name, Source: source}
	entry := f.NewBlock("entry")
	f.Entry = entry
	return f
}

// NewBlock creates a new basic block in this function.
func (f *Func) NewBlock(name string) *Block {
	blk := &Block{
		ID:   f.nextBlkID,
		Name: name,
		Func: f,
	}
	f.nextBlkID++
	f.Blocks = append(f.Blocks, blk)
	return blk
}

// NewValue creates a new SSA value in the given block.
func (f *Func) NewValue(blk *Block, op Op, typ Type, args ...*Value) *Value {
	v := &Value{
		ID:    f.nextValID,
		Op:    op,
		Type:  typ,
		Args:  args,
		Block: blk,
	}
	f.nextValID++
	blk.Values = append(blk.Values, v)
	return v
}

// Program is the top-level SSA container.
type Program struct {
	Funcs []*Func
}

// Print returns a textual representation of the entire program.
func (p *Program) Print() string {
	var b strings.Builder
	for _, f := range p.Funcs {
		fmt.Fprintf(&b, "func %s (%s):\n", f.Name, strings.Join(f.Params, ", "))
		for _, blk := range f.Blocks {
			fmt.Fprintf(&b, "  %s:", blk)
			if len(blk.Preds) > 0 {
				b.WriteString(" <- ")
				for i, p := range blk.Preds {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%s", p)
				}
			}
			b.WriteString("\n")
			for _, v := range blk.Values {
				fmt.Fprintf(&b, "    %s\n", v)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
