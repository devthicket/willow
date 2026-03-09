package types

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
