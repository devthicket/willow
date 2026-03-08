package willow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Constructor defaults ---

func TestNewContainerDefaults(t *testing.T) {
	n := NewContainer("test")
	assertNodeDefaults(t, n, "test", NodeTypeContainer)
}

func TestNewSpriteDefaults(t *testing.T) {
	region := TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32}
	n := NewSprite("spr", region)
	assertNodeDefaults(t, n, "spr", NodeTypeSprite)
	if n.TextureRegion_ != region {
		t.Errorf("TextureRegion = %v, want %v", n.TextureRegion_, region)
	}
}

func TestNewMeshDefaults(t *testing.T) {
	verts := []ebiten.Vertex{{DstX: 0, DstY: 0}}
	inds := []uint16{0}
	n := NewMesh("mesh", nil, verts, inds)
	assertNodeDefaults(t, n, "mesh", NodeTypeMesh)
	if len(n.Mesh.Vertices) != 1 || len(n.Mesh.Indices) != 1 {
		t.Errorf("Vertices/Indices not set")
	}
}

func TestNewParticleEmitterDefaults(t *testing.T) {
	n := NewParticleEmitter("emitter", EmitterConfig{})
	assertNodeDefaults(t, n, "emitter", NodeTypeParticleEmitter)
}

func TestNewTextDefaults(t *testing.T) {
	n := NewText("text", "hello", nil)
	assertNodeDefaults(t, n, "text", NodeTypeText)
}

func assertNodeDefaults(t *testing.T, n *Node, name string, typ NodeType) {
	t.Helper()
	if n.ID == 0 {
		t.Error("ID should be non-zero")
	}
	if n.Name != name {
		t.Errorf("Name = %q, want %q", n.Name, name)
	}
	if n.Type != typ {
		t.Errorf("Type = %d, want %d", n.Type, typ)
	}
	if n.ScaleX_ != 1 || n.ScaleY_ != 1 {
		t.Errorf("Scale = (%v, %v), want (1, 1)", n.ScaleX_, n.ScaleY_)
	}
	if n.Alpha_ != 1 {
		t.Errorf("Alpha = %v, want 1", n.Alpha_)
	}
	if n.Color_ != RGBA(1, 1, 1, 1) {
		t.Errorf("Color = %v, want white", n.Color_)
	}
	if !n.Visible_ {
		t.Error("Visible should be true")
	}
	if !n.Renderable_ {
		t.Error("Renderable should be true")
	}
	if !n.TransformDirty {
		t.Error("transformDirty should be true")
	}
}

// --- Unique IDs ---

func TestUniqueIDs(t *testing.T) {
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewSprite("c", TextureRegion{})
	if a.ID == b.ID || b.ID == c.ID || a.ID == c.ID {
		t.Errorf("IDs should be unique: %d, %d, %d", a.ID, b.ID, c.ID)
	}
}

// --- AddChild ---

func TestAddChildBasic(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	if child.Parent != parent {
		t.Error("child.Parent should be parent")
	}
	if parent.NumChildren() != 1 {
		t.Errorf("NumChildren = %d, want 1", parent.NumChildren())
	}
	if parent.ChildAt(0) != child {
		t.Error("ChildAt(0) should be child")
	}
}

func TestAddChildReparent(t *testing.T) {
	p1 := NewContainer("p1")
	p2 := NewContainer("p2")
	child := NewContainer("child")

	p1.AddChild(child)
	if p1.NumChildren() != 1 {
		t.Fatal("p1 should have 1 child")
	}

	p2.AddChild(child)
	if p1.NumChildren() != 0 {
		t.Error("p1 should have 0 children after reparent")
	}
	if p2.NumChildren() != 1 {
		t.Error("p2 should have 1 child")
	}
	if child.Parent != p2 {
		t.Error("child.Parent should be p2")
	}
}

func TestAddChildCyclePanic(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	grandchild := NewContainer("grandchild")
	parent.AddChild(child)
	child.AddChild(grandchild)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for cycle, got none")
		}
	}()
	grandchild.AddChild(parent) // should panic
}

func TestAddChildSelfPanic(t *testing.T) {
	n := NewContainer("self")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for self-add, got none")
		}
	}()
	n.AddChild(n)
}

func TestAddChildNilPanic(t *testing.T) {
	n := NewContainer("n")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil child, got none")
		}
	}()
	n.AddChild(nil)
}

// --- AddChildAt ---

func TestAddChildAt(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewContainer("c")
	parent.AddChild(a)
	parent.AddChild(c)

	parent.AddChildAt(b, 1) // insert between a and c

	if parent.NumChildren() != 3 {
		t.Fatalf("NumChildren = %d, want 3", parent.NumChildren())
	}
	if parent.ChildAt(0) != a || parent.ChildAt(1) != b || parent.ChildAt(2) != c {
		t.Error("children order should be [a, b, c]")
	}
}

func TestAddChildAtBeginning(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	parent.AddChild(a)
	parent.AddChildAt(b, 0)

	if parent.ChildAt(0) != b || parent.ChildAt(1) != a {
		t.Error("children order should be [b, a]")
	}
}

func TestAddChildAtEnd(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	parent.AddChild(a)
	parent.AddChildAt(b, 1)

	if parent.ChildAt(0) != a || parent.ChildAt(1) != b {
		t.Error("children order should be [a, b]")
	}
}

// --- RemoveChild ---

func TestRemoveChild(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)
	parent.RemoveChild(child)

	if parent.NumChildren() != 0 {
		t.Error("parent should have 0 children")
	}
	if child.Parent != nil {
		t.Error("child.Parent should be nil")
	}
}

func TestRemoveChildWrongParentPanic(t *testing.T) {
	p1 := NewContainer("p1")
	p2 := NewContainer("p2")
	child := NewContainer("child")
	p1.AddChild(child)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for wrong parent, got none")
		}
	}()
	p2.RemoveChild(child)
}

// --- RemoveChildAt ---

func TestRemoveChildAt(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewContainer("c")
	parent.AddChild(a)
	parent.AddChild(b)
	parent.AddChild(c)

	removed := parent.RemoveChildAt(1)
	if removed != b {
		t.Error("removed should be b")
	}
	if parent.NumChildren() != 2 {
		t.Errorf("NumChildren = %d, want 2", parent.NumChildren())
	}
	if parent.ChildAt(0) != a || parent.ChildAt(1) != c {
		t.Error("remaining children should be [a, c]")
	}
}

func TestRemoveChildAtOutOfBoundsPanic(t *testing.T) {
	parent := NewContainer("parent")
	parent.AddChild(NewContainer("a"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out of bounds, got none")
		}
	}()
	parent.RemoveChildAt(5)
}

// --- RemoveFromParent ---

func TestRemoveFromParent(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)
	child.RemoveFromParent()

	if parent.NumChildren() != 0 {
		t.Error("parent should have 0 children")
	}
	if child.Parent != nil {
		t.Error("child.Parent should be nil")
	}
}

func TestRemoveFromParentNoOp(t *testing.T) {
	n := NewContainer("orphan")
	n.RemoveFromParent() // should not panic
	if n.Parent != nil {
		t.Error("Parent should remain nil")
	}
}

// --- RemoveChildren ---

func TestRemoveChildren(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	parent.AddChild(a)
	parent.AddChild(b)
	parent.RemoveChildren()

	if parent.NumChildren() != 0 {
		t.Error("parent should have 0 children")
	}
	if a.Parent != nil || b.Parent != nil {
		t.Error("detached children should have nil Parent")
	}
}

// --- SetChildIndex ---

func TestSetChildIndex(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewContainer("c")
	parent.AddChild(a)
	parent.AddChild(b)
	parent.AddChild(c)

	// Move c to front
	parent.SetChildIndex(c, 0)
	if parent.ChildAt(0) != c || parent.ChildAt(1) != a || parent.ChildAt(2) != b {
		t.Errorf("after move to front: got [%s, %s, %s], want [c, a, b]",
			parent.ChildAt(0).Name, parent.ChildAt(1).Name, parent.ChildAt(2).Name)
	}

	// Move c to back
	parent.SetChildIndex(c, 2)
	if parent.ChildAt(0) != a || parent.ChildAt(1) != b || parent.ChildAt(2) != c {
		t.Errorf("after move to back: got [%s, %s, %s], want [a, b, c]",
			parent.ChildAt(0).Name, parent.ChildAt(1).Name, parent.ChildAt(2).Name)
	}
}

func TestSetChildIndexFirstToLast(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	parent.AddChild(a)
	parent.AddChild(b)

	// Move a (index 0) to index 1  -  the old bug case.
	parent.SetChildIndex(a, 1)
	if parent.ChildAt(0) != b || parent.ChildAt(1) != a {
		t.Errorf("got [%s, %s], want [b, a]",
			parent.ChildAt(0).Name, parent.ChildAt(1).Name)
	}
}

func TestSetChildIndexMiddle(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewContainer("c")
	d := NewContainer("d")
	parent.AddChild(a)
	parent.AddChild(b)
	parent.AddChild(c)
	parent.AddChild(d)

	// Move a (index 0) to index 2.
	parent.SetChildIndex(a, 2)
	names := ""
	for _, ch := range parent.Children() {
		names += ch.Name
	}
	if names != "bcad" {
		t.Errorf("got %q, want %q", names, "bcad")
	}

	// Move d (index 3) to index 1.
	parent.SetChildIndex(d, 1)
	names = ""
	for _, ch := range parent.Children() {
		names += ch.Name
	}
	if names != "bdca" {
		t.Errorf("got %q, want %q", names, "bdca")
	}
}

func TestSetChildIndexSamePosition(t *testing.T) {
	parent := NewContainer("parent")
	a := NewContainer("a")
	b := NewContainer("b")
	parent.AddChild(a)
	parent.AddChild(b)

	parent.SetChildIndex(a, 0) // no-op
	if parent.ChildAt(0) != a || parent.ChildAt(1) != b {
		t.Error("order should be unchanged")
	}
}

// --- Children / NumChildren / ChildAt consistency ---

func TestChildrenConsistency(t *testing.T) {
	parent := NewContainer("parent")
	nodes := make([]*Node, 5)
	for i := range nodes {
		nodes[i] = NewContainer("")
		parent.AddChild(nodes[i])
	}

	children := parent.Children()
	if len(children) != parent.NumChildren() {
		t.Errorf("Children() len = %d, NumChildren() = %d", len(children), parent.NumChildren())
	}
	for i, c := range children {
		if c != parent.ChildAt(i) {
			t.Errorf("Children()[%d] != ChildAt(%d)", i, i)
		}
	}
}

// --- Dispose ---

func TestDispose(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	grandchild := NewContainer("grandchild")
	root := NewContainer("root")
	root.AddChild(parent)
	parent.AddChild(child)
	child.AddChild(grandchild)

	parent.Dispose()

	if !parent.IsDisposed() {
		t.Error("parent should be disposed")
	}
	if !child.IsDisposed() {
		t.Error("child should be disposed")
	}
	if !grandchild.IsDisposed() {
		t.Error("grandchild should be disposed")
	}
	if parent.ID != 0 || child.ID != 0 || grandchild.ID != 0 {
		t.Error("disposed nodes should have ID = 0")
	}
	if root.NumChildren() != 0 {
		t.Error("root should have 0 children after dispose")
	}
}

func TestDisposeIdempotent(t *testing.T) {
	n := NewContainer("n")
	n.Dispose()
	n.Dispose() // should not panic
	if !n.IsDisposed() {
		t.Error("should still be disposed")
	}
}

// --- Dirty propagation ---

func TestDirtyPropagationOnAddChild(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	grandchild := NewContainer("grandchild")
	child.AddChild(grandchild)

	// Clear dirty flags manually
	child.TransformDirty = false
	grandchild.TransformDirty = false

	parent.AddChild(child)

	if !child.TransformDirty {
		t.Error("child should be dirty after AddChild")
	}
	// Grandchild inherits recomputation via parentRecomputed during
	// updateWorldTransform (upward-only dirty model). Its own flag
	// may not be set, but its world transform will be recomputed.
}

func TestDirtyPropagationOnRemoveChild(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	child.TransformDirty = false
	parent.RemoveChild(child)

	if !child.TransformDirty {
		t.Error("child should be dirty after RemoveChild")
	}
}
