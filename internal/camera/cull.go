package camera

import (
	"github.com/devthicket/willow/internal/mesh"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

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
		if n.Mesh != nil {
			mesh.RecomputeMeshAABB(n)
			return n.Mesh.Aabb.Width, n.Mesh.Aabb.Height
		}
		return 0, 0
	case types.NodeTypeParticleEmitter:
		if n.Emitter != nil {
			r := &n.Emitter.Config.Region
			return float64(r.OriginalW), float64(r.OriginalH)
		}
		return 0, 0
	case types.NodeTypeText:
		if n.TextBlock != nil {
			n.TextBlock.Layout()
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

	aabb := types.WorldAABB(viewWorld, w, h)
	return !aabb.Intersects(cullBounds)
}
