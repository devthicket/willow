package willow

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/types"
)

// Color represents an RGBA color with components in [0, 1]. Not premultiplied.
// Premultiplication occurs at render submission time.
type Color = types.Color

// Color constructors re-exported from internal/types.
var (
	RGB          = types.RGB
	ColorFromRGBA = types.ColorFromRGBA
	ColorFromHSV = types.ColorFromHSV
)

// RGBA creates a Color from red, green, blue, alpha components in [0, 1].
// Re-exported as a function rather than var to preserve the exact signature.
func RGBA(r, g, b, a float64) Color { return types.RGBA(r, g, b, a) }

// Color constants.
var (
	ColorWhite       = types.ColorWhite
	ColorBlack       = types.ColorBlack
	ColorTransparent = types.ColorTransparent
)

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
