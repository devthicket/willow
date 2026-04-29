package render

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Painter is the typed view onto the render pipeline passed to a Node's
// custom-paint handler. It exposes safe methods for appending render commands
// while keeping the pipeline's internal state hidden.
//
// The pointer is only valid for the duration of the callback. Do not store it.
type Painter struct {
	p         *Pipeline
	n         *node.Node
	viewWorld [6]float64
	building  bool
}

// Pipeline returns the underlying render pipeline. The Pipeline type is
// re-exported as willow.Pipeline but is engine-internal: its fields and
// methods may change between minor versions. Prefer the other Painter
// methods unless you need something they don't expose.
func (p *Painter) Pipeline() *Pipeline { return p.p }

// Node returns the node whose custom-paint handler is currently executing.
func (p *Painter) Node() *node.Node { return p.n }

// ViewTransform returns the camera transform (world to screen).
func (p *Painter) ViewTransform() [6]float64 { return p.p.ViewTransform }

// WorldTransform returns the node's world-space transform pre-multiplied by
// the camera view. This is the matrix the node's own emit would use.
func (p *Painter) WorldTransform() [6]float64 { return p.viewWorld }

// CullBounds returns the camera's world-space cull rectangle and whether
// culling is currently active. When inactive, the rect is undefined.
func (p *Painter) CullBounds() (types.Rect, bool) {
	return p.p.CullBounds, p.p.CullActive
}

// IsBuildingCache reports whether this paint pass is happening inside a
// CacheAsTree build. Use it to skip emitting transient or animated content
// that should not be folded into a cached subtree.
func (p *Painter) IsBuildingCache() bool { return p.building }

// AppendCommand pushes a raw RenderCommand. The caller is responsible for
// incrementing *treeOrder and assigning cmd.TreeOrder before calling.
//
// During a CacheAsTree build (see IsBuildingCache), the host node's ID is
// stamped onto cmd.EmittingNodeID so the cache invalidator can attribute
// the command back to its source.
func (p *Painter) AppendCommand(cmd RenderCommand) {
	if p.building {
		cmd.EmittingNodeID = p.n.ID
	}
	p.p.Commands = append(p.p.Commands, cmd)
}

// PaintDefault emits the host node's normal render commands (sprite, mesh,
// text, particles). Use this when you want to add custom geometry alongside
// the node's standard rendering rather than replacing it. Containers and
// non-renderable nodes emit nothing here.
func (p *Painter) PaintDefault(treeOrder *int) {
	if !p.n.Renderable_ {
		return
	}
	p.p.emitNodeInline(p.n, p.viewWorld, treeOrder, p.building)
}

// TrianglesPaint describes a single batch of textured triangles for AppendTriangles.
type TrianglesPaint struct {
	Verts       []ebiten.Vertex
	Inds        []uint16
	Image       *ebiten.Image
	BlendMode   types.BlendMode
	RenderLayer uint8
	GlobalOrder int
}

// AppendTriangles emits a CommandMesh batch and increments *treeOrder. Vertex
// destination coordinates should already be in screen space (apply
// WorldTransform yourself when writing them).
func (p *Painter) AppendTriangles(t TrianglesPaint, treeOrder *int) {
	*treeOrder++
	cmd := RenderCommand{
		Type:        CommandMesh,
		BlendMode:   t.BlendMode,
		RenderLayer: t.RenderLayer,
		GlobalOrder: t.GlobalOrder,
		TreeOrder:   *treeOrder,
		MeshVerts:   t.Verts,
		MeshInds:    t.Inds,
		MeshImage:   t.Image,
	}
	if p.building {
		cmd.EmittingNodeID = p.n.ID
	}
	p.p.Commands = append(p.p.Commands, cmd)
}
