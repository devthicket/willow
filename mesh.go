package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/mesh"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// transformVertices applies an affine transform and color tint to src vertices.
// Delegates to mesh.TransformVertices.
func transformVertices(src, dst []ebiten.Vertex, transform [6]float64, tint Color) {
	mesh.TransformVertices(src, dst, transform, tint)
}

// computeMeshAABB scans DstX/DstY of the given vertices and returns the AABB.
// Delegates to mesh.ComputeMeshAABB.
func computeMeshAABB(verts []ebiten.Vertex) Rect {
	return mesh.ComputeMeshAABB(verts)
}

// ensureTransformedVerts grows the node's transformedVerts buffer to fit len(n.Mesh.Vertices).
// Delegates to mesh.EnsureTransformedVerts.
func ensureTransformedVerts(n *Node) []ebiten.Vertex {
	return mesh.EnsureTransformedVerts(n)
}

// recomputeMeshAABB recomputes the cached local-space AABB if dirty.
// Delegates to mesh.RecomputeMeshAABB.
func recomputeMeshAABB(n *Node) {
	mesh.RecomputeMeshAABB(n)
}

// ensureWhitePixel returns the lazily-initialized 1x1 white pixel image.
func ensureWhitePixel() *ebiten.Image {
	return mesh.EnsureWhitePixel()
}

// meshWorldAABB computes the world-space AABB for a mesh node.
// Delegates to mesh.MeshWorldAABB using camera.WorldAABB.
func meshWorldAABB(n *node.Node, transform [6]float64) types.Rect {
	return mesh.MeshWorldAABB(n, transform, camera.WorldAABB)
}

// meshWorldAABBOffset computes the mesh world AABB by transforming the four
// corners of the local AABB. Delegates to mesh.MeshWorldAABBOffset.
func meshWorldAABBOffset(n *node.Node, transform [6]float64) types.Rect {
	return mesh.MeshWorldAABBOffset(n, transform)
}
