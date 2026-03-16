package camera

import (
	"math"

	"github.com/devthicket/willow/internal/mesh"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

// WorldAABB computes the axis-aligned bounding box for a rectangle of size (w, h)
// transformed by the given affine matrix. Zero allocations.
func WorldAABB(transform [6]float64, w, h float64) types.Rect {
	a, b, cc, d, tx, ty := transform[0], transform[2], transform[1], transform[3], transform[4], transform[5]

	// Transform four corners: (0,0), (w,0), (w,h), (0,h)
	x0, y0 := tx, ty
	x1, y1 := a*w+tx, cc*w+ty
	x2, y2 := a*w+b*h+tx, cc*w+d*h+ty
	x3, y3 := b*h+tx, d*h+ty

	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))

	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// NodeDimensions returns the width and height used for AABB culling.
func NodeDimensions(n *node.Node) (w, h float64) {
	switch n.Type {
	case types.NodeTypeSprite:
		if n.CustomImage_ != nil {
			b := n.CustomImage_.Bounds()
			return float64(b.Dx()), float64(b.Dy())
		}
		return float64(n.TextureRegion_.OriginalW), float64(n.TextureRegion_.OriginalH)
	case types.NodeTypeMesh:
		mesh.RecomputeMeshAABB(n)
		return n.Mesh.Aabb.Width, n.Mesh.Aabb.Height
	case types.NodeTypeParticleEmitter:
		if n.Emitter != nil {
			r := &n.Emitter.Config.Region
			return float64(r.OriginalW), float64(r.OriginalH)
		}
		return 0, 0
	case types.NodeTypeText:
		if n.TextBlock != nil {
			n.TextBlock.Layout() // ensure measured dims are current
			return n.TextBlock.MeasuredW, n.TextBlock.MeasuredH
		}
		return 0, 0
	default:
		return 0, 0
	}
}

// ShouldCull returns true if the node should be skipped during rendering.
// viewWorld is the coordinate-space transform used for AABB computation
// (typically view * worldTransform for screen-space culling).
// Containers are never culled. Text nodes are culled when measured dimensions are available.
func ShouldCull(n *node.Node, viewWorld [6]float64, cullBounds types.Rect) bool {
	switch n.Type {
	case types.NodeTypeContainer:
		return false
	case types.NodeTypeMesh:
		aabb := mesh.MeshWorldAABBOffset(n, viewWorld)
		if aabb.Width == 0 && aabb.Height == 0 {
			return false
		}
		return !aabb.Intersects(cullBounds)
	}

	w, h := NodeDimensions(n)
	if w == 0 && h == 0 {
		return false // Can't determine size; don't cull.
	}

	aabb := WorldAABB(viewWorld, w, h)
	return !aabb.Intersects(cullBounds)
}
