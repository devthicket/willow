package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
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

// traverse walks the node tree depth-first, emitting render commands for
// visible, renderable leaf nodes. It is read-only w.r.t. worldTransform  -
// transforms are computed by updateWorldTransform in Update, and traverse
// applies s.viewTransform locally for screen-space output.
func (s *Scene) traverse(n *Node, treeOrder *int) {
	if !n.Visible_ {
		return
	}

	// Compute view-adjusted transform for this node (screen-space).
	viewWorld := multiplyAffine(s.viewTransform, n.WorldTransform)

	// Determine if this node is culled. Culling only suppresses this node's
	// command emission  -  children are ALWAYS traversed because any node type
	// may have children whose world positions differ from the parent's AABB.
	culled := s.cullActive && n.Renderable_ && shouldCull(n, viewWorld, s.cullBounds)

	// CacheAsTree: replay cached commands (hit) or build cache (miss).
	if n.CacheData != nil && !culled {
		containerTransform32 := affine32(viewWorld)
		containerAlpha := float32(n.WorldAlpha)
		ct := getCacheTree(n)

		if !ct.Dirty && len(ct.Commands) > 0 {
			// Cache hit  -  delta remap and replay.
			s.replayCacheAsTree(n, containerTransform32, containerAlpha, treeOrder)
			return
		}
		// Cache miss  -  traverse normally, then capture.
		s.buildCacheAsTree(n, containerTransform32, containerAlpha, treeOrder)
		return
	}

	// Special path: nodes with masks, cache, or filters render their subtree
	// to an offscreen image and emit a single directImage command.
	if !culled && (n.MaskNode != nil || n.CacheEnabled || len(n.Filters) > 0) {
		s.renderSpecialNode(n, treeOrder)
		return
	}

	// Custom emit hook (used by TileMapLayer for CommandTilemap).
	if n.CustomEmit != nil && !culled {
		n.CustomEmit(s, treeOrder)
		s.commandsDirtyThisFrame = true
		// Still traverse children (sandwich layers).
		if len(n.Children_) > 0 {
			children := n.Children_
			if !n.ChildrenSorted {
				s.rebuildSortedChildren(n)
			}
			if n.SortedChildren != nil {
				children = n.SortedChildren
			}
			for _, child := range children {
				s.traverse(child, treeOrder)
			}
		}
		return
	}

	// Track whether we're building under a CacheAsTree ancestor.
	building := s.buildingCacheFor != nil

	// Emit command for renderable leaf-type nodes
	preCmdLen := len(s.commands)
	if n.Renderable_ && !culled {
		switch n.Type {
		case NodeTypeSprite:
			*treeOrder++
			cmd := RenderCommand{
				Type:        CommandSprite,
				Transform:   affine32(viewWorld),
				Color:       color32{R: float32(n.Color_.R()), G: float32(n.Color_.G()), B: float32(n.Color_.B()), A: float32(n.Color_.A() * n.WorldAlpha)},
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
			s.commands = append(s.commands, cmd)
		case NodeTypeMesh:
			if len(n.Mesh.Vertices) == 0 || len(n.Mesh.Indices) == 0 {
				break
			}
			tintColor := RGBA(n.Color_.R(), n.Color_.G(), n.Color_.B(), n.Color_.A()*n.WorldAlpha)
			dst := ensureTransformedVerts(n)
			transformVertices(n.Mesh.Vertices, dst, viewWorld, tintColor)
			*treeOrder++
			s.commands = append(s.commands, RenderCommand{
				Type:        CommandMesh,
				Transform:   affine32(viewWorld),
				BlendMode:   n.BlendMode_,
				RenderLayer: n.RenderLayer,
				GlobalOrder: n.GlobalOrder,
				TreeOrder:   *treeOrder,
				MeshVerts:   dst,
				MeshInds:    n.Mesh.Indices,
				MeshImage:   n.Mesh.Image,
			})
		case NodeTypeParticleEmitter:
			if n.Emitter != nil && n.Emitter.Alive > 0 {
				*treeOrder++
				particleTransform := viewWorld
				ws := n.Emitter.Config.WorldSpace
				if ws {
					particleTransform = s.viewTransform
				}
				s.commands = append(s.commands, RenderCommand{
					Type:               CommandParticle,
					Transform:          affine32(particleTransform),
					TextureRegion:      n.TextureRegion_,
					DirectImage:        n.CustomImage_,
					Color:              color32{R: float32(n.Color_.R()), G: float32(n.Color_.G()), B: float32(n.Color_.B()), A: float32(n.Color_.A() * n.WorldAlpha)},
					BlendMode:          n.BlendMode_,
					RenderLayer:        n.RenderLayer,
					GlobalOrder:        n.GlobalOrder,
					TreeOrder:          *treeOrder,
					Emitter:            n.Emitter,
					WorldSpaceParticle: ws,
				})
			}
		case NodeTypeText:
			if n.TextBlock != nil && n.TextBlock.Font != nil {
				switch n.TextBlock.Font.(type) {
				case *SpriteFont:
					s.commands = emitSDFTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
				case *PixelFont:
					s.commands = emitPixelTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
				}
			}
			// NodeTypeContainer doesn't emit commands
		}
	}
	if !building && len(s.commands) > preCmdLen {
		s.commandsDirtyThisFrame = true
	}

	// Traverse children (ZIndex sorted if needed)
	if len(n.Children_) == 0 {
		return
	}
	children := n.Children_
	if !n.ChildrenSorted {
		s.rebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		s.traverse(child, treeOrder)
	}
}

// rebuildSortedChildren rebuilds the ZIndex-sorted traversal order for a node.
// Uses insertion sort: zero allocations, stable, and optimal for the typical
// case of few children that are nearly sorted (O(n) when already sorted).
func (s *Scene) rebuildSortedChildren(n *Node) {
	nc := len(n.Children_)
	if cap(n.SortedChildren) < nc {
		n.SortedChildren = make([]*Node, nc)
	}
	n.SortedChildren = n.SortedChildren[:nc]
	copy(n.SortedChildren, n.Children_)
	// Stable insertion sort by ZIndex.
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

// --- Merge sort ---

// commandLessOrEqual delegates to render.CommandLessOrEqual.
func commandLessOrEqual(a, b RenderCommand) bool {
	return render.CommandLessOrEqual(a, b)
}

// mergeSort sorts s.commands in-place using s.sortBuf as scratch space.
func (s *Scene) mergeSort() {
	render.MergeSort(s.commands, &s.sortBuf)
}

// --- Special node rendering (Phase 09) ---

// renderSpecialNode handles nodes with masks, cache, or filters.
// Processing order: bounds → adjust transform → cache check → render subtree → apply mask → apply filters → cache store → emit command.
func (s *Scene) renderSpecialNode(n *Node, treeOrder *int) {
	// Compute bounds up-front  -  needed by all paths to build the correct
	// screen-space transform.  RT pixel (0,0) corresponds to local
	// (bounds.X, bounds.Y), not local (0,0), so the world transform must be
	// shifted by the world-space equivalent of that offset.
	bounds := subtreeBounds(n)
	padding := filterChainPadding(n.Filters)
	bounds.X -= float64(padding)
	bounds.Y -= float64(padding)
	bounds.Width += float64(padding * 2)
	bounds.Height += float64(padding * 2)

	// adjustedTransform places RT(0,0) = local(bounds.X, bounds.Y) at the
	// correct screen position: screen(tx + a*bX + c*bY, ty + b*bX + d*bY).
	viewWorld := multiplyAffine(s.viewTransform, n.WorldTransform)
	bX, bY := bounds.X, bounds.Y
	a, b, c, d := viewWorld[0], viewWorld[1], viewWorld[2], viewWorld[3]
	adjustedTransform := viewWorld
	adjustedTransform[4] += a*bX + c*bY
	adjustedTransform[5] += b*bX + d*bY

	// Cache hit: reuse existing cached texture.
	at32 := affine32(adjustedTransform)
	if n.CacheEnabled && n.CacheTexture != nil && !n.CacheDirty {
		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       color32{R: 1, G: 1, B: 1, A: float32(n.WorldAlpha)},
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

	// Render subtree to offscreen.
	rt := s.rtPool.Acquire(w, h)
	renderSubtree(s, n, rt, bounds)
	result := rt

	// Apply mask if present.
	if n.MaskNode != nil {
		maskRT := s.rtPool.Acquire(w, h)
		renderSubtree(s, n.MaskNode, maskRT, bounds)

		// Composite: keep only the parts of result where mask has alpha.
		var op ebiten.DrawImageOptions
		op.Blend = BlendMask.EbitenBlend()
		result.DrawImage(maskRT, &op)

		s.rtPool.Release(maskRT)
	}

	// Apply filter chain.
	if len(n.Filters) > 0 {
		filtered := applyFilters(n.Filters, result, &s.rtPool)
		if filtered != result {
			// The old result RT needs to be released.
			s.rtPool.Release(result)
			result = filtered
		}
	}

	// Cache the result if caching is enabled.
	if n.CacheEnabled {
		// Dispose old cache texture if present.
		if n.CacheTexture != nil {
			n.CacheTexture.Deallocate()
		}
		// Copy result to a non-pooled texture for caching.
		cacheImg := ebiten.NewImage(w, h)
		var op ebiten.DrawImageOptions
		cacheImg.DrawImage(result, &op)
		n.CacheTexture = cacheImg
		n.CacheDirty = false

		// Release the pooled RT immediately since we copied to cache.
		s.rtPool.Release(result)

		// Emit using the cached texture.
		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       color32{R: 1, G: 1, B: 1, A: float32(n.WorldAlpha)},
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
			DirectImage: n.CacheTexture,
		})
		return
	}

	// Not cached: defer release until after submitBatches.
	s.rtDeferred = append(s.rtDeferred, result)

	*treeOrder++
	s.commands = append(s.commands, RenderCommand{
		Type:                 CommandSprite,
		Transform:            at32,
		Color:                color32{R: 1, G: 1, B: 1, A: float32(n.WorldAlpha)},
		BlendMode:            n.BlendMode_,
		RenderLayer:          n.RenderLayer,
		GlobalOrder:          n.GlobalOrder,
		TreeOrder:            *treeOrder,
		DirectImage:          result,
		TransientDirectImage: true,
	})
}

// --- Subtree command caching (CacheAsTree) ---

// multiplyAffine32 delegates to render.MultiplyAffine32.
func multiplyAffine32(p, c [6]float32) [6]float32 {
	return render.MultiplyAffine32(p, c)
}

// invertAffine32 delegates to render.InvertAffine32.
func invertAffine32(m [6]float32) [6]float32 {
	return render.InvertAffine32(m)
}

// replayCacheAsTree replays cached commands with delta remap for transform
// and alpha changes. Animated tiles read live TextureRegion from source node.
func (s *Scene) replayCacheAsTree(n *Node, containerTransform32 [6]float32, containerAlpha float32, treeOrder *int) {
	ct := getCacheTree(n)
	transformChanged := containerTransform32 != ct.ParentTransform
	alphaChanged := containerAlpha != ct.ParentAlpha

	var delta [6]float32
	if transformChanged {
		inv := invertAffine32(ct.ParentTransform)
		delta = multiplyAffine32(containerTransform32, inv)
	}
	var alphaRatio float32 = 1.0
	if alphaChanged && ct.ParentAlpha > 1e-6 {
		alphaRatio = containerAlpha / ct.ParentAlpha
	}

	for i := range ct.Commands {
		src := &ct.Commands[i]
		cmd := src.Cmd // copy from cache
		if transformChanged {
			cmd.Transform = multiplyAffine32(delta, cmd.Transform)
		}
		if alphaChanged {
			cmd.Color.R *= alphaRatio
			cmd.Color.G *= alphaRatio
			cmd.Color.B *= alphaRatio
			cmd.Color.A *= alphaRatio
		}
		// Two-tier texture: nil source = static, non-nil = animated
		if src.Source != nil {
			cmd.TextureRegion = src.Source.TextureRegion_
		}
		*treeOrder++
		cmd.TreeOrder = *treeOrder
		s.commands = append(s.commands, cmd)
	}

	// NOTE: Do NOT update cachedParentTransform/cachedParentAlpha here.
	// The cached cmd.Transform values are always from build time, so the
	// delta must always be relative to the build-time parent transform.
}

// buildCacheAsTree traverses the container's subtree normally, captures
// the emitted commands, and stores them for future replay.
func (s *Scene) buildCacheAsTree(n *Node, containerTransform32 [6]float32, containerAlpha float32, treeOrder *int) {
	startIdx := len(s.commands)
	s.commandsDirtyThisFrame = true

	// Set flag so traverse tags emitted commands with emittingNodeID.
	prevBuilding := s.buildingCacheFor
	s.buildingCacheFor = n

	// Emit the container's own command if renderable.
	viewWorld := multiplyAffine(s.viewTransform, n.WorldTransform)
	culled := s.cullActive && n.Renderable_ && shouldCull(n, viewWorld, s.cullBounds)
	if n.Renderable_ && !culled {
		s.emitNodeCommandInline(n, treeOrder)
		// Tag the just-emitted command(s) with the node ID.
		for i := startIdx; i < len(s.commands); i++ {
			s.commands[i].EmittingNodeID = n.ID
		}
	}

	// Traverse children with culling disabled  -  the cache must capture ALL
	// children regardless of current viewport so panning replays correctly.
	prevCull := s.cullActive
	s.cullActive = false
	if len(n.Children_) > 0 {
		children := n.Children_
		if !n.ChildrenSorted {
			s.rebuildSortedChildren(n)
		}
		if n.SortedChildren != nil {
			children = n.SortedChildren
		}
		for _, child := range children {
			s.traverse(child, treeOrder)
		}
	}
	s.cullActive = prevCull

	// Restore building flag.
	s.buildingCacheFor = prevBuilding

	// Capture emitted commands. Check for uncacheable types.
	newCmds := s.commands[startIdx:]
	blocked := false
	for i := range newCmds {
		cmd := &newCmds[i]
		if cmd.Type == CommandMesh || cmd.Type == CommandParticle || cmd.TransientDirectImage {
			blocked = true
			break
		}
	}

	ct := getCacheTree(n)
	if blocked {
		// Can't cache  -  disable and fall through. Commands are already emitted.
		ct.Dirty = true
		return
	}

	// Store in cachedCommands.
	if cap(ct.Commands) < len(newCmds) {
		ct.Commands = make([]cachedCmd, len(newCmds))
	}
	ct.Commands = ct.Commands[:len(newCmds)]

	for i := range newCmds {
		ct.Commands[i] = cachedCmd{
			Cmd:          newCmds[i],
			Source:       nil, // static by default (two-tier)
			SourceNodeID: newCmds[i].EmittingNodeID,
		}
	}

	ct.ParentTransform = containerTransform32
	ct.ParentAlpha = containerAlpha
	ct.Dirty = false
}

// emitNodeCommandInline emits a command for the node itself (used by buildCacheAsTree
// when the cached container is also renderable).
func (s *Scene) emitNodeCommandInline(n *Node, treeOrder *int) {
	viewWorld := multiplyAffine(s.viewTransform, n.WorldTransform)
	switch n.Type {
	case NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   affine32(viewWorld),
			Color:       color32{R: float32(n.Color_.R()), G: float32(n.Color_.G()), B: float32(n.Color_.B()), A: float32(n.Color_.A() * n.WorldAlpha)},
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
		s.commands = append(s.commands, cmd)
	case NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			switch n.TextBlock.Font.(type) {
			case *SpriteFont:
				s.commands = emitSDFTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
			case *PixelFont:
				s.commands = emitPixelTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
			}
		}
	}
	// Mesh/Particle handled as "blocked"  -  won't reach here during cache build
}
