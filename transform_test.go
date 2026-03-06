package willow

import (
	"math"
	"testing"
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

// --- computeLocalTransform ---

func TestLocalTransformIdentity(t *testing.T) {
	n := NewContainer("test")
	got := computeLocalTransform(n)
	assertMatrix(t, "identity", got, [6]float64{1, 0, 0, 1, 0, 0})
}

func TestLocalTransformTranslation(t *testing.T) {
	n := NewContainer("test")
	n.x = 10
	n.y = 20
	got := computeLocalTransform(n)
	assertMatrix(t, "translation", got, [6]float64{1, 0, 0, 1, 10, 20})
}

func TestLocalTransformScale(t *testing.T) {
	n := NewContainer("test")
	n.scaleX = 2
	n.scaleY = 3
	got := computeLocalTransform(n)
	assertMatrix(t, "scale", got, [6]float64{2, 0, 0, 3, 0, 0})
}

func TestLocalTransformRotation90(t *testing.T) {
	n := NewContainer("test")
	n.rotation = math.Pi / 2
	got := computeLocalTransform(n)
	// cos(90)=0, sin(90)=1 → a=0, b=1, c=-1, d=0
	assertMatrix(t, "rot90", got, [6]float64{0, 1, -1, 0, 0, 0})
}

func TestLocalTransformPivot(t *testing.T) {
	n := NewContainer("test")
	n.x = 100
	n.y = 200
	n.pivotX = 16
	n.pivotY = 16
	got := computeLocalTransform(n)
	// T(100,200) * T(-16,-16) = [1,0,0,1, 84, 184]
	assertMatrix(t, "pivot", got, [6]float64{1, 0, 0, 1, 84, 184})
}

func TestLocalTransformSkew(t *testing.T) {
	n := NewContainer("test")
	n.skewX = math.Pi / 4 // tan = 1
	got := computeLocalTransform(n)
	// After skew(π/4, 0): a=1, b=0, c=tan(π/4)=1, d=1
	// No rotation, so stays the same
	assertMatrix(t, "skew", got, [6]float64{1, 0, 1, 1, 0, 0})
}

func TestLocalTransformCombined(t *testing.T) {
	n := NewContainer("test")
	n.x = 50
	n.y = 100
	n.scaleX = 2
	n.scaleY = 2
	n.rotation = math.Pi / 2

	got := computeLocalTransform(n)
	// Scale(2,2) then Rotate(90°):
	// a = cos*sx - sin*0 = 0*2 = 0
	// b = sin*sx + cos*0 = 1*2 = 2
	// c = cos*0 - sin*sy = -1*2 = -2
	// d = sin*0 + cos*sy = 0 + 0*2 = 0
	// tx = 50, ty = 100
	assertMatrix(t, "combined", got, [6]float64{0, 2, -2, 0, 50, 100})
}

// --- multiplyAffine ---

func TestMultiplyAffineIdentity(t *testing.T) {
	id := identityTransform
	m := [6]float64{2, 1, 3, 4, 5, 6}
	assertMatrix(t, "id*m", multiplyAffine(id, m), m)
	assertMatrix(t, "m*id", multiplyAffine(m, id), m)
}

func TestMultiplyAffineTranslations(t *testing.T) {
	a := [6]float64{1, 0, 0, 1, 10, 20}
	b := [6]float64{1, 0, 0, 1, 5, 3}
	got := multiplyAffine(a, b)
	assertMatrix(t, "translations", got, [6]float64{1, 0, 0, 1, 15, 23})
}

// --- invertAffine ---

func TestInvertAffine(t *testing.T) {
	m := [6]float64{2, 0, 0, 3, 10, 20}
	inv := invertAffine(m)
	result := multiplyAffine(m, inv)
	assertMatrix(t, "m*inv=id", result, identityTransform)
}

func TestInvertAffineComplex(t *testing.T) {
	// Scale + rotation
	n := NewContainer("test")
	n.scaleX = 2
	n.rotation = math.Pi / 3
	m := computeLocalTransform(n)
	inv := invertAffine(m)
	result := multiplyAffine(m, inv)
	assertMatrix(t, "m*inv=id", result, identityTransform)
}

// --- updateWorldTransform ---

func TestWorldTransformParentChild(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.x = 100
	child.x = 10

	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	// parent world: [1,0,0,1,100,0]
	assertNear(t, "parent.tx", parent.worldTransform[4], 100)
	// child world: parent * child_local = [1,0,0,1,110,0]
	assertNear(t, "child.tx", child.worldTransform[4], 110)
}

func TestAlphaPropagation(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.alpha = 0.5
	child.alpha = 0.5

	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	assertNear(t, "parent.worldAlpha", parent.worldAlpha, 0.5)
	assertNear(t, "child.worldAlpha", child.worldAlpha, 0.25)
}

func TestDirtyFlagSkipsClean(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.x = 100
	child.x = 10
	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	// Clear dirty, change child X directly (without setter → stays clean)
	child.transformDirty = false
	parent.transformDirty = false
	child.x = 999 // dirty flag NOT set

	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	// Child should NOT have been recomputed since it's not dirty
	assertNear(t, "child.tx (stale)", child.worldTransform[4], 110)
}

func TestDirtyFlagRecomputes(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.x = 100
	child.x = 10
	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	child.SetPosition(20, 0) // marks dirty
	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	assertNear(t, "child.tx (updated)", child.worldTransform[4], 120)
}

func TestParentRecomputedPropagates(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.x = 100
	child.x = 10
	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	// Move parent  -  child is not directly dirty but must update
	parent.SetPosition(200, 0)
	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	assertNear(t, "child.tx (from parent)", child.worldTransform[4], 210)
}

// --- WorldToLocal / LocalToWorld ---

func TestWorldToLocalRoundtrip(t *testing.T) {
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child)

	parent.x = 100
	parent.y = 50
	child.x = 10
	child.y = 20
	child.scaleX = 2
	child.scaleY = 3
	child.rotation = math.Pi / 6

	updateWorldTransform(parent, identityTransform, 1.0, false, false)

	// Roundtrip test
	wx, wy := 150.0, 80.0
	lx, ly := child.WorldToLocal(wx, wy)
	wx2, wy2 := child.LocalToWorld(lx, ly)
	assertNear(t, "roundtrip.x", wx2, wx)
	assertNear(t, "roundtrip.y", wy2, wy)
}

func TestLocalToWorldIdentity(t *testing.T) {
	n := NewContainer("test")
	n.x = 50
	n.y = 100
	updateWorldTransform(n, identityTransform, 1.0, true, true)

	wx, wy := n.LocalToWorld(0, 0)
	assertNear(t, "origin.x", wx, 50)
	assertNear(t, "origin.y", wy, 100)
}

// --- Deep hierarchy ---

func TestDeepHierarchy(t *testing.T) {
	nodes := make([]*Node, 10)
	for i := range nodes {
		nodes[i] = NewContainer("")
		nodes[i].x = 10
		if i > 0 {
			nodes[i-1].AddChild(nodes[i])
		}
	}

	updateWorldTransform(nodes[0], identityTransform, 1.0, false, false)

	// Each level adds 10 to tx, so deepest (index 9) should have tx=100
	assertNear(t, "deep.tx", nodes[9].worldTransform[4], 100)
}

// --- Setters ---

func TestSettersDirty(t *testing.T) {
	n := NewContainer("test")
	n.transformDirty = false

	n.SetPosition(1, 2)
	if !n.transformDirty {
		t.Error("SetPosition should set dirty")
	}
	n.transformDirty = false

	n.SetScale(2, 2)
	if !n.transformDirty {
		t.Error("SetScale should set dirty")
	}
	n.transformDirty = false

	n.SetRotation(1)
	if !n.transformDirty {
		t.Error("SetRotation should set dirty")
	}
	n.transformDirty = false

	n.SetSkew(0.1, 0.2)
	if !n.transformDirty {
		t.Error("SetSkew should set dirty")
	}
	n.transformDirty = false

	n.SetPivot(5, 5)
	if !n.transformDirty {
		t.Error("SetPivot should set dirty")
	}
	n.transformDirty = false

	n.SetAlpha(0.5)
	if !n.alphaDirty {
		t.Error("SetAlpha should set alphaDirty")
	}
	n.alphaDirty = false

	n.Invalidate()
	if !n.transformDirty {
		t.Error("Invalidate should set dirty")
	}
}

// --- Singular matrix safety ---

func TestInvertAffineSingularReturnsIdentity(t *testing.T) {
	// ScaleX=0 produces a singular matrix (determinant=0).
	m := [6]float64{0, 0, 0, 1, 10, 20}
	inv := invertAffine(m)
	assertMatrix(t, "singular→identity", inv, identityTransform)
}

func TestInvertAffineBothZeroScales(t *testing.T) {
	// Scale(0, 0) → fully degenerate.
	m := [6]float64{0, 0, 0, 0, 50, 100}
	inv := invertAffine(m)
	assertMatrix(t, "zero-scale→identity", inv, identityTransform)
}

func TestWorldToLocalZeroScale(t *testing.T) {
	n := NewContainer("test")
	n.scaleX = 0
	n.scaleY = 0
	updateWorldTransform(n, identityTransform, 1.0, true, true)

	// Should not panic; returns identity-transformed point.
	lx, ly := n.WorldToLocal(100, 200)
	// With identity inverse, output = input.
	assertNear(t, "lx", lx, 100)
	assertNear(t, "ly", ly, 200)
}

// --- Benchmarks ---

func BenchmarkComputeLocalTransform(b *testing.B) {
	n := NewContainer("bench")
	n.x = 100
	n.y = 200
	n.scaleX = 2
	n.scaleY = 3
	n.rotation = 0.5
	n.pivotX = 16
	n.pivotY = 16
	b.ReportAllocs()
	for b.Loop() {
		_ = computeLocalTransform(n)
	}
}

func BenchmarkMultiplyAffine(b *testing.B) {
	a := [6]float64{2, 0.1, 0.3, 3, 100, 200}
	c := [6]float64{1.5, 0.2, 0.1, 2.5, 50, 30}
	b.ReportAllocs()
	for b.Loop() {
		_ = multiplyAffine(a, c)
	}
}

func BenchmarkUpdateWorldTransform10k(b *testing.B) {
	// Build a wide tree: root with 100 children, each with 100 grandchildren = 10,001 nodes
	root := NewContainer("root")
	for i := 0; i < 100; i++ {
		parent := NewContainer("")
		parent.x = float64(i)
		root.AddChild(parent)
		for j := 0; j < 100; j++ {
			child := NewContainer("")
			child.x = float64(j)
			parent.AddChild(child)
		}
	}

	// Warm up
	updateWorldTransform(root, identityTransform, 1.0, true, true)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Mark root dirty to force full recomputation
		root.transformDirty = true
		updateWorldTransform(root, identityTransform, 1.0, false, false)
	}
}

func BenchmarkUpdateWorldTransformStatic(b *testing.B) {
	root := NewContainer("root")
	for i := 0; i < 100; i++ {
		parent := NewContainer("")
		root.AddChild(parent)
		for j := 0; j < 100; j++ {
			child := NewContainer("")
			parent.AddChild(child)
		}
	}

	// Initial computation (clears dirty flags)
	updateWorldTransform(root, identityTransform, 1.0, true, true)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// All clean  -  should be near-zero cost
		updateWorldTransform(root, identityTransform, 1.0, false, false)
	}
}
