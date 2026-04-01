# Tasks: Bootstrap VM Self-Hosting Milestone

## 1. Verify dynamic eval (nested interpretation)
- [x] 1.1 Test that `eval_source("print(1 + 2)")` works when called from within interpreted .glyph code -- the interpreter calls its own eval function recursively
- [x] 1.2 Test nested eval: `eval_source("eval_source(\"print(42)\")")` -- two levels of interpretation

## 2. Run interpreter.glyph on a simple test program
- [ ] 2.1 Ensure interpreter.glyph can `import "./lexer"` and `import "./parser"` via module resolution, then read and execute a simple test program (e.g., `print(2 + 3)`) producing output `5`
- [ ] 2.2 Debug and fix any issues found in string handling, for-in loops, or function calls during the interpretation of the test program

## 3. Meta-circular test: interpreter interprets itself interpreting a program
- [ ] 3.1 Run `interpreter.glyph` interpreting itself interpreting `test_fibonacci.glyph` where fibonacci(10) produces 55. Verify the final output is correct.
- [ ] 3.2 Document performance metrics: time to execute, memory usage, any timeouts or limits hit during the 3-level interpretation stack
