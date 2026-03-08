package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// --- Function pointers for atlas page resolution (wired by root) ---

var (
	// PageFn resolves an atlas page index to its *ebiten.Image.
	PageFn func(pageIdx int) *ebiten.Image

	// MagentaImageFn returns the magenta placeholder image.
	MagentaImageFn func() *ebiten.Image

	// MagentaPlaceholderPage is the sentinel page index for the magenta placeholder.
	MagentaPlaceholderPage uint16 = 0xFFFF

	// NewSpriteFn creates a new sprite node (wired by root constructors).
	NewSpriteFn func(name string, region types.TextureRegion) *node.Node
)

// ResolvePage returns the atlas page image for a page index.
func ResolvePage(pageIdx uint16) *ebiten.Image {
	if pageIdx == MagentaPlaceholderPage {
		if MagentaImageFn != nil {
			return MagentaImageFn()
		}
		return nil
	}
	if PageFn != nil {
		return PageFn(int(pageIdx))
	}
	return nil
}

// RenderTexture is a persistent offscreen canvas that can be attached to a
// sprite node via SetCustomImage. Unlike pooled render targets used internally,
// a RenderTexture is owned by the caller and is NOT recycled between frames.
type RenderTexture struct {
	Img  *ebiten.Image
	W, H int
}

// NewRenderTexture creates a persistent offscreen canvas of the given size.
func NewRenderTexture(w, h int) *RenderTexture {
	return &RenderTexture{
		Img: ebiten.NewImage(w, h),
		W:   w,
		H:   h,
	}
}

// Image returns the underlying *ebiten.Image for direct manipulation.
func (rt *RenderTexture) Image() *ebiten.Image {
	return rt.Img
}

// Width returns the texture width in pixels.
func (rt *RenderTexture) Width() int {
	return rt.W
}

// Height returns the texture height in pixels.
func (rt *RenderTexture) Height() int {
	return rt.H
}

// Clear fills the texture with transparent black.
func (rt *RenderTexture) Clear() {
	rt.Img.Clear()
}

// Fill fills the entire texture with the given color.
func (rt *RenderTexture) Fill(c types.Color) {
	rt.Img.Fill(ColorToRGBA(c))
}

// DrawImage draws src onto this texture using the provided options.
func (rt *RenderTexture) DrawImage(src *ebiten.Image, op *ebiten.DrawImageOptions) {
	rt.Img.DrawImage(src, op)
}

// DrawImageAt draws src at the given position with the specified blend mode.
func (rt *RenderTexture) DrawImageAt(src *ebiten.Image, x, y float64, blend types.BlendMode) {
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(x, y)
	op.Blend = blend.EbitenBlend()
	rt.Img.DrawImage(src, &op)
}

// NewSpriteNode creates a NodeTypeSprite with CustomImage pre-set to this
// texture. The returned node will display the RenderTexture contents.
func (rt *RenderTexture) NewSpriteNode(name string) *node.Node {
	if NewSpriteFn != nil {
		n := NewSpriteFn(name, types.TextureRegion{})
		n.CustomImage_ = rt.Img
		return n
	}
	n := node.NewNode(name, types.NodeTypeSprite)
	n.CustomImage_ = rt.Img
	return n
}

// RenderTextureDrawOpts controls how an image or sprite is drawn onto a
// RenderTexture when using the "Colored" draw methods.
type RenderTextureDrawOpts struct {
	X, Y           float64
	ScaleX, ScaleY float64
	Rotation       float64
	PivotX, PivotY float64
	Color          types.Color
	Alpha          float64
	BlendMode      types.BlendMode
}

// DrawSprite draws a TextureRegion from the atlas onto this texture at (x, y).
func (rt *RenderTexture) DrawSprite(region types.TextureRegion, x, y float64, blend types.BlendMode, pages []*ebiten.Image) {
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
	rt.Img.DrawImage(sub, &op)
}

// DrawSpriteColored draws a TextureRegion with full transform, color, and alpha.
func (rt *RenderTexture) DrawSpriteColored(region types.TextureRegion, opts RenderTextureDrawOpts, pages []*ebiten.Image) {
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
	ApplyDrawOpts(&op, opts, float64(region.OffsetX), float64(region.OffsetY))
	rt.Img.DrawImage(sub, &op)
}

// DrawImageColored draws a raw *ebiten.Image with full transform, color, and alpha.
func (rt *RenderTexture) DrawImageColored(img *ebiten.Image, opts RenderTextureDrawOpts) {
	var op ebiten.DrawImageOptions
	ApplyDrawOpts(&op, opts, 0, 0)
	rt.Img.DrawImage(img, &op)
}

// Resize deallocates the old image and creates a new one at the given dimensions.
func (rt *RenderTexture) Resize(width, height int) {
	if rt.Img != nil {
		rt.Img.Deallocate()
	}
	rt.Img = ebiten.NewImage(width, height)
	rt.W = width
	rt.H = height
}

// Dispose deallocates the underlying image.
func (rt *RenderTexture) Dispose() {
	if rt.Img != nil {
		rt.Img.Deallocate()
		rt.Img = nil
	}
}

// resolvePageImage returns the atlas page image for a region, handling the
// magenta placeholder sentinel.
func resolvePageImage(region types.TextureRegion, pages []*ebiten.Image) *ebiten.Image {
	if region.Page == MagentaPlaceholderPage {
		if MagentaImageFn != nil {
			return MagentaImageFn()
		}
		return nil
	}
	idx := int(region.Page)
	if idx < len(pages) {
		return pages[idx]
	}
	return nil
}

// ApplyDrawOpts configures an ebiten.DrawImageOptions from RenderTextureDrawOpts.
func ApplyDrawOpts(op *ebiten.DrawImageOptions, opts RenderTextureDrawOpts, offsetX, offsetY float64) {
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
	if c == (types.Color{}) {
		c = types.ColorWhite
	}
	op.ColorScale.Scale(
		float32(c.R()*c.A()*alpha),
		float32(c.G()*c.A()*alpha),
		float32(c.B()*c.A()*alpha),
		float32(c.A()*alpha),
	)
	op.Blend = opts.BlendMode.EbitenBlend()
}

// ColorToRGBA converts a types.Color to a color.Color (premultiplied) for image.Fill.
func ColorToRGBA(c types.Color) colorRGBA {
	return colorRGBA{
		R: uint8(Clamp01(c.R()*c.A()) * 255),
		G: uint8(Clamp01(c.G()*c.A()) * 255),
		B: uint8(Clamp01(c.B()*c.A()) * 255),
		A: uint8(Clamp01(c.A()) * 255),
	}
}

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

// Clamp01 clamps a float64 to [0, 1].
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
