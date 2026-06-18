//go:build physics

package physics

import "github.com/jakecoffman/cp/v2"

// shapeKind partitions the pool by concrete cp.Shape type. Box and Polygon
// share skPoly because both are *cp.PolyShape under the hood.
type shapeKind uint8

const (
	skCircle shapeKind = iota
	skSegment
	skPoly
)

// Shape describes a collider attached to a body. Concrete implementations
// build the underlying *cp.Shape via the unexported buildShape; descriptors
// are passed by value through BodyDef and the package owns construction so
// future changes (e.g. pooling) don't break call sites.
type Shape interface {
	buildShape(body *cp.Body) *cp.Shape
	momentFor(mass float64) float64
	kind() shapeKind
	applyTo(s *cp.Shape)
}

type Circle struct {
	Radius float64
	Offset cp.Vector
}

func (c Circle) buildShape(body *cp.Body) *cp.Shape {
	return cp.NewCircle(body, c.Radius, c.Offset)
}

func (c Circle) momentFor(mass float64) float64 {
	return cp.MomentForCircle(mass, 0, c.Radius, c.Offset)
}

func (c Circle) kind() shapeKind { return skCircle }

func (c Circle) applyTo(s *cp.Shape) {
	// cp does not expose a setter for circle offset; the offset baked in
	// at construction sticks across pool reuse. Callers spawning many
	// circles with identical Offset (the common bullet/particle pattern)
	// pool cleanly; mixed offsets cause the pool to inherit the original.
	s.Class.(*cp.Circle).SetRadius(c.Radius)
}

type Box struct {
	Width, Height float64
	CornerRadius  float64
}

func (b Box) buildShape(body *cp.Body) *cp.Shape {
	return cp.NewBox(body, b.Width, b.Height, b.CornerRadius)
}

func (b Box) momentFor(mass float64) float64 {
	return cp.MomentForBox(mass, b.Width, b.Height)
}

func (b Box) kind() shapeKind { return skPoly }

func (b Box) applyTo(s *cp.Shape) {
	hw := b.Width / 2
	hh := b.Height / 2
	verts := [4]cp.Vector{
		{X: hw, Y: -hh},
		{X: hw, Y: hh},
		{X: -hw, Y: hh},
		{X: -hw, Y: -hh},
	}
	poly := s.Class.(*cp.PolyShape)
	poly.SetVerts(4, verts[:])
	poly.SetRadius(b.CornerRadius)
}

type Segment struct {
	A, B   cp.Vector
	Radius float64
}

func (s Segment) buildShape(body *cp.Body) *cp.Shape {
	return cp.NewSegment(body, s.A, s.B, s.Radius)
}

func (s Segment) momentFor(mass float64) float64 {
	return cp.MomentForSegment(mass, s.A, s.B, s.Radius)
}

func (s Segment) kind() shapeKind { return skSegment }

func (s Segment) applyTo(sh *cp.Shape) {
	seg := sh.Class.(*cp.Segment)
	seg.SetEndpoints(s.A, s.B)
	seg.SetRadius(s.Radius)
}

type Polygon struct {
	Verts        []cp.Vector
	CornerRadius float64
}

func (p Polygon) buildShape(body *cp.Body) *cp.Shape {
	return cp.NewPolyShapeRaw(body, len(p.Verts), p.Verts, p.CornerRadius)
}

func (p Polygon) momentFor(mass float64) float64 {
	return cp.MomentForPoly(mass, len(p.Verts), p.Verts, cp.Vector{}, p.CornerRadius)
}

func (p Polygon) kind() shapeKind { return skPoly }

func (p Polygon) applyTo(s *cp.Shape) {
	poly := s.Class.(*cp.PolyShape)
	poly.SetVerts(len(p.Verts), p.Verts)
	poly.SetRadius(p.CornerRadius)
}
