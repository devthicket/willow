package types

import "math"

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

// toHSV converts the color to HSV components. Hue is in [0, 1].
func (c Color) toHSV() (h, s, v float64) {
	r, g, b := c.r, c.g, c.b
	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}
	v = max
	if max == 0 {
		return 0, 0, v
	}
	s = (max - min) / max
	if max == min {
		return 0, s, v
	}
	d := max - min
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6
	return
}

// AdjustHue rotates the hue by degrees (0-360). Wraps around.
// Returns a new Color with the same saturation, value, and alpha.
func (c Color) AdjustHue(degrees float64) Color {
	h, s, v := c.toHSV()
	h += degrees / 360
	h -= math.Floor(h)
	out := ColorFromHSV(h, s, v)
	out.a = c.a
	return out
}

// AdjustSaturation multiplies the HSV saturation by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = grayscale, 2.0 = double saturation.
func (c Color) AdjustSaturation(factor float64) Color {
	h, s, v := c.toHSV()
	out := ColorFromHSV(h, Clamp01(s*factor), v)
	out.a = c.a
	return out
}

// AdjustValue multiplies the HSV value by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = black, 2.0 = double brightness.
func (c Color) AdjustValue(factor float64) Color {
	h, s, v := c.toHSV()
	out := ColorFromHSV(h, s, Clamp01(v*factor))
	out.a = c.a
	return out
}

// AdjustAlpha multiplies the alpha by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = fully transparent, 0.5 = half opacity.
func (c Color) AdjustAlpha(factor float64) Color {
	return Color{r: c.r, g: c.g, b: c.b, a: Clamp01(c.a * factor)}
}

// AdjustRed multiplies the red channel by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = no red, 2.0 = double red.
func (c Color) AdjustRed(factor float64) Color {
	return Color{r: Clamp01(c.r * factor), g: c.g, b: c.b, a: c.a}
}

// AdjustGreen multiplies the green channel by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = no green, 2.0 = double green.
func (c Color) AdjustGreen(factor float64) Color {
	return Color{r: c.r, g: Clamp01(c.g * factor), b: c.b, a: c.a}
}

// AdjustBlue multiplies the blue channel by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = no blue, 2.0 = double blue.
func (c Color) AdjustBlue(factor float64) Color {
	return Color{r: c.r, g: c.g, b: Clamp01(c.b * factor), a: c.a}
}

// AdjustBrightness multiplies all RGB channels by factor, clamped to [0, 1].
// 1.0 = no change, 0.0 = black, 1.5 = 50% brighter. Alpha is not affected.
func (c Color) AdjustBrightness(factor float64) Color {
	return Color{
		r: Clamp01(c.r * factor),
		g: Clamp01(c.g * factor),
		b: Clamp01(c.b * factor),
		a: c.a,
	}
}

// AdjustContrast scales each RGB channel's distance from the midpoint (0.5)
// by factor. Values above 1 increase contrast; values below 1 reduce it.
// 1.0 = no change. Alpha is not affected.
func (c Color) AdjustContrast(factor float64) Color {
	return Color{
		r: Clamp01((c.r-0.5)*factor + 0.5),
		g: Clamp01((c.g-0.5)*factor + 0.5),
		b: Clamp01((c.b-0.5)*factor + 0.5),
		a: c.a,
	}
}

// ColorFieldPtrs returns pointers to the individual color component fields.
// Used by tween code that needs to mutate color fields via pointer.
func ColorFieldPtrs(c *Color) (r, g, b, a *float64) {
	return &c.r, &c.g, &c.b, &c.a
}

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
