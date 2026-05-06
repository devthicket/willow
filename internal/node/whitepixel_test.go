package node

import (
	"image/color"
	"testing"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// newWhitePixelSprite mirrors what willow.NewSprite does for an empty
// TextureRegion: assigns the shared 1×1 WhitePixelImage and seeds the
// intrinsic display size to 1×1. Restores the package-level WhitePixelImage
// to its prior value at test end so the global mutation doesn't leak.
func newWhitePixelSprite(t *testing.T) *Node {
	t.Helper()
	prev := WhitePixelImage
	if WhitePixelImage == nil {
		img := ebiten.NewImage(1, 1)
		img.Fill(color.White)
		WhitePixelImage = img
		t.Cleanup(func() { WhitePixelImage = prev })
	}
	n := NewNode("wp", types.NodeTypeSprite)
	n.CustomImage_ = WhitePixelImage
	n.WhitePixelW_ = 1
	n.WhitePixelH_ = 1
	return n
}

// TestWhitePixelPivotInDisplayPixels verifies that on a white-pixel sprite
// resized via SetSize(64, 96), SetPivot(32, 48) places the world transform's
// translation at (-32, -48) — i.e. pivot is interpreted in display-pixel
// space, not in 1×1 source-pixel space.
func TestWhitePixelPivotInDisplayPixels(t *testing.T) {
	n := newWhitePixelSprite(t)
	n.SetSize(64, 96)
	n.SetPivot(32, 48)

	got := ComputeLocalTransform(n)
	// Scale stays at 1×1; the W×H size lives in WhitePixelW_/H_, so:
	//   a=1, d=1, tx = -PivotX*ScaleX + X = -32, ty = -48.
	assertMatrix(t, "wp pivot", got, [6]float64{1, 0, 0, 1, -32, -48})
}

// TestWhitePixelPivotPercentCenters verifies SetPivotPercent(0.5, 0.5) on a
// white-pixel sprite yields the same centered transform as SetPivot(W/2, H/2).
func TestWhitePixelPivotPercentCenters(t *testing.T) {
	n := newWhitePixelSprite(t)
	n.SetSize(64, 96)
	n.SetPivotPercent(0.5, 0.5)

	got := ComputeLocalTransform(n)
	assertMatrix(t, "wp anchor", got, [6]float64{1, 0, 0, 1, -32, -48})
}

// TestWhitePixelSetSizeKeepsScale verifies that SetSize on a white-pixel
// sprite writes intrinsic size and leaves Scale at its current value, so
// later SetScale calls compose with the size cleanly.
func TestWhitePixelSetSizeKeepsScale(t *testing.T) {
	n := newWhitePixelSprite(t)
	n.SetSize(50, 30)
	if n.ScaleX_ != 1 || n.ScaleY_ != 1 {
		t.Fatalf("Scale = (%v, %v), want (1, 1)", n.ScaleX_, n.ScaleY_)
	}
	if n.WhitePixelW_ != 50 || n.WhitePixelH_ != 30 {
		t.Fatalf("WhitePixelW/H = (%v, %v), want (50, 30)", n.WhitePixelW_, n.WhitePixelH_)
	}
	if w, h := n.Width(), n.Height(); w != 50 || h != 30 {
		t.Fatalf("Width/Height = (%v, %v), want (50, 30)", w, h)
	}
	n.SetScale(2, 3)
	if w, h := n.Width(), n.Height(); w != 100 || h != 90 {
		t.Fatalf("Width/Height after SetScale = (%v, %v), want (100, 90)", w, h)
	}
}

// TestWhitePixelWorldBounds verifies WorldBounds reports the W×H quad for a
// white-pixel sprite — the camera follow path reads this for AABB culling
// and for screen-space queries.
func TestWhitePixelWorldBounds(t *testing.T) {
	n := newWhitePixelSprite(t)
	n.SetSize(64, 96)
	n.WorldTransform = [6]float64{1, 0, 0, 1, 100, 200}
	b := n.WorldBounds()
	if b.X != 100 || b.Y != 200 || b.Width != 64 || b.Height != 96 {
		t.Fatalf("WorldBounds = %v, want {100, 200, 64, 96}", b)
	}
}
