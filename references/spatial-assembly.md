# .glyph Spatial Assembly Opcode Reference

Low-level stack-based instruction set using Reverse Polish Notation (RPN). Eliminates parentheses and operator precedence for deterministic AI parsing.

## Stack Architecture

- **Data Stack:** Unified numeric/pointer stack for immediate values and function arguments
- **Registers:** 26 isolated registers (A-Z) for state across routines
- **Instruction Pointer (IP):** Current execution position on the spatial grid

## Register Protocol

| Case | Instruction | Effect |
|------|-------------|--------|
| Lowercase `a`-`z` | Store | Pop top of stack → store in register |
| Uppercase `A`-`Z` | Load | Load from register → push onto stack |

Register isolation enables parallel agents to maintain independent states on the same visual grid.

## Complete Instruction Set

### Literals
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `0`-`9` | `( -- n)` | Push single-digit integer onto stack |

Multi-digit numbers are composed: `3 5` pushes 3 then 5 separately.

### Arithmetic
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `+` | `( a b -- r)` | Add: r = a + b |
| `-` | `( a b -- r)` | Subtract: r = a - b |
| `*` | `( a b -- r)` | Multiply: r = a * b |
| `/` | `( a b -- r)` | Divide: r = a / b |

### Comparison
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `>` | `( a b -- bool)` | Greater than: pushes 1 if a > b, else 0 |
| `<` | `( a b -- bool)` | Less than: pushes 1 if a < b, else 0 |
| `=` | `( a b -- bool)` | Equal: pushes 1 if a = b, else 0 |

### Control Flow
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `?` | `( c t f -- r)` | Conditional: if c=1 push t, else push f |
| `L` | `( s e -- [r])` | Loop: generate range from s to e |

### Metamorphic (Ouroboros)
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `M` | `( v o -- )` | **Mutator:** Overwrites code at IP + o with value v. Self-modification opcode. |

### Biological (Parallelism)
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `S` | `( o -- id)` | **Mitosis:** Clone entire VMState (stack, registers, IP) into new thread at spatial offset o. Returns thread ID. |

### I/O and Termination
| Opcode | Stack Effect | Description |
|--------|-------------|-------------|
| `.` | `( v -- )` | Output top value to visual debug grid |
| `@` | `( -- )` | Cleanly terminate execution thread |
| `@>` | `( -- )` | Request natural language (human) intervention |

## Example Programs

### Basic Arithmetic
```
3 4 + .          # Push 3, push 4, add, print → 7
5 dup * .        # Push 5, duplicate, multiply, print → 25
```

### Conditional
```
3 5 >            # Compare: is 3 > 5? → pushes 0
"yes" "no" ? .   # Conditional: 0 means false → prints "no"
```

### Mitosis (Spawn Parallel Agents)
```
5 S              # Spawn Agent 1 at IP + 5 spatial units
10 S             # Spawn Agent 2 at IP + 10 spatial units
15 S             # Spawn Agent 3 at IP + 15 spatial units
.                # Print spawn states to grid
```

Each spawned agent inherits full VMState and begins processing independently.

### Self-Modification (Ouroboros Mutator)
```
\ Self-Correcting Loop
1 10 L           # Push range 1 to 10
dup 5 >          # Check if current > 5
99 2 M           # If true, overwrite instruction 2 ticks ahead with '99'
.                # Output: 1, 2, 3, 4, 5, 99, 99, 99, 99, 99
```

The `M` opcode physically rewrites the program's future instructions. The "bug" is permanently removed from the execution path.

## Spatial Grid Concepts

### Hilbert Curve Mapping
Instructions are mapped to a spatial grid via Hilbert space-filling curves, which preserve locality: linearly adjacent instructions remain spatially adjacent on the grid.

### Dimensional Layers
| Layer | Representation | Function |
|-------|---------------|----------|
| 1D | Token Stream | AI I/O and serial persistence |
| 2D | Shader Grid | Visual execution substrate / VLM interface |
| 4D | Projection | Temporal state mapping |
| 8D | Orthoplex Lattice | High-density logical state and reversibility |

### Visual Consistency Contract (VCC)
- Green flash at (X,Y) → `PATCH_SUCCESS`
- Red flash at (X,Y) → `PATCH_FAIL`
- VLM identifies visual cues instantly for state verification
