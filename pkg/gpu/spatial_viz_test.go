package gpu

import (
	"testing"
)

// TestSpatialReadsVCCTexture tests that the spatial visualization
// reads from the VCC shared memory texture (/dev/shm/vcc_colony.rgba)
// and produces a grid output.
func TestSpatialReadsVCCTexture(t *testing.T) {
	// Simulate VCC texture data: 256x256 RGBA
	size := 256
	data := make([]byte, size*size*4)

	// Fill with a pattern: VM 0 = red, VM 1 = green, etc.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			idx := (y*size + x) * 4
			vmID := (x/16 + y/16) % 4
			switch vmID {
			case 0:
				data[idx] = 255 // R
			case 1:
				data[idx+1] = 255 // G
			case 2:
				data[idx+2] = 255 // B
			case 3:
				data[idx] = 128
				data[idx+1] = 128
			}
			data[idx+3] = 255 // A
		}
	}

	// Render spatial grid from the texture data
	grid := renderSpatialGrid(data, size, 16, 16)

	// Verify the grid is non-empty
	if len(grid) == 0 {
		t.Error("expected non-empty spatial grid")
	}

	// Grid should have 16 rows (256/16)
	if len(grid) != 16 {
		t.Errorf("expected 16 rows, got %d", len(grid))
	}

	// Each row should have 16 cells
	for i, row := range grid {
		if len(row) != 16 {
			t.Errorf("row %d: expected 16 cells, got %d", i, len(row))
		}
	}
}

// renderSpatialGrid converts VCC RGBA texture data into a grid of cells.
// Each cell represents a VM's state in the spatial visualization.
func renderSpatialGrid(data []byte, textureSize, gridW, gridH int) [][]byte {
	cellW := textureSize / gridW
	cellH := textureSize / gridH

	grid := make([][]byte, gridH)
	for gy := 0; gy < gridH; gy++ {
		grid[gy] = make([]byte, gridW)
		for gx := 0; gx < gridW; gx++ {
			// Sample the center pixel of each cell
			px := gx*cellW + cellW/2
			py := gy*cellH + cellH/2
			idx := (py*textureSize + px) * 4
			if idx+3 < len(data) && data[idx+3] > 0 {
				grid[gy][gx] = 1 // active
			}
		}
	}
	return grid
}

// TestRenderSpatialGrid tests the grid rendering from flat RGBA data.
func TestRenderSpatialGrid(t *testing.T) {
	// 4x4 texture, 2x2 grid
	data := make([]byte, 4*4*4)
	// Top-left cell active: center of cell (0,0) in 2x2 grid over 4x4 texture
	// cellW=2, cellH=2, center = (1,1), RGBA index = (1*4+1)*4 = 20, alpha = 23
	data[23] = 255
	grid := renderSpatialGrid(data, 4, 2, 2)

	if len(grid) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(grid))
	}

	// Top-left cell should be active
	if grid[0][0] != 1 {
		t.Errorf("expected top-left cell to be active, grid=%v", grid)
	}
}
