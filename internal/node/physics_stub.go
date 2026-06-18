//go:build !physics

package node

import "github.com/devthicket/willow/internal/physics"

const physicsUnavailable = "willow: physics not compiled in — rebuild with -tags physics"

// physicsRoot is empty in the default (no-physics) build so the
// Node.PhysicsRoot field still has a defined pointer target without dragging
// cp into the build.
type physicsRoot struct{}

// physicsRootsActive is referenced by tree-mutation paths (children.go) and
// Dispose (methods.go). In the default build nothing ever increments it, so
// the short-circuit `atomic.LoadInt32(&physicsRootsActive) == 0` keeps
// physics bookkeeping out of the hot path entirely.
var physicsRootsActive int32

// markPhysicsListDirty matches the real signature for tree-mutation call
// sites (AddChild / RemoveFromParent / etc.). No-op in the default build.
func markPhysicsListDirty(n *Node) {}

// disposePhysicsSubtree is called from Dispose under
// `physicsRootsActive > 0` — that branch never fires here, but the symbol
// must exist for the file to link.
func (n *Node) disposePhysicsSubtree() {}

// EnablePhysics, SetBody, SetBodyEnabled, StepPhysics are the user-facing
// entry points for opting into physics. In a build without -tags physics
// they panic, so a missing physics build is a loud, immediately-traceable
// failure rather than silently doing nothing (which would turn into a
// debugging nightmare: "why isn't anything falling?").
func (n *Node) EnablePhysics(cfg physics.Config) { panic(physicsUnavailable) }
func (n *Node) SetBody(def physics.BodyDef)       { panic(physicsUnavailable) }
func (n *Node) SetBodyEnabled(enabled bool)       { panic(physicsUnavailable) }
func (n *Node) StepPhysics(dt float64)            { panic(physicsUnavailable) }

// DisablePhysics, RemoveBody, GetBody, BodyEnabled, TickPhysicsTree are
// exempted from panicking: they sit on cleanup, query, and per-frame paths
// where panicking would poison the rest of the engine. TickPhysicsTree in
// particular is invoked unconditionally from Scene.Update every frame, so it
// must be a safe no-op. A program that never enabled physics in the first
// place trivially has nothing to disable, remove, query, or tick.
func (n *Node) DisablePhysics()        {}
func (n *Node) RemoveBody()            {}
func (n *Node) GetBody() *physics.Body { return nil }
func (n *Node) BodyEnabled() bool      { return false }
func (n *Node) TickPhysicsTree(dt float64) {}
