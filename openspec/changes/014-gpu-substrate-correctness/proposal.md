# GPU Substrate Correctness

## Why
Once GPU is wired to CLI, the substrate needs to behave correctly. Mitosis spawning, depth guards, and error propagation are untested or broken.

## Scope

### SEC-1: CPU Mitosis implementation
- OP_MITOSIS silently ignored in CPU path -- either implement or return clear error

### SEC-2: Spawn offset semantics
- spawn(offset) should set child PC = parent.PC + offset
- Verify PUSH 0; MITOSIS doesn't cause infinite loops

### SEC-3: Depth guard testing
- MAX_PASSES=8 and MAX_VMS=65536 boundary tested

### SEC-4: Error propagation from Rust daemon
- Rust daemon crash should not hang Go side

## Acceptance Criteria
- `go test ./...` passes
