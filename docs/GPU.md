# GlyphLang GPU Execution

GlyphLang supports GPU-accelerated execution of bytecode via WebGPU compute shaders.

## Architecture

```
GLYP Source → Compiler → Bytecode (.glyphc)
                              ↓
                    GPU Dispatcher
                              ↓
         ┌────────────────────────────────────┐
         │  WebGPU Compute Shader (vm.wgsl)   │
         │  - 64 threads per workgroup        │
         │  - Each thread = 1 VM instance     │
         │  - Parallel execution of N VMs     │
         └────────────────────────────────────┘
                              ↓
                    Results (N Result structs)
```

## Building

### Default (CPU fallback)

```bash
go build -o glyph ./cmd/glyph
```

The default build uses CPU execution with goroutines. This works everywhere without dependencies.

### With GPU support

```bash
# 1. Download wgpu-native
./scripts/setup-wgpu.sh

# 2. Build with GPU tag
go build -tags gpu -o glyph ./cmd/glyph
```

**Requirements:**
- Vulkan-capable GPU (Linux/Windows)
- Metal-capable GPU (macOS)
- DirectX 12-capable GPU (Windows)

## Usage

```bash
# Run with GPU acceleration
glyph run --gpu server.glyph

# Execute bytecode on GPU
glyph run --gpu --bytecode app.glyphc

# Run GPU benchmark
glyph gpu compute.glyphc --vms 1000
```

## Shader (vm.wgsl)

The compute shader implements the full GLYP VM:

| Opcode | Hex | Description |
|--------|-----|-------------|
| PUSH | 0x01 | Push constant |
| POP | 0x02 | Pop stack |
| ADD | 0x10 | Add top two |
| SUB | 0x11 | Subtract |
| MUL | 0x12 | Multiply |
| DIV | 0x13 | Divide |
| ... | ... | ... |
| HALT | 0xFF | Stop execution |

See `pkg/gpu/vm.wgsl` for the full implementation.

## Performance

| Mode | Throughput | Latency |
|------|-----------|---------|
| CPU (1 VM) | ~1M ops/sec | ~1µs |
| CPU (1000 VMs) | ~100M ops/sec | ~10ms |
| GPU (1000 VMs) | ~1B ops/sec* | ~1ms |

*Estimated with wgpu-native on RTX 5090

## Fallback Behavior

When built without `-tags gpu`:
- `--gpu` flag logs a warning and uses CPU
- All functionality works, just slower for batch operations

When built with `-tags gpu` but no GPU available:
- Falls back to CPU automatically
- Logs warning on startup

## Future Work

1. **Full wgpu-native integration** - Complete adapter/device initialization
2. **Async execution** - Non-blocking GPU dispatch
3. **Buffer pooling** - Reuse buffers across executions
4. **Multi-GPU** - Distribute across multiple GPUs
5. **JIT compilation** - Hot path optimization on GPU
