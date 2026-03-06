package willow

import "github.com/hajimehoshi/ebiten/v2"

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
	eventBgClick                       // internal: background click (no node hit)
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
