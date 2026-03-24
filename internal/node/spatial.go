package node

import "math"

// DistanceBetween returns the world-space distance between two nodes.
func DistanceBetween(a, b *Node) float64 {
	ax, ay := a.WorldTransform[4], a.WorldTransform[5]
	bx, by := b.WorldTransform[4], b.WorldTransform[5]
	dx := bx - ax
	dy := by - ay
	return math.Sqrt(dx*dx + dy*dy)
}

// DirectionBetween returns the normalized direction vector from a to b in world space.
// Returns zero vector if the nodes are at the same position.
func DirectionBetween(a, b *Node) (x, y float64) {
	ax, ay := a.WorldTransform[4], a.WorldTransform[5]
	bx, by := b.WorldTransform[4], b.WorldTransform[5]
	dx := bx - ax
	dy := by - ay
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1e-12 {
		return 0, 0
	}
	return dx / dist, dy / dist
}
