# GlyphLang Bootstrap Self-Host Roadmap

## Current Status (2026-03-31)

| Component | Status | Details |
|-----------|--------|---------|
| Bootstrap interpreter | **230/230 tests passing** | Lexer, parser, interpreter in .glyph |
| Bootstrap compiler + VM | **Working** | Basic programs, for-in loops, nested functions |
| Go toolchain | **Working** | Nested `! inner()` function definitions |
| GPU substrate | **Partial** | 70fps live mode on RTX 5090, Mitosis broken (#88) |

## Session Fixes Applied

1. ITER_NEXT operand width: 4 bytes -> 1 byte (compiler.glyph)
2. Jump target patching: patch copy, not original (compiler.glyph)
3. Nested `! inner()` function support (Go parser/AST/interpreter)

---

## Dependency Graph

```
Phase 1 (Tests)          Phase 4 (GPU, parallel)
  001-fix-tests            006-gpu-mitosis-fix *
       |
       v
Phase 2 (VM Gap Closure)
  002-vm-string-support
       |
       +---> 003-for-in-key-value
       |
       v
Phase 3 (Self-Hosting)
  004-module-imports
       |
       v
  005-vm-self-host (milestone)
```

* Change 006 is independent and runs in parallel with the bootstrap track.

## Change Index

| # | ID | Title | Priority | Status | Depends On |
|---|----|-------|----------|--------|------------|
| 001 | fix-broken-bootstrap-tests | Fix test_compiler_mutable_closure + test_vm_bootstrap | CRITICAL | draft | none |
| 002 | bootstrap-vm-string-support | String type tagging and concatenation in VM | HIGH | draft | 001 |
| 003 | bootstrap-for-in-key-value | Verify for-in key,value pairs end-to-end | HIGH | draft | 001, 002 |
| 004 | bootstrap-module-imports | Module import via readFile builtin | HIGH | draft | 001, 002 |
| 005 | bootstrap-vm-self-host | Self-hosting: interpreter interprets itself | HIGH | draft | 001, 002, 003, 004 |
| 006 | gpu-mitosis-fix | Fix #88 Mitosis children PC=0 bug | CRITICAL | draft | none |

---

## Success Criteria

### Change 001: Fix Broken Tests
- [ ] test_compiler_mutable_closure.glyph passes (no parse error)
- [ ] test_vm_bootstrap.glyph passes (run_tests via @ wrapper)
- [ ] New nested-functions e2e test passes (parse -> compile -> vm_exec)
- [ ] Existing 230/230 tests remain green

### Change 002: VM String Support
- [ ] VM stack values have type tags (INT, STRING, REF)
- [ ] String pool in VM, OP_LOAD_STRING opcode works
- [ ] `"hello" + " " + "world"` produces `"hello world"`
- [ ] All existing int-only tests still pass

### Change 003: For-In Key,Value
- [ ] `for k, v in [10, 20, 30]` works through full pipeline
- [ ] ITER_NEXT hasKey=1 verified in bytecode dump
- [ ] VM pushes both key and value correctly

### Change 004: Module Imports
- [ ] `import "./parser"` resolves, loads, executes parser.glyph
- [ ] Imported functions callable as `module.func()`
- [ ] Circular import guard prevents infinite recursion

### Change 005: Self-Hosting Milestone
- [ ] interpreter.glyph can load and execute a simple .glyph program
- [ ] Meta-circular: interpreter interprets itself interpreting fibonacci(10) == 55
- [ ] Performance documented (expected: slow but correct)

### Change 006: GPU Mitosis Fix
- [ ] Mitosis children execute with correct PC (not 0)
- [ ] 4-child parallel test passes on GPU
- [ ] CPU fallback detection infrastructure in place

---

## Recommended Execution Order

```
Sprint 1 (both tracks in parallel):
  -> 001-fix-broken-bootstrap-tests  (1-2 AIPM cycles)
  -> 006-gpu-mitosis-fix             (1-2 AIPM cycles)

Sprint 2:
  -> 002-bootstrap-vm-string-support (2-3 AIPM cycles)

Sprint 3:
  -> 003-bootstrap-for-in-key-value  (1 AIPM cycle)
  -> 004-bootstrap-module-imports    (1-2 AIPM cycles)

Sprint 4 (milestone):
  -> 005-bootstrap-vm-self-host      (2-3 AIPM cycles)
```

## AIPM SpecQueue Integration

Each change directory under `openspec/changes/` contains:
- `status.yaml` -- machine-readable metadata (id, title, status, priority, depends_on, labels)
- `proposal.md` -- human-readable Why/What/Impact
- `tasks.md` -- actionable checklist (## Section, - [ ] N.M steps)

The AIPM agent reads these to:
1. Pick the next draft change (lowest number with met dependencies)
2. Set status to `in_progress`
3. Execute tasks one at a time
4. Set status to `completed` when all tasks checked off

GitHub issues should carry labels matching the `labels:` field in status.yaml plus the `autospec` tag for AIPM visibility.
