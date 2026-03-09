package node

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/text"
	"github.com/phanxgames/willow/internal/types"
)

// --- Getters ---

func (n *Node) X() float64              { return n.X_ }
func (n *Node) Y() float64              { return n.Y_ }
func (n *Node) ScaleX() float64         { return n.ScaleX_ }
func (n *Node) ScaleY() float64         { return n.ScaleY_ }
func (n *Node) Rotation() float64       { return n.Rotation_ }
func (n *Node) Alpha() float64          { return n.Alpha_ }
func (n *Node) Color() types.Color      { return n.Color_ }
func (n *Node) BlendMode() types.BlendMode { return n.BlendMode_ }
func (n *Node) ZIndex() int             { return n.ZIndex_ }
func (n *Node) Visible() bool           { return n.Visible_ }
func (n *Node) Renderable() bool        { return n.Renderable_ }
func (n *Node) CustomImage() *ebiten.Image { return n.CustomImage_ }
func (n *Node) TextureRegion() types.TextureRegion { return n.TextureRegion_ }
func (n *Node) SkewX() float64          { return n.SkewX_ }
func (n *Node) SkewY() float64          { return n.SkewY_ }
func (n *Node) PivotX() float64         { return n.PivotX_ }
func (n *Node) PivotY() float64         { return n.PivotY_ }

// --- Visual property setters ---

func (n *Node) SetColor(c types.Color) {
	n.Color_ = c
	invalidateAncestorCache(n)
}

func (n *Node) SetBlendMode(b types.BlendMode) {
	n.BlendMode_ = b
	invalidateAncestorCache(n)
}

func (n *Node) SetVisible(v bool) {
	n.Visible_ = v
	invalidateAncestorCache(n)
}

func (n *Node) SetRenderable(r bool) {
	n.Renderable_ = r
	invalidateAncestorCache(n)
}

func (n *Node) SetTextureRegion(r types.TextureRegion) {
	pageChanged := n.TextureRegion_.Page != r.Page
	n.TextureRegion_ = r
	if pageChanged {
		invalidateAncestorCache(n)
		return
	}
	if RegisterAnimatedInCacheFn != nil {
		RegisterAnimatedInCacheFn(n)
	}
}

func (n *Node) SetRenderLayer(l uint8) {
	n.RenderLayer = l
	invalidateAncestorCache(n)
}

func (n *Node) SetGlobalOrder(o int) {
	n.GlobalOrder = o
	invalidateAncestorCache(n)
}

// --- Text convenience setters ---

func (n *Node) SetContent(s string) {
	n.TextBlock.Content = s
	n.TextBlock.LayoutDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetFont(f interface{ MeasureString(string) (float64, float64); LineHeight() float64 }) {
	n.TextBlock.Font = f
	n.TextBlock.LayoutDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetFontSize(size float64) {
	n.TextBlock.FontSize = size
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

func (n *Node) SetTextColor(c types.Color) {
	n.TextBlock.Color = c
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

func (n *Node) SetAlign(a types.TextAlign) {
	n.TextBlock.Align = a
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

func (n *Node) SetWrapWidth(w float64) {
	n.TextBlock.WrapWidth = w
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

func (n *Node) SetLineHeight(h float64) {
	n.TextBlock.LineHeight = h
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

func (n *Node) SetTextEffects(e *text.TextEffects) {
	n.TextBlock.TextEffects = e
	n.TextBlock.Invalidate()
	invalidateAncestorCache(n)
}

// --- Mesh accessors ---

func (n *Node) MeshVertices() []ebiten.Vertex {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Vertices
}

func (n *Node) MeshIndices() []uint16 {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Indices
}

func (n *Node) MeshImage() *ebiten.Image {
	if n.Mesh == nil {
		return nil
	}
	return n.Mesh.Image
}

func (n *Node) SetMeshData(vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image) {
	if n.Mesh == nil {
		n.Mesh = &MeshData{}
	}
	n.Mesh.Vertices = vertices
	n.Mesh.Indices = indices
	n.Mesh.Image = img
	n.Mesh.AabbDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetMeshVertices(vertices []ebiten.Vertex) {
	if n.Mesh == nil {
		n.Mesh = &MeshData{}
	}
	n.Mesh.Vertices = vertices
	n.Mesh.AabbDirty = true
	invalidateAncestorCache(n)
}

func (n *Node) SetMeshIndices(indices []uint16) {
	if n.Mesh == nil {
		n.Mesh = &MeshData{}
	}
	n.Mesh.Indices = indices
	invalidateAncestorCache(n)
}

func (n *Node) SetMeshImage(img *ebiten.Image) {
	if n.Mesh == nil {
		n.Mesh = &MeshData{}
	}
	n.Mesh.Image = img
	invalidateAncestorCache(n)
}

func (n *Node) EnsureMesh() *MeshData {
	if n.Mesh == nil {
		n.Mesh = &MeshData{}
	}
	return n.Mesh
}

// --- Size ---

func (n *Node) Width() float64 {
	if n.Type != types.NodeTypeSprite {
		return 0
	}
	if n.CustomImage_ != nil {
		return n.ScaleX_ * float64(n.CustomImage_.Bounds().Dx())
	}
	if n.TextureRegion_ == (types.TextureRegion{}) {
		return n.ScaleX_
	}
	return n.ScaleX_ * float64(n.TextureRegion_.OriginalW)
}

func (n *Node) Height() float64 {
	if n.Type != types.NodeTypeSprite {
		return 0
	}
	if n.CustomImage_ != nil {
		return n.ScaleY_ * float64(n.CustomImage_.Bounds().Dy())
	}
	if n.TextureRegion_ == (types.TextureRegion{}) {
		return n.ScaleY_
	}
	return n.ScaleY_ * float64(n.TextureRegion_.OriginalH)
}

func (n *Node) SetSize(w, h float64) {
	if n.CustomImage_ == WhitePixelImage || n.TextureRegion_ == (types.TextureRegion{}) {
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

func (n *Node) SetCustomImage(img *ebiten.Image) {
	n.CustomImage_ = img
	invalidateAncestorCache(n)
}

// --- ZIndex ---

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

// --- Cache ---

func (n *Node) SetCacheAsTree(enabled bool, mode ...types.CacheTreeMode) {
	if SetCacheAsTreeFn != nil {
		SetCacheAsTreeFn(n, enabled, mode...)
	}
}

func (n *Node) InvalidateCacheTree() {
	if InvalidateCacheTreeFn != nil {
		InvalidateCacheTreeFn(n)
	}
}

func (n *Node) IsCacheAsTreeEnabled() bool {
	if IsCacheAsTreeEnabledFn != nil {
		return IsCacheAsTreeEnabledFn(n)
	}
	return n.CacheData != nil
}

// --- Mask ---

func (n *Node) SetMask(maskNode *Node) {
	n.MaskNode = maskNode
	invalidateAncestorCache(n)
}

func (n *Node) ClearMask() {
	n.MaskNode = nil
	invalidateAncestorCache(n)
}

func (n *Node) GetMask() *Node {
	return n.MaskNode
}

// --- CacheAsTexture ---

func (n *Node) SetCacheAsTexture(enabled bool) {
	if n.CacheEnabled == enabled {
		return
	}
	n.CacheEnabled = enabled
	if !enabled {
		if n.CacheTexture != nil {
			n.CacheTexture.Deallocate()
			n.CacheTexture = nil
		}
		n.CacheDirty = false
	} else {
		n.CacheDirty = true
	}
	invalidateAncestorCache(n)
}

func (n *Node) InvalidateCache() {
	if n.CacheEnabled {
		n.CacheDirty = true
	}
}

func (n *Node) IsCacheEnabled() bool {
	return n.CacheEnabled
}

// --- Mesh AABB ---

func (n *Node) InvalidateMeshAABB() {
	if n.Mesh != nil {
		n.Mesh.AabbDirty = true
	}
}

// --- Interaction callback setters ---

func (n *Node) OnPointerDown(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerDown = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnPointerUp(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerUp = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnPointerMove(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerMove = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnClick(fn func(ClickContext)) {
	n.EnsureCallbacks().OnClick = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnDragStart(fn func(DragContext)) {
	n.EnsureCallbacks().OnDragStart = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnDrag(fn func(DragContext)) {
	n.EnsureCallbacks().OnDrag = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnDragEnd(fn func(DragContext)) {
	n.EnsureCallbacks().OnDragEnd = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnPinch(fn func(PinchContext)) {
	n.EnsureCallbacks().OnPinch = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnPointerEnter(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerEnter = fn
	if fn != nil {
		n.Interactable = true
	}
}

func (n *Node) OnPointerLeave(fn func(PointerContext)) {
	n.EnsureCallbacks().OnPointerLeave = fn
	if fn != nil {
		n.Interactable = true
	}
}

// --- Callback getters ---

func (n *Node) GetOnPointerDown() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerDown
}

func (n *Node) GetOnPointerUp() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerUp
}

func (n *Node) GetOnPointerMove() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerMove
}

func (n *Node) GetOnClick() func(ClickContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnClick
}

func (n *Node) GetOnDragStart() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDragStart
}

func (n *Node) GetOnDrag() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDrag
}

func (n *Node) GetOnDragEnd() func(DragContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnDragEnd
}

func (n *Node) GetOnPinch() func(PinchContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPinch
}

func (n *Node) GetOnPointerEnter() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerEnter
}

func (n *Node) GetOnPointerLeave() func(PointerContext) {
	if n.Callbacks == nil {
		return nil
	}
	return n.Callbacks.OnPointerLeave
}

// --- Disposal ---

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
	n.CacheData = nil
	n.CustomImage_ = nil
	n.CustomEmit = nil
	n.Mesh = nil
	n.Emitter = nil
	n.TextBlock = nil
	n.UserData = nil
	n.Callbacks = nil
}

func (n *Node) IsDisposed() bool {
	return n.Disposed
}

// Scene returns the Scene this node belongs to, or nil if not in a scene graph.
// Returns any because node/ cannot import the root package. Callers in root
// should type-assert to *Scene.
func (n *Node) Scene() any {
	return n.Scene_
}

// --- Helpers ---

func invalidateAncestorCache(n *Node) {
	if InvalidateAncestorCacheFn != nil {
		InvalidateAncestorCacheFn(n)
	}
}

// MarkSubtreeDirty marks a node as needing transform and alpha recomputation.
func MarkSubtreeDirty(n *Node) {
	invalidateAncestorCache(n)
	n.TransformDirty = true
	n.AlphaDirty = true
}

// IsAncestor reports whether candidate is an ancestor of node.
func IsAncestor(candidate, node *Node) bool {
	for p := node; p != nil; p = p.Parent {
		if p == candidate {
			return true
		}
	}
	return false
}

// Invalidate marks the node's transform and alpha as dirty.
func (n *Node) Invalidate() {
	n.TransformDirty = true
	n.AlphaDirty = true
	if n.TextBlock != nil {
		n.TextBlock.LayoutDirty = true
		n.TextBlock.UniformsDirty = true
	}
	invalidateAncestorCache(n)
}
