package willow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Rect.Contains ---

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

// --- Rect.Intersects ---

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

// --- Enum constant values (catch accidental iota drift) ---

func TestEnumValues(t *testing.T) {
	// BlendMode
	if BlendNormal != 0 {
		t.Errorf("BlendNormal = %d, want 0", BlendNormal)
	}
	if BlendNone != 7 {
		t.Errorf("BlendNone = %d, want 7", BlendNone)
	}

	// NodeType
	if NodeTypeContainer != 0 {
		t.Errorf("NodeTypeContainer = %d, want 0", NodeTypeContainer)
	}
	if NodeTypeText != 4 {
		t.Errorf("NodeTypeText = %d, want 4", NodeTypeText)
	}

	// EventType
	if EventPointerDown != 0 {
		t.Errorf("EventPointerDown = %d, want 0", EventPointerDown)
	}
	if EventPinch != 7 {
		t.Errorf("EventPinch = %d, want 7", EventPinch)
	}

	// MouseButton
	if MouseButtonLeft != 0 {
		t.Errorf("MouseButtonLeft = %d, want 0", MouseButtonLeft)
	}
	if MouseButtonMiddle != 2 {
		t.Errorf("MouseButtonMiddle = %d, want 2", MouseButtonMiddle)
	}

	// KeyModifiers (bitmask)
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

	// TextAlign
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
