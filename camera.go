package willow

import (
	"math"

	"github.com/tanema/gween"
)

// scrollAnim holds active scroll-to tweens for camera X and Y.
type scrollAnim struct {
	tweenX *gween.Tween
	tweenY *gween.Tween
	doneX  bool
	doneY  bool
}

// Camera controls the view into the scene: position, zoom, rotation, and viewport.
type Camera struct {
	// X and Y are the world-space position the camera centers on.
	X, Y float64
	// Zoom is the scale factor (1.0 = no zoom, >1 = zoom in, <1 = zoom out).
	Zoom float64
	// Rotation is the camera rotation in radians (clockwise).
	Rotation float64
	// Viewport is the screen-space rectangle this camera renders into.
	Viewport Rect

	// CullEnabled skips nodes whose world AABB doesn't intersect the
	// camera's visible bounds.
	CullEnabled bool

	followTarget  *Node
	followOffsetX float64
	followOffsetY float64
	followLerp    float64

	// BoundsEnabled clamps the camera position so the visible area stays
	// within Bounds.
	BoundsEnabled bool
	// Bounds is the world-space rectangle the camera is clamped to when
	// BoundsEnabled is true.
	Bounds Rect

	viewMatrix    [6]float64
	invViewMatrix [6]float64
	dirty         bool

	scrollTween *scrollAnim
}

// newCamera creates a Camera with default values and the given viewport.
func newCamera(viewport Rect) *Camera {
	return &Camera{
		Zoom:        1.0,
		Viewport:    viewport,
		CullEnabled: true,
		dirty:       true,
	}
}

// Follow makes the camera track a target node with the given offset and lerp factor.
// A lerp of 1.0 snaps immediately; lower values give smoother following.
func (c *Camera) Follow(node *Node, offsetX, offsetY, lerp float64) {
	c.followTarget = node
	c.followOffsetX = offsetX
	c.followOffsetY = offsetY
	c.followLerp = lerp
}

// Unfollow stops tracking the current target node.
func (c *Camera) Unfollow() {
	c.followTarget = nil
}

// ScrollTo animates the camera to the given world position using the provided
// TweenConfig for duration and easing.
func (c *Camera) ScrollTo(x, y float64, cfg TweenConfig) {
	fn := tweenEase(cfg)
	c.scrollTween = &scrollAnim{
		tweenX: gween.New(float32(c.X), float32(x), cfg.Duration, fn),
		tweenY: gween.New(float32(c.Y), float32(y), cfg.Duration, fn),
	}
}

// ScrollToTile scrolls to the center of the given tile in a tile-based layout.
func (c *Camera) ScrollToTile(tileX, tileY int, tileW, tileH float64, cfg TweenConfig) {
	worldX := float64(tileX)*tileW + tileW/2
	worldY := float64(tileY)*tileH + tileH/2
	c.ScrollTo(worldX, worldY, cfg)
}

// SetBounds enables camera bounds clamping.
func (c *Camera) SetBounds(bounds Rect) {
	c.BoundsEnabled = true
	c.Bounds = bounds
}

// ClearBounds disables camera bounds clamping.
func (c *Camera) ClearBounds() {
	c.BoundsEnabled = false
}

// ClampToBounds immediately clamps the camera position so the visible area
// stays within Bounds. Call this after modifying X/Y directly (e.g. in a
// drag callback) to prevent a single frame where the camera sees outside
// the bounds. No-op if BoundsEnabled is false.
func (c *Camera) ClampToBounds() {
	if c.BoundsEnabled {
		c.clampToBounds()
	}
}

// update advances follow, scroll, and bounds clamping. Called from Scene.Update().
func (c *Camera) update(dt float32) {
	prevX, prevY := c.X, c.Y
	prevZoom, prevRot := c.Zoom, c.Rotation

	// Follow target
	if c.followTarget != nil && !c.followTarget.IsDisposed() {
		targetX := c.followTarget.WorldTransform[4] + c.followOffsetX
		targetY := c.followTarget.WorldTransform[5] + c.followOffsetY
		c.X += (targetX - c.X) * c.followLerp
		c.Y += (targetY - c.Y) * c.followLerp
	}

	// Scroll animation
	if c.scrollTween != nil {
		if !c.scrollTween.doneX {
			val, done := c.scrollTween.tweenX.Update(dt)
			c.X = float64(val)
			c.scrollTween.doneX = done
		}
		if !c.scrollTween.doneY {
			val, done := c.scrollTween.tweenY.Update(dt)
			c.Y = float64(val)
			c.scrollTween.doneY = done
		}
		if c.scrollTween.doneX && c.scrollTween.doneY {
			c.scrollTween = nil
		}
	}

	// Bounds clamping
	if c.BoundsEnabled {
		c.clampToBounds()
	}

	if c.X != prevX || c.Y != prevY || c.Zoom != prevZoom || c.Rotation != prevRot {
		c.dirty = true
	}
}

// clampToBounds restricts camera position so the visible area stays within Bounds.
func (c *Camera) clampToBounds() {
	halfW := c.Viewport.Width / (2 * c.Zoom)
	halfH := c.Viewport.Height / (2 * c.Zoom)

	minX := c.Bounds.X + halfW
	maxX := c.Bounds.X + c.Bounds.Width - halfW
	minY := c.Bounds.Y + halfH
	maxY := c.Bounds.Y + c.Bounds.Height - halfH

	// If bounds are smaller than visible area, center the camera.
	if minX > maxX {
		c.X = c.Bounds.X + c.Bounds.Width/2
	} else {
		c.X = math.Max(minX, math.Min(c.X, maxX))
	}
	if minY > maxY {
		c.Y = c.Bounds.Y + c.Bounds.Height/2
	} else {
		c.Y = math.Max(minY, math.Min(c.Y, maxY))
	}
}

// computeViewMatrix recomputes the cached view matrix if dirty.
//
// viewMatrix = Translate(cx, cy) * Scale(zoom) * Rotate(-rotation) * Translate(-X, -Y)
// where cx, cy = viewport center.
func (c *Camera) computeViewMatrix() [6]float64 {
	if !c.dirty {
		return c.viewMatrix
	}
	c.dirty = false

	cx := c.Viewport.X + c.Viewport.Width/2
	cy := c.Viewport.Y + c.Viewport.Height/2

	cos := math.Cos(-c.Rotation)
	sin := math.Sin(-c.Rotation)
	z := c.Zoom

	// Combined: Translate(cx,cy) * Scale(z) * Rotate(-rot) * Translate(-X,-Y)
	// [a b tx]   [z*cos  -z*sin  cx + z*(- cos*X + sin*Y)]
	// [c d ty] = [z*sin   z*cos  cy + z*(-sin*X - cos*Y)]
	a := z * cos
	b := -z * sin
	cc := z * sin
	d := z * cos
	tx := cx + z*(-cos*c.X+sin*c.Y)
	ty := cy + z*(-sin*c.X-cos*c.Y)

	c.viewMatrix = [6]float64{a, cc, b, d, tx, ty}
	c.invViewMatrix = invertAffine(c.viewMatrix)
	return c.viewMatrix
}

// WorldToScreen converts world coordinates to screen coordinates.
func (c *Camera) WorldToScreen(wx, wy float64) (sx, sy float64) {
	c.computeViewMatrix()
	sx, sy = transformPoint(c.viewMatrix, wx, wy)
	return
}

// ScreenToWorld converts screen coordinates to world coordinates.
func (c *Camera) ScreenToWorld(sx, sy float64) (wx, wy float64) {
	c.computeViewMatrix()
	wx, wy = transformPoint(c.invViewMatrix, sx, sy)
	return
}

// VisibleBounds returns the axis-aligned bounding rect of the camera's visible
// area in world space.
func (c *Camera) VisibleBounds() Rect {
	c.computeViewMatrix()
	inv := c.invViewMatrix

	vx := c.Viewport.X
	vy := c.Viewport.Y
	vr := vx + c.Viewport.Width
	vb := vy + c.Viewport.Height

	// Transform the four viewport corners to world space.
	x0, y0 := transformPoint(inv, vx, vy)
	x1, y1 := transformPoint(inv, vr, vy)
	x2, y2 := transformPoint(inv, vr, vb)
	x3, y3 := transformPoint(inv, vx, vb)

	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))

	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// Invalidate forces a recomputation of the view matrix.
func (c *Camera) Invalidate() {
	c.dirty = true
}

// --- Culling ---

// worldAABB computes the axis-aligned bounding box for a rectangle of size (w, h)
// transformed by the given affine matrix. Zero allocations.
func worldAABB(transform [6]float64, w, h float64) Rect {
	a, b, cc, d, tx, ty := transform[0], transform[2], transform[1], transform[3], transform[4], transform[5]

	// Transform four corners: (0,0), (w,0), (w,h), (0,h)
	x0, y0 := tx, ty
	x1, y1 := a*w+tx, cc*w+ty
	x2, y2 := a*w+b*h+tx, cc*w+d*h+ty
	x3, y3 := b*h+tx, d*h+ty

	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))

	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// nodeDimensions returns the width and height used for AABB culling.
func nodeDimensions(n *Node) (w, h float64) {
	switch n.Type {
	case NodeTypeSprite:
		if n.CustomImage_ != nil {
			b := n.CustomImage_.Bounds()
			return float64(b.Dx()), float64(b.Dy())
		}
		return float64(n.TextureRegion_.OriginalW), float64(n.TextureRegion_.OriginalH)
	case NodeTypeMesh:
		n.recomputeMeshAABB()
		return n.Mesh.Aabb.Width, n.Mesh.Aabb.Height
	case NodeTypeParticleEmitter:
		if n.Emitter != nil {
			r := &n.Emitter.Config.Region
			return float64(r.OriginalW), float64(r.OriginalH)
		}
		return 0, 0
	case NodeTypeText:
		if n.TextBlock != nil {
			textBlockLayout(n.TextBlock) // ensure measured dims are current
			return n.TextBlock.MeasuredW, n.TextBlock.MeasuredH
		}
		return 0, 0
	default:
		return 0, 0
	}
}

// shouldCull returns true if the node should be skipped during rendering.
// viewWorld is the coordinate-space transform used for AABB computation
// (typically view * worldTransform for screen-space culling).
// Containers are never culled. Text nodes are culled when measured dimensions are available.
func shouldCull(n *Node, viewWorld [6]float64, cullBounds Rect) bool {
	switch n.Type {
	case NodeTypeContainer:
		return false
	case NodeTypeMesh:
		aabb := meshWorldAABBOffset(n, viewWorld)
		if aabb.Width == 0 && aabb.Height == 0 {
			return false
		}
		return !aabb.Intersects(cullBounds)
	}

	w, h := nodeDimensions(n)
	if w == 0 && h == 0 {
		return false // Can't determine size; don't cull.
	}

	aabb := worldAABB(viewWorld, w, h)
	return !aabb.Intersects(cullBounds)
}
