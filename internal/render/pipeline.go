package render

import (
	"math"

	"github.com/devthicket/willow/internal/filter"
	"github.com/devthicket/willow/internal/mesh"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/text"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Pipeline owns all render state that was previously on Scene:
// command buffer, sort buffer, batch vertex/index buffers, RT pool,
// deferred RT list, offscreen commands, culling, and batch mode.
type Pipeline struct {
	Commands               []RenderCommand
	SortBuf                []RenderCommand
	BatchVerts             []ebiten.Vertex
	BatchInds              []uint32
	RtPool                 RenderTexturePool
	RtDeferred             []*ebiten.Image
	OffscreenCmds          []RenderCommand
	CullBounds             types.Rect
	CullActive             bool
	ViewTransform          [6]float64
	BuildingCacheFor       *node.Node
	CommandsDirtyThisFrame bool
	BatchMode              BatchMode
	SortKeys               []sortEntry
	SortKeysBuf            []sortEntry
}

// --- Function pointers (wired by root/core) ---

var (
	// AtlasPageFn resolves an atlas page index to its *ebiten.Image.
	AtlasPageFn func(pageIdx int) *ebiten.Image

	// EnsureMagentaImageFn returns the magenta placeholder image.
	EnsureMagentaImageFn func() *ebiten.Image

	// ShouldCullFn determines whether a node should be culled.
	// Imported from camera/ via root wiring.
	ShouldCullFn func(n *node.Node, viewWorld [6]float64, cullBounds types.Rect) bool

	// EnsureSDFShaderFn lazily compiles and returns the SDF shader.
	EnsureSDFShaderFn func() *ebiten.Shader

	// EnsureMSDFShaderFn lazily compiles and returns the MSDF shader.
	EnsureMSDFShaderFn func() *ebiten.Shader

	// BlendMaskFn returns the blend mode for masking operations.
	BlendMaskFn func() types.BlendMode
)

// Traverse walks the node tree depth-first, emitting render commands for
// visible, renderable leaf nodes.
func (p *Pipeline) Traverse(n *node.Node, treeOrder *int) {
	if !n.Visible_ {
		return
	}

	viewWorld := node.MultiplyAffine(p.ViewTransform, n.WorldTransform)

	culled := p.CullActive && n.Renderable_ && ShouldCullFn != nil && ShouldCullFn(n, viewWorld, p.CullBounds)

	// CacheAsTree: replay cached commands (hit) or build cache (miss).
	ctd := GetCacheTreeData(n)
	if ctd != nil && !culled {
		containerTransform32 := Affine32(viewWorld)
		containerAlpha := float32(n.WorldAlpha)

		if !ctd.Dirty && len(ctd.Commands) > 0 {
			p.replayCacheAsTree(n, ctd, containerTransform32, containerAlpha, treeOrder)
			return
		}
		p.buildCacheAsTree(n, ctd, containerTransform32, containerAlpha, treeOrder)
		return
	}

	// Special path: nodes with masks, cache, or filters.
	if !culled && (n.MaskNode != nil || n.CacheEnabled || len(n.Filters) > 0) {
		p.renderSpecialNode(n, treeOrder)
		return
	}

	// Custom emit hook.
	if n.CustomEmit != nil && !culled {
		n.CustomEmit(p, treeOrder)
		p.CommandsDirtyThisFrame = true
		if len(n.Children_) > 0 {
			children := p.sortedChildren(n)
			for _, child := range children {
				p.Traverse(child, treeOrder)
			}
		}
		return
	}

	building := p.BuildingCacheFor != nil

	preCmdLen := len(p.Commands)
	if n.Renderable_ && !culled {
		p.emitNodeInline(n, viewWorld, treeOrder, building)
	}
	if !building && len(p.Commands) > preCmdLen {
		p.CommandsDirtyThisFrame = true
	}

	if len(n.Children_) == 0 {
		return
	}
	children := p.sortedChildren(n)
	for _, child := range children {
		p.Traverse(child, treeOrder)
	}
}

// AppendCommand appends a render command. Used by CustomEmit callbacks.
func (p *Pipeline) AppendCommand(cmd RenderCommand) {
	p.Commands = append(p.Commands, cmd)
}

// emitNodeInline emits a command for a node during main traversal.
func (p *Pipeline) emitNodeInline(n *node.Node, viewWorld [6]float64, treeOrder *int, building bool) {
	switch n.Type {
	case types.NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   Affine32(viewWorld),
			Color:       nodeColor32(n),
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
		}
		if n.CustomImage_ != nil {
			cmd.DirectImage = n.CustomImage_
		} else {
			cmd.TextureRegion = n.TextureRegion_
		}
		if building {
			cmd.EmittingNodeID = n.ID
		}
		p.Commands = append(p.Commands, cmd)

	case types.NodeTypeMesh:
		if n.Mesh == nil || len(n.Mesh.Vertices) == 0 || len(n.Mesh.Indices) == 0 {
			break
		}
		tintColor := types.RGBA(n.Color_.R(), n.Color_.G(), n.Color_.B(), n.Color_.A()*n.WorldAlpha)
		dst := mesh.EnsureTransformedVerts(n)
		mesh.TransformVertices(n.Mesh.Vertices, dst, viewWorld, tintColor)
		*treeOrder++
		p.Commands = append(p.Commands, RenderCommand{
			Type:        CommandMesh,
			Transform:   Affine32(viewWorld),
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
			MeshVerts:   dst,
			MeshInds:    n.Mesh.Indices,
			MeshImage:   n.Mesh.Image,
		})

	case types.NodeTypeParticleEmitter:
		if n.Emitter != nil && n.Emitter.AliveCount() > 0 {
			*treeOrder++
			particleTransform := viewWorld
			ws := n.Emitter.Config.WorldSpace
			if ws {
				particleTransform = p.ViewTransform
			}
			p.Commands = append(p.Commands, RenderCommand{
				Type:               CommandParticle,
				Transform:          Affine32(particleTransform),
				TextureRegion:      n.TextureRegion_,
				DirectImage:        n.CustomImage_,
				Color:              nodeColor32(n),
				BlendMode:          n.BlendMode_,
				RenderLayer:        n.RenderLayer,
				GlobalOrder:        n.GlobalOrder,
				TreeOrder:          *treeOrder,
				Emitter:            n.Emitter,
				WorldSpaceParticle: ws,
			})
		}

	case types.NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			if text.IsPixelFont(n.TextBlock.Font) {
				p.Commands = EmitPixelTextCommand(n.TextBlock, n, viewWorld, p.Commands, treeOrder)
			} else {
				p.Commands = EmitSDFTextCommand(n.TextBlock, n, viewWorld, p.Commands, treeOrder)
			}
		}
	}
}

// Sort sorts the command buffer using radix sort.
func (p *Pipeline) Sort() {
	RadixSort(p.Commands, &p.SortBuf, &p.SortKeys, &p.SortKeysBuf)
}

// ReleaseDeferred releases all deferred render targets back to the pool.
func (p *Pipeline) ReleaseDeferred() {
	for _, img := range p.RtDeferred {
		p.RtPool.Release(img)
	}
	p.RtDeferred = p.RtDeferred[:0]
}

// --- Sorted children helper ---

func (p *Pipeline) sortedChildren(n *node.Node) []*node.Node {
	children := n.Children_
	if !n.ChildrenSorted {
		RebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	return children
}

// RebuildSortedChildren rebuilds the ZIndex-sorted traversal order for a node.
// Uses insertion sort: zero allocations, stable, and optimal for the typical
// case of few children that are nearly sorted.
func RebuildSortedChildren(n *node.Node) {
	nc := len(n.Children_)
	if cap(n.SortedChildren) < nc {
		n.SortedChildren = make([]*node.Node, nc)
	}
	n.SortedChildren = n.SortedChildren[:nc]
	copy(n.SortedChildren, n.Children_)
	for i := 1; i < nc; i++ {
		key := n.SortedChildren[i]
		j := i - 1
		for j >= 0 && n.SortedChildren[j].ZIndex_ > key.ZIndex_ {
			n.SortedChildren[j+1] = n.SortedChildren[j]
			j--
		}
		n.SortedChildren[j+1] = key
	}
	n.ChildrenSorted = true
}

// --- CacheAsTree ---

func (p *Pipeline) replayCacheAsTree(n *node.Node, ctd *CacheTreeData, containerTransform32 [6]float32, containerAlpha float32, treeOrder *int) {
	transformChanged := containerTransform32 != ctd.ParentTransform
	alphaChanged := containerAlpha != ctd.ParentAlpha

	var delta [6]float32
	if transformChanged {
		inv := InvertAffine32(ctd.ParentTransform)
		delta = MultiplyAffine32(containerTransform32, inv)
	}
	var alphaRatio float32 = 1.0
	if alphaChanged && ctd.ParentAlpha > 1e-6 {
		alphaRatio = containerAlpha / ctd.ParentAlpha
	}

	for i := range ctd.Commands {
		src := &ctd.Commands[i]
		cmd := src.Cmd
		if transformChanged {
			cmd.Transform = MultiplyAffine32(delta, cmd.Transform)
		}
		if alphaChanged {
			cmd.Color.R *= alphaRatio
			cmd.Color.G *= alphaRatio
			cmd.Color.B *= alphaRatio
			cmd.Color.A *= alphaRatio
		}
		if src.Source != nil {
			cmd.TextureRegion = src.Source.TextureRegion_
		}
		*treeOrder++
		cmd.TreeOrder = *treeOrder
		p.Commands = append(p.Commands, cmd)
	}
}

func (p *Pipeline) buildCacheAsTree(n *node.Node, ctd *CacheTreeData, containerTransform32 [6]float32, containerAlpha float32, treeOrder *int) {
	startIdx := len(p.Commands)
	p.CommandsDirtyThisFrame = true

	prevBuilding := p.BuildingCacheFor
	p.BuildingCacheFor = n

	viewWorld := node.MultiplyAffine(p.ViewTransform, n.WorldTransform)
	culled := p.CullActive && n.Renderable_ && ShouldCullFn != nil && ShouldCullFn(n, viewWorld, p.CullBounds)
	if n.Renderable_ && !culled {
		p.emitNodeCommandInline(n, treeOrder)
		for i := startIdx; i < len(p.Commands); i++ {
			p.Commands[i].EmittingNodeID = n.ID
		}
	}

	prevCull := p.CullActive
	p.CullActive = false
	if len(n.Children_) > 0 {
		children := p.sortedChildren(n)
		for _, child := range children {
			p.Traverse(child, treeOrder)
		}
	}
	p.CullActive = prevCull
	p.BuildingCacheFor = prevBuilding

	newCmds := p.Commands[startIdx:]
	blocked := false
	for i := range newCmds {
		cmd := &newCmds[i]
		if cmd.Type == CommandMesh || cmd.Type == CommandParticle || cmd.TransientDirectImage {
			blocked = true
			break
		}
	}

	if blocked {
		ctd.Dirty = true
		return
	}

	if cap(ctd.Commands) < len(newCmds) {
		ctd.Commands = make([]CachedCmd, len(newCmds))
	}
	ctd.Commands = ctd.Commands[:len(newCmds)]

	for i := range newCmds {
		ctd.Commands[i] = CachedCmd{
			Cmd:          newCmds[i],
			Source:       nil,
			SourceNodeID: newCmds[i].EmittingNodeID,
		}
	}

	ctd.ParentTransform = containerTransform32
	ctd.ParentAlpha = containerAlpha
	ctd.Dirty = false
}

func (p *Pipeline) emitNodeCommandInline(n *node.Node, treeOrder *int) {
	viewWorld := node.MultiplyAffine(p.ViewTransform, n.WorldTransform)
	switch n.Type {
	case types.NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   Affine32(viewWorld),
			Color:       nodeColor32(n),
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
		}
		if n.CustomImage_ != nil {
			cmd.DirectImage = n.CustomImage_
		} else {
			cmd.TextureRegion = n.TextureRegion_
		}
		p.Commands = append(p.Commands, cmd)
	case types.NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			if text.IsPixelFont(n.TextBlock.Font) {
				p.Commands = EmitPixelTextCommand(n.TextBlock, n, viewWorld, p.Commands, treeOrder)
			} else {
				p.Commands = EmitSDFTextCommand(n.TextBlock, n, viewWorld, p.Commands, treeOrder)
			}
		}
	}
}

// --- Special node rendering ---

func (p *Pipeline) renderSpecialNode(n *node.Node, treeOrder *int) {
	bounds := SubtreeBounds(n)
	padding := filterChainPaddingAny(n.Filters)
	bounds.X -= float64(padding)
	bounds.Y -= float64(padding)
	bounds.Width += float64(padding * 2)
	bounds.Height += float64(padding * 2)

	viewWorld := node.MultiplyAffine(p.ViewTransform, n.WorldTransform)
	bX, bY := bounds.X, bounds.Y
	a, b, c, d := viewWorld[0], viewWorld[1], viewWorld[2], viewWorld[3]
	adjustedTransform := viewWorld
	adjustedTransform[4] += a*bX + c*bY
	adjustedTransform[5] += b*bX + d*bY

	at32 := Affine32(adjustedTransform)
	if n.CacheEnabled && n.CacheTexture != nil && !n.CacheDirty {
		*treeOrder++
		p.Commands = append(p.Commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       Color32{1, 1, 1, float32(n.WorldAlpha)},
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
			DirectImage: n.CacheTexture,
		})
		return
	}

	w := int(math.Ceil(bounds.Width))
	h := int(math.Ceil(bounds.Height))
	if w <= 0 || h <= 0 {
		return
	}

	rt := p.RtPool.Acquire(w, h)
	p.RenderSubtree(n, rt, bounds)
	result := rt

	if n.MaskNode != nil {
		maskRT := p.RtPool.Acquire(w, h)
		p.RenderSubtree(n.MaskNode, maskRT, bounds)
		var op ebiten.DrawImageOptions
		if BlendMaskFn != nil {
			op.Blend = BlendMaskFn().EbitenBlend()
		}
		result.DrawImage(maskRT, &op)
		p.RtPool.Release(maskRT)
	}

	if len(n.Filters) > 0 {
		filtered := ApplyFiltersAny(n.Filters, result, &p.RtPool)
		if filtered != result {
			p.RtPool.Release(result)
			result = filtered
		}
	}

	if n.CacheEnabled {
		if n.CacheTexture != nil {
			n.CacheTexture.Deallocate()
		}
		cacheImg := ebiten.NewImage(w, h)
		var op ebiten.DrawImageOptions
		cacheImg.DrawImage(result, &op)
		n.CacheTexture = cacheImg
		n.CacheDirty = false
		p.RtPool.Release(result)
		*treeOrder++
		p.Commands = append(p.Commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       Color32{1, 1, 1, float32(n.WorldAlpha)},
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
			DirectImage: n.CacheTexture,
		})
		return
	}

	p.RtDeferred = append(p.RtDeferred, result)
	*treeOrder++
	p.Commands = append(p.Commands, RenderCommand{
		Type:                 CommandSprite,
		Transform:            at32,
		Color:                Color32{1, 1, 1, float32(n.WorldAlpha)},
		BlendMode:            n.BlendMode_,
		RenderLayer:          n.RenderLayer,
		GlobalOrder:          n.GlobalOrder,
		TreeOrder:            *treeOrder,
		DirectImage:          result,
		TransientDirectImage: true,
	})
}

// --- Subtree rendering ---

func (p *Pipeline) RenderSubtree(n *node.Node, target *ebiten.Image, bounds types.Rect) {
	savedCmds := p.Commands
	p.Commands = p.OffscreenCmds[:0]

	offsetTransform := [6]float64{1, 0, 0, 1, -bounds.X, -bounds.Y}
	treeOrder := 0

	EmitNodeCommand(p, n, offsetTransform, 1.0, &treeOrder)

	children := n.Children_
	if !n.ChildrenSorted {
		RebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		p.renderSubtreeWalk(child, offsetTransform, 1.0, &treeOrder)
	}

	MergeSort(p.Commands, &p.SortBuf)
	p.SubmitBatches(target)

	p.OffscreenCmds = p.Commands[:0]
	p.Commands = savedCmds
}

func (p *Pipeline) renderSubtreeWalk(n *node.Node, parentTransform [6]float64, parentAlpha float64, treeOrder *int) {
	if !n.Visible_ {
		return
	}

	local := node.ComputeLocalTransform(n)
	transform := node.MultiplyAffine(parentTransform, local)
	alpha := parentAlpha * n.Alpha_

	if n.MaskNode != nil || n.CacheEnabled || len(n.Filters) > 0 {
		p.renderSpecialSubtreeNode(n, transform, alpha, treeOrder)
		return
	}

	EmitNodeCommand(p, n, transform, alpha, treeOrder)

	children := n.Children_
	if !n.ChildrenSorted {
		RebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		p.renderSubtreeWalk(child, transform, alpha, treeOrder)
	}
}

func (p *Pipeline) renderSpecialSubtreeNode(n *node.Node, localTransform [6]float64, alpha float64, treeOrder *int) {
	bounds := SubtreeBounds(n)
	padding := filterChainPaddingAny(n.Filters)
	bounds.X -= float64(padding)
	bounds.Y -= float64(padding)
	bounds.Width += float64(padding * 2)
	bounds.Height += float64(padding * 2)

	bX, bY := bounds.X, bounds.Y
	a, b, c, d := localTransform[0], localTransform[1], localTransform[2], localTransform[3]
	adjustedTransform := localTransform
	adjustedTransform[4] += a*bX + c*bY
	adjustedTransform[5] += b*bX + d*bY

	w := int(math.Ceil(bounds.Width))
	h := int(math.Ceil(bounds.Height))
	if w <= 0 || h <= 0 {
		return
	}

	rt := p.RtPool.Acquire(w, h)
	p.RenderSubtree(n, rt, bounds)
	result := rt

	if n.MaskNode != nil {
		maskRT := p.RtPool.Acquire(w, h)
		p.RenderSubtree(n.MaskNode, maskRT, bounds)
		var op ebiten.DrawImageOptions
		if BlendMaskFn != nil {
			op.Blend = BlendMaskFn().EbitenBlend()
		}
		result.DrawImage(maskRT, &op)
		p.RtPool.Release(maskRT)
	}

	if len(n.Filters) > 0 {
		filtered := ApplyFiltersAny(n.Filters, result, &p.RtPool)
		if filtered != result {
			p.RtPool.Release(result)
			result = filtered
		}
	}

	p.RtDeferred = append(p.RtDeferred, result)
	*treeOrder++
	p.Commands = append(p.Commands, RenderCommand{
		Type:        CommandSprite,
		Transform:   Affine32(adjustedTransform),
		Color:       Color32{1, 1, 1, float32(alpha)},
		BlendMode:   n.BlendMode_,
		RenderLayer: n.RenderLayer,
		GlobalOrder: n.GlobalOrder,
		TreeOrder:   *treeOrder,
		DirectImage: result,
	})
}

// --- Subtree bounds ---

// SubtreeBounds computes the bounding rectangle of a node and all its
// descendants in the node's local coordinate space.
func SubtreeBounds(n *node.Node) types.Rect {
	var r types.Rect
	first := true
	subtreeBoundsWalk(n, node.IdentityTransform, &r, &first)
	return r
}

func subtreeBoundsWalk(n *node.Node, localTransform [6]float64, bounds *types.Rect, first *bool) {
	var aabb types.Rect
	var hasAABB bool

	if n.Type == types.NodeTypeMesh && n.Mesh != nil {
		aabb = mesh.MeshWorldAABB(n, localTransform, WorldAABB)
		hasAABB = aabb.Width > 0 || aabb.Height > 0
	} else if n.Type == types.NodeTypeParticleEmitter && n.Emitter != nil && n.Emitter.AliveCount() > 0 {
		pb := n.Emitter.Bounds()
		if pb.Width > 0 && pb.Height > 0 {
			if n.Emitter.Config.WorldSpace {
				aabb = pb
			} else {
				aabb = TransformRect(localTransform, pb)
			}
			hasAABB = true
		}
	} else {
		w, h := NodeDimensions(n)
		if w > 0 && h > 0 {
			aabb = WorldAABB(localTransform, w, h)
			hasAABB = true
		}
	}

	if hasAABB {
		if *first {
			*bounds = aabb
			*first = false
		} else {
			*bounds = RectUnion(*bounds, aabb)
		}
	}

	for _, child := range n.Children_ {
		childLocal := node.ComputeLocalTransform(child)
		childTransform := node.MultiplyAffine(localTransform, childLocal)
		subtreeBoundsWalk(child, childTransform, bounds, first)
	}
}

// --- Helpers ---

func nodeColor32(n *node.Node) Color32 {
	return Color32{
		R: float32(n.Color_.R()),
		G: float32(n.Color_.G()),
		B: float32(n.Color_.B()),
		A: float32(n.Color_.A() * n.WorldAlpha),
	}
}

// filterChainPaddingAny computes cumulative filter padding from []any filters.
func filterChainPaddingAny(filters []any) int {
	pad := 0
	for _, f := range filters {
		if ff, ok := f.(filter.Filter); ok {
			pad += ff.Padding()
		}
	}
	return pad
}

// ApplyFiltersAny runs a filter chain on src using []any filters,
// ping-ponging between two images.
func ApplyFiltersAny(filters []any, src *ebiten.Image, pool *RenderTexturePool) *ebiten.Image {
	if len(filters) == 0 {
		return src
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	current := src
	var scratch *ebiten.Image

	for _, f := range filters {
		ff, ok := f.(filter.Filter)
		if !ok {
			continue
		}
		if scratch == nil {
			scratch = pool.Acquire(w, h)
		} else {
			scratch.Clear()
		}
		ff.Apply(current, scratch)
		current, scratch = scratch, current
	}

	return current
}

// WorldAABB computes the axis-aligned bounding box for a rectangle of size
// (w, h) at the origin, transformed by the given affine matrix.
func WorldAABB(transform [6]float64, w, h float64) types.Rect {
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	x0, y0 := tx, ty
	x1, y1 := a*w+tx, b*w+ty
	x2, y2 := c*h+tx, d*h+ty
	x3, y3 := a*w+c*h+tx, b*w+d*h+ty
	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// NodeDimensions returns the width and height used for bounds/culling.
func NodeDimensions(n *node.Node) (w, h float64) {
	switch n.Type {
	case types.NodeTypeSprite:
		if n.CustomImage_ != nil {
			b := n.CustomImage_.Bounds()
			return float64(b.Dx()), float64(b.Dy())
		}
		return float64(n.TextureRegion_.OriginalW), float64(n.TextureRegion_.OriginalH)
	case types.NodeTypeMesh:
		if n.Mesh != nil {
			mesh.RecomputeMeshAABB(n)
			return n.Mesh.Aabb.Width, n.Mesh.Aabb.Height
		}
		return 0, 0
	case types.NodeTypeParticleEmitter:
		if n.Emitter != nil {
			r := &n.Emitter.Config.Region
			return float64(r.OriginalW), float64(r.OriginalH)
		}
		return 0, 0
	case types.NodeTypeText:
		if n.TextBlock != nil {
			tb := n.TextBlock
			tb.Layout()
			return tb.MeasuredW, tb.MeasuredH
		}
		return 0, 0
	default:
		return 0, 0
	}
}

// TransformRect transforms an arbitrary Rect through an affine matrix and
// returns the axis-aligned bounding box of the result.
func TransformRect(t [6]float64, r types.Rect) types.Rect {
	a, b, c, d, tx, ty := t[0], t[2], t[1], t[3], t[4], t[5]
	x0 := a*r.X + b*r.Y + tx
	y0 := c*r.X + d*r.Y + ty
	x1 := a*(r.X+r.Width) + b*r.Y + tx
	y1 := c*(r.X+r.Width) + d*r.Y + ty
	x2 := a*(r.X+r.Width) + b*(r.Y+r.Height) + tx
	y2 := c*(r.X+r.Width) + d*(r.Y+r.Height) + ty
	x3 := a*r.X + b*(r.Y+r.Height) + tx
	y3 := c*r.X + d*(r.Y+r.Height) + ty
	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// RectUnion returns the smallest Rect containing both a and b.
func RectUnion(a, b types.Rect) types.Rect {
	minX := math.Min(a.X, b.X)
	minY := math.Min(a.Y, b.Y)
	maxX := math.Max(a.X+a.Width, b.X+b.Width)
	maxY := math.Max(a.Y+a.Height, b.Y+b.Height)
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}
