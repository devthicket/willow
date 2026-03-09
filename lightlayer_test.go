package willow

import "testing"

func TestNewLightLayerCreatesNode(t *testing.T) {
	ll := NewLightLayer(256, 256, 0.7)
	defer ll.Dispose()

	n := ll.Node()
	if n == nil {
		t.Fatal("Node() should not be nil")
	}
	if n.Type != NodeTypeSprite {
		t.Errorf("node Type = %d, want NodeTypeSprite", n.Type)
	}
	if n.BlendMode_ != BlendMultiply {
		t.Errorf("BlendMode = %d, want BlendMultiply", n.BlendMode_)
	}
	if n.CustomImage_ == nil {
		t.Error("node should have customImage set")
	}
}

func TestLightLayerAddRemoveClearLights(t *testing.T) {
	ll := NewLightLayer(64, 64, 0.5)
	defer ll.Dispose()

	l1 := &Light{X: 10, Y: 10, Radius: 20, Intensity: 1, Enabled: true}
	l2 := &Light{X: 30, Y: 30, Radius: 15, Intensity: 0.8, Enabled: true}

	ll.AddLight(l1)
	ll.AddLight(l2)
	if len(ll.Lights()) != 2 {
		t.Fatalf("Lights = %d, want 2", len(ll.Lights()))
	}

	ll.RemoveLight(l1)
	if len(ll.Lights()) != 1 {
		t.Fatalf("Lights = %d after remove, want 1", len(ll.Lights()))
	}
	if ll.Lights()[0] != l2 {
		t.Error("remaining light should be l2")
	}

	ll.ClearLights()
	if len(ll.Lights()) != 0 {
		t.Errorf("Lights = %d after clear, want 0", len(ll.Lights()))
	}
}

func TestLightLayerRemoveNonexistent(t *testing.T) {
	ll := NewLightLayer(64, 64, 0.5)
	defer ll.Dispose()

	l := &Light{X: 10, Y: 10, Radius: 5, Intensity: 1, Enabled: true}
	// Should not panic.
	ll.RemoveLight(l)
}

func TestLightLayerAmbientAlphaRoundTrip(t *testing.T) {
	ll := NewLightLayer(64, 64, 0.3)
	defer ll.Dispose()

	if ll.AmbientAlpha() != 0.3 {
		t.Errorf("AmbientAlpha = %v, want 0.3", ll.AmbientAlpha())
	}

	ll.SetAmbientAlpha(0.9)
	if ll.AmbientAlpha() != 0.9 {
		t.Errorf("AmbientAlpha = %v, want 0.9", ll.AmbientAlpha())
	}
}

func TestLightLayerRedrawNoPanic(t *testing.T) {
	ll := NewLightLayer(128, 128, 0.5)
	defer ll.Dispose()

	// No lights  -  should not panic.
	ll.Redraw()

	// Enabled light.
	l1 := &Light{X: 50, Y: 50, Radius: 30, Intensity: 1, Enabled: true}
	ll.AddLight(l1)
	ll.Redraw()

	// Disabled light.
	l2 := &Light{X: 80, Y: 80, Radius: 20, Intensity: 0.5, Enabled: false}
	ll.AddLight(l2)
	ll.Redraw()

	// Zero-radius light (should be skipped).
	l3 := &Light{X: 10, Y: 10, Radius: 0, Intensity: 1, Enabled: true}
	ll.AddLight(l3)
	ll.Redraw()
}

func TestLightLayerSetCircleRadius(t *testing.T) {
	ll := NewLightLayer(64, 64, 0.5)
	defer ll.Dispose()

	// SetCircleRadius should not panic and should pre-generate the circle.
	ll.SetCircleRadius(25)
	ll.SetCircleRadius(50)

	// Verify circles are reused (redraw should not panic with these radii).
	ll.AddLight(&Light{X: 32, Y: 32, Radius: 25, Intensity: 1, Enabled: true})
	ll.AddLight(&Light{X: 16, Y: 16, Radius: 50, Intensity: 1, Enabled: true})
	ll.Redraw()
}

func TestLightLayerDispose(t *testing.T) {
	ll := NewLightLayer(64, 64, 0.5)
	ll.AddLight(&Light{X: 10, Y: 10, Radius: 10, Intensity: 1, Enabled: true})
	ll.SetCircleRadius(10)

	ll.Dispose()

	// After dispose, Node() and RenderTextureImage() should be nil.
	if ll.Node() != nil {
		t.Error("Node() should be nil after Dispose")
	}
	if ll.RenderTextureImage() != nil {
		t.Error("RenderTextureImage() should be nil after Dispose")
	}
	if len(ll.Lights()) != 0 {
		t.Error("lights should be empty after Dispose")
	}

	// Double dispose should not panic.
	ll.Dispose()
}

func TestLightLayerNodeEmitsDirectImage(t *testing.T) {
	s := NewScene()
	ll := NewLightLayer(64, 64, 0.5)
	defer ll.Dispose()

	s.Root.AddChild(ll.Node())

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	cmd := s.Pipeline.Commands[0]
	if cmd.DirectImage == nil {
		t.Error("LightLayer node should emit a directImage command")
	}
	if cmd.DirectImage != ll.RenderTextureImage() {
		t.Error("directImage should be the LightLayer's render texture image")
	}
}

func TestGenerateCircle(t *testing.T) {
	img := generateCircle(16)
	if img == nil {
		t.Fatal("generateCircle should return non-nil image")
	}
	b := img.Bounds()
	if b.Dx() != 32 || b.Dy() != 32 {
		t.Errorf("circle size = %dx%d, want 32x32", b.Dx(), b.Dy())
	}
}

func TestGenerateCircleSmallRadius(t *testing.T) {
	// Very small radius should not panic.
	img := generateCircle(0.5)
	if img == nil {
		t.Fatal("generateCircle should return non-nil image")
	}
}

func benchLightLayerRedraw(b *testing.B, n int) {
	b.Helper()
	ll := NewLightLayer(512, 512, 0.7)
	defer ll.Dispose()

	for i := 0; i < n; i++ {
		ll.AddLight(&Light{
			X:         float64(i%10) * 50,
			Y:         float64(i/10) * 50,
			Radius:    25,
			Intensity: 0.9,
			Enabled:   true,
		})
	}
	ll.Redraw() // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ll.Redraw()
	}
}

func BenchmarkLightLayerRedraw_10(b *testing.B)  { benchLightLayerRedraw(b, 10) }
func BenchmarkLightLayerRedraw_50(b *testing.B)  { benchLightLayerRedraw(b, 50) }
func BenchmarkLightLayerRedraw_100(b *testing.B) { benchLightLayerRedraw(b, 100) }

func TestLightColorTinting(t *testing.T) {
	ll := NewLightLayer(128, 128, 0.8)
	defer ll.Dispose()

	// White light: no tint pass.
	l1 := &Light{X: 30, Y: 30, Radius: 20, Intensity: 1, Enabled: true, Color: ColorWhite}
	ll.AddLight(l1)
	ll.Redraw() // should not panic

	// Colored light: triggers additive tint pass.
	l2 := &Light{X: 80, Y: 80, Radius: 25, Intensity: 0.8, Enabled: true, Color: RGBA(1, 0, 0, 1)}
	ll.AddLight(l2)
	ll.Redraw() // should not panic

	// Zero-color light: no tint pass (zero value == no color set).
	l3 := &Light{X: 50, Y: 50, Radius: 15, Intensity: 1, Enabled: true}
	ll.AddLight(l3)
	ll.Redraw() // should not panic
}
