# WGSL Direct Lowering

GlyphLang supports direct lowering from SSA intermediate representation to WGSL compute shaders. This bypasses the bytecode virtual machine entirely for GPU-bound routes, enabling native execution on modern GPUs (RTX 5090).

## Pipeline

1. **AST**: High-level GlyphLang source.
2. **SSA**: Static Single Assignment IR with typed values.
3. **Optimizer**: Constant folding, dead code elimination, algebraic simplification.
4. **WGSL Lowering**: Emits structured WGSL code.
5. **wgpu**: Compiles WGSL to SPIR-V/MSL/DXIL for the target hardware.

## Architecture

The generated WGSL uses a storage-buffer based approach for telemetry and state management:

- `@binding(0)`: `states` - Array of `VMState` structs (result tags, return data).
- `@binding(1)`: `locals` - Array of `i32` for persistent variable storage.

Control flow is implemented using a structured `while-switch` state machine, which correctly handles arbitrary SSA jumps and branches.

## Example Output

For a simple route:
```glyph
@ GET /calc {
  $ a = 40
  $ b = 2
  > a + b
}
```

The generated WGSL looks like:
```wgsl
fn main(...) {
    var v1: i32; // const 40
    var v2: i32; // const 2
    var v3: i32; // result 42
    
    // Main loop
    while (!halted) {
        switch (block_id) {
            case 0: {
                v1 = 40i;
                v2 = 2i;
                v3 = v1 + v2;
                states[id].result_tag = 1;
                states[id].result_data = v3;
                halted = true;
            }
        }
    }
}
```

## Benefits

- **Performance**: No bytecode decoding overhead.
- **Optimization**: Leverages the GPU driver's compiler for register allocation.
- **Telemetry**: Zero-copy writes to the unified `vm_stats` memory space.
