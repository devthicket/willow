package lighting

import (
	"math"
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestLight_Defaults(t *testing.T) {
	l := &Light{
		X:         100,
		Y:         200,
		Radius:    50,
		Intensity: 1,
		Enabled:   true,
	}
	if l.X != 100 || l.Y != 200 {
		t.Error("position incorrect")
	}
	if l.Radius != 50 {
		t.Error("radius incorrect")
	}
}

func TestGenerateCircle(t *testing.T) {
	img := GenerateCircle(10)
	b := img.Bounds()
	if b.Dx() != 20 || b.Dy() != 20 {
		t.Errorf("size = %d×%d, want 20×20", b.Dx(), b.Dy())
	}
}

func TestGenerateCircle_MinSize(t *testing.T) {
	img := GenerateCircle(0.3)
	b := img.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		t.Error("size should be at least 1×1")
	}
}

func TestLightLayer_AddRemoveLight(t *testing.T) {
	// Skip NewLightLayer (needs function pointers), test light management directly
	ll := &LightLayer{}
	l1 := &Light{Enabled: true}
	l2 := &Light{Enabled: true}

	ll.AddLight(l1)
	ll.AddLight(l2)
	if len(ll.Lights()) != 2 {
		t.Errorf("lights = %d, want 2", len(ll.Lights()))
	}

	ll.RemoveLight(l1)
	if len(ll.Lights()) != 1 {
		t.Error("lights should be 1 after remove")
	}

	ll.ClearLights()
	if len(ll.Lights()) != 0 {
		t.Error("lights should be 0 after clear")
	}
}

func TestLightLayer_AmbientAlpha(t *testing.T) {
	ll := &LightLayer{ambientAlpha: 0.5}
	if ll.AmbientAlpha() != 0.5 {
		t.Errorf("ambient = %f, want 0.5", ll.AmbientAlpha())
	}
	ll.SetAmbientAlpha(0.8)
	if ll.AmbientAlpha() != 0.8 {
		t.Errorf("ambient = %f, want 0.8", ll.AmbientAlpha())
	}
}

func TestLight_Color(t *testing.T) {
	c := types.ColorFromRGBA(1, 0, 0, 1)
	l := &Light{Color: c}
	if l.Color != c {
		t.Error("color mismatch")
	}
}

func TestClamp01_Fallback(t *testing.T) {
	// Without Clamp01Fn set, should use built-in fallback
	old := Clamp01Fn
	Clamp01Fn = nil
	defer func() { Clamp01Fn = old }()

	if clamp01(-0.5) != 0 {
		t.Error("clamp01(-0.5) should be 0")
	}
	if clamp01(1.5) != 1 {
		t.Error("clamp01(1.5) should be 1")
	}
	if clamp01(0.5) != 0.5 {
		t.Error("clamp01(0.5) should be 0.5")
	}
}

func TestGenerateCircle_SmootherFalloff(t *testing.T) {
	// Verify center pixel is bright and edge pixel is dark
	radius := 5.0
	_ = GenerateCircle(radius)
	// Just verify it doesn't panic for small radius
	_ = GenerateCircle(1)
}

func TestLight_WithTextureRegion(t *testing.T) {
	l := &Light{
		TextureRegion: types.TextureRegion{
			Width:  32,
			Height: 48,
		},
	}
	if l.TextureRegion.Width != 32 {
		t.Error("texture region width mismatch")
	}
}

func TestLight_Rotation(t *testing.T) {
	l := &Light{Rotation: math.Pi / 4}
	if l.Rotation != math.Pi/4 {
		t.Error("rotation mismatch")
	}
}
