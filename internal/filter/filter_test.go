package filter

import (
	"math"
	"testing"

	"github.com/phanxgames/willow/internal/types"
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
