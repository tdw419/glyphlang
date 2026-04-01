# Tasks: Fix #88 Mitosis Children Not Executing

## 1. Fix PC initialization in Rust runner
- [x] 1.1 Locate the Mitosis child spawn code in the Rust runner and identify where PC is initialized to 0; change it to use the child function's bytecode entry offset from the spawn instruction's operand
- [x] 1.2 Add debug logging to print child PC, parent PC, and spawn offset when MITOSIS_SPAWN opcode executes

## 2. Add Mitosis integration test
- [x] 2.1 Write a test .glyph program that spawns 4 Mitosis children, each computing `result[pid] = pid * 2`, then verify all 4 results are correct after synchronization
- [x] 2.2 Run the test on GPU and verify children execute with correct PC values and produce expected output

## 3. Add CPU Mitosis fallback detection
- [ ] 3.1 Add error handling in the Mitosis path that catches GPU execution failures and logs a warning, setting up the infrastructure for CPU fallback (full fallback is #78)
