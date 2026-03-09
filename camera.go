package willow

import "github.com/phanxgames/willow/internal/camera"

// Camera controls the view into the scene: position, zoom, rotation, and viewport.
// Aliased from internal/camera.Camera; all methods live there.
type Camera = camera.Camera

// NewCamera creates a standalone camera with the given viewport.
// Prefer [Scene.NewCamera] to automatically register the camera with a scene.
func NewCamera(viewport Rect) *Camera {
	return camera.NewCamera(viewport)
}

// worldAABB is a package-level shim delegating to camera.WorldAABB.
// Used internally and by tests.
func worldAABB(transform [6]float64, w, h float64) Rect {
	return camera.WorldAABB(transform, w, h)
}

// nodeDimensions is a package-level shim delegating to camera.NodeDimensions.
// Used internally and by tests.
func nodeDimensions(n *Node) (float64, float64) {
	return camera.NodeDimensions(n)
}

// shouldCull is a package-level shim delegating to camera.ShouldCull.
// Used internally and by tests.
func shouldCull(n *Node, viewWorld [6]float64, cullBounds Rect) bool {
	return camera.ShouldCull(n, viewWorld, cullBounds)
}
