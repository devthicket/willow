package node

import (
	"math"
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

const epsilon = 1e-9

func assertNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > epsilon {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertMatrix(t *testing.T, name string, got, want [6]float64) {
	t.Helper()
	for i := range got {
		if math.Abs(got[i]-want[i]) > epsilon {
			t.Errorf("%s[%d] = %v, want %v (full: %v vs %v)", name, i, got[i], want[i], got, want)
		}
	}
}

// --- ComputeLocalTransform ---

func TestLocalTransformIdentity(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	got := ComputeLocalTransform(n)
	assertMatrix(t, "identity", got, [6]float64{1, 0, 0, 1, 0, 0})
}

func TestLocalTransformTranslation(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.X_ = 10
	n.Y_ = 20
	got := ComputeLocalTransform(n)
	assertMatrix(t, "translation", got, [6]float64{1, 0, 0, 1, 10, 20})
}

func TestLocalTransformScale(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.ScaleX_ = 2
	n.ScaleY_ = 3
	got := ComputeLocalTransform(n)
	assertMatrix(t, "scale", got, [6]float64{2, 0, 0, 3, 0, 0})
}

func TestLocalTransformRotation90(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.Rotation_ = math.Pi / 2
	got := ComputeLocalTransform(n)
	// cos(90)=0, sin(90)=1 -> a=0, b=1, c=-1, d=0
	assertMatrix(t, "rot90", got, [6]float64{0, 1, -1, 0, 0, 0})
}

func TestLocalTransformPivot(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.X_ = 100
	n.Y_ = 200
	n.PivotX_ = 16
	n.PivotY_ = 16
	got := ComputeLocalTransform(n)
	// T(100,200) * T(-16,-16) = [1,0,0,1, 84, 184]
	assertMatrix(t, "pivot", got, [6]float64{1, 0, 0, 1, 84, 184})
}

func TestLocalTransformSkew(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.SkewX_ = math.Pi / 4 // tan = 1
	got := ComputeLocalTransform(n)
	// After skew(pi/4, 0): a=1, b=0, c=tan(pi/4)=1, d=1
	assertMatrix(t, "skew", got, [6]float64{1, 0, 1, 1, 0, 0})
}

func TestLocalTransformCombined(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.X_ = 50
	n.Y_ = 100
	n.ScaleX_ = 2
	n.ScaleY_ = 2
	n.Rotation_ = math.Pi / 2

	got := ComputeLocalTransform(n)
	// Scale(2,2) then Rotate(90deg):
	// a = cos*sx = 0*2 = 0
	// b = sin*sx = 1*2 = 2
	// c = -sin*sy = -1*2 = -2
	// d = cos*sy = 0*2 = 0
	// tx = 50, ty = 100
	assertMatrix(t, "combined", got, [6]float64{0, 2, -2, 0, 50, 100})
}

// --- MultiplyAffine ---

func TestMultiplyAffineIdentity(t *testing.T) {
	id := IdentityTransform
	m := [6]float64{2, 1, 3, 4, 5, 6}
	assertMatrix(t, "id*m", MultiplyAffine(id, m), m)
	assertMatrix(t, "m*id", MultiplyAffine(m, id), m)
}

func TestMultiplyAffineTranslations(t *testing.T) {
	a := [6]float64{1, 0, 0, 1, 10, 20}
	b := [6]float64{1, 0, 0, 1, 5, 3}
	got := MultiplyAffine(a, b)
	assertMatrix(t, "translations", got, [6]float64{1, 0, 0, 1, 15, 23})
}

// --- InvertAffine ---

func TestInvertAffine(t *testing.T) {
	m := [6]float64{2, 0, 0, 3, 10, 20}
	inv := InvertAffine(m)
	result := MultiplyAffine(m, inv)
	assertMatrix(t, "m*inv=id", result, IdentityTransform)
}

func TestInvertAffineComplex(t *testing.T) {
	// Scale + rotation
	n := NewNode("test", types.NodeTypeContainer)
	n.ScaleX_ = 2
	n.Rotation_ = math.Pi / 3
	m := ComputeLocalTransform(n)
	inv := InvertAffine(m)
	result := MultiplyAffine(m, inv)
	assertMatrix(t, "m*inv=id", result, IdentityTransform)
}

// --- UpdateWorldTransform ---

func TestWorldTransformParentChild(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.X_ = 100
	child.X_ = 10

	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	// parent world: [1,0,0,1,100,0]
	assertNear(t, "parent.tx", parent.WorldTransform[4], 100)
	// child world: parent * child_local = [1,0,0,1,110,0]
	assertNear(t, "child.tx", child.WorldTransform[4], 110)
}

func TestAlphaPropagation(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.Alpha_ = 0.5
	child.Alpha_ = 0.5

	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	assertNear(t, "parent.WorldAlpha", parent.WorldAlpha, 0.5)
	assertNear(t, "child.WorldAlpha", child.WorldAlpha, 0.25)
}

func TestDirtyFlagSkipsClean(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.X_ = 100
	child.X_ = 10
	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	// Clear dirty, change child X directly (without setter -> stays clean)
	child.TransformDirty = false
	parent.TransformDirty = false
	child.X_ = 999 // dirty flag NOT set

	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	// Child should NOT have been recomputed since it's not dirty
	assertNear(t, "child.tx (stale)", child.WorldTransform[4], 110)
}

func TestDirtyFlagRecomputes(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.X_ = 100
	child.X_ = 10
	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	child.SetPosition(20, 0) // marks dirty
	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	assertNear(t, "child.tx (updated)", child.WorldTransform[4], 120)
}

func TestParentRecomputedPropagates(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.X_ = 100
	child.X_ = 10
	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	// Move parent - child is not directly dirty but must update
	parent.SetPosition(200, 0)
	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	assertNear(t, "child.tx (from parent)", child.WorldTransform[4], 210)
}

// --- WorldToLocal / LocalToWorld ---

func TestWorldToLocalRoundtrip(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	parent.X_ = 100
	parent.Y_ = 50
	child.X_ = 10
	child.Y_ = 20
	child.ScaleX_ = 2
	child.ScaleY_ = 3
	child.Rotation_ = math.Pi / 6

	UpdateWorldTransform(parent, IdentityTransform, 1.0, false, false)

	// Roundtrip test
	wx, wy := 150.0, 80.0
	lx, ly := child.WorldToLocal(wx, wy)
	wx2, wy2 := child.LocalToWorld(lx, ly)
	assertNear(t, "roundtrip.x", wx2, wx)
	assertNear(t, "roundtrip.y", wy2, wy)
}

func TestLocalToWorldIdentity(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.X_ = 50
	n.Y_ = 100
	UpdateWorldTransform(n, IdentityTransform, 1.0, true, true)

	wx, wy := n.LocalToWorld(0, 0)
	assertNear(t, "origin.x", wx, 50)
	assertNear(t, "origin.y", wy, 100)
}

// --- Deep hierarchy ---

func TestDeepHierarchy(t *testing.T) {
	nodes := make([]*Node, 10)
	for i := range nodes {
		nodes[i] = NewNode("", types.NodeTypeContainer)
		nodes[i].X_ = 10
		if i > 0 {
			nodes[i-1].AddChild(nodes[i])
		}
	}

	UpdateWorldTransform(nodes[0], IdentityTransform, 1.0, false, false)

	// Each level adds 10 to tx, so deepest (index 9) should have tx=100
	assertNear(t, "deep.tx", nodes[9].WorldTransform[4], 100)
}

// --- Setters ---

func TestSettersDirty(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.TransformDirty = false

	n.SetPosition(1, 2)
	if !n.TransformDirty {
		t.Error("SetPosition should set dirty")
	}
	n.TransformDirty = false

	n.SetScale(2, 2)
	if !n.TransformDirty {
		t.Error("SetScale should set dirty")
	}
	n.TransformDirty = false

	n.SetRotation(1)
	if !n.TransformDirty {
		t.Error("SetRotation should set dirty")
	}
	n.TransformDirty = false

	n.SetSkew(0.1, 0.2)
	if !n.TransformDirty {
		t.Error("SetSkew should set dirty")
	}
	n.TransformDirty = false

	n.SetPivot(5, 5)
	if !n.TransformDirty {
		t.Error("SetPivot should set dirty")
	}
	n.TransformDirty = false

	n.SetAlpha(0.5)
	if !n.AlphaDirty {
		t.Error("SetAlpha should set alphaDirty")
	}
	n.AlphaDirty = false

	n.Invalidate()
	if !n.TransformDirty {
		t.Error("Invalidate should set dirty")
	}
}

// --- Singular matrix safety ---

func TestInvertAffineSingularReturnsIdentity(t *testing.T) {
	// ScaleX=0 produces a singular matrix (determinant=0).
	m := [6]float64{0, 0, 0, 1, 10, 20}
	inv := InvertAffine(m)
	assertMatrix(t, "singular->identity", inv, IdentityTransform)
}

func TestInvertAffineBothZeroScales(t *testing.T) {
	// Scale(0, 0) -> fully degenerate.
	m := [6]float64{0, 0, 0, 0, 50, 100}
	inv := InvertAffine(m)
	assertMatrix(t, "zero-scale->identity", inv, IdentityTransform)
}

func TestWorldToLocalZeroScale(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.ScaleX_ = 0
	n.ScaleY_ = 0
	UpdateWorldTransform(n, IdentityTransform, 1.0, true, true)

	// Should not panic; returns identity-transformed point.
	lx, ly := n.WorldToLocal(100, 200)
	// With identity inverse, output = input.
	assertNear(t, "lx", lx, 100)
	assertNear(t, "ly", ly, 200)
}

// --- Benchmarks ---

func BenchmarkComputeLocalTransform(b *testing.B) {
	n := NewNode("bench", types.NodeTypeContainer)
	n.X_ = 100
	n.Y_ = 200
	n.ScaleX_ = 2
	n.ScaleY_ = 3
	n.Rotation_ = 0.5
	n.PivotX_ = 16
	n.PivotY_ = 16
	b.ReportAllocs()
	for b.Loop() {
		_ = ComputeLocalTransform(n)
	}
}

func BenchmarkMultiplyAffine(b *testing.B) {
	a := [6]float64{2, 0.1, 0.3, 3, 100, 200}
	c := [6]float64{1.5, 0.2, 0.1, 2.5, 50, 30}
	b.ReportAllocs()
	for b.Loop() {
		_ = MultiplyAffine(a, c)
	}
}

func BenchmarkUpdateWorldTransform10k(b *testing.B) {
	// Build a wide tree: root with 100 children, each with 100 grandchildren = 10,001 nodes
	root := NewNode("root", types.NodeTypeContainer)
	for i := 0; i < 100; i++ {
		parent := NewNode("", types.NodeTypeContainer)
		parent.X_ = float64(i)
		root.AddChild(parent)
		for j := 0; j < 100; j++ {
			child := NewNode("", types.NodeTypeContainer)
			child.X_ = float64(j)
			parent.AddChild(child)
		}
	}

	// Warm up
	UpdateWorldTransform(root, IdentityTransform, 1.0, true, true)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Mark root dirty to force full recomputation
		root.TransformDirty = true
		UpdateWorldTransform(root, IdentityTransform, 1.0, false, false)
	}
}

func BenchmarkUpdateWorldTransformStatic(b *testing.B) {
	root := NewNode("root", types.NodeTypeContainer)
	for i := 0; i < 100; i++ {
		parent := NewNode("", types.NodeTypeContainer)
		root.AddChild(parent)
		for j := 0; j < 100; j++ {
			child := NewNode("", types.NodeTypeContainer)
			parent.AddChild(child)
		}
	}

	// Initial computation (clears dirty flags)
	UpdateWorldTransform(root, IdentityTransform, 1.0, true, true)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// All clean - should be near-zero cost
		UpdateWorldTransform(root, IdentityTransform, 1.0, false, false)
	}
}
