package willow

import (
	"math"
	"testing"

	"github.com/tanema/gween/ease"
)

func TestTweenPositionReachesTarget(t *testing.T) {
	node := NewContainer("pos")
	node.X = 10
	node.Y = 20

	g := TweenPosition(node, 100, 200, 1.0, ease.Linear)

	// Run for full duration using exact halves to avoid float32 accumulation drift.
	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.X-100) > 0.5 {
		t.Errorf("X = %f, want ~100", node.X)
	}
	if math.Abs(node.Y-200) > 0.5 {
		t.Errorf("Y = %f, want ~200", node.Y)
	}
}

func TestTweenScaleReachesTarget(t *testing.T) {
	node := NewContainer("scale")

	g := TweenScale(node, 2.0, 3.0, 0.5, ease.Linear)

	g.Update(0.25)
	g.Update(0.25)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.ScaleX-2.0) > 0.01 {
		t.Errorf("ScaleX = %f, want ~2.0", node.ScaleX)
	}
	if math.Abs(node.ScaleY-3.0) > 0.01 {
		t.Errorf("ScaleY = %f, want ~3.0", node.ScaleY)
	}
}

func TestTweenColorAllComponents(t *testing.T) {
	node := NewContainer("color")
	node.Color = Color{R: 1, G: 0, B: 0, A: 1}
	target := Color{R: 0, G: 1, B: 0.5, A: 0.5}

	g := TweenColor(node, target, 1.0, ease.Linear)

	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.Color.R-target.R) > 0.01 {
		t.Errorf("R = %f, want %f", node.Color.R, target.R)
	}
	if math.Abs(node.Color.G-target.G) > 0.01 {
		t.Errorf("G = %f, want %f", node.Color.G, target.G)
	}
	if math.Abs(node.Color.B-target.B) > 0.01 {
		t.Errorf("B = %f, want %f", node.Color.B, target.B)
	}
	if math.Abs(node.Color.A-target.A) > 0.01 {
		t.Errorf("A = %f, want %f", node.Color.A, target.A)
	}
}

func TestTweenAlphaInterpolates(t *testing.T) {
	node := NewContainer("alpha")
	node.Alpha = 1.0

	tw := TweenAlpha(node, 0.0, 1.0, ease.Linear)

	// Halfway through.
	tw.Update(0.5)
	if tw.Done {
		t.Fatal("should not be done at halfway")
	}
	if math.Abs(node.Alpha-0.5) > 0.05 {
		t.Errorf("Alpha = %f, want ~0.5 at halfway", node.Alpha)
	}

	// Finish.
	tw.Update(0.5)
	if !tw.Done {
		t.Fatal("should be done after full duration")
	}
	if math.Abs(node.Alpha-0.0) > 0.01 {
		t.Errorf("Alpha = %f, want ~0.0", node.Alpha)
	}
}

func TestTweenRotationReachesTarget(t *testing.T) {
	node := NewContainer("rot")
	node.Rotation = 0

	tw := TweenRotation(node, math.Pi, 1.0, ease.Linear)

	tw.Update(0.5)
	tw.Update(0.5)

	if !tw.Done {
		t.Fatal("expected done after full duration")
	}
	if math.Abs(node.Rotation-math.Pi) > 0.05 {
		t.Errorf("Rotation = %f, want ~%f", node.Rotation, math.Pi)
	}
}

func TestTweenGroupDoneFlagTransition(t *testing.T) {
	node := NewContainer("done")
	g := TweenPosition(node, 50, 50, 0.5, ease.Linear)

	if g.Done {
		t.Fatal("should not be Done at start")
	}

	// Partway through  -  not done.
	g.Update(0.25)
	if g.Done {
		t.Fatal("should not be Done partway through")
	}

	// Complete.
	g.Update(0.25)
	if !g.Done {
		t.Fatal("should be Done after full duration")
	}

	// Update after done  -  should be a no-op, not panic.
	g.Update(0.1)
	if !g.Done {
		t.Fatal("should remain Done")
	}
}

func TestTweenGroupMarksDirty(t *testing.T) {
	node := NewContainer("dirty")

	// Clear the dirty flag first.
	node.transformDirty = false

	g := TweenPosition(node, 100, 100, 1.0, ease.Linear)
	g.Update(0.1)

	if !node.transformDirty {
		t.Fatal("expected node to be marked dirty after TweenGroup update")
	}
}

func TestTweenGroupDisposedNode(t *testing.T) {
	node := NewContainer("disposed")
	node.X = 10
	node.Y = 20

	g := TweenPosition(node, 100, 200, 1.0, ease.Linear)

	// Dispose the node before tweening.
	node.Dispose()

	g.Update(0.1)

	if !g.Done {
		t.Fatal("expected Done after disposed node detected")
	}
	// Values should not have changed.
	if node.X != 10 {
		t.Errorf("X changed to %f on disposed node", node.X)
	}
	if node.Y != 20 {
		t.Errorf("Y changed to %f on disposed node", node.Y)
	}
}

func TestTweenGroupDisposedMidAnimation(t *testing.T) {
	node := NewContainer("mid-dispose")

	g := TweenPosition(node, 100, 100, 1.0, ease.Linear)

	// Run a few frames.
	g.Update(0.1)
	g.Update(0.1)
	if g.Done {
		t.Fatal("should not be Done yet")
	}

	// Dispose mid-animation.
	node.Dispose()
	savedX := node.X
	savedY := node.Y

	g.Update(0.1)
	if !g.Done {
		t.Fatal("expected Done after node disposed mid-animation")
	}
	if node.X != savedX || node.Y != savedY {
		t.Error("node fields should not change after disposal")
	}
}

func TestTweenEasingFunctionsProduceDifferentCurves(t *testing.T) {
	// Spot-check: linear vs OutCubic at the midpoint should differ.
	nodeL := NewContainer("linear")
	nodeC := NewContainer("cubic")

	gL := TweenPosition(nodeL, 100, 0, 1.0, ease.Linear)
	gC := TweenPosition(nodeC, 100, 0, 1.0, ease.OutCubic)

	// Advance to midpoint.
	gL.Update(0.5)
	gC.Update(0.5)

	// OutCubic should be ahead of linear at midpoint.
	if math.Abs(nodeL.X-nodeC.X) < 1.0 {
		t.Errorf("easing curves should produce different values at midpoint: linear=%f cubic=%f", nodeL.X, nodeC.X)
	}
}

func TestTweenGroupUpdateZeroAlloc(t *testing.T) {
	node := NewContainer("alloc")
	g := TweenPosition(node, 100, 100, 1.0, ease.Linear)

	// Warm up  -  first call might differ.
	g.Update(0.01)

	result := testing.AllocsPerRun(100, func() {
		g.Update(0.001)
	})
	if result > 0 {
		t.Errorf("TweenGroup.Update allocated %f times per run, want 0", result)
	}
}
