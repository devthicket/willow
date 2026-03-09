package willow

import (
	"fmt"
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
	if sprite.TextureRegion_.Width != 64 || sprite.TextureRegion_.Height != 64 {
		t.Errorf("manual region size = %dx%d, want 64x64", sprite.TextureRegion_.Width, sprite.TextureRegion_.Height)
	}
}

// --- Scene.LoadAtlas tests ---

func TestScene_LoadAtlas_RegistersPages(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	scene := NewScene()
	page := ebiten.NewImage(256, 256)
	_, err := LoadSceneAtlas(scene, []byte(singlePageJSON), []*ebiten.Image{page})
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
	atlas1, err := LoadSceneAtlas(scene, []byte(singlePageJSON), []*ebiten.Image{page0})
	if err != nil {
		t.Fatalf("Scene.LoadAtlas (first): %v", err)
	}

	page1 := ebiten.NewImage(256, 256)
	atlas2, err := LoadSceneAtlas(scene, []byte(singlePageJSON), []*ebiten.Image{page1})
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

// --- NewAtlas / Add tests ---

func TestNewAtlas_Add_Basic(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 48)
	r, err := atlas.Add("sprite1", img)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if r.Width != 32 || r.Height != 48 {
		t.Errorf("region size = %d×%d, want 32×48", r.Width, r.Height)
	}
	if r.OriginalW != 32 || r.OriginalH != 48 {
		t.Errorf("originalSize = %d×%d, want 32×48", r.OriginalW, r.OriginalH)
	}
	if !atlas.Has("sprite1") {
		t.Error("Has(sprite1) = false, want true")
	}
	if atlas.RegionCount() != 1 {
		t.Errorf("RegionCount = %d, want 1", atlas.RegionCount())
	}

	// Region lookup should return the same values.
	r2 := atlas.Region("sprite1")
	if r2 != r {
		t.Errorf("Region mismatch: %+v vs %+v", r2, r)
	}
}

func TestNewAtlas_Add_DuplicateIdempotent(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(16, 16)
	r1, err := atlas.Add("dup", img)
	if err != nil {
		t.Fatalf("Add 1: %v", err)
	}

	r2, err := atlas.Add("dup", img)
	if err != nil {
		t.Fatalf("Add 2: %v", err)
	}

	if r1 != r2 {
		t.Errorf("duplicate Add returned different regions: %+v vs %+v", r1, r2)
	}
	if atlas.RegionCount() != 1 {
		t.Errorf("RegionCount = %d, want 1", atlas.RegionCount())
	}
}

func TestNewAtlas_Add_MultipleSamePage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img1 := ebiten.NewImage(32, 32)
	img2 := ebiten.NewImage(64, 32)
	r1, err := atlas.Add("a", img1)
	if err != nil {
		t.Fatalf("Add a: %v", err)
	}
	r2, err := atlas.Add("b", img2)
	if err != nil {
		t.Fatalf("Add b: %v", err)
	}

	if r1.Page != r2.Page {
		t.Errorf("items on different pages: %d vs %d, want same", r1.Page, r2.Page)
	}
}

func TestNewAtlas_Add_PageOverflow(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 32, PageHeight: 32}.NoPadding())

	img1 := ebiten.NewImage(32, 32)
	img2 := ebiten.NewImage(16, 16)
	r1, err := atlas.Add("fill", img1)
	if err != nil {
		t.Fatalf("Add fill: %v", err)
	}
	r2, err := atlas.Add("overflow", img2)
	if err != nil {
		t.Fatalf("Add overflow: %v", err)
	}

	if r1.Page == r2.Page {
		t.Error("items should be on different pages after overflow")
	}
}

func TestNewAtlas_Add_AfterLoadAtlas(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	scene := NewScene()
	page := ebiten.NewImage(256, 256)
	atlas, err := LoadSceneAtlas(scene, []byte(singlePageJSON), []*ebiten.Image{page})
	if err != nil {
		t.Fatalf("Scene.LoadAtlas: %v", err)
	}

	staticCount := atlas.RegionCount()

	// Dynamically add a region to the same atlas.
	img := ebiten.NewImage(16, 16)
	r, err := atlas.Add("dynamic1", img)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if atlas.RegionCount() != staticCount+1 {
		t.Errorf("RegionCount = %d, want %d", atlas.RegionCount(), staticCount+1)
	}
	if !atlas.Has("dynamic1") {
		t.Error("Has(dynamic1) = false after Add")
	}

	// Dynamic page should be different from the static page index.
	staticRegion := atlas.Region("hero.png")
	if r.Page == staticRegion.Page {
		t.Log("dynamic region shares page with static (may happen if packer reuses, but unlikely with defaults)")
	}
}

func TestNewAtlas_Remove(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(16, 16)
	atlas.Add("removeme", img)

	if !atlas.Has("removeme") {
		t.Fatal("Has = false before Remove")
	}

	atlas.Remove("removeme")

	if atlas.Has("removeme") {
		t.Error("Has = true after Remove")
	}
	if atlas.RegionCount() != 0 {
		t.Errorf("RegionCount = %d, want 0", atlas.RegionCount())
	}
}

func TestNewAtlas_DefaultConfig(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas()

	img := ebiten.NewImage(8, 8)
	_, err := atlas.Add("tiny", img)
	if err != nil {
		t.Fatalf("Add with default config: %v", err)
	}
	if !atlas.Has("tiny") {
		t.Error("Has = false after Add with default config")
	}
}

func TestNewAtlas_DefaultConfig_HasPadding(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Default config should use padding=1. Verify by packing two items on the
	// same shelf: second item should start at width+1 (padded), not width+0.
	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256})

	img := ebiten.NewImage(10, 10)
	r1, _ := atlas.Add("a", img)
	r2, _ := atlas.Add("b", img)

	// With padding=1, padded width is 11. Second item at x=11.
	if r1.X != 0 {
		t.Errorf("r1.X = %d, want 0", r1.X)
	}
	if r2.X != 11 {
		t.Errorf("r2.X = %d, want 11 (10 + 1 padding)", r2.X)
	}
}

func TestNewAtlas_Add_1x1Edge(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256})
	img := ebiten.NewImage(1, 1)
	_, err := atlas.Add("1x1", img)
	if err != nil {
		t.Fatalf("unexpected error for 1×1 image: %v", err)
	}
}

func TestNewAtlas_Add_NilImageError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas()
	_, err := atlas.Add("nil", nil)
	if err == nil {
		t.Error("expected error for nil image")
	}
}

func TestNewAtlas_Add_OversizedError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 32, PageHeight: 32, Padding: 1})
	img := ebiten.NewImage(32, 32) // 32+1=33 > 32
	_, err := atlas.Add("toobig", img)
	if err == nil {
		t.Error("expected error for oversized image with padding")
	}
}

func TestNewAtlas_Add_ManyItems(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256, Padding: 1})

	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("item_%d", i)
		w := 8 + (i % 12)
		h := 8 + ((i * 3) % 10)
		img := ebiten.NewImage(w, h)
		_, err := atlas.Add(name, img)
		if err != nil {
			t.Fatalf("Add %s (%d×%d): %v", name, w, h, err)
		}
	}

	if atlas.RegionCount() != 50 {
		t.Errorf("RegionCount = %d, want 50", atlas.RegionCount())
	}
}

func TestNewAtlas_Add_SyncsPages(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 64, PageHeight: 64}.NoPadding())

	img := ebiten.NewImage(16, 16)
	r, err := atlas.Add("sync", img)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	pageIdx := int(r.Page)
	if pageIdx >= len(atlas.Pages) {
		t.Fatalf("atlas.Pages length %d, expected at least %d", len(atlas.Pages), pageIdx+1)
	}
	if atlas.Pages[pageIdx] == nil {
		t.Error("atlas.Pages[pageIdx] is nil after Add")
	}

	// Verify it matches AtlasManager.
	am := atlasManager()
	if atlas.Pages[pageIdx] != am.Page(pageIdx) {
		t.Error("atlas.Pages and AtlasManager disagree on page image")
	}
}

// --- Free tests ---

func TestNewAtlas_Free_Basic(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	atlas.Add("a", img)

	if !atlas.Has("a") {
		t.Fatal("Has(a) = false before Free")
	}

	err := atlas.Free("a")
	if err != nil {
		t.Fatalf("Free: %v", err)
	}

	if atlas.Has("a") {
		t.Error("Has(a) = true after Free")
	}
	if atlas.RegionCount() != 0 {
		t.Errorf("RegionCount = %d, want 0", atlas.RegionCount())
	}
}

func TestNewAtlas_Free_SpaceReused(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	r1, _ := atlas.Add("a", img)

	atlas.Free("a")

	img2 := ebiten.NewImage(32, 32)
	r2, err := atlas.Add("b", img2)
	if err != nil {
		t.Fatalf("Add after Free: %v", err)
	}

	// New item should reuse the freed space (same position).
	if r2.X != r1.X || r2.Y != r1.Y {
		t.Errorf("reused position = (%d,%d), want (%d,%d)", r2.X, r2.Y, r1.X, r1.Y)
	}
}

func TestNewAtlas_Free_NotFound(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas()
	err := atlas.Free("nonexistent")
	if err == nil {
		t.Error("expected error for Free of nonexistent name")
	}
}

func TestNewAtlas_Free_BatchError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	err := atlas.Free("anything")
	if err == nil {
		t.Error("expected error for Free on batch atlas")
	}
}

// --- Replace tests ---

func TestNewAtlas_Replace_SameSize(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	r1, _ := atlas.Add("a", img)

	newImg := ebiten.NewImage(32, 32)
	err := atlas.Replace("a", newImg)
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	// Region should be unchanged.
	r2 := atlas.Region("a")
	if r2 != r1 {
		t.Errorf("region changed after Replace: %+v vs %+v", r2, r1)
	}
}

func TestNewAtlas_Replace_WrongSize(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	atlas.Add("a", img)

	wrongSize := ebiten.NewImage(64, 64)
	err := atlas.Replace("a", wrongSize)
	if err == nil {
		t.Error("expected error for Replace with different size")
	}
}

func TestNewAtlas_Replace_NotFound(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas()
	err := atlas.Replace("nonexistent", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Replace of nonexistent name")
	}
}

func TestNewAtlas_Replace_BatchError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	err := atlas.Replace("anything", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Replace on batch atlas")
	}
}

// --- Update tests ---

func TestNewAtlas_Update_SameSize(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	r1, _ := atlas.Add("a", img)

	newImg := ebiten.NewImage(32, 32)
	r2, err := atlas.Update("a", newImg)
	if err != nil {
		t.Fatalf("Update same size: %v", err)
	}

	// Same size → acts like Replace, region unchanged.
	if r2 != r1 {
		t.Errorf("region changed after same-size Update: %+v vs %+v", r2, r1)
	}
}

func TestNewAtlas_Update_DifferentSize(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img := ebiten.NewImage(32, 32)
	r1, _ := atlas.Add("a", img)

	newImg := ebiten.NewImage(64, 64)
	r2, err := atlas.Update("a", newImg)
	if err != nil {
		t.Fatalf("Update different size: %v", err)
	}

	// Different size → Free + Add, may have new region.
	if r2.Width != 64 || r2.Height != 64 {
		t.Errorf("new region size = %d×%d, want 64×64", r2.Width, r2.Height)
	}
	_ = r1
}

func TestNewAtlas_Update_BatchError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	_, err := atlas.Update("anything", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Update on batch atlas")
	}
}

// --- Batch mode tests ---

func TestBatchAtlas_StageAndPack(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	img1 := ebiten.NewImage(32, 48)
	img2 := ebiten.NewImage(64, 32)
	img3 := ebiten.NewImage(16, 16)

	if err := atlas.Stage("a", img1); err != nil {
		t.Fatalf("Stage a: %v", err)
	}
	if err := atlas.Stage("b", img2); err != nil {
		t.Fatalf("Stage b: %v", err)
	}
	if err := atlas.Stage("c", img3); err != nil {
		t.Fatalf("Stage c: %v", err)
	}

	if err := atlas.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if atlas.RegionCount() != 3 {
		t.Errorf("RegionCount = %d, want 3", atlas.RegionCount())
	}
	if !atlas.Has("a") || !atlas.Has("b") || !atlas.Has("c") {
		t.Error("missing regions after Pack")
	}

	r := atlas.Region("a")
	if r.Width != 32 || r.Height != 48 {
		t.Errorf("region a size = %d×%d, want 32×48", r.Width, r.Height)
	}
}

func TestBatchAtlas_StageAfterPack(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas(PackerConfig{PageWidth: 256, PageHeight: 256}.NoPadding())

	atlas.Stage("a", ebiten.NewImage(8, 8))
	atlas.Pack()

	err := atlas.Stage("b", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Stage after Pack")
	}
}

func TestBatchAtlas_PackEmpty(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	err := atlas.Pack()
	if err != nil {
		t.Fatalf("Pack empty: %v", err)
	}
}

func TestBatchAtlas_PackTwice(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	atlas.Pack()
	err := atlas.Pack()
	if err == nil {
		t.Error("expected error for double Pack")
	}
}

func TestBatchAtlas_MultiPage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Tiny page: 32x32. Each 32x32 image fills an entire page.
	atlas := NewBatchAtlas(PackerConfig{PageWidth: 32, PageHeight: 32}.NoPadding())

	atlas.Stage("a", ebiten.NewImage(32, 32))
	atlas.Stage("b", ebiten.NewImage(32, 32))

	if err := atlas.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	r1 := atlas.Region("a")
	r2 := atlas.Region("b")
	if r1.Page == r2.Page {
		t.Error("expected items on different pages")
	}
}

func TestBatchAtlas_AddErrors(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()

	// Add on batch atlas should error.
	_, err := atlas.Add("a", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Add on batch atlas")
	}
}

func TestBatchAtlas_StageOnShelfErrors(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewAtlas()

	err := atlas.Stage("a", ebiten.NewImage(8, 8))
	if err == nil {
		t.Error("expected error for Stage on shelf atlas")
	}
}

func TestBatchAtlas_StageDuplicate(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()

	atlas.Stage("a", ebiten.NewImage(8, 8))
	err := atlas.Stage("a", ebiten.NewImage(16, 16))
	if err == nil {
		t.Error("expected error for duplicate Stage name")
	}
}

func TestBatchAtlas_StageNilImage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas()
	err := atlas.Stage("nil", nil)
	if err == nil {
		t.Error("expected error for nil image")
	}
}

func TestBatchAtlas_PackRegionsCorrect(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas(PackerConfig{PageWidth: 512, PageHeight: 512}.NoPadding())

	sizes := map[string][2]int{
		"small":  {16, 16},
		"medium": {64, 48},
		"large":  {128, 128},
	}

	for name, sz := range sizes {
		atlas.Stage(name, ebiten.NewImage(sz[0], sz[1]))
	}

	if err := atlas.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	for name, sz := range sizes {
		r := atlas.Region(name)
		if r.Width != uint16(sz[0]) || r.Height != uint16(sz[1]) {
			t.Errorf("region %q size = %d×%d, want %d×%d", name, r.Width, r.Height, sz[0], sz[1])
		}
		if r.Page == magentaPlaceholderPage {
			t.Errorf("region %q is magenta placeholder", name)
		}
	}
}

func TestBatchAtlas_PagesSliceSynced(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	atlas := NewBatchAtlas(PackerConfig{PageWidth: 64, PageHeight: 64}.NoPadding())
	atlas.Stage("a", ebiten.NewImage(16, 16))
	atlas.Pack()

	r := atlas.Region("a")
	pageIdx := int(r.Page)
	if pageIdx >= len(atlas.Pages) {
		t.Fatalf("Pages length %d, expected at least %d", len(atlas.Pages), pageIdx+1)
	}
	if atlas.Pages[pageIdx] == nil {
		t.Error("atlas.Pages[pageIdx] is nil after Pack")
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
