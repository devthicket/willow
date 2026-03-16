package integration

import (
	"testing"

	. "github.com/devthicket/willow"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewRenderTextureDimensions(t *testing.T) {
	rt := NewRenderTexture(128, 64)
	defer rt.Dispose()

	if rt.Width() != 128 {
		t.Errorf("Width = %d, want 128", rt.Width())
	}
	if rt.Height() != 64 {
		t.Errorf("Height = %d, want 64", rt.Height())
	}
	if rt.Image() == nil {
		t.Error("Image() should not be nil")
	}
}

func TestNewSpriteNodeSetsCustomImage(t *testing.T) {
	rt := NewRenderTexture(32, 32)
	defer rt.Dispose()

	node := rt.NewSpriteNode("test")
	if node.CustomImage_ != rt.Image() {
		t.Error("NewSpriteNode should set customImage to rt.Image()")
	}
	if node.Type != NodeTypeSprite {
		t.Errorf("Type = %d, want NodeTypeSprite", node.Type)
	}
}

func TestSetCustomImageAccessorRoundTrip(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	defer img.Deallocate()

	n := NewSprite("s", TextureRegion{})
	if n.CustomImage() != WhitePixel {
		t.Error("CustomImage should be WhitePixel initially for zero TextureRegion")
	}

	n.SetCustomImage(img)
	if n.CustomImage() != img {
		t.Error("CustomImage should return the image set by SetCustomImage")
	}

	n.SetCustomImage(nil)
	if n.CustomImage() != nil {
		t.Error("CustomImage should be nil after setting nil")
	}
}

func TestDisposeNilsCustomImage(t *testing.T) {
	img := ebiten.NewImage(16, 16)
	n := NewSprite("s", TextureRegion{})
	n.SetCustomImage(img)

	n.Dispose()

	if n.CustomImage_ != nil {
		t.Error("customImage should be nil after Dispose")
	}
}

func TestCustomImageSpriteEmitsDirectImage(t *testing.T) {
	s := NewScene()
	img := ebiten.NewImage(32, 32)
	defer img.Deallocate()

	sprite := NewSprite("s", TextureRegion{Width: 10, Height: 10})
	sprite.SetCustomImage(img)
	s.Root.AddChild(sprite)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	cmd := s.Pipeline.Commands[0]
	if cmd.DirectImage != img {
		t.Error("directImage should be the custom image")
	}
	if cmd.TextureRegion.Width != 0 || cmd.TextureRegion.Height != 0 {
		t.Error("TextureRegion should be zero when customImage is set")
	}
}

func TestNodeDimensionsCustomImage(t *testing.T) {
	img := ebiten.NewImage(64, 48)
	defer img.Deallocate()

	n := NewSprite("s", TextureRegion{OriginalW: 10, OriginalH: 10})
	n.SetCustomImage(img)

	w, h := nodeDimensions(n)
	if w != 64 || h != 48 {
		t.Errorf("nodeDimensions = (%v, %v), want (64, 48)", w, h)
	}
}

func TestRegularSpriteNoDirectImage(t *testing.T) {
	s := NewScene()
	region := TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32}
	sprite := NewSprite("s", region)
	s.Root.AddChild(sprite)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	cmd := s.Pipeline.Commands[0]
	if cmd.DirectImage != nil {
		t.Error("directImage should be nil for regular sprites")
	}
	if cmd.TextureRegion.Width != 32 {
		t.Errorf("TextureRegion.Width = %d, want 32", cmd.TextureRegion.Width)
	}
}

func TestCustomImageInEmitNodeCommand(t *testing.T) {
	s := NewScene()
	img := ebiten.NewImage(24, 24)
	defer img.Deallocate()

	n := NewSprite("s", TextureRegion{Width: 10, Height: 10})
	n.SetCustomImage(img)

	treeOrder := 0
	render.EmitNodeCommand(&s.Pipeline, n, node.IdentityTransform, 1.0, &treeOrder)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].DirectImage != img {
		t.Error("emitNodeCommand should set directImage for customImage nodes")
	}
}

func TestRenderTextureClearAndFill(t *testing.T) {
	rt := NewRenderTexture(4, 4)
	defer rt.Dispose()

	// Should not panic.
	rt.Clear()
	rt.Fill(RGBA(1, 0, 0, 1))
}

func TestRenderTextureDrawImageAt(t *testing.T) {
	rt := NewRenderTexture(32, 32)
	defer rt.Dispose()

	src := ebiten.NewImage(8, 8)
	defer src.Deallocate()

	// Should not panic.
	rt.DrawImageAt(src, 10, 10, BlendNormal)
}

func TestRenderTextureDispose(t *testing.T) {
	rt := NewRenderTexture(16, 16)
	rt.Dispose()

	if rt.Image() != nil {
		t.Error("Image() should be nil after Dispose")
	}

	// Double dispose should not panic.
	rt.Dispose()
}

func TestRenderTextureDrawSprite(t *testing.T) {
	rt := NewRenderTexture(64, 64)
	defer rt.Dispose()

	page := ebiten.NewImage(32, 32)
	defer page.Deallocate()

	region := TextureRegion{Page: 0, X: 0, Y: 0, Width: 16, Height: 16, OriginalW: 16, OriginalH: 16}
	// Should not panic.
	rt.DrawSprite(region, 10, 10, BlendNormal, []*ebiten.Image{page})
}

func TestRenderTextureDrawSpriteMagenta(t *testing.T) {
	rt := NewRenderTexture(64, 64)
	defer rt.Dispose()

	region := TextureRegion{Page: magentaPlaceholderPage, Width: 1, Height: 1}
	// Should not panic  -  resolves to magenta placeholder.
	rt.DrawSprite(region, 0, 0, BlendNormal, nil)
}

func TestRenderTextureDrawSpriteColored(t *testing.T) {
	rt := NewRenderTexture(64, 64)
	defer rt.Dispose()

	page := ebiten.NewImage(32, 32)
	defer page.Deallocate()

	region := TextureRegion{Page: 0, X: 0, Y: 0, Width: 16, Height: 16, OriginalW: 16, OriginalH: 16}
	opts := RenderTextureDrawOpts{
		X: 5, Y: 5,
		ScaleX: 2, ScaleY: 2,
		Color: RGBA(1, 0, 0, 1),
		Alpha: 0.5,
	}
	// Should not panic.
	rt.DrawSpriteColored(region, opts, []*ebiten.Image{page})
}

func TestRenderTextureDrawImageColored(t *testing.T) {
	rt := NewRenderTexture(64, 64)
	defer rt.Dispose()

	src := ebiten.NewImage(16, 16)
	defer src.Deallocate()

	opts := RenderTextureDrawOpts{
		X: 10, Y: 10,
		Rotation: 0.5,
		PivotX:   8, PivotY: 8,
		Color: ColorWhite,
		Alpha: 1,
	}
	// Should not panic.
	rt.DrawImageColored(src, opts)
}

func TestRenderTextureResize(t *testing.T) {
	rt := NewRenderTexture(32, 32)
	defer rt.Dispose()

	if rt.Width() != 32 || rt.Height() != 32 {
		t.Fatalf("initial size = %dx%d, want 32x32", rt.Width(), rt.Height())
	}

	rt.Resize(128, 64)
	if rt.Width() != 128 || rt.Height() != 64 {
		t.Errorf("after Resize: size = %dx%d, want 128x64", rt.Width(), rt.Height())
	}
	if rt.Image() == nil {
		t.Error("Image() should not be nil after Resize")
	}
	b := rt.Image().Bounds()
	if b.Dx() != 128 || b.Dy() != 64 {
		t.Errorf("image bounds = %dx%d, want 128x64", b.Dx(), b.Dy())
	}
}

func TestRenderTextureDrawOptsDefaults(t *testing.T) {
	rt := NewRenderTexture(64, 64)
	defer rt.Dispose()

	src := ebiten.NewImage(8, 8)
	defer src.Deallocate()

	// Zero-value opts: ScaleX/ScaleY should default to 1, Color to white, Alpha to 1.
	rt.DrawImageColored(src, RenderTextureDrawOpts{})
}

func TestResolvePageImageOutOfRange(t *testing.T) {
	region := TextureRegion{Page: 5}
	result := resolvePageImage(region, []*ebiten.Image{ebiten.NewImage(1, 1)})
	if result != nil {
		t.Error("resolvePageImage should return nil for out-of-range page index")
	}
}

// --- Benchmarks ---

func BenchmarkRenderTextureClear(b *testing.B) {
	rt := NewRenderTexture(256, 256)
	defer rt.Dispose()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.Clear()
	}
}

func BenchmarkRenderTextureDrawImage(b *testing.B) {
	rt := NewRenderTexture(256, 256)
	defer rt.Dispose()

	src := ebiten.NewImage(64, 64)
	defer src.Deallocate()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.DrawImageAt(src, 10, 10, BlendNormal)
	}
}
