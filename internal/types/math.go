package types

import "math"

// Lerp linearly interpolates between a and b by t.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// Lerp32 linearly interpolates between a and b by t (float32).
func Lerp32(a, b, t float32) float32 {
	return a + (b-a)*t
}

// WorldAABB computes the axis-aligned bounding box for a rectangle of size
// (w, h) at the origin, transformed by the given affine matrix.
func WorldAABB(transform [6]float64, w, h float64) Rect {
	a, b, c, d, tx, ty := transform[0], transform[1], transform[2], transform[3], transform[4], transform[5]
	x0, y0 := tx, ty
	x1, y1 := a*w+tx, b*w+ty
	x2, y2 := c*h+tx, d*h+ty
	x3, y3 := a*w+c*h+tx, b*w+d*h+ty
	minX := math.Min(math.Min(x0, x1), math.Min(x2, x3))
	minY := math.Min(math.Min(y0, y1), math.Min(y2, y3))
	maxX := math.Max(math.Max(x0, x1), math.Max(x2, x3))
	maxY := math.Max(math.Max(y0, y1), math.Max(y2, y3))
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}
