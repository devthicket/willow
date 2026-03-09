package willow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSetMask(t *testing.T) {
	n := NewSprite("target", TextureRegion{Width: 32, Height: 32})
	m := NewSprite("mask", TextureRegion{Width: 32, Height: 32})
	n.SetMask(m)
	if n.MaskNode != m {
		t.Error("mask should be set")
	}
}

func TestClearMask(t *testing.T) {
	n := NewSprite("target", TextureRegion{Width: 32, Height: 32})
	m := NewSprite("mask", TextureRegion{Width: 32, Height: 32})
	n.SetMask(m)
	n.ClearMask()
	if n.MaskNode != nil {
		t.Error("mask should be nil after ClearMask")
	}
}

func TestGetMask(t *testing.T) {
	n := NewSprite("target", TextureRegion{Width: 32, Height: 32})
	if n.GetMask() != nil {
		t.Error("GetMask should return nil by default")
	}
	m := NewSprite("mask", TextureRegion{Width: 32, Height: 32})
	n.SetMask(m)
	if n.GetMask() != m {
		t.Error("GetMask should return the mask node")
	}
}

func TestMaskNodeNotInSceneTree(t *testing.T) {
	s := NewScene()
	target := NewSprite("target", TextureRegion{Width: 32, Height: 32})
	maskNode := NewSprite("mask", TextureRegion{Width: 32, Height: 32})
	target.SetMask(maskNode)
	s.Root().AddChild(target)

	// The mask node should not have a parent (it's not in the tree).
	if maskNode.Parent != nil {
		t.Error("mask node should not be in the scene tree")
	}
}

func TestDisposeCleansMask(t *testing.T) {
	n := NewSprite("target", TextureRegion{Width: 32, Height: 32})
	m := NewSprite("mask", TextureRegion{Width: 32, Height: 32})
	n.SetMask(m)
	n.Dispose()
	if n.MaskNode != nil {
		t.Error("mask should be nil after dispose")
	}
}

func TestDisposeCleansCache(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.SetCacheAsTexture(true)
	n.Dispose()
	if n.CacheEnabled {
		t.Error("cacheEnabled should be false after dispose")
	}
	if n.CacheTexture != nil {
		t.Error("cacheTexture should be nil after dispose")
	}
}

// --- Special node detection in traverse ---

func TestSpecialNodeWithFilterSkipsNormalEmission(t *testing.T) {
	s := NewScene()
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	n.Filters = []any{NewColorMatrixFilter()}
	s.Root().AddChild(n)

	traverseScene(s)

	// Should emit exactly 1 command (the directImage command from renderSpecialNode).
	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	if s.commands[0].directImage == nil {
		t.Error("command should have directImage set for filtered node")
	}
}

func TestSpecialNodeWithCacheEmitsDirectImage(t *testing.T) {
	s := NewScene()
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	n.SetCacheAsTexture(true)
	s.Root().AddChild(n)

	traverseScene(s)

	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	if s.commands[0].directImage == nil {
		t.Error("command should have directImage set for cached node")
	}
}

// --- Benchmarks ---

func BenchmarkMaskedNode(b *testing.B) {
	s := NewScene()
	screen := ebiten.NewImage(640, 480)

	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     64,
		Height:    64,
		OriginalW: 64,
		OriginalH: 64,
	}
	target := NewSprite("target", region)
	maskNode := NewSprite("mask", region)
	target.SetMask(maskNode)
	s.Root().AddChild(target)

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}
