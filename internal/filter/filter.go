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

// ChainPadding returns the cumulative padding required by a slice of filters.
func ChainPadding(filters []Filter) int {
	pad := 0
	for _, f := range filters {
		pad += f.Padding()
	}
	return pad
}
