package physics

import "github.com/jakecoffman/cp/v2"

// Body bundles a cp body with its primary shape. The embedded *cp.Body
// exposes the full cp API (ApplyImpulseAtLocalPoint, SetVelocityVector,
// Position, Angle, ...) so callers don't need a wrapper for every verb.
//
// v1 supports a single shape per body; multi-shape compounds are deferred.
//
// No back-pointer to *Node lives here on purpose — the orchestration layer
// (internal/node) holds the Node↔Body mapping and writes back to Node
// fields directly. Keeping physics leaf preserves the dependency direction
// established by particle/ and text/.
type Body struct {
	*cp.Body
	Shape *cp.Shape
}

// NewBody builds a body+shape from a BodyDef.
//
// The result is unattached: the caller adds it to a *cp.Space (typically
// via PhysicsParent.AddBody) before stepping.
func NewBody(def BodyDef) *Body {
	body, shape, _ := def.buildBody()
	return &Body{Body: body, Shape: shape}
}
