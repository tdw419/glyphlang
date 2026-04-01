# Tasks: Bootstrap For-In Key,Value

## 1. Compiler bytecode verification for key,value for-in
- [ ] 1.1 Write a test program `for k, v in [10, 20, 30] { print(k) print(v) }`, compile it, dump bytecode, and verify ITER_NEXT operand has hasKey=1 flag set and two locals are allocated for k and v
- [ ] 1.2 If hasKey flag is wrong, fix the compiler emit logic for for-in with two loop variables

## 2. VM execution verification for ITER_NEXT hasKey=1
- [ ] 2.1 Trace VM execution of the key,value for-in test and verify both key and value are pushed to stack in correct order and stored in correct local variable slots
- [ ] 2.2 If VM does not handle hasKey=1 correctly, fix ITER_NEXT handler to push both values when flag is set

## 3. End-to-end integration test
- [ ] 3.1 Create `test_for_in_key_value.glyph` that iterates `for k, v in [10, 20, 30]`, collects results, and asserts k=0,v=10 through k=2,v=30
- [ ] 3.2 Run the test through full pipeline (parse -> compile -> vm_exec) and verify pass
