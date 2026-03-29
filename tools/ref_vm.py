#!/usr/bin/env python3
"""
Reference interpreter for GlyphLang bootstrap VM.
Used to verify bytecode behavior and trace execution.
"""
import sys

# Opcodes (matching bootstrap/vm.glyph)
OP_PUSH          = 1
OP_POP           = 2
OP_DUP           = 3
OP_SWAP          = 4
OP_DUP_SECOND    = 5
OP_ADD           = 16
OP_SUB           = 17
OP_MUL           = 18
OP_DIV           = 19
OP_MOD           = 20
OP_EQ            = 32
OP_NE            = 33
OP_LT            = 34
OP_GT            = 35
OP_GE            = 36
OP_LE            = 37
OP_AND           = 38
OP_OR            = 39
OP_NOT           = 40
OP_NEG           = 41
OP_LOAD_VAR      = 64
OP_STORE_VAR     = 65
OP_LOAD_FP       = 66
OP_JUMP          = 80
OP_JUMP_IF_FALSE = 81
OP_JUMP_IF_TRUE  = 82
OP_GET_INDEX     = 86
OP_SET_INDEX     = 87
OP_RETURN        = 97
OP_CALL          = 98
OP_BUILD_OBJECT  = 112
OP_GET_FIELD     = 113
OP_SET_FIELD     = 114
OP_DEF_FUNC      = 115
OP_BUILD_ARRAY   = 128
OP_HALT          = 255

OPCODE_NAMES = {
    OP_PUSH: "PUSH", OP_POP: "POP", OP_DUP: "DUP", OP_SWAP: "SWAP",
    OP_DUP_SECOND: "DUP_SECOND", OP_ADD: "ADD", OP_SUB: "SUB",
    OP_MUL: "MUL", OP_DIV: "DIV", OP_MOD: "MOD", OP_EQ: "EQ",
    OP_NE: "NE", OP_LT: "LT", OP_GT: "GT", OP_GE: "GE", OP_LE: "LE",
    OP_AND: "AND", OP_OR: "OR", OP_NOT: "NOT", OP_NEG: "NEG",
    OP_LOAD_VAR: "LOAD_VAR", OP_STORE_VAR: "STORE_VAR",
    OP_LOAD_FP: "LOAD_FP", OP_JUMP: "JUMP",
    OP_JUMP_IF_FALSE: "JUMP_IF_FALSE", OP_JUMP_IF_TRUE: "JUMP_IF_TRUE",
    OP_GET_INDEX: "GET_INDEX", OP_SET_INDEX: "SET_INDEX",
    OP_RETURN: "RETURN", OP_CALL: "CALL",
    OP_BUILD_OBJECT: "BUILD_OBJECT", OP_GET_FIELD: "GET_FIELD",
    OP_SET_FIELD: "SET_FIELD", OP_DEF_FUNC: "DEF_FUNC",
    OP_BUILD_ARRAY: "BUILD_ARRAY", OP_HALT: "HALT",
}

HEAP_BASE = 2000000


def read_addr(code, pc):
    """Read 4-byte little-endian address."""
    return (code[pc] | (code[pc+1] << 8) |
            (code[pc+2] << 16) | (code[pc+3] << 24))


class VM:
    def __init__(self, trace=False):
        self.stack = []
        self.pc = 0
        self.fp = 0
        self.halted = False
        self.heap = []
        self.trace = trace
        self.step_count = 0

    def push(self, val):
        self.stack.append(val)

    def pop(self):
        if not self.stack:
            raise RuntimeError(f"Stack underflow! pc={self.pc} fp={self.fp}")
        return self.stack.pop()

    def peek(self):
        if not self.stack:
            raise RuntimeError("Stack underflow in peek!")
        return self.stack[-1]

    def heap_alloc(self, data):
        """Allocate on heap, return negative ref."""
        offset = len(self.heap)
        self.heap.extend(data)
        return -(HEAP_BASE + offset)

    def heap_read(self, ref, idx):
        """Read from heap via ref."""
        offset = HEAP_BASE + ref  # ref is negative
        return self.heap[offset + idx]

    def heap_write(self, ref, idx, val):
        """Write to heap via ref."""
        offset = HEAP_BASE + ref  # ref is negative
        self.heap[offset + idx] = val

    def log(self, msg):
        if self.trace:
            s = self.stack[-5:] if len(self.stack) > 5 else self.stack[:]
            print(f"  [{self.step_count:4d}] pc={self.pc:3d} fp={self.fp:2d} | {msg:40s} | stack={s} (len={len(self.stack)})")

    def step(self, code, constants):
        if self.halted or self.pc >= len(code):
            return False

        op = code[self.pc]
        name = OPCODE_NAMES.get(op, f"UNK({op})")
        self.step_count += 1

        if op == OP_PUSH:
            idx = code[self.pc + 1]
            val = constants[idx]
            self.log(f"PUSH const[{idx}]={val}")
            self.push(val)
            self.pc += 2

        elif op == OP_POP:
            v = self.pop()
            self.log(f"POP ({v})")
            self.pc += 1

        elif op == OP_DUP:
            v = self.peek()
            self.log(f"DUP ({v})")
            self.push(v)
            self.pc += 1

        elif op == OP_SWAP:
            a = self.pop()
            b = self.pop()
            self.push(a)
            self.push(b)
            self.log(f"SWAP")
            self.pc += 1

        elif op == OP_DUP_SECOND:
            v = self.stack[-2]
            self.log(f"DUP_SECOND ({v})")
            self.push(v)
            self.pc += 1

        elif op == OP_ADD:
            b = self.pop()
            a = self.pop()
            self.push(a + b)
            self.log(f"ADD {a}+{b}={a+b}")
            self.pc += 1

        elif op == OP_SUB:
            b = self.pop()
            a = self.pop()
            self.push(a - b)
            self.log(f"SUB {a}-{b}={a-b}")
            self.pc += 1

        elif op == OP_MUL:
            b = self.pop()
            a = self.pop()
            self.push(a * b)
            self.log(f"MUL {a}*b={a*b}")
            self.pc += 1

        elif op == OP_DIV:
            b = self.pop()
            a = self.pop()
            self.push(a // b)
            self.log(f"DIV {a}/{b}")
            self.pc += 1

        elif op == OP_MOD:
            b = self.pop()
            a = self.pop()
            self.push(a % b)
            self.log(f"MOD {a}%{b}")
            self.pc += 1

        elif op == OP_EQ:
            b = self.pop()
            a = self.pop()
            self.push(1 if a == b else 0)
            self.log(f"EQ {a}=={b}")
            self.pc += 1

        elif op == OP_NE:
            b = self.pop()
            a = self.pop()
            self.push(1 if a != b else 0)
            self.log(f"NE {a}!={b}")
            self.pc += 1

        elif op == OP_LT:
            b = self.pop()
            a = self.pop()
            self.push(1 if a < b else 0)
            self.log(f"LT {a}<{b}")
            self.pc += 1

        elif op == OP_GT:
            b = self.pop()
            a = self.pop()
            self.push(1 if a > b else 0)
            self.log(f"GT {a}>{b}")
            self.pc += 1

        elif op == OP_GE:
            b = self.pop()
            a = self.pop()
            self.push(1 if a >= b else 0)
            self.log(f"GE {a}>={b}")
            self.pc += 1

        elif op == OP_LE:
            b = self.pop()
            a = self.pop()
            self.push(1 if a <= b else 0)
            self.log(f"LE {a}<={b}")
            self.pc += 1

        elif op == OP_AND:
            b = self.pop()
            a = self.pop()
            self.push(1 if (a != 0 and b != 0) else 0)
            self.log(f"AND {a}&&{b}")
            self.pc += 1

        elif op == OP_OR:
            b = self.pop()
            a = self.pop()
            self.push(1 if (a != 0 or b != 0) else 0)
            self.log(f"OR {a}||{b}")
            self.pc += 1

        elif op == OP_NOT:
            a = self.pop()
            self.push(1 if a == 0 else 0)
            self.log(f"NOT {a}")
            self.pc += 1

        elif op == OP_NEG:
            a = self.pop()
            self.push(-a)
            self.log(f"NEG {a}")
            self.pc += 1

        elif op == OP_LOAD_VAR:
            idx = code[self.pc + 1]
            self.push(constants[idx])
            self.log(f"LOAD_VAR [{idx}]={constants[idx]}")
            self.pc += 2

        elif op == OP_STORE_VAR:
            self.pop()
            self.log(f"STORE_VAR")
            self.pc += 1

        elif op == OP_LOAD_FP:
            offset = code[self.pc + 1]
            idx = self.fp + offset
            if idx < 0 or idx >= len(self.stack):
                self.log(f"LOAD_FP FP{self.fp:+d}={idx} OVERFLOW!")
                self.halted = True
                return False
            val = self.stack[idx]
            self.push(val)
            self.log(f"LOAD_FP fp={self.fp} off={offset} -> stack[{idx}]={val}")
            self.pc += 2

        elif op == OP_JUMP:
            target = read_addr(code, self.pc + 1)
            self.log(f"JUMP {target}")
            self.pc = target

        elif op == OP_JUMP_IF_FALSE:
            target = read_addr(code, self.pc + 1)
            cond = self.pop()
            if cond == 0:
                self.log(f"JUMP_IF_FALSE {cond} -> JUMP {target}")
                self.pc = target
            else:
                self.log(f"JUMP_IF_FALSE {cond} -> fallthrough")
                self.pc += 5

        elif op == OP_JUMP_IF_TRUE:
            target = read_addr(code, self.pc + 1)
            cond = self.pop()
            if cond != 0:
                self.log(f"JUMP_IF_TRUE {cond} -> JUMP {target}")
                self.pc = target
            else:
                self.log(f"JUMP_IF_TRUE {cond} -> fallthrough")
                self.pc += 5

        elif op == OP_CALL:
            target = read_addr(code, self.pc + 1)
            return_addr = self.pc + 5
            stack_base = len(self.stack)
            self.push(self.fp)         # old_fp
            self.push(return_addr)     # return_addr
            self.push(stack_base)      # stack_base
            self.fp = len(self.stack) - 3  # FP points to old_fp
            self.log(f"CALL target={target} ret={return_addr} sb={stack_base}")
            self.pc = target

        elif op == OP_RETURN:
            # Frame layout at FP: [old_fp, return_addr, stack_base]
            # Read frame values by position (FP-relative), not by popping.
            # This correctly skips any leftover temps between result and frame.
            result = self.pop()
            if self.trace:
                print(f"  >>> RETURN: fp={self.fp}, stack_len={len(self.stack)}, stack={self.stack}")
            old_fp = self.stack[self.fp]
            return_addr = self.stack[self.fp + 1]
            stack_base = self.stack[self.fp + 2]
            self.log(f"RETURN result={result} ret={return_addr} ofp={old_fp} sb={stack_base}")
            # Trim stack to stack_base (removes leftover args/temps)
            self.stack = self.stack[:stack_base]
            self.fp = old_fp
            self.pc = return_addr
            self.push(result)

        elif op == OP_BUILD_ARRAY:
            count = code[self.pc + 1]
            elems = []
            for _ in range(count):
                elems.insert(0, self.pop())
            ref = self.heap_alloc([count] + elems)
            self.push(ref)
            self.log(f"BUILD_ARRAY [{','.join(str(e) for e in elems)}] ref={ref}")
            self.pc += 2

        elif op == OP_BUILD_OBJECT:
            field_count = code[self.pc + 1]
            fields = []
            for _ in range(field_count):
                val = self.pop()
                key = self.pop()
                fields.insert(0, (key, val))
            data = [field_count * 2]
            for key, val in fields:
                data.extend([key, val])
            ref = self.heap_alloc(data)
            self.push(ref)
            self.log(f"BUILD_OBJECT ref={ref}")
            self.pc += 2

        elif op == OP_GET_FIELD:
            key = self.pop()
            ref = self.pop()
            count = self.heap_read(ref, 0)
            found = 0
            for i in range(count):
                k = self.heap_read(ref, 1 + i * 2)
                v = self.heap_read(ref, 1 + i * 2 + 1)
                if k == key:
                    self.push(v)
                    found = 1
                    break
            if not found:
                self.push(0)
            self.log(f"GET_FIELD key={key} ref={ref}")
            self.pc += 1

        elif op == OP_SET_FIELD:
            val = self.pop()
            key = self.pop()
            ref = self.pop()
            count = self.heap_read(ref, 0)
            for i in range(count):
                k = self.heap_read(ref, 1 + i * 2)
                if k == key:
                    self.heap_write(ref, 1 + i * 2 + 1, val)
                    break
            self.push(ref)
            self.log(f"SET_FIELD key={key} val={val}")
            self.pc += 1

        elif op == OP_DEF_FUNC:
            self.pc += 2
            self.log(f"DEF_FUNC (skipped)")

        elif op == OP_GET_INDEX:
            idx = self.pop()
            ref = self.pop()
            count = self.heap_read(ref, 0)
            if 0 <= idx < count:
                val = self.heap_read(ref, 1 + idx)
            else:
                val = 0
            self.push(val)
            self.log(f"GET_INDEX [{idx}] = {val}")
            self.pc += 1

        elif op == OP_SET_INDEX:
            val = self.pop()
            idx = self.pop()
            ref = self.pop()
            count = self.heap_read(ref, 0)
            if 0 <= idx < count:
                self.heap_write(ref, 1 + idx, val)
            self.push(ref)
            self.log(f"SET_INDEX [{idx}] = {val}")
            self.pc += 1

        elif op == OP_HALT:
            self.log(f"HALT")
            self.halted = True
            return False

        else:
            self.log(f"UNKNOWN OP {op}")
            self.halted = True
            return False

        return True

    def run(self, code, constants):
        while self.step(code, constants):
            if self.step_count > 10000:
                print("STEP LIMIT REACHED")
                break
        if self.stack:
            return self.stack[-1]
        return 0


def test_recursive_factorial():
    """Exact bytecode from bootstrap/vm.glyph test_recursive_factorial."""
    code = [
        OP_PUSH, 0,           # 0-1: PUSH 5
        OP_CALL, 9, 0, 0, 0,  # 2-6: CALL factorial at offset 9
        OP_HALT,              # 7: HALT
        OP_POP,               # 8: (padding)

        # factorial function (offset 9):
        OP_LOAD_FP, -1,       # 9: Load n from FP-1
        OP_PUSH, 1,           # 10: PUSH 1
        OP_LE,                # 11: n <= 1?
        OP_JUMP_IF_FALSE, 22, 0, 0, 0,  # 12-16: If false, jump to recursive case

        # Base case (offset 17):
        OP_PUSH, 1,           # 17: Return 1
        OP_RETURN,            # 18: RETURN

        # Recursive case (offset 19):  -- WAIT, the original has offset 22
        # Let me re-check the bytecode...
    ]

    # Actually let me re-read the exact offsets from the source
    # The test says:
    # OP_JUMP_IF_FALSE, 22, 0, 0, 0,  # 12-16: If false, jump to recursive case
    # That's 5 bytes at offsets 12,13,14,15,16. Next instruction is at offset 17.
    # OP_PUSH, 1 at offset 17-18
    # OP_RETURN at offset 19
    # Recursive case starts at offset 20? No, the comment says 22.

    # Let me recount carefully:
    # 0: OP_PUSH
    # 1: 0
    # 2: OP_CALL
    # 3: 9
    # 4: 0
    # 5: 0
    # 6: 0
    # 7: OP_HALT
    # 8: OP_POP
    # 9: OP_LOAD_FP
    # 10: -1
    # 11: OP_PUSH
    # 12: 1
    # 13: OP_LE
    # 14: OP_JUMP_IF_FALSE
    # 15: 22
    # 16: 0
    # 17: 0
    # 18: 0
    # 19: OP_PUSH  (base case)
    # 20: 1
    # 21: OP_RETURN
    # 22: OP_LOAD_FP  (recursive case)
    # ...

    # WAIT. The source says OP_PUSH 1 is at offset 10, but offset 10 is the operand of LOAD_FP.
    # I need to recount from the source comments more carefully.
    pass


def test_recursive_factorial_v2():
    """Reconstructed bytecode with correct offsets."""
    code = [
        # Offset 0: Entry
        OP_PUSH, 0,                   # 0-1: PUSH constants[0] = 5
        OP_CALL, 9, 0, 0, 0,          # 2-6: CALL factorial at offset 9
        OP_HALT,                      # 7: HALT
        OP_POP,                       # 8: padding

        # Offset 9: factorial function start
        OP_LOAD_FP, 0xFF,             # 9-10: LOAD_FP -1 (0xFF = -1 signed? or just use -1)

        # Hmm, GlyphLang uses signed ints in the array. In Python let me just use -1.
    ]

    # Let me just use the exact array from the source, Python handles negative ints fine
    code = [
        OP_PUSH, 0,           # 0-1
        OP_CALL, 9, 0, 0, 0,  # 2-6
        OP_HALT,              # 7
        OP_POP,               # 8

        # factorial (offset 9):
        OP_LOAD_FP, -1,       # 9-10: Load n from FP-1
        OP_PUSH, 1,           # 11-12: PUSH 1
        OP_LE,                # 13: n <= 1?
        OP_JUMP_IF_FALSE, 22, 0, 0, 0,  # 14-18: If false, jump to 22

        # Base case (offset 19):
        OP_PUSH, 1,           # 19-20: Return 1
        OP_RETURN,            # 21: RETURN

        # Recursive case (offset 22):
        OP_LOAD_FP, -1,       # 22-23: Load n
        OP_PUSH, 1,           # 24-25: Push 1
        OP_SUB,               # 26: n - 1
        OP_CALL, 9, 0, 0, 0,  # 27-31: CALL factorial(n-1)
        OP_LOAD_FP, -1,       # 32-33: Load n again
        OP_MUL,               # 34: n * factorial(n-1)
        OP_RETURN,            # 35: RETURN result
    ]
    constants = [5, 1]

    print("=" * 70)
    print("RECURSIVE FACTORIAL TEST (5! = 120)")
    print("=" * 70)
    vm = VM(trace=True)
    result = vm.run(code, constants)
    print(f"\nResult: {result}")
    print(f"Steps: {vm.step_count}")
    print(f"Expected: 120")
    print(f"PASS: {result == 120}")


def test_simple_call_return():
    """From test_simple_call_return."""
    code = [
        OP_CALL, 6, 0, 0, 0,  # 0-4: CALL target=6
        OP_HALT,               # 5
        OP_PUSH, 0,            # 6-7: PUSH 42
        OP_RETURN,             # 8
    ]
    constants = [42]

    print("=" * 70)
    print("SIMPLE CALL/RETURN TEST (expect 42)")
    print("=" * 70)
    vm = VM(trace=True)
    result = vm.run(code, constants)
    print(f"\nResult: {result}")
    print(f"PASS: {result == 42}")


if __name__ == "__main__":
    test_simple_call_return()
    print("\n")
    test_recursive_factorial_v2()
