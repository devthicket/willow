package integration

import (
	"math"
	"testing"

	"github.com/tanema/gween/ease"

	. "github.com/devthicket/willow"
)

func TestTweenPositionReachesTarget(t *testing.T) {
	node := NewContainer("pos")
	node.X_ = 10
	node.Y_ = 20

	g := TweenPosition(node, 100, 200, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.X_-100) > 0.5 {
		t.Errorf("X = %f, want ~100", node.X_)
	}
	if math.Abs(node.Y_-200) > 0.5 {
		t.Errorf("Y = %f, want ~200", node.Y_)
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
	if math.Abs(node.ScaleX_-2.0) > 0.01 {
		t.Errorf("ScaleX = %f, want ~2.0", node.ScaleX_)
	}
	if math.Abs(node.ScaleY_-3.0) > 0.01 {
		t.Errorf("ScaleY = %f, want ~3.0", node.ScaleY_)
	}
}

func TestTweenColorAllComponents(t *testing.T) {
	node := NewContainer("color")
	node.Color_ = RGBA(1, 0, 0, 1)
	target := RGBA(0, 1, 0.5, 0.5)

	g := TweenColor(node, target, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	g.Update(0.5)
	g.Update(0.5)

	if !g.Done {
		t.Fatal("expected Done after full duration")
	}
	if math.Abs(node.Color_.R()-target.R()) > 0.01 {
		t.Errorf("R = %f, want %f", node.Color_.R(), target.R())
	}
	if math.Abs(node.Color_.G()-target.G()) > 0.01 {
		t.Errorf("G = %f, want %f", node.Color_.G(), target.G())
	}
	if math.Abs(node.Color_.B()-target.B()) > 0.01 {
		t.Errorf("B = %f, want %f", node.Color_.B(), target.B())
	}
	if math.Abs(node.Color_.A()-target.A()) > 0.01 {
		t.Errorf("A = %f, want %f", node.Color_.A(), target.A())
	}
}

func TestTweenAlphaInterpolates(t *testing.T) {
	node := NewContainer("alpha")
	node.Alpha_ = 1.0

	tw := TweenAlpha(node, 0.0, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	tw.Update(0.5)
	if tw.Done {
		t.Fatal("should not be done at halfway")
	}
	if math.Abs(node.Alpha_-0.5) > 0.05 {
		t.Errorf("Alpha = %f, want ~0.5 at halfway", node.Alpha_)
	}

	tw.Update(0.5)
	if !tw.Done {
		t.Fatal("should be done after full duration")
	}
	if math.Abs(node.Alpha_-0.0) > 0.01 {
		t.Errorf("Alpha = %f, want ~0.0", node.Alpha_)
	}
}

func TestTweenRotationReachesTarget(t *testing.T) {
	node := NewContainer("rot")
	node.Rotation_ = 0

	tw := TweenRotation(node, math.Pi, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	tw.Update(0.5)
	tw.Update(0.5)

	if !tw.Done {
		t.Fatal("expected done after full duration")
	}
	if math.Abs(node.Rotation_-math.Pi) > 0.05 {
		t.Errorf("Rotation = %f, want ~%f", node.Rotation_, math.Pi)
	}
}

func TestTweenGroupDoneFlagTransition(t *testing.T) {
	node := NewContainer("done")
	g := TweenPosition(node, 50, 50, TweenConfig{Duration: 0.5, Ease: ease.Linear})

	if g.Done {
		t.Fatal("should not be Done at start")
	}

	g.Update(0.25)
	if g.Done {
		t.Fatal("should not be Done partway through")
	}

	g.Update(0.25)
	if !g.Done {
		t.Fatal("should be Done after full duration")
	}

	g.Update(0.1)
	if !g.Done {
		t.Fatal("should remain Done")
	}
}

func TestTweenGroupMarksDirty(t *testing.T) {
	node := NewContainer("dirty")

	node.TransformDirty = false

	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})
	g.Update(0.1)

	if !node.TransformDirty {
		t.Fatal("expected node to be marked dirty after TweenGroup update")
	}
}

func TestTweenGroupDisposedNode(t *testing.T) {
	node := NewContainer("disposed")
	node.X_ = 10
	node.Y_ = 20

	g := TweenPosition(node, 100, 200, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	node.Dispose()

	g.Update(0.1)

	if !g.Done {
		t.Fatal("expected Done after disposed node detected")
	}
	if node.X_ != 10 {
		t.Errorf("X changed to %f on disposed node", node.X_)
	}
	if node.Y_ != 20 {
		t.Errorf("Y changed to %f on disposed node", node.Y_)
	}
}

func TestTweenGroupDisposedMidAnimation(t *testing.T) {
	node := NewContainer("mid-dispose")

	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	g.Update(0.1)
	g.Update(0.1)
	if g.Done {
		t.Fatal("should not be Done yet")
	}

	node.Dispose()
	savedX := node.X_
	savedY := node.Y_

	g.Update(0.1)
	if !g.Done {
		t.Fatal("expected Done after node disposed mid-animation")
	}
	if node.X_ != savedX || node.Y_ != savedY {
		t.Error("node fields should not change after disposal")
	}
}

func TestTweenEasingFunctionsProduceDifferentCurves(t *testing.T) {
	nodeL := NewContainer("linear")
	nodeC := NewContainer("cubic")

	gL := TweenPosition(nodeL, 100, 0, TweenConfig{Duration: 1.0, Ease: ease.Linear})
	gC := TweenPosition(nodeC, 100, 0, TweenConfig{Duration: 1.0, Ease: ease.OutCubic})

	gL.Update(0.5)
	gC.Update(0.5)

	if math.Abs(nodeL.X_-nodeC.X_) < 1.0 {
		t.Errorf("easing curves should produce different values at midpoint: linear=%f cubic=%f", nodeL.X_, nodeC.X_)
	}
}

func TestTweenGroupUpdateZeroAlloc(t *testing.T) {
	node := NewContainer("alloc")
	g := TweenPosition(node, 100, 100, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	g.Update(0.01)

	result := testing.AllocsPerRun(100, func() {
		g.Update(0.001)
	})
	if result > 0 {
		t.Errorf("TweenGroup.Update allocated %f times per run, want 0", result)
	}
}
