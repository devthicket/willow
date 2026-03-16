package mesh

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// DistortionGrid provides a grid mesh that can be deformed per-vertex.
type DistortionGrid struct {
	node_   *node.Node
	cols    int
	rows    int
	restPos []types.Vec2
}

// NewDistortionGrid creates a grid mesh over the given image. cols and rows
// define the number of cells (vertices = (cols+1) * (rows+1)).
func NewDistortionGrid(name string, img *ebiten.Image, cols, rows int) *DistortionGrid {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	var imgW, imgH float64
	if img != nil {
		b := img.Bounds()
		imgW = float64(b.Dx())
		imgH = float64(b.Dy())
	}

	vcols := cols + 1
	vrows := rows + 1
	numVerts := vcols * vrows
	numInds := cols * rows * 6

	verts := make([]ebiten.Vertex, numVerts)
	inds := make([]uint16, numInds)
	restPos := make([]types.Vec2, numVerts)

	cellW := imgW / float64(cols)
	cellH := imgH / float64(rows)

	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			x := float64(c) * cellW
			y := float64(r) * cellH
			u := float32(x)
			v := float32(y)
			verts[idx] = ebiten.Vertex{
				DstX: float32(x), DstY: float32(y),
				SrcX: u, SrcY: v,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			restPos[idx] = types.Vec2{X: x, Y: y}
		}
	}

	// Build indices.
	ii := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tl := uint16(r*vcols + c)
			tr := tl + 1
			bl := uint16((r+1)*vcols + c)
			br := bl + 1
			inds[ii+0] = tl
			inds[ii+1] = bl
			inds[ii+2] = tr
			inds[ii+3] = tr
			inds[ii+4] = bl
			inds[ii+5] = br
			ii += 6
		}
	}

	n := NewMeshFn(name, img, verts, inds)
	g := &DistortionGrid{node_: n, cols: cols, rows: rows, restPos: restPos}
	return g
}

// Node returns the underlying mesh node.
func (g *DistortionGrid) Node() *node.Node {
	return g.node_
}

// Cols returns the number of grid columns.
func (g *DistortionGrid) Cols() int { return g.cols }

// Rows returns the number of grid rows.
func (g *DistortionGrid) Rows() int { return g.rows }

// SetVertex offsets a single grid vertex by (dx, dy) from its rest position.
func (g *DistortionGrid) SetVertex(col, row int, dx, dy float64) {
	vcols := g.cols + 1
	idx := row*vcols + col
	rest := g.restPos[idx]
	g.node_.Mesh.Vertices[idx].DstX = float32(rest.X + dx)
	g.node_.Mesh.Vertices[idx].DstY = float32(rest.Y + dy)
	g.node_.Mesh.AabbDirty = true
}

// SetAllVertices calls fn for each vertex, passing (col, row, restX, restY).
// fn returns the (dx, dy) displacement from the rest position.
func (g *DistortionGrid) SetAllVertices(fn func(col, row int, restX, restY float64) (dx, dy float64)) {
	vcols := g.cols + 1
	vrows := g.rows + 1
	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			rest := g.restPos[idx]
			dx, dy := fn(c, r, rest.X, rest.Y)
			g.node_.Mesh.Vertices[idx].DstX = float32(rest.X + dx)
			g.node_.Mesh.Vertices[idx].DstY = float32(rest.Y + dy)
		}
	}
	g.node_.Mesh.AabbDirty = true
}

// Reset returns all vertices to their original positions.
func (g *DistortionGrid) Reset() {
	vcols := g.cols + 1
	vrows := g.rows + 1
	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			rest := g.restPos[idx]
			g.node_.Mesh.Vertices[idx].DstX = float32(rest.X)
			g.node_.Mesh.Vertices[idx].DstY = float32(rest.Y)
		}
	}
	g.node_.Mesh.AabbDirty = true
}
