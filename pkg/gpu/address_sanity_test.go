package gpu

import (
	"testing"
)

func TestAddressSanity(t *testing.T) {
	constants := []interface{}{42, 2}
	code := append(pushConst(0), pushConst(1)...)
	code = append(code, 0xC1)
	code = append(code, 0xFF, 0x00, 0x00, 0x00)
	
	bc := buildBytecode(constants, code)
	conf, _ := parseBytecodeLayout(bc)
	t.Logf("CodeOffset: %d, CodeLen: %d, Total: %d", conf.CodeOffset, conf.BytecodeLen, len(bc))
	t.Logf("PC of MUTATOR: %d", conf.CodeOffset + 10)
}
