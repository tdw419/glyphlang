# GlyphLang v0.11.0 — Interpreter Fix Task

## Project Location
`~/zion/projects/glyphlang/`

## What You're Working On
A tree-walking interpreter written IN GlyphLang, FOR GlyphLang. The goal is self-hosting — 
removing the Go dependency so GlyphLang can run its own code without the Go compiler.

## Current File
`bootstrap/interpreter.glyph` — the new interpreter (~530 lines)

## The Bug
GlyphLang does NOT allow bare index-assignment as a statement. This is INVALID:
```
arr[idx] = val          # ERROR
obj[field] = val        # ERROR
env.bindings[name] = v  # ERROR
```
It MUST use the `$` variable declaration prefix:
```
$ arr[idx] = val        # VALID
$ obj[field] = val      # VALID
```
BUT: you CANNOT use `$` to mutate a captured variable's nested field. If `keys` was
declared in an outer scope, `$ keys[name] = val` creates a NEW local variable, it doesn't
mutate the outer one.

## The Real Fix
The env_set function and all mutation helpers need to be restructured. Options:
1. Use a helper that returns the mutated dict, then reassign: `$ new_bindings = set_key(env.bindings, name, val)` then `$ env.bindings = new_bindings`
2. Build a `set_key` helper that copies the dict with the new value
3. Use field assignment syntax: `$ obj.field = val` (if the object supports it)

Similarly, in the eval_expr and exec_stmt functions, anywhere you see `obj[field] = val` 
or `cr.value[idx] = vr.value`, those need fixing.

## GlyphLang Syntax Rules
- `$ name = expr` — variable declaration/assignment
- `$ name[idx] = expr` — index assignment
- `$ obj.field = expr` — field assignment (use dot notation, not bracket)
- `> expr` — return from function
- `! fname(params) -> Type { }` — function definition
- `: TypeName { field: Type! }` — type definition
- `@ command name { }` — CLI command entry point
- `const NAME = expr` — top-level constant
- `import "./module"` — import relative module

## Test Command
```bash
cd ~/zion/projects/glyphlang && ./glyph exec bootstrap/interpreter.glyph run
```

## Expected Success Output
```
========================================
  GlyphLang Bootstrap Interpreter v0.11
========================================

Parsing: const X = 42
const Y = X + 3
Executing...
X = 42
Y = 45

SELF-TEST PASSED
{ "ok": true, "X": 42, "Y": 45 }
```

## Key Files to Reference
- `bootstrap/ast.glyph` — AST node definitions (tells you what fields each node has)
- `bootstrap/parser.glyph` — the parser (to understand what AST the parser produces)
- `bootstrap/vm.glyph` — the existing bytecode VM (reference for opcode constants)
- `bootstrap/compiler.glyph` — the existing compiler (reference for how AST nodes are handled)

## Approach
1. Read `bootstrap/interpreter.glyph` first
2. Fix all index-assignment syntax errors (the `$` prefix issue)
3. For mutations that need to update outer-scope objects, create helper functions 
   like `dict_set(dict, key, val)` that return new dicts
4. Run `./glyph exec bootstrap/interpreter.glyph run` to test
5. Iterate until the self-test passes
6. Commit with message: "feat: bootstrap tree-walking interpreter v0.11.0 — self-test passing"

## After Self-Test Passes
If the self-test works, try these progressive tests by modifying the `src` variable in the `run` command:
1. Functions: `! add(a: int, b: int) -> int { > a + b }` then call it
2. While loops: `$ i = 0\nwhile i < 10 { ... }`
3. Objects: `$ obj = { x: 1, y: 2 }`
4. Arrays: `$ arr = [1, 2, 3]`
5. String operations: `toString`, concatenation

Commit each successful test expansion.

## Important Notes
- The bootstrap parser (parser.glyph) is already working and tested
- The interpreter just needs to WALK the AST the parser produces
- `typeOf()` is a built-in in GlyphLang (returns "int", "str", "bool", "object", "null")
- `toString()`, `toInt()`, `length()`, `print()` are all builtins
- Array concatenation uses `+`: `[1,2] + [3] = [1,2,3]`
- String concatenation uses `+`: `"hello" + " " + "world"`
