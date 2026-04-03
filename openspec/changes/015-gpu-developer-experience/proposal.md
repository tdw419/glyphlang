# GPU Developer Experience

## Why
Developers need live visualization and REPL access to GPU execution.

## Scope

### SEC-1: REPL GPU mode
- `glyph repl --gpu` executes expressions on the GPU daemon

### SEC-2: Live VCC streaming
- Replace file writes with SHM/pipe, target 30Hz

### SEC-3: Terminal spatial visualization
- `--spatial` reads from live VCC texture, not static analysis

### SEC-4: Glyph file examples
- colony_linear.glyph, colony_recursive.glyph, colony_conditional.glyph

## Acceptance Criteria
- `go test ./...` passes
