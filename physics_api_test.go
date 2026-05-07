//go:build !nophysics

package willow_test

import (
	"testing"

	"github.com/devthicket/willow"
	"github.com/jakecoffman/cp/v2"
)

// TestPublicAPI_EnablePhysicsAndStep exercises the physics surface using
// only willow.* symbols — proves a user can build a working physics scene
// without reaching into internal packages.
func TestPublicAPI_EnablePhysicsAndStep(t *testing.T) {
	root := willow.NewContainer("root")
	ball := willow.NewContainer("ball")
	root.AddChild(ball)

	root.EnablePhysics(willow.PhysicsConfig{Gravity: cp.Vector{X: 0, Y: 900}})
	defer root.DisablePhysics()

	ball.SetBody(willow.PhysicsDynamic{
		Shape: willow.PhysicsCircle{Radius: 8},
		Mass:  1,
	})

	startY := ball.Y()
	const steps = 30
	const dt = 1.0 / 60.0
	for i := 0; i < steps; i++ {
		ball.StepPhysics(dt)
	}
	if ball.Y() <= startY {
		t.Fatalf("ball did not fall under gravity: startY=%v endY=%v", startY, ball.Y())
	}
}

// TestPublicAPI_BodyAccess verifies the embedded *cp.Body API is reachable
// through willow.PhysicsBody returned by GetBody.
func TestPublicAPI_BodyAccess(t *testing.T) {
	root := willow.NewContainer("root")
	n := willow.NewContainer("n")
	root.AddChild(n)

	root.EnablePhysics(willow.PhysicsConfig{})
	defer root.DisablePhysics()

	n.SetBody(willow.PhysicsDynamic{
		Shape: willow.PhysicsBox{Width: 10, Height: 10},
		Mass:  1,
	})

	body := n.GetBody()
	if body == nil {
		t.Fatal("GetBody returned nil")
	}
	body.ApplyImpulseAtLocalPoint(cp.Vector{X: 0, Y: -500}, cp.Vector{})
	if v := body.Velocity(); v.Y >= 0 {
		t.Fatalf("expected upward velocity after impulse, got %v", v)
	}
}

// Compile-time assertion: every exported physics alias resolves and the
// shape descriptors satisfy willow.PhysicsShape.
var (
	_ willow.PhysicsBodyDef = willow.PhysicsDynamic{}
	_ willow.PhysicsBodyDef = willow.PhysicsStatic{}
	_ willow.PhysicsBodyDef = willow.PhysicsKinematic{}
	_ willow.PhysicsShape   = willow.PhysicsCircle{}
	_ willow.PhysicsShape   = willow.PhysicsBox{}
	_ willow.PhysicsShape   = willow.PhysicsSegment{}
	_ willow.PhysicsShape   = willow.PhysicsPolygon{}
)
