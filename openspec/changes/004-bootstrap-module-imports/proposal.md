# Bootstrap Module Import Resolution

## Why

The self-hosting milestone requires the bootstrap interpreter to load and interpret `.glyph` source files. Currently `import "./parser"` has no resolution mechanism. The interpreter needs to:

1. Resolve the import path to a file on disk
2. Read the file using a `readFile` builtin
3. Parse and evaluate the imported source
4. Make the exported symbols available in the importing module's scope

Without module imports, the interpreter cannot load parser.glyph, compiler.glyph, etc. as separate files -- they would all need to be concatenated into one monolithic file, which is not practical for self-hosting.

## What Changes

1. **readFile builtin**: Add a `readFile(path) -> string` builtin to the bootstrap interpreter that reads a file from disk and returns its contents as a string.
2. **Import resolution**: When the interpreter encounters `import "./parser"`, resolve the path relative to the current file's directory, call readFile, parse the source, and execute it to populate a module namespace.
3. **Module namespace**: Imported modules export their top-level `!` functions. The importing module accesses them as `parser.parse(source)` etc.
4. **Circular import guard**: Add a simple loaded-modules cache to prevent infinite recursion on circular imports.

## Impact

- Required for self-hosting (change 005)
- Enables the interpreter to be split across multiple files
- No change to existing single-file programs
- Depends on string support (change 002) for file path handling and source code storage
