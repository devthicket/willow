package willow

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/types"
)

// --- Color (kept in root — unexported fields accessed by animation, render) ---

// Color represents an RGBA color with components in [0, 1]. Not premultiplied.
// Premultiplication occurs at render submission time.
//
// Use [RGB] or [RGBA] to construct colors. RGB defaults alpha to 1 (fully opaque),
// eliminating the common mistake of forgetting alpha.
type Color struct {
	r, g, b, a float64
}

// RGB creates an opaque Color from red, green, blue components in [0, 1].
// Alpha is set to 1 (fully opaque).
func RGB(r, g, b float64) Color {
	return Color{r: r, g: g, b: b, a: 1}
}

// RGBA creates a Color from red, green, blue, alpha components in [0, 1].
func RGBA(r, g, b, a float64) Color {
	return Color{r: r, g: g, b: b, a: a}
}

// R returns the red component in [0, 1].
func (c Color) R() float64 { return c.r }

// G returns the green component in [0, 1].
func (c Color) G() float64 { return c.g }

// B returns the blue component in [0, 1].
func (c Color) B() float64 { return c.b }

// A returns the alpha component in [0, 1].
func (c Color) A() float64 { return c.a }

// RGBA returns the alpha-premultiplied red, green, blue and alpha values
// scaled to [0, 0xFFFF]. This satisfies the image/color.Color interface.
// Note: this is the interface method, not the RGBA constructor function.
func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(clamp01(c.r*c.a) * 0xffff)
	g = uint32(clamp01(c.g*c.a) * 0xffff)
	b = uint32(clamp01(c.b*c.a) * 0xffff)
	a = uint32(clamp01(c.a) * 0xffff)
	return
}

// ColorFromRGBA creates a Color from 8-bit RGBA components (0–255).
// The returned Color is non-premultiplied with components in [0, 1].
func ColorFromRGBA(r, g, b, a uint8) Color {
	return Color{
		r: float64(r) / 255,
		g: float64(g) / 255,
		b: float64(b) / 255,
		a: float64(a) / 255,
	}
}

// ColorFromHSV creates a Color from HSV values, all in [0, 1].
// Hue wraps (0 and 1 are both red), saturation 0 is grayscale, value 0 is black.
func ColorFromHSV(h, s, v float64) Color {
	h6 := h * 6
	i := int(h6)
	f := h6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch i % 6 {
	case 0:
		return Color{v, t, p, 1}
	case 1:
		return Color{q, v, p, 1}
	case 2:
		return Color{p, v, t, 1}
	case 3:
		return Color{p, q, v, 1}
	case 4:
		return Color{t, p, v, 1}
	default:
		return Color{v, p, q, 1}
	}
}

// ColorWhite is the default tint (no color modification).
var ColorWhite = Color{1, 1, 1, 1}

// ColorBlack is opaque black.
var ColorBlack = Color{0, 0, 0, 1}

// ColorTransparent is fully transparent black.
var ColorTransparent = Color{0, 0, 0, 0}

// --- Type aliases from internal/types ---

// Vec2 is a 2D vector used for positions, offsets, sizes, and directions
// throughout the API.
type Vec2 = types.Vec2

// Rect is an axis-aligned rectangle. The coordinate system has its origin at
// the top-left, with Y increasing downward.
type Rect = types.Rect

// Range is a general-purpose min/max range.
// Used by the particle system (EmitterConfig) and potentially other systems.
type Range = types.Range

// BlendMode selects a compositing operation. Each maps to a specific ebiten.Blend value.
type BlendMode = types.BlendMode

// Blend mode constants.
const (
	BlendNormal   = types.BlendNormal
	BlendAdd      = types.BlendAdd
	BlendMultiply = types.BlendMultiply
	BlendScreen   = types.BlendScreen
	BlendErase    = types.BlendErase
	BlendMask     = types.BlendMask
	BlendBelow    = types.BlendBelow
	BlendNone     = types.BlendNone
)

// NodeType distinguishes rendering behavior for a Node.
type NodeType = types.NodeType

// Node type constants.
const (
	NodeTypeContainer       = types.NodeTypeContainer
	NodeTypeSprite          = types.NodeTypeSprite
	NodeTypeMesh            = types.NodeTypeMesh
	NodeTypeParticleEmitter = types.NodeTypeParticleEmitter
	NodeTypeText            = types.NodeTypeText
)

// EventType identifies a kind of interaction event.
type EventType = types.EventType

// Event type constants.
const (
	EventPointerDown  = types.EventPointerDown
	EventPointerUp    = types.EventPointerUp
	EventPointerMove  = types.EventPointerMove
	EventClick        = types.EventClick
	EventDragStart    = types.EventDragStart
	EventDrag         = types.EventDrag
	EventDragEnd      = types.EventDragEnd
	EventPinch        = types.EventPinch
	EventPointerEnter = types.EventPointerEnter
	EventPointerLeave = types.EventPointerLeave
)

// eventBgClick is the internal background click event type.
var eventBgClick = types.EventBgClick

// MouseButton identifies a mouse button.
type MouseButton = types.MouseButton

// Mouse button constants.
const (
	MouseButtonLeft   = types.MouseButtonLeft
	MouseButtonRight  = types.MouseButtonRight
	MouseButtonMiddle = types.MouseButtonMiddle
)

// KeyModifiers is a bitmask of keyboard modifier keys.
// Values can be combined with bitwise OR (e.g. ModShift | ModCtrl).
type KeyModifiers = types.KeyModifiers

// Modifier key constants.
const (
	ModShift = types.ModShift
	ModCtrl  = types.ModCtrl
	ModAlt   = types.ModAlt
	ModMeta  = types.ModMeta
)

// TextAlign controls horizontal text alignment within a TextBlock.
type TextAlign = types.TextAlign

// Text alignment constants.
const (
	TextAlignLeft   = types.TextAlignLeft
	TextAlignCenter = types.TextAlignCenter
	TextAlignRight  = types.TextAlignRight
)

// TextureRegion describes a sub-rectangle within an atlas page.
// Value type (32 bytes) — stored directly on Node, no pointer.
type TextureRegion = types.TextureRegion

// CacheTreeMode controls how a cached subtree invalidates.
type CacheTreeMode = types.CacheTreeMode

// Cache tree mode constants.
const (
	CacheTreeManual = types.CacheTreeManual
	CacheTreeAuto   = types.CacheTreeAuto
)

// HitShape is implemented by custom hit-test shapes attached to a Node.
type HitShape = types.HitShape

// --- WhitePixel ---

// WhitePixel is a 1x1 white image used by default for solid color sprites.
var WhitePixel *ebiten.Image

func init() {
	WhitePixel = ebiten.NewImage(1, 1)
	WhitePixel.Fill(color.White)
}
