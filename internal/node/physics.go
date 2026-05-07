package node

import (
	"sync/atomic"

	"github.com/devthicket/willow/internal/physics"
	"github.com/jakecoffman/cp/v2"
)

// physicsRoot bundles the cp-side PhysicsParent with node-side bookkeeping
// for a physics subtree. It lives on the Node that owns the subtree
// (the "physics root"); descendants discover it by walking up via
// findPhysicsRoot.
type physicsRoot struct {
	Parent           *physics.PhysicsParent
	BodiedNodes      []*Node
	ListDirty        bool
	SteppedThisFrame bool
}

// physicsRootsActive counts active physics subtrees across the process.
// Tree-mutation paths short-circuit on a zero load: when no scene uses
// physics, the only cost on AddChild/RemoveFromParent is one atomic load.
var physicsRootsActive int32

// EnablePhysics turns n into a physics root, creating an isolated cp space
// configured by cfg. Panics if n already has a physics ancestor or any
// physics descendant.
func (n *Node) EnablePhysics(cfg physics.Config) {
	if findPhysicsAncestor(n) != nil {
		panic("willow: EnablePhysics called on a node that already has a physics ancestor")
	}
	if hasPhysicsDescendant(n) {
		panic("willow: EnablePhysics nested under another physics root")
	}
	n.PhysicsRoot = &physicsRoot{
		Parent: physics.NewPhysicsParent(cfg),
	}
	atomic.AddInt32(&physicsRootsActive, 1)
}

// DisablePhysics tears down the physics subtree rooted at n: every bodied
// descendant has its Body removed from the space and cleared, then the
// root itself is dismantled.
func (n *Node) DisablePhysics() {
	if n.PhysicsRoot == nil {
		return
	}
	walkSubtree(n, func(c *Node) {
		if c.Body != nil {
			n.PhysicsRoot.Parent.RemoveBody(c.Body)
			c.Body = nil
		}
	})
	n.PhysicsRoot = nil
	atomic.AddInt32(&physicsRootsActive, -1)
}

// SetBody attaches a physics body+shape (built from def) to n and registers
// it with the enclosing physics space. Panics if n has no physics ancestor.
func (n *Node) SetBody(def physics.BodyDef) {
	root := n.findPhysicsRoot()
	if root == nil {
		panic("willow: SetBody called on a node with no physics ancestor")
	}
	body := physics.NewBody(def)
	body.SetPosition(cp.Vector{X: n.X_, Y: n.Y_})
	body.SetAngle(n.Rotation_)
	root.Parent.AddBody(body)
	n.Body = body
	root.ListDirty = true

	// Auto-center pivot iff still default — physics expects rotation/scale
	// about the body centroid.
	if n.PivotX_ == 0 && n.PivotY_ == 0 {
		n.SetPivotPercent(0.5, 0.5)
	}
}

// RemoveBody detaches n's body from the physics space and clears the field.
// No-op if n has no body.
func (n *Node) RemoveBody() {
	if n.Body == nil {
		return
	}
	if root := n.findPhysicsRoot(); root != nil {
		root.Parent.RemoveBody(n.Body)
		root.ListDirty = true
	}
	n.Body = nil
}

// GetBody returns n.Body. Provided for surface symmetry with EnablePhysics
// and SetBody; n.Body is also directly accessible.
func (n *Node) GetBody() *physics.Body { return n.Body }

// findPhysicsRoot walks up from n (inclusive) and returns the nearest
// physicsRoot, or nil.
func (n *Node) findPhysicsRoot() *physicsRoot {
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.PhysicsRoot != nil {
			return cur.PhysicsRoot
		}
	}
	return nil
}

// findPhysicsAncestor is like findPhysicsRoot but excludes n itself; used
// by EnablePhysics to detect nesting under an existing root.
func findPhysicsAncestor(n *Node) *physicsRoot {
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		if cur.PhysicsRoot != nil {
			return cur.PhysicsRoot
		}
	}
	return nil
}

// hasPhysicsDescendant reports whether any node in the subtree below n
// (excluding n) is a physics root.
func hasPhysicsDescendant(n *Node) bool {
	for _, c := range n.Children_ {
		if c.PhysicsRoot != nil {
			return true
		}
		if hasPhysicsDescendant(c) {
			return true
		}
	}
	return false
}

// disposePhysicsSubtree releases bodies for every node in the subtree
// rooted at n and tears down the physics root if n owns one. Called from
// Dispose when physicsRootsActive > 0.
func (n *Node) disposePhysicsSubtree() {
	walkSubtree(n, func(c *Node) {
		if c.Body != nil {
			c.RemoveBody()
		}
	})
	if n.PhysicsRoot != nil {
		n.DisablePhysics()
	}
}

// markPhysicsListDirty flags the physicsRoot enclosing n (if any) so the
// per-frame tick rebuilds its bodied-node list. Cheap when no physics is
// active: a single atomic load.
func markPhysicsListDirty(n *Node) {
	if atomic.LoadInt32(&physicsRootsActive) == 0 {
		return
	}
	if root := n.findPhysicsRoot(); root != nil {
		root.ListDirty = true
	}
}

// walkSubtree visits n and every descendant (depth-first, current children).
func walkSubtree(n *Node, fn func(*Node)) {
	fn(n)
	for _, c := range n.Children_ {
		walkSubtree(c, fn)
	}
}
