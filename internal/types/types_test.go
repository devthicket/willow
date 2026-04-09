package types

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRGB(t *testing.T) {
	c := RGB(1, 0, 0)
	if c.R() != 1 || c.G() != 0 || c.B() != 0 || c.A() != 1 {
		t.Errorf("RGB(1,0,0) = %v, want r=1,g=0,b=0,a=1", c)
	}
}

func TestRGBA(t *testing.T) {
	c := RGBA(0.5, 0.5, 0.5, 0.5)
	if c.R() != 0.5 || c.A() != 0.5 {
		t.Errorf("RGBA(0.5,0.5,0.5,0.5) = %v", c)
	}
}

func TestColor_RGBA8(t *testing.T) {
	c := RGB(1, 0, 0)
	r, g, b, a := c.RGBA8()
	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Errorf("RGBA8() = (%d,%d,%d,%d), want (255,0,0,255)", r, g, b, a)
	}
}

func TestColorFromRGBA(t *testing.T) {
	c := ColorFromRGBA(255, 128, 0, 255)
	if c.R() != 1 || c.A() != 1 {
		t.Errorf("ColorFromRGBA(255,128,0,255).R()=%f, A()=%f", c.R(), c.A())
	}
	if c.G() < 0.5 || c.G() > 0.51 {
		t.Errorf("ColorFromRGBA(255,128,0,255).G()=%f, want ~0.502", c.G())
	}
}

func TestColorFromHSV(t *testing.T) {
	c := ColorFromHSV(0, 1, 1) // pure red
	if c.R() != 1 || c.G() != 0 || c.B() != 0 {
		t.Errorf("ColorFromHSV(0,1,1) = r=%f g=%f b=%f, want r=1,g=0,b=0", c.R(), c.G(), c.B())
	}
}

func TestColor_AdjustHue(t *testing.T) {
	// Red (hue 0) shifted by 120 degrees should be green
	red := ColorFromHSV(0, 1, 1)
	green := red.AdjustHue(120)
	if green.G() < 0.99 || green.R() > 0.01 {
		t.Errorf("AdjustHue(120) on red: got r=%f g=%f b=%f, want green", green.R(), green.G(), green.B())
	}
	// Wrap: shifting by 360 returns the same hue
	same := red.AdjustHue(360)
	if same.R() < 0.99 {
		t.Errorf("AdjustHue(360) should wrap back to red, got r=%f", same.R())
	}
	// Alpha is preserved
	c := RGBA(1, 0, 0, 0.5)
	out := c.AdjustHue(180)
	if out.A() != 0.5 {
		t.Errorf("AdjustHue should preserve alpha, got %f", out.A())
	}
}

func TestColor_AdjustSaturation(t *testing.T) {
	c := ColorFromHSV(0, 0.5, 1)
	// Double saturation
	doubled := c.AdjustSaturation(2.0)
	_, s1, _ := doubled.toHSV()
	if s1 < 0.99 {
		t.Errorf("AdjustSaturation(2.0): s=%f, want ~1", s1)
	}
	// Zero saturation = grayscale
	gray := c.AdjustSaturation(0)
	_, s2, _ := gray.toHSV()
	if s2 > 0.01 {
		t.Errorf("AdjustSaturation(0): s=%f, want ~0", s2)
	}
	// 1.0 = no change
	same := c.AdjustSaturation(1.0)
	_, s3, _ := same.toHSV()
	if s3 < 0.49 || s3 > 0.51 {
		t.Errorf("AdjustSaturation(1.0): s=%f, want ~0.5", s3)
	}
}

func TestColor_AdjustValue(t *testing.T) {
	c := ColorFromHSV(0, 1, 0.5)
	// Double value
	doubled := c.AdjustValue(2.0)
	_, _, v := doubled.toHSV()
	if v < 0.99 {
		t.Errorf("AdjustValue(2.0): v=%f, want ~1", v)
	}
	// 1.0 = no change
	same := c.AdjustValue(1.0)
	_, _, v2 := same.toHSV()
	if v2 < 0.49 || v2 > 0.51 {
		t.Errorf("AdjustValue(1.0): v=%f, want ~0.5", v2)
	}
}

func TestColor_AdjustAlpha(t *testing.T) {
	c := RGBA(1, 0, 0, 0.8)
	// Halve alpha
	out := c.AdjustAlpha(0.5)
	if out.A() < 0.39 || out.A() > 0.41 {
		t.Errorf("AdjustAlpha(0.5): got %f, want ~0.4", out.A())
	}
	// 1.0 = no change
	if c.AdjustAlpha(1.0).A() != 0.8 {
		t.Error("AdjustAlpha(1.0) should not change alpha")
	}
	// Clamp at 1
	if c.AdjustAlpha(999).A() != 1 {
		t.Error("AdjustAlpha should clamp to 1")
	}
}

func TestColor_AdjustChannels(t *testing.T) {
	c := RGB(0.5, 0.5, 0.5)
	// Double red
	if r := c.AdjustRed(2.0).R(); r < 0.99 {
		t.Errorf("AdjustRed(2.0): got %f, want ~1", r)
	}
	// Halve green
	if g := c.AdjustGreen(0.5).G(); g < 0.24 || g > 0.26 {
		t.Errorf("AdjustGreen(0.5): got %f, want ~0.25", g)
	}
	// Zero blue
	if b := c.AdjustBlue(0).B(); b != 0 {
		t.Errorf("AdjustBlue(0): got %f, want 0", b)
	}
	// 1.0 = no change, other channels untouched
	out := c.AdjustRed(1.0)
	if out.R() != 0.5 || out.G() != 0.5 || out.B() != 0.5 {
		t.Errorf("AdjustRed(1.0) should be identity")
	}
}

func TestColor_AdjustBrightness(t *testing.T) {
	c := RGB(0.5, 0.5, 0.5)
	// 1.4x brighter
	brighter := c.AdjustBrightness(1.4)
	if brighter.R() < 0.69 || brighter.R() > 0.71 {
		t.Errorf("AdjustBrightness(1.4): R=%f, want ~0.7", brighter.R())
	}
	// 1.0 = no change
	same := c.AdjustBrightness(1.0)
	if same.R() != 0.5 {
		t.Errorf("AdjustBrightness(1.0): R=%f, want 0.5", same.R())
	}
	// Alpha unchanged
	ca := RGBA(0.5, 0.5, 0.5, 0.5)
	if ca.AdjustBrightness(1.4).A() != 0.5 {
		t.Error("AdjustBrightness should not affect alpha")
	}
	// Clamp
	if RGB(0.9, 0.9, 0.9).AdjustBrightness(2.0).R() != 1 {
		t.Error("AdjustBrightness should clamp at 1")
	}
}

func TestColor_AdjustContrast(t *testing.T) {
	// Midpoint (0.5) should be unaffected regardless of factor
	c := RGB(0.5, 0.5, 0.5)
	out := c.AdjustContrast(2.0)
	if out.R() < 0.49 || out.R() > 0.51 {
		t.Errorf("AdjustContrast: midpoint should stay ~0.5, got %f", out.R())
	}
	// 1.0 = no change
	bright := RGB(0.8, 0.5, 0.2)
	same := bright.AdjustContrast(1.0)
	if same.R() < 0.79 || same.R() > 0.81 {
		t.Errorf("AdjustContrast(1.0): R=%f, want ~0.8", same.R())
	}
	// Factor > 1 pushes values away from midpoint
	out2 := bright.AdjustContrast(2.0)
	if out2.R() <= 0.8 {
		t.Errorf("AdjustContrast(2.0): R should increase above 0.8, got %f", out2.R())
	}
	// Clamp
	clamped := RGB(1, 0, 0).AdjustContrast(100)
	if clamped.R() != 1 || clamped.G() != 0 {
		t.Errorf("AdjustContrast should clamp channels")
	}
}

func TestRect_Contains(t *testing.T) {
	r := Rect{X: 10, Y: 10, Width: 100, Height: 50}
	if !r.Contains(50, 30) {
		t.Error("Contains(50,30) should be true")
	}
	if r.Contains(5, 5) {
		t.Error("Contains(5,5) should be false")
	}
	// Edge is inside
	if !r.Contains(10, 10) {
		t.Error("Contains(10,10) edge should be true")
	}
}

func TestRect_Intersects(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 100, Height: 100}
	b := Rect{X: 50, Y: 50, Width: 100, Height: 100}
	if !a.Intersects(b) {
		t.Error("overlapping rects should intersect")
	}
	c := Rect{X: 200, Y: 200, Width: 10, Height: 10}
	if a.Intersects(c) {
		t.Error("non-overlapping rects should not intersect")
	}
}

func TestClamp01(t *testing.T) {
	if Clamp01(-0.5) != 0 {
		t.Error("Clamp01(-0.5) should be 0")
	}
	if Clamp01(1.5) != 1 {
		t.Error("Clamp01(1.5) should be 1")
	}
	if Clamp01(0.5) != 0.5 {
		t.Error("Clamp01(0.5) should be 0.5")
	}
}

func TestBlendMode_Values(t *testing.T) {
	// Verify enum ordering is stable
	if BlendNormal != 0 {
		t.Errorf("BlendNormal = %d, want 0", BlendNormal)
	}
	if BlendNone != 7 {
		t.Errorf("BlendNone = %d, want 7", BlendNone)
	}
}

func TestNodeType_Values(t *testing.T) {
	if NodeTypeContainer != 0 {
		t.Errorf("NodeTypeContainer = %d, want 0", NodeTypeContainer)
	}
	if NodeTypeText != 4 {
		t.Errorf("NodeTypeText = %d, want 4", NodeTypeText)
	}
}

func TestTextureRegion_ZeroValue(t *testing.T) {
	var r TextureRegion
	if r.Page != 0 || r.Width != 0 || r.Rotated {
		t.Error("zero TextureRegion should have all zero fields")
	}
}

// --- Thorough Rect.Contains (from archived willow_test.go) ---

func TestRectContains(t *testing.T) {
	r := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	tests := []struct {
		name   string
		x, y   float64
		expect bool
	}{
		{"inside", 50, 40, true},
		{"top-left corner", 10, 20, true},
		{"bottom-right corner", 110, 70, true},
		{"left edge", 10, 40, true},
		{"right edge", 110, 40, true},
		{"top edge", 50, 20, true},
		{"bottom edge", 50, 70, true},
		{"outside left", 9, 40, false},
		{"outside right", 111, 40, false},
		{"outside above", 50, 19, false},
		{"outside below", 50, 71, false},
		{"far outside", 999, 999, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Contains(tt.x, tt.y)
			if got != tt.expect {
				t.Errorf("Rect%v.Contains(%v, %v) = %v, want %v", r, tt.x, tt.y, got, tt.expect)
			}
		})
	}
}

// --- Thorough Rect.Intersects (from archived willow_test.go) ---

func TestRectIntersects(t *testing.T) {
	base := Rect{X: 10, Y: 10, Width: 100, Height: 100}
	tests := []struct {
		name   string
		other  Rect
		expect bool
	}{
		{"overlapping", Rect{X: 50, Y: 50, Width: 100, Height: 100}, true},
		{"fully contained", Rect{X: 20, Y: 20, Width: 10, Height: 10}, true},
		{"containing", Rect{X: 0, Y: 0, Width: 200, Height: 200}, true},
		{"adjacent right", Rect{X: 110, Y: 10, Width: 50, Height: 50}, true},
		{"adjacent bottom", Rect{X: 10, Y: 110, Width: 50, Height: 50}, true},
		{"adjacent left", Rect{X: -50, Y: 10, Width: 60, Height: 50}, true},
		{"adjacent top", Rect{X: 10, Y: -50, Width: 50, Height: 60}, true},
		{"disjoint right", Rect{X: 111, Y: 10, Width: 50, Height: 50}, false},
		{"disjoint left", Rect{X: -100, Y: 10, Width: 50, Height: 50}, false},
		{"disjoint above", Rect{X: 10, Y: -100, Width: 50, Height: 50}, false},
		{"disjoint below", Rect{X: 10, Y: 111, Width: 50, Height: 50}, false},
		{"same rect", Rect{X: 10, Y: 10, Width: 100, Height: 100}, true},
		{"zero-size at corner", Rect{X: 110, Y: 110, Width: 0, Height: 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base.Intersects(tt.other)
			if got != tt.expect {
				t.Errorf("Rect%v.Intersects(Rect%v) = %v, want %v", base, tt.other, got, tt.expect)
			}
		})
	}
}

// --- BlendMode.EbitenBlend ---

func TestBlendModeEbitenBlend(t *testing.T) {
	modes := []struct {
		mode   BlendMode
		name   string
		expect ebiten.Blend
	}{
		{BlendNormal, "BlendNormal", ebiten.BlendSourceOver},
		{BlendAdd, "BlendAdd", ebiten.BlendLighter},
		{BlendErase, "BlendErase", ebiten.BlendDestinationOut},
		{BlendBelow, "BlendBelow", ebiten.BlendDestinationOver},
		{BlendNone, "BlendNone", ebiten.BlendCopy},
	}
	for _, tt := range modes {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.EbitenBlend()
			if got != tt.expect {
				t.Errorf("%s.EbitenBlend() = %v, want %v", tt.name, got, tt.expect)
			}
		})
	}

	// Custom blends: verify they return non-zero (custom structs)
	customModes := []struct {
		mode BlendMode
		name string
	}{
		{BlendMultiply, "BlendMultiply"},
		{BlendScreen, "BlendScreen"},
		{BlendMask, "BlendMask"},
	}
	zero := ebiten.Blend{}
	for _, tt := range customModes {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.EbitenBlend()
			if got == zero {
				t.Errorf("%s.EbitenBlend() returned zero blend", tt.name)
			}
		})
	}
}

// --- Additional enum constant values (catch accidental iota drift) ---

func TestEventTypeValues(t *testing.T) {
	if EventPointerDown != 0 {
		t.Errorf("EventPointerDown = %d, want 0", EventPointerDown)
	}
	if EventPinch != 7 {
		t.Errorf("EventPinch = %d, want 7", EventPinch)
	}
}

func TestMouseButtonValues(t *testing.T) {
	if MouseButtonLeft != 0 {
		t.Errorf("MouseButtonLeft = %d, want 0", MouseButtonLeft)
	}
	if MouseButtonMiddle != 2 {
		t.Errorf("MouseButtonMiddle = %d, want 2", MouseButtonMiddle)
	}
}

func TestKeyModifierValues(t *testing.T) {
	if ModShift != 1 {
		t.Errorf("ModShift = %d, want 1", ModShift)
	}
	if ModCtrl != 2 {
		t.Errorf("ModCtrl = %d, want 2", ModCtrl)
	}
	if ModAlt != 4 {
		t.Errorf("ModAlt = %d, want 4", ModAlt)
	}
	if ModMeta != 8 {
		t.Errorf("ModMeta = %d, want 8", ModMeta)
	}
}

func TestTextAlignValues(t *testing.T) {
	if TextAlignLeft != 0 {
		t.Errorf("TextAlignLeft = %d, want 0", TextAlignLeft)
	}
	if TextAlignRight != 2 {
		t.Errorf("TextAlignRight = %d, want 2", TextAlignRight)
	}
}

func TestColorWhite(t *testing.T) {
	if ColorWhite.R() != 1 || ColorWhite.G() != 1 || ColorWhite.B() != 1 || ColorWhite.A() != 1 {
		t.Errorf("ColorWhite = %v, want {1,1,1,1}", ColorWhite)
	}
}

// --- Benchmarks (verify zero allocations) ---

func BenchmarkRectContains(b *testing.B) {
	r := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	b.ReportAllocs()
	for b.Loop() {
		_ = r.Contains(50, 40)
	}
}

func BenchmarkRectIntersects(b *testing.B) {
	r := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	other := Rect{X: 50, Y: 40, Width: 80, Height: 60}
	b.ReportAllocs()
	for b.Loop() {
		_ = r.Intersects(other)
	}
}

func BenchmarkBlendModeMapping(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = BlendNormal.EbitenBlend()
		_ = BlendAdd.EbitenBlend()
		_ = BlendMultiply.EbitenBlend()
		_ = BlendScreen.EbitenBlend()
		_ = BlendErase.EbitenBlend()
		_ = BlendMask.EbitenBlend()
		_ = BlendBelow.EbitenBlend()
		_ = BlendNone.EbitenBlend()
	}
}
