package willow

import (
	"math"
	"testing"

	"github.com/tanema/gween/ease"
)

func TestTweenPositionReachesTarget(t *testing.T) {
	node := NewContainer("pos")
	node.x = 10
	node.y = 20

	g := TweenPosition(node, 100, 200, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Run for full duration using exact halves to avoid float32 accumulation drift.
	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.x-100) > 0.5 {
		t.Errorf("X = %f, want ~100", node.x)
	}
	if math.Abs(node.y-200) > 0.5 {
		t.Errorf("Y = %f, want ~200", node.y)
	}
}

func TestTweenScaleReachesTarget(t *testing.T) {
	node := NewContainer("scale")

	g := TweenScale(node, 2.0, 3.0, TweenConfig{Duration: 0.5, Ease: ease.Linear})

	g.Update(0.25)
	g.Update(0.25)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.scaleX-2.0) > 0.01 {
		t.Errorf("ScaleX = %f, want ~2.0", node.scaleX)
	}
	if math.Abs(node.scaleY-3.0) > 0.01 {
		t.Errorf("ScaleY = %f, want ~3.0", node.scaleY)
	}
}

func TestTweenColorAllComponents(t *testing.T) {
	node := NewContainer("color")
	node.color = Color{r: 1, g: 0, b: 0, a: 1}
	target := Color{r: 0, g: 1, b: 0.5, a: 0.5}

	g := TweenColor(node, target, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.color.r-target.r) > 0.01 {
		t.Errorf("R = %f, want %f", node.color.r, target.r)
	}
	if math.Abs(node.color.g-target.g) > 0.01 {
		t.Errorf("G = %f, want %f", node.color.g, target.g)
	}
	if math.Abs(node.color.b-target.b) > 0.01 {
		t.Errorf("B = %f, want %f", node.color.b, target.b)
	}
	if math.Abs(node.color.a-target.a) > 0.01 {
		t.Errorf("A = %f, want %f", node.color.a, target.a)
	}
}

func TestTweenAlphaInterpolates(t *testing.T) {
	node := NewContainer("alpha")
	node.alpha = 1.0

	tw := TweenAlpha(node, 0.0, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Halfway through.
	tw.Update(0.5)
	if tw.Done {
		t.Fatal("should not be done at halfway")
	}
	if math.Abs(node.alpha-0.5) > 0.05 {
		t.Errorf("Alpha = %f, want ~0.5 at halfway", node.alpha)
	}

	// Finish.
	tw.Update(0.5)
	if !tw.Done {
		t.Fatal("should be done after full duration")
	}
	if math.Abs(node.alpha-0.0) > 0.01 {
		t.Errorf("Alpha = %f, want ~0.0", node.alpha)
	}
}

func TestTweenRotationReachesTarget(t *testing.T) {
	node := NewContainer("rot")
	node.rotation = 0

	tw := TweenRotation(node, math.Pi, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	tw.Update(0.5)
	tw.Update(0.5)

	if !tw.Done {
		t.Fatal("expected done after full duration")
	}
	if math.Abs(node.rotation-math.Pi) > 0.05 {
		t.Errorf("Rotation = %f, want ~%f", node.rotation, math.Pi)
	}
}

func TestTweenGroupDoneFlagTransition(t *testing.T) {
	node := NewContainer("done")
	g := TweenPosition(node, 50, 50, TweenConfig{Duration: 0.5, Ease: ease.Linear})

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

	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})
	g.Update(0.1)

	if !node.transformDirty {
		t.Fatal("expected node to be marked dirty after TweenGroup update")
	}
}

func TestTweenGroupDisposedNode(t *testing.T) {
	node := NewContainer("disposed")
	node.x = 10
	node.y = 20

	g := TweenPosition(node, 100, 200, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Dispose the node before tweening.
	node.Dispose()

	g.Update(0.1)

	if !g.Done {
		t.Fatal("expected Done after disposed node detected")
	}
	// Values should not have changed.
	if node.x != 10 {
		t.Errorf("X changed to %f on disposed node", node.x)
	}
	if node.y != 20 {
		t.Errorf("Y changed to %f on disposed node", node.y)
	}
}

func TestTweenGroupDisposedMidAnimation(t *testing.T) {
	node := NewContainer("mid-dispose")

	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Run a few frames.
	g.Update(0.1)
	g.Update(0.1)
	if g.Done {
		t.Fatal("should not be Done yet")
	}

	// Dispose mid-animation.
	node.Dispose()
	savedX := node.x
	savedY := node.y

	g.Update(0.1)
	if !g.Done {
		t.Fatal("expected Done after node disposed mid-animation")
	}
	if node.x != savedX || node.y != savedY {
		t.Error("node fields should not change after disposal")
	}
}

func TestTweenEasingFunctionsProduceDifferentCurves(t *testing.T) {
	// Spot-check: linear vs OutCubic at the midpoint should differ.
	nodeL := NewContainer("linear")
	nodeC := NewContainer("cubic")

	gL := TweenPosition(nodeL, 100, 0, TweenConfig{Duration: 1.0, Ease: ease.Linear})
	gC := TweenPosition(nodeC, 100, 0, TweenConfig{Duration: 1.0, Ease: ease.OutCubic})

	// Advance to midpoint.
	gL.Update(0.5)
	gC.Update(0.5)

	// OutCubic should be ahead of linear at midpoint.
	if math.Abs(nodeL.x-nodeC.x) < 1.0 {
		t.Errorf("easing curves should produce different values at midpoint: linear=%f cubic=%f", nodeL.x, nodeC.x)
	}
}

func TestTweenGroupUpdateZeroAlloc(t *testing.T) {
	node := NewContainer("alloc")
	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Warm up  -  first call might differ.
	g.Update(0.01)

	result := testing.AllocsPerRun(100, func() {
		g.Update(0.001)
	})
	if result > 0 {
		t.Errorf("TweenGroup.Update allocated %f times per run, want 0", result)
	}
}
