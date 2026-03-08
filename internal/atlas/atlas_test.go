package atlas

import (
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestNormalizePackerConfig_Defaults(t *testing.T) {
	c := NormalizePackerConfig(PackerConfig{})
	if c.PageWidth != 2048 || c.PageHeight != 2048 {
		t.Errorf("page size = %d×%d, want 2048×2048", c.PageWidth, c.PageHeight)
	}
	if c.Padding != 1 {
		t.Errorf("padding = %d, want 1", c.Padding)
	}
}

func TestNormalizePackerConfig_NoPadding(t *testing.T) {
	c := NormalizePackerConfig(PackerConfig{}.NoPadding())
	if c.Padding != 0 {
		t.Errorf("padding = %d, want 0", c.Padding)
	}
}

func TestMagentaRegion(t *testing.T) {
	r := MagentaRegion()
	if r.Page != MagentaPlaceholderPage {
		t.Errorf("page = %d, want %d", r.Page, MagentaPlaceholderPage)
	}
	if r.Width != 1 || r.Height != 1 {
		t.Errorf("size = %d×%d, want 1×1", r.Width, r.Height)
	}
}

func TestLoadAtlas_InvalidJSON(t *testing.T) {
	_, err := LoadAtlas([]byte("{bad json"), nil)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadAtlas_MissingKeys(t *testing.T) {
	_, err := LoadAtlas([]byte(`{}`), nil)
	if err == nil {
		t.Error("expected error for missing frames/textures")
	}
}

func TestLoadAtlas_HashFormat(t *testing.T) {
	jsonData := []byte(`{
		"frames": {
			"hero": {
				"frame": {"x": 10, "y": 20, "w": 32, "h": 48},
				"rotated": false,
				"trimmed": false,
				"spriteSourceSize": {"x": 0, "y": 0, "w": 32, "h": 48},
				"sourceSize": {"w": 32, "h": 48}
			}
		}
	}`)
	a, err := LoadAtlas(jsonData, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := a.Region("hero")
	if r.X != 10 || r.Y != 20 || r.Width != 32 || r.Height != 48 {
		t.Errorf("region = {X:%d Y:%d W:%d H:%d}, want {10 20 32 48}", r.X, r.Y, r.Width, r.Height)
	}
}

func TestAtlas_Has(t *testing.T) {
	a := &Atlas{Regions: map[string]types.TextureRegion{"a": {}}}
	if !a.Has("a") {
		t.Error("Has('a') should be true")
	}
	if a.Has("b") {
		t.Error("Has('b') should be false")
	}
}

func TestAtlas_RegionCount(t *testing.T) {
	a := &Atlas{Regions: map[string]types.TextureRegion{"a": {}, "b": {}}}
	if a.RegionCount() != 2 {
		t.Errorf("RegionCount = %d, want 2", a.RegionCount())
	}
}

func TestAtlas_Remove(t *testing.T) {
	a := &Atlas{Regions: map[string]types.TextureRegion{"a": {}}}
	a.Remove("a")
	if a.Has("a") {
		t.Error("Has('a') should be false after Remove")
	}
}

func TestManager_AllocPage(t *testing.T) {
	ResetGlobalManager()
	am := GlobalManager()
	idx0 := am.AllocPage()
	idx1 := am.AllocPage()
	if idx0 != 0 || idx1 != 1 {
		t.Errorf("AllocPage = %d, %d; want 0, 1", idx0, idx1)
	}
	ResetGlobalManager()
}

func TestManager_RetainRelease(t *testing.T) {
	ResetGlobalManager()
	am := GlobalManager()
	am.Retain(0)
	am.Retain(0)
	am.Release(0)
	// Should not go below zero on extra release.
	am.Release(0)
	am.Release(0)
	ResetGlobalManager()
}
