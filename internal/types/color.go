package types

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

// RGBA8 returns the color as 8-bit RGBA components (0–255), premultiplied.
func (c Color) RGBA8() (r, g, b, a uint8) {
	r = uint8(Clamp01(c.r*c.a) * 255)
	g = uint8(Clamp01(c.g*c.a) * 255)
	b = uint8(Clamp01(c.b*c.a) * 255)
	a = uint8(Clamp01(c.a) * 255)
	return
}

// RGBA returns the alpha-premultiplied red, green, blue and alpha values
// scaled to [0, 0xFFFF]. This satisfies the image/color.Color interface.
func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(Clamp01(c.r*c.a) * 0xffff)
	g = uint32(Clamp01(c.g*c.a) * 0xffff)
	b = uint32(Clamp01(c.b*c.a) * 0xffff)
	a = uint32(Clamp01(c.a) * 0xffff)
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

// Clamp01 clamps a value to [0, 1].
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
