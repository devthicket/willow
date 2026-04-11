package filter

import (
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- Circular distance outline shader (thickness ≤ 6) ---
//
// Single-pass: searches a circular area for the nearest opaque neighbor.
// True circular outline with anti-aliased edges. MaxThickness is capped at 6
// to keep the loop small (13×13 = 169 iterations max).

const outlineCircularShaderSrc = `//kage:unit pixels
package main

var OutlineColor vec4
var Thickness float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		return c
	}
	const MaxThickness = 6.0
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
						return OutlineColor
					}
				}
			}
		}
	}
	if minDist2 <= t2 {
		edge := clamp(t - sqrt(minDist2) + 0.5, 0.0, 1.0)
		return OutlineColor * edge
	}
	return vec4(0)
}
`

// --- Separable expansion shaders (thickness > 6) ---
//
// Two passes: horizontal then vertical. Each pass scans ±thickness along
// one axis. Opaque pixels pass through unchanged. Transparent pixels near
// an opaque neighbor get OutlineColor with distance-based alpha falloff.
// Total cost: 4×thickness texture fetches across 2 pipeline submissions.

const outlineHorzShaderSrc = `//kage:unit pixels
package main

var OutlineColor vec4
var Thickness float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		return c
	}
	const MaxThickness = 128.0
	t := Thickness
	bestDist := t + 1.0
	for i := 1; i < int(MaxThickness)+1; i++ {
		d := float(i)
		if d > t {
			break
		}
		if imageSrc0At(src + vec2(-d, 0)).a > 0 || imageSrc0At(src + vec2(d, 0)).a > 0 {
			bestDist = d
			break
		}
	}
	if bestDist <= t {
		edge := clamp(t - bestDist + 0.5, 0.0, 1.0)
		return OutlineColor * edge
	}
	return vec4(0)
}
`

const outlineVertShaderSrc = `//kage:unit pixels
package main

var OutlineColor vec4
var Thickness float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a > 0 {
		return c
	}
	const MaxThickness = 128.0
	t := Thickness
	bestDist := t + 1.0
	for i := 1; i < int(MaxThickness)+1; i++ {
		d := float(i)
		if d > t {
			break
		}
		if imageSrc0At(src + vec2(0, -d)).a > 0 || imageSrc0At(src + vec2(0, d)).a > 0 {
			bestDist = d
			break
		}
	}
	if bestDist <= t {
		edge := clamp(t - bestDist + 0.5, 0.0, 1.0)
		return OutlineColor * edge
	}
	return vec4(0)
}
`

// --- Shader singletons ---

var (
	outlineCircularShader *ebiten.Shader
	outlineHorzShader     *ebiten.Shader
	outlineVertShader     *ebiten.Shader
)

func ensureOutlineShader() *ebiten.Shader {
	if outlineCircularShader == nil {
		s, err := ebiten.NewShader([]byte(outlineCircularShaderSrc))
		if err != nil {
			panic("willow: failed to compile outline circular shader: " + err.Error())
		}
		outlineCircularShader = s
	}
	return outlineCircularShader
}

func ensureOutlineHorzShader() *ebiten.Shader {
	if outlineHorzShader == nil {
		s, err := ebiten.NewShader([]byte(outlineHorzShaderSrc))
		if err != nil {
			panic("willow: failed to compile outline horizontal shader: " + err.Error())
		}
		outlineHorzShader = s
	}
	return outlineHorzShader
}

func ensureOutlineVertShader() *ebiten.Shader {
	if outlineVertShader == nil {
		s, err := ebiten.NewShader([]byte(outlineVertShaderSrc))
		if err != nil {
			panic("willow: failed to compile outline vertical shader: " + err.Error())
		}
		outlineVertShader = s
	}
	return outlineVertShader
}

// outlineThreshold is the thickness at which the filter switches from the
// single-pass circular shader to the two-pass separable expansion.
const outlineThreshold = 6

// OutlineFilter draws an outline around non-transparent pixels.
//
// For thickness 1-6: single-pass circular distance shader (true circle, AA edges).
// For thickness 7+: two-pass separable expansion via MultiPass (horizontal then
// vertical, near-circular with distance falloff). The separable approach scales
// linearly with thickness instead of quadratically.
type OutlineFilter struct {
	Thickness  int
	Color      types.Color
	pass       int // current pass index (set by MultiPass.SetPass)
	uniforms   map[string]any
	colorF32   [4]float32
	colorSlice []float32
	thickF32   [1]float32
	thickSlice []float32
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewOutlineFilter creates an outline filter.
// For 1px outlines prefer PixelPerfectOutlineFilter which avoids all loops.
func NewOutlineFilter(thickness int, c types.Color) *OutlineFilter {
	if thickness < 0 {
		thickness = 0
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

// Passes returns the number of pipeline passes needed.
// Thickness ≤ 6: 1 pass (circular shader).
// Thickness > 6: 2 passes (horizontal + vertical expansion).
func (f *OutlineFilter) Passes() int {
	if f.Thickness <= 0 {
		return 0
	}
	if f.Thickness <= outlineThreshold {
		return 1
	}
	return 2
}

// SetPass configures the filter for the given pass index.
func (f *OutlineFilter) SetPass(pass int) {
	f.pass = pass
}

// Apply runs the appropriate shader for the current pass.
func (f *OutlineFilter) Apply(src, dst *ebiten.Image) {
	r, g, b, a := f.Color.R(), f.Color.G(), f.Color.B(), f.Color.A()
	f.colorF32[0] = float32(r * a)
	f.colorF32[1] = float32(g * a)
	f.colorF32[2] = float32(b * a)
	f.colorF32[3] = float32(a)
	f.thickF32[0] = float32(f.Thickness)

	var shader *ebiten.Shader
	if f.Thickness <= outlineThreshold {
		shader = ensureOutlineShader()
	} else if f.pass == 0 {
		shader = ensureOutlineHorzShader()
	} else {
		shader = ensureOutlineVertShader()
	}

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
