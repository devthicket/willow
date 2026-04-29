package render

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Emitter is the typed view onto the render pipeline passed to a Node's
// custom-emit handler. It exposes safe methods for appending render commands
// while keeping the pipeline's internal state hidden.
//
// The pointer is only valid for the duration of the callback. Do not store it.
type Emitter struct {
	p         *Pipeline
	n         *node.Node
	viewWorld [6]float64
	building  bool
}

// Pipeline returns the underlying render pipeline. The Pipeline type is
// re-exported as willow.Pipeline but is engine-internal — its fields and
// methods may change between minor versions. Prefer the other Emitter
// methods unless you need something they don't expose.
func (e *Emitter) Pipeline() *Pipeline { return e.p }

// Node returns the node whose custom-emit handler is currently executing.
func (e *Emitter) Node() *node.Node { return e.n }

// ViewTransform returns the camera transform (world → screen).
func (e *Emitter) ViewTransform() [6]float64 { return e.p.ViewTransform }

// WorldTransform returns the node's world-space transform pre-multiplied by
// the camera view. This is the matrix the node's own emit would use.
func (e *Emitter) WorldTransform() [6]float64 { return e.viewWorld }

// CullBounds returns the camera's world-space cull rectangle and whether
// culling is currently active. When inactive, the rect is undefined.
func (e *Emitter) CullBounds() (types.Rect, bool) {
	return e.p.CullBounds, e.p.CullActive
}

// IsBuildingCache reports whether this emit is happening inside a CacheAsTree
// build pass. Use it to skip emitting transient or animated content that
// should not be folded into a cached subtree.
func (e *Emitter) IsBuildingCache() bool { return e.building }

// AppendCommand pushes a raw RenderCommand. The caller is responsible for
// incrementing *treeOrder and assigning cmd.TreeOrder before calling.
func (e *Emitter) AppendCommand(cmd RenderCommand) {
	e.p.Commands = append(e.p.Commands, cmd)
}

// EmitDefault emits the host node's normal render commands (sprite, mesh,
// text, particles). Use this when you want to add custom geometry alongside
// the node's standard rendering rather than replacing it. Containers and
// non-renderable nodes emit nothing here.
func (e *Emitter) EmitDefault(treeOrder *int) {
	if !e.n.Renderable_ {
		return
	}
	e.p.emitNodeInline(e.n, e.viewWorld, treeOrder, e.building)
}

// TrianglesEmit describes a single batch of textured triangles for AppendTriangles.
type TrianglesEmit struct {
	Transform   [6]float32
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
func (e *Emitter) AppendTriangles(t TrianglesEmit, treeOrder *int) {
	*treeOrder++
	e.p.Commands = append(e.p.Commands, RenderCommand{
		Type:        CommandMesh,
		Transform:   t.Transform,
		BlendMode:   t.BlendMode,
		RenderLayer: t.RenderLayer,
		GlobalOrder: t.GlobalOrder,
		TreeOrder:   *treeOrder,
		MeshVerts:   t.Verts,
		MeshInds:    t.Inds,
		MeshImage:   t.Image,
	})
}
