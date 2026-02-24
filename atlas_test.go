package willow

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Test JSON fixtures ---

const singlePageJSON = `{
  "frames": {
    "hero.png": {
      "frame": {"x": 0, "y": 0, "w": 64, "h": 64},
      "rotated": false,
      "trimmed": false,
      "spriteSourceSize": {"x": 0, "y": 0, "w": 64, "h": 64},
      "sourceSize": {"w": 64, "h": 64}
    },
    "enemy.png": {
      "frame": {"x": 64, "y": 0, "w": 32, "h": 48},
      "rotated": false,
      "trimmed": false,
      "spriteSourceSize": {"x": 0, "y": 0, "w": 32, "h": 48},
      "sourceSize": {"w": 32, "h": 48}
    },
    "trimmed.png": {
      "frame": {"x": 100, "y": 50, "w": 60, "h": 58},
      "rotated": false,
      "trimmed": true,
      "spriteSourceSize": {"x": 2, "y": 3, "w": 60, "h": 58},
      "sourceSize": {"w": 64, "h": 64}
    },
    "rotated.png": {
      "frame": {"x": 200, "y": 0, "w": 48, "h": 32},
      "rotated": true,
      "trimmed": false,
      "spriteSourceSize": {"x": 0, "y": 0, "w": 48, "h": 32},
      "sourceSize": {"w": 32, "h": 48}
    }
  },
  "meta": {
    "image": "atlas.png",
    "size": {"w": 1024, "h": 1024}
  }
}`

const multiPageJSON = `{
  "textures": [
    {
      "image": "atlas-0.png",
      "frames": {
        "page0_sprite.png": {
          "frame": {"x": 0, "y": 0, "w": 64, "h": 64},
          "rotated": false,
          "trimmed": false,
          "spriteSourceSize": {"x": 0, "y": 0, "w": 64, "h": 64},
          "sourceSize": {"w": 64, "h": 64}
        }
      }
    },
    {
      "image": "atlas-1.png",
      "frames": {
        "page1_sprite.png": {
          "frame": {"x": 10, "y": 20, "w": 50, "h": 50},
          "rotated": false,
          "trimmed": false,
          "spriteSourceSize": {"x": 0, "y": 0, "w": 50, "h": 50},
          "sourceSize": {"w": 50, "h": 50}
        }
      }
    }
  ]
}`

// --- LoadAtlas tests ---

func TestLoadAtlas_SinglePage_RegionCount(t *testing.T) {
	page := ebiten.NewImage(1024, 1024)
	atlas, err := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}
	if got := len(atlas.regions); got != 4 {
		t.Errorf("region count = %d, want 4", got)
	}
}

func TestLoadAtlas_RegionLookup_Exists(t *testing.T) {
	page := ebiten.NewImage(1024, 1024)
	atlas, err := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}

	r := atlas.Region("hero.png")
	if r.X != 0 || r.Y != 0 || r.Width != 64 || r.Height != 64 {
		t.Errorf("hero.png region = {X:%d Y:%d W:%d H:%d}, want {0 0 64 64}", r.X, r.Y, r.Width, r.Height)
	}
	if r.Page != 0 {
		t.Errorf("hero.png Page = %d, want 0", r.Page)
	}

	r2 := atlas.Region("enemy.png")
	if r2.X != 64 || r2.Y != 0 || r2.Width != 32 || r2.Height != 48 {
		t.Errorf("enemy.png region = {X:%d Y:%d W:%d H:%d}, want {64 0 32 48}", r2.X, r2.Y, r2.Width, r2.Height)
	}
}

func TestLoadAtlas_RegionLookup_Missing_ReturnsMagenta(t *testing.T) {
	page := ebiten.NewImage(1024, 1024)
	atlas, err := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}

	r := atlas.Region("nonexistent.png")
	if r.Page != magentaPlaceholderPage {
		t.Errorf("missing region Page = %d, want %d", r.Page, magentaPlaceholderPage)
	}
	if r.Width != 1 || r.Height != 1 {
		t.Errorf("missing region size = %dx%d, want 1x1", r.Width, r.Height)
	}
}

func TestLoadAtlas_TrimmedRegion(t *testing.T) {
	page := ebiten.NewImage(1024, 1024)
	atlas, err := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}

	r := atlas.Region("trimmed.png")
	if r.OffsetX != 2 || r.OffsetY != 3 {
		t.Errorf("trimmed OffsetX/Y = %d/%d, want 2/3", r.OffsetX, r.OffsetY)
	}
	if r.OriginalW != 64 || r.OriginalH != 64 {
		t.Errorf("trimmed OriginalW/H = %d/%d, want 64/64", r.OriginalW, r.OriginalH)
	}
	if r.Width != 60 || r.Height != 58 {
		t.Errorf("trimmed Width/Height = %d/%d, want 60/58", r.Width, r.Height)
	}
}

func TestLoadAtlas_RotatedRegion(t *testing.T) {
	page := ebiten.NewImage(1024, 1024)
	atlas, err := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}

	r := atlas.Region("rotated.png")
	if !r.Rotated {
		t.Error("rotated.png Rotated = false, want true")
	}
	// In atlas, rotated regions store w/h as the rotated dimensions
	if r.Width != 48 || r.Height != 32 {
		t.Errorf("rotated Width/Height = %d/%d, want 48/32", r.Width, r.Height)
	}
}

func TestLoadAtlas_MultiPage(t *testing.T) {
	page0 := ebiten.NewImage(512, 512)
	page1 := ebiten.NewImage(512, 512)
	atlas, err := LoadAtlas([]byte(multiPageJSON), []*ebiten.Image{page0, page1})
	if err != nil {
		t.Fatalf("LoadAtlas: %v", err)
	}

	if got := len(atlas.regions); got != 2 {
		t.Errorf("region count = %d, want 2", got)
	}

	r0 := atlas.Region("page0_sprite.png")
	if r0.Page != 0 {
		t.Errorf("page0_sprite Page = %d, want 0", r0.Page)
	}

	r1 := atlas.Region("page1_sprite.png")
	if r1.Page != 1 {
		t.Errorf("page1_sprite Page = %d, want 1", r1.Page)
	}
	if r1.X != 10 || r1.Y != 20 {
		t.Errorf("page1_sprite X/Y = %d/%d, want 10/20", r1.X, r1.Y)
	}
}

func TestLoadAtlas_InvalidJSON(t *testing.T) {
	_, err := LoadAtlas([]byte(`{invalid`), nil)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoadAtlas_NoFramesOrTextures(t *testing.T) {
	_, err := LoadAtlas([]byte(`{"meta":{}}`), nil)
	if err == nil {
		t.Error("expected error for JSON with no frames/textures, got nil")
	}
	if !strings.Contains(err.Error(), "neither") {
		t.Errorf("error message = %q, want mention of neither", err.Error())
	}
}

func TestLoadAtlas_ManualTextureRegion(t *testing.T) {
	region := TextureRegion{
		Page: 0, X: 0, Y: 0, Width: 64, Height: 64,
		OriginalW: 64, OriginalH: 64,
	}
	sprite := NewSprite("manual", region)
	if sprite.TextureRegion.Width != 64 || sprite.TextureRegion.Height != 64 {
		t.Errorf("manual region size = %dx%d, want 64x64", sprite.TextureRegion.Width, sprite.TextureRegion.Height)
	}
}

// --- Scene.LoadAtlas tests ---

func TestScene_LoadAtlas_RegistersPages(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	scene := NewScene()
	page := ebiten.NewImage(256, 256)
	_, err := scene.LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("Scene.LoadAtlas: %v", err)
	}
	// Page should be registered at index 0
	am := atlasManager()
	if am.PageCount() == 0 || am.Page(0) != page {
		t.Error("atlas page not registered at expected index")
	}
}

func TestScene_LoadAtlas_MultipleAtlases(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	scene := NewScene()
	page0 := ebiten.NewImage(256, 256)
	atlas1, err := scene.LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page0})
	if err != nil {
		t.Fatalf("Scene.LoadAtlas (first): %v", err)
	}

	page1 := ebiten.NewImage(256, 256)
	atlas2, err := scene.LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page1})
	if err != nil {
		t.Fatalf("Scene.LoadAtlas (second): %v", err)
	}

	// First atlas should use page 0, second atlas should use page 1 (offset)
	am := atlasManager()
	r1 := atlas1.Region("hero.png")
	r2 := atlas2.Region("hero.png")
	if r1.Page == r2.Page {
		t.Errorf("both atlases use same page index %d, want different", r1.Page)
	}
	if am.Page(0) != page0 {
		t.Error("first atlas page not at index 0")
	}
	if r2.Page != 1 {
		t.Errorf("second atlas region page = %d, want 1", r2.Page)
	}
	if am.Page(1) != page1 {
		t.Error("second atlas page not at index 1")
	}
}

// --- MagentaImage tests ---

func TestEnsureMagentaImage_Singleton(t *testing.T) {
	img1 := ensureMagentaImage()
	img2 := ensureMagentaImage()
	if img1 != img2 {
		t.Error("ensureMagentaImage returned different images")
	}
	w, h := img1.Bounds().Dx(), img1.Bounds().Dy()
	if w != 1 || h != 1 {
		t.Errorf("magenta image size = %dx%d, want 1x1", w, h)
	}
}

// --- Benchmarks ---

func BenchmarkLoadAtlas_SinglePage(b *testing.B) {
	data := []byte(singlePageJSON)
	page := ebiten.NewImage(1024, 1024)
	pages := []*ebiten.Image{page}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadAtlas(data, pages)
	}
}

func BenchmarkAtlas_Region_Hit(b *testing.B) {
	page := ebiten.NewImage(1024, 1024)
	atlas, _ := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atlas.Region("hero.png")
	}
}

func BenchmarkAtlas_Region_Miss(b *testing.B) {
	page := ebiten.NewImage(1024, 1024)
	atlas, _ := LoadAtlas([]byte(singlePageJSON), []*ebiten.Image{page})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = atlas.Region("nonexistent.png")
	}
}
