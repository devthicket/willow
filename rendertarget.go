package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/render"
)

// --- Render texture pool (aliased from internal/render) ---

type renderTexturePool = render.RenderTexturePool

// nextPowerOfTwo delegates to render.NextPowerOfTwo.
func nextPowerOfTwo(n int) int {
	return render.NextPowerOfTwo(n)
}

// ToTexture renders a node's subtree to a new offscreen image and returns it.
// The caller owns the returned image (it is NOT pooled). Requires a Scene
// reference to use the render pipeline.
func ToTexture(n *Node, s *Scene) *ebiten.Image {
	bounds := render.SubtreeBounds(n)
	w := int(math.Ceil(bounds.Width))
	h := int(math.Ceil(bounds.Height))
	if w <= 0 || h <= 0 {
		return ebiten.NewImage(1, 1)
	}
	img := ebiten.NewImage(w, h)
	s.pipeline.RenderSubtree(n, img, bounds)
	return img
}
