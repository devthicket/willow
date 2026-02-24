package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Filter is the interface for visual effects applied to a node's rendered output.
type Filter interface {
	// Apply renders src into dst with the filter effect.
	Apply(src, dst *ebiten.Image)
	// Padding returns the extra pixels needed around the source to accommodate
	// the effect (e.g. blur radius, outline thickness). Zero means no padding.
	Padding() int
}

// --- Kage shader sources ---
// All shaders use //kage:unit pixels as required by Ebitengine.
// Ebitengine uses premultiplied alpha; shaders un-premultiply before processing
// and re-premultiply output where needed.

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

// --- Lazy shader compilation (no sync.Once  -  willow is single-threaded) ---

var (
	colorMatrixShader *ebiten.Shader
	ppOutlineShader   *ebiten.Shader
	ppInlineShader    *ebiten.Shader
	paletteShader     *ebiten.Shader
)

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

// --- ColorMatrixFilter ---

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
func (f *ColorMatrixFilter) SetBrightness(b float64) {
	f.Matrix = [20]float64{
		1, 0, 0, 0, b,
		0, 1, 0, 0, b,
		0, 0, 1, 0, b,
		0, 0, 0, 1, 0,
	}
}

// SetContrast sets the matrix to adjust contrast. c=1 is normal, 0=gray, >1 is higher.
func (f *ColorMatrixFilter) SetContrast(c float64) {
	t := (1.0 - c) / 2.0
	f.Matrix = [20]float64{
		c, 0, 0, 0, t,
		0, c, 0, 0, t,
		0, 0, c, 0, t,
		0, 0, 0, 1, 0,
	}
}

// SetSaturation sets the matrix to adjust saturation. s=1 is normal, 0=grayscale.
func (f *ColorMatrixFilter) SetSaturation(s float64) {
	sr := (1 - s) * 0.299
	sg := (1 - s) * 0.587
	sb := (1 - s) * 0.114
	f.Matrix = [20]float64{
		sr + s, sg, sb, 0, 0,
		sr, sg + s, sb, 0, 0,
		sr, sg, sb + s, 0, 0,
		0, 0, 0, 1, 0,
	}
}

// Apply renders the color matrix transformation from src into dst.
func (f *ColorMatrixFilter) Apply(src, dst *ebiten.Image) {
	shader := ensureColorMatrixShader()
	// Convert [20]float64 to [20]float32 in-place (no allocation  -  matrixSlice
	// already points into matrixF32 and is pre-stored in the uniforms map).
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

// --- BlurFilter ---

// BlurFilter applies a Kawase iterative blur using downscale/upscale passes.
// No Kage shader needed  -  bilinear filtering during DrawImage does the work.
type BlurFilter struct {
	Radius int
	temps  []*ebiten.Image
	imgOp  ebiten.DrawImageOptions
}

// NewBlurFilter creates a blur filter with the given radius (in pixels).
func NewBlurFilter(radius int) *BlurFilter {
	if radius < 0 {
		radius = 0
	}
	return &BlurFilter{Radius: radius}
}

// Apply renders a Kawase blur from src into dst using iterative downscale/upscale.
func (f *BlurFilter) Apply(src, dst *ebiten.Image) {
	if f.Radius <= 0 {
		f.imgOp.GeoM.Reset()
		f.imgOp.ColorScale.Reset()
		f.imgOp.Filter = ebiten.FilterNearest
		dst.DrawImage(src, &f.imgOp)
		return
	}

	// Number of iterations: log2(radius), minimum 1.
	passes := int(math.Ceil(math.Log2(float64(f.Radius))))
	if passes < 1 {
		passes = 1
	}

	// Ensure we have enough temp images. Grow/shrink as needed.
	srcBounds := src.Bounds()
	w, h := srcBounds.Dx(), srcBounds.Dy()

	// We need 'passes' temp images for downscale + the same for upscale.
	// We can reuse the downscale chain for upscale.
	needed := passes
	for len(f.temps) < needed {
		f.temps = append(f.temps, nil)
	}
	// Deallocate excess temp images from previous larger radius.
	for i := needed; i < len(f.temps); i++ {
		if f.temps[i] != nil {
			f.temps[i].Deallocate()
			f.temps[i] = nil
		}
	}
	f.temps = f.temps[:needed]

	op := &f.imgOp

	// Downscale passes: each half-size
	current := src
	for i := 0; i < passes; i++ {
		w = max(w/2, 1)
		h = max(h/2, 1)
		if f.temps[i] == nil || f.temps[i].Bounds().Dx() != w || f.temps[i].Bounds().Dy() != h {
			if f.temps[i] != nil {
				f.temps[i].Deallocate()
			}
			f.temps[i] = ebiten.NewImage(w, h)
		} else {
			f.temps[i].Clear()
		}
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		op.GeoM.Scale(float64(w)/sw, float64(h)/sh)
		op.Filter = ebiten.FilterLinear
		f.temps[i].DrawImage(current, op)
		current = f.temps[i]
	}

	// Upscale passes: draw each back up
	for i := passes - 2; i >= 0; i-- {
		f.temps[i].Clear()
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		tw := float64(f.temps[i].Bounds().Dx())
		th := float64(f.temps[i].Bounds().Dy())
		op.GeoM.Scale(tw/sw, th/sh)
		op.Filter = ebiten.FilterLinear
		f.temps[i].DrawImage(current, op)
		current = f.temps[i]
	}

	// Final upscale to dst.
	op.GeoM.Reset()
	op.ColorScale.Reset()
	sw := float64(current.Bounds().Dx())
	sh := float64(current.Bounds().Dy())
	tw := float64(dst.Bounds().Dx())
	th := float64(dst.Bounds().Dy())
	op.GeoM.Scale(tw/sw, th/sh)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(current, op)
}

// Padding returns the blur radius; the offscreen buffer is expanded to avoid clipping.
func (f *BlurFilter) Padding() int { return f.Radius }

// --- OutlineFilter ---

// OutlineFilter draws the source in 8 cardinal/diagonal offsets with the outline
// color, then draws the original on top. Works at any thickness.
type OutlineFilter struct {
	Thickness int
	Color     Color
	imgOp     ebiten.DrawImageOptions
}

// NewOutlineFilter creates an outline filter.
func NewOutlineFilter(thickness int, c Color) *OutlineFilter {
	return &OutlineFilter{Thickness: thickness, Color: c}
}

// Apply draws an 8-direction offset outline behind the source image.
func (f *OutlineFilter) Apply(src, dst *ebiten.Image) {
	// 8-direction offsets scaled by thickness
	t := float64(f.Thickness)
	offsets := [8][2]float64{
		{-t, 0}, {t, 0}, {0, -t}, {0, t},
		{-t, -t}, {t, -t}, {-t, t}, {t, t},
	}

	op := &f.imgOp

	// Draw outline passes: tint the source with the outline color
	for _, off := range offsets {
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Translate(off[0], off[1])
		op.ColorScale.Scale(
			float32(f.Color.R*f.Color.A),
			float32(f.Color.G*f.Color.A),
			float32(f.Color.B*f.Color.A),
			float32(f.Color.A),
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

// PixelPerfectOutlineFilter uses a Kage shader to draw a 1-pixel outline
// around non-transparent pixels by testing cardinal neighbors.
type PixelPerfectOutlineFilter struct {
	Color      Color
	uniforms   map[string]any
	colorF32   [4]float32 // persistent buffer
	colorSlice []float32  // persistent slice header
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewPixelPerfectOutlineFilter creates a pixel-perfect outline filter.
func NewPixelPerfectOutlineFilter(c Color) *PixelPerfectOutlineFilter {
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
	// Premultiply the outline color for the shader (write in-place, no alloc).
	f.colorF32[0] = float32(f.Color.R * f.Color.A)
	f.colorF32[1] = float32(f.Color.G * f.Color.A)
	f.colorF32[2] = float32(f.Color.B * f.Color.A)
	f.colorF32[3] = float32(f.Color.A)
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns 1; the outline extends 1 pixel beyond the source bounds.
func (f *PixelPerfectOutlineFilter) Padding() int { return 1 }

// --- PixelPerfectInlineFilter ---

// PixelPerfectInlineFilter uses a Kage shader to recolor edge pixels that
// border transparent areas.
type PixelPerfectInlineFilter struct {
	Color      Color
	uniforms   map[string]any
	colorF32   [4]float32 // persistent buffer
	colorSlice []float32  // persistent slice header
	shaderOp   ebiten.DrawRectShaderOptions
}

// NewPixelPerfectInlineFilter creates a pixel-perfect inline filter.
func NewPixelPerfectInlineFilter(c Color) *PixelPerfectInlineFilter {
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
	// Write in-place, no alloc.
	f.colorF32[0] = float32(f.Color.R * f.Color.A)
	f.colorF32[1] = float32(f.Color.G * f.Color.A)
	f.colorF32[2] = float32(f.Color.B * f.Color.A)
	f.colorF32[3] = float32(f.Color.A)
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), shader, &f.shaderOp)
}

// Padding returns 0; inlines only affect existing opaque pixels.
func (f *PixelPerfectInlineFilter) Padding() int { return 0 }

// --- PaletteFilter ---

// PaletteFilter remaps pixel colors through a 256-entry color palette based on
// luminance. Supports a cycle offset for palette animation.
type PaletteFilter struct {
	Palette      [256]Color
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
	// Pre-store the constant PaletteSize; CycleOffset and TexWidth set each Apply.
	// Scalar float32 boxing is unavoidable with Ebitengine's uniform API.
	f.uniforms["PaletteSize"] = float32(256)
	// Initialize with a grayscale palette.
	for i := 0; i < 256; i++ {
		v := float64(i) / 255.0
		f.Palette[i] = Color{v, v, v, 1}
	}
	return f
}

// SetPalette sets the palette colors and marks the texture for rebuild.
func (f *PaletteFilter) SetPalette(palette [256]Color) {
	f.Palette = palette
	f.paletteDirty = true
}

// ensurePaletteTex rebuilds the palette texture to match the given dimensions.
// DrawRectShader requires all source images to have the same size, so the
// palette data is scaled across the full texture width.
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
	// Grow pixel buffer as needed.
	needed := w * h * 4
	if cap(f.pixBuf) < needed {
		f.pixBuf = make([]byte, needed)
	} else {
		f.pixBuf = f.pixBuf[:needed]
	}
	// Write palette data: 256 entries scaled across w pixels, repeated per row.
	for row := 0; row < h; row++ {
		for x := 0; x < w; x++ {
			idx := int((float64(x) + 0.5) * 256.0 / float64(w))
			if idx > 255 {
				idx = 255
			}
			c := f.Palette[idx]
			off := (row*w + x) * 4
			f.pixBuf[off+0] = byte(c.R*c.A*255 + 0.5)
			f.pixBuf[off+1] = byte(c.G*c.A*255 + 0.5)
			f.pixBuf[off+2] = byte(c.B*c.A*255 + 0.5)
			f.pixBuf[off+3] = byte(c.A*255 + 0.5)
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
	// Scalar float32 boxing is unavoidable with Ebitengine's uniform API,
	// but this only runs for nodes with PaletteFilter active.
	f.uniforms["CycleOffset"] = float32(f.CycleOffset)
	f.uniforms["TexWidth"] = float32(w)
	f.shaderOp.Images[0] = src
	f.shaderOp.Images[1] = f.paletteTex
	f.shaderOp.Uniforms = f.uniforms
	dst.DrawRectShader(w, h, shader, &f.shaderOp)
}

// Padding returns 0; palette remapping doesn't expand the image bounds.
func (f *PaletteFilter) Padding() int { return 0 }

// --- CustomShaderFilter ---

// CustomShaderFilter wraps a user-provided Kage shader, exposing Ebitengine's
// shader system directly. Images[0] is auto-filled with the source texture;
// the user may set Images[1] and Images[2] for additional textures.
type CustomShaderFilter struct {
	Shader   *ebiten.Shader
	Uniforms map[string]any
	Images   [3]*ebiten.Image
	padding  int
	shaderOp ebiten.DrawRectShaderOptions
}

// NewCustomShaderFilter creates a custom shader filter with the given shader and padding.
func NewCustomShaderFilter(shader *ebiten.Shader, padding int) *CustomShaderFilter {
	return &CustomShaderFilter{
		Shader:   shader,
		Uniforms: make(map[string]any),
		padding:  padding,
	}
}

// Apply runs the user-provided Kage shader with src as Images[0].
func (f *CustomShaderFilter) Apply(src, dst *ebiten.Image) {
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Images[1] = f.Images[1]
	f.shaderOp.Images[2] = f.Images[2]
	f.shaderOp.Uniforms = f.Uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), f.Shader, &f.shaderOp)
}

// Padding returns the padding value set at construction time.
func (f *CustomShaderFilter) Padding() int { return f.padding }

// --- Filter padding helper ---

// filterChainPadding returns the cumulative padding required by a slice of filters.
// Per spec section 13.5: "Padding is cumulative: the initial RenderTexture is
// sized to accommodate the sum of all filters' Padding() values."
func filterChainPadding(filters []Filter) int {
	pad := 0
	for _, f := range filters {
		pad += f.Padding()
	}
	return pad
}

// --- Filter application helper ---

// applyFilters runs a filter chain on src, ping-ponging between two images.
// Returns the image containing the final result (either src or the provided
// scratch image). The caller must handle releasing scratch if pooled.
func applyFilters(filters []Filter, src *ebiten.Image, pool *renderTexturePool) *ebiten.Image {
	if len(filters) == 0 {
		return src
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	current := src
	var scratch *ebiten.Image

	for _, f := range filters {
		if scratch == nil {
			scratch = pool.Acquire(w, h)
		} else {
			scratch.Clear()
		}
		f.Apply(current, scratch)
		current, scratch = scratch, current
	}

	return current
}
