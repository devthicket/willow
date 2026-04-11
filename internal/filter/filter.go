package filter

import "github.com/hajimehoshi/ebiten/v2"

// Filter is the interface for visual effects applied to a node's rendered output.
type Filter interface {
	// Apply renders src into dst with the filter effect.
	Apply(src, dst *ebiten.Image)
	// Padding returns the extra pixels needed around the source to accommodate
	// the effect (e.g. blur radius, outline thickness). Zero means no padding.
	Padding() int
}

// DrawFilter is an optional interface for per-pixel filters that can be
// applied at draw time without an offscreen render target. When a leaf
// sprite node has a single DrawFilter (no mask, no cache, padding == 0),
// the pipeline skips the offscreen RT and uses the filter's shader directly
// when drawing the sprite. This eliminates intermediate buffers and batch
// breaks for the common case of tinted/recolored sprites.
type DrawFilter interface {
	DrawShader() *ebiten.Shader
	DrawUniforms() map[string]any
	DrawImages() [3]*ebiten.Image // extra textures (Images[1..3]); Images[0] is the source
}

// MultiPass is an optional interface for filters that require multiple
// shader passes (e.g. iterative blur). When a filter implements MultiPass,
// the render pipeline calls Apply once per pass using its own pooled scratch
// images, so the filter needs no internal temporaries.
//
// Passes returns the total number of Apply calls needed. Zero means the
// filter is a no-op and will be skipped entirely.
// SetPass is called before each Apply with the zero-based pass index so
// the filter can update per-pass state (e.g. a shader uniform).
type MultiPass interface {
	Passes() int
	SetPass(pass int)
}

// InitShaders eagerly compiles all built-in filter shaders so that any
// compilation failure panics at startup rather than at an unpredictable frame.
func InitShaders() {
	ensureKawaseBlurShader()
	ensureOutlineShader()
	ensureColorMatrixShader()
	ensurePPOutlineShader()
	ensurePPInlineShader()
	ensurePaletteShader()
}

// ChainPadding returns the cumulative padding required by a slice of filters.
func ChainPadding(filters []Filter) int {
	pad := 0
	for _, f := range filters {
		pad += f.Padding()
	}
	return pad
}
