package core

import (
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
)

// ToTexture renders a node's subtree to a new offscreen image and returns it.
func ToTexture(n *node.Node, s *Scene) *ebiten.Image {
	bounds := render.SubtreeBounds(n)
	w := int(math.Ceil(bounds.Width))
	h := int(math.Ceil(bounds.Height))
	if w <= 0 || h <= 0 {
		return ebiten.NewImage(1, 1)
	}
	img := ebiten.NewImage(w, h)
	s.Pipeline.RenderSubtree(n, img, bounds)
	return img
}
