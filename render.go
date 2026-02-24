package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// CommandType identifies the kind of render command.
type CommandType uint8

const (
	CommandSprite   CommandType = iota // DrawImage
	CommandMesh                        // DrawTriangles
	CommandParticle                    // particle quads (batches as sprites)
	CommandTilemap                     // DrawTriangles for tilemap layers
)

// color32 is a compact RGBA color using float32, for render commands only.
type color32 struct {
	R, G, B, A float32
}

// RenderCommand is a single draw instruction emitted during scene traversal.
type RenderCommand struct {
	Type          CommandType
	Transform     [6]float32
	TextureRegion TextureRegion
	Color         color32
	BlendMode     BlendMode
	ShaderID      uint16
	TargetID      uint16
	RenderLayer   uint8
	GlobalOrder   int
	treeOrder     int // assigned during traversal for stable sort

	// Mesh-only fields (slice headers, not copies of vertex data).
	meshVerts []ebiten.Vertex
	meshInds  []uint16
	meshImage *ebiten.Image

	// directImage, when non-nil, is drawn directly instead of looking up an
	// atlas page. Used for cached/filtered/masked node output (Phase 09).
	directImage *ebiten.Image

	// emitter references the particle emitter for CommandParticle commands.
	emitter            *ParticleEmitter
	worldSpaceParticle bool // particles store world positions; Transform is view-only

	// transientDirectImage is true when directImage points to a pooled RT
	// that is released after the frame. Such commands cannot be static-cached
	// because the image pointer becomes stale on replay.
	transientDirectImage bool

	// emittingNodeID is set during CacheAsTree builds to track which Node
	// emitted this command. Only populated when building under a cached ancestor.
	emittingNodeID uint32

	// Tilemap fields (CommandTilemap only — slice headers, not copies).
	tilemapVerts []ebiten.Vertex
	tilemapInds  []uint16
	tilemapImage *ebiten.Image
}

// identityTransform32 is the identity affine matrix as float32.
var identityTransform32 = [6]float32{1, 0, 0, 1, 0, 0}

// affine32 converts a [6]float64 affine matrix to [6]float32.
func affine32(m [6]float64) [6]float32 {
	return [6]float32{float32(m[0]), float32(m[1]), float32(m[2]), float32(m[3]), float32(m[4]), float32(m[5])}
}

// traverse walks the node tree depth-first, emitting render commands for
// visible, renderable leaf nodes. It is read-only w.r.t. worldTransform —
// transforms are computed by updateWorldTransform in Update, and traverse
// applies s.viewTransform locally for screen-space output.
func (s *Scene) traverse(n *Node, treeOrder *int) {
	if !n.Visible {
		return
	}

	// Compute view-adjusted transform for this node (screen-space).
	viewWorld := multiplyAffine(s.viewTransform, n.worldTransform)

	// Determine if this node is culled. Culling only suppresses this node's
	// command emission — children are ALWAYS traversed because any node type
	// may have children whose world positions differ from the parent's AABB.
	culled := s.cullActive && n.Renderable && shouldCull(n, viewWorld, s.cullBounds)

	// CacheAsTree: replay cached commands (hit) or build cache (miss).
	if n.cacheTreeEnabled && !culled {
		containerTransform32 := affine32(viewWorld)
		containerAlpha := float32(n.worldAlpha)

		if !n.cacheTreeDirty && len(n.cachedCommands) > 0 {
			// Cache hit — delta remap and replay.
			s.replayCacheAsTree(n, containerTransform32, containerAlpha, treeOrder)
			return
		}
		// Cache miss — traverse normally, then capture.
		s.buildCacheAsTree(n, containerTransform32, containerAlpha, treeOrder)
		return
	}

	// Special path: nodes with masks, cache, or filters render their subtree
	// to an offscreen image and emit a single directImage command.
	if !culled && (n.mask != nil || n.cacheEnabled || len(n.Filters) > 0) {
		s.renderSpecialNode(n, treeOrder)
		return
	}

	// Custom emit hook (used by TileMapLayer for CommandTilemap).
	if n.customEmit != nil && !culled {
		n.customEmit(s, treeOrder)
		s.commandsDirtyThisFrame = true
		// Still traverse children (sandwich layers).
		if len(n.children) > 0 {
			children := n.children
			if !n.childrenSorted {
				s.rebuildSortedChildren(n)
			}
			if n.sortedChildren != nil {
				children = n.sortedChildren
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
	if n.Renderable && !culled {
		switch n.Type {
		case NodeTypeSprite:
			*treeOrder++
			cmd := RenderCommand{
				Type:        CommandSprite,
				Transform:   affine32(viewWorld),
				Color:       color32{float32(n.Color.R), float32(n.Color.G), float32(n.Color.B), float32(n.Color.A * n.worldAlpha)},
				BlendMode:   n.BlendMode,
				RenderLayer: n.RenderLayer,
				GlobalOrder: n.GlobalOrder,
				treeOrder:   *treeOrder,
			}
			if n.customImage != nil {
				cmd.directImage = n.customImage
			} else {
				cmd.TextureRegion = n.TextureRegion
			}
			if building {
				cmd.emittingNodeID = n.ID
			}
			s.commands = append(s.commands, cmd)
		case NodeTypeMesh:
			if len(n.Vertices) == 0 || len(n.Indices) == 0 {
				break
			}
			tintColor := Color{n.Color.R, n.Color.G, n.Color.B, n.Color.A * n.worldAlpha}
			dst := ensureTransformedVerts(n)
			transformVertices(n.Vertices, dst, viewWorld, tintColor)
			*treeOrder++
			s.commands = append(s.commands, RenderCommand{
				Type:        CommandMesh,
				Transform:   affine32(viewWorld),
				BlendMode:   n.BlendMode,
				RenderLayer: n.RenderLayer,
				GlobalOrder: n.GlobalOrder,
				treeOrder:   *treeOrder,
				meshVerts:   dst,
				meshInds:    n.Indices,
				meshImage:   n.MeshImage,
			})
		case NodeTypeParticleEmitter:
			if n.Emitter != nil && n.Emitter.alive > 0 {
				*treeOrder++
				particleTransform := viewWorld
				ws := n.Emitter.config.WorldSpace
				if ws {
					particleTransform = s.viewTransform
				}
				s.commands = append(s.commands, RenderCommand{
					Type:               CommandParticle,
					Transform:          affine32(particleTransform),
					TextureRegion:      n.TextureRegion,
					directImage:        n.customImage,
					Color:              color32{float32(n.Color.R), float32(n.Color.G), float32(n.Color.B), float32(n.Color.A * n.worldAlpha)},
					BlendMode:          n.BlendMode,
					RenderLayer:        n.RenderLayer,
					GlobalOrder:        n.GlobalOrder,
					treeOrder:          *treeOrder,
					emitter:            n.Emitter,
					worldSpaceParticle: ws,
				})
			}
		case NodeTypeText:
			if n.TextBlock != nil && n.TextBlock.Font != nil {
				s.commands = emitSDFTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
			}
			// NodeTypeContainer doesn't emit commands
		}
	}
	if !building && len(s.commands) > preCmdLen {
		s.commandsDirtyThisFrame = true
	}

	// Traverse children (ZIndex sorted if needed)
	if len(n.children) == 0 {
		return
	}
	children := n.children
	if !n.childrenSorted {
		s.rebuildSortedChildren(n)
	}
	if n.sortedChildren != nil {
		children = n.sortedChildren
	}
	for _, child := range children {
		s.traverse(child, treeOrder)
	}
}

// rebuildSortedChildren rebuilds the ZIndex-sorted traversal order for a node.
// Uses insertion sort: zero allocations, stable, and optimal for the typical
// case of few children that are nearly sorted (O(n) when already sorted).
func (s *Scene) rebuildSortedChildren(n *Node) {
	nc := len(n.children)
	if cap(n.sortedChildren) < nc {
		n.sortedChildren = make([]*Node, nc)
	}
	n.sortedChildren = n.sortedChildren[:nc]
	copy(n.sortedChildren, n.children)
	// Stable insertion sort by ZIndex.
	for i := 1; i < nc; i++ {
		key := n.sortedChildren[i]
		j := i - 1
		for j >= 0 && n.sortedChildren[j].ZIndex > key.ZIndex {
			n.sortedChildren[j+1] = n.sortedChildren[j]
			j--
		}
		n.sortedChildren[j+1] = key
	}
	n.childrenSorted = true
}

// --- Merge sort ---

// commandLessOrEqual returns true if a should sort before or at the same position as b.
// Using <= for treeOrder ensures stability.
func commandLessOrEqual(a, b RenderCommand) bool {
	if a.RenderLayer != b.RenderLayer {
		return a.RenderLayer < b.RenderLayer
	}
	if a.GlobalOrder != b.GlobalOrder {
		return a.GlobalOrder < b.GlobalOrder
	}
	return a.treeOrder <= b.treeOrder
}

// mergeSort sorts s.commands in-place using s.sortBuf as scratch space.
// Bottom-up merge sort: zero allocations after the sort buffer reaches high-water mark.
func (s *Scene) mergeSort() {
	n := len(s.commands)
	if n <= 1 {
		return
	}

	// Quick O(n) check: if commands are already sorted, skip the full sort.
	// This is the common case for static scenes where traverse emits commands
	// in the same order every frame (Pixi v8's structureDidChange pattern).
	sorted := true
	for i := 1; i < n; i++ {
		if !commandLessOrEqual(s.commands[i-1], s.commands[i]) {
			sorted = false
			break
		}
	}
	if sorted {
		return
	}

	if cap(s.sortBuf) < n {
		s.sortBuf = make([]RenderCommand, n)
	}
	s.sortBuf = s.sortBuf[:n]

	a := s.commands
	b := s.sortBuf
	swapped := false

	for width := 1; width < n; width *= 2 {
		for i := 0; i < n; i += 2 * width {
			lo := i
			mid := lo + width
			if mid > n {
				mid = n
			}
			hi := lo + 2*width
			if hi > n {
				hi = n
			}
			mergeRun(a, b, lo, mid, hi)
		}
		a, b = b, a
		swapped = !swapped
	}

	if swapped {
		copy(s.commands, s.sortBuf)
	}
}

// mergeRun merges two sorted runs [lo, mid) and [mid, hi) from src into dst.
func mergeRun(src, dst []RenderCommand, lo, mid, hi int) {
	i, j, k := lo, mid, lo
	for i < mid && j < hi {
		if commandLessOrEqual(src[i], src[j]) {
			dst[k] = src[i]
			i++
		} else {
			dst[k] = src[j]
			j++
		}
		k++
	}
	for i < mid {
		dst[k] = src[i]
		i++
		k++
	}
	for j < hi {
		dst[k] = src[j]
		j++
		k++
	}
}

// --- Special node rendering (Phase 09) ---

// renderSpecialNode handles nodes with masks, cache, or filters.
// Processing order: bounds → adjust transform → cache check → render subtree → apply mask → apply filters → cache store → emit command.
func (s *Scene) renderSpecialNode(n *Node, treeOrder *int) {
	// Compute bounds up-front — needed by all paths to build the correct
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
	viewWorld := multiplyAffine(s.viewTransform, n.worldTransform)
	bX, bY := bounds.X, bounds.Y
	a, b, c, d := viewWorld[0], viewWorld[1], viewWorld[2], viewWorld[3]
	adjustedTransform := viewWorld
	adjustedTransform[4] += a*bX + c*bY
	adjustedTransform[5] += b*bX + d*bY

	// Cache hit: reuse existing cached texture.
	at32 := affine32(adjustedTransform)
	if n.cacheEnabled && n.cacheTexture != nil && !n.cacheDirty {
		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       color32{1, 1, 1, float32(n.worldAlpha)},
			BlendMode:   n.BlendMode,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			treeOrder:   *treeOrder,
			directImage: n.cacheTexture,
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
	if n.mask != nil {
		maskRT := s.rtPool.Acquire(w, h)
		renderSubtree(s, n.mask, maskRT, bounds)

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
	if n.cacheEnabled {
		// Dispose old cache texture if present.
		if n.cacheTexture != nil {
			n.cacheTexture.Deallocate()
		}
		// Copy result to a non-pooled texture for caching.
		cacheImg := ebiten.NewImage(w, h)
		var op ebiten.DrawImageOptions
		cacheImg.DrawImage(result, &op)
		n.cacheTexture = cacheImg
		n.cacheDirty = false

		// Release the pooled RT immediately since we copied to cache.
		s.rtPool.Release(result)

		// Emit using the cached texture.
		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:        CommandSprite,
			Transform:   at32,
			Color:       color32{1, 1, 1, float32(n.worldAlpha)},
			BlendMode:   n.BlendMode,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			treeOrder:   *treeOrder,
			directImage: n.cacheTexture,
		})
		return
	}

	// Not cached: defer release until after submitBatches.
	s.rtDeferred = append(s.rtDeferred, result)

	*treeOrder++
	s.commands = append(s.commands, RenderCommand{
		Type:                 CommandSprite,
		Transform:            at32,
		Color:                color32{1, 1, 1, float32(n.worldAlpha)},
		BlendMode:            n.BlendMode,
		RenderLayer:          n.RenderLayer,
		GlobalOrder:          n.GlobalOrder,
		treeOrder:            *treeOrder,
		directImage:          result,
		transientDirectImage: true,
	})
}

// --- Subtree command caching (CacheAsTree) ---

// cachedCmd is a render command with transform/color stored at cache time.
// The two-tier source pointer supports animated tiles: nil = static (use
// cmd.TextureRegion), non-nil = animated (read live TextureRegion from node).
type cachedCmd struct {
	cmd          RenderCommand // Transform/Color stored as-is at cache time
	source       *Node         // nil = static, non-nil = animated (read live TextureRegion)
	sourceNodeID uint32        // emitting node's ID, for deferred animated registration lookup
}

// multiplyAffine32 multiplies two 2D affine matrices stored as [6]float32.
// Used on the replay hot path to avoid float64 conversions.
func multiplyAffine32(p, c [6]float32) [6]float32 {
	return [6]float32{
		p[0]*c[0] + p[2]*c[1],
		p[1]*c[0] + p[3]*c[1],
		p[0]*c[2] + p[2]*c[3],
		p[1]*c[2] + p[3]*c[3],
		p[0]*c[4] + p[2]*c[5] + p[4],
		p[1]*c[4] + p[3]*c[5] + p[5],
	}
}

// invertAffine32 computes the inverse of a 2D affine matrix in float32.
func invertAffine32(m [6]float32) [6]float32 {
	det := m[0]*m[3] - m[2]*m[1]
	if det > -1e-6 && det < 1e-6 {
		return identityTransform32
	}
	invDet := 1.0 / det
	a := m[3] * invDet
	b := -m[1] * invDet
	c := -m[2] * invDet
	d := m[0] * invDet
	return [6]float32{
		a, b, c, d,
		-(a*m[4] + c*m[5]),
		-(b*m[4] + d*m[5]),
	}
}

// replayCacheAsTree replays cached commands with delta remap for transform
// and alpha changes. Animated tiles read live TextureRegion from source node.
func (s *Scene) replayCacheAsTree(n *Node, containerTransform32 [6]float32, containerAlpha float32, treeOrder *int) {
	transformChanged := containerTransform32 != n.cachedParentTransform
	alphaChanged := containerAlpha != n.cachedParentAlpha

	var delta [6]float32
	if transformChanged {
		inv := invertAffine32(n.cachedParentTransform)
		delta = multiplyAffine32(containerTransform32, inv)
	}
	var alphaRatio float32 = 1.0
	if alphaChanged && n.cachedParentAlpha > 1e-6 {
		alphaRatio = containerAlpha / n.cachedParentAlpha
	}

	for i := range n.cachedCommands {
		src := &n.cachedCommands[i]
		cmd := src.cmd // copy from cache
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
		if src.source != nil {
			cmd.TextureRegion = src.source.TextureRegion
		}
		*treeOrder++
		cmd.treeOrder = *treeOrder
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
	viewWorld := multiplyAffine(s.viewTransform, n.worldTransform)
	culled := s.cullActive && n.Renderable && shouldCull(n, viewWorld, s.cullBounds)
	if n.Renderable && !culled {
		s.emitNodeCommandInline(n, treeOrder)
		// Tag the just-emitted command(s) with the node ID.
		for i := startIdx; i < len(s.commands); i++ {
			s.commands[i].emittingNodeID = n.ID
		}
	}

	// Traverse children with culling disabled — the cache must capture ALL
	// children regardless of current viewport so panning replays correctly.
	prevCull := s.cullActive
	s.cullActive = false
	if len(n.children) > 0 {
		children := n.children
		if !n.childrenSorted {
			s.rebuildSortedChildren(n)
		}
		if n.sortedChildren != nil {
			children = n.sortedChildren
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
		if cmd.Type == CommandMesh || cmd.Type == CommandParticle || cmd.transientDirectImage {
			blocked = true
			break
		}
	}

	if blocked {
		// Can't cache — disable and fall through. Commands are already emitted.
		n.cacheTreeDirty = true
		return
	}

	// Store in cachedCommands.
	if cap(n.cachedCommands) < len(newCmds) {
		n.cachedCommands = make([]cachedCmd, len(newCmds))
	}
	n.cachedCommands = n.cachedCommands[:len(newCmds)]

	for i := range newCmds {
		n.cachedCommands[i] = cachedCmd{
			cmd:          newCmds[i],
			source:       nil, // static by default (two-tier)
			sourceNodeID: newCmds[i].emittingNodeID,
		}
	}

	n.cachedParentTransform = containerTransform32
	n.cachedParentAlpha = containerAlpha
	n.cacheTreeDirty = false
}

// emitNodeCommandInline emits a command for the node itself (used by buildCacheAsTree
// when the cached container is also renderable).
func (s *Scene) emitNodeCommandInline(n *Node, treeOrder *int) {
	viewWorld := multiplyAffine(s.viewTransform, n.worldTransform)
	switch n.Type {
	case NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   affine32(viewWorld),
			Color:       color32{float32(n.Color.R), float32(n.Color.G), float32(n.Color.B), float32(n.Color.A * n.worldAlpha)},
			BlendMode:   n.BlendMode,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			treeOrder:   *treeOrder,
		}
		if n.customImage != nil {
			cmd.directImage = n.customImage
		} else {
			cmd.TextureRegion = n.TextureRegion
		}
		s.commands = append(s.commands, cmd)
	case NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			s.commands = emitSDFTextCommand(n.TextBlock, n, viewWorld, s.commands, treeOrder)
		}
	}
	// Mesh/Particle handled as "blocked" — won't reach here during cache build
}
