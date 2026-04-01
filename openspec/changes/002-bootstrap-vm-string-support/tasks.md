# Tasks: Bootstrap VM String Support

## 1. Add tagged value representation to VM
- [-] 1.1 Define value tag constants (INT_TAG=0, STRING_TAG=1, REF_TAG=2) and add parallel `tags []int` array to VM state, or refactor stack/locals to use a Value struct with tag field
- [-] 1.2 Update all existing opcode handlers to set INT_TAG when pushing values, and verify 230/230 existing tests still pass

## 2. Add string pool to VM
- [x] 2.1 Add `string_pool []string` to VM struct and `OP_LOAD_STRING` opcode that loads a string literal by index into the string pool and pushes a tagged STRING value onto the stack
- [x] 2.2 Update compiler to emit `OP_LOAD_STRING` for string literals with the string stored in a constant pool section of the bytecode

## 3. Implement OP_ADD type dispatch for strings
- [ ] 3.1 Modify OP_ADD handler: if both operands are INT_TAG, do integer add; if either is STRING_TAG, convert both to string representation and concatenate, pushing result as STRING_TAG
- [ ] 3.2 Write test: `a = "hello" + " " + "world"` compiles and executes to produce `"hello world"` via the VM
- [ ] 3.3 Write test: `a = "count: " + str(42)` produces `"count: 42"` (int-to-string coercion in concatenation)
