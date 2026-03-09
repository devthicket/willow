package text

import "github.com/phanxgames/willow/internal/types"

// TextEffects configures text effects (outline, glow, shadow) rendered in a
// single shader pass. All widths are in distance-field units (relative to the
// font's DistanceRange).
type TextEffects struct {
	OutlineWidth   float64 // 0 = no outline
	OutlineColor   types.Color
	GlowWidth      float64 // 0 = no glow
	GlowColor      types.Color
	ShadowOffset   types.Vec2 // (0,0) = no shadow
	ShadowColor    types.Color
	ShadowSoftness float64
}
