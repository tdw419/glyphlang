# GPU Developer Experience

## Why
Developers need live visualization and REPL access to GPU execution.

## Scope

### SEC-1: REPL GPU mode
- `glyph repl --gpu` evaluates expressions on the GPU daemon
- Falls back to CPU when daemon unavailable

### SEC-2: Live VCC streaming
- Replace file writes with SHM/pipe, target 30Hz
- HTTP endpoint already exists at :8080/vcc/colony.rgba

### SEC-3: Terminal spatial visualization
- `--spatial` reads from live VCC texture, not static analysis
- Show colony grid updates in real-time

### SEC-4: Glyph file examples
- colony_linear.glyph, colony_recursive.glyph, colony_conditional.glyph
- Each demonstrates a different Mitosis pattern

## Acceptance Criteria
- `go test ./...` passes
