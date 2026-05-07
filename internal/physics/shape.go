package physics

import "github.com/jakecoffman/cp/v2"

// Shape describes a collider attached to a body. Concrete implementations
// build the underlying *cp.Shape via the unexported buildShape; descriptors
// are passed by value through BodyDef and the package owns construction so
// future changes (e.g. pooling) don't break call sites.
type Shape interface {
	buildShape(body *cp.Body) *cp.Shape
	momentFor(mass float64) float64
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
