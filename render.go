package willow

import (
	"github.com/phanxgames/willow/internal/render"
)

// --- Render type aliases from internal/render ---

// CommandType identifies the kind of render command.
type CommandType = render.CommandType

const (
	CommandSprite     = render.CommandSprite
	CommandMesh       = render.CommandMesh
	CommandParticle   = render.CommandParticle
	CommandTilemap    = render.CommandTilemap
	CommandSDF        = render.CommandSDF
	CommandBitmapText = render.CommandBitmapText
)

// color32 is a compact RGBA color using float32, for render commands only.
type color32 = render.Color32

// RenderCommand is a single draw instruction emitted during scene traversal.
type RenderCommand = render.RenderCommand

// identityTransform32 is the identity affine matrix as float32.
var identityTransform32 = render.IdentityTransform32

// affine32 converts a [6]float64 affine matrix to [6]float32.
func affine32(m [6]float64) [6]float32 {
	return render.Affine32(m)
}

// subtreeBounds delegates to render.SubtreeBounds.
func subtreeBounds(n *Node) Rect {
	return render.SubtreeBounds(n)
}

// rectUnion delegates to render.RectUnion.
func rectUnion(a, b Rect) Rect {
	return render.RectUnion(a, b)
}
