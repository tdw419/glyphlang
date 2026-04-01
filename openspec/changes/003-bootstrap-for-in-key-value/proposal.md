# Bootstrap For-In Key,Value Support

## Why

The ITER_NEXT opcode supports a `hasKey=1` flag that tells the VM to push both the key and value onto the stack during iteration. The compiler was recently fixed (ITER_NEXT operand width: 4 bytes -> 1 byte) but we have not verified that:

1. The compiler correctly emits ITER_NEXT with hasKey=1 for `for k, v in expr` syntax
2. The VM correctly pushes both key and value when hasKey=1
3. The full end-to-end pipeline works: parse for-in with two loop vars, compile to bytecode with correct ITER_NEXT, execute on VM with correct results

This is a prerequisite for self-hosting because the interpreter uses for-in loops to iterate over token arrays and AST node children with index tracking.

## What Changes

1. **Compiler verification**: Trace the bytecode output for a `for k, v in items` loop and confirm ITER_NEXT has hasKey=1 and that the compiler allocates two local slots for k and v.
2. **VM verification**: Trace VM execution of ITER_NEXT with hasKey=1 and confirm both key (int index) and value are pushed and stored in the correct local slots.
3. **End-to-end test**: Write a test that uses `for k, v in [10, 20, 30]` and asserts k=0,v=10 then k=1,v=20 then k=2,v=30.
4. **String-keyed iteration test**: If string support is in place (depends on change 002), test `for k, v in string_map` where keys are strings.

## Impact

- Validates the ITER_NEXT operand width fix from this session
- Confirms the for-in loop is production-ready for the self-hosting interpreter
- No breaking changes if currently broken (just untested)
