// GlyphLang GPU VM - WebGPU Compute Shader
// Executes GLYP bytecode on GPU compute units.

struct VMState {
    pc: u32, sp: u32, halted: u32, error: u32, steps: u32, result_tag: u32, result_data: i32, pad: u32,
}

struct Config {
    bytecode_len: u32, num_constants: u32, constants_offset: u32, code_offset: u32, num_vms: u32,
    _pad1: u32, _pad2: u32, _pad3: u32,
}

struct GpuValue {
    tag: u32, data: i32,
}

@group(0) @binding(0) var<storage, read> config: Config;
@group(0) @binding(1) var<storage, read_write> bytecode: array<u32>;
@group(0) @binding(2) var<storage, read_write> vm_states: array<VMState>;
@group(0) @binding(3) var<storage, read_write> stacks: array<GpuValue>;
@group(0) @binding(4) var<storage, read_write> vars: array<GpuValue>;
@group(0) @binding(5) var<storage, read_write> spawn_requests: array<atomic<u32>>;
@group(0) @binding(6) var vcc_texture: texture_storage_2d<rgba8unorm, write>;

const TAG_NULL: u32  = 0u;
const TAG_INT: u32   = 1u;
const TAG_FLOAT: u32 = 2u;
const TAG_BOOL: u32  = 3u;

const MAX_STACK: u32 = 256u;
const MAX_VARS: u32  = 64u;
const MAX_STEPS: u32 = 100000u;

const CONST_NULL: u32  = 0u;
const CONST_INT: u32   = 1u;
const CONST_FLOAT: u32 = 2u;
const CONST_BOOL: u32  = 3u;
const CONST_STRING: u32 = 4u;

// Named opcode constants (synchronized with pkg/vm/vm.go)
const OP_PUSH: u32        = 0x01u;
const OP_POP: u32         = 0x02u;
const OP_ADD: u32         = 0x10u;
const OP_SUB: u32         = 0x11u;
const OP_MUL: u32         = 0x12u;
const OP_DIV: u32         = 0x13u;
const OP_MOD: u32         = 0x14u;
const OP_EQ: u32          = 0x20u;
const OP_NE: u32          = 0x21u;
const OP_LT: u32          = 0x22u;
const OP_GT: u32          = 0x23u;
const OP_GE: u32          = 0x24u;
const OP_LE: u32          = 0x25u;
const OP_AND: u32         = 0x26u;
const OP_OR: u32          = 0x27u;
const OP_NOT: u32         = 0x28u;
const OP_NEG: u32         = 0x29u;
const OP_LOAD_VAR: u32    = 0x40u;
const OP_STORE_VAR: u32   = 0x41u;
const OP_JUMP: u32        = 0x50u;
const OP_JUMP_IF_FALSE: u32 = 0x51u;
const OP_RETURN: u32      = 0x61u;
const OP_MITOSIS: u32     = 0xC0u;
const OP_MUTATOR: u32     = 0xC1u;
const OP_TELEMETRY: u32   = 0xC2u;
const OP_SYSCALL: u32     = 0xDDu;
const OP_HALT: u32        = 0xFFu;

// Error codes
const ERR_STACK_OVERFLOW: u32  = 1u;
const ERR_STACK_UNDERFLOW: u32 = 2u;
const ERR_DIV_BY_ZERO: u32     = 3u;
const ERR_MUTATOR_OOB: u32     = 5u;

fn read_byte(offset: u32) -> u32 {
    let word = bytecode[offset / 4u];
    return (word >> ((offset % 4u) * 8u)) & 0xFFu;
}

fn read_u32(offset: u32) -> u32 {
    let b0 = read_byte(offset);
    let b1 = read_byte(offset + 1u);
    let b2 = read_byte(offset + 2u);
    let b3 = read_byte(offset + 3u);
    return b0 | (b1 << 8u) | (b2 << 16u) | (b3 << 24u);
}

fn push(vm_id: u32, tag: u32, data: i32) {
    let sp = vm_states[vm_id].sp;
    if (sp < MAX_STACK) {
        stacks[vm_id * MAX_STACK + sp] = GpuValue(tag, data);
        vm_states[vm_id].sp = sp + 1u;
    } else {
        vm_states[vm_id].error = 1u; vm_states[vm_id].halted = 1u;
    }
}

fn pop(vm_id: u32) -> GpuValue {
    let sp = vm_states[vm_id].sp;
    if (sp > 0u) {
        let new_sp = sp - 1u;
        vm_states[vm_id].sp = new_sp;
        return stacks[vm_id * MAX_STACK + new_sp];
    }
    vm_states[vm_id].error = 2u; vm_states[vm_id].halted = 1u;
    return GpuValue(TAG_NULL, 0);
}

fn load_constant(const_idx: u32) -> GpuValue {
    var offset = config.constants_offset;
    // Skip preceding constants with correct stride per type
    for (var i = 0u; i < const_idx; i = i + 1u) {
        let ctype = read_byte(offset);
        offset = offset + 1u;
        if (ctype == CONST_INT || ctype == CONST_FLOAT) {
            offset = offset + 8u;  // 8-byte payload
        } else if (ctype == CONST_BOOL) {
            offset = offset + 1u;  // 1-byte payload
        } else if (ctype == CONST_STRING) {
            let slen = read_u32(offset);
            offset = offset + 4u + slen;
        }
        // CONST_NULL (0): no payload bytes
    }
    let ctype = read_byte(offset);
    offset = offset + 1u;
    if (ctype == CONST_INT) {
        return GpuValue(TAG_INT, i32(read_u32(offset)));
    }
    if (ctype == CONST_FLOAT) {
        // Store as f32 bits in data field (lower 32 bits of f64 → f32)
        let lo = read_u32(offset);
        return GpuValue(TAG_FLOAT, i32(lo));
    }
    if (ctype == CONST_BOOL) {
        return GpuValue(TAG_BOOL, i32(read_byte(offset)));
    }
    // CONST_NULL or CONST_STRING → null on GPU
    return GpuValue(TAG_NULL, 0);
}

fn exec_step(vm_id: u32) {
    let pc = vm_states[vm_id].pc;
    if (pc >= config.code_offset + config.bytecode_len) { vm_states[vm_id].halted = 1u; return; }
    let op = read_byte(pc);
    var next_pc = pc + 1u;
    switch (op) {
        case 0x00u: { /* NOP */ }
        case 0x01u: { push(vm_id, load_constant(read_u32(pc + 1u)).tag, load_constant(read_u32(pc + 1u)).data); next_pc = pc + 5u; }
        case 0x02u: { pop(vm_id); }
        case 0x10u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_INT, a.data + b.data); }
        case 0x11u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_INT, a.data - b.data); }
        case 0x12u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_INT, a.data * b.data); }
        case 0x13u: {
            let b = pop(vm_id); let a = pop(vm_id);
            if (b.data == 0) { vm_states[vm_id].error = ERR_DIV_BY_ZERO; vm_states[vm_id].halted = 1u; }
            else { push(vm_id, TAG_INT, a.data / b.data); }
        }
        case 0x14u: {
            let b = pop(vm_id); let a = pop(vm_id);
            if (b.data == 0) { vm_states[vm_id].error = ERR_DIV_BY_ZERO; vm_states[vm_id].halted = 1u; }
            else { push(vm_id, TAG_INT, a.data % b.data); }
        }
        case 0x20u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.tag == b.tag && a.data == b.data)); }
        case 0x21u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.tag != b.tag || a.data != b.data)); }
        case 0x22u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data < b.data)); }
        case 0x23u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data > b.data)); }
        case 0x24u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data >= b.data)); }
        case 0x25u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data <= b.data)); }
        case 0x26u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data != 0 && b.data != 0)); }
        case 0x27u: { let b = pop(vm_id); let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data != 0 || b.data != 0)); }
        case 0x28u: { let a = pop(vm_id); push(vm_id, TAG_BOOL, select(0, 1, a.data == 0)); }
        case 0x29u: { let a = pop(vm_id); push(vm_id, a.tag, -a.data); }
        case 0x40u: { let vidx = read_u32(pc + 1u); let val = vars[vm_id * MAX_VARS + (vidx % MAX_VARS)]; push(vm_id, val.tag, val.data); next_pc = pc + 5u; }
        case 0x41u: { let vidx = read_u32(pc + 1u); let val = pop(vm_id); vars[vm_id * MAX_VARS + (vidx % MAX_VARS)] = val; next_pc = pc + 5u; }
        case 0x50u: { next_pc = read_u32(pc + 1u) + config.code_offset; }
        case 0x51u: { let cond = pop(vm_id); if (cond.data == 0) { next_pc = read_u32(pc + 1u) + config.code_offset; } else { next_pc = pc + 5u; } }
        case 0x61u: { let val = pop(vm_id); vm_states[vm_id].result_tag = val.tag; vm_states[vm_id].result_data = val.data; vm_states[vm_id].halted = 1u; }
        case OP_MITOSIS: { 
            let offset_val = pop(vm_id); let slot = atomicAdd(&spawn_requests[0], 1u);
            push(vm_id, TAG_INT, 0); 
            if (slot < 4096u) {
                atomicStore(&spawn_requests[1u + slot * 3u], vm_id);
                atomicStore(&spawn_requests[2u + slot * 3u], next_pc);
                atomicStore(&spawn_requests[3u + slot * 3u], u32(offset_val.data));
            }
        }
        case OP_MUTATOR: {
            // MUTATOR: pop offset then value, write value byte to bytecode[PC + offset]
            let offset_val = pop(vm_id);
            let value_val = pop(vm_id);
            let tgt = pc + u32(offset_val.data);
            if (tgt >= config.code_offset + config.bytecode_len) {
                vm_states[vm_id].error = ERR_MUTATOR_OOB; vm_states[vm_id].halted = 1u;
            } else {
                // Write a byte: compute word index and shift
                let word_idx = tgt / 4u;
                let byte_off = (tgt % 4u) * 8u;
                let mask = ~(0xFFu << byte_off);
                bytecode[word_idx] = (bytecode[word_idx] & mask) | ((u32(value_val.data) & 0xFFu) << byte_off);
            }
        }
        case OP_TELEMETRY: { /* telemetry: no-op for now */ }
        case OP_SYSCALL: {
            // Syscalls require host runtime support — trap on GPU.
            // Read and skip the syscall number byte so pc stays consistent.
            let _nr = read_byte(pc + 1u);
            next_pc = pc + 2u;
            vm_states[vm_id].error = 3u; // 3 = syscall trap
            vm_states[vm_id].halted = 1u;
        }
        case 0xFFu: {
            let sp = vm_states[vm_id].sp;
            if (sp > 0u) {
                let val = stacks[vm_id * MAX_STACK + sp - 1u];
                vm_states[vm_id].result_tag = val.tag;
                vm_states[vm_id].result_data = val.data;
            }
            vm_states[vm_id].halted = 1u;
        }
        default: { vm_states[vm_id].halted = 1u; }
    }
    vm_states[vm_id].pc = next_pc;
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let vm_id = gid.x;
    if (vm_id >= config.num_vms || vm_states[vm_id].halted != 0u) { return; }
    for (var i = 0u; i < MAX_STEPS; i = i + 1u) {
        if (vm_states[vm_id].halted != 0u) { break; }
        exec_step(vm_id);
        vm_states[vm_id].steps = vm_states[vm_id].steps + 1u;
    }
    write_vcc_pixel(vm_id);
}

fn rot(n: u32, x: u32, y: u32, rx: u32, ry: u32) -> vec2<u32> {
    var ox = x; var oy = y;
    if (ry == 0u) {
        if (rx == 1u) { ox = n - 1u - x; oy = n - 1u - y; }
        return vec2<u32>(oy, ox);
    }
    return vec2<u32>(ox, oy);
}

fn d2xy(n: u32, d: u32) -> vec2<u32> {
    var x = 0u; var y = 0u; var t = d; var s = 1u;
    while (s < n) {
        let rx = 1u & (t / 2u); let ry = 1u & (t ^ rx);
        let res = rot(s, x, y, rx, ry);
        x = res.x + s * rx; y = res.y + s * ry;
        t = t / 4u; s = s * 2u;
    }
    return vec2<u32>(x, y);
}

fn write_vcc_pixel(vm_id: u32) {
    let pos = d2xy(256u, vm_id);
    var color = vec4<f32>(0.0, 0.0, 0.0, 1.0);
    let state = vm_states[vm_id];
    if (state.steps > 0u) {
        if (state.error != 0u) { color = vec4<f32>(1.0, 0.0, 0.0, 1.0); }
        else if (state.halted == 0u) { color = vec4<f32>(0.0, 1.0, 0.0, 1.0); }
        else { color = vec4<f32>(0.0, 0.5, 1.0, 1.0); }
    }
    textureStore(vcc_texture, pos, color);
}
