package gpu

import (
	"testing"
)

func TestHilbertRoundTrip(t *testing.T) {
	// For various grid sizes, map→unmap should return the original index
	for order := 1; order <= 5; order++ {
		size := 1 << order
		total := size * size
		mapper := &HilbertMapper{Order: order, Size: size}

		for i := 0; i < total; i++ {
			p := mapper.Map(i)
			got := mapper.Unmap(p)
			if got != i {
				t.Fatalf("order=%d: Map(%d)=(%d,%d), Unmap=(%d) != %d",
					order, i, p.X, p.Y, got, i)
			}
		}
	}
}

func TestHilbertLocality(t *testing.T) {
	// Adjacent indices should map to adjacent grid cells (Manhattan distance <= 1)
	mapper := NewHilbertMapper(64) // 8x8 grid

	for i := 0; i < 63; i++ {
		p1 := mapper.Map(i)
		p2 := mapper.Map(i + 1)
		dx := abs(p2.X - p1.X)
		dy := abs(p2.Y - p1.Y)
		manhattan := dx + dy
		if manhattan > 1 {
			t.Fatalf("non-adjacent: Map(%d)=(%d,%d), Map(%d)=(%d,%d), dist=%d",
				i, p1.X, p1.Y, i+1, p2.X, p2.Y, manhattan)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestNewHilbertMapper(t *testing.T) {
	tests := []struct {
		length    int
		wantOrder int
		wantSize  int
	}{
		{1, 1, 2},
		{4, 1, 2},
		{5, 2, 4},
		{16, 2, 4},
		{17, 3, 8},
		{256, 4, 16},
		{1000, 5, 32},
	}

	for _, tt := range tests {
		m := NewHilbertMapper(tt.length)
		if m.Order != tt.wantOrder {
			t.Errorf("NewHilbertMapper(%d): order=%d, want %d", tt.length, m.Order, tt.wantOrder)
		}
		if m.Size != tt.wantSize {
			t.Errorf("NewHilbertMapper(%d): size=%d, want %d", tt.length, m.Size, tt.wantSize)
		}
	}
}

func TestSpatialGrid(t *testing.T) {
	// Simple bytecode: GLYP + 2 int constants + PUSH 0, PUSH 1, ADD, HALT
	constants := []interface{}{10, 5}
	var code []byte
	code = append(code, pushConst(0)...)
	code = append(code, pushConst(1)...)
	code = append(code, 0x10) // ADD
	code = append(code, 0xFF) // HALT

	bytecode := buildBytecode(constants, code)

	// Find code offset
	config, err := parseBytecodeLayout(bytecode)
	if err != nil {
		t.Fatal(err)
	}

	grid := NewSpatialGrid(bytecode, int(config.CodeOffset))
	if grid.Width == 0 || grid.Height == 0 {
		t.Fatal("grid has zero dimensions")
	}

	// Verify the grid has content
	text := grid.RenderText()
	if len(text) == 0 {
		t.Fatal("empty grid render")
	}
	t.Logf("Grid %dx%d:\n%s", grid.Width, grid.Height, text)

	// Verify first instruction is at Hilbert position 0
	p := grid.Mapper.Map(0)
	cell := grid.CellAt(p.X, p.Y)
	if cell.Opcode != 0x01 { // OP_PUSH
		t.Fatalf("first cell should be PUSH (0x01), got 0x%02x", cell.Opcode)
	}
}

func TestSpatialGridOpcodeName(t *testing.T) {
	tests := map[byte]string{
		0x01: "PUSH", 0x10: "ADD", 0x11: "SUB",
		0x41: "STORE", 0xFF: "HALT",
	}
	for op, want := range tests {
		got := opcodeName(op)
		if got != want {
			t.Errorf("opcodeName(0x%02x) = %q, want %q", op, got, want)
		}
	}
}

func BenchmarkHilbertMap(b *testing.B) {
	mapper := NewHilbertMapper(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.Map(i % 10000)
	}
}
