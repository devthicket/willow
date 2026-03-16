package node

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/particle"
	"github.com/phanxgames/willow/internal/text"
	"github.com/phanxgames/willow/internal/types"
)

// --- Package-level hooks (wired by root/core init) ---

var (
	// Debug enables debug checks on tree operations.
	Debug bool

	// DebugCheckDisposed panics if a disposed node is used.
	DebugCheckDisposed func(n *Node, op string)

	// DebugCheckTreeDepth warns if tree depth exceeds a threshold.
	DebugCheckTreeDepth func(n *Node)

	// DebugCheckChildCount warns if a node has too many children.
	DebugCheckChildCount func(n *Node)

	// WhitePixelImage is the 1x1 white pixel used for solid color sprites.
	WhitePixelImage *ebiten.Image

	// InvalidateAncestorCacheFn walks up from n to the nearest cached ancestor
	// and marks it dirty (auto mode only). Wired by render/.
	InvalidateAncestorCacheFn func(n *Node)

	// RegisterAnimatedInCacheFn registers a node's cached command as animated
	// so replay reads the live TextureRegion. Wired by render/.
	RegisterAnimatedInCacheFn func(n *Node)

	// SetCacheAsTreeFn enables/disables subtree command caching. Wired by core/.
	SetCacheAsTreeFn func(n *Node, enabled bool, mode ...types.CacheTreeMode)

	// InvalidateCacheTreeFn marks a node's cache as stale. Wired by core/.
	InvalidateCacheTreeFn func(n *Node)

	// IsCacheAsTreeEnabledFn reports whether cache is enabled. Wired by core/.
	IsCacheAsTreeEnabledFn func(n *Node) bool

	// PropagatSceneFn recursively sets the scene back-pointer. Wired by core/.
	PropagateSceneFn func(n *Node, scene any)
)

// EmitFn is the type for render emission functions stored on Node.
// The `any` parameter is an opaque *Scene — node/ never inspects it.
type EmitFn func(n *Node, sceneRef any, viewWorld [6]float64, treeOrder *int)

// --- ID counter ---

var nodeIDCounter uint32

func nextNodeID() uint32 {
	nodeIDCounter++
	return nodeIDCounter
}

// --- Mesh data ---

// MeshData holds mesh-specific fields for NodeTypeMesh nodes.
// Allocated by NewMesh. Only mesh nodes carry this data.
type MeshData struct {
	Vertices         []ebiten.Vertex
	Indices          []uint16
	Image            *ebiten.Image
	TransformedVerts []ebiten.Vertex // preallocated transform buffer
	Aabb             types.Rect      // cached local-space AABB
	AabbDirty        bool            // recompute AABB when true
}

// --- Callback contexts ---

// PointerContext carries pointer event data passed to pointer callbacks.
type PointerContext struct {
	Node      *Node              // the node under the pointer, or nil if none
	EntityID  uint32             // the hit node's EntityID (for ECS bridging)
	UserData  any                // the hit node's UserData
	GlobalX   float64            // pointer X in world coordinates
	GlobalY   float64            // pointer Y in world coordinates
	LocalX    float64            // pointer X in the hit node's local coordinates
	LocalY    float64            // pointer Y in the hit node's local coordinates
	Button    types.MouseButton  // which mouse button is involved
	PointerID int                // 0 = mouse, 1-9 = touch contacts
	Modifiers types.KeyModifiers // keyboard modifier keys held during the event
}

// ClickContext carries click event data passed to click callbacks.
type ClickContext struct {
	Node      *Node              // the clicked node
	EntityID  uint32             // the clicked node's EntityID (for ECS bridging)
	UserData  any                // the clicked node's UserData
	GlobalX   float64            // click X in world coordinates
	GlobalY   float64            // click Y in world coordinates
	LocalX    float64            // click X in the node's local coordinates
	LocalY    float64            // click Y in the node's local coordinates
	Button    types.MouseButton  // which mouse button was clicked
	PointerID int                // 0 = mouse, 1-9 = touch contacts
	Modifiers types.KeyModifiers // keyboard modifier keys held during the click
}

// DragContext carries drag event data passed to drag callbacks.
type DragContext struct {
	Node         *Node              // the node being dragged
	EntityID     uint32             // the dragged node's EntityID (for ECS bridging)
	UserData     any                // the dragged node's UserData
	GlobalX      float64            // current pointer X in world coordinates
	GlobalY      float64            // current pointer Y in world coordinates
	LocalX       float64            // current pointer X in the node's local coordinates
	LocalY       float64            // current pointer Y in the node's local coordinates
	StartX       float64            // world X where the drag began
	StartY       float64            // world Y where the drag began
	DeltaX       float64            // X movement since the previous drag event
	DeltaY       float64            // Y movement since the previous drag event
	ScreenDeltaX float64            // X movement in screen pixels since the previous drag event
	ScreenDeltaY float64            // Y movement in screen pixels since the previous drag event
	Button       types.MouseButton  // which mouse button initiated the drag
	PointerID    int                // 0 = mouse, 1-9 = touch contacts
	Modifiers    types.KeyModifiers // keyboard modifier keys held during the drag
}

// PinchContext carries two-finger pinch/rotate gesture data.
type PinchContext struct {
	CenterX, CenterY  float64 // midpoint between the two touch points in world coordinates
	Scale, ScaleDelta  float64 // cumulative scale factor and frame-to-frame change
	Rotation, RotDelta float64 // cumulative rotation (radians) and frame-to-frame change
}

// --- Node ---

// Node is the fundamental scene graph element. A single flat struct is used for
// all node types to avoid interface dispatch on the hot path.
type Node struct {
	// ---- HOT: traverse flags (cache line 0) ----
	Visible_       bool
	Renderable_    bool
	TransformDirty bool
	AlphaDirty     bool
	ChildrenSorted bool
	Type           types.NodeType
	RenderLayer    uint8
	BlendMode_     types.BlendMode

	// ---- HOT: computed world state ----
	WorldAlpha     float64
	Alpha_         float64
	WorldTransform [6]float64

	// ---- HOT: children for iteration ----
	Children_      []*Node
	SortedChildren []*Node

	// ---- HOT: render command fields ----
	CustomImage_  *ebiten.Image
	CacheData     any // opaque *render.CacheTreeData — node/ never inspects
	GlobalOrder   int
	TextureRegion_ types.TextureRegion
	Color_        types.Color

	// ---- WARM: local transform ----
	X_, Y_       float64
	ScaleX_      float64
	ScaleY_      float64
	Rotation_    float64
	SkewX_, SkewY_ float64
	PivotX_        float64
	PivotY_        float64

	// ---- COLD: hierarchy, identity, metadata ----
	Parent   *Node
	Scene_   any // opaque *Scene — node/ never inspects
	Name     string
	UserData any

	// ---- COLD: pointers ----
	Mesh         *MeshData
	Emitter      *particle.Emitter
	TextBlock    *text.TextBlock
	Callbacks    *NodeCallbacks
	MaskNode     *Node
	CacheTexture *ebiten.Image

	// ---- COLD: interfaces and slices ----
	HitShape types.HitShape
	Filters  []any // opaque []filter.Filter — node/ never inspects

	// ---- COLD: functions ----
	OnUpdate   func(dt float64)
	CustomEmit func(s any, treeOrder *int)
	EmitFn     EmitFn

	// ---- COLD: int-sized ----
	ZIndex_ int
	ID      uint32
	EntityID uint32

	// ---- COLD: bools (packed) ----
	Interactable bool
	Draggable    bool
	Pinchable    bool
	CacheEnabled bool
	CacheDirty   bool
	Disposed     bool
}

// NewNode creates a Node with default field values.
func NewNode(name string, nodeType types.NodeType) *Node {
	n := &Node{
		Name:           name,
		Type:           nodeType,
		ID:             nextNodeID(),
		ScaleX_:        1,
		ScaleY_:        1,
		Alpha_:         1,
		Color_:         types.ColorWhite,
		Visible_:       true,
		Renderable_:    true,
		TransformDirty: true,
		AlphaDirty:     true,
		ChildrenSorted: true,
	}
	return n
}
