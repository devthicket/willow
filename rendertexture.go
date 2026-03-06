package willow

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// RenderTexture is a persistent offscreen canvas that can be attached to a
// sprite node via SetCustomImage. Unlike pooled render targets used internally,
// a RenderTexture is owned by the caller and is NOT recycled between frames.
type RenderTexture struct {
	image *ebiten.Image
	w, h  int
}

// NewRenderTexture creates a persistent offscreen canvas of the given size.
func NewRenderTexture(w, h int) *RenderTexture {
	return &RenderTexture{
		image: ebiten.NewImage(w, h),
		w:     w,
		h:     h,
	}
}

// Image returns the underlying *ebiten.Image for direct manipulation.
func (rt *RenderTexture) Image() *ebiten.Image {
	return rt.image
}

// Width returns the texture width in pixels.
func (rt *RenderTexture) Width() int {
	return rt.w
}

// Height returns the texture height in pixels.
func (rt *RenderTexture) Height() int {
	return rt.h
}

// Clear fills the texture with transparent black.
func (rt *RenderTexture) Clear() {
	rt.image.Clear()
}

// Fill fills the entire texture with the given color.
func (rt *RenderTexture) Fill(c Color) {
	rt.image.Fill(c.toRGBA())
}

// DrawImage draws src onto this texture using the provided options.
func (rt *RenderTexture) DrawImage(src *ebiten.Image, op *ebiten.DrawImageOptions) {
	rt.image.DrawImage(src, op)
}

// DrawImageAt draws src at the given position with the specified blend mode.
func (rt *RenderTexture) DrawImageAt(src *ebiten.Image, x, y float64, blend BlendMode) {
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(x, y)
	op.Blend = blend.EbitenBlend()
	rt.image.DrawImage(src, &op)
}

// NewSpriteNode creates a NodeTypeSprite with customImage pre-set to this
// texture. The returned node will display the RenderTexture contents.
func (rt *RenderTexture) NewSpriteNode(name string) *Node {
	n := &Node{Name: name, Type: NodeTypeSprite}
	nodeDefaults(n)
	n.customImage = rt.image
	return n
}

// RenderTextureDrawOpts controls how an image or sprite is drawn onto a
// RenderTexture when using the "Colored" draw methods.
type RenderTextureDrawOpts struct {
	// X and Y are the draw position in pixels.
	X, Y float64
	// ScaleX and ScaleY are scale factors. Zero defaults to 1.0.
	ScaleX, ScaleY float64
	// Rotation is the rotation in radians (clockwise).
	Rotation float64
	// PivotX and PivotY are the transform origin for scale and rotation.
	PivotX, PivotY float64
	// Color is a multiplicative tint. Zero value defaults to white (no tint).
	Color Color
	// Alpha is the opacity multiplier. Zero defaults to 1.0 (fully opaque).
	Alpha float64
	// BlendMode selects the compositing operation.
	BlendMode BlendMode
}

// DrawSprite draws a TextureRegion from the atlas onto this texture at (x, y).
func (rt *RenderTexture) DrawSprite(region TextureRegion, x, y float64, blend BlendMode, pages []*ebiten.Image) {
	page := resolvePageImage(region, pages)
	if page == nil {
		return
	}

	var subRect image.Rectangle
	if region.Rotated {
		subRect = image.Rect(int(region.X), int(region.Y), int(region.X)+int(region.Height), int(region.Y)+int(region.Width))
	} else {
		subRect = image.Rect(int(region.X), int(region.Y), int(region.X)+int(region.Width), int(region.Y)+int(region.Height))
	}
	sub := page.SubImage(subRect).(*ebiten.Image)

	var op ebiten.DrawImageOptions
	if region.Rotated {
		op.GeoM.Rotate(-1.5707963267948966) // -π/2
		op.GeoM.Translate(0, float64(region.Width))
	}
	op.GeoM.Translate(x+float64(region.OffsetX), y+float64(region.OffsetY))
	op.Blend = blend.EbitenBlend()
	rt.image.DrawImage(sub, &op)
}

// DrawSpriteColored draws a TextureRegion with full transform, color, and alpha.
func (rt *RenderTexture) DrawSpriteColored(region TextureRegion, opts RenderTextureDrawOpts, pages []*ebiten.Image) {
	page := resolvePageImage(region, pages)
	if page == nil {
		return
	}

	var subRect image.Rectangle
	if region.Rotated {
		subRect = image.Rect(int(region.X), int(region.Y), int(region.X)+int(region.Height), int(region.Y)+int(region.Width))
	} else {
		subRect = image.Rect(int(region.X), int(region.Y), int(region.X)+int(region.Width), int(region.Y)+int(region.Height))
	}
	sub := page.SubImage(subRect).(*ebiten.Image)

	var op ebiten.DrawImageOptions
	if region.Rotated {
		op.GeoM.Rotate(-1.5707963267948966) // -π/2
		op.GeoM.Translate(0, float64(region.Width))
	}
	applyDrawOpts(&op, opts, float64(region.OffsetX), float64(region.OffsetY))
	rt.image.DrawImage(sub, &op)
}

// DrawImageColored draws a raw *ebiten.Image with full transform, color, and alpha.
func (rt *RenderTexture) DrawImageColored(img *ebiten.Image, opts RenderTextureDrawOpts) {
	var op ebiten.DrawImageOptions
	applyDrawOpts(&op, opts, 0, 0)
	rt.image.DrawImage(img, &op)
}

// Resize deallocates the old image and creates a new one at the given dimensions.
func (rt *RenderTexture) Resize(width, height int) {
	if rt.image != nil {
		rt.image.Deallocate()
	}
	rt.image = ebiten.NewImage(width, height)
	rt.w = width
	rt.h = height
}

// resolvePageImage returns the atlas page image for a region, handling the
// magenta placeholder sentinel.
func resolvePageImage(region TextureRegion, pages []*ebiten.Image) *ebiten.Image {
	if region.Page == magentaPlaceholderPage {
		return ensureMagentaImage()
	}
	idx := int(region.Page)
	if idx < len(pages) {
		return pages[idx]
	}
	return nil
}

// applyDrawOpts configures an ebiten.DrawImageOptions from RenderTextureDrawOpts.
func applyDrawOpts(op *ebiten.DrawImageOptions, opts RenderTextureDrawOpts, offsetX, offsetY float64) {
	op.GeoM.Translate(-opts.PivotX, -opts.PivotY)
	sx, sy := opts.ScaleX, opts.ScaleY
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	op.GeoM.Scale(sx, sy)
	if opts.Rotation != 0 {
		op.GeoM.Rotate(opts.Rotation)
	}
	op.GeoM.Translate(opts.X+offsetX+opts.PivotX, opts.Y+offsetY+opts.PivotY)

	alpha := opts.Alpha
	if alpha == 0 {
		alpha = 1
	}
	c := opts.Color
	if c == (Color{}) {
		c = ColorWhite
	}
	op.ColorScale.Scale(
		float32(c.r*c.a*alpha),
		float32(c.g*c.a*alpha),
		float32(c.b*c.a*alpha),
		float32(c.a*alpha),
	)
	op.Blend = opts.BlendMode.EbitenBlend()
}

// Dispose deallocates the underlying image. The RenderTexture should not be
// used after calling Dispose.
func (rt *RenderTexture) Dispose() {
	if rt.image != nil {
		rt.image.Deallocate()
		rt.image = nil
	}
}

// toRGBA converts a willow Color to a color.RGBA (premultiplied).
func (c Color) toRGBA() colorRGBA {
	return colorRGBA{
		R: uint8(clamp01(c.r*c.a) * 255),
		G: uint8(clamp01(c.g*c.a) * 255),
		B: uint8(clamp01(c.b*c.a) * 255),
		A: uint8(clamp01(c.a) * 255),
	}
}

// colorRGBA implements the color.Color interface for image.Fill.
type colorRGBA struct {
	R, G, B, A uint8
}

func (c colorRGBA) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R) * 0x101
	g = uint32(c.G) * 0x101
	b = uint32(c.B) * 0x101
	a = uint32(c.A) * 0x101
	return
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
