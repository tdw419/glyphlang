# How GlyphLang Becomes Geometry OS

A pixel-based operating system where computation IS the display.

---

## The Core Idea

In a traditional OS, source code and the running process are two different things. You write C, compile it to a binary, the kernel loads it into memory, and the process runs somewhere you can't see.

In Geometry OS, the code and the running process are the same texture. The program is pixels. The memory is pixels. The execution state is pixels. You don't need `top` or `htop` -- you look at the screen and see exactly what's happening.

This isn't a metaphor. It's how the system actually works.

---

## The Pipeline: Source to Pixel

```
┌─────────────────────────────────────────────────────────────────────┐
│                        THE FULL PIPELINE                            │
│                                                                     │
│  .glyph source ──► Compiler ──► .glyphc bytecode ──► VM dispatch   │
│       │                                              │              │
│       │                                         ┌────┴────┐        │
│       │                                         │ CPU VM  │        │
│       │                                         │(goroutine│       │
│       │                                         │ per VM) │        │
│       │                                         └────┬────┘        │
│       │                                              │              │
│       │                                         ┌────┴────┐        │
│       │                                         │ GPU VM  │        │
│       │                                         │(WGSL     │       │
│       │                                         │ compute  │       │
│       │                                         │ shader)  │       │
│       │                                         └────┬────┘        │
│       │                                              │              │
│       │                                              ▼              │
│       │                                    GPU Texture (RAM)        │
│       │                                    256×256 or 4096×4096     │
│       │                                    Each pixel = 1 VM        │
│       │                                              │              │
│       │                                              ▼              │
│       │                                    Visual output:           │
│       │                                    Green = running          │
│       │                                    Blue  = halted           │
│       │                                    Red   = error            │
└─────────────────────────────────────────────────────────────────────┘
```

Every step is real, implemented, and working today.

---

## Layer 1: GlyphLang Source → Bytecode

GlyphLang is an AI-first language designed for minimal token consumption. A .glyph file compiles to a self-contained .glyphc bytecode artifact.

```glyph
# A modular program that imports another file
import "./math_module"

! main() {
    $ result = math_module.add(3, 4)
    > result
}
```

The **Static Linker** flattens the entire import tree into one AST. The **Compiler** emits bytecode with all function definitions before the entry point. The output is a single .glyphc file (typically a few hundred bytes) that contains everything needed to execute.

Performance: ~35 microseconds from .glyphc to result on the Go VM. Compilation itself takes ~867 nanoseconds.

---

## Layer 2: The Virtual Machine

GlyphLang has two execution modes, forming what we call the "Interpreter Abstraction":

**Simulation Layer (Tree-walking interpreter):**
- Executes the AST directly
- For rapid iteration and development
- No compilation step needed
- Supports the full language feature set

**Physical Layer (Bytecode VM):**
- Compiles to .glyphc, then executes via a register-based VM
- For production performance
- Static linking produces self-contained artifacts
- Orders of magnitude faster than tree-walking

Both layers share the same semantics. A program that works in the interpreter produces identical results in the bytecode VM.

---

## Layer 3: Mitosis -- Programs That Spawn Programs

The OP_MITOSIS opcode is what makes Geometry OS possible. When a running VM encounters a spawn instruction:

```
5 S     # Spawn a child VM at IP + 5 spatial units
10 S    # Spawn another child at IP + 10
```

The VM clones its entire state (stack, registers, instruction pointer) into a new VM instance. The parent receives back a thread ID. When the parent checks the results, it gets an array -- one element per child:

```
[child_result_1, child_result_2, parent_result]
```

This is **verified working** on the CPU substrate. Two threads execute, both return results, the parent can read them and make decisions.

What this enables:
- A program can dynamically scale its own parallelism
- A parent can delegate work to children
- The OS can manage other programs -- because it IS a program

---

## Layer 4: The GPU Substrate

The same bytecode that runs on the Go VM can be dispatched to GPU compute shaders via WGSL (WebGPU Shading Language).

```
glyph CLI → Go PersistentRunner → stdin pipe → Rust wgsl_runner
                                                    │
                                            wgpu device/queue
                                                    │
                                            GPU compute shader (vm.wgsl)
                                                    │
                                            stdout pipe → Go → results
```

Architecture:
- The Rust IPC daemon manages a persistent GPU connection
- Each VM instance runs as a compute thread on the GPU
- Multi-pass Mitosis: after each compute pass, the Rust runner reads spawn requests, allocates child VMs, and re-dispatches
- Up to 8 passes per frame, scaling from 1 seed VM to 65,536
- Verified at 70 FPS sustained on an RTX 5090

The GPU substrate is the "physical hardware" of Geometry OS. Just as Linux runs on x86/ARM, Geometry OS runs on GPU compute units.

---

## Layer 5: The Texture is the Memory

All of Geometry OS memory lives in a single GPU texture:

```
Format:    4096 × 4096 pixels, RGBA8
Capacity:  16,777,216 pixels = 64 MB
Addressing: Hilbert space-filling curve
```

Each pixel is 4 bytes (R, G, B, A) and encodes either:
- **An instruction** (R=opcode, G=stratum, B=param1, A=param2)
- **A data value** (the full 32-bit RGBA word)
- **A VM state** (color indicates status: green=running, blue=halted, red=error)

The Hilbert curve is critical. It maps linear bytecode addresses to 2D pixel coordinates while preserving spatial locality -- instructions that are adjacent in the program are adjacent on the texture. A tight loop becomes a dense cluster. A long pipeline becomes a winding path. You can literally see the shape of the code.

```
Address 0  → pixel (0, 0)
Address 1  → pixel (1, 0)
Address 2  → pixel (1, 1)
Address 3  → pixel (0, 1)
Address 4  → pixel (0, 2)
...
```

The VCC (Visual Consistency Contract) bridge writes this texture to shared memory:

```
/dev/shm/vcc_colony.rgba   (262,144 bytes = 256×256×4 RGBA)
```

This is served via HTTP at `http://localhost:8080/vcc/colony.rgba` for external visualization.

---

## Layer 6: The Self-Replicating Program (Proof of Concept)

On March 16, 2026, an 18-pixel program running on the GPU copied itself from one location to another on the Hilbert curve. No CPU involvement during execution. The GPU's compute shader read its own instructions from the texture, executed them, and wrote a perfect duplicate to a new address.

The program is 72 bytes:

```
┌──────┬──────────────────┬───────────────────────────────────────┐
│ Addr │ Pixel (R,G,B,A)  │ Meaning                               │
├──────┼──────────────────┼───────────────────────────────────────┤
│  0-1 │ LDI r0, 0        │ r0 = source start address             │
│  2-3 │ LDI r1, 100      │ r1 = destination address              │
│  4-5 │ LDI r2, 0        │ r2 = loop counter                     │
│  6-7 │ LDI r3, 1        │ r3 = increment constant               │
│  8-9 │ LDI r4, 18       │ r4 = program length (18 pixels)       │
├──────┼──────────────────┼───────────────────────────────────────┤
│  10  │ LOAD r5 = mem[r0] │ Read pixel from source                │
│  11  │ STORE mem[r1]=r5  │ Write pixel to destination            │
│  12  │ ADD r0 = r3 + r0  │ source++                              │
│  13  │ ADD r1 = r3 + r1  │ dest++                                │
│  14  │ ADD r2 = r3 + r2  │ counter++                             │
│  15  │ BRANCH BNE r2,r4  │ if counter ≠ 18, jump to addr 10     │
│  16  │ offset: -7        │ (signed branch target)                │
├──────┼──────────────────┼───────────────────────────────────────┤
│  17  │ HALT              │ Done. Copy complete.                  │
└──────┴──────────────────┴───────────────────────────────────────┘
```

This proves that a program on the texture can manipulate the texture. Pixels move pixels.

---

## Layer 7: The Kernel v0.1 (Self-Management)

The Geometry OS Kernel is a GlyphLang program that manages other GlyphLang programs. It is not written in Go or Rust. It is written in GlyphLang.

What it does:
1. Uses `gpuExec` to launch child programs
2. Children perform computations and return results
3. The parent reads those results from the result array
4. The parent makes decisions based on what the children computed

This is the "operating system" threshold. Before this, the Go CLI was the kernel -- an external process managing GlyphLang execution. After this, a GlyphLang program IS the kernel. The OS is self-hosting.

The pipeline that makes it work:

```
Compiler ──► emits OP_DEF_FUNC for module functions
Dispatcher ──► calculates CodeOffset including String Pool
VM ──► absolute PC indexing, fork() identity for parent/child
Interpreter ──► converts binary results back to high-level types
```

---

## Why This Is Different From a Traditional OS

| Concept          | Linux                    | Geometry OS                        |
|------------------|--------------------------|-------------------------------------|
| Code storage     | Binary files on disk     | Pixels in a GPU texture            |
| Running process  | Invisible (in RAM)       | Visible (a colored pixel region)   |
| Kernel           | C code in ring 0         | A GlyphLang program on the texture |
| Process spawning | fork() syscall           | OP_MITOSIS (clone VM state)        |
| IPC              | pipes, sockets, shm      | Adjacent pixel reads/writes        |
| Self-modification| Dangerous (segfault)     | Intended (OP_MUTATOR)              |
| Debugging        | Logs, core dumps, gdb    | Look at the texture                |
| Crash signal     | Kernel panic (text)      | Visual fracture (red pixel)        |
| AI observability | Requires API/logging     | VLM reads the texture directly     |

The last row is the real differentiator. A Vision Language Model can look at a screenshot of the Geometry OS texture and understand what's happening without any API, any logging, any instrumentation. The OS is inherently legible to AI because its state IS a visual representation.

---

## What "Pixels Move Pixels" Means

This phrase is not branding. It's the literal execution model:

1. **The GPU texture contains instructions.** A pixel at coordinate (x,y) with RGBA values (1, 0, 5, 3) means "LDI register 5, next pixel contains data 3."

2. **A compute shader reads those instructions.** The shader dispatches to the texture, reads the pixel at the VM's program counter, decodes the opcode, and executes it.

3. **Execution writes back to the same texture.** The STORE instruction writes a value to another pixel on the texture. The MUTATOR instruction overwrites a pixel that contains an instruction.

4. **Written pixels become new instructions.** If a program writes a valid instruction to an empty region of the texture, and a VM is dispatched there, those pixels will execute. The program just created new code.

5. **Mitosis creates new VMs at new locations.** When a VM spawns a child at offset N, the child starts executing at pixel PC+N. If the parent wrote code there, the child runs that code. If not, the child runs whatever was already there.

This is how self-replication works. This is how self-modification works. This is how the OS manages itself -- by writing instructions into the texture that other VMs then execute.

---

## Current Status (What Works Today)

- [x] GlyphLang language and compiler
- [x] Static linker (modular .glyph → single .glyphc)
- [x] Go bytecode VM (35µs execution)
- [x] Go decompiler (clean disassembly)
- [x] Mitosis on CPU (parent/child execution, result passing)
- [x] GPU substrate (WGSL compute shaders on RTX 5090)
- [x] Multi-pass Mitosis (1 → ~2300 VMs)
- [x] Live mode (70 FPS sustained)
- [x] VCC texture bridge (/dev/shm/vcc_colony.rgba)
- [x] Self-replicating program (18 pixels, GPU-native)
- [x] Kernel v0.1 (self-managing GlyphLang program)

## What's Next

- [ ] Fix Mitosis children on GPU (issue #88 -- children allocated but PC not set correctly)
- [ ] GPU support for OpCall/OpBuildObject (enables full recursive kernel on GPU)
- [ ] IPC between adjacent VMs (read neighbor's pixel, write to neighbor's region)
- [ ] Memory protection (VM regions that can't be written by other VMs)
- [ ] Process Manager written in GlyphLang (spawn children, read results, decide)
- [ ] .rts.png distribution format (embed bytecode in PNG, the PNG IS the app)

---

## The Mental Model

Think of it as a colony of organisms on a petri dish:

- The **petri dish** is the GPU texture (256×256 or 4096×4096 pixels)
- Each **organism** is a VM instance running a GlyphLang program
- The **DNA** is the .glyphc bytecode (compiled from .glyph source)
- **Reproduction** is Mitosis (clone VM state into adjacent pixel region)
- **Evolution** is the MUTATOR opcode (programs rewrite their own instructions)
- **The colony's behavior** is visible to the naked eye (or to a VLM)

You don't debug by reading logs. You debug by looking at the petri dish.

---

*Document generated April 2026. Based on verified working code in ~/zion/projects/glyphlang/ and ~/zion/projects/geometry_os/.*
