package filter

import (
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

const paletteShaderSrc = `//kage:unit pixels
package main

var PaletteSize float
var CycleOffset float
var TexWidth float

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a == 0 {
		return vec4(0)
	}
	// Un-premultiply.
	if c.a > 0 {
		c.rgb /= c.a
	}
	// Luminance.
	lum := 0.299*c.r + 0.587*c.g + 0.114*c.b
	// Map lum [0,1] to index [0,255] with cycle offset.
	idx := lum*(PaletteSize-1.0) + CycleOffset
	idx = mod(idx, PaletteSize)
	// Look up in palette texture (scaled to match source dimensions).
	u := (idx + 0.5) / PaletteSize * TexWidth
	pal := imageSrc1At(vec2(u, 0.5))
	// Un-premultiply palette color.
	if pal.a > 0 {
		pal.rgb /= pal.a
	}
	// Re-premultiply with original alpha.
	return vec4(pal.rgb*c.a, c.a)
}
`

var paletteShader *ebiten.Shader

func ensurePaletteShader() *ebiten.Shader {
	if paletteShader == nil {
		s, err := ebiten.NewShader([]byte(paletteShaderSrc))
		if err != nil {
			panic("willow: failed to compile palette shader: " + err.Error())
		}
		paletteShader = s
	}
	return paletteShader
}

// PaletteFilter remaps pixel colors through a 256-entry color palette based on
// luminance. Supports a cycle offset for palette animation.
type PaletteFilter struct {
	Palette      [256]types.Color
	CycleOffset  float64
	paletteTex   *ebiten.Image
	paletteDirty bool
	texW, texH   int // current palette texture dimensions
	uniforms     map[string]any
	shaderOp     ebiten.DrawRectShaderOptions
	pixBuf       []byte // grows to match source dimensions
}

// NewPaletteFilter creates a palette filter with a default grayscale palette.
func NewPaletteFilter() *PaletteFilter {
	f := &PaletteFilter{
		paletteDirty: true,
		uniforms:     make(map[string]any, 3),
	}
	f.uniforms["PaletteSize"] = float32(256)
	// Initialize with a grayscale palette.
	for i := 0; i < 256; i++ {
		v := float64(i) / 255.0
		f.Palette[i] = types.RGBA(v, v, v, 1)
	}
	return f
}

// SetPalette sets the palette colors and marks the texture for rebuild.
func (f *PaletteFilter) SetPalette(palette [256]types.Color) {
	f.Palette = palette
	f.paletteDirty = true
}

// ensurePaletteTex rebuilds the palette texture to match the given dimensions.
func (f *PaletteFilter) ensurePaletteTex(w, h int) {
	sizeChanged := f.texW != w || f.texH != h
	if !f.paletteDirty && !sizeChanged && f.paletteTex != nil {
		return
	}
	if f.paletteTex == nil || sizeChanged {
		if f.paletteTex != nil {
			f.paletteTex.Deallocate()
		}
		f.paletteTex = ebiten.NewImage(w, h)
		f.texW = w
		f.texH = h
	}
	needed := w * h * 4
	if cap(f.pixBuf) < needed {
		f.pixBuf = make([]byte, needed)
	} else {
		f.pixBuf = f.pixBuf[:needed]
	}
	for row := 0; row < h; row++ {
		for x := 0; x < w; x++ {
			idx := int((float64(x) + 0.5) * 256.0 / float64(w))
			if idx > 255 {
				idx = 255
			}
			c := f.Palette[idx]
			r, g, b, a := c.R(), c.G(), c.B(), c.A()
			off := (row*w + x) * 4
			f.pixBuf[off+0] = byte(r*a*255 + 0.5)
			f.pixBuf[off+1] = byte(g*a*255 + 0.5)
			f.pixBuf[off+2] = byte(b*a*255 + 0.5)
			f.pixBuf[off+3] = byte(a*255 + 0.5)
		}
	}
	f.paletteTex.WritePixels(f.pixBuf)
	f.paletteDirty = false
}

// Apply remaps pixel colors through the palette based on luminance.
func (f *PaletteFilter) Apply(src, dst *ebiten.Image) {
	shader := ensurePaletteShader()
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	f.ensurePaletteTex(w, h)
	f.uniforms["CycleOffset"] = float32(f.CycleOffset)
	f.uniforms["TexWidth"] = float32(w)
	f.shaderOp.Images[0] = src
	f.shaderOp.Images[1] = f.paletteTex
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(w, h, shader, &f.shaderOp)
}

// Padding returns 0; palette remapping doesn't expand the image bounds.
func (f *PaletteFilter) Padding() int { return 0 }
