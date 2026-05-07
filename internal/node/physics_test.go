//go:build !nophysics

package node

import (
	"testing"

	"github.com/devthicket/willow/internal/physics"
	"github.com/devthicket/willow/internal/types"
	"github.com/jakecoffman/cp/v2"
)

func countBodies(p *physics.PhysicsParent) int {
	n := 0
	p.Space.EachBody(func(*cp.Body) { n++ })
	return n
}

func mustPanic(t *testing.T, label string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("%s: expected panic, got none", label)
		}
	}()
	fn()
}

func TestEnablePhysics_NestedAncestor_Panics(t *testing.T) {
	root, by := buildTree("root", "child")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	mustPanic(t, "nested under existing root", func() {
		by["child"].EnablePhysics(physics.Config{})
	})
}

func TestEnablePhysics_NestedDescendant_Panics(t *testing.T) {
	root, by := buildTree("root", "child")
	by["child"].EnablePhysics(physics.Config{})
	defer by["child"].DisablePhysics()
	mustPanic(t, "ancestor over existing root", func() {
		root.EnablePhysics(physics.Config{})
	})
}

func TestSetBody_NoAncestor_Panics(t *testing.T) {
	n := NewNode("orphan", types.NodeTypeSprite)
	mustPanic(t, "SetBody without root", func() {
		attachBody(n, nil)
	})
}

func TestSetBody_AddsToSpace(t *testing.T) {
	root, by := buildTree("root", "a")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], nil)
	if got := countBodies(root.PhysicsRoot.Parent); got != 1 {
		t.Fatalf("space body count = %d, want 1", got)
	}
	if by["a"].Body == nil {
		t.Fatal("Body field should be set after SetBody")
	}
}

func TestSetBody_Replaces_DetachesPrevious(t *testing.T) {
	// Calling SetBody on a node that already has a body must detach and
	// release the old one — otherwise the previous body stays orphaned in
	// the cp space and continues to be stepped without ever being synced
	// back to a node.
	root, by := buildTree("root", "a")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], physics.Dynamic{Shape: physics.Circle{Radius: 1}, Mass: 1})
	attachBody(by["a"], physics.Dynamic{Shape: physics.Circle{Radius: 2}, Mass: 1})
	if got := countBodies(root.PhysicsRoot.Parent); got != 1 {
		t.Fatalf("post-replace body count = %d, want 1 (old body should be removed)", got)
	}
	if by["a"].Body == nil {
		t.Fatal("Body field should be populated after replace")
	}
}

func TestRemoveBody_RemovesFromSpace(t *testing.T) {
	root, by := buildTree("root", "a")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], nil)
	by["a"].RemoveBody()
	if got := countBodies(root.PhysicsRoot.Parent); got != 0 {
		t.Fatalf("space body count = %d, want 0", got)
	}
	if by["a"].Body != nil {
		t.Fatal("Body field should be nil after RemoveBody")
	}
}

func TestRemoveBody_NilBody_Noop(t *testing.T) {
	root, by := buildTree("root", "a")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	by["a"].RemoveBody() // no panic, no allocation
}

func TestDisablePhysics_TearsDownSubtree(t *testing.T) {
	root, by := buildTree("root", "a", "b")
	root.EnablePhysics(physics.Config{})
	attachBody(by["a"], nil)
	attachBody(by["b"], nil)
	parent := root.PhysicsRoot.Parent
	root.DisablePhysics()
	if got := countBodies(parent); got != 0 {
		t.Fatalf("post-DisablePhysics body count = %d, want 0", got)
	}
	if by["a"].Body != nil || by["b"].Body != nil {
		t.Fatal("Body fields should be cleared after DisablePhysics")
	}
	if root.PhysicsRoot != nil {
		t.Fatal("PhysicsRoot should be nil after DisablePhysics")
	}
}

func TestAddChild_FlipsDirtyWhenInPhysicsTree(t *testing.T) {
	root, _ := buildTree("root")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	root.PhysicsRoot.ListDirty = false

	c := NewNode("c", types.NodeTypeContainer)
	root.AddChild(c)
	if !root.PhysicsRoot.ListDirty {
		t.Fatal("AddChild into physics subtree should set ListDirty")
	}
}

func TestAddChild_NoOpWhenNoPhysics(t *testing.T) {
	// physicsRootsActive == 0; AddChild must not touch any physics state.
	parent := NewNode("p", types.NodeTypeContainer)
	child := NewNode("c", types.NodeTypeSprite)
	parent.AddChild(child)
	if parent.PhysicsRoot != nil || child.PhysicsRoot != nil {
		t.Fatal("no PhysicsRoot should be allocated")
	}
}

func TestDispose_BodiedNode_RemovesFromSpace(t *testing.T) {
	root, by := buildTree("root", "a")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], nil)
	parent := root.PhysicsRoot.Parent
	by["a"].Dispose()
	if got := countBodies(parent); got != 0 {
		t.Fatalf("post-Dispose body count = %d, want 0", got)
	}
}

func TestDispose_SubtreeWithMultipleBodies(t *testing.T) {
	root, by := buildTree("root", "a", "b", "c")
	root.EnablePhysics(physics.Config{})
	attachBody(by["a"], nil)
	attachBody(by["b"], nil)
	attachBody(by["c"], nil)
	parent := root.PhysicsRoot.Parent
	if got := countBodies(parent); got != 3 {
		t.Fatalf("setup body count = %d, want 3", got)
	}
	root.Dispose()
	if got := countBodies(parent); got != 0 {
		t.Fatalf("post-Dispose body count = %d, want 0", got)
	}
}

func TestAutoPivot_OnlyWhenDefault(t *testing.T) {
	// White-pixel sprites of known size so SetPivotPercent has a basis.
	root := NewNode("root", types.NodeTypeContainer)
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()

	mkSprite := func(name string, w, h float64) *Node {
		n := NewNode(name, types.NodeTypeSprite)
		n.TextureRegion_.OriginalW = uint16(w)
		n.TextureRegion_.OriginalH = uint16(h)
		root.AddChild(n)
		return n
	}

	// Default pivot — auto-centers to (w/2, h/2).
	a := mkSprite("a", 10, 20)
	attachBody(a, nil)
	if a.PivotX_ != 5 || a.PivotY_ != 10 {
		t.Errorf("default pivot should auto-center to (5, 10), got (%v, %v)", a.PivotX_, a.PivotY_)
	}

	// Explicit non-default pivot — preserved.
	b := mkSprite("b", 10, 20)
	b.SetPivot(2, 7)
	attachBody(b, nil)
	if b.PivotX_ != 2 || b.PivotY_ != 7 {
		t.Errorf("explicit pivot should be preserved, got (%v, %v)", b.PivotX_, b.PivotY_)
	}
}

func BenchmarkAddChild_NoPhysics(b *testing.B) {
	parent := NewNode("p", types.NodeTypeContainer)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := NewNode("c", types.NodeTypeContainer)
		parent.AddChild(c)
	}
}

func BenchmarkAddChild_WithPhysics(b *testing.B) {
	root := NewNode("root", types.NodeTypeContainer)
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := NewNode("c", types.NodeTypeContainer)
		root.AddChild(c)
	}
}
