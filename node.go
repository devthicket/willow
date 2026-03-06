package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// CacheTreeMode controls how a SetCacheAsTree node detects stale caches.
type CacheTreeMode uint8

const (
	// CacheTreeManual requires the user to call InvalidateCacheTree() when
	// the subtree changes. Zero overhead on setters. Best for large tilemaps.
	CacheTreeManual CacheTreeMode = iota + 1
	// CacheTreeAuto (the default) auto-invalidates the cache when setters on
	// descendant nodes are called. Small per-setter overhead. Always correct.
	CacheTreeAuto
)

// --- Placeholder types (replaced by later phases) ---

// Font and TextBlock are defined in text.go (Phase 07).
// ParticleEmitter and EmitterConfig are defined in particle.go (Phase 10).

// Filter is defined in filter.go (Phase 09).

// HitShape defines a custom hit testing region in local coordinates.
// Implement this interface and assign it to Node.HitShape to override
// the default axis-aligned bounding box test.
type HitShape interface {
	// Contains reports whether the local-space point (x, y) is inside the shape.
	Contains(x, y float64) bool
}

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
	visible        bool
	renderable     bool
	transformDirty bool
	alphaDirty     bool
	childrenSorted bool
	// Type determines how this node is rendered (container, sprite, mesh, etc.).
	Type        NodeType
	renderLayer uint8
	blendMode   BlendMode

	// ---- HOT: computed world state (cache lines 0-1) ----

	worldAlpha     float64
	alpha          float64
	worldTransform [6]float64

	// ---- HOT: children for iteration (cache line 1-2) ----

	children       []*Node
	sortedChildren []*Node // reused buffer for ZIndex-sorted traversal order

	// ---- HOT: render command fields (cache lines 2-3) ----

	customImage *ebiten.Image // user-provided offscreen canvas (RenderTexture)

	// Subtree command cache (Phase 15): stores commands in local space,
	// replays with delta remap on cache hit.
	cacheTreeEnabled      bool
	cacheTreeMode         CacheTreeMode
	cacheTreeDirty        bool
	cachedCommands        []cachedCmd
	cachedParentTransform [6]float32
	cachedParentAlpha     float32
	globalOrder           int
	textureRegion         TextureRegion
	color                 Color

	// ---- WARM: local transform (read only when dirty) ----

	x, y         float64
	scaleX       float64
	scaleY       float64
	rotation     float64
	skewX, skewY float64
	pivotX       float64
	pivotY       float64

	// ---- COLD: hierarchy, identity, metadata ----

	// Parent points to this node's parent, or nil for the root.
	Parent *Node
	scene  *Scene
	// ID is a unique auto-assigned identifier (never zero for live nodes).
	ID     uint32
	zIndex int
	// EntityID links this node to an ECS entity.
	EntityID uint32
	// Name is a human-readable label for debugging.
	Name string
	// UserData is an arbitrary value the application can attach to a node.
	UserData any

	// ---- COLD: mesh fields (NodeTypeMesh) ----

	// Vertices holds the mesh vertex data for DrawTriangles.
	Vertices []ebiten.Vertex
	// Indices holds the triangle index list for DrawTriangles.
	Indices []uint16
	// MeshImage is the texture sampled by DrawTriangles.
	MeshImage        *ebiten.Image
	transformedVerts []ebiten.Vertex // preallocated transform buffer
	meshAABB         Rect            // cached local-space AABB
	meshAABBDirty    bool            // recompute AABB when true

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
	customEmit func(s *Scene, treeOrder *int)

	// ---- COLD: hit testing ----

	// Interactable controls whether this node responds to pointer events.
	// When false the entire subtree is excluded from hit testing.
	Interactable bool
	// HitShape overrides the default AABB hit test with a custom shape.
	// Nil means use the node's bounding box.
	HitShape HitShape

	// ---- COLD: filters, cache, mask ----

	// Filters is the chain of visual effects applied to this node's rendered
	// output. Filters are applied in order; each reads from the previous
	// result and writes to a new buffer.
	Filters      []Filter
	cacheEnabled bool
	cacheTexture *ebiten.Image
	cacheDirty   bool
	mask         *Node

	// ---- COLD: per-node pointer callbacks (nil by default; zero cost when unused) ----
	// Scene-level handlers fire before per-node callbacks.
	// Use the setter methods (e.g. OnClick) to assign callbacks; they auto-enable Interactable.

	onPointerDown  func(PointerContext)
	onPointerUp    func(PointerContext)
	onPointerMove  func(PointerContext)
	onClick        func(ClickContext)
	onDragStart    func(DragContext)
	onDrag         func(DragContext)
	onDragEnd      func(DragContext)
	onPinch        func(PinchContext)
	onPointerEnter func(PointerContext)
	onPointerLeave func(PointerContext)

	// ---- COLD: internal ----
	disposed bool
}

// nodeDefaults sets the common default field values shared by all constructors.
func nodeDefaults(n *Node) {
	n.ID = nextNodeID()
	n.scaleX = 1
	n.scaleY = 1
	n.alpha = 1
	n.color = Color{1, 1, 1, 1}
	n.visible = true
	n.renderable = true
	n.transformDirty = true
	n.alphaDirty = true
	n.childrenSorted = true
}

// --- Getters ---

// X returns the local-space X position in pixels.
func (n *Node) X() float64 { return n.x }

// Y returns the local-space Y position in pixels.
func (n *Node) Y() float64 { return n.y }

// ScaleX returns the local X scale factor.
func (n *Node) ScaleX() float64 { return n.scaleX }

// ScaleY returns the local Y scale factor.
func (n *Node) ScaleY() float64 { return n.scaleY }

// Rotation returns the local rotation in radians (clockwise).
func (n *Node) Rotation() float64 { return n.rotation }

// SkewX returns the X shear angle in radians.
func (n *Node) SkewX() float64 { return n.skewX }

// SkewY returns the Y shear angle in radians.
func (n *Node) SkewY() float64 { return n.skewY }

// PivotX returns the X transform origin in local pixels.
func (n *Node) PivotX() float64 { return n.pivotX }

// PivotY returns the Y transform origin in local pixels.
func (n *Node) PivotY() float64 { return n.pivotY }

// Alpha returns the node's opacity in [0, 1].
func (n *Node) Alpha() float64 { return n.alpha }

// Color returns the node's multiplicative tint color.
func (n *Node) Color() Color { return n.color }

// Visible returns whether this node and its subtree are drawn.
func (n *Node) Visible() bool { return n.visible }

// Renderable returns whether this node emits render commands.
func (n *Node) Renderable() bool { return n.renderable }

// BlendMode returns the compositing operation used when drawing this node.
func (n *Node) BlendMode() BlendMode { return n.blendMode }

// RenderLayer returns the primary sort key for render commands.
func (n *Node) RenderLayer() uint8 { return n.renderLayer }

// GlobalOrder returns the secondary sort key within the same RenderLayer.
func (n *Node) GlobalOrder() int { return n.globalOrder }

// ZIndex returns the draw order among siblings.
func (n *Node) ZIndex() int { return n.zIndex }

// TextureRegion returns the sub-image region within an atlas page.
func (n *Node) TextureRegion() TextureRegion { return n.textureRegion }

// Width returns the effective pixel width of this node.
// For WhitePixel sprites, this equals ScaleX. For textured sprites,
// this equals ScaleX * region width. For non-sprite nodes, returns 0.
func (n *Node) Width() float64 {
	if n.Type != NodeTypeSprite {
		return 0
	}
	if n.customImage != nil {
		return n.scaleX * float64(n.customImage.Bounds().Dx())
	}
	if n.textureRegion == (TextureRegion{}) {
		return n.scaleX // WhitePixel: 1x1, scale IS the size
	}
	return n.scaleX * float64(n.textureRegion.OriginalW)
}

// Height returns the effective pixel height of this node.
// For WhitePixel sprites, this equals ScaleY. For textured sprites,
// this equals ScaleY * region height. For non-sprite nodes, returns 0.
func (n *Node) Height() float64 {
	if n.Type != NodeTypeSprite {
		return 0
	}
	if n.customImage != nil {
		return n.scaleY * float64(n.customImage.Bounds().Dy())
	}
	if n.textureRegion == (TextureRegion{}) {
		return n.scaleY // WhitePixel: 1x1, scale IS the size
	}
	return n.scaleY * float64(n.textureRegion.OriginalH)
}

// SetSize sets the node's visual size in pixels. For WhitePixel sprites,
// this sets ScaleX/ScaleY directly. For textured sprites, it computes
// the appropriate scale to achieve the given pixel dimensions.
func (n *Node) SetSize(w, h float64) {
	if n.customImage == WhitePixel || n.textureRegion == (TextureRegion{}) {
		n.scaleX = w
		n.scaleY = h
	} else if n.customImage != nil {
		b := n.customImage.Bounds()
		if b.Dx() > 0 {
			n.scaleX = w / float64(b.Dx())
		}
		if b.Dy() > 0 {
			n.scaleY = h / float64(b.Dy())
		}
	} else if n.textureRegion.OriginalW > 0 && n.textureRegion.OriginalH > 0 {
		n.scaleX = w / float64(n.textureRegion.OriginalW)
		n.scaleY = h / float64(n.textureRegion.OriginalH)
	}
	n.transformDirty = true
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
	n := &Node{Name: name, Type: NodeTypeSprite, textureRegion: region}
	nodeDefaults(n)
	// If no region is specified (zero value), default to WhitePixel
	if region == (TextureRegion{}) {
		n.customImage = WhitePixel
	}
	return n
}

// NewRect creates a solid-color rectangle node. This is a convenience wrapper
// around NewSprite with an empty TextureRegion (WhitePixel), sized via ScaleX/Y.
func NewRect(name string, w, h float64, c Color) *Node {
	n := NewSprite(name, TextureRegion{})
	n.SetSize(w, h)
	n.color = c
	return n
}

// NewMesh creates a mesh node that uses DrawTriangles for rendering.
func NewMesh(name string, img *ebiten.Image, vertices []ebiten.Vertex, indices []uint16) *Node {
	n := &Node{
		Name:          name,
		Type:          NodeTypeMesh,
		MeshImage:     img,
		Vertices:      vertices,
		Indices:       indices,
		meshAABBDirty: true,
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
		textureRegion: cfg.Region,
		blendMode:     cfg.BlendMode,
		Emitter:       emitter,
	}
	nodeDefaults(n)
	// If no region is specified (zero value), default to WhitePixel so particles
	// render as solid-color quads without needing an atlas.
	if cfg.Region == (TextureRegion{}) {
		n.customImage = WhitePixel
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
			Color:         Color{1, 1, 1, 1},
			layoutDirty:   true,
			uniformsDirty: true,
		},
	}
	nodeDefaults(n)
	return n
}

// SetCustomImage sets a user-provided *ebiten.Image to display instead of TextureRegion.
// Used by RenderTexture to attach a persistent offscreen canvas to a sprite node.
func (n *Node) SetCustomImage(img *ebiten.Image) {
	n.customImage = img
	invalidateAncestorCache(n)
}

// CustomImage returns the user-provided image, or nil if not set.
func (n *Node) CustomImage() *ebiten.Image {
	return n.customImage
}

// --- Visual property setters ---
// These setters update the field and invalidate ancestor static caches.
// The underlying fields remain public for reads.

// SetColor sets the node's tint color and invalidates ancestor static caches.
func (n *Node) SetColor(c Color) {
	n.color = c
	invalidateAncestorCache(n)
}

// SetBlendMode sets the node's blend mode and invalidates ancestor static caches.
func (n *Node) SetBlendMode(b BlendMode) {
	n.blendMode = b
	invalidateAncestorCache(n)
}

// SetVisible sets the node's visibility and invalidates ancestor caches.
func (n *Node) SetVisible(v bool) {
	n.visible = v
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
	}
	invalidateAncestorCache(n)
}

// SetRenderable sets whether the node emits render commands and invalidates ancestor static caches.
func (n *Node) SetRenderable(r bool) {
	n.renderable = r
	invalidateAncestorCache(n)
}

// SetTextureRegion sets the node's texture region and invalidates ancestor caches.
// If the atlas page is unchanged (e.g. animated tile UV swap), the CacheAsTree
// cache is NOT invalidated  -  instead the node is registered as animated so replay
// reads the live TextureRegion. Page changes always invalidate.
func (n *Node) SetTextureRegion(r TextureRegion) {
	pageChanged := n.textureRegion.Page != r.Page
	n.textureRegion = r
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
	n.renderLayer = l
	invalidateAncestorCache(n)
}

// SetContent sets the text content of a text node and invalidates its layout.
// Panics if called on a node without a TextBlock (programmer error).
func (n *Node) SetContent(s string) {
	n.TextBlock.Content = s
	n.TextBlock.layoutDirty = true
	invalidateAncestorCache(n)
}

// SetFont changes the font on this text node's TextBlock and invalidates layout.
// Panics if called on a node without a TextBlock (programmer error).
func (n *Node) SetFont(f Font) {
	n.TextBlock.Font = f
	n.TextBlock.layoutDirty = true
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
	n.globalOrder = o
	invalidateAncestorCache(n)
}

// --- Interaction callback setters ---
// Each setter stores the callback and auto-enables Interactable.
// Pass nil to clear a callback. Interactable is not auto-disabled on nil
// because other callbacks may still be set.

// OnPointerDown sets the callback for pointer-down events on this node.
func (n *Node) OnPointerDown(fn func(PointerContext)) {
	n.onPointerDown = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerUp sets the callback for pointer-up events on this node.
func (n *Node) OnPointerUp(fn func(PointerContext)) {
	n.onPointerUp = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerMove sets the callback for pointer-move (hover) events on this node.
func (n *Node) OnPointerMove(fn func(PointerContext)) {
	n.onPointerMove = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnClick sets the callback for click events on this node.
func (n *Node) OnClick(fn func(ClickContext)) {
	n.onClick = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDragStart sets the callback for drag-start events on this node.
func (n *Node) OnDragStart(fn func(DragContext)) {
	n.onDragStart = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDrag sets the callback for drag events on this node.
func (n *Node) OnDrag(fn func(DragContext)) {
	n.onDrag = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnDragEnd sets the callback for drag-end events on this node.
func (n *Node) OnDragEnd(fn func(DragContext)) {
	n.onDragEnd = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPinch sets the callback for pinch gesture events on this node.
func (n *Node) OnPinch(fn func(PinchContext)) {
	n.onPinch = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerEnter sets the callback for pointer-enter events on this node.
func (n *Node) OnPointerEnter(fn func(PointerContext)) {
	n.onPointerEnter = fn
	if fn != nil {
		n.Interactable = true
	}
}

// OnPointerLeave sets the callback for pointer-leave events on this node.
func (n *Node) OnPointerLeave(fn func(PointerContext)) {
	n.onPointerLeave = fn
	if fn != nil {
		n.Interactable = true
	}
}

// --- Callback getters ---

func (n *Node) HasOnPointerDown() bool                 { return n.onPointerDown != nil }
func (n *Node) GetOnPointerDown() func(PointerContext) { return n.onPointerDown }

func (n *Node) HasOnPointerUp() bool                 { return n.onPointerUp != nil }
func (n *Node) GetOnPointerUp() func(PointerContext) { return n.onPointerUp }

func (n *Node) HasOnPointerMove() bool                 { return n.onPointerMove != nil }
func (n *Node) GetOnPointerMove() func(PointerContext) { return n.onPointerMove }

func (n *Node) HasOnClick() bool               { return n.onClick != nil }
func (n *Node) GetOnClick() func(ClickContext) { return n.onClick }

func (n *Node) HasOnDragStart() bool              { return n.onDragStart != nil }
func (n *Node) GetOnDragStart() func(DragContext) { return n.onDragStart }

func (n *Node) HasOnDrag() bool              { return n.onDrag != nil }
func (n *Node) GetOnDrag() func(DragContext) { return n.onDrag }

func (n *Node) HasOnDragEnd() bool              { return n.onDragEnd != nil }
func (n *Node) GetOnDragEnd() func(DragContext) { return n.onDragEnd }

func (n *Node) HasOnPinch() bool               { return n.onPinch != nil }
func (n *Node) GetOnPinch() func(PinchContext) { return n.onPinch }

func (n *Node) HasOnPointerEnter() bool                 { return n.onPointerEnter != nil }
func (n *Node) GetOnPointerEnter() func(PointerContext) { return n.onPointerEnter }

func (n *Node) HasOnPointerLeave() bool                 { return n.onPointerLeave != nil }
func (n *Node) GetOnPointerLeave() func(PointerContext) { return n.onPointerLeave }

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
	n.children = append(n.children, child)
	n.childrenSorted = false
	propagateScene(child, n.scene)
	markSubtreeDirty(child)
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
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
	if index < 0 || index > len(n.children) {
		panic("willow: child index out of range")
	}
	if child.Parent != nil {
		child.Parent.removeChildByPtr(child)
	}
	child.Parent = n
	n.children = append(n.children, nil)
	copy(n.children[index+1:], n.children[index:])
	n.children[index] = child
	n.childrenSorted = false
	propagateScene(child, n.scene)
	markSubtreeDirty(child)
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
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
	n.childrenSorted = false
	markSubtreeDirty(child)
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
	}
	invalidateAncestorCache(n)
}

// RemoveChildAt removes and returns the child at the given index.
// Panics if the index is out of range.
func (n *Node) RemoveChildAt(index int) *Node {
	if globalDebug {
		debugCheckDisposed(n, "RemoveChildAt")
	}
	if index < 0 || index >= len(n.children) {
		panic("willow: child index out of range")
	}
	child := n.children[index]
	copy(n.children[index:], n.children[index+1:])
	n.children[len(n.children)-1] = nil
	n.children = n.children[:len(n.children)-1]
	child.Parent = nil
	propagateScene(child, nil)
	n.childrenSorted = false
	markSubtreeDirty(child)
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
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
	for i, child := range n.children {
		child.Parent = nil
		propagateScene(child, nil)
		markSubtreeDirty(child)
		n.children[i] = nil // allow GC of removed children
	}
	n.children = n.children[:0]
	n.childrenSorted = true
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
	}
	invalidateAncestorCache(n)
}

// Children returns the child list. The returned slice MUST NOT be mutated by the caller.
func (n *Node) Children() []*Node {
	return n.children
}

// NumChildren returns the number of children.
func (n *Node) NumChildren() int {
	return len(n.children)
}

// ChildAt returns the child at the given index.
// Panics if the index is out of range.
func (n *Node) ChildAt(index int) *Node {
	return n.children[index]
}

// SetChildIndex moves child to a new index among its siblings.
// Panics if child is not a child of n or if index is out of range.
func (n *Node) SetChildIndex(child *Node, index int) {
	if child.Parent != n {
		panic("willow: child's parent is not this node")
	}
	nc := len(n.children)
	if index < 0 || index >= nc {
		panic("willow: child index out of range")
	}
	oldIndex := -1
	for i, c := range n.children {
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
		copy(n.children[oldIndex:], n.children[oldIndex+1:index+1])
	} else {
		copy(n.children[index+1:], n.children[index:oldIndex])
	}
	n.children[index] = child
	n.childrenSorted = false
}

// SetZIndex sets the node's ZIndex and marks the parent's children as unsorted,
// so the next traversal will re-sort siblings by ZIndex.
func (n *Node) SetZIndex(z int) {
	if n.zIndex == z {
		return
	}
	n.zIndex = z
	if n.Parent != nil {
		n.Parent.childrenSorted = false
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
		n.cacheTreeEnabled = true
		if len(mode) > 0 {
			n.cacheTreeMode = mode[0]
		} else {
			n.cacheTreeMode = CacheTreeAuto
		}
		n.cacheTreeDirty = true
	} else {
		n.cacheTreeEnabled = false
		n.cacheTreeMode = 0
		n.cacheTreeDirty = false
		n.cachedCommands = nil
	}
}

// InvalidateCacheTree marks the cache as stale. Next Draw() re-traverses.
// Works with both Auto and Manual modes.
func (n *Node) InvalidateCacheTree() {
	if n.cacheTreeEnabled {
		n.cacheTreeDirty = true
	}
}

// IsCacheAsTreeEnabled reports whether subtree command caching is enabled.
func (n *Node) IsCacheAsTreeEnabled() bool {
	return n.cacheTreeEnabled
}

// registerAnimatedInCache walks up to the nearest CacheAsTree ancestor and
// promotes this node's cachedCmd from static to animated (source = n) so
// replay reads the live TextureRegion. O(N) scan, runs once per tile that
// starts animating  -  not per frame.
func (n *Node) registerAnimatedInCache() {
	for p := n.Parent; p != nil; p = p.Parent {
		if !p.cacheTreeEnabled {
			continue
		}
		if len(p.cachedCommands) == 0 {
			return // cache not built yet; source will be set on first build
		}
		for i := range p.cachedCommands {
			if p.cachedCommands[i].source == n || p.cachedCommands[i].sourceNodeID == n.ID {
				p.cachedCommands[i].source = n
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
		if p.cacheTreeEnabled {
			if p.cacheTreeMode == CacheTreeAuto {
				p.cacheTreeDirty = true
			}
			return
		}
	}
}

// --- Disposal ---

// Dispose removes this node from its parent, marks it as disposed,
// and recursively disposes all descendants.
func (n *Node) Dispose() {
	if n.disposed {
		return
	}
	n.RemoveFromParent()
	n.dispose()
}

func (n *Node) dispose() {
	n.disposed = true
	n.scene = nil
	n.ID = 0
	for _, child := range n.children {
		child.Parent = nil
		child.dispose()
	}
	n.children = nil
	n.sortedChildren = nil
	n.Parent = nil
	n.HitShape = nil
	n.Filters = nil
	n.cacheEnabled = false
	if n.cacheTexture != nil {
		n.cacheTexture.Deallocate()
		n.cacheTexture = nil
	}
	n.cacheDirty = false
	n.mask = nil
	n.cacheTreeEnabled = false
	n.cacheTreeDirty = false
	n.cachedCommands = nil
	n.customImage = nil
	n.customEmit = nil
	n.MeshImage = nil
	n.transformedVerts = nil
	n.Emitter = nil
	n.TextBlock = nil
	n.UserData = nil
	n.onPointerDown = nil
	n.onPointerUp = nil
	n.onPointerMove = nil
	n.onClick = nil
	n.onDragStart = nil
	n.onDrag = nil
	n.onDragEnd = nil
	n.onPinch = nil
	n.onPointerEnter = nil
	n.onPointerLeave = nil
}

// IsDisposed returns true if this node has been disposed.
func (n *Node) IsDisposed() bool {
	return n.disposed
}

// Scene returns the Scene this node belongs to, or nil if not in a scene graph.
func (n *Node) Scene() *Scene {
	return n.scene
}

// propagateScene recursively sets the scene back-pointer on n and all descendants.
// Early-outs if the scene is already the target value.
func propagateScene(n *Node, s *Scene) {
	if n.scene == s {
		return
	}
	n.scene = s
	for _, child := range n.children {
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

// removeChildByPtr removes child from n.children without clearing child.Parent.
// Uses copy+nil to avoid retaining a dangling pointer in the backing array.
func (n *Node) removeChildByPtr(child *Node) {
	for i, c := range n.children {
		if c == child {
			copy(n.children[i:], n.children[i+1:])
			n.children[len(n.children)-1] = nil
			n.children = n.children[:len(n.children)-1]
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
	node.transformDirty = true
	node.alphaDirty = true
}
