package physics

import (
	"math"
	"testing"

	"github.com/jakecoffman/cp/v2"
)

func newTestBody() *cp.Body { return cp.NewBody(1, 1) }

func TestShapeBuild_Circle(t *testing.T) {
	t.Parallel()
	c := Circle{Radius: 10, Offset: cp.Vector{X: 1, Y: 2}}
	shape := c.buildShape(newTestBody())
	cc, ok := shape.Class.(*cp.Circle)
	if !ok {
		t.Fatalf("expected *cp.Circle, got %T", shape.Class)
	}
	if cc.Radius() != 10 {
		t.Errorf("radius = %v, want 10", cc.Radius())
	}
}

func TestShapeBuild_Box(t *testing.T) {
	t.Parallel()
	b := Box{Width: 20, Height: 10, CornerRadius: 2}
	shape := b.buildShape(newTestBody())
	pp, ok := shape.Class.(*cp.PolyShape)
	if !ok {
		t.Fatalf("expected *cp.PolyShape, got %T", shape.Class)
	}
	if pp.Count() != 4 {
		t.Errorf("box vertex count = %d, want 4", pp.Count())
	}
}

func TestShapeBuild_Segment(t *testing.T) {
	t.Parallel()
	s := Segment{A: cp.Vector{X: 0, Y: 0}, B: cp.Vector{X: 10, Y: 0}, Radius: 1.5}
	shape := s.buildShape(newTestBody())
	seg, ok := shape.Class.(*cp.Segment)
	if !ok {
		t.Fatalf("expected *cp.Segment, got %T", shape.Class)
	}
	if seg.A() != (cp.Vector{X: 0, Y: 0}) || seg.B() != (cp.Vector{X: 10, Y: 0}) {
		t.Errorf("endpoints A=%v B=%v", seg.A(), seg.B())
	}
	if seg.Radius() != 1.5 {
		t.Errorf("radius = %v, want 1.5", seg.Radius())
	}
}

func TestShapeBuild_Polygon(t *testing.T) {
	t.Parallel()
	verts := []cp.Vector{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	p := Polygon{Verts: verts, CornerRadius: 0}
	shape := p.buildShape(newTestBody())
	poly, ok := shape.Class.(*cp.PolyShape)
	if !ok {
		t.Fatalf("expected *cp.PolyShape, got %T", shape.Class)
	}
	if poly.Count() != len(verts) {
		t.Errorf("vertex count = %d, want %d", poly.Count(), len(verts))
	}
}

func TestBodyDef_Dynamic_Moment_AutoCompute(t *testing.T) {
	t.Parallel()
	d := Dynamic{Shape: Circle{Radius: 10}, Mass: 1}
	body, _, kind := d.buildBody()
	if kind != kindDynamic {
		t.Errorf("kind = %v, want dynamic", kind)
	}
	want := cp.MomentForCircle(1, 0, 10, cp.Vector{})
	if math.Abs(body.Moment()-want) > 1e-9 {
		t.Errorf("moment = %v, want %v", body.Moment(), want)
	}
}

func TestBodyDef_Dynamic_Moment_Override(t *testing.T) {
	t.Parallel()
	d := Dynamic{Shape: Circle{Radius: 10}, Mass: 1, Moment: 5}
	body, _, _ := d.buildBody()
	if body.Moment() != 5 {
		t.Errorf("moment = %v, want 5 (explicit override)", body.Moment())
	}
}

func TestBodyDef_Static_BodyType(t *testing.T) {
	t.Parallel()
	body, _, kind := Static{Shape: Segment{A: cp.Vector{}, B: cp.Vector{X: 10}}}.buildBody()
	if kind != kindStatic {
		t.Errorf("kind = %v, want static", kind)
	}
	if body.GetType() != cp.BODY_STATIC {
		t.Errorf("body type = %v, want BODY_STATIC", body.GetType())
	}
}

func TestBodyDef_Kinematic_BodyType(t *testing.T) {
	t.Parallel()
	body, _, kind := Kinematic{Shape: Circle{Radius: 5}}.buildBody()
	if kind != kindKinematic {
		t.Errorf("kind = %v, want kinematic", kind)
	}
	if body.GetType() != cp.BODY_KINEMATIC {
		t.Errorf("body type = %v, want BODY_KINEMATIC", body.GetType())
	}
}

func TestPhysicsParent_GravityApplied(t *testing.T) {
	t.Parallel()
	p := NewPhysicsParent(Config{Gravity: cp.Vector{Y: 100}})
	if got := p.Space.Gravity(); got != (cp.Vector{Y: 100}) {
		t.Errorf("gravity = %v, want {0 100}", got)
	}
}

func TestPhysicsParent_DefaultIterations(t *testing.T) {
	t.Parallel()
	p := NewPhysicsParent(Config{})
	if p.Space.Iterations != 10 {
		t.Errorf("iterations = %d, want 10 (cp default)", p.Space.Iterations)
	}
}

func TestPhysicsParent_CustomIterations(t *testing.T) {
	t.Parallel()
	p := NewPhysicsParent(Config{Iterations: 25})
	if p.Space.Iterations != 25 {
		t.Errorf("iterations = %d, want 25", p.Space.Iterations)
	}
}
