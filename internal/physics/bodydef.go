//go:build physics

package physics

import "github.com/jakecoffman/cp/v2"

// bodyKind identifies the cp body type without leaking cp constants through
// our API. It also serves as part of the pool key.
type bodyKind uint8

const (
	kindDynamic bodyKind = iota
	kindStatic
	kindKinematic
)

// BodyDef constructs a cp body and its primary shape. Implementations are
// the Dynamic / Static / Kinematic value types; the unexported buildBody
// keeps the interface closed to this package.
type BodyDef interface {
	buildBody() (*cp.Body, *cp.Shape, bodyKind)
	kinds() (bodyKind, shapeKind)
	apply(b *Body, x, y, angle float64)
}

// Dynamic bodies respond to forces, gravity, and collisions.
//
// Moment of 0 auto-computes from the shape (cp.MomentForCircle/Box/...);
// pass an explicit Moment to override.
type Dynamic struct {
	Shape      Shape
	Mass       float64
	Friction   float64
	Elasticity float64
	Moment     float64
}

func (d Dynamic) buildBody() (*cp.Body, *cp.Shape, bodyKind) {
	moment := d.Moment
	if moment == 0 {
		moment = d.Shape.momentFor(d.Mass)
	}
	body := cp.NewBody(d.Mass, moment)
	shape := d.Shape.buildShape(body)
	shape.SetFriction(d.Friction)
	shape.SetElasticity(d.Elasticity)
	return body, shape, kindDynamic
}

func (d Dynamic) kinds() (bodyKind, shapeKind) { return kindDynamic, d.Shape.kind() }

func (d Dynamic) apply(b *Body, x, y, angle float64) {
	moment := d.Moment
	if moment == 0 {
		moment = d.Shape.momentFor(d.Mass)
	}
	b.Body.SetMass(d.Mass)
	b.Body.SetMoment(moment)
	d.Shape.applyTo(b.Shape)
	b.Shape.SetFriction(d.Friction)
	b.Shape.SetElasticity(d.Elasticity)
	b.Body.SetPosition(cp.Vector{X: x, Y: y})
	b.Body.SetAngle(angle)
}

// Static bodies never move. They form the unmovable scenery (floors, walls).
type Static struct {
	Shape      Shape
	Friction   float64
	Elasticity float64
}

func (s Static) buildBody() (*cp.Body, *cp.Shape, bodyKind) {
	body := cp.NewStaticBody()
	shape := s.Shape.buildShape(body)
	shape.SetFriction(s.Friction)
	shape.SetElasticity(s.Elasticity)
	return body, shape, kindStatic
}

func (s Static) kinds() (bodyKind, shapeKind) { return kindStatic, s.Shape.kind() }

func (s Static) apply(b *Body, x, y, angle float64) {
	s.Shape.applyTo(b.Shape)
	b.Shape.SetFriction(s.Friction)
	b.Shape.SetElasticity(s.Elasticity)
	b.Body.SetPosition(cp.Vector{X: x, Y: y})
	b.Body.SetAngle(angle)
}

// Kinematic bodies are user-driven (via SetPosition / velocity); they push
// dynamic bodies but are unaffected by forces or collisions themselves.
type Kinematic struct {
	Shape    Shape
	Friction float64
}

func (k Kinematic) buildBody() (*cp.Body, *cp.Shape, bodyKind) {
	body := cp.NewKinematicBody()
	shape := k.Shape.buildShape(body)
	shape.SetFriction(k.Friction)
	return body, shape, kindKinematic
}

func (k Kinematic) kinds() (bodyKind, shapeKind) { return kindKinematic, k.Shape.kind() }

func (k Kinematic) apply(b *Body, x, y, angle float64) {
	k.Shape.applyTo(b.Shape)
	b.Shape.SetFriction(k.Friction)
	b.Body.SetPosition(cp.Vector{X: x, Y: y})
	b.Body.SetAngle(angle)
}
