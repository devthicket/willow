package filter

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Iterative blur shader: samples 4 diagonal taps per pass with manual
// bilinear interpolation (Ebitengine's imageSrc0At is point-sampled).
// Each pass uses an increasing Offset (iteration + 0.5), and multiple
// passes at full resolution produce a progressive blur. This replaces
// the old downscale/upscale approach that thrashed Ebitengine's atlas
// with varying-size temporary images.
const blurShaderSrc = `//kage:unit pixels
package main

var Offset float

// bilinear does manual bilinear sampling so we get sub-texel interpolation
// from point-sampled imageSrc0At.
func bilinear(pos vec2) vec4 {
	p := pos - 0.5
	f := fract(p)
	i := floor(p) + 0.5
	return mix(
		mix(imageSrc0At(i), imageSrc0At(i+vec2(1, 0)), f.x),
		mix(imageSrc0At(i+vec2(0, 1)), imageSrc0At(i+vec2(1, 1)), f.x),
		f.y,
	)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	s := bilinear(src + vec2(-Offset, -Offset))
	s += bilinear(src + vec2(Offset, -Offset))
	s += bilinear(src + vec2(-Offset, Offset))
	s += bilinear(src + vec2(Offset, Offset))
	return s / 4.0
}
`

var blurShader *ebiten.Shader

func ensureBlurShader() *ebiten.Shader {
	if blurShader == nil {
		s, err := ebiten.NewShader([]byte(blurShaderSrc))
		if err != nil {
			panic("willow: failed to compile blur shader: " + err.Error())
		}
		blurShader = s
	}
	return blurShader
}

// BlurFilter applies an iterative multi-pass blur using a Kage shader.
// Each pass samples 4 diagonal taps with manual bilinear interpolation.
// It implements MultiPass so the render pipeline drives the ping-pong
// externally — the filter owns zero temporary images.
type BlurFilter struct {
	Radius      int
	uniforms    map[string]any
	offsetBuf   [1]float32  // persistent buffer for the Offset uniform
	offsetSlice []float32   // persistent slice header into offsetBuf
	shaderOp    ebiten.DrawRectShaderOptions
}

// NewBlurFilter creates a blur filter with the given radius (in pixels).
func NewBlurFilter(radius int) *BlurFilter {
	if radius < 0 {
		radius = 0
	}
	f := &BlurFilter{
		Radius:   radius,
		uniforms: make(map[string]any, 1),
	}
	f.offsetSlice = f.offsetBuf[:]
	f.uniforms["Offset"] = f.offsetSlice
	return f
}

// blurPasses returns the number of shader passes for the given radius.
// Each pass with offset i+0.5 blurs by roughly i+1 pixels, so cumulative
// blur grows quadratically. passes ≈ ceil(sqrt(2·radius)) gives a good fit.
func blurPasses(radius int) int {
	if radius <= 0 {
		return 0
	}
	p := int(math.Ceil(math.Sqrt(2.0 * float64(radius))))
	if p < 1 {
		return 1
	}
	return p
}

// Passes returns the number of shader iterations needed for this blur radius.
// Zero means the filter is a no-op (radius ≤ 0).
func (f *BlurFilter) Passes() int { return blurPasses(f.Radius) }

// SetPass configures the diagonal offset for the given pass index.
func (f *BlurFilter) SetPass(pass int) {
	f.offsetBuf[0] = float32(pass) + 0.5
}

// Apply runs a single blur pass from src into dst.
// The render pipeline calls this once per pass via the MultiPass interface.
func (f *BlurFilter) Apply(src, dst *ebiten.Image) {
	shader := ensureBlurShader()
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns the blur radius; the offscreen buffer is expanded to avoid clipping.
func (f *BlurFilter) Padding() int { return f.Radius }
