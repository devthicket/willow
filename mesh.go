package willow

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// transformVertices applies an affine transform and color tint to src vertices,
// writing the result into dst. dst must be at least len(src) in length.
//
// Matrix layout: [0]=a, [1]=b, [2]=c, [3]=d, [4]=tx, [5]=ty
// newX = a*x + c*y + tx, newY = b*x + d*y + ty
//
// Color components are multiplied (vertex color * tint). The tint's alpha
// already has worldAlpha baked in, so no double-alpha correction is needed.
func transformVertices(src, dst []ebiten.Vertex, transform [6]float64, tint Color) {
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	cr := float32(tint.R)
	cg := float32(tint.G)
	cb := float32(tint.B)
	ca := float32(tint.A)

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

// computeMeshAABB scans DstX/DstY of the given vertices and returns
// the axis-aligned bounding box in local space.
func computeMeshAABB(verts []ebiten.Vertex) Rect {
	if len(verts) == 0 {
		return Rect{}
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
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// ensureTransformedVerts grows the node's transformedVerts buffer to fit
// len(n.Vertices), using a high-water-mark strategy (never shrinks).
// Returns the resliced buffer.
func ensureTransformedVerts(n *Node) []ebiten.Vertex {
	need := len(n.Vertices)
	if cap(n.transformedVerts) < need {
		n.transformedVerts = make([]ebiten.Vertex, need)
	}
	n.transformedVerts = n.transformedVerts[:need]
	return n.transformedVerts
}

// InvalidateMeshAABB marks the mesh's cached AABB as needing recomputation.
// Call this after modifying Vertices.
func (n *Node) InvalidateMeshAABB() {
	n.meshAABBDirty = true
}

// recomputeMeshAABB recomputes the cached local-space AABB if dirty.
func (n *Node) recomputeMeshAABB() {
	if !n.meshAABBDirty {
		return
	}
	n.meshAABB = computeMeshAABB(n.Vertices)
	n.meshAABBDirty = false
}

// --- White pixel singleton (no sync.Once  -  willow is single-threaded) ---

var whitePixelImage *ebiten.Image

// ensureWhitePixel returns a lazily-initialized 1x1 white pixel image.
// Used by untextured polygon meshes.
func ensureWhitePixel() *ebiten.Image {
	if whitePixelImage == nil {
		whitePixelImage = ebiten.NewImage(1, 1)
		whitePixelImage.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	return whitePixelImage
}

// meshWorldAABB computes the world-space AABB for a mesh node, accounting for
// the fact that mesh vertices may not start at origin (0,0). It shifts the
// transform by the AABB origin before computing worldAABB.
func meshWorldAABB(n *Node, transform [6]float64) Rect {
	n.recomputeMeshAABB()
	aabb := n.meshAABB
	if aabb.Width == 0 && aabb.Height == 0 {
		return Rect{}
	}
	// Shift transform origin to AABB min corner.
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	shiftedTx := a*aabb.X + c*aabb.Y + tx
	shiftedTy := b*aabb.X + d*aabb.Y + ty
	shifted := [6]float64{a, b, c, d, shiftedTx, shiftedTy}
	return worldAABB(shifted, aabb.Width, aabb.Height)
}

// meshWorldAABBOffset is a helper for computing mesh world AABB using the
// camera culling path. The transform parameter is the coordinate-space
// transform to use (e.g. view*world for culling in screen space).
func meshWorldAABBOffset(n *Node, transform [6]float64) Rect {
	n.recomputeMeshAABB()
	aabb := n.meshAABB
	if aabb.Width == 0 && aabb.Height == 0 {
		return Rect{}
	}
	wt := transform
	// Transform the four corners of the local AABB.
	x0 := aabb.X
	y0 := aabb.Y
	x1 := aabb.X + aabb.Width
	y1 := aabb.Y + aabb.Height

	// Transform all four corners.
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

	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}
