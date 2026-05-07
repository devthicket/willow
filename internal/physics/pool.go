package physics

import (
	"sync/atomic"

	"github.com/jakecoffman/cp/v2"
)

// poolKey partitions the body+shape pool by (body kind, shape kind). cp v2
// does not safely allow runtime body-type transitions and does not expose a
// shape→body reattach API, so pooled pairs only flow back into requests
// that match both kinds exactly.
type poolKey struct {
	bk bodyKind
	sk shapeKind
}

// pools holds released *Body wrappers (each carrying its cp body and shape)
// keyed by kind pair. Willow runs single-threaded today; if Ebitengine ever
// goes multi-threaded, swap these slices for sync.Pool.
var pools = make(map[poolKey][]*Body)

var wrappersFreed []*Body

var poolEnabled atomic.Bool

func init() { poolEnabled.Store(true) }

// SetPoolEnabled toggles pool reuse. When disabled, AcquireBody always
// allocates fresh and ReleaseBody discards. Test escape hatch — production
// code should leave this on.
func SetPoolEnabled(enabled bool) { poolEnabled.Store(enabled) }

// PoolEnabled reports the current toggle state.
func PoolEnabled() bool { return poolEnabled.Load() }

// PoolStats reports the current free-list sizes. Both counts are equal in
// the current single-shape-per-body design but kept separate for forward
// compatibility with multi-shape compounds.
func PoolStats() (bodies, shapes int) {
	n := 0
	for _, items := range pools {
		n += len(items)
	}
	return n, n
}

// PoolReset clears every free list. Tests use this to start from a known
// state; production code never needs it.
func PoolReset() {
	for k := range pools {
		delete(pools, k)
	}
	wrappersFreed = wrappersFreed[:0]
}

// AcquireBody returns a *Body configured per def, positioned at (x, y) with
// the supplied angle. Reuses a pooled body+shape pair when available;
// allocates fresh otherwise. The returned body is unattached — the caller
// adds it to a *cp.Space (typically via PhysicsParent.AddBody).
func AcquireBody(def BodyDef, x, y, angle float64) *Body {
	bk, sk := def.kinds()
	if poolEnabled.Load() {
		key := poolKey{bk, sk}
		if items := pools[key]; len(items) > 0 {
			n := len(items)
			b := items[n-1]
			items[n-1] = nil
			pools[key] = items[:n-1]
			def.apply(b, x, y, angle)
			return b
		}
	}
	body, shape, _ := def.buildBody()
	body.SetPosition(cp.Vector{X: x, Y: y})
	body.SetAngle(angle)
	w := acquireWrapper()
	w.Body = body
	w.Shape = shape
	return w
}

// ReleaseBody returns a *Body to the free list. The caller MUST have
// already detached it from its space (PhysicsParent.RemoveBody) — pooling a
// still-attached body would corrupt cp's internal lists.
//
// Velocities, forces, and torque are zeroed here so a recycled body cannot
// leak motion state from its previous life. Mass/moment/friction/elasticity
// and shape geometry are reset on Acquire from the new def.
func ReleaseBody(b *Body) {
	if b == nil || b.Body == nil {
		return
	}
	b.Body.SetVelocityVector(cp.Vector{})
	b.Body.SetAngularVelocity(0)
	b.Body.SetForce(cp.Vector{})
	b.Body.SetTorque(0)
	b.Body.UserData = nil
	if b.Shape != nil {
		b.Shape.UserData = nil
		b.Shape.SetSensor(false)
		b.Shape.SetCollisionType(0)
		b.Shape.SetFilter(cp.SHAPE_FILTER_ALL)
	}

	if !poolEnabled.Load() {
		// Drop body and shape; release the wrapper for reuse separately so
		// disabling the pool still avoids wrapper churn.
		b.Body = nil
		b.Shape = nil
		releaseWrapper(b)
		return
	}

	bk, ok := bodyKindFromCp(b.Body)
	if !ok {
		// Unknown body type — refuse to pool rather than risk corruption.
		b.Body = nil
		b.Shape = nil
		releaseWrapper(b)
		return
	}
	sk, ok := shapeKindFromCp(b.Shape)
	if !ok {
		b.Body = nil
		b.Shape = nil
		releaseWrapper(b)
		return
	}
	key := poolKey{bk, sk}
	pools[key] = append(pools[key], b)
}

func acquireWrapper() *Body {
	n := len(wrappersFreed)
	if n == 0 {
		return &Body{}
	}
	w := wrappersFreed[n-1]
	wrappersFreed[n-1] = nil
	wrappersFreed = wrappersFreed[:n-1]
	return w
}

func releaseWrapper(w *Body) {
	wrappersFreed = append(wrappersFreed, w)
}

func bodyKindFromCp(b *cp.Body) (bodyKind, bool) {
	switch b.GetType() {
	case cp.BODY_DYNAMIC:
		return kindDynamic, true
	case cp.BODY_STATIC:
		return kindStatic, true
	case cp.BODY_KINEMATIC:
		return kindKinematic, true
	}
	return 0, false
}

func shapeKindFromCp(s *cp.Shape) (shapeKind, bool) {
	if s == nil {
		return 0, false
	}
	switch s.Class.(type) {
	case *cp.Circle:
		return skCircle, true
	case *cp.Segment:
		return skSegment, true
	case *cp.PolyShape:
		return skPoly, true
	}
	return 0, false
}
