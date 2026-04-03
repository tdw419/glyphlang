# Acceptance Criteria

## Tests
- [x] SEC-1.1: Test that NewGPUDispatcher() sets hasGPU=true when Rust runner binary exists
- [x] SEC-1.2: Test that NewDispatcher() detects GPU availability
- [ ] SEC-2.1: Test that glyph gpu command uses GPU dispatcher when available
- [ ] SEC-3.1: Test that --vms flag works with glyph run --gpu
- [ ] SEC-4.1: Test that execution mode is printed correctly

## Verification
- [ ] `go test ./...` passes
