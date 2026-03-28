// GlyphLang GPU VM - WebGPU Compute Shader
// Executes GLYP bytecode on GPU compute units.
// Each workgroup thread runs an independent VM instance.

// Opcodes must match pkg/vm/vm.go
const OP_PUSH: u32         = 0x01u;
const OP_POP: u32          = 0x02u;
const OP_ADD: u32          = 0x10u;
const OP_SUB: u32          = 0x11u;
const OP_MUL: u32          = 0x12u;
const OP_DIV: u32          = 0x13u;
const OP_MOD: u32          = 0x14u;
const OP_EQ: u32           = 0x20u;
const OP_NE: u32           = 0x21u;
const OP_LT: u32           = 0x22u;
const OP_GT: u32           = 0x23u;
const OP_GE: u32           = 0x24u;
const OP_LE: u32           = 0x25u;
const OP_AND: u32          = 0x26u;
const OP_OR: u32           = 0x27u;
const OP_NOT: u32          = 0x28u;
const OP_NEG: u32          = 0x29u;
const OP_LOAD_VAR: u32     = 0x40u;
const OP_STORE_VAR: u32    = 0x41u;
const OP_JUMP: u32         = 0x50u;
const OP_JUMP_IF_FALSE: u32 = 0x51u;
const OP_JUMP_IF_TRUE: u32 = 0x52u;
const OP_GET_INDEX: u32    = 0x56u;
const OP_SET_INDEX: u32    = 0x57u;
const OP_RETURN: u32       = 0x61u;
const OP_CALL: u32         = 0x62u;
const OP_BUILD_OBJECT: u32 = 0x70u;
const OP_GET_FIELD: u32    = 0x71u;
const OP_SET_FIELD: u32    = 0x72u;
const OP_BUILD_ARRAY: u32  = 0x80u;
const OP_HALT: u32         = 0xFFu;

// Constants for VM dimensions
const MAX_STACK: u32 = 256u;
const MAX_VARS: u32 = 64u;
const MAX_STEPS: u32 = 100000u;

// Constant types (match bootstrap compiler encoding)
const CONST_NULL: u32 = 0x00u;
const CONST_INT: u32  = 0x01u;
const CONST_FLOAT: u32 = 0x02u;
const CONST_BOOL: u32 = 0x03u;
const CONST_STR: u32  = 0x04u;

// Value type tags for GPU values
const TAG_NULL: u32  = 0u;
const TAG_INT: u32   = 1u;
const TAG_FLOAT: u32 = 2u;
const TAG_BOOL: u32  = 3u;

// Packed value: tag in high 4 bits, data in low 28 bits (for integers)
// For floats, we use bitcast from f32 in a second word
struct GpuValue {
    tag: u32,
    data: i32,   // integer value or bitcast float
}

// Per-thread VM state
struct VMState {
    pc: u32,           // program counter
    sp: u32,           // stack pointer
    halted: u32,       // 0 = running, 1 = halted
    error: u32,        // 0 = ok, error code otherwise
    steps: u32,        // execution step counter
    result_tag: u32,   // result value tag
    result_data: i32,  // result value data
    _pad: u32,
}

// Dispatch configuration
struct Config {
    bytecode_len: u32,    // length of bytecode (after GLYP header)
    num_constants: u32,   // number of constants
    constants_offset: u32, // offset into bytecode where constants start
    code_offset: u32,     // offset into bytecode where instructions start
    num_vms: u32,         // number of VM instances to run
    _pad1: u32,
    _pad2: u32,
    _pad3: u32,
}

// Bindings
@group(0) @binding(0) var<storage, read> config: Config;
@group(0) @binding(1) var<storage, read> bytecode: array<u32>;
@group(0) @binding(2) var<storage, read_write> vm_states: array<VMState>;
@group(0) @binding(3) var<storage, read_write> stacks: array<GpuValue>;
@group(0) @binding(4) var<storage, read_write> vars: array<GpuValue>;

// Helper: get stack base for a VM instance
fn stack_base(vm_id: u32) -> u32 {
    return vm_id * MAX_STACK;
}

// Helper: get vars base for a VM instance
fn vars_base(vm_id: u32) -> u32 {
    return vm_id * MAX_VARS;
}

// Helper: read a u32 from bytecode at byte offset (bytecode is packed as u32 words)
fn read_byte(offset: u32) -> u32 {
    let word_idx = offset / 4u;
    let byte_idx = offset % 4u;
    let word = bytecode[word_idx];
    return (word >> (byte_idx * 8u)) & 0xFFu;
}

// Helper: read 4 bytes as little-endian u32 from bytecode
fn read_u32(offset: u32) -> u32 {
    let b0 = read_byte(offset);
    let b1 = read_byte(offset + 1u);
    let b2 = read_byte(offset + 2u);
    let b3 = read_byte(offset + 3u);
    return b0 | (b1 << 8u) | (b2 << 16u) | (b3 << 24u);
}

// Helper: push value onto VM stack
fn push(vm_id: u32, tag: u32, data: i32) {
    let sp = vm_states[vm_id].sp;
    if sp >= MAX_STACK {
        vm_states[vm_id].error = 1u; // stack overflow
        vm_states[vm_id].halted = 1u;
        return;
    }
    let idx = stack_base(vm_id) + sp;
    stacks[idx].tag = tag;
    stacks[idx].data = data;
    vm_states[vm_id].sp = sp + 1u;
}

// Helper: pop value from VM stack
fn pop(vm_id: u32) -> GpuValue {
    let sp = vm_states[vm_id].sp;
    if sp == 0u {
        vm_states[vm_id].error = 2u; // stack underflow
        vm_states[vm_id].halted = 1u;
        return GpuValue(TAG_NULL, 0);
    }
    let new_sp = sp - 1u;
    vm_states[vm_id].sp = new_sp;
    let idx = stack_base(vm_id) + new_sp;
    return stacks[idx];
}

// Helper: peek at top of stack without popping
fn peek(vm_id: u32) -> GpuValue {
    let sp = vm_states[vm_id].sp;
    if sp == 0u {
        return GpuValue(TAG_NULL, 0);
    }
    let idx = stack_base(vm_id) + sp - 1u;
    return stacks[idx];
}

// Load a constant from the constant pool
fn load_constant(const_idx: u32) -> GpuValue {
    // Constants are stored sequentially after the header in bytecode
    // Each constant: 1 byte type + variable length data
    var offset = config.constants_offset;
    var i = 0u;
    for (; i < const_idx && offset < config.code_offset; i = i + 1u) {
        let ctype = read_byte(offset);
        offset = offset + 1u;
        if ctype == CONST_NULL {
            // no data
        } else if ctype == CONST_INT {
            offset = offset + 8u; // 8 bytes for int64
        } else if ctype == CONST_FLOAT {
            offset = offset + 8u; // 8 bytes for float64
        } else if ctype == CONST_BOOL {
            offset = offset + 1u; // 1 byte for bool
        } else if ctype == CONST_STR {
            let str_len = read_u32(offset);
            offset = offset + 4u + str_len;
        }
    }

    if i != const_idx || offset >= config.code_offset {
        return GpuValue(TAG_NULL, 0);
    }

    let ctype = read_byte(offset);
    offset = offset + 1u;

    if ctype == CONST_NULL {
        return GpuValue(TAG_NULL, 0);
    } else if ctype == CONST_INT {
        // Read 8-byte int, truncate to i32 for GPU
        let hi = read_u32(offset);
        let lo = read_u32(offset + 4u);
        return GpuValue(TAG_INT, i32(lo));
    } else if ctype == CONST_FLOAT {
        let hi = read_u32(offset);
        let lo = read_u32(offset + 4u);
        // Approximate: use low 32 bits as float bits
        return GpuValue(TAG_FLOAT, i32(lo));
    } else if ctype == CONST_BOOL {
        let val = read_byte(offset);
        return GpuValue(TAG_BOOL, i32(val));
    }

    // String constants return their index as a reference
    return GpuValue(TAG_INT, i32(const_idx));
}

// Execute one instruction for a VM instance
fn exec_step(vm_id: u32) {
    if vm_states[vm_id].halted != 0u {
        return;
    }

    let pc = vm_states[vm_id].pc;
    let base = config.code_offset;

    if pc >= config.bytecode_len {
        vm_states[vm_id].halted = 1u;
        return;
    }

    let op = read_byte(base + pc);
    var next_pc = pc + 1u;

    switch op {
        case OP_PUSH: {
            let const_idx = read_u32(base + pc + 1u);
            next_pc = pc + 5u;
            let val = load_constant(const_idx);
            push(vm_id, val.tag, val.data);
        }

        case OP_POP: {
            _ = pop(vm_id);
        }

        case OP_ADD: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            if a.tag == TAG_INT && b.tag == TAG_INT {
                push(vm_id, TAG_INT, a.data + b.data);
            } else if a.tag == TAG_FLOAT || b.tag == TAG_FLOAT {
                let af = bitcast<f32>(a.data);
                let bf = bitcast<f32>(b.data);
                push(vm_id, TAG_FLOAT, bitcast<i32>(af + bf));
            } else {
                push(vm_id, TAG_INT, a.data + b.data);
            }
        }

        case OP_SUB: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            push(vm_id, TAG_INT, a.data - b.data);
        }

        case OP_MUL: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            push(vm_id, TAG_INT, a.data * b.data);
        }

        case OP_DIV: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            if b.data == 0 {
                vm_states[vm_id].error = 3u; // division by zero
                vm_states[vm_id].halted = 1u;
            } else {
                push(vm_id, TAG_INT, a.data / b.data);
            }
        }

        case OP_MOD: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            if b.data == 0 {
                vm_states[vm_id].error = 3u;
                vm_states[vm_id].halted = 1u;
            } else {
                push(vm_id, TAG_INT, a.data % b.data);
            }
        }

        case OP_EQ: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data == b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_NE: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data != b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_LT: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data < b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_GT: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data > b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_GE: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data >= b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_LE: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data <= b.data);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_AND: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data != 0 && b.data != 0);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_OR: {
            let b = pop(vm_id);
            let a = pop(vm_id);
            let result = select(0, 1, a.data != 0 || b.data != 0);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_NOT: {
            let a = pop(vm_id);
            let result = select(0, 1, a.data == 0);
            push(vm_id, TAG_BOOL, i32(result));
        }

        case OP_NEG: {
            let a = pop(vm_id);
            push(vm_id, a.tag, -a.data);
        }

        case OP_STORE_VAR: {
            let var_idx = read_u32(base + pc + 1u);
            next_pc = pc + 5u;
            let val = pop(vm_id);
            let vidx = vars_base(vm_id) + (var_idx % MAX_VARS);
            vars[vidx] = val;
        }

        case OP_LOAD_VAR: {
            let var_idx = read_u32(base + pc + 1u);
            next_pc = pc + 5u;
            let vidx = vars_base(vm_id) + (var_idx % MAX_VARS);
            let val = vars[vidx];
            push(vm_id, val.tag, val.data);
        }

        case OP_JUMP: {
            let target = read_u32(base + pc + 1u);
            next_pc = target;
        }

        case OP_JUMP_IF_FALSE: {
            let target = read_u32(base + pc + 1u);
            next_pc = pc + 5u;
            let cond = pop(vm_id);
            if cond.data == 0 {
                next_pc = target;
            }
        }

        case OP_JUMP_IF_TRUE: {
            let target = read_u32(base + pc + 1u);
            next_pc = pc + 5u;
            let cond = pop(vm_id);
            if cond.data != 0 {
                next_pc = target;
            }
        }

        case OP_RETURN: {
            let val = pop(vm_id);
            vm_states[vm_id].result_tag = val.tag;
            vm_states[vm_id].result_data = val.data;
            vm_states[vm_id].halted = 1u;
        }

        case OP_HALT: {
            // Capture top of stack as result if available
            if vm_states[vm_id].sp > 0u {
                let val = peek(vm_id);
                vm_states[vm_id].result_tag = val.tag;
                vm_states[vm_id].result_data = val.data;
            }
            vm_states[vm_id].halted = 1u;
        }

        default: {
            // Unknown opcode — skip it (forward compatibility)
            // For opcodes with operands we don't handle, this may desync,
            // but it's better than halting on extension opcodes
        }
    }

    vm_states[vm_id].pc = next_pc;
    vm_states[vm_id].steps = vm_states[vm_id].steps + 1u;

    // Safety: halt if too many steps
    if vm_states[vm_id].steps >= MAX_STEPS {
        vm_states[vm_id].error = 4u; // max steps exceeded
        vm_states[vm_id].halted = 1u;
    }
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let vm_id = gid.x;
    if vm_id >= config.num_vms {
        return;
    }

    // Execute until halted
    for (var i = 0u; i < MAX_STEPS; i = i + 1u) {
        if vm_states[vm_id].halted != 0u {
            break;
        }
        exec_step(vm_id);
    }
}
