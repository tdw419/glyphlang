// Package runtime provides a minimal GlyphLang VM runtime.
// This is a stripped-down implementation optimized for self-hosted execution.
// Only includes opcodes and builtins used by the self-hosted compiler.
package runtime

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// Opcode definitions - only those needed for self-hosting
const (
	OP_PUSH = iota + 1
	OP_POP
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 3-15 reserved
	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD // 16-20
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 21-31 reserved
	OP_EQ
	OP_NE
	OP_LT
	OP_GT
	OP_GE
	OP_LE
	OP_AND
	OP_OR
	OP_NOT
	OP_NEG // 32-41
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 42-63 reserved
	OP_LOAD_VAR
	OP_STORE_VAR // 64-65
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 66-79 reserved
	OP_JUMP
	OP_JUMP_IF_FALSE
	_
	_
	_
	OP_GET_INDEX
	OP_SET_INDEX // 80-87
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 88-96 reserved
	OP_RETURN
	OP_CALL // 97-98
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 99-111 reserved
	OP_BUILD_OBJECT
	OP_GET_FIELD
	OP_SET_FIELD
	OP_DEF_FUNC // 112-115
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_ // 116-127 reserved
	OP_BUILD_ARRAY // 128
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	OP_HALT = 255
)

// Value represents a runtime value
type Value struct {
	Type  ValueType
	Int   int64
	Str   string
	Array []Value
	Map   map[string]Value
	Func  *Function
}

type ValueType int

const (
	TypeNone ValueType = iota
	TypeInt
	TypeString
	TypeArray
	TypeMap
	TypeFunction
)

// Function represents a defined function
type Function struct {
	Name       string
	ParamCount int
	StartIP    int
	Closure    map[string]Value
}

// VM is a minimal virtual machine
type VM struct {
	bytecode   []byte
	ip         int
	stack      []Value
	sp         int
	globals    map[string]Value
	constants  []Value
	functions  map[string]*Function
	callStack  []callFrame
	callSP     int
	output     strings.Builder
}

type callFrame struct {
	returnIP int
	locals   map[string]Value
	baseSP   int
}

// NewVM creates a new minimal VM
func NewVM() *VM {
	return &VM{
		stack:     make([]Value, 1024),
		globals:   make(map[string]Value),
		functions: make(map[string]*Function),
		callStack: make([]callFrame, 256),
	}
}

// Load loads bytecode into the VM
func (vm *VM) Load(bytecode []byte) error {
	// Verify magic header
	if len(bytecode) < 12 {
		return fmt.Errorf("bytecode too short")
	}
	if string(bytecode[0:4]) != "GLYP" {
		return fmt.Errorf("invalid magic header")
	}

	vm.bytecode = bytecode
	vm.ip = 12 // Skip header

	// Load constants
	constCount := int(binary.LittleEndian.Uint32(bytecode[8:12]))
	vm.constants = make([]Value, constCount)

	offset := 12
	for i := 0; i < constCount; i++ {
		if offset >= len(bytecode) {
			return fmt.Errorf("truncated constants")
		}
		// Read length prefix (4 bytes)
		if offset+4 > len(bytecode) {
			return fmt.Errorf("truncated constant length")
		}
		length := int(binary.LittleEndian.Uint32(bytecode[offset : offset+4]))
		offset += 4

		// Read value
		if offset+length > len(bytecode) {
			return fmt.Errorf("truncated constant value")
		}
		data := bytecode[offset : offset+length]
		offset += length

		// Parse value based on first byte type marker
		if len(data) > 0 {
			switch data[0] {
			case 'I': // Integer
				if len(data) >= 9 {
					vm.constants[i] = Value{Type: TypeInt, Int: int64(binary.LittleEndian.Uint64(data[1:9]))}
				}
			case 'S': // String
				vm.constants[i] = Value{Type: TypeString, Str: string(data[1:])}
			}
		}
	}

	// Skip to instructions (after constant count u32)
	instCount := int(binary.LittleEndian.Uint32(bytecode[offset : offset+4]))
	_ = instCount // Used for bounds checking
	vm.ip = offset + 4

	return nil
}

// Run executes the loaded bytecode
func (vm *VM) Run() (string, error) {
	for vm.ip < len(vm.bytecode) {
		op := vm.bytecode[vm.ip]
		vm.ip++

		switch op {
		case OP_PUSH:
			idx := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			vm.push(vm.constants[idx])

		case OP_POP:
			vm.pop()

		case OP_ADD:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: a.Int + b.Int})

		case OP_SUB:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: a.Int - b.Int})

		case OP_MUL:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: a.Int * b.Int})

		case OP_DIV:
			b, a := vm.pop(), vm.pop()
			if b.Int == 0 {
				return "", fmt.Errorf("division by zero")
			}
			vm.push(Value{Type: TypeInt, Int: a.Int / b.Int})

		case OP_MOD:
			b, a := vm.pop(), vm.pop()
			if b.Int == 0 {
				return "", fmt.Errorf("modulo by zero")
			}
			vm.push(Value{Type: TypeInt, Int: a.Int % b.Int})

		case OP_EQ:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int == b.Int && a.Type == b.Type)})

		case OP_NE:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int != b.Int || a.Type != b.Type)})

		case OP_LT:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int < b.Int)})

		case OP_GT:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int > b.Int)})

		case OP_GE:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int >= b.Int)})

		case OP_LE:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int <= b.Int)})

		case OP_AND:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int != 0 && b.Int != 0)})

		case OP_OR:
			b, a := vm.pop(), vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int != 0 || b.Int != 0)})

		case OP_NOT:
			a := vm.pop()
			vm.push(Value{Type: TypeInt, Int: boolToInt(a.Int == 0)})

		case OP_NEG:
			a := vm.pop()
			vm.push(Value{Type: TypeInt, Int: -a.Int})

		case OP_LOAD_VAR:
			idx := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			name := vm.constants[idx].Str
			if vm.callSP > 0 {
				if v, ok := vm.callStack[vm.callSP-1].locals[name]; ok {
					vm.push(v)
					break
				}
			}
			vm.push(vm.globals[name])

		case OP_STORE_VAR:
			idx := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			name := vm.constants[idx].Str
			val := vm.pop()
			if vm.callSP > 0 {
				vm.callStack[vm.callSP-1].locals[name] = val
			} else {
				vm.globals[name] = val
			}

		case OP_JUMP:
			target := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip = target

		case OP_JUMP_IF_FALSE:
			target := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			cond := vm.pop()
			if cond.Int == 0 {
				vm.ip = target
			}

		case OP_GET_INDEX:
			idx := vm.pop()
			arr := vm.pop()
			if arr.Type == TypeArray {
				if int(idx.Int) < len(arr.Array) {
					vm.push(arr.Array[idx.Int])
				}
			}

		case OP_SET_INDEX:
			val := vm.pop()
			idx := vm.pop()
			arr := vm.pop()
			if arr.Type == TypeArray && int(idx.Int) < len(arr.Array) {
				arr.Array[idx.Int] = val
				vm.push(arr)
			}

		case OP_RETURN:
			if vm.callSP > 0 {
				frame := vm.callStack[vm.callSP-1]
				vm.callSP--
				vm.ip = frame.returnIP
				// Trim stack to base
				vm.sp = frame.baseSP
			}

		case OP_CALL:
			idx := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			name := vm.constants[idx].Str

			// Check for builtin
			if fn, ok := builtins[name]; ok {
				fn(vm)
				break
			}

			// User function
			if fn, ok := vm.functions[name]; ok {
				frame := &vm.callStack[vm.callSP]
				frame.returnIP = vm.ip
				frame.locals = make(map[string]Value)
				frame.baseSP = vm.sp - fn.ParamCount
				vm.callSP++
				vm.ip = fn.StartIP
			}

		case OP_BUILD_OBJECT:
			count := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			obj := make(map[string]Value)
			for i := 0; i < count; i++ {
				val := vm.pop()
				key := vm.pop()
				obj[key.Str] = val
			}
			vm.push(Value{Type: TypeMap, Map: obj})

		case OP_GET_FIELD:
			field := vm.pop()
			obj := vm.pop()
			if obj.Type == TypeMap {
				vm.push(obj.Map[field.Str])
			}

		case OP_SET_FIELD:
			val := vm.pop()
			field := vm.pop()
			obj := vm.pop()
			if obj.Type == TypeMap {
				obj.Map[field.Str] = val
				vm.push(obj)
			}

		case OP_DEF_FUNC:
			nameIdx := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			paramCount := int(vm.bytecode[vm.ip+4])
			vm.ip += 5
			startIP := vm.ip

			name := vm.constants[nameIdx].Str
			vm.functions[name] = &Function{
				Name:       name,
				ParamCount: paramCount,
				StartIP:    startIP,
			}

		case OP_BUILD_ARRAY:
			count := int(binary.LittleEndian.Uint32(vm.bytecode[vm.ip : vm.ip+4]))
			vm.ip += 4
			arr := make([]Value, count)
			for i := count - 1; i >= 0; i-- {
				arr[i] = vm.pop()
			}
			vm.push(Value{Type: TypeArray, Array: arr})

		case OP_HALT:
			return vm.output.String(), nil

		default:
			return "", fmt.Errorf("unknown opcode: %d at ip=%d", op, vm.ip-1)
		}
	}

	return vm.output.String(), nil
}

func (vm *VM) push(v Value) {
	if vm.sp >= len(vm.stack) {
		vm.stack = append(vm.stack, v)
	} else {
		vm.stack[vm.sp] = v
	}
	vm.sp++
}

func (vm *VM) pop() Value {
	if vm.sp <= 0 {
		return Value{}
	}
	vm.sp--
	return vm.stack[vm.sp]
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// Builtin functions
var builtins = map[string]func(*VM){
	"length": func(vm *VM) {
		v := vm.pop()
		var l int64
		switch v.Type {
		case TypeArray:
			l = int64(len(v.Array))
		case TypeString:
			l = int64(len(v.Str))
		}
		vm.push(Value{Type: TypeInt, Int: l})
	},
	"toString": func(vm *VM) {
		v := vm.pop()
		var s string
		switch v.Type {
		case TypeInt:
			s = strconv.FormatInt(v.Int, 10)
		case TypeString:
			s = v.Str
		case TypeArray:
			s = "[...]"
		case TypeMap:
			s = "{...}"
		}
		vm.push(Value{Type: TypeString, Str: s})
	},
	"toInt": func(vm *VM) {
		v := vm.pop()
		var n int64
		switch v.Type {
		case TypeInt:
			n = v.Int
		case TypeString:
			n, _ = strconv.ParseInt(v.Str, 10, 64)
		}
		vm.push(Value{Type: TypeInt, Int: n})
	},
	"print": func(vm *VM) {
		v := vm.pop()
		vm.output.WriteString(v.Str)
		vm.output.WriteString("\n")
	},
	"readFile": func(vm *VM) {
		v := vm.pop()
		data, err := ioutil.ReadFile(v.Str)
		if err != nil {
			vm.push(Value{Type: TypeString, Str: ""})
			return
		}
		vm.push(Value{Type: TypeString, Str: string(data)})
	},
	"writeFile": func(vm *VM) {
		data := vm.pop()
		path := vm.pop()
		err := ioutil.WriteFile(path.Str, []byte(data.Str), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "writeFile error: %v\n", err)
		}
		vm.push(Value{Type: TypeInt, Int: boolToInt(err == nil)})
	},
	"typeOf": func(vm *VM) {
		v := vm.pop()
		var t string
		switch v.Type {
		case TypeNone:
			t = "none"
		case TypeInt:
			t = "int"
		case TypeString:
			t = "string"
		case TypeArray:
			t = "array"
		case TypeMap:
			t = "map"
		case TypeFunction:
			t = "function"
		}
		vm.push(Value{Type: TypeString, Str: t})
	},
	"intToBytes4": func(vm *VM) {
		v := vm.pop()
		b := make([]Value, 4)
		n := uint32(v.Int)
		b[0] = Value{Type: TypeInt, Int: int64(n & 0xFF)}
		b[1] = Value{Type: TypeInt, Int: int64((n >> 8) & 0xFF)}
		b[2] = Value{Type: TypeInt, Int: int64((n >> 16) & 0xFF)}
		b[3] = Value{Type: TypeInt, Int: int64((n >> 24) & 0xFF)}
		vm.push(Value{Type: TypeArray, Array: b})
	},
}
