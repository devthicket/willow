//go:build nophysics

// Package physics — nophysics stub. This file provides empty types matching
// the identifier set of the real package so the rest of Willow compiles
// without a cp dependency. The stub is paired with internal/node/physics_stub.go,
// which supplies panicking method bodies for the public verbs.
//
// No identifier here imports or depends on github.com/jakecoffman/cp/v2.
package physics

// Config is the zero-info stub of physics.Config. Constructing
// willow.PhysicsConfig{} compiles; passing a Gravity field does not (cp.Vector
// has no stub here on purpose — pulling cp back in defeats the build tag).
type Config struct{}

// BodyDef and Shape are open empty interfaces under nophysics so that user
// code constructing descriptor values still type-checks at the call sites
// (`SetBody(PhysicsDynamic{Shape: PhysicsCircle{Radius: 10}})`). The real
// package's interfaces are closed (unexported methods); both compile.
type BodyDef interface{}
type Shape interface{}

// Body is an opaque handle. Under nophysics it is never populated; the
// stub node methods return nil and the field on Node stays nil for the
// lifetime of the program.
type Body struct{}

// Dynamic / Static / Kinematic — value descriptors. Field shapes mirror the
// real package only for the cp-free fields so user struct literals like
// PhysicsDynamic{Mass: 1} keep compiling. cp.Vector-bearing fields (e.g.
// Circle.Offset, Segment.A/B, Polygon.Verts) are intentionally omitted.
type Dynamic struct {
	Shape      Shape
	Mass       float64
	Friction   float64
	Elasticity float64
	Moment     float64
}

type Static struct {
	Shape      Shape
	Friction   float64
	Elasticity float64
}

type Kinematic struct {
	Shape    Shape
	Friction float64
}

type Circle struct {
	Radius float64
}

type Box struct {
	Width, Height float64
	CornerRadius  float64
}

type Segment struct {
	Radius float64
}

type Polygon struct {
	CornerRadius float64
}

// PhysicsParent is referenced by the real internal/node physicsRoot field,
// which is excluded under nophysics. Defined here for symmetry with the
// real identifier set in case external tooling lists package contents.
type PhysicsParent struct{}
