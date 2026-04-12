package filter

import (
	"math"
	"testing"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestColorMatrixFilter_Identity(t *testing.T) {
	f := NewColorMatrixFilter()
	if f.Matrix[0] != 1 || f.Matrix[6] != 1 || f.Matrix[12] != 1 || f.Matrix[18] != 1 {
		t.Error("identity matrix diagonal should be 1")
	}
	if f.Padding() != 0 {
		t.Error("padding should be 0")
	}
}

func TestColorMatrixFilter_SetBrightness(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetBrightness(0.5)
	if f.Matrix[4] != 0.5 || f.Matrix[9] != 0.5 || f.Matrix[14] != 0.5 {
		t.Error("brightness offset should be 0.5")
	}
}

func TestColorMatrixFilter_SetContrast(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetContrast(2.0)
	if f.Matrix[0] != 2.0 {
		t.Errorf("contrast diagonal = %f, want 2.0", f.Matrix[0])
	}
}

func TestColorMatrixFilter_SetSaturation(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetSaturation(0)
	// At saturation 0, should be grayscale weights
	if f.Matrix[0] != 0.299 {
		t.Errorf("grayscale R weight = %f, want 0.299", f.Matrix[0])
	}
}

func TestColorMatrixFilter_SetInvert(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetInvert()
	if f.Matrix[0] != -1 || f.Matrix[4] != 1 {
		t.Error("invert should negate and offset")
	}
}

func TestColorMatrixFilter_SetHueRotation(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetHueRotation(math.Pi)
	// Just verify it doesn't crash and changes the matrix
	if f.Matrix[0] == 1 && f.Matrix[6] == 1 {
		t.Error("hue rotation should change matrix from identity")
	}
}

func TestBlurFilter(t *testing.T) {
	f := NewBlurFilter(5)
	if f.Padding() != 5 {
		t.Errorf("padding = %d, want 5", f.Padding())
	}
}

func TestBlurFilter_NegativeRadius(t *testing.T) {
	f := NewBlurFilter(-3)
	if f.Radius != 0 {
		t.Errorf("radius = %d, want 0", f.Radius)
	}
}

func TestOutlineFilter(t *testing.T) {
	c := types.RGBA(1, 0, 0, 1)
	f := NewOutlineFilter(3, c)
	if f.Padding() != 3 {
		t.Errorf("padding = %d, want 3", f.Padding())
	}
}

func TestPixelPerfectOutlineFilter(t *testing.T) {
	c := types.RGBA(0, 1, 0, 1)
	f := NewPixelPerfectOutlineFilter(c)
	if f.Padding() != 1 {
		t.Errorf("padding = %d, want 1", f.Padding())
	}
}

func TestPixelPerfectInlineFilter(t *testing.T) {
	c := types.RGBA(0, 0, 1, 1)
	f := NewPixelPerfectInlineFilter(c)
	if f.Padding() != 0 {
		t.Errorf("padding = %d, want 0", f.Padding())
	}
}

func TestPaletteFilter(t *testing.T) {
	f := NewPaletteFilter()
	if f.Padding() != 0 {
		t.Errorf("padding = %d, want 0", f.Padding())
	}
	// Check default grayscale palette
	if f.Palette[0] != types.RGBA(0, 0, 0, 1) {
		t.Error("palette[0] should be black")
	}
	if f.Palette[255] != types.RGBA(1, 1, 1, 1) {
		t.Error("palette[255] should be white")
	}
}

func TestCustomShaderFilter(t *testing.T) {
	f := NewCustomShaderFilter(nil, 10)
	if f.Padding() != 10 {
		t.Errorf("padding = %d, want 10", f.Padding())
	}
}

func TestChainPadding(t *testing.T) {
	filters := []Filter{
		NewBlurFilter(5),
		NewOutlineFilter(3, types.Color{}),
		NewColorMatrixFilter(),
	}
	pad := ChainPadding(filters)
	if pad != 8 {
		t.Errorf("chain padding = %d, want 8", pad)
	}
}

// --- Apply tests (require ebiten image context) ---

func newTestImage(w, h int) *ebiten.Image {
	return ebiten.NewImage(w, h)
}

func TestBlurFilter_Apply_SinglePass(t *testing.T) {
	f := NewBlurFilter(4)
	src := newTestImage(64, 64)
	dst := newTestImage(64, 64)
	f.SetPass(0)
	f.Apply(src, dst) // should not panic
}

func TestBlurFilter_Apply_LargeRadius(t *testing.T) {
	f := NewBlurFilter(32)
	src := newTestImage(64, 64)
	dst := newTestImage(64, 64)
	f.SetPass(0)
	f.Apply(src, dst) // should not panic with large radius
}

func TestBlurFilter_MultiPass(t *testing.T) {
	f := NewBlurFilter(8)
	passes := f.Passes()
	if passes <= 0 {
		t.Fatal("expected >0 passes for radius 8")
	}
	src := newTestImage(64, 64)
	a := newTestImage(64, 64)
	b := newTestImage(64, 64)

	current, scratch := src, a
	for i := 0; i < passes; i++ {
		f.SetPass(i)
		scratch.Clear()
		f.Apply(current, scratch)
		current, scratch = scratch, current
	}
	// Final result is in current; verify we didn't panic
	_ = current
	_ = b // b unused in this test, just matching pool pattern
}

func TestBlurFilter_ZeroRadius_NoPasses(t *testing.T) {
	f := NewBlurFilter(0)
	if f.Passes() != 0 {
		t.Errorf("zero-radius blur should have 0 passes, got %d", f.Passes())
	}
}

func TestOutlineFilter_Apply(t *testing.T) {
	f := NewOutlineFilter(2, types.RGBA(1, 0, 0, 1))
	src := newTestImage(32, 32)
	dst := newTestImage(36, 36) // padded
	f.Apply(src, dst)           // should not panic
}

func TestColorMatrixFilter_Apply(t *testing.T) {
	f := NewColorMatrixFilter()
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // identity — should not panic
}

func TestColorMatrixFilter_Apply_Invert(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetInvert()
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // should not panic
}

func TestPixelPerfectOutlineFilter_Apply(t *testing.T) {
	f := NewPixelPerfectOutlineFilter(types.RGBA(0, 1, 0, 1))
	src := newTestImage(32, 32)
	dst := newTestImage(34, 34) // 1px padding
	f.Apply(src, dst)           // should not panic
}

func TestPixelPerfectInlineFilter_Apply(t *testing.T) {
	f := NewPixelPerfectInlineFilter(types.RGBA(0, 0, 1, 1))
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // should not panic
}

func TestPaletteFilter_Apply(t *testing.T) {
	f := NewPaletteFilter()
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // should not panic
}

func TestPaletteFilter_SetPalette(t *testing.T) {
	f := NewPaletteFilter()
	var pal [256]types.Color
	for i := range pal {
		pal[i] = types.RGBA(1, 0, 0, 1) // all red
	}
	f.SetPalette(pal)
	if !f.paletteDirty {
		t.Error("expected palette to be marked dirty after SetPalette")
	}
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // should rebuild palette texture
	if f.paletteDirty {
		t.Error("expected palette dirty to be cleared after Apply")
	}
}

func TestPaletteFilter_CycleOffset(t *testing.T) {
	f := NewPaletteFilter()
	f.CycleOffset = 128.0
	src := newTestImage(32, 32)
	dst := newTestImage(32, 32)
	f.Apply(src, dst) // should not panic with offset
}

func TestInitShaders(t *testing.T) {
	// Should not panic — shaders compile successfully
	InitShaders()
}

func TestColorMatrixFilter_SetGrayscale(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetGrayscale()
	// Should be same as SetSaturation(0)
	if f.Matrix[0] != 0.299 {
		t.Errorf("grayscale R weight = %f, want 0.299", f.Matrix[0])
	}
}

func TestColorMatrixFilter_Chaining(t *testing.T) {
	f := NewColorMatrixFilter()
	result := f.SetBrightness(0.5)
	if result != f {
		t.Error("SetBrightness should return the same filter for chaining")
	}
	result = f.SetContrast(2.0)
	if result != f {
		t.Error("SetContrast should return the same filter for chaining")
	}
}
