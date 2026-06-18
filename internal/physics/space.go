//go:build physics

package physics

import "github.com/jakecoffman/cp/v2"

// PhysicsParent owns a cp.Space and the Config it was built with. The node
// layer attaches one to whichever Node is the physics root for a subtree;
// physics/ itself has no concept of nodes or scenes.
type PhysicsParent struct {
	Space  *cp.Space
	Config Config
}

// NewPhysicsParent builds a Space with the supplied configuration applied.
// Zero-valued Iterations and SleepTime fall through to cp's defaults
// (Iterations=10, never sleep).
func NewPhysicsParent(cfg Config) *PhysicsParent {
	s := cp.NewSpace()
	s.SetGravity(cfg.Gravity)
	if cfg.Iterations > 0 {
		s.Iterations = uint(cfg.Iterations)
	}
	if cfg.SleepTime > 0 {
		s.SleepTimeThreshold = cfg.SleepTime
	}
	return &PhysicsParent{Space: s, Config: cfg}
}

// Step advances the simulation by dt seconds. The node layer is
// responsible for any dirty-list rebuild and for syncing body state back
// to Node transforms after Step returns.
func (p *PhysicsParent) Step(dt float64) { p.Space.Step(dt) }

// AddBody registers a body and its primary shape with the underlying space
// and marks the wrapper Enabled.
func (p *PhysicsParent) AddBody(b *Body) {
	p.Space.AddBody(b.Body)
	if b.Shape != nil {
		p.Space.AddShape(b.Shape)
	}
	b.Enabled = true
}

// RemoveBody unregisters a body and its shape from the underlying space and
// clears the Enabled flag. No-op when the body is already detached, so
// callers may invoke this idempotently (e.g. during teardown of subtrees
// containing a mix of enabled and disabled bodies).
func (p *PhysicsParent) RemoveBody(b *Body) {
	if !b.Enabled {
		return
	}
	if b.Shape != nil {
		p.Space.RemoveShape(b.Shape)
	}
	p.Space.RemoveBody(b.Body)
	b.Enabled = false
}
