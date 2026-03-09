package willow

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestBatchKeySameAtlasSameBlend(t *testing.T) {
	a := RenderCommand{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}}
	b := RenderCommand{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}}
	if commandBatchKey(&a) != commandBatchKey(&b) {
		t.Error("same atlas + same blend should produce same batch key")
	}
}

func TestBatchKeyDifferentBlend(t *testing.T) {
	a := RenderCommand{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}}
	b := RenderCommand{BlendMode: BlendAdd, TextureRegion: TextureRegion{Page: 0}}
	if commandBatchKey(&a) == commandBatchKey(&b) {
		t.Error("different blend modes should produce different batch keys")
	}
}

func TestBatchKeyDifferentPage(t *testing.T) {
	a := RenderCommand{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}}
	b := RenderCommand{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 1}}
	if commandBatchKey(&a) == commandBatchKey(&b) {
		t.Error("different pages should produce different batch keys")
	}
}

func TestBatchKeyDifferentShader(t *testing.T) {
	a := RenderCommand{ShaderID: 0}
	b := RenderCommand{ShaderID: 1}
	if commandBatchKey(&a) == commandBatchKey(&b) {
		t.Error("different shaders should produce different batch keys")
	}
}

func TestBatchKeyDifferentTarget(t *testing.T) {
	a := RenderCommand{TargetID: 0}
	b := RenderCommand{TargetID: 1}
	if commandBatchKey(&a) == commandBatchKey(&b) {
		t.Error("different targets should produce different batch keys")
	}
}

func TestBatchCountSameAtlas(t *testing.T) {
	cmds := []RenderCommand{
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
	}
	if got := countBatches(cmds); got != 1 {
		t.Errorf("batches = %d, want 1", got)
	}
}

func TestBatchCountDifferentBlends(t *testing.T) {
	cmds := []RenderCommand{
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{BlendMode: BlendAdd, TextureRegion: TextureRegion{Page: 0}},
	}
	if got := countBatches(cmds); got != 2 {
		t.Errorf("batches = %d, want 2", got)
	}
}

func TestBatchCountDifferentPages(t *testing.T) {
	cmds := []RenderCommand{
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 1}},
	}
	if got := countBatches(cmds); got != 2 {
		t.Errorf("batches = %d, want 2", got)
	}
}

func TestBatchCountEmpty(t *testing.T) {
	if got := countBatches(nil); got != 0 {
		t.Errorf("batches = %d, want 0", got)
	}
}

// --- Coalesced batching tests ---

func assertVertexNear(t *testing.T, label string, got, want float32) {
	t.Helper()
	if diff := got - want; diff > 0.001 || diff < -0.001 {
		t.Errorf("%s = %f, want %f", label, got, want)
	}
}

func TestAppendSpriteQuad_NonRotated(t *testing.T) {
	s := NewScene()
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, X: 10, Y: 20, Width: 32, Height: 16,
			OriginalW: 32, OriginalH: 16,
		},
		Transform: identityTransform32,
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	if len(s.Pipeline.BatchVerts) != 4 {
		t.Fatalf("verts = %d, want 4", len(s.Pipeline.BatchVerts))
	}
	if len(s.Pipeline.BatchInds) != 6 {
		t.Fatalf("inds = %d, want 6", len(s.Pipeline.BatchInds))
	}

	// TL should be at (0,0) with src (10,20)
	assertVertexNear(t, "TL.DstX", s.Pipeline.BatchVerts[0].DstX, 0)
	assertVertexNear(t, "TL.DstY", s.Pipeline.BatchVerts[0].DstY, 0)
	assertVertexNear(t, "TL.SrcX", s.Pipeline.BatchVerts[0].SrcX, 10)
	assertVertexNear(t, "TL.SrcY", s.Pipeline.BatchVerts[0].SrcY, 20)

	// TR should be at (32,0) with src (42,20)
	assertVertexNear(t, "TR.DstX", s.Pipeline.BatchVerts[1].DstX, 32)
	assertVertexNear(t, "TR.DstY", s.Pipeline.BatchVerts[1].DstY, 0)
	assertVertexNear(t, "TR.SrcX", s.Pipeline.BatchVerts[1].SrcX, 42)
	assertVertexNear(t, "TR.SrcY", s.Pipeline.BatchVerts[1].SrcY, 20)

	// BL should be at (0,16) with src (10,36)
	assertVertexNear(t, "BL.DstX", s.Pipeline.BatchVerts[2].DstX, 0)
	assertVertexNear(t, "BL.DstY", s.Pipeline.BatchVerts[2].DstY, 16)
	assertVertexNear(t, "BL.SrcX", s.Pipeline.BatchVerts[2].SrcX, 10)
	assertVertexNear(t, "BL.SrcY", s.Pipeline.BatchVerts[2].SrcY, 36)

	// BR should be at (32,16) with src (42,36)
	assertVertexNear(t, "BR.DstX", s.Pipeline.BatchVerts[3].DstX, 32)
	assertVertexNear(t, "BR.DstY", s.Pipeline.BatchVerts[3].DstY, 16)
	assertVertexNear(t, "BR.SrcX", s.Pipeline.BatchVerts[3].SrcX, 42)
	assertVertexNear(t, "BR.SrcY", s.Pipeline.BatchVerts[3].SrcY, 36)

	// Check index winding
	wantInds := []uint32{0, 1, 2, 1, 3, 2}
	for i, w := range wantInds {
		if s.Pipeline.BatchInds[i] != w {
			t.Errorf("ind[%d] = %d, want %d", i, s.Pipeline.BatchInds[i], w)
		}
	}
}

func TestAppendSpriteQuad_Rotated(t *testing.T) {
	s := NewScene()
	// A 32×16 sprite stored rotated in atlas at (10,20).
	// Stored rect is Width×Height with the names referring to the *original*
	// dimensions, so the atlas rect is (r.Height × r.Width) = (16 × 32).
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, X: 10, Y: 20,
			Width: 32, Height: 16,
			OriginalW: 32, OriginalH: 16,
			Rotated: true,
		},
		Transform: identityTransform32,
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	if len(s.Pipeline.BatchVerts) != 4 {
		t.Fatalf("verts = %d, want 4", len(s.Pipeline.BatchVerts))
	}

	// Visual dimensions should be the same (32×16), placed at identity.
	assertVertexNear(t, "TL.DstX", s.Pipeline.BatchVerts[0].DstX, 0)
	assertVertexNear(t, "TL.DstY", s.Pipeline.BatchVerts[0].DstY, 0)
	assertVertexNear(t, "TR.DstX", s.Pipeline.BatchVerts[1].DstX, 32)
	assertVertexNear(t, "BR.DstY", s.Pipeline.BatchVerts[3].DstY, 16)

	// Rotated UVs: TL → atlas (X+Height, Y) = (26, 20)
	assertVertexNear(t, "TL.SrcX", s.Pipeline.BatchVerts[0].SrcX, 26)
	assertVertexNear(t, "TL.SrcY", s.Pipeline.BatchVerts[0].SrcY, 20)
	// TR → atlas (X+Height, Y+Width) = (26, 52)
	assertVertexNear(t, "TR.SrcX", s.Pipeline.BatchVerts[1].SrcX, 26)
	assertVertexNear(t, "TR.SrcY", s.Pipeline.BatchVerts[1].SrcY, 52)
	// BL → atlas (X, Y) = (10, 20)
	assertVertexNear(t, "BL.SrcX", s.Pipeline.BatchVerts[2].SrcX, 10)
	assertVertexNear(t, "BL.SrcY", s.Pipeline.BatchVerts[2].SrcY, 20)
	// BR → atlas (X, Y+Width) = (10, 52)
	assertVertexNear(t, "BR.SrcX", s.Pipeline.BatchVerts[3].SrcX, 10)
	assertVertexNear(t, "BR.SrcY", s.Pipeline.BatchVerts[3].SrcY, 52)
}

func TestAppendSpriteQuad_TrimOffset(t *testing.T) {
	s := NewScene()
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, X: 0, Y: 0, Width: 10, Height: 10,
			OriginalW: 20, OriginalH: 20,
			OffsetX: 5, OffsetY: 3,
		},
		Transform: identityTransform32,
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	// TL should be offset by (5, 3)
	assertVertexNear(t, "TL.DstX", s.Pipeline.BatchVerts[0].DstX, 5)
	assertVertexNear(t, "TL.DstY", s.Pipeline.BatchVerts[0].DstY, 3)
	// BR at (5+10, 3+10) = (15, 13)
	assertVertexNear(t, "BR.DstX", s.Pipeline.BatchVerts[3].DstX, 15)
	assertVertexNear(t, "BR.DstY", s.Pipeline.BatchVerts[3].DstY, 13)
}

func TestAppendSpriteQuad_ZeroColor(t *testing.T) {
	s := NewScene()
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, Width: 10, Height: 10, OriginalW: 10, OriginalH: 10,
		},
		Transform: identityTransform32,
		Color:     color32{R: 0, G: 0, B: 0, A: 0}, // zero sentinel
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	// Should be treated as opaque white
	assertVertexNear(t, "ColorR", s.Pipeline.BatchVerts[0].ColorR, 1)
	assertVertexNear(t, "ColorG", s.Pipeline.BatchVerts[0].ColorG, 1)
	assertVertexNear(t, "ColorB", s.Pipeline.BatchVerts[0].ColorB, 1)
	assertVertexNear(t, "ColorA", s.Pipeline.BatchVerts[0].ColorA, 1)
}

func TestAppendSpriteQuad_PremultipliedColor(t *testing.T) {
	s := NewScene()
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, Width: 10, Height: 10, OriginalW: 10, OriginalH: 10,
		},
		Transform: identityTransform32,
		Color:     color32{R: 1.0, G: 0.5, B: 0.25, A: 0.5},
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	// Premultiplied: R*A, G*A, B*A, A
	assertVertexNear(t, "ColorR", s.Pipeline.BatchVerts[0].ColorR, 0.5)
	assertVertexNear(t, "ColorG", s.Pipeline.BatchVerts[0].ColorG, 0.25)
	assertVertexNear(t, "ColorB", s.Pipeline.BatchVerts[0].ColorB, 0.125)
	assertVertexNear(t, "ColorA", s.Pipeline.BatchVerts[0].ColorA, 0.5)
}

func TestAppendSpriteQuad_Transform(t *testing.T) {
	s := NewScene()
	// 2x scale transform: [2, 0, 0, 2, 100, 200]
	cmd := &RenderCommand{
		Type: CommandSprite,
		TextureRegion: TextureRegion{
			Page: 0, Width: 10, Height: 10, OriginalW: 10, OriginalH: 10,
		},
		Transform: [6]float32{2, 0, 0, 2, 100, 200},
	}
	s.Pipeline.AppendSpriteQuad(cmd)

	// TL at (100, 200)
	assertVertexNear(t, "TL.DstX", s.Pipeline.BatchVerts[0].DstX, 100)
	assertVertexNear(t, "TL.DstY", s.Pipeline.BatchVerts[0].DstY, 200)
	// BR at (2*10+100, 2*10+200) = (120, 220)
	assertVertexNear(t, "BR.DstX", s.Pipeline.BatchVerts[3].DstX, 120)
	assertVertexNear(t, "BR.DstY", s.Pipeline.BatchVerts[3].DstY, 220)
}

func TestCoalescedBatchCount(t *testing.T) {
	cmds := []RenderCommand{
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
	}
	// 3 same-key sprites → 1 draw call in coalesced mode
	if got := countDrawCallsCoalesced(cmds); got != 1 {
		t.Errorf("coalesced draw calls = %d, want 1", got)
	}
}

func TestCoalescedBatchCountKeyChange(t *testing.T) {
	cmds := []RenderCommand{
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendAdd, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendAdd, TextureRegion: TextureRegion{Page: 0}},
	}
	// 2 runs → 2 draw calls
	if got := countDrawCallsCoalesced(cmds); got != 2 {
		t.Errorf("coalesced draw calls = %d, want 2", got)
	}
}

func TestCoalescedDirectImageFallback(t *testing.T) {
	directImg := ebiten.NewImage(1, 1)
	cmds := []RenderCommand{
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}, DirectImage: directImg},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
	}
	// Run 1 (atlas), direct image, run 2 (atlas) → 3 draw calls
	if got := countDrawCallsCoalesced(cmds); got != 3 {
		t.Errorf("coalesced draw calls = %d, want 3", got)
	}
}

func TestCoalescedParticleCount(t *testing.T) {
	e := &ParticleEmitter{Alive: 50}
	cmds := []RenderCommand{
		{Type: CommandParticle, Emitter: e, BlendMode: BlendNormal},
	}
	// 1 emitter → 1 draw call
	if got := countDrawCallsCoalesced(cmds); got != 1 {
		t.Errorf("coalesced draw calls = %d, want 1", got)
	}
}

func TestCoalescedMixed(t *testing.T) {
	e := &ParticleEmitter{Alive: 10}
	cmds := []RenderCommand{
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
		{Type: CommandParticle, Emitter: e, BlendMode: BlendNormal},
		{Type: CommandSprite, BlendMode: BlendNormal, TextureRegion: TextureRegion{Page: 0}},
	}
	// sprite run (1) + particle (1) + sprite run (1) = 3
	if got := countDrawCallsCoalesced(cmds); got != 3 {
		t.Errorf("coalesced draw calls = %d, want 3", got)
	}
}

func TestSubmitBatchesCoalesced_Integration(t *testing.T) {
	// Integration test: verify submitBatchesCoalesced runs without panic
	// on a scene with magenta placeholder sprites.
	s := NewScene()
	s.SetBatchMode(BatchModeCoalesced)
	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     32,
		Height:    32,
		OriginalW: 32,
		OriginalH: 32,
	}
	for i := 0; i < 100; i++ {
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%10) * 40
		sp.Y_ = float64(i/10) * 40
		s.Root.AddChild(sp)
	}
	screen := ebiten.NewImage(640, 480)
	s.Draw(screen)

	// Verify batch mode getter
	if s.GetBatchMode() != BatchModeCoalesced {
		t.Error("GetBatchMode should return BatchModeCoalesced")
	}
}

func TestSubmitBatchesCoalesced_Rotated(t *testing.T) {
	s := NewScene()
	s.SetBatchMode(BatchModeCoalesced)
	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     32,
		Height:    16,
		OriginalW: 32,
		OriginalH: 16,
		Rotated:   true,
	}
	sp := NewSprite("sp", region)
	s.Root.AddChild(sp)
	screen := ebiten.NewImage(640, 480)
	// Should not panic
	s.Draw(screen)
}

func TestSubmitParticlesBatched_Integration(t *testing.T) {
	s := NewScene()
	s.SetBatchMode(BatchModeCoalesced)

	cfg := EmitterConfig{
		MaxParticles: 100,
		EmitRate:     100000,
		Lifetime:     Range{Min: 10, Max: 10},
		Speed:        Range{Min: 10, Max: 50},
		Angle:        Range{Min: 0, Max: 2 * math.Pi},
		StartScale:   Range{Min: 1, Max: 1},
		EndScale:     Range{Min: 0.1, Max: 0.1},
		StartAlpha:   Range{Min: 1, Max: 1},
		EndAlpha:     Range{Min: 0, Max: 0},
		StartColor:   RGBA(1, 1, 1, 1),
		EndColor:     RGBA(1, 0, 0, 1),
		Region: TextureRegion{
			Page:      magentaPlaceholderPage,
			Width:     8,
			Height:    8,
			OriginalW: 8,
			OriginalH: 8,
		},
	}
	emitterNode := NewParticleEmitter("particles", cfg)
	emitterNode.Emitter.Start()
	// Fill particles
	for emitterNode.Emitter.Alive < 50 {
		emitterNode.Emitter.Update(1.0 / 60.0)
	}
	s.Root.AddChild(emitterNode)

	screen := ebiten.NewImage(640, 480)
	// Should not panic
	s.Draw(screen)
}
