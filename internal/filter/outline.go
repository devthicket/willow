package filter

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/types"
)

// OutlineFilter draws the source in 8 cardinal/diagonal offsets with the outline
// color, then draws the original on top. Works at any thickness.
type OutlineFilter struct {
	Thickness int
	Color     types.Color
	imgOp     ebiten.DrawImageOptions
}

// NewOutlineFilter creates an outline filter.
func NewOutlineFilter(thickness int, c types.Color) *OutlineFilter {
	return &OutlineFilter{Thickness: thickness, Color: c}
}

// Apply draws an 8-direction offset outline behind the source image.
func (f *OutlineFilter) Apply(src, dst *ebiten.Image) {
	t := float64(f.Thickness)
	offsets := [8][2]float64{
		{-t, 0}, {t, 0}, {0, -t}, {0, t},
		{-t, -t}, {t, -t}, {-t, t}, {t, t},
	}

	op := &f.imgOp
	r, g, b, a := f.Color.R(), f.Color.G(), f.Color.B(), f.Color.A()

	for _, off := range offsets {
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Translate(off[0], off[1])
		op.ColorScale.Scale(
			float32(r*a),
			float32(g*a),
			float32(b*a),
			float32(a),
		)
		dst.DrawImage(src, op)
	}

	// Draw original on top
	op.GeoM.Reset()
	op.ColorScale.Reset()
	dst.DrawImage(src, op)
}

// Padding returns the outline thickness; the offscreen buffer is expanded by this amount.
func (f *OutlineFilter) Padding() int { return f.Thickness }

// --- PixelPerfectOutlineFilter ---

const pixelPerfectOutlineShaderSrc = `//kage:unit pixels
package main

var OutlineColor vec4

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		return c
	}
	// Check cardinal neighbors.
	if imageSrc0At(src + vec2(1, 0)).a > 0 ||
		imageSrc0At(src + vec2(-1, 0)).a > 0 ||
		imageSrc0At(src + vec2(0, 1)).a > 0 ||
		imageSrc0At(src + vec2(0, -1)).a > 0 {
		return OutlineColor
	}
	return vec4(0)
}
`

var ppOutlineShader *ebiten.Shader

func ensurePPOutlineShader() *ebiten.Shader {
	if ppOutlineShader == nil {
		s, err := ebiten.NewShader([]byte(pixelPerfectOutlineShaderSrc))
		if err != nil {
			panic("willow: failed to compile pixel-perfect outline shader: " + err.Error())
		}
		ppOutlineShader = s
	}
	return ppOutlineShader
}

// PixelPerfectOutlineFilter uses a Kage shader to draw a 1-pixel outline
// around non-transparent pixels by testing cardinal neighbors.
type PixelPerfectOutlineFilter struct {
	Color      types.Color
	uniforms   map[string]any
	colorF32   [4]float32 // persistent buffer
	colorSlice []float32  // persistent slice header
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewPixelPerfectOutlineFilter creates a pixel-perfect outline filter.
func NewPixelPerfectOutlineFilter(c types.Color) *PixelPerfectOutlineFilter {
	f := &PixelPerfectOutlineFilter{
		Color:    c,
		uniforms: make(map[string]any, 1),
	}
	f.colorSlice = f.colorF32[:]
	f.uniforms["OutlineColor"] = f.colorSlice
	return f
}

// Apply renders a 1-pixel outline via a Kage shader testing cardinal neighbors.
func (f *PixelPerfectOutlineFilter) Apply(src, dst *ebiten.Image) {
	shader := ensurePPOutlineShader()
	r, g, b, a := f.Color.R(), f.Color.G(), f.Color.B(), f.Color.A()
	f.colorF32[0] = float32(r * a)
	f.colorF32[1] = float32(g * a)
	f.colorF32[2] = float32(b * a)
	f.colorF32[3] = float32(a)
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns 1; the outline extends 1 pixel beyond the source bounds.
func (f *PixelPerfectOutlineFilter) Padding() int { return 1 }

// --- PixelPerfectInlineFilter ---

const pixelPerfectInlineShaderSrc = `//kage:unit pixels
package main

var InlineColor vec4

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a == 0 {
		return vec4(0)
	}
	// If any cardinal neighbor is transparent, this is an edge pixel.
	if imageSrc0At(src + vec2(1, 0)).a == 0 ||
		imageSrc0At(src + vec2(-1, 0)).a == 0 ||
		imageSrc0At(src + vec2(0, 1)).a == 0 ||
		imageSrc0At(src + vec2(0, -1)).a == 0 {
		return InlineColor
	}
	return c
}
`

var ppInlineShader *ebiten.Shader

func ensurePPInlineShader() *ebiten.Shader {
	if ppInlineShader == nil {
		s, err := ebiten.NewShader([]byte(pixelPerfectInlineShaderSrc))
		if err != nil {
			panic("willow: failed to compile pixel-perfect inline shader: " + err.Error())
		}
		ppInlineShader = s
	}
	return ppInlineShader
}

// PixelPerfectInlineFilter uses a Kage shader to recolor edge pixels that
// border transparent areas.
type PixelPerfectInlineFilter struct {
	Color      types.Color
	uniforms   map[string]any
	colorF32   [4]float32 // persistent buffer
	colorSlice []float32  // persistent slice header
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewPixelPerfectInlineFilter creates a pixel-perfect inline filter.
func NewPixelPerfectInlineFilter(c types.Color) *PixelPerfectInlineFilter {
	f := &PixelPerfectInlineFilter{
		Color:    c,
		uniforms: make(map[string]any, 1),
	}
	f.colorSlice = f.colorF32[:]
	f.uniforms["InlineColor"] = f.colorSlice
	return f
}

// Apply recolors edge pixels that border transparent areas via a Kage shader.
func (f *PixelPerfectInlineFilter) Apply(src, dst *ebiten.Image) {
	shader := ensurePPInlineShader()
	r, g, b, a := f.Color.R(), f.Color.G(), f.Color.B(), f.Color.A()
	f.colorF32[0] = float32(r * a)
	f.colorF32[1] = float32(g * a)
	f.colorF32[2] = float32(b * a)
	f.colorF32[3] = float32(a)
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns 0; inlines only affect existing opaque pixels.
func (f *PixelPerfectInlineFilter) Padding() int { return 0 }
