package lighting

import (
	"image"
	"image/color"
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Function pointers wired by root to break dependency on render/ and atlas/.
var (
	// NewRenderTextureFn creates a RenderTexture and returns it as any.
	NewRenderTextureFn func(w, h int) any

	// RenderTextureImageFn returns the *ebiten.Image from a RenderTexture (any).
	RenderTextureImageFn func(rt any) *ebiten.Image

	// RenderTextureNewSpriteFn creates a sprite node from a RenderTexture.
	RenderTextureNewSpriteFn func(rt any, name string) *node.Node

	// RenderTextureDisposeFn disposes a RenderTexture.
	RenderTextureDisposeFn func(rt any)

	// EnsureMagentaImageFn returns the 1x1 magenta placeholder image.
	EnsureMagentaImageFn func() *ebiten.Image
)

// Light represents a light source in a LightLayer.
type Light struct {
	X, Y      float64
	Radius    float64
	Rotation  float64
	Intensity float64
	Enabled   bool
	Color     types.Color
	// TextureRegion, if non-zero, uses this sprite sheet region instead of
	// the default feathered circle.
	TextureRegion types.TextureRegion
	// Target, if set, makes the light follow this node's pivot point each Redraw.
	Target  *node.Node
	OffsetX float64
	OffsetY float64
}

// lightDrawInfo caches the resolved image and computed GeoM for a single light.
type lightDrawInfo struct {
	img       *ebiten.Image
	geoM      ebiten.GeoM
	intensity float32
	hasColor  bool
}

// LightLayer provides a convenient 2D lighting effect using erase blending.
type LightLayer struct {
	rt           any // RenderTexture (opaque, managed via function pointers)
	node_        *node.Node
	lights       []*Light
	pages        []*ebiten.Image
	ambientAlpha float64
	circleCache  map[int]*ebiten.Image
	imgOp        ebiten.DrawImageOptions
	drawScratch  []lightDrawInfo
	eraseBlend   ebiten.Blend
	addBlend     ebiten.Blend
}

// NewLightLayer creates a light layer covering (w x h) pixels.
func NewLightLayer(w, h int, ambientAlpha float64) *LightLayer {
	rt := NewRenderTextureFn(w, h)
	n := RenderTextureNewSpriteFn(rt, "light_layer")
	n.SetBlendMode(types.BlendMultiply)

	ll := &LightLayer{
		rt:           rt,
		node_:        n,
		ambientAlpha: ambientAlpha,
		eraseBlend:   types.BlendErase.EbitenBlend(),
		addBlend:     types.BlendAdd.EbitenBlend(),
	}
	return ll
}

// Node returns the sprite node that displays the light layer.
func (ll *LightLayer) Node() *node.Node {
	return ll.node_
}

// RenderTextureImage returns the underlying *ebiten.Image of the render texture.
func (ll *LightLayer) RenderTextureImage() *ebiten.Image {
	if ll.rt == nil || RenderTextureImageFn == nil {
		return nil
	}
	return RenderTextureImageFn(ll.rt)
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
func (ll *LightLayer) SetPages(pages []*ebiten.Image) {
	ll.pages = pages
}

// SetCircleRadius pre-generates a feathered circle texture at the given radius.
func (ll *LightLayer) SetCircleRadius(radius float64) {
	ll.getCircle(radius)
}

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
	img := GenerateCircle(float64(key))
	ll.circleCache[key] = img
	return img
}

// Redraw fills the texture with ambient darkness then erases light shapes.
func (ll *LightLayer) Redraw() {
	for _, l := range ll.lights {
		if l.Target == nil || l.Target.IsDisposed() {
			continue
		}
		wx, wy := l.Target.LocalToWorld(l.Target.PivotX_, l.Target.PivotY_)
		lx, ly := ll.node_.WorldToLocal(wx, wy)
		l.X = lx + l.OffsetX
		l.Y = ly + l.OffsetY
	}

	target := RenderTextureImageFn(ll.rt)

	a := types.Clamp01(ll.ambientAlpha)
	target.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: uint8(a * 255)})

	n := len(ll.lights)
	if cap(ll.drawScratch) < n {
		ll.drawScratch = make([]lightDrawInfo, n)
	} else {
		ll.drawScratch = ll.drawScratch[:n]
	}

	op := &ll.imgOp

	// Erase pass
	for i, l := range ll.lights {
		info := &ll.drawScratch[i]
		info.img = nil
		if !l.Enabled || l.Radius <= 0 {
			continue
		}
		img, srcW, srcH := ll.resolveLightImage(l)
		if img == nil {
			continue
		}
		intensity := types.Clamp01(l.Intensity)
		ll.setupLightGeoM(op, l, srcW, srcH)

		info.img = img
		info.geoM = op.GeoM
		info.intensity = float32(intensity)
		c := l.Color
		info.hasColor = c != (types.Color{}) && c != types.ColorWhite

		op.ColorScale.Reset()
		op.ColorScale.Scale(info.intensity, info.intensity, info.intensity, info.intensity)
		op.Blend = ll.eraseBlend
		target.DrawImage(img, op)
	}

	// Tint pass
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
			float32(c.R())*tintAlpha,
			float32(c.G())*tintAlpha,
			float32(c.B())*tintAlpha,
			tintAlpha,
		)
		op.Blend = ll.addBlend
		target.DrawImage(info.img, op)
	}
}

func (ll *LightLayer) resolveLightImage(l *Light) (img *ebiten.Image, srcW, srcH float64) {
	r := &l.TextureRegion
	if r.Width > 0 && r.Height > 0 {
		var page *ebiten.Image
		if r.Page == types.MagentaPlaceholderPage && EnsureMagentaImageFn != nil {
			page = EnsureMagentaImageFn()
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
	circle := ll.getCircle(l.Radius)
	sz := float64(circle.Bounds().Dx())
	return circle, sz, sz
}

func (ll *LightLayer) setupLightGeoM(op *ebiten.DrawImageOptions, l *Light, srcW, srcH float64) {
	op.GeoM.Reset()

	r := &l.TextureRegion
	if r.Width > 0 && r.Height > 0 && r.Rotated {
		op.GeoM.Rotate(-math.Pi / 2)
		op.GeoM.Translate(0, float64(r.Width))
	}

	if r.OffsetX != 0 || r.OffsetY != 0 {
		op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
	}

	desiredW := l.Radius * 2
	desiredH := l.Radius * 2
	if srcW > 0 && srcH > 0 {
		op.GeoM.Scale(desiredW/srcW, desiredH/srcH)
	}

	op.GeoM.Translate(-desiredW/2, -desiredH/2)

	if l.Rotation != 0 {
		op.GeoM.Rotate(l.Rotation)
	}

	op.GeoM.Translate(l.X, l.Y)
}

// Dispose releases all resources owned by the light layer.
func (ll *LightLayer) Dispose() {
	if ll.rt != nil && RenderTextureDisposeFn != nil {
		RenderTextureDisposeFn(ll.rt)
		ll.rt = nil
	}
	for _, img := range ll.circleCache {
		img.Deallocate()
	}
	ll.circleCache = nil
	if ll.node_ != nil {
		ll.node_.SetCustomImage(nil)
		ll.node_ = nil
	}
	ll.lights = nil
}

// GenerateCircle creates a feathered white circle image with the given radius.
func GenerateCircle(radius float64) *ebiten.Image {
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
				t := 1 - dist
				alpha = t * t * (3 - 2*t)
			}

			a := uint8(alpha * 255)
			off := (y*size + x) * 4
			pix[off+0] = a
			pix[off+1] = a
			pix[off+2] = a
			pix[off+3] = a
		}
	}
	img.WritePixels(pix)
	return img
}
