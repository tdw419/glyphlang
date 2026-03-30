# GlyphLang v0.11.0 — Drop Go Dependency Plan

## Current State
- Bootstrap parser (3.5K lines) ✅ — parses all GlyphLang syntax
- Bootstrap compiler (640 lines) ✅ — compiles to bytecode (int-only VM)
- Bootstrap VM (1.2K lines) ✅ — executes bytecode (int-only)
- Go interpreter (1.6K lines eval) — handles full language, `glyph exec` uses this
- Go CLI — file I/O, import resolution, command dispatch

## Strategy: Tree-Walking Interpreter
The `glyph exec` command already uses a tree-walking interpreter, not the bytecode VM.
The bootstrap code should do the same — walk the AST directly.

This avoids the int-only VM limitation and lets us work with native GlyphLang values
(strings, objects, arrays) directly.

## Phase 1: Core Interpreter (eval_expr + exec_stmt)
Files: bootstrap/interpreter.glyph

### eval_expr cases needed:
- literal → return value
- variable → env.get(name)
- binary_op → eval left, eval right, apply op
- unary_op → eval operand, apply op
- call → eval callee, eval args, invoke function
- field → eval object, get field
- object → eval each field, construct map
- array → eval each element, construct list
- index → eval collection, eval index, lookup
- import_prefix → resolve module, eval call on result

### exec_stmt cases needed:
- assign → eval expr, env.set(name, value)
- return → eval expr, signal return
- if → eval condition, exec block
- while → eval condition loop, exec block
- for_in → eval iterable, loop with iterator
- expression → eval expr (discard result)
- field_assign → eval object, eval value, set field
- index_assign → eval collection, eval index, eval value, set
- break/continue → signal

## Phase 2: Environment
- Scopes: global → function → block
- Function definitions stored as values
- Closures capture environment

## Phase 3: Builtins
- print, toString, length, typeOf, toInt
- Array operations: push, pop, concat (+), index
- String operations: split, join, indexOf, substring

## Phase 4: Module System
- import "./module" → parse file, create module scope
- Module exports: functions defined with ! become accessible
- Circular import detection

## Phase 5: Command Dispatch
- @ command name → register function
- Entry point: find command by name, call it with args

## Phase 6: File I/O
- Read .glyph source files
- Import path resolution (relative to current file)

## Phase 7: Self-Hosting Test
- interpreter.glyph interprets itself interpreting a test program
- Full bootstrap: glyph exec bootstrap/interpreter.glyph run

## Estimated effort: 800-1200 lines of GlyphLang
## Reference: pkg/interpreter/evaluator.go (1646 lines)
