//go:build nophysics

package node

import "github.com/devthicket/willow/internal/physics"

const physicsUnavailable = "willow: built with -tags nophysics — physics is unavailable"

// physicsRoot is empty under nophysics so the Node.PhysicsRoot field still
// has a defined pointer target without dragging cp into the build.
type physicsRoot struct{}

// physicsRootsActive is referenced by tree-mutation paths (children.go) and
// Dispose (methods.go). Under nophysics nothing ever increments it, so the
// short-circuit `atomic.LoadInt32(&physicsRootsActive) == 0` keeps physics
// bookkeeping out of the hot path entirely.
var physicsRootsActive int32

// markPhysicsListDirty matches the real signature for tree-mutation call
// sites (AddChild / RemoveFromParent / etc.). No-op under nophysics.
func markPhysicsListDirty(n *Node) {}

// disposePhysicsSubtree is called from Dispose under
// `physicsRootsActive > 0` — that branch never fires here, but the symbol
// must exist for the file to link.
func (n *Node) disposePhysicsSubtree() {}

// EnablePhysics, DisablePhysics, SetBody, StepPhysics panic when invoked so
// a missing physics build is a loud, immediately-traceable failure rather
// than silently doing nothing (which would turn into a debugging nightmare:
// "why isn't anything falling?").
func (n *Node) EnablePhysics(cfg physics.Config) { panic(physicsUnavailable) }
func (n *Node) SetBody(def physics.BodyDef)      { panic(physicsUnavailable) }
func (n *Node) StepPhysics(dt float64)            { panic(physicsUnavailable) }

// DisablePhysics, RemoveBody, GetBody, TickPhysicsTree are exempted from
// panicking: they sit on cleanup and per-frame paths where panicking would
// poison the rest of the engine. A program that never enabled physics in
// the first place trivially has nothing to disable, remove, or tick.
func (n *Node) DisablePhysics()                  {}
func (n *Node) RemoveBody()                      {}
func (n *Node) GetBody() *physics.Body           { return nil }
func (n *Node) TickPhysicsTree(dt float64)        {}
func (n *Node) SetBodyEnabled(enabled bool)      {}
func (n *Node) BodyEnabled() bool                { return false }
