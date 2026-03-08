package filter

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const colorMatrixShaderSrc = `//kage:unit pixels
package main

var Matrix [20]float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	// Un-premultiply alpha.
	if c.a > 0 {
		c.rgb /= c.a
	}
	// Apply 4x5 color matrix (row-major, offset in elements 4,9,14,19).
	r := Matrix[0]*c.r + Matrix[1]*c.g + Matrix[2]*c.b + Matrix[3]*c.a + Matrix[4]
	g := Matrix[5]*c.r + Matrix[6]*c.g + Matrix[7]*c.b + Matrix[8]*c.a + Matrix[9]
	b := Matrix[10]*c.r + Matrix[11]*c.g + Matrix[12]*c.b + Matrix[13]*c.a + Matrix[14]
	a := Matrix[15]*c.r + Matrix[16]*c.g + Matrix[17]*c.b + Matrix[18]*c.a + Matrix[19]
	// Clamp and re-premultiply.
	r = clamp(r, 0, 1)
	g = clamp(g, 0, 1)
	b = clamp(b, 0, 1)
	a = clamp(a, 0, 1)
	return vec4(r*a, g*a, b*a, a)
}
`

var colorMatrixShader *ebiten.Shader

func ensureColorMatrixShader() *ebiten.Shader {
	if colorMatrixShader == nil {
		s, err := ebiten.NewShader([]byte(colorMatrixShaderSrc))
		if err != nil {
			panic("willow: failed to compile color matrix shader: " + err.Error())
		}
		colorMatrixShader = s
	}
	return colorMatrixShader
}

// ColorMatrixFilter applies a 4x5 color matrix transformation using a Kage shader.
// The matrix is stored in row-major order: [R_r, R_g, R_b, R_a, R_offset, G_r, ...].
type ColorMatrixFilter struct {
	Matrix      [20]float64
	uniforms    map[string]any
	matrixF32   [20]float32 // persistent buffer to avoid per-frame slice escape
	matrixSlice []float32   // persistent slice header pointing into matrixF32
	shaderOp    ebiten.DrawRectShaderOptions
}

// NewColorMatrixFilter creates a color matrix filter initialized to the identity.
func NewColorMatrixFilter() *ColorMatrixFilter {
	f := &ColorMatrixFilter{
		uniforms: make(map[string]any, 1),
	}
	f.matrixSlice = f.matrixF32[:]
	f.uniforms["Matrix"] = f.matrixSlice
	// Identity matrix: diagonal = 1
	f.Matrix[0] = 1  // R_r
	f.Matrix[6] = 1  // G_g
	f.Matrix[12] = 1 // B_b
	f.Matrix[18] = 1 // A_a
	return f
}

// SetBrightness sets the matrix to adjust brightness by the given offset [-1, 1].
func (f *ColorMatrixFilter) SetBrightness(b float64) *ColorMatrixFilter {
	f.Matrix = [20]float64{
		1, 0, 0, 0, b,
		0, 1, 0, 0, b,
		0, 0, 1, 0, b,
		0, 0, 0, 1, 0,
	}
	return f
}

// SetContrast sets the matrix to adjust contrast. c=1 is normal, 0=gray, >1 is higher.
func (f *ColorMatrixFilter) SetContrast(c float64) *ColorMatrixFilter {
	t := (1.0 - c) / 2.0
	f.Matrix = [20]float64{
		c, 0, 0, 0, t,
		0, c, 0, 0, t,
		0, 0, c, 0, t,
		0, 0, 0, 1, 0,
	}
	return f
}

// SetSaturation sets the matrix to adjust saturation. s=1 is normal, 0=grayscale.
func (f *ColorMatrixFilter) SetSaturation(s float64) *ColorMatrixFilter {
	sr := (1 - s) * 0.299
	sg := (1 - s) * 0.587
	sb := (1 - s) * 0.114
	f.Matrix = [20]float64{
		sr + s, sg, sb, 0, 0,
		sr, sg + s, sb, 0, 0,
		sr, sg, sb + s, 0, 0,
		0, 0, 0, 1, 0,
	}
	return f
}

// SetHueRotation sets the matrix to rotate hue by the given angle in radians.
func (f *ColorMatrixFilter) SetHueRotation(radians float64) *ColorMatrixFilter {
	cos := math.Cos(radians)
	sin := math.Sin(radians)
	lr, lg, lb := 0.299, 0.587, 0.114
	f.Matrix = [20]float64{
		lr + cos*(1-lr) + sin*(-lr), lg + cos*(-lg) + sin*(-lg), lb + cos*(-lb) + sin*(1-lb), 0, 0,
		lr + cos*(-lr) + sin*(0.143), lg + cos*(1-lg) + sin*(0.140), lb + cos*(-lb) + sin*(-0.283), 0, 0,
		lr + cos*(-lr) + sin*(-(1 - lr)), lg + cos*(-lg) + sin*(lg), lb + cos*(1-lb) + sin*(lb), 0, 0,
		0, 0, 0, 1, 0,
	}
	return f
}

// SetInvert sets the matrix to invert RGB channels (negate and offset).
func (f *ColorMatrixFilter) SetInvert() *ColorMatrixFilter {
	f.Matrix = [20]float64{
		-1, 0, 0, 0, 1,
		0, -1, 0, 0, 1,
		0, 0, -1, 0, 1,
		0, 0, 0, 1, 0,
	}
	return f
}

// SetGrayscale sets the matrix to full grayscale (equivalent to SetSaturation(0)).
func (f *ColorMatrixFilter) SetGrayscale() *ColorMatrixFilter {
	return f.SetSaturation(0)
}

// Apply renders the color matrix transformation from src into dst.
func (f *ColorMatrixFilter) Apply(src, dst *ebiten.Image) {
	shader := ensureColorMatrixShader()
	for i, v := range f.Matrix {
		f.matrixF32[i] = float32(v)
	}
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns 0; color matrix transforms don't expand the image bounds.
func (f *ColorMatrixFilter) Padding() int { return 0 }
