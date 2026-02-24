package willow

import "github.com/hajimehoshi/ebiten/v2"

// Color represents an RGBA color with components in [0, 1]. Not premultiplied.
// Premultiplication occurs at render submission time.
type Color struct {
	R, G, B, A float64
}

// ColorWhite is the default tint (no color modification).
var ColorWhite = Color{1, 1, 1, 1}

// Vec2 is a 2D vector used for positions, offsets, sizes, and directions
// throughout the API.
type Vec2 struct {
	X, Y float64
}

// WhitePixel is a 1x1 white image used by default for solid color sprites.
var WhitePixel *ebiten.Image

func init() {
	WhitePixel = ebiten.NewImage(1, 1)
	WhitePixel.Fill(ColorWhite.toRGBA())
}

// Rect is an axis-aligned rectangle. The coordinate system has its origin at
// the top-left, with Y increasing downward.
type Rect struct {
	X, Y, Width, Height float64
}

// Contains reports whether the point (x, y) lies inside the rectangle.
// Points on the edge are considered inside.
func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x <= r.X+r.Width &&
		y >= r.Y && y <= r.Y+r.Height
}

// Intersects reports whether r and other overlap.
// Adjacent rectangles (sharing only an edge) are considered intersecting.
func (r Rect) Intersects(other Rect) bool {
	return r.X <= other.X+other.Width &&
		r.X+r.Width >= other.X &&
		r.Y <= other.Y+other.Height &&
		r.Y+r.Height >= other.Y
}

// Range is a general-purpose min/max range.
// Used by the particle system (EmitterConfig) and potentially other systems.
type Range struct {
	Min, Max float64
}

// BlendMode selects a compositing operation. Each maps to a specific ebiten.Blend value.
type BlendMode uint8

const (
	BlendNormal   BlendMode = iota // source-over (standard alpha blending)
	BlendAdd                       // additive / lighter
	BlendMultiply                  // multiply (source * destination; only darkens)
	BlendScreen                    // screen (1 - (1-src)*(1-dst); only brightens)
	BlendErase                     // destination-out (punch transparent holes)
	BlendMask                      // clip destination to source alpha
	BlendBelow                     // destination-over (draw behind existing content)
	BlendNone                      // opaque copy (skip blending)
)

// EbitenBlend returns the ebiten.Blend value corresponding to this BlendMode.
func (b BlendMode) EbitenBlend() ebiten.Blend {
	switch b {
	case BlendNormal:
		return ebiten.BlendSourceOver
	case BlendAdd:
		return ebiten.BlendLighter
	case BlendMultiply:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
			BlendFactorSourceAlpha:      ebiten.BlendFactorDestinationAlpha,
			BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
			BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendScreen:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorOne,
			BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
			BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceColor,
			BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendErase:
		return ebiten.BlendDestinationOut
	case BlendMask:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorZero,
			BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
			BlendFactorDestinationRGB:   ebiten.BlendFactorSourceAlpha,
			BlendFactorDestinationAlpha: ebiten.BlendFactorSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendBelow:
		return ebiten.BlendDestinationOver
	case BlendNone:
		return ebiten.BlendCopy
	default:
		return ebiten.BlendSourceOver
	}
}

// NodeType distinguishes rendering behavior for a Node.
type NodeType uint8

const (
	NodeTypeContainer       NodeType = iota // group node with no visual output
	NodeTypeSprite                          // renders a TextureRegion or custom image
	NodeTypeMesh                            // renders arbitrary triangles via DrawTriangles
	NodeTypeParticleEmitter                 // CPU-simulated particle system
	NodeTypeText                            // renders text via SpriteFont
)

// EventType identifies a kind of interaction event.
type EventType uint8

const (
	EventPointerDown  EventType = iota // fires when a pointer button is pressed
	EventPointerUp                     // fires when a pointer button is released
	EventPointerMove                   // fires when the pointer moves (hover, no button)
	EventClick                         // fires on press then release over the same node
	EventDragStart                     // fires when movement exceeds the drag dead zone
	EventDrag                          // fires each frame while dragging
	EventDragEnd                       // fires when the pointer is released after dragging
	EventPinch                         // fires during a two-finger pinch/rotate gesture
	EventPointerEnter                  // fires when the pointer enters a node's bounds
	EventPointerLeave                  // fires when the pointer leaves a node's bounds
)

// MouseButton identifies a mouse button.
type MouseButton uint8

const (
	MouseButtonLeft   MouseButton = iota // primary (left) mouse button
	MouseButtonRight                     // secondary (right) mouse button
	MouseButtonMiddle                    // middle mouse button (scroll wheel click)
)

// KeyModifiers is a bitmask of keyboard modifier keys.
// Values can be combined with bitwise OR (e.g. ModShift | ModCtrl).
type KeyModifiers uint8

const (
	ModShift KeyModifiers = 1 << iota // Shift key
	ModCtrl                           // Control key
	ModAlt                            // Alt / Option key
	ModMeta                           // Meta / Command / Windows key
)

// TextAlign controls horizontal text alignment within a TextBlock.
type TextAlign uint8

const (
	TextAlignLeft   TextAlign = iota // align text to the left edge (default)
	TextAlignCenter                  // center text horizontally
	TextAlignRight                   // align text to the right edge
)
