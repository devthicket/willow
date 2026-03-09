package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/input"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/types"
)

// --- Type aliases from internal/node ---

// Node is the fundamental scene graph element, aliased from internal/node.
type Node = node.Node

// PointerContext carries pointer event data passed to pointer callbacks.
type PointerContext = node.PointerContext

// ClickContext carries click event data passed to click callbacks.
type ClickContext = node.ClickContext

// DragContext carries drag event data passed to drag callbacks.
type DragContext = node.DragContext

// PinchContext carries two-finger pinch/rotate gesture data.
type PinchContext = node.PinchContext

// nodeCallbacks holds per-node event handler functions.
type nodeCallbacks = node.NodeCallbacks

// meshData is an alias for the internal MeshData type.
type meshData = node.MeshData

// --- Cache type aliases from internal/render ---

// cacheTreeData holds subtree command cache state. Aliased from internal/render.
type cacheTreeData = render.CacheTreeData

// cachedCmd is a render command with cached transform/color. Aliased from internal/render.
type cachedCmd = render.CachedCmd

// getCacheTree returns the *cacheTreeData from n.CacheData, or nil if unset.
func getCacheTree(n *Node) *cacheTreeData {
	return render.GetCacheTreeData(n)
}

// --- Function pointer wiring ---

func init() {
	// Node function pointers
	node.InvalidateAncestorCacheFn = invalidateAncestorCache
	node.PropagateSceneFn = propagateScene
	node.SetCacheAsTreeFn = setCacheAsTreeImpl
	node.InvalidateCacheTreeFn = invalidateCacheTreeImpl
	node.IsCacheAsTreeEnabledFn = func(n *Node) bool { return n.CacheData != nil }
	node.WhitePixelImage = WhitePixel
	node.RegisterAnimatedInCacheFn = registerAnimatedInCache
	node.DebugCheckDisposed = debugCheckDisposed
	node.DebugCheckTreeDepth = debugCheckTreeDepth
	node.DebugCheckChildCount = debugCheckChildCount

	// Render pipeline function pointers
	render.AtlasPageFn = func(pageIdx int) *ebiten.Image {
		return atlasManager().Page(pageIdx)
	}
	render.EnsureMagentaImageFn = ensureMagentaImage
	render.ShouldCullFn = shouldCull
	render.BlendMaskFn = func() types.BlendMode { return types.BlendMask }

	// Render texture function pointers (rendertexture.go path)
	render.PageFn = func(pageIdx int) *ebiten.Image {
		return atlasManager().Page(pageIdx)
	}
	render.MagentaImageFn = ensureMagentaImage
	render.NewSpriteFn = NewSprite

	// Input function pointers
	input.NodeDimensionsFn = nodeDimensions
	input.RebuildSortedChildrenFn = render.RebuildSortedChildren
}

// setCacheAsTreeImpl enables or disables subtree command caching.
func setCacheAsTreeImpl(n *Node, enabled bool, mode ...CacheTreeMode) {
	if enabled {
		if n.CacheData == nil {
			n.CacheData = &cacheTreeData{}
		}
		ct := getCacheTree(n)
		if len(mode) > 0 {
			ct.Mode = mode[0]
		} else {
			ct.Mode = CacheTreeAuto
		}
		ct.Dirty = true
	} else {
		n.CacheData = nil
	}
}

// invalidateCacheTreeImpl marks the cache as stale. Next Draw() re-traverses.
func invalidateCacheTreeImpl(n *Node) {
	if n.CacheData != nil {
		getCacheTree(n).Dirty = true
	}
}

// registerAnimatedInCache walks up to the nearest CacheAsTree ancestor and
// promotes this node's cachedCmd from static to animated (source = n) so
// replay reads the live TextureRegion.
func registerAnimatedInCache(n *Node) {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.CacheData == nil {
			continue
		}
		ct := getCacheTree(p)
		if len(ct.Commands) == 0 {
			return // cache not built yet; source will be set on first build
		}
		for i := range ct.Commands {
			if ct.Commands[i].Source == n || ct.Commands[i].SourceNodeID == n.ID {
				ct.Commands[i].Source = n
				return
			}
		}
		return
	}
}

// invalidateAncestorCache walks up the tree from n to find the nearest
// CacheAsTree ancestor and marks it dirty (auto mode only).
// Also marks n's own cache dirty if n itself is a cache root.
func invalidateAncestorCache(n *Node) {
	// Check n's own cache data (handles self-invalidation for SetVisible,
	// AddChild, etc. — the cache records n's own command, so changes to n
	// or its children should mark the cache stale).
	if n.CacheData != nil {
		ct := getCacheTree(n)
		if ct.Mode == CacheTreeAuto {
			ct.Dirty = true
		}
	}
	for p := n.Parent; p != nil; p = p.Parent {
		if p.CacheData != nil {
			ct := getCacheTree(p)
			if ct.Mode == CacheTreeAuto {
				ct.Dirty = true
			}
			return
		}
	}
}

// --- Constructors ---

// NewContainer creates a container node with no visual representation.
func NewContainer(name string) *Node {
	return node.NewNode(name, NodeTypeContainer)
}

// NewSprite creates a sprite node that renders a texture region.
func NewSprite(name string, region TextureRegion) *Node {
	n := node.NewNode(name, NodeTypeSprite)
	n.TextureRegion_ = region
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
	n := node.NewNode(name, NodeTypeMesh)
	n.Mesh = &meshData{
		Vertices:  vertices,
		Indices:   indices,
		Image:     img,
		AabbDirty: true,
	}
	return n
}

// NewParticleEmitter creates a particle emitter node with a preallocated pool.
func NewParticleEmitter(name string, cfg EmitterConfig) *Node {
	n := node.NewNode(name, NodeTypeParticleEmitter)
	n.TextureRegion_ = cfg.Region
	n.BlendMode_ = cfg.BlendMode
	n.Emitter = newParticleEmitter(cfg)
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
	n := node.NewNode(name, NodeTypeText)
	n.TextBlock = &TextBlock{
		Content:       content,
		Font:          font,
		FontSize:      16,
		Color:         RGBA(1, 1, 1, 1),
		LayoutDirty:   true,
		UniformsDirty: true,
	}
	return n
}

// --- Free functions ---

// propagateScene recursively sets the scene back-pointer on n and all descendants.
// Early-outs if the scene is already the target value.
func propagateScene(n *Node, s any) {
	if n.Scene_ == s {
		return
	}
	n.Scene_ = s
	for _, child := range n.Children_ {
		propagateScene(child, s)
	}
}

// isAncestor reports whether candidate is an ancestor of node.
func isAncestor(candidate, n *Node) bool {
	for p := n; p != nil; p = p.Parent {
		if p == candidate {
			return true
		}
	}
	return false
}

// markSubtreeDirty marks a node as needing transform and alpha recomputation.
// Children inherit the recomputation via parentRecomputed/parentAlphaChanged
// during updateWorldTransform and traverse, so only the subtree root needs
// the flag set (upward-only dirty model, matching Pixi v8 and Starling).
func markSubtreeDirty(n *Node) {
	invalidateAncestorCache(n)
	n.TransformDirty = true
	n.AlphaDirty = true
}
