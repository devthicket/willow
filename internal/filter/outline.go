package filter

import (
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Outline shader: searches a circular area of radius Thickness for any
// opaque neighbor. Returns the original pixel if opaque, the outline color
// if a nearby opaque pixel exists, or transparent otherwise.
// One DrawRectShader call replaces the old 9-DrawImage stamp approach,
// producing a true circular outline with optional anti-aliased edges.
const outlineShaderSrc = `//kage:unit pixels
package main

var OutlineColor vec4
var Thickness float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		return c
	}
	// Search a circular area for any opaque neighbor.
	// Kage requires constant loop bounds, so we use a const max and break.
	const MaxThickness = 32.0
	t := Thickness
	t2 := t * t
	minDist2 := t2 + 1.0
	for iy := 0; iy < int(MaxThickness*2+1); iy++ {
		y := float(iy) - MaxThickness
		if y < -t || y > t {
			continue
		}
		for ix := 0; ix < int(MaxThickness*2+1); ix++ {
			x := float(ix) - MaxThickness
			if x < -t || x > t {
				continue
			}
			d2 := x*x + y*y
			if d2 > t2 {
				continue
			}
			if imageSrc0At(src + vec2(x, y)).a > 0 {
				if d2 < minDist2 {
					minDist2 = d2
					if d2 <= 1.0 {
						// Adjacent pixel — can't get closer; skip remaining samples.
						return OutlineColor
					}
				}
			}
		}
	}
	if minDist2 <= t2 {
		// Smooth edge: anti-alias the last pixel of the outline.
		edge := clamp(t - sqrt(minDist2) + 0.5, 0.0, 1.0)
		return OutlineColor * edge
	}
	return vec4(0)
}
`

var outlineShader *ebiten.Shader

func ensureOutlineShader() *ebiten.Shader {
	if outlineShader == nil {
		s, err := ebiten.NewShader([]byte(outlineShaderSrc))
		if err != nil {
			panic("willow: failed to compile outline shader: " + err.Error())
		}
		outlineShader = s
	}
	return outlineShader
}

// OutlineFilter uses a Kage shader to draw a circular outline of arbitrary
// thickness around non-transparent pixels. One DrawRectShader call replaces
// the old 9-DrawImage stamp approach.
type OutlineFilter struct {
	Thickness  int
	Color      types.Color
	uniforms   map[string]any
	colorF32   [4]float32
	colorSlice []float32
	thickF32   [1]float32
	thickSlice []float32
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewOutlineFilter creates an outline filter. Thickness is clamped to [0, 32];
// for 1px outlines prefer PixelPerfectOutlineFilter which avoids the loop.
func NewOutlineFilter(thickness int, c types.Color) *OutlineFilter {
	if thickness < 0 {
		thickness = 0
	} else if thickness > 32 {
		thickness = 32
	}
	f := &OutlineFilter{
		Thickness: thickness,
		Color:     c,
		uniforms:  make(map[string]any, 2),
	}
	f.colorSlice = f.colorF32[:]
	f.thickSlice = f.thickF32[:]
	f.uniforms["OutlineColor"] = f.colorSlice
	f.uniforms["Thickness"] = f.thickSlice
	return f
}

// Apply draws a circular outline via a Kage shader.
func (f *OutlineFilter) Apply(src, dst *ebiten.Image) {
	shader := ensureOutlineShader()
	r, g, b, a := f.Color.R(), f.Color.G(), f.Color.B(), f.Color.A()
	f.colorF32[0] = float32(r * a)
	f.colorF32[1] = float32(g * a)
	f.colorF32[2] = float32(b * a)
	f.colorF32[3] = float32(a)
	f.thickF32[0] = float32(f.Thickness)
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
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
