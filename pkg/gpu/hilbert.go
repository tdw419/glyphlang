package gpu

// Hilbert curve mapping for spatial bytecode visualization.
// Maps 1D bytecode offsets to 2D grid coordinates using Hilbert
// space-filling curves, preserving locality: adjacent instructions
// remain spatially adjacent on the grid.

// HilbertPoint represents a 2D coordinate on the spatial grid.
type HilbertPoint struct {
	X int
	Y int
}

// HilbertMapper maps bytecode to a 2D spatial grid via Hilbert curves.
type HilbertMapper struct {
	Order int // Hilbert curve order (grid is 2^order x 2^order)
	Size  int // Grid side length (2^order)
}

// NewHilbertMapper creates a mapper with sufficient resolution for the given bytecode length.
func NewHilbertMapper(bytecodeLen int) *HilbertMapper {
	order := 1
	for (1 << (2 * order)) < bytecodeLen {
		order++
	}
	return &HilbertMapper{
		Order: order,
		Size:  1 << order,
	}
}

// Map converts a 1D bytecode offset to a 2D Hilbert grid coordinate.
func (h *HilbertMapper) Map(index int) HilbertPoint {
	x, y := d2xy(h.Size, index)
	return HilbertPoint{X: x, Y: y}
}

// Unmap converts a 2D grid coordinate back to a 1D bytecode offset.
func (h *HilbertMapper) Unmap(p HilbertPoint) int {
	return xy2d(h.Size, p.X, p.Y)
}

// MapAll converts all bytecode offsets to grid coordinates.
func (h *HilbertMapper) MapAll(length int) []HilbertPoint {
	points := make([]HilbertPoint, length)
	for i := 0; i < length; i++ {
		points[i] = h.Map(i)
	}
	return points
}

// GridSize returns the side length of the spatial grid.
func (h *HilbertMapper) GridSize() int {
	return h.Size
}

// TotalCells returns the total number of cells in the grid.
func (h *HilbertMapper) TotalCells() int {
	return h.Size * h.Size
}

// d2xy converts a distance along the Hilbert curve to (x, y) coordinates.
// Standard algorithm from Wikipedia / "Hacker's Delight".
func d2xy(n, d int) (int, int) {
	var rx, ry, x, y int
	t := d
	for s := 1; s < n; s *= 2 {
		rx = 1 & (t / 2)
		ry = 1 & (t ^ rx)
		x, y = rot(s, x, y, rx, ry)
		x += s * rx
		y += s * ry
		t /= 4
	}
	return x, y
}

// xy2d converts (x, y) coordinates to a distance along the Hilbert curve.
func xy2d(n, x, y int) int {
	var rx, ry, d int
	for s := n / 2; s > 0; s /= 2 {
		if (x & s) > 0 {
			rx = 1
		} else {
			rx = 0
		}
		if (y & s) > 0 {
			ry = 1
		} else {
			ry = 0
		}
		d += s * s * ((3 * rx) ^ ry)
		x, y = rot(s, x, y, rx, ry)
	}
	return d
}

// rot rotates/flips a quadrant appropriately for the Hilbert curve.
func rot(n, x, y, rx, ry int) (int, int) {
	if ry == 0 {
		if rx == 1 {
			x = n - 1 - x
			y = n - 1 - y
		}
		x, y = y, x
	}
	return x, y
}

// SpatialGrid represents bytecode mapped onto a 2D grid for visualization.
type SpatialGrid struct {
	Width    int
	Height   int
	Cells    []GridCell
	Mapper   *HilbertMapper
}

// GridCell represents one cell in the spatial grid.
type GridCell struct {
	Offset int    // Bytecode offset (-1 if empty)
	Opcode byte   // Opcode at this position
	IsCode bool   // true if this is an instruction (not constant/operand)
	Label  string // Human-readable label (opcode name)
}

// NewSpatialGrid creates a spatial grid from bytecode.
func NewSpatialGrid(bytecode []byte, codeOffset int) *SpatialGrid {
	codeLen := len(bytecode) - codeOffset
	if codeLen <= 0 {
		return &SpatialGrid{Width: 1, Height: 1, Cells: []GridCell{{Offset: -1}}}
	}

	mapper := NewHilbertMapper(codeLen)
	size := mapper.GridSize()
	cells := make([]GridCell, size*size)

	// Initialize all cells as empty
	for i := range cells {
		cells[i].Offset = -1
	}

	// Map bytecode instructions to grid positions
	for i := 0; i < codeLen && codeOffset+i < len(bytecode); i++ {
		p := mapper.Map(i)
		idx := p.Y*size + p.X
		if idx < len(cells) {
			cells[idx] = GridCell{
				Offset: i,
				Opcode: bytecode[codeOffset+i],
				IsCode: true,
				Label:  opcodeName(bytecode[codeOffset+i]),
			}
		}
	}

	return &SpatialGrid{
		Width:  size,
		Height: size,
		Cells:  cells,
		Mapper: mapper,
	}
}

// CellAt returns the cell at grid position (x, y).
func (g *SpatialGrid) CellAt(x, y int) GridCell {
	if x < 0 || x >= g.Width || y < 0 || y >= g.Height {
		return GridCell{Offset: -1}
	}
	return g.Cells[y*g.Width+x]
}

// RenderText produces a text visualization of the spatial grid.
func (g *SpatialGrid) RenderText() string {
	var buf []byte
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			cell := g.CellAt(x, y)
			if cell.Offset == -1 {
				buf = append(buf, '.', ' ')
			} else if cell.IsCode {
				b := cell.Opcode
				// Use hex character for the opcode
				buf = append(buf, hexChar(b>>4), hexChar(b&0x0f))
			} else {
				buf = append(buf, '.', '.')
			}
			buf = append(buf, ' ')
		}
		buf = append(buf, '\n')
	}
	return string(buf)
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

func opcodeName(op byte) string {
	names := map[byte]string{
		0x01: "PUSH", 0x02: "POP",
		0x10: "ADD", 0x11: "SUB", 0x12: "MUL", 0x13: "DIV", 0x14: "MOD",
		0x20: "EQ", 0x21: "NE", 0x22: "LT", 0x23: "GT", 0x24: "GE", 0x25: "LE",
		0x26: "AND", 0x27: "OR", 0x28: "NOT", 0x29: "NEG",
		0x40: "LOAD", 0x41: "STORE",
		0x50: "JMP", 0x51: "JF", 0x52: "JT",
		0x56: "GETI", 0x57: "SETI",
		0x61: "RET", 0x62: "CALL",
		0x70: "OBJ", 0x71: "GETF", 0x72: "SETF", 0x73: "DEFF",
		0x80: "ARR",
		0x90: "HTTP",
		0xFF: "HALT",
	}
	if name, ok := names[op]; ok {
		return name
	}
	return "?"
}
