# Bootstrap VM String Support

## Why

The bootstrap VM is currently int-only. Its heap is `int[]` and there is no way to distinguish between integer, string, and reference values. This blocks:

- String concatenation via `OP_ADD` (no type dispatch)
- Module import resolution (needs to pass file paths as strings)
- Self-hosting (the interpreter needs string manipulation to parse source code)
- For-in key,value pairs where keys are often strings

The VM must support tagged values so that a single value slot can hold either an int, a string reference, or a closure/struct reference.

## What Changes

1. **Value type tagging**: Introduce a tagged value representation in the VM. Each value is either `(int, INT_TAG)`, `(string_pool_index, STRING_TAG)`, or `(heap_ref, REF_TAG)`. This can be done with a parallel tag array or a struct-based value type.
2. **String pool**: Add a `string_pool []string` to the VM. String literals from bytecode get loaded into the pool. `OP_LOAD_CONST` for string operands pushes a tagged string reference.
3. **OP_ADD type dispatch**: When OP_ADD executes, check the tag of operands. If both are INT, do integer add. If either is STRING, concatenate (converting int to string representation if needed).
4. **String builtins**: Add `str_len`, `str_concat`, `str_char_at` as VM-accessible builtins or opcodes.

## Impact

- Unblocks module imports (strings needed for file paths)
- Unblocks self-hosting (interpreter needs string operations for lexing/parsing)
- Enables for-in over string-keyed collections
- VM bytecode format unchanged (operand encoding stays the same)
- Existing int-only tests must continue to pass after the change
