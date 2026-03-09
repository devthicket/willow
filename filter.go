package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/filter"
	"github.com/phanxgames/willow/internal/render"
)

// Filter is the interface for visual effects applied to a node's rendered output.
type Filter = filter.Filter

// --- Type aliases from internal/filter ---

// ColorMatrixFilter applies a 4x5 color matrix transformation using a Kage shader.
type ColorMatrixFilter = filter.ColorMatrixFilter

// BlurFilter applies a Kawase iterative blur using downscale/upscale passes.
type BlurFilter = filter.BlurFilter

// OutlineFilter draws the source in 8 cardinal/diagonal offsets with the outline
// color, then draws the original on top. Works at any thickness.
type OutlineFilter = filter.OutlineFilter

// PixelPerfectOutlineFilter uses a Kage shader to draw a 1-pixel outline
// around non-transparent pixels by testing cardinal neighbors.
type PixelPerfectOutlineFilter = filter.PixelPerfectOutlineFilter

// PixelPerfectInlineFilter uses a Kage shader to recolor edge pixels that
// border transparent areas.
type PixelPerfectInlineFilter = filter.PixelPerfectInlineFilter

// PaletteFilter remaps pixel colors through a 256-entry color palette based on
// luminance. Supports a cycle offset for palette animation.
type PaletteFilter = filter.PaletteFilter

// CustomShaderFilter wraps a user-provided Kage shader, exposing Ebitengine's
// shader system directly.
type CustomShaderFilter = filter.CustomShaderFilter

// --- Constructors re-exported ---

// NewColorMatrixFilter creates a color matrix filter initialized to the identity.
var NewColorMatrixFilter = filter.NewColorMatrixFilter

// NewBlurFilter creates a blur filter with the given radius (in pixels).
var NewBlurFilter = filter.NewBlurFilter

// NewOutlineFilter creates an outline filter.
var NewOutlineFilter = filter.NewOutlineFilter

// NewPixelPerfectOutlineFilter creates a pixel-perfect outline filter.
var NewPixelPerfectOutlineFilter = filter.NewPixelPerfectOutlineFilter

// NewPixelPerfectInlineFilter creates a pixel-perfect inline filter.
var NewPixelPerfectInlineFilter = filter.NewPixelPerfectInlineFilter

// NewPaletteFilter creates a palette filter with a default grayscale palette.
var NewPaletteFilter = filter.NewPaletteFilter

// NewCustomShaderFilter creates a custom shader filter with the given shader and padding.
var NewCustomShaderFilter = filter.NewCustomShaderFilter

// --- Filter padding helper ---

// filterChainPadding returns the cumulative padding required by a slice of filters.
// Per spec section 13.5: "Padding is cumulative: the initial RenderTexture is
// sized to accommodate the sum of all filters' Padding() values."
func filterChainPadding(filters []any) int {
	pad := 0
	for _, f := range filters {
		pad += f.(Filter).Padding()
	}
	return pad
}

// --- Filter application helper ---

// applyFilters runs a filter chain on src, ping-ponging between two images.
// Returns the image containing the final result (either src or the provided
// scratch image). The caller must handle releasing scratch if pooled.
func applyFilters(filters []any, src *ebiten.Image, pool *renderTexturePool) *ebiten.Image {
	return render.ApplyFiltersAny(filters, src, pool)
}
