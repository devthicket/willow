package willow

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Render texture pool ---

// renderTexturePool manages reusable offscreen ebiten.Images keyed by
// power-of-two dimensions. After warmup, Acquire/Release are zero-alloc.
type renderTexturePool struct {
	buckets map[uint64][]*ebiten.Image
}

// poolKey packs power-of-two width and height into a single uint64.
func poolKey(w, h int) uint64 {
	return uint64(w)<<32 | uint64(h)
}

// Acquire returns a cleared offscreen image with at least (w, h) pixels.
// Dimensions are rounded up to the next power of two.
func (p *renderTexturePool) Acquire(w, h int) *ebiten.Image {
	pw := nextPowerOfTwo(w)
	ph := nextPowerOfTwo(h)
	key := poolKey(pw, ph)

	if p.buckets != nil {
		if stack := p.buckets[key]; len(stack) > 0 {
			img := stack[len(stack)-1]
			p.buckets[key] = stack[:len(stack)-1]
			img.Clear()
			return img
		}
	}

	return ebiten.NewImageWithOptions(
		image.Rect(0, 0, pw, ph),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
}

// Release returns an image to the pool for reuse. The image is cleared on
// next Acquire, not here (avoids redundant GPU work if released then
// immediately re-acquired).
func (p *renderTexturePool) Release(img *ebiten.Image) {
	if img == nil {
		return
	}
	b := img.Bounds()
	key := poolKey(b.Dx(), b.Dy())

	if p.buckets == nil {
		p.buckets = make(map[uint64][]*ebiten.Image)
	}
	p.buckets[key] = append(p.buckets[key], img)
}

// nextPowerOfTwo returns the smallest power of two >= n (minimum 1).
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	// Use float64 log2 then ceil, convert back.
	return 1 << int(math.Ceil(math.Log2(float64(n))))
}

// ToTexture renders a node's subtree to a new offscreen image and returns it.
// The caller owns the returned image (it is NOT pooled). Requires a Scene
// reference to use the render pipeline.
func ToTexture(n *Node, s *Scene) *ebiten.Image {
	bounds := subtreeBounds(n)
	w := int(math.Ceil(bounds.Width))
	h := int(math.Ceil(bounds.Height))
	if w <= 0 || h <= 0 {
		return ebiten.NewImage(1, 1)
	}
	img := ebiten.NewImage(w, h)
	renderSubtree(s, n, img, bounds)
	return img
}

// --- Subtree bounds ---

// subtreeBounds computes the bounding rectangle of a node and all its
// descendants in the node's local coordinate space.
func subtreeBounds(n *Node) Rect {
	var r Rect
	first := true
	subtreeBoundsWalk(n, identityTransform, &r, &first)
	return r
}

// subtreeBoundsWalk recursively accumulates bounds.
func subtreeBoundsWalk(n *Node, localTransform [6]float64, bounds *Rect, first *bool) {
	var aabb Rect
	var hasAABB bool

	if n.Type == NodeTypeMesh {
		aabb = meshWorldAABB(n, localTransform)
		hasAABB = aabb.Width > 0 || aabb.Height > 0
	} else if n.Type == NodeTypeParticleEmitter && n.Emitter != nil && n.Emitter.Alive > 0 {
		pb := n.Emitter.Bounds()
		if pb.Width > 0 && pb.Height > 0 {
			if n.Emitter.Config.WorldSpace {
				// Particle positions are already in world space; the
				// localTransform only contributes the emitter's offset.
				// Use the raw particle bounds directly.
				aabb = pb
			} else {
				// Local-space particles: transform the bounds rectangle
				// through the emitter's local transform.
				aabb = transformRect(localTransform, pb)
			}
			hasAABB = true
		}
	} else {
		w, h := nodeDimensions(n)
		if w > 0 && h > 0 {
			aabb = worldAABB(localTransform, w, h)
			hasAABB = true
		}
	}

	if hasAABB {
		if *first {
			*bounds = aabb
			*first = false
		} else {
			*bounds = rectUnion(*bounds, aabb)
		}
	}

	for _, child := range n.Children_ {
		childLocal := computeLocalTransform(child)
		childTransform := multiplyAffine(localTransform, childLocal)
		subtreeBoundsWalk(child, childTransform, bounds, first)
	}
}

// transformRect transforms an arbitrary Rect through an affine matrix and
// returns the axis-aligned bounding box of the result.
func transformRect(t [6]float64, r Rect) Rect {
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
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// rectUnion returns the smallest Rect containing both a and b.
func rectUnion(a, b Rect) Rect {
	minX := math.Min(a.X, b.X)
	minY := math.Min(a.Y, b.Y)
	maxX := math.Max(a.X+a.Width, b.X+b.Width)
	maxY := math.Max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// --- Subtree rendering ---

// renderSubtree renders a node and its children to the given target image.
// It temporarily swaps the scene's command buffer to avoid disturbing the
// main render pass. The node's content is rendered at local-space origin,
// offset by -bounds.X, -bounds.Y so everything fits in the target.
func renderSubtree(s *Scene, n *Node, target *ebiten.Image, bounds Rect) {
	// Save main command buffer.
	savedCmds := s.commands
	s.commands = s.offscreenCmds[:0]

	// Build an offset transform so the subtree content starts at (0,0) in the target.
	offsetTransform := [6]float64{1, 0, 0, 1, -bounds.X, -bounds.Y}

	treeOrder := 0

	// Emit the node itself if renderable.
	// Use alpha=1.0 as the base; worldAlpha is applied once by the final
	// composite command in renderSpecialNode (avoids double-application).
	emitNodeCommand(s, n, offsetTransform, 1.0, &treeOrder)

	// Traverse children using ZIndex-sorted order when available.
	children := n.Children_
	if !n.ChildrenSorted {
		s.rebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		renderSubtreeWalk(s, child, offsetTransform, 1.0, &treeOrder)
	}

	// Sort and submit to offscreen target.
	s.mergeSort()
	s.submitBatches(target)

	// Restore. Keep offscreenCmds at high-water capacity.
	s.offscreenCmds = s.commands[:0]
	s.commands = savedCmds
}

// renderSubtreeWalk traverses a node subtree, emitting commands into the
// current s.commands buffer. Similar to traverse() but uses explicit
// transforms rather than world transforms.
func renderSubtreeWalk(s *Scene, n *Node, parentTransform [6]float64, parentAlpha float64, treeOrder *int) {
	if !n.Visible_ {
		return
	}

	local := computeLocalTransform(n)
	transform := multiplyAffine(parentTransform, local)
	alpha := parentAlpha * n.Alpha_

	// Nested special node (mask, cache, or filter): render it to its own RT
	// and emit a command using the computed local transform.
	if n.MaskNode != nil || n.CacheEnabled || len(n.Filters) > 0 {
		renderSpecialSubtreeNode(s, n, transform, alpha, treeOrder)
		return
	}

	emitNodeCommand(s, n, transform, alpha, treeOrder)

	// Use ZIndex-sorted children order, consistent with main traverse.
	children := n.Children_
	if !n.ChildrenSorted {
		s.rebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		renderSubtreeWalk(s, child, transform, alpha, treeOrder)
	}
}

// renderSpecialSubtreeNode handles a masked/cached/filtered node encountered
// inside a subtree rendering pass. It mirrors renderSpecialNode but uses an
// explicit local transform instead of n.WorldTransform.
func renderSpecialSubtreeNode(s *Scene, n *Node, localTransform [6]float64, alpha float64, treeOrder *int) {
	bounds := subtreeBounds(n)
	padding := filterChainPadding(n.Filters)
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

	rt := s.rtPool.Acquire(w, h)
	renderSubtree(s, n, rt, bounds)
	result := rt

	if n.MaskNode != nil {
		maskRT := s.rtPool.Acquire(w, h)
		renderSubtree(s, n.MaskNode, maskRT, bounds)
		var op ebiten.DrawImageOptions
		op.Blend = BlendMask.EbitenBlend()
		result.DrawImage(maskRT, &op)
		s.rtPool.Release(maskRT)
	}

	if len(n.Filters) > 0 {
		filtered := applyFilters(n.Filters, result, &s.rtPool)
		if filtered != result {
			s.rtPool.Release(result)
			result = filtered
		}
	}

	s.rtDeferred = append(s.rtDeferred, result)
	*treeOrder++
	s.commands = append(s.commands, RenderCommand{
		Type:        CommandSprite,
		Transform:   affine32(adjustedTransform),
		Color:       color32{1, 1, 1, float32(alpha)},
		BlendMode:   n.BlendMode_,
		RenderLayer: n.RenderLayer,
		GlobalOrder: n.GlobalOrder,
		treeOrder:   *treeOrder,
		directImage: result,
	})
}

// emitNodeCommand emits a render command for a single node at the given transform.
func emitNodeCommand(s *Scene, n *Node, transform [6]float64, alpha float64, treeOrder *int) {
	if !n.Renderable_ {
		return
	}
	t32 := affine32(transform)
	switch n.Type {
	case NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   t32,
			Color:       color32{float32(n.Color_.R()), float32(n.Color_.G()), float32(n.Color_.B()), float32(n.Color_.A() * alpha)},
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			treeOrder:   *treeOrder,
		}
		if n.CustomImage_ != nil {
			cmd.directImage = n.CustomImage_
		} else {
			cmd.TextureRegion = n.TextureRegion_
		}
		s.commands = append(s.commands, cmd)
	case NodeTypeMesh:
		if len(n.Mesh.Vertices) == 0 || len(n.Mesh.Indices) == 0 {
			return
		}
		tintColor := RGBA(n.Color_.R(), n.Color_.G(), n.Color_.B(), n.Color_.A()*alpha)
		dst := ensureTransformedVerts(n)
		transformVertices(n.Mesh.Vertices, dst, transform, tintColor)
		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:        CommandMesh,
			Transform:   t32,
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			treeOrder:   *treeOrder,
			meshVerts:   dst,
			meshInds:    n.Mesh.Indices,
			meshImage:   n.Mesh.Image,
		})
	case NodeTypeParticleEmitter:
		if n.Emitter != nil && n.Emitter.Alive > 0 {
			*treeOrder++
			particleTransform := transform
			ws := n.Emitter.Config.WorldSpace
			if ws {
				particleTransform = s.viewTransform
			}
			s.commands = append(s.commands, RenderCommand{
				Type:               CommandParticle,
				Transform:          affine32(particleTransform),
				TextureRegion:      n.TextureRegion_,
				directImage:        n.CustomImage_,
				Color:              color32{float32(n.Color_.R()), float32(n.Color_.G()), float32(n.Color_.B()), float32(n.Color_.A() * alpha)},
				BlendMode:          n.BlendMode_,
				RenderLayer:        n.RenderLayer,
				GlobalOrder:        n.GlobalOrder,
				treeOrder:          *treeOrder,
				emitter:            n.Emitter,
				worldSpaceParticle: ws,
			})
		}
	case NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			s.commands = emitSDFTextCommand(n.TextBlock, n, transform, s.commands, treeOrder)
		}
	}
}
