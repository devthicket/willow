//go:build !nophysics

package physics

import (
	"testing"

	"github.com/jakecoffman/cp/v2"
)

func setupPool(t *testing.T) *PhysicsParent {
	t.Helper()
	PoolReset()
	return NewPhysicsParent(Config{Gravity: cp.Vector{Y: 100}})
}

func dynCircleDef() Dynamic {
	return Dynamic{Shape: Circle{Radius: 5}, Mass: 1, Friction: 0.3, Elasticity: 0.2}
}

func TestPool_BodyReused(t *testing.T) {
	p := setupPool(t)

	a := AcquireBody(dynCircleDef(), 10, 20, 0)
	p.AddBody(a)
	cpBody := a.Body
	cpShape := a.Shape

	p.RemoveBody(a)
	ReleaseBody(a)

	b := AcquireBody(dynCircleDef(), 30, 40, 0)
	if b.Body != cpBody {
		t.Errorf("expected cp body to be reused; got new pointer")
	}
	if b.Shape != cpShape {
		t.Errorf("expected cp shape to be reused; got new pointer")
	}
}

func TestPool_ShapeReused_Circle(t *testing.T) {
	setupPool(t)
	a := AcquireBody(dynCircleDef(), 0, 0, 0)
	original := a.Shape
	ReleaseBody(a)

	b := AcquireBody(Dynamic{Shape: Circle{Radius: 9}, Mass: 1}, 0, 0, 0)
	if b.Shape != original {
		t.Errorf("circle shape should have been pulled from the pool")
	}
	if got := b.Shape.Class.(*cp.Circle).Radius(); got != 9 {
		t.Errorf("radius after reuse = %v, want 9", got)
	}
}

func TestPool_BodyReset_NoStateLeak(t *testing.T) {
	setupPool(t)
	a := AcquireBody(dynCircleDef(), 0, 0, 0)
	a.Body.SetVelocityVector(cp.Vector{X: 5, Y: 5})
	a.Body.SetAngularVelocity(2)
	a.Body.SetForce(cp.Vector{X: 7, Y: 0})
	a.Body.SetTorque(3)
	ReleaseBody(a)

	b := AcquireBody(dynCircleDef(), 0, 0, 0)
	if v := b.Body.Velocity(); v.X != 0 || v.Y != 0 {
		t.Errorf("velocity leaked across pool: %v", v)
	}
	if w := b.Body.AngularVelocity(); w != 0 {
		t.Errorf("angular velocity leaked: %v", w)
	}
	if f := b.Body.Force(); f.X != 0 || f.Y != 0 {
		t.Errorf("force leaked: %v", f)
	}
	if tq := b.Body.Torque(); tq != 0 {
		t.Errorf("torque leaked: %v", tq)
	}
}

func TestPool_KindIsolation(t *testing.T) {
	setupPool(t)
	staticDef := Static{Shape: Segment{A: cp.Vector{}, B: cp.Vector{X: 10}}}
	a := AcquireBody(staticDef, 0, 0, 0)
	staticBody := a.Body
	ReleaseBody(a)

	dyn := AcquireBody(dynCircleDef(), 0, 0, 0)
	if dyn.Body == staticBody {
		t.Errorf("dynamic request reused a static body")
	}
}

func TestPool_DispatchPath(t *testing.T) {
	setupPool(t)
	box := AcquireBody(Dynamic{Shape: Box{Width: 5, Height: 5}, Mass: 1}, 0, 0, 0)
	circle := AcquireBody(Dynamic{Shape: Circle{Radius: 3}, Mass: 1}, 0, 0, 0)
	seg := AcquireBody(Static{Shape: Segment{A: cp.Vector{}, B: cp.Vector{X: 1}}}, 0, 0, 0)
	ReleaseBody(box)
	ReleaseBody(circle)
	ReleaseBody(seg)

	if got := pools[poolKey{kindDynamic, skPoly}]; len(got) != 1 {
		t.Errorf("box should land in the dynamic+poly pool; got %d", len(got))
	}
	if got := pools[poolKey{kindDynamic, skCircle}]; len(got) != 1 {
		t.Errorf("circle should land in the dynamic+circle pool; got %d", len(got))
	}
	if got := pools[poolKey{kindStatic, skSegment}]; len(got) != 1 {
		t.Errorf("segment should land in the static+segment pool; got %d", len(got))
	}

	// Polygon (Verts) reuses the same poly pool as Box.
	AcquireBody(Dynamic{
		Shape: Polygon{Verts: []cp.Vector{{X: -1, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 1}, {X: -1, Y: 1}}},
		Mass:  1,
	}, 0, 0, 0)
	if got := pools[poolKey{kindDynamic, skPoly}]; len(got) != 0 {
		t.Errorf("polygon should have drained the box from the pool")
	}
}

func TestPool_UserDataAndFilterCleared(t *testing.T) {
	setupPool(t)
	type marker struct{ id int }

	a := AcquireBody(dynCircleDef(), 0, 0, 0)
	a.Body.UserData = &marker{id: 7}
	a.Shape.UserData = &marker{id: 9}
	a.Shape.SetSensor(true)
	a.Shape.SetCollisionType(42)
	a.Shape.SetFilter(cp.ShapeFilter{Group: 1, Categories: 2, Mask: 4})
	ReleaseBody(a)

	b := AcquireBody(dynCircleDef(), 0, 0, 0)
	if b.Body.UserData != nil {
		t.Errorf("body UserData leaked across pool: %v", b.Body.UserData)
	}
	if b.Shape.UserData != nil {
		t.Errorf("shape UserData leaked across pool: %v", b.Shape.UserData)
	}
	if b.Shape.Sensor() {
		t.Errorf("sensor flag leaked across pool")
	}
	if b.Shape.CollisionType() != 0 {
		t.Errorf("collision type leaked: %v", b.Shape.CollisionType())
	}
	if b.Shape.Filter != cp.SHAPE_FILTER_ALL {
		t.Errorf("shape filter leaked: %+v", b.Shape.Filter)
	}
}

func TestPool_DisableSwitch(t *testing.T) {
	setupPool(t)
	SetPoolEnabled(false)
	defer SetPoolEnabled(true)

	a := AcquireBody(dynCircleDef(), 0, 0, 0)
	cpBody := a.Body
	ReleaseBody(a)
	if bodies, _ := PoolStats(); bodies != 0 {
		t.Errorf("disabled pool should not hold anything; got %d", bodies)
	}

	b := AcquireBody(dynCircleDef(), 0, 0, 0)
	if b.Body == cpBody {
		t.Errorf("disabled pool should allocate fresh bodies")
	}

	SetPoolEnabled(true)
	ReleaseBody(b)
	if bodies, _ := PoolStats(); bodies != 1 {
		t.Errorf("re-enabled pool should reclaim; got %d", bodies)
	}
}

func TestPool_NoArbiterLeak(t *testing.T) {
	p := setupPool(t)
	floor := AcquireBody(
		Static{Shape: Segment{A: cp.Vector{X: -1000, Y: 100}, B: cp.Vector{X: 1000, Y: 100}, Radius: 1}},
		0, 0, 0,
	)
	p.AddBody(floor)

	const cycles = 200
	for i := 0; i < cycles; i++ {
		b := AcquireBody(dynCircleDef(), 0, 90, 0)
		p.AddBody(b)
		p.Step(1.0 / 60.0)
		p.Step(1.0 / 60.0)
		p.RemoveBody(b)
		ReleaseBody(b)
	}

	// One last step lets the space filter any stragglers.
	p.Step(1.0 / 60.0)

	bodies, _ := PoolStats()
	if bodies < 1 {
		t.Errorf("pool should hold at least one recycled dynamic body; got %d", bodies)
	}
}

func BenchmarkSpawnDespawn_Pooled(b *testing.B) {
	PoolReset()
	p := NewPhysicsParent(Config{})
	def := dynCircleDef()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := AcquireBody(def, 0, 0, 0)
		p.AddBody(body)
		p.RemoveBody(body)
		ReleaseBody(body)
	}
}

func BenchmarkSpawnDespawn_NoPool(b *testing.B) {
	PoolReset()
	SetPoolEnabled(false)
	defer SetPoolEnabled(true)
	p := NewPhysicsParent(Config{})
	def := dynCircleDef()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := AcquireBody(def, 0, 0, 0)
		p.AddBody(body)
		p.RemoveBody(body)
		ReleaseBody(body)
	}
}
