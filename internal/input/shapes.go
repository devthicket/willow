package input

import "github.com/phanxgames/willow/internal/types"

// HitRect is an axis-aligned rectangular hit area in local coordinates.
type HitRect struct {
	X, Y, Width, Height float64
}

// Contains reports whether (x, y) lies inside the rectangle.
func (r HitRect) Contains(x, y float64) bool {
	return x >= r.X && x <= r.X+r.Width &&
		y >= r.Y && y <= r.Y+r.Height
}

// HitCircle is a circular hit area in local coordinates.
type HitCircle struct {
	CenterX, CenterY, Radius float64
}

// Contains reports whether (x, y) lies inside or on the circle.
func (c HitCircle) Contains(x, y float64) bool {
	dx := x - c.CenterX
	dy := y - c.CenterY
	return dx*dx+dy*dy <= c.Radius*c.Radius
}

// HitPolygon is a convex polygon hit area in local coordinates.
type HitPolygon struct {
	Points []types.Vec2
}

// Contains reports whether (x, y) lies inside a convex polygon.
func (p HitPolygon) Contains(x, y float64) bool {
	n := len(p.Points)
	if n < 3 {
		return false
	}
	var positive, negative bool
	for i := 0; i < n; i++ {
		x1 := p.Points[i].X
		y1 := p.Points[i].Y
		j := (i + 1) % n
		x2 := p.Points[j].X
		y2 := p.Points[j].Y
		cross := (x2-x1)*(y-y1) - (y2-y1)*(x-x1)
		if cross > 0 {
			positive = true
		} else if cross < 0 {
			negative = true
		}
		if positive && negative {
			return false
		}
	}
	return true
}
