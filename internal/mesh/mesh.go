package mesh

import (
	"image/color"
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Function pointers wired by root to break dependency on constructors.
var (
	// NewMeshFn creates a mesh node. Set by root package.
	NewMeshFn func(name string, img *ebiten.Image, verts []ebiten.Vertex, inds []uint16) *node.Node
)

// --- White pixel singleton ---

var whitePixelImage *ebiten.Image

// EnsureWhitePixel returns a lazily-initialized 1x1 white pixel image.
func EnsureWhitePixel() *ebiten.Image {
	if whitePixelImage == nil {
		whitePixelImage = ebiten.NewImage(1, 1)
		whitePixelImage.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	return whitePixelImage
}

// --- Mesh utility functions ---

// TransformVertices applies an affine transform and color tint to src vertices,
// writing the result into dst. dst must be at least len(src) in length.
func TransformVertices(src, dst []ebiten.Vertex, transform [6]float64, tint types.Color) {
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	cr := float32(tint.R())
	cg := float32(tint.G())
	cb := float32(tint.B())
	ca := float32(tint.A())

	for i := range src {
		s := &src[i]
		ox := float64(s.DstX)
		oy := float64(s.DstY)
		dst[i] = ebiten.Vertex{
			DstX:   float32(a*ox + c*oy + tx),
			DstY:   float32(b*ox + d*oy + ty),
			SrcX:   s.SrcX,
			SrcY:   s.SrcY,
			ColorR: s.ColorR * cr * ca,
			ColorG: s.ColorG * cg * ca,
			ColorB: s.ColorB * cb * ca,
			ColorA: s.ColorA * ca,
		}
	}
}

// ComputeMeshAABB scans DstX/DstY of the given vertices and returns
// the axis-aligned bounding box in local space.
func ComputeMeshAABB(verts []ebiten.Vertex) types.Rect {
	if len(verts) == 0 {
		return types.Rect{}
	}
	minX := float64(verts[0].DstX)
	minY := float64(verts[0].DstY)
	maxX := minX
	maxY := minY
	for i := 1; i < len(verts); i++ {
		x := float64(verts[i].DstX)
		y := float64(verts[i].DstY)
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// EnsureTransformedVerts grows the node's transformedVerts buffer to fit
// len(n.Mesh.Vertices), using a high-water-mark strategy (never shrinks).
func EnsureTransformedVerts(n *node.Node) []ebiten.Vertex {
	need := len(n.Mesh.Vertices)
	if cap(n.Mesh.TransformedVerts) < need {
		n.Mesh.TransformedVerts = make([]ebiten.Vertex, need)
	}
	n.Mesh.TransformedVerts = n.Mesh.TransformedVerts[:need]
	return n.Mesh.TransformedVerts
}

// MeshWorldAABB computes the world-space AABB for a mesh node, accounting for
// the fact that mesh vertices may not start at origin (0,0).
func MeshWorldAABB(n *node.Node, transform [6]float64, worldAABBFn func([6]float64, float64, float64) types.Rect) types.Rect {
	RecomputeMeshAABB(n)
	aabb := n.Mesh.Aabb
	if aabb.Width == 0 && aabb.Height == 0 {
		return types.Rect{}
	}
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	shiftedTx := a*aabb.X + c*aabb.Y + tx
	shiftedTy := b*aabb.X + d*aabb.Y + ty
	shifted := [6]float64{a, b, c, d, shiftedTx, shiftedTy}
	return worldAABBFn(shifted, aabb.Width, aabb.Height)
}

// MeshWorldAABBOffset computes mesh world AABB by transforming the four corners.
func MeshWorldAABBOffset(n *node.Node, transform [6]float64) types.Rect {
	RecomputeMeshAABB(n)
	aabb := n.Mesh.Aabb
	if aabb.Width == 0 && aabb.Height == 0 {
		return types.Rect{}
	}
	wt := transform
	x0 := aabb.X
	y0 := aabb.Y
	x1 := aabb.X + aabb.Width
	y1 := aabb.Y + aabb.Height

	cx0 := wt[0]*x0 + wt[2]*y0 + wt[4]
	cy0 := wt[1]*x0 + wt[3]*y0 + wt[5]
	cx1 := wt[0]*x1 + wt[2]*y0 + wt[4]
	cy1 := wt[1]*x1 + wt[3]*y0 + wt[5]
	cx2 := wt[0]*x1 + wt[2]*y1 + wt[4]
	cy2 := wt[1]*x1 + wt[3]*y1 + wt[5]
	cx3 := wt[0]*x0 + wt[2]*y1 + wt[4]
	cy3 := wt[1]*x0 + wt[3]*y1 + wt[5]

	minX := math.Min(math.Min(cx0, cx1), math.Min(cx2, cx3))
	minY := math.Min(math.Min(cy0, cy1), math.Min(cy2, cy3))
	maxX := math.Max(math.Max(cx0, cx1), math.Max(cx2, cx3))
	maxY := math.Max(math.Max(cy0, cy1), math.Max(cy2, cy3))

	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// RecomputeMeshAABB recomputes the cached local-space AABB if dirty.
func RecomputeMeshAABB(n *node.Node) {
	if !n.Mesh.AabbDirty {
		return
	}
	n.Mesh.Aabb = ComputeMeshAABB(n.Mesh.Vertices)
	n.Mesh.AabbDirty = false
}
