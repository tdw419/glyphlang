# GlyphLang Minimal Runtime v0.8.0

## Purpose

A stripped-down runtime optimized for self-hosted execution. Only includes opcodes and builtins actually used by the self-hosted compiler.

## Opcodes Required (24)

| Opcode | Value | Description |
|--------|-------|-------------|
| OP_PUSH | 1 | Push constant |
| OP_POP | 2 | Discard top |
| OP_ADD | 16 | Integer add |
| OP_SUB | 17 | Integer subtract |
| OP_MUL | 18 | Integer multiply |
| OP_DIV | 19 | Integer divide |
| OP_MOD | 20 | Integer modulo |
| OP_EQ | 32 | Equality test |
| OP_NE | 33 | Not equal |
| OP_LT | 34 | Less than |
| OP_GT | 35 | Greater than |
| OP_GE | 36 | Greater or equal |
| OP_LE | 37 | Less or equal |
| OP_AND | 38 | Logical and |
| OP_OR | 39 | Logical or |
| OP_NOT | 40 | Logical not |
| OP_NEG | 41 | Arithmetic negate |
| OP_LOAD_VAR | 64 | Load variable |
| OP_STORE_VAR | 65 | Store variable |
| OP_JUMP | 80 | Unconditional jump |
| OP_JUMP_IF_FALSE | 81 | Conditional jump |
| OP_GET_INDEX | 86 | Array index get |
| OP_SET_INDEX | 87 | Array index set |
| OP_RETURN | 97 | Return from call |
| OP_CALL | 98 | Function call |
| OP_BUILD_OBJECT | 112 | Create object |
| OP_GET_FIELD | 113 | Object field get |
| OP_SET_FIELD | 114 | Object field set |
| OP_DEF_FUNC | 115 | Define function |
| OP_BUILD_ARRAY | 128 | Create array |
| OP_HALT | 255 | Stop execution |

## Builtins Required (8)

| Builtin | Purpose |
|---------|---------|
| `length(arr)` | Array/string length |
| `toString(val)` | Convert to string |
| `toInt(val)` | Convert to integer |
| `intToBytes4(n)` | Integer to 4 bytes |
| `readFile(path)` | Read file contents |
| `writeFile(path, data)` | Write file contents |
| `print(msg)` | Output message |
| `typeOf(val)` | Get value type |

## Implementation Strategy

1. **Opcode switch** - Direct dispatch, no function table overhead
2. **Inline builtins** - No external calls for core operations
3. **Stack-optimized** - Pre-allocated stack with bounds checking
4. **No garbage collection** - Simple arena allocator for self-hosted execution

## Size Target

- C implementation: < 50KB compiled
- Go implementation: < 500KB compiled (current)
- Rust implementation: < 200KB compiled (future)

## Bytecode Format

```
[0:3]   Magic: "GLYP"
[4:7]   Version: 0x00000001
[8:11]  Constant count (little-endian u32)
[12:N]  Constants (length-prefixed strings, u32 for ints)
[N+0:3] Instruction count (little-endian u32)
[N+4:M] Instructions (opcode + operands)
```

## Usage

```bash
# Compile with self-hosted compiler
./glyph exec bootstrap/compiler.glyph compile --src input.glyph --out output.glyphc

# Run with minimal runtime
./glyph run output.glyphc
```
