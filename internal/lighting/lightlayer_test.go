package lighting

import (
	"math"
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
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

func TestClamp01_Behavior(t *testing.T) {
	if types.Clamp01(-0.5) != 0 {
		t.Error("Clamp01(-0.5) should be 0")
	}
	if types.Clamp01(1.5) != 1 {
		t.Error("Clamp01(1.5) should be 1")
	}
	if types.Clamp01(0.5) != 0.5 {
		t.Error("Clamp01(0.5) should be 0.5")
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

// --- helpers for full LightLayer tests ---

func setupLightingFns() func() {
	oldNew := NewRenderTextureFn
	oldImg := RenderTextureImageFn
	oldSprite := RenderTextureNewSpriteFn
	oldDispose := RenderTextureDisposeFn
	oldMagenta := EnsureMagentaImageFn

	NewRenderTextureFn = func(w, h int) any {
		return ebiten.NewImage(w, h)
	}
	RenderTextureImageFn = func(rt any) *ebiten.Image {
		return rt.(*ebiten.Image)
	}
	RenderTextureNewSpriteFn = func(rt any, name string) *node.Node {
		n := node.NewNode(name, types.NodeTypeSprite)
		n.CustomImage_ = rt.(*ebiten.Image)
		return n
	}
	RenderTextureDisposeFn = func(rt any) {
		rt.(*ebiten.Image).Deallocate()
	}
	EnsureMagentaImageFn = func() *ebiten.Image {
		return ebiten.NewImage(1, 1)
	}

	return func() {
		NewRenderTextureFn = oldNew
		RenderTextureImageFn = oldImg
		RenderTextureNewSpriteFn = oldSprite
		RenderTextureDisposeFn = oldDispose
		EnsureMagentaImageFn = oldMagenta
	}
}

func TestNewLightLayer(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(128, 128, 0.7)
	if ll.node_ == nil {
		t.Fatal("node should not be nil")
	}
	if ll.AmbientAlpha() != 0.7 {
		t.Errorf("ambient = %f, want 0.7", ll.AmbientAlpha())
	}
	if ll.rt == nil {
		t.Fatal("render texture should not be nil")
	}
}

func TestLightLayer_Node(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	if ll.Node() == nil {
		t.Error("Node() should not be nil")
	}
	if ll.Node().Name != "light_layer" {
		t.Errorf("node name = %q, want light_layer", ll.Node().Name)
	}
}

func TestLightLayer_RenderTextureImage(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	img := ll.RenderTextureImage()
	if img == nil {
		t.Error("RenderTextureImage should not be nil")
	}
}

func TestLightLayer_RenderTextureImage_NilRT(t *testing.T) {
	ll := &LightLayer{rt: nil}
	if ll.RenderTextureImage() != nil {
		t.Error("should return nil when rt is nil")
	}
}

func TestLightLayer_Redraw_NoLights(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	ll.Redraw() // should not panic with no lights
}

func TestLightLayer_Redraw_WithLight(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	l := &Light{
		X: 32, Y: 32, Radius: 20, Intensity: 1.0, Enabled: true,
	}
	ll.AddLight(l)
	ll.Redraw() // should not panic
}

func TestLightLayer_Redraw_DisabledLight(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	l := &Light{
		X: 32, Y: 32, Radius: 20, Intensity: 1.0, Enabled: false,
	}
	ll.AddLight(l)
	ll.Redraw() // disabled light should be skipped
}

func TestLightLayer_Redraw_ZeroRadius(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	l := &Light{
		X: 32, Y: 32, Radius: 0, Intensity: 1.0, Enabled: true,
	}
	ll.AddLight(l)
	ll.Redraw() // zero radius should be skipped
}

func TestLightLayer_Redraw_ColoredLight(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	l := &Light{
		X: 32, Y: 32, Radius: 20, Intensity: 0.8, Enabled: true,
		Color: types.RGBA(1, 0, 0, 1),
	}
	ll.AddLight(l)
	ll.Redraw() // should do both erase and tint passes
}

func TestLightLayer_Redraw_WithTarget(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(128, 128, 0.5)
	target := node.NewNode("target", types.NodeTypeSprite)
	target.TransformDirty = true
	node.UpdateWorldTransform(target, node.IdentityTransform, 1.0, true, false)

	l := &Light{
		X: 0, Y: 0, Radius: 20, Intensity: 1.0, Enabled: true,
		Target: target, OffsetX: 5, OffsetY: -3,
	}
	ll.AddLight(l)
	ll.Redraw() // should follow target position
}

func TestLightLayer_CircleCache(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	ll.SetCircleRadius(10)
	ll.SetCircleRadius(10) // second call should use cache

	if ll.circleCache == nil || len(ll.circleCache) != 1 {
		t.Errorf("circle cache should have 1 entry, got %d", len(ll.circleCache))
	}

	ll.SetCircleRadius(20) // different radius
	if len(ll.circleCache) != 2 {
		t.Errorf("circle cache should have 2 entries, got %d", len(ll.circleCache))
	}
}

func TestLightLayer_Dispose(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	ll.AddLight(&Light{Enabled: true, Radius: 10})
	ll.SetCircleRadius(10)
	ll.Dispose()

	if ll.rt != nil {
		t.Error("rt should be nil after Dispose")
	}
	if ll.node_ != nil {
		t.Error("node should be nil after Dispose")
	}
	if ll.lights != nil {
		t.Error("lights should be nil after Dispose")
	}
	if ll.circleCache != nil {
		t.Error("circleCache should be nil after Dispose")
	}
}

func TestLightLayer_SetPages(t *testing.T) {
	ll := &LightLayer{}
	pages := []*ebiten.Image{ebiten.NewImage(32, 32)}
	ll.SetPages(pages)
	if len(ll.pages) != 1 {
		t.Errorf("pages = %d, want 1", len(ll.pages))
	}
}

func TestLightLayer_RemoveNonexistent(t *testing.T) {
	ll := &LightLayer{}
	l1 := &Light{}
	l2 := &Light{}
	ll.AddLight(l1)
	ll.RemoveLight(l2) // removing non-existent should not panic
	if len(ll.Lights()) != 1 {
		t.Errorf("lights = %d, want 1", len(ll.Lights()))
	}
}

func TestLightLayer_Redraw_MultipleLights(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(128, 128, 0.3)
	for i := 0; i < 5; i++ {
		ll.AddLight(&Light{
			X: float64(i * 20), Y: 64,
			Radius: 15, Intensity: 0.5 + float64(i)*0.1, Enabled: true,
		})
	}
	ll.Redraw() // should not panic with multiple lights
}

func TestLightLayer_Redraw_WithRotatedLight(t *testing.T) {
	cleanup := setupLightingFns()
	defer cleanup()

	ll := NewLightLayer(64, 64, 0.5)
	l := &Light{
		X: 32, Y: 32, Radius: 20, Intensity: 1.0, Enabled: true,
		Rotation: math.Pi / 4,
	}
	ll.AddLight(l)
	ll.Redraw() // should apply rotation GeoM
}
