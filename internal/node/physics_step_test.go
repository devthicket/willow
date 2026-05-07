package node

import (
	"math"
	"testing"

	"github.com/devthicket/willow/internal/physics"
	"github.com/devthicket/willow/internal/types"
	"github.com/jakecoffman/cp/v2"
)

const stepDT = 1.0 / 60.0

func TestStep_GravityFall(t *testing.T) {
	root, by := buildTree("root", "ball")
	root.EnablePhysics(physics.Config{Gravity: cp.Vector{X: 0, Y: 900}})
	defer root.DisablePhysics()
	attachBody(by["ball"], nil)

	const steps = 30
	for i := 0; i < steps; i++ {
		root.stepPhysicsRoot(stepDT)
	}

	// cp uses semi-implicit Euler with velocity update first; after N steps
	// position has accumulated v_1..v_N where v_i = (i-1)*g*dt + g*dt, but
	// the position increment uses pre-step velocity, giving N(N-1)/2.
	want := 900.0 * stepDT * stepDT * float64(steps*(steps-1)) / 2.0
	got := by["ball"].Y_
	if math.Abs(got-want) > 1.0 {
		t.Fatalf("Y after %d steps = %v, want ~%v", steps, got, want)
	}
}

func TestStep_FastPath_Detected(t *testing.T) {
	root, by := buildTree("root", "ball")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["ball"], nil)

	UpdateWorldTransform(root, IdentityTransform, 1.0, false, false)
	root.stepPhysicsRoot(stepDT)
	if got := root.PhysicsRoot.FastPaths; len(got) != 1 || !got[0] {
		t.Fatalf("expected fastPath true for direct child of identity-transform root, got %v", got)
	}
}

func TestStep_FastPath_NotDetected_NestedParent(t *testing.T) {
	root := NewNode("root", types.NodeTypeContainer)
	mid := NewNode("mid", types.NodeTypeContainer)
	leaf := NewNode("leaf", types.NodeTypeContainer)
	root.AddChild(mid)
	mid.AddChild(leaf)
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(leaf, nil)

	UpdateWorldTransform(root, IdentityTransform, 1.0, false, false)
	root.stepPhysicsRoot(stepDT)
	if got := root.PhysicsRoot.FastPaths; len(got) != 1 || got[0] {
		t.Fatalf("expected fastPath false for nested bodied node, got %v", got)
	}
}

func TestStep_GeneralPath_TransformedRoot(t *testing.T) {
	root, by := buildTree("root", "ball")
	root.SetPosition(100, 50)
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["ball"], nil)

	UpdateWorldTransform(root, IdentityTransform, 1.0, false, false)
	by["ball"].Body.SetPosition(cp.Vector{X: 250, Y: 75})
	root.stepPhysicsRoot(0)

	// Local should be world - parent translation.
	if math.Abs(by["ball"].X_-150) > 1e-6 || math.Abs(by["ball"].Y_-25) > 1e-6 {
		t.Fatalf("local pos = (%v, %v), want (150, 25)", by["ball"].X_, by["ball"].Y_)
	}
}

func TestStep_RebuildReconciles_Reparent(t *testing.T) {
	root, by := buildTree("root", "ball")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["ball"], nil)

	root.stepPhysicsRoot(stepDT) // populate BodiedNodes
	if got := countBodies(root.PhysicsRoot.Parent); got != 1 {
		t.Fatalf("setup body count = %d, want 1", got)
	}

	// Re-parent ball out of the physics tree. AddChild dirties the source
	// root before detach, so reconciliation runs on the next tick.
	outside := NewNode("outside", types.NodeTypeContainer)
	outside.AddChild(by["ball"])

	root.stepPhysicsRoot(stepDT)
	if got := countBodies(root.PhysicsRoot.Parent); got != 0 {
		t.Fatalf("post-reparent body count = %d, want 0", got)
	}
	if by["ball"].Body != nil {
		t.Fatal("re-parented node should have Body cleared")
	}
}

func TestStep_RebuildReconciles_ReparentMiddleOfList(t *testing.T) {
	// Regression: when a body in the middle of the list is re-parented out,
	// the rebuild must still detect the escapee. A naive in-place rebuild
	// (newList sharing storage with old) overwrites the escapee's slot
	// before the reconciliation scan runs, leaking its body in the space.
	root, by := buildTree("root", "a", "b", "c")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], nil)
	attachBody(by["b"], nil)
	attachBody(by["c"], nil)
	root.stepPhysicsRoot(stepDT)

	outside := NewNode("outside", types.NodeTypeContainer)
	outside.AddChild(by["b"])

	root.stepPhysicsRoot(stepDT)
	if got := countBodies(root.PhysicsRoot.Parent); got != 2 {
		t.Fatalf("post-reparent body count = %d, want 2", got)
	}
	if by["b"].Body != nil {
		t.Fatal("escapee should have Body cleared")
	}
}

func TestStep_RebuildReconciles_Dispose(t *testing.T) {
	root, by := buildTree("root", "a", "b")
	root.EnablePhysics(physics.Config{})
	defer root.DisablePhysics()
	attachBody(by["a"], nil)
	attachBody(by["b"], nil)
	root.stepPhysicsRoot(stepDT)

	by["a"].Dispose()
	root.stepPhysicsRoot(stepDT) // must not crash; sync list shrinks

	if got := len(root.PhysicsRoot.BodiedNodes); got != 1 {
		t.Fatalf("BodiedNodes len = %d, want 1", got)
	}
	if got := countBodies(root.PhysicsRoot.Parent); got != 1 {
		t.Fatalf("space body count = %d, want 1", got)
	}
}

func TestStep_ManualSuppressesAuto(t *testing.T) {
	root, by := buildTree("root", "ball")
	root.EnablePhysics(physics.Config{Gravity: cp.Vector{X: 0, Y: 900}})
	defer root.DisablePhysics()
	attachBody(by["ball"], nil)

	by["ball"].StepPhysics(stepDT)
	yAfterManual := by["ball"].Y_

	// TickPhysicsTree should now be a no-op for this root because
	// SteppedThisFrame was set.
	root.TickPhysicsTree(stepDT)
	if by["ball"].Y_ != yAfterManual {
		t.Fatalf("auto-tick should have been suppressed; Y went %v -> %v",
			yAfterManual, by["ball"].Y_)
	}

	// Next call to TickPhysicsTree (next "frame") should step normally.
	root.TickPhysicsTree(stepDT)
	if by["ball"].Y_ == yAfterManual {
		t.Fatal("auto-tick on the next frame should have advanced Y")
	}
}

func TestStep_NoAncestorWalkPerBody(t *testing.T) {
	// Count InvalidateAncestorCacheFn calls during sync write-back; with the
	// fast-path direct-field write, this should be exactly zero.
	prev := InvalidateAncestorCacheFn
	defer func() { InvalidateAncestorCacheFn = prev }()

	root := NewNode("root", types.NodeTypeContainer)
	root.EnablePhysics(physics.Config{Gravity: cp.Vector{X: 0, Y: 900}})
	defer root.DisablePhysics()

	const N = 100
	for i := 0; i < N; i++ {
		c := NewNode("b", types.NodeTypeSprite)
		root.AddChild(c)
		attachBody(c, nil)
	}
	root.stepPhysicsRoot(stepDT) // initial rebuild + step (warm)

	calls := 0
	InvalidateAncestorCacheFn = func(*Node) { calls++ }
	root.stepPhysicsRoot(stepDT)
	if calls != 0 {
		t.Fatalf("write-back triggered %d ancestor invalidations, want 0", calls)
	}
}

func BenchmarkStep_100Bodies_FastPath(b *testing.B) {
	root := NewNode("root", types.NodeTypeContainer)
	root.EnablePhysics(physics.Config{Gravity: cp.Vector{X: 0, Y: 900}})
	defer root.DisablePhysics()
	for i := 0; i < 100; i++ {
		c := NewNode("b", types.NodeTypeSprite)
		root.AddChild(c)
		attachBody(c, nil)
	}
	root.stepPhysicsRoot(stepDT)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.stepPhysicsRoot(stepDT)
	}
}
