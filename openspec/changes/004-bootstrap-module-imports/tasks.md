# Tasks: Bootstrap Module Imports

## 1. Add readFile builtin to bootstrap interpreter
- [ ] 1.1 Implement `readFile(path)` as a builtin function accessible from .glyph code, returning the file contents as a string value
- [ ] 1.2 Test: call `readFile("test.glyph")` from a .glyph program and verify it returns the file contents

## 2. Implement import path resolution and module loading
- [ ] 2.1 When interpreter encounters `import "./parser"`, resolve to `./parser.glyph` relative to current file directory, call readFile, parse source, execute in a new module scope, and bind exported `!` functions to the module namespace
- [ ] 2.2 Test: create two files where `main.glyph` does `import "./helper"` and calls `helper.add(1, 2)`, verify result is 3

## 3. Add circular import guard and module cache
- [ ] 3.1 Add a `loaded_modules` map to interpreter state. Before loading a module, check if already loaded and return cached namespace. Prevents infinite recursion on circular imports.
- [ ] 3.2 Test: create two files that import each other and verify no infinite loop (second import returns cached/empty namespace)
