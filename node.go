package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/node"
)

// CacheTreeMode, HitShape, and other foundation types are aliased
// from internal/types in willow.go.

// --- Callback context placeholders (Phase 08) ---

// PointerContext carries pointer event data passed to pointer callbacks.
type PointerContext struct {
	Node      *Node        // the node under the pointer, or nil if none
	EntityID  uint32       // the hit node's EntityID (for ECS bridging)
	UserData  any          // the hit node's UserData
	GlobalX   float64      // pointer X in world coordinates
	GlobalY   float64      // pointer Y in world coordinates
	LocalX    float64      // pointer X in the hit node's local coordinates
	LocalY    float64      // pointer Y in the hit node's local coordinates
	Button    MouseButton  // which mouse button is involved
	PointerID int          // 0 = mouse, 1-9 = touch contacts
	Modifiers KeyModifiers // keyboard modifier keys held during the event
}

// ClickContext carries click event data passed to click callbacks.
type ClickContext struct {
	Node      *Node        // the clicked node
	EntityID  uint32       // the clicked node's EntityID (for ECS bridging)
	UserData  any          // the clicked node's UserData
	GlobalX   float64      // click X in world coordinates
	GlobalY   float64      // click Y in world coordinates
	LocalX    float64      // click X in the node's local coordinates
	LocalY    float64      // click Y in the node's local coordinates
	Button    MouseButton  // which mouse button was clicked
	PointerID int          // 0 = mouse, 1-9 = touch contacts
	Modifiers KeyModifiers // keyboard modifier keys held during the click
}

// DragContext carries drag event data passed to drag callbacks.
type DragContext struct {
	Node         *Node        // the node being dragged
	EntityID     uint32       // the dragged node's EntityID (for ECS bridging)
	UserData     any          // the dragged node's UserData
	GlobalX      float64      // current pointer X in world coordinates
	GlobalY      float64      // current pointer Y in world coordinates
	LocalX       float64      // current pointer X in the node's local coordinates
	LocalY       float64      // current pointer Y in the node's local coordinates
	StartX       float64      // world X where the drag began
	StartY       float64      // world Y where the drag began
	DeltaX       float64      // X movement since the previous drag event
	DeltaY       float64      // Y movement since the previous drag event
	ScreenDeltaX float64      // X movement in screen pixels since the previous drag event
	ScreenDeltaY float64      // Y movement in screen pixels since the previous drag event
	Button       MouseButton  // which mouse button initiated the drag
	PointerID    int          // 0 = mouse, 1-9 = touch contacts
	Modifiers    KeyModifiers // keyboard modifier keys held during the drag
}

// PinchContext carries two-finger pinch/rotate gesture data.
type PinchContext struct {
	CenterX, CenterY   float64 // midpoint between the two touch points in world coordinates
	Scale, ScaleDelta  float64 // cumulative scale factor and frame-to-frame change
	Rotation, RotDelta float64 // cumulative rotation (radians) and frame-to-frame change
}

// --- ID counter ---

// nodeIDCounter is a plain counter (no atomic  -  willow is single-threaded).
var nodeIDCounter uint32

func nextNodeID() uint32 {
	nodeIDCounter++
	return nodeIDCounter
}

// --- Extracted sub-structs (lazily allocated to reduce Node size) ---

// cacheTreeData holds subtree command cache state. Allocated on first
// SetCacheAsTree(true) call. Only a handful of nodes ever use this.
type cacheTreeData struct {
	mode            CacheTreeMode
	dirty           bool
	commands        []cachedCmd
	parentTransform [6]float32
	parentAlpha     float32
}

// meshData is an alias for the internal MeshData type.
type meshData = node.MeshData

// nodeCallbacks holds per-node event handler functions. Allocated on
// first callback setter call. Most nodes never receive callbacks.
type nodeCallbacks struct {
	OnPointerDown  func(PointerContext)
	OnPointerUp    func(PointerContext)
	OnPointerMove  func(PointerContext)
	OnClick        func(ClickContext)
	OnDragStart    func(DragContext)
	OnDrag         func(DragContext)
	OnDragEnd      func(DragContext)
	OnPinch        func(PinchContext)
	OnPointerEnter func(PointerContext)
	OnPointerLeave func(PointerContext)
}

// ensureCallbacks lazily allocates the callback struct.
func (n *Node) EnsureCallbacks() *nodeCallbacks {
	if n.Callbacks == nil {
		n.Callbacks = &nodeCallbacks{}
	}
	return n.Callbacks
}

// --- Node ---

// Node is the fundamental scene graph element. A single flat struct is used for
// all node types to avoid interface dispatch on the hot path.
//
// Field order is performance-critical: the gc compiler lays out fields in
// declaration order. Hot-path fields (traverse, updateWorldTransform) are
// packed first so they share cache lines. Cold fields (callbacks, mesh data,
// metadata) are pushed to the tail. Do not reorder without benchmarking.
type Node struct {
	// ---- HOT: traverse flags (cache line 0) ----
	// These 8 single-byte fields pack into the first 8 bytes with zero padding.

	// ---- HOT: traverse flags (cache line 0) ----
	Visible_       bool
	Renderable_    bool
	TransformDirty bool
	AlphaDirty     bool
	ChildrenSorted bool
	// Type determines how this node is rendered (container, sprite, mesh, etc.).
	Type        NodeType
	RenderLayer    uint8
	BlendMode_     BlendMode

	// ---- HOT: computed world state (cache lines 0-1) ----

	WorldAlpha     float64
	Alpha_         float64
	WorldTransform [6]float64

	// ---- HOT: children for iteration (cache line 1-2) ----

	Children_      []*Node
	SortedChildren []*Node // reused buffer for ZIndex-sorted traversal order

	// ---- HOT: render command fields (cache lines 2-3) ----

	CustomImage_ *ebiten.Image // user-provided offscreen canvas (RenderTexture)

	cacheTree     *cacheTreeData // lazily allocated subtree command cache
	GlobalOrder   int
	TextureRegion_ TextureRegion
	Color_        Color

	// ---- WARM: local transform (read only when dirty) ----

	X_, Y_       float64
	ScaleX_      float64
	ScaleY_      float64
	Rotation_    float64
	SkewX_, SkewY_ float64
	PivotX_      float64
	PivotY_      float64

	// ---- COLD: hierarchy, identity, metadata ----

	// Parent points to this node's parent, or nil for the root.
	Parent *Node
	Scene_ *Scene
	// ID is a unique auto-assigned identifier (never zero for live nodes).
	ID     uint32
	ZIndex_ int
	// EntityID links this node to an ECS entity.
	EntityID uint32
	// Name is a human-readable label for debugging.
	Name string
	// UserData is an arbitrary value the application can attach to a node.
	UserData any

	// ---- COLD: mesh data (NodeTypeMesh, lazily allocated) ----
	Mesh *meshData

	// ---- COLD: particle and text ----

	// Emitter manages the particle pool and simulation for this node.
	Emitter *ParticleEmitter
	// TextBlock holds the text content, font, and cached layout state.
	TextBlock *TextBlock

	// ---- COLD: update callback ----

	// OnUpdate is called once per tick during Scene.Update if set.
	OnUpdate func(dt float64)

	// customEmit, when non-nil, is called during traverse instead of the
	// normal command-emit path. Used by TileMapLayer to emit
	// CommandTilemap commands. The callback receives the scene and a
	// pointer to the current treeOrder counter.
	CustomEmit func(s *Scene, treeOrder *int)

	// ---- COLD: hit testing ----

	// Interactable controls whether this node responds to pointer events.
	// When false the entire subtree is excluded from hit testing.
	Interactable bool
	// HitShape overrides the default AABB hit test with a custom shape.
	// Nil means use the node's bounding box.
	HitShape  HitShape
	Draggable bool
	Pinchable bool

	// ---- COLD: filters, cache, mask ----

	// Filters is the chain of visual effects applied to this node's rendered
	// output. Filters are applied in order; each reads from the previous
	// result and writes to a new buffer.
	Filters      []Filter
	CacheEnabled bool
	CacheTexture *ebiten.Image
	CacheDirty   bool
	MaskNode     *Node

	// ---- COLD: per-node pointer callbacks (lazily allocated) ----
	Callbacks *nodeCallbacks

	// ---- COLD: render integration ----
	EmitFn node.EmitFn

	// ---- COLD: internal ----
	Disposed bool
}

// nodeDefaults sets the common default field values shared by all constructors.
func nodeDefaults(n *Node) {
	n.ID = nextNodeID()
	n.ScaleX_ = 1
	n.ScaleY_ = 1
	n.Alpha_ = 1
	n.Color_ = RGBA(1, 1, 1, 1)
	n.Visible_ = true
	n.Renderable_ = true
	n.TransformDirty = true
	n.AlphaDirty = true
	n.ChildrenSorted = true
}

// --- Getters ---

// X returns the local-space X position in pixels.
func (n *Node) X() float64 { return n.X_ }

// Y returns the local-space Y position in pixels.
func (n *Node) Y() float64 { return n.Y_ }

// ScaleX returns the local X scale factor.
func (n *Node) ScaleX() float64 { return n.ScaleX_ }

// ScaleY returns the local Y scale factor.
func (n *Node) ScaleY() float64 { return n.ScaleY_ }

// Rotation returns the local rotation in radians (clockwise).
func (n *Node) Rotation() float64 { return n.Rotation_ }

// SkewX returns the X shear angle in radians.
func (n *Node) SkewX() float64 { return n.SkewX_ }

// SkewY returns the Y shear angle in radians.
func (n *Node) SkewY() float64 { return n.SkewY_ }

// PivotX returns the X transform origin in local pixels.
func (n *Node) PivotX() float64 { return n.PivotX_ }

// PivotY returns the Y transform origin in local pixels.
func (n *Node) PivotY() float64 { return n.PivotY_ }

// Alpha returns the node's opacity in [0, 1].
func (n *Node) Alpha() float64 { return n.Alpha_ }

// Color returns the node's multiplicative tint color.
func (n *Node) Color() Color { return n.Color_ }

// Visible returns whether this node and its subtree are drawn.
func (n *Node) Visible() bool { return n.Visible_ }

// Renderable returns whether this node emits render commands.
func (n *Node) Renderable() bool { return n.Renderable_ }

// BlendMode returns the compositing operation used when drawing this node.
func (n *Node) BlendMode() BlendMode { return n.BlendMode_ }


// ZIndex returns the draw order among siblings.
func (n *Node) ZIndex() int { return n.ZIndex_ }

// TextureRegion returns the sub-image region within an atlas page.
func (n *Node) TextureRegion() TextureRegion { return n.TextureRegion_ }

// --- Mesh accessors ---

// MeshVertices returns the mesh vertex data, or nil if not a mesh node.
func (n *Node) MeshVertices() []ebiten.Vertex {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Vertices
}

// MeshIndices returns the triangle index list, or nil if not a mesh node.
func (n *Node) MeshIndices() []uint16 {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Indices
}

// MeshImage returns the texture sampled by DrawTriangles, or nil if not a mesh node.
func (n *Node) MeshImage() *ebiten.Image {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Image
}

// SetMeshData replaces the mesh vertex, index, and image data.
// Marks the AABB as dirty. Only valid for NodeTypeMesh nodes.
func (n *Node) SetMeshData(vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image) {
	if n.Mesh == nil {
		n.Mesh = &meshData{}
	}
	n.Mesh.Vertices = vertices
	n.Mesh.Indices = indices
	n.Mesh.Image = img
	n.Mesh.AabbDirty = true
	invalidateAncestorCache(n)
}

// SetMeshVertices replaces just the vertex data and marks the AABB dirty.
func (n *Node) SetMeshVertices(vertices []ebiten.Vertex) {
	if n.Mesh == nil {
		n.Mesh = &meshData{}
	}
	n.Mesh.Vertices = vertices
	n.Mesh.AabbDirty = true
	invalidateAncestorCache(n)
}

// SetMeshIndices replaces just the index data.
func (n *Node) SetMeshIndices(indices []uint16) {
	if n.Mesh == nil {
		n.Mesh = &meshData{}
	}
	n.Mesh.Indices = indices
	invalidateAncestorCache(n)
}

// SetMeshImage replaces just the mesh texture.
func (n *Node) SetMeshImage(img *ebiten.Image) {
	if n.Mesh == nil {
		n.Mesh = &meshData{}
	}
	n.Mesh.Image = img
	invalidateAncestorCache(n)
}

// ensureMesh returns the mesh data, allocating if nil.
func (n *Node) EnsureMesh() *meshData {
	if n.Mesh == nil {
		n.Mesh = &meshData{}
	}
	return n.Mesh
}

// Width returns the effective pixel width of this node.
// For WhitePixel sprites, this equals ScaleX. For textured sprites,
// this equals ScaleX * region width. For non-sprite nodes, returns 0.
func (n *Node) Width() float64 {
	if n.Type != NodeTypeSprite {
		return 0
	}
	if n.CustomImage_ != nil {
		return n.ScaleX_ * float64(n.CustomImage_.Bounds().Dx())
	}
	if n.TextureRegion_ == (TextureRegion{}) {
		return n.ScaleX_ // WhitePixel: 1x1, scale IS the size
	}
	return n.ScaleX_ * float64(n.TextureRegion_.OriginalW)
}

// Height returns the effective pixel height of this node.
// For WhitePixel sprites, this equals ScaleY. For textured sprites,
// this equals ScaleY * region height. For non-sprite nodes, returns 0.
func (n *Node) Height() float64 {
	if n.Type != NodeTypeSprite {
		return 0
	}
	if n.CustomImage_ != nil {
		return n.ScaleY_ * float64(n.CustomImage_.Bounds().Dy())
	}
	if n.TextureRegion_ == (TextureRegion{}) {
		return n.ScaleY_ // WhitePixel: 1x1, scale IS the size
	}
	return n.ScaleY_ * float64(n.TextureRegion_.OriginalH)
}

// SetSize sets the node's visual size in pixels. For WhitePixel sprites,
// this sets ScaleX/ScaleY directly. For textured sprites, it computes
// the appropriate scale to achieve the given pixel dimensions.
func (n *Node) SetSize(w, h float64) {
	if n.CustomImage_ == WhitePixel || n.TextureRegion_ == (TextureRegion{}) {
		n.ScaleX_ = w
		n.ScaleY_ = h
	} else if n.CustomImage_ != nil {
		b := n.CustomImage_.Bounds()
		if b.Dx() > 0 {
			n.ScaleX_ = w / float64(b.Dx())
		}
		if b.Dy() > 0 {
			n.ScaleY_ = h / float64(b.Dy())
		}
	} else if n.TextureRegion_.OriginalW > 0 && n.TextureRegion_.OriginalH > 0 {
		n.ScaleX_ = w / float64(n.TextureRegion_.OriginalW)
		n.ScaleY_ = h / float64(n.TextureRegion_.OriginalH)
	}
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// NewContainer creates a container node with no visual representation.
func NewContainer(name string) *Node {
	n := &Node{Name: name, Type: NodeTypeContainer}
	nodeDefaults(n)
	return n
}

// NewSprite creates a sprite node that renders a texture region.
func NewSprite(name string, region TextureRegion) *Node {
	n := &Node{Name: name, Type: NodeTypeSprite, TextureRegion_: region}
	nodeDefaults(n)
	// If no region is specified (zero value), default to WhitePixel
	if region == (TextureRegion{}) {
		n.CustomImage_ = WhitePixel
	}
	return n
}

// NewRect creates a solid-color rectangle node. This is a convenience wrapper
// around NewSprite with an empty TextureRegion (WhitePixel), sized via ScaleX/Y.
func NewRect(name string, w, h float64, c Color) *Node {
	n := NewSprite(name, TextureRegion{})
	n.SetSize(w, h)
	n.Color_ = c
	return n
}

// NewMesh creates a mesh node that uses DrawTriangles for rendering.
func NewMesh(name string, img *ebiten.Image, vertices []ebiten.Vertex, indices []uint16) *Node {
	n := &Node{
		Name: name,
		Type: NodeTypeMesh,
		Mesh: &meshData{
			Vertices:  vertices,
			Indices:   indices,
			Image:     img,
			AabbDirty: true,
		},
	}
	nodeDefaults(n)
	return n
}

// NewParticleEmitter creates a particle emitter node with a preallocated pool.
func NewParticleEmitter(name string, cfg EmitterConfig) *Node {
	emitter := newParticleEmitter(cfg)
	n := &Node{
		Name:          name,
		Type:          NodeTypeParticleEmitter,
		TextureRegion_: cfg.Region,
		BlendMode_:     cfg.BlendMode,
		Emitter:       emitter,
	}
	nodeDefaults(n)
	// If no region is specified (zero value), default to WhitePixel so particles
	// render as solid-color quads without needing an atlas.
	if cfg.Region == (TextureRegion{}) {
		n.CustomImage_ = WhitePixel
	}
	return n
}

// NewText creates a text node that renders the given string using font.
// FontSize defaults to 16 and is applied at render time as a scale factor,
// independent of ScaleX/ScaleY. Set TextBlock.FontSize after construction
// to change the display size.
func NewText(name string, content string, font Font) *Node {
	n := &Node{
		Name: name,
		Type: NodeTypeText,
		TextBlock: &TextBlock{
			Content:       content,
			Font:          font,
			FontSize:      16,
			Color:         RGBA(1, 1, 1, 1),
			LayoutDirty:   true,
			UniformsDirty: true,
		},
	}
	nodeDefaults(n)
	return n
}

// SetCustomImage sets a user-provided *ebiten.Image to display instead of TextureRegion.
// Used by RenderTexture to attach a persistent offscreen canvas to a sprite node.
func (n *Node) SetCustomImage(img *ebiten.Image) {
	n.CustomImage_ = img
	invalidateAncestorCache(n)
}

// CustomImage returns the user-provided image, or nil if not set.
func (n *Node) CustomImage() *ebiten.Image {
	return n.CustomImage_
}

// --- Visual property setters ---
// These setters update the field and invalidate ancestor static caches.
// The underlying fields remain public for reads.

// SetColor sets the node's tint color and invalidates ancestor static caches.
func (n *Node) SetColor(c Color) {
	n.Color_ = c
	invalidateAncestorCache(n)
}

// SetBlendMode sets the node's blend mode and invalidates ancestor static caches.
func (n *Node) SetBlendMode(b BlendMode) {
	n.BlendMode_ = b
	invalidateAncestorCache(n)
}

// SetVisible sets the node's visibility and invalidates ancestor caches.
func (n *Node) SetVisible(v bool) {
	n.Visible_ = v
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
}

// SetRenderable sets whether the node emits render commands and invalidates ancestor static caches.
func (n *Node) SetRenderable(r bool) {
	n.Renderable_ = r
	invalidateAncestorCache(n)
}

// SetTextureRegion sets the node's texture region and invalidates ancestor caches.
// If the atlas page is unchanged (e.g. animated tile UV swap), the CacheAsTree
// cache is NOT invalidated  -  instead the node is registered as animated so replay
// reads the live TextureRegion. Page changes always invalidate.
func (n *Node) SetTextureRegion(r TextureRegion) {
	pageChanged := n.TextureRegion_.Page != r.Page
	n.TextureRegion_ = r
	if pageChanged {
		invalidateAncestorCache(n)
		return
	}
	// Same page, different UVs → register this node as animated in its
	// cached ancestor so replay reads the live TextureRegion.
	n.registerAnimatedInCache()
}

// SetRenderLayer sets the node's render layer and invalidates ancestor static caches.
func (n *Node) SetRenderLayer(l uint8) {
	n.RenderLayer = l
	invalidateAncestorCache(n)
}

// SetContent sets the text content of a text node and invalidates its layout.
// Panics if called on a node without a TextBlock (programmer error).
func (n *Node) SetContent(s string) {
	n.TextBlock.Content = s
	n.TextBlock.LayoutDirty = true
	invalidateAncestorCache(n)
}

// SetFont changes the font on this text node's TextBlock and invalidates layout.
// Panics if called on a node without a TextBlock (programmer error).
func (n *Node) SetFont(f Font) {
	n.TextBlock.Font = f
	n.TextBlock.LayoutDirty = true
	invalidateAncestorCache(n)
}

// --- Text convenience setters ---
// These delegate to TextBlock fields and auto-invalidate layout.
// Panics if called on a node without a TextBlock (programmer error).

// SetFontSize sets the text display size in pixels and invalidates layout.
func (n *Node) SetFontSize(size float64) {
	n.TextBlock.FontSize = size
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetTextColor sets the text fill color and invalidates the shader uniforms.
func (n *Node) SetTextColor(c Color) {
	n.TextBlock.Color = c
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetAlign sets horizontal text alignment and invalidates layout.
func (n *Node) SetAlign(a TextAlign) {
	n.TextBlock.Align = a
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetWrapWidth sets the maximum line width for word wrapping and invalidates layout.
func (n *Node) SetWrapWidth(w float64) {
	n.TextBlock.WrapWidth = w
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetLineHeight overrides the font's default line height and invalidates layout.
func (n *Node) SetLineHeight(h float64) {
	n.TextBlock.LineHeight = h
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetTextEffects sets text effects (outline, glow, shadow) and invalidates uniforms.
func (n *Node) SetTextEffects(e *TextEffects) {
	n.TextBlock.TextEffects = e
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// SetGlobalOrder sets the node's global order and invalidates ancestor static caches.
func (n *Node) SetGlobalOrder(o int) {
	n.GlobalOrder = o
	invalidateAncestorCache(n)
}

// --- Interaction callback setters ---
// Each setter stores the callback and auto-enables Interactable.
// Pass nil to clear a callback. Interactable is not auto-disabled on nil
// because other callbacks may still be set.

// OnPointerDown sets the callback for pointer-down events on this node.
func (n *Node) OnPointerDown(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerDown = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerUp sets the callback for pointer-up events on this node.
func (n *Node) OnPointerUp(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerUp = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerMove sets the callback for pointer-move (hover) events on this node.
func (n *Node) OnPointerMove(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerMove = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnClick sets the callback for click events on this node.
func (n *Node) OnClick(fn func(ClickContext)) {
	n.EnsureCallbacks().OnClick = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDragStart sets the callback for drag-start events on this node.
func (n *Node) OnDragStart(fn func(DragContext)) {
	n.EnsureCallbacks().OnDragStart = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDrag sets the callback for drag events on this node.
func (n *Node) OnDrag(fn func(DragContext)) {
	n.EnsureCallbacks().OnDrag = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDragEnd sets the callback for drag-end events on this node.
func (n *Node) OnDragEnd(fn func(DragContext)) {
	n.EnsureCallbacks().OnDragEnd = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPinch sets the callback for pinch gesture events on this node.
func (n *Node) OnPinch(fn func(PinchContext)) {
	n.EnsureCallbacks().OnPinch = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerEnter sets the callback for pointer-enter events on this node.
func (n *Node) OnPointerEnter(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerEnter = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerLeave sets the callback for pointer-leave events on this node.
func (n *Node) OnPointerLeave(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerLeave = fn
	if fn != nil {
		n.Interactable = true
	}
}

// --- Callback getters ---

func (n *Node) HasOnPointerDown() bool { return n.Callbacks != nil && n.Callbacks.OnPointerDown != nil }
func (n *Node) GetOnPointerDown() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerDown
}

func (n *Node) HasOnPointerUp() bool { return n.Callbacks != nil && n.Callbacks.OnPointerUp != nil }
func (n *Node) GetOnPointerUp() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerUp
}

func (n *Node) HasOnPointerMove() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerMove != nil
}
func (n *Node) GetOnPointerMove() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerMove
}

func (n *Node) HasOnClick() bool { return n.Callbacks != nil && n.Callbacks.OnClick != nil }
func (n *Node) GetOnClick() func(ClickContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnClick
}

func (n *Node) HasOnDragStart() bool { return n.Callbacks != nil && n.Callbacks.OnDragStart != nil }
func (n *Node) GetOnDragStart() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDragStart
}

func (n *Node) HasOnDrag() bool { return n.Callbacks != nil && n.Callbacks.OnDrag != nil }
func (n *Node) GetOnDrag() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDrag
}

func (n *Node) HasOnDragEnd() bool { return n.Callbacks != nil && n.Callbacks.OnDragEnd != nil }
func (n *Node) GetOnDragEnd() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDragEnd
}

func (n *Node) HasOnPinch() bool { return n.Callbacks != nil && n.Callbacks.OnPinch != nil }
func (n *Node) GetOnPinch() func(PinchContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPinch
}

func (n *Node) HasOnPointerEnter() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerEnter != nil
}
func (n *Node) GetOnPointerEnter() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerEnter
}

func (n *Node) HasOnPointerLeave() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerLeave != nil
}
func (n *Node) GetOnPointerLeave() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerLeave
}

// --- Tree manipulation ---

// AddChild appends child to this node's children.
// If child already has a parent, it is removed from that parent first.
// Panics if child is nil or child is an ancestor of this node (cycle).
func (n *Node) AddChild(child *Node) {
	if child == nil {
		panic("willow: cannot add nil child")
	}
	if globalDebug {
		debugCheckDisposed(n, "AddChild (parent)")
		debugCheckDisposed(child, "AddChild (child)")
	}
	if isAncestor(child, n) {
		panic("willow: adding child would create a cycle")
	}
	if child.Parent != nil {
		child.Parent.removeChildByPtr(child)
	}
	child.Parent = n
	n.Children_ = append(n.Children_, child)
	n.ChildrenSorted = false
	propagateScene(child, n.Scene_)
	markSubtreeDirty(child)
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
	if globalDebug {
		debugCheckTreeDepth(child)
		debugCheckChildCount(n)
	}
}

// AddChildAt inserts child at the given index.
// Same reparenting and cycle-check behavior as AddChild.
func (n *Node) AddChildAt(child *Node, index int) {
	if child == nil {
		panic("willow: cannot add nil child")
	}
	if globalDebug {
		debugCheckDisposed(n, "AddChildAt (parent)")
		debugCheckDisposed(child, "AddChildAt (child)")
	}
	if isAncestor(child, n) {
		panic("willow: adding child would create a cycle")
	}
	if index < 0 || index > len(n.Children_) {
		panic("willow: child index out of range")
	}
	if child.Parent != nil {
		child.Parent.removeChildByPtr(child)
	}
	child.Parent = n
	n.Children_ = append(n.Children_, nil)
	copy(n.Children_[index+1:], n.Children_[index:])
	n.Children_[index] = child
	n.ChildrenSorted = false
	propagateScene(child, n.Scene_)
	markSubtreeDirty(child)
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
	if globalDebug {
		debugCheckTreeDepth(child)
		debugCheckChildCount(n)
	}
}

// RemoveChild detaches child from this node.
// Panics if child.Parent != n.
func (n *Node) RemoveChild(child *Node) {
	if globalDebug {
		debugCheckDisposed(n, "RemoveChild (parent)")
		debugCheckDisposed(child, "RemoveChild (child)")
	}
	if child.Parent != n {
		panic("willow: child's parent is not this node")
	}
	n.removeChildByPtr(child)
	child.Parent = nil
	propagateScene(child, nil)
	n.ChildrenSorted = false
	markSubtreeDirty(child)
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
}

// RemoveChildAt removes and returns the child at the given index.
// Panics if the index is out of range.
func (n *Node) RemoveChildAt(index int) *Node {
	if globalDebug {
		debugCheckDisposed(n, "RemoveChildAt")
	}
	if index < 0 || index >= len(n.Children_) {
		panic("willow: child index out of range")
	}
	child := n.Children_[index]
	copy(n.Children_[index:], n.Children_[index+1:])
	n.Children_[len(n.Children_)-1] = nil
	n.Children_ = n.Children_[:len(n.Children_)-1]
	child.Parent = nil
	propagateScene(child, nil)
	n.ChildrenSorted = false
	markSubtreeDirty(child)
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
	return child
}

// RemoveFromParent detaches this node from its parent.
// No-op if this node has no parent.
func (n *Node) RemoveFromParent() {
	if n.Parent == nil {
		return
	}
	n.Parent.RemoveChild(n)
}

// RemoveChildren detaches all children from this node.
// Children are NOT disposed.
func (n *Node) RemoveChildren() {
	for i, child := range n.Children_ {
		child.Parent = nil
		propagateScene(child, nil)
		markSubtreeDirty(child)
		n.Children_[i] = nil // allow GC of removed children
	}
	n.Children_ = n.Children_[:0]
	n.ChildrenSorted = true
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
	invalidateAncestorCache(n)
}

// Children returns the child list. The returned slice MUST NOT be mutated by the caller.
func (n *Node) Children() []*Node {
	return n.Children_
}

// NumChildren returns the number of children.
func (n *Node) NumChildren() int {
	return len(n.Children_)
}

// ChildAt returns the child at the given index.
// Panics if the index is out of range.
func (n *Node) ChildAt(index int) *Node {
	return n.Children_[index]
}

// FindChild returns the first direct child matching the name pattern, or nil.
// Supports % wildcards: "enemy" (exact), "enemy%" (starts with),
// "%boss" (ends with), "%ene%" (contains).
func (n *Node) FindChild(pattern string) *Node {
	inner, mode := parsePattern(pattern)
	for _, c := range n.Children_ {
		if matchPattern(c.Name, inner, mode) {
			return c
		}
	}
	return nil
}

// FindDescendant returns the first descendant (depth-first) matching the name
// pattern, or nil. Supports % wildcards (see FindChild).
func (n *Node) FindDescendant(pattern string) *Node {
	inner, mode := parsePattern(pattern)
	return n.findDescendant(inner, mode)
}

func (n *Node) findDescendant(inner string, mode wildMode) *Node {
	for _, c := range n.Children_ {
		if matchPattern(c.Name, inner, mode) {
			return c
		}
		if found := c.findDescendant(inner, mode); found != nil {
			return found
		}
	}
	return nil
}

// SetChildIndex moves child to a new index among its siblings.
// Panics if child is not a child of n or if index is out of range.
func (n *Node) SetChildIndex(child *Node, index int) {
	if child.Parent != n {
		panic("willow: child's parent is not this node")
	}
	nc := len(n.Children_)
	if index < 0 || index >= nc {
		panic("willow: child index out of range")
	}
	oldIndex := -1
	for i, c := range n.Children_ {
		if c == child {
			oldIndex = i
			break
		}
	}
	if oldIndex == index {
		return
	}
	// Shift elements to fill the gap and open the target slot.
	if oldIndex < index {
		copy(n.Children_[oldIndex:], n.Children_[oldIndex+1:index+1])
	} else {
		copy(n.Children_[index+1:], n.Children_[index:oldIndex])
	}
	n.Children_[index] = child
	n.ChildrenSorted = false
}

// SetZIndex sets the node's ZIndex and marks the parent's children as unsorted,
// so the next traversal will re-sort siblings by ZIndex.
func (n *Node) SetZIndex(z int) {
	if n.ZIndex_ == z {
		return
	}
	n.ZIndex_ = z
	if n.Parent != nil {
		n.Parent.ChildrenSorted = false
	}
	invalidateAncestorCache(n)
}

// --- Subtree command cache API ---

// SetCacheAsTree enables or disables subtree command caching.
// When enabled, traverse skips this node's subtree and replays cached commands.
// Camera movement is handled automatically via delta remapping.
//
// Mode is optional and defaults to CacheTreeAuto (safe, always correct).
//
// Modes:
//
//	CacheTreeAuto    -  (default) setters on descendant nodes auto-invalidate
//	                  the cache. Small per-setter overhead. Always correct.
//	CacheTreeManual  -  user calls InvalidateCacheTree() when subtree changes.
//	                  Zero overhead on setters. Best for large tilemaps where
//	                  the developer knows exactly when tiles change.
func (n *Node) SetCacheAsTree(enabled bool, mode ...CacheTreeMode) {
	if enabled {
		if n.cacheTree == nil {
			n.cacheTree = &cacheTreeData{}
		}
		if len(mode) > 0 {
			n.cacheTree.mode = mode[0]
		} else {
			n.cacheTree.mode = CacheTreeAuto
		}
		n.cacheTree.dirty = true
	} else {
		n.cacheTree = nil
	}
}

// InvalidateCacheTree marks the cache as stale. Next Draw() re-traverses.
// Works with both Auto and Manual modes.
func (n *Node) InvalidateCacheTree() {
	if n.cacheTree != nil {
		n.cacheTree.dirty = true
	}
}

// IsCacheAsTreeEnabled reports whether subtree command caching is enabled.
func (n *Node) IsCacheAsTreeEnabled() bool {
	return n.cacheTree != nil
}

// registerAnimatedInCache walks up to the nearest CacheAsTree ancestor and
// promotes this node's cachedCmd from static to animated (source = n) so
// replay reads the live TextureRegion. O(N) scan, runs once per tile that
// starts animating  -  not per frame.
func (n *Node) registerAnimatedInCache() {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.cacheTree == nil {
			continue
		}
		if len(p.cacheTree.commands) == 0 {
			return // cache not built yet; source will be set on first build
		}
		for i := range p.cacheTree.commands {
			if p.cacheTree.commands[i].source == n || p.cacheTree.commands[i].sourceNodeID == n.ID {
				p.cacheTree.commands[i].source = n
				return
			}
		}
		return
	}
}

// invalidateAncestorCache walks up the tree from n to find the nearest
// CacheAsTree ancestor and marks it dirty (auto mode only).
// Manual mode stops bubbling  -  user manages invalidation.
func invalidateAncestorCache(n *Node) {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.cacheTree != nil {
			if p.cacheTree.mode == CacheTreeAuto {
				p.cacheTree.dirty = true
			}
			return
		}
	}
}

// --- Disposal ---

// Dispose removes this node from its parent, marks it as disposed,
// and recursively disposes all descendants.
func (n *Node) Dispose() {
	if n.Disposed {
		return
	}
	n.RemoveFromParent()
	n.dispose()
}

func (n *Node) dispose() {
	n.Disposed = true
	n.Scene_ = nil
	n.ID = 0
	for _, child := range n.Children_ {
		child.Parent = nil
		child.dispose()
	}
	n.Children_ = nil
	n.SortedChildren = nil
	n.Parent = nil
	n.HitShape = nil
	n.Filters = nil
	n.CacheEnabled = false
	if n.CacheTexture != nil {
		n.CacheTexture.Deallocate()
		n.CacheTexture = nil
	}
	n.CacheDirty = false
	n.MaskNode = nil
	n.cacheTree = nil
	n.CustomImage_ = nil
	n.CustomEmit = nil
	n.Mesh = nil
	n.Emitter = nil
	n.TextBlock = nil
	n.UserData = nil
	n.Callbacks = nil
}

// IsDisposed returns true if this node has been disposed.
func (n *Node) IsDisposed() bool {
	return n.Disposed
}

// Scene returns the Scene this node belongs to, or nil if not in a scene graph.
func (n *Node) Scene() *Scene {
	return n.Scene_
}

// propagateScene recursively sets the scene back-pointer on n and all descendants.
// Early-outs if the scene is already the target value.
func propagateScene(n *Node, s *Scene) {
	if n.Scene_ == s {
		return
	}
	n.Scene_ = s
	for _, child := range n.Children_ {
		propagateScene(child, s)
	}
}

// --- Helpers ---

// isAncestor reports whether candidate is an ancestor of node.
func isAncestor(candidate, node *Node) bool {
	for p := node; p != nil; p = p.Parent {
		if p == candidate {
			return true
		}
	}
	return false
}

// removeChildByPtr removes child from n.Children_ without clearing child.Parent.
// Uses copy+nil to avoid retaining a dangling pointer in the backing array.
func (n *Node) removeChildByPtr(child *Node) {
	for i, c := range n.Children_ {
		if c == child {
			copy(n.Children_[i:], n.Children_[i+1:])
			n.Children_[len(n.Children_)-1] = nil
			n.Children_ = n.Children_[:len(n.Children_)-1]
			return
		}
	}
}

// markSubtreeDirty marks a node as needing transform and alpha recomputation.
// Children inherit the recomputation via parentRecomputed/parentAlphaChanged
// during updateWorldTransform and traverse, so only the subtree root needs
// the flag set (upward-only dirty model, matching Pixi v8 and Starling).
func markSubtreeDirty(node *Node) {
	invalidateAncestorCache(node)
	node.TransformDirty = true
	node.AlphaDirty = true
}
