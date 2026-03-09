package willow

import "github.com/phanxgames/willow/internal/camera"

// --- Built-in HitShape types ---

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
// Points must define a convex polygon in either winding order.
// Concave polygons will produce incorrect results.
type HitPolygon struct {
	Points []Vec2
}

// nodeContainsLocal tests whether (lx, ly) falls inside a node's hit region.
// Uses HitShape if set; otherwise derives AABB from node dimensions.
// Containers with no HitShape are not hit-testable.
func nodeContainsLocal(n *Node, lx, ly float64) bool {
	if n.HitShape != nil {
		return n.HitShape.Contains(lx, ly)
	}
	w, h := camera.NodeDimensions(n)
	if w == 0 && h == 0 {
		return false
	}
	return lx >= 0 && lx <= w && ly >= 0 && ly <= h
}

// Contains reports whether (x, y) lies inside a convex polygon using cross-product sign test.
func (p HitPolygon) Contains(x, y float64) bool {
	n := len(p.Points)
	if n < 3 {
		return false
	}

	// Check that the point is on the same side of every edge.
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
