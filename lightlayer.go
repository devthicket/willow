package willow

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Light represents a light source in a LightLayer.
type Light struct {
	// X and Y are the light's position in the light layer's local coordinate space.
	X, Y float64
	// Radius controls the drawn size (diameter = Radius*2 pixels).
	Radius float64
	// Rotation is the light's rotation in radians; useful for directional shapes.
	Rotation float64
	// Intensity controls light brightness in the range [0, 1].
	Intensity float64
	// Enabled determines whether this light is drawn during Redraw.
	// Disabled lights are skipped entirely.
	Enabled bool
	// Color is the tint color. Zero value or white means neutral (no tint).
	Color Color
	// TextureRegion, if non-zero, uses this sprite sheet region instead of
	// the default feathered circle.
	TextureRegion TextureRegion
	// Target, if set, makes the light follow this node's pivot point each Redraw.
	Target *Node
	// OffsetX and OffsetY offset the light from the target's pivot in
	// light-layer space.
	OffsetX float64
	OffsetY float64
}

// lightDrawInfo caches the resolved image and computed GeoM for a single light
// between the erase and tint passes in Redraw, so neither needs to be computed
// twice per colored light.
type lightDrawInfo struct {
	img       *ebiten.Image
	geoM      ebiten.GeoM
	intensity float32
	hasColor  bool
}

// LightLayer provides a convenient 2D lighting effect using erase blending.
// It manages a set of lights, renders them into an offscreen texture filled
// with an ambient darkness color, and erases feathered circles at each light
// position. The resulting texture is displayed as a sprite node with
// BlendMultiply so it darkens the scene everywhere except where lights shine.
type LightLayer struct {
	rt           *RenderTexture
	node         *Node
	lights       []*Light
	pages        []*ebiten.Image // atlas pages for resolving Light.TextureRegion
	ambientAlpha float64
	circleCache  map[int]*ebiten.Image // cached circle textures keyed by quantized radius
	imgOp        ebiten.DrawImageOptions
	drawScratch  []lightDrawInfo // pre-allocated cache shared between erase and tint passes
	eraseBlend   ebiten.Blend    // precomputed once; avoids a method call per light
	addBlend     ebiten.Blend    // precomputed once
}

// NewLightLayer creates a light layer covering (w x h) pixels.
// ambientAlpha controls the base darkness (0 = fully transparent, 1 = fully opaque black).
func NewLightLayer(w, h int, ambientAlpha float64) *LightLayer {
	rt := NewRenderTexture(w, h)
	node := rt.NewSpriteNode("light_layer")
	node.blendMode = BlendMultiply

	ll := &LightLayer{
		rt:           rt,
		node:         node,
		ambientAlpha: ambientAlpha,
		eraseBlend:   BlendErase.EbitenBlend(),
		addBlend:     BlendAdd.EbitenBlend(),
	}
	return ll
}

// Node returns the sprite node that displays the light layer.
// Add this to the scene graph to render the lighting effect.
func (ll *LightLayer) Node() *Node {
	return ll.node
}

// RenderTexture returns the underlying RenderTexture.
func (ll *LightLayer) RenderTexture() *RenderTexture {
	return ll.rt
}

// AddLight adds a light to the layer.
func (ll *LightLayer) AddLight(l *Light) {
	ll.lights = append(ll.lights, l)
}

// RemoveLight removes a light from the layer.
func (ll *LightLayer) RemoveLight(l *Light) {
	for i, existing := range ll.lights {
		if existing == l {
			ll.lights = append(ll.lights[:i], ll.lights[i+1:]...)
			return
		}
	}
}

// ClearLights removes all lights from the layer.
func (ll *LightLayer) ClearLights() {
	ll.lights = ll.lights[:0]
}

// Lights returns the current light list. The returned slice MUST NOT be mutated.
func (ll *LightLayer) Lights() []*Light {
	return ll.lights
}

// SetAmbientAlpha sets the base darkness level.
func (ll *LightLayer) SetAmbientAlpha(a float64) {
	ll.ambientAlpha = a
}

// AmbientAlpha returns the current ambient darkness level.
func (ll *LightLayer) AmbientAlpha() float64 {
	return ll.ambientAlpha
}

// SetPages stores the atlas page images used to resolve Light.TextureRegion.
// Typically called with the Scene's pages after loading atlases.
func (ll *LightLayer) SetPages(pages []*ebiten.Image) {
	ll.pages = pages
}

// SetCircleRadius pre-generates a feathered circle texture at the given radius
// and stores it in the cache.
func (ll *LightLayer) SetCircleRadius(radius float64) {
	ll.getCircle(radius)
}

// getCircle returns a cached circle texture for the given radius, generating
// one if it doesn't exist. Radius is quantized to the nearest integer to
// avoid generating separate textures for tiny differences.
func (ll *LightLayer) getCircle(radius float64) *ebiten.Image {
	key := int(math.Ceil(radius))
	if key < 1 {
		key = 1
	}
	if ll.circleCache == nil {
		ll.circleCache = make(map[int]*ebiten.Image)
	}
	if img, ok := ll.circleCache[key]; ok {
		return img
	}
	img := generateCircle(float64(key))
	ll.circleCache[key] = img
	return img
}

// Redraw fills the texture with ambient darkness then erases light shapes at
// each enabled light position. Lights with a TextureRegion use that sprite;
// lights without fall back to a generated feathered circle.
// Call this every frame (or whenever lights change) before drawing the scene.
func (ll *LightLayer) Redraw() {
	// Sync attached lights to their target node positions.
	for _, l := range ll.lights {
		if l.Target == nil || l.Target.IsDisposed() {
			continue
		}
		// Get target's pivot in world space.
		wx, wy := l.Target.LocalToWorld(l.Target.pivotX, l.Target.pivotY)
		// Convert to light layer's local coordinate space.
		lx, ly := ll.node.WorldToLocal(wx, wy)
		l.X = lx + l.OffsetX
		l.Y = ly + l.OffsetY
	}

	target := ll.rt.Image()

	// Fill with ambient darkness. Fill overwrites every pixel, so a prior
	// Clear() call would be redundant and is omitted.
	a := clamp01(ll.ambientAlpha)
	target.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: uint8(a * 255)})

	// Grow the per-light scratch slice without allocating after warmup.
	n := len(ll.lights)
	if cap(ll.drawScratch) < n {
		ll.drawScratch = make([]lightDrawInfo, n)
	} else {
		ll.drawScratch = ll.drawScratch[:n]
	}

	op := &ll.imgOp

	// ---- Erase pass ---------------------------------------------------------
	// All erase draws are issued before all tint draws so that Ebitengine can
	// batch consecutive calls that share the same source image and blend mode
	// into a single GPU draw call. The interleaved layout used previously
	// (erase₀, tint₀, erase₁, tint₁, …) prevented any batching.
	for i, l := range ll.lights {
		info := &ll.drawScratch[i]
		info.img = nil // sentinel: skip this light in the tint pass
		if !l.Enabled || l.Radius <= 0 {
			continue
		}
		img, srcW, srcH := ll.resolveLightImage(l)
		if img == nil {
			continue
		}
		intensity := clamp01(l.Intensity)
		ll.setupLightGeoM(op, l, srcW, srcH)

		// Cache resolved image, GeoM, and intensity for the tint pass so we
		// avoid a second resolveLightImage call and a second GeoM computation.
		info.img = img
		info.geoM = op.GeoM
		info.intensity = float32(intensity)
		c := l.Color
		info.hasColor = c != (Color{}) && c != ColorWhite

		op.ColorScale.Reset()
		op.ColorScale.Scale(info.intensity, info.intensity, info.intensity, info.intensity)
		op.Blend = ll.eraseBlend
		target.DrawImage(img, op)
	}

	// ---- Tint pass ----------------------------------------------------------
	// Reuses the image and GeoM cached above; no second resolveLightImage or
	// setupLightGeoM call is needed.
	for i, l := range ll.lights {
		info := &ll.drawScratch[i]
		if info.img == nil || !info.hasColor {
			continue
		}
		c := l.Color
		op.GeoM = info.geoM
		op.ColorScale.Reset()
		tintAlpha := info.intensity * 0.3
		op.ColorScale.Scale(
			float32(c.r)*tintAlpha,
			float32(c.g)*tintAlpha,
			float32(c.b)*tintAlpha,
			tintAlpha,
		)
		op.Blend = ll.addBlend
		target.DrawImage(info.img, op)
	}
}

// resolveLightImage returns the image to draw for a light, along with its
// source dimensions. Uses TextureRegion if set, otherwise the default circle.
func (ll *LightLayer) resolveLightImage(l *Light) (img *ebiten.Image, srcW, srcH float64) {
	r := &l.TextureRegion
	if r.Width > 0 && r.Height > 0 {
		// Resolve atlas page.
		var page *ebiten.Image
		if r.Page == magentaPlaceholderPage {
			page = ensureMagentaImage()
		} else if int(r.Page) < len(ll.pages) {
			page = ll.pages[r.Page]
		}
		if page == nil {
			return nil, 0, 0
		}
		var subRect image.Rectangle
		if r.Rotated {
			subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Height), int(r.Y)+int(r.Width))
		} else {
			subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Width), int(r.Y)+int(r.Height))
		}
		sub := page.SubImage(subRect).(*ebiten.Image)
		return sub, float64(r.OriginalW), float64(r.OriginalH)
	}
	// Default: generated circle.
	circle := ll.getCircle(l.Radius)
	sz := float64(circle.Bounds().Dx())
	return circle, sz, sz
}

// setupLightGeoM configures the GeoM on op to position, scale, and rotate
// a light image so it's centered on the light's (X, Y) at the correct size.
func (ll *LightLayer) setupLightGeoM(op *ebiten.DrawImageOptions, l *Light, srcW, srcH float64) {
	op.GeoM.Reset()

	// Handle rotated atlas regions.
	r := &l.TextureRegion
	if r.Width > 0 && r.Height > 0 && r.Rotated {
		op.GeoM.Rotate(-math.Pi / 2)
		op.GeoM.Translate(0, float64(r.Width))
	}

	// Apply trim offset for atlas regions.
	if r.OffsetX != 0 || r.OffsetY != 0 {
		op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
	}

	// Scale to desired size (Radius*2 x Radius*2).
	desiredW := l.Radius * 2
	desiredH := l.Radius * 2
	if srcW > 0 && srcH > 0 {
		op.GeoM.Scale(desiredW/srcW, desiredH/srcH)
	}

	// Center on origin for rotation.
	op.GeoM.Translate(-desiredW/2, -desiredH/2)

	// Apply rotation.
	if l.Rotation != 0 {
		op.GeoM.Rotate(l.Rotation)
	}

	// Translate to light position.
	op.GeoM.Translate(l.X, l.Y)
}

// Dispose releases all resources owned by the light layer.
func (ll *LightLayer) Dispose() {
	if ll.rt != nil {
		ll.rt.Dispose()
		ll.rt = nil
	}
	for _, img := range ll.circleCache {
		img.Deallocate()
	}
	ll.circleCache = nil
	if ll.node != nil {
		ll.node.customImage = nil
		ll.node = nil
	}
	ll.lights = nil
}

// generateCircle creates a feathered white circle image with the given radius.
// Uses smoothstep falloff and premultiplied alpha.
func generateCircle(radius float64) *ebiten.Image {
	size := int(math.Ceil(radius * 2))
	if size < 1 {
		size = 1
	}
	img := ebiten.NewImage(size, size)
	pix := make([]byte, size*size*4)

	cx, cy := radius, radius
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			dist := math.Sqrt(dx*dx+dy*dy) / radius

			var alpha float64
			if dist >= 1 {
				alpha = 0
			} else {
				// smoothstep: 1 at center, 0 at edge
				t := 1 - dist
				alpha = t * t * (3 - 2*t)
			}

			a := uint8(alpha * 255)
			off := (y*size + x) * 4
			pix[off+0] = a // premultiplied white
			pix[off+1] = a
			pix[off+2] = a
			pix[off+3] = a
		}
	}
	img.WritePixels(pix)
	return img
}
