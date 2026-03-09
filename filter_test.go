package willow

import "testing"

// --- Padding ---

func TestColorMatrixFilterPadding(t *testing.T) {
	f := NewColorMatrixFilter()
	if f.Padding() != 0 {
		t.Errorf("ColorMatrixFilter Padding() = %d, want 0", f.Padding())
	}
}

func TestBlurFilterPadding(t *testing.T) {
	f := NewBlurFilter(8)
	if f.Padding() != 8 {
		t.Errorf("BlurFilter Padding() = %d, want 8", f.Padding())
	}
}

func TestBlurFilterNegativeRadius(t *testing.T) {
	f := NewBlurFilter(-5)
	if f.Radius != 0 {
		t.Errorf("negative radius should clamp to 0, got %d", f.Radius)
	}
}

func TestOutlineFilterPadding(t *testing.T) {
	f := NewOutlineFilter(3, ColorWhite)
	if f.Padding() != 3 {
		t.Errorf("OutlineFilter Padding() = %d, want 3", f.Padding())
	}
}

func TestPixelPerfectOutlineFilterPadding(t *testing.T) {
	f := NewPixelPerfectOutlineFilter(ColorWhite)
	if f.Padding() != 1 {
		t.Errorf("PixelPerfectOutlineFilter Padding() = %d, want 1", f.Padding())
	}
}

func TestPixelPerfectInlineFilterPadding(t *testing.T) {
	f := NewPixelPerfectInlineFilter(ColorWhite)
	if f.Padding() != 0 {
		t.Errorf("PixelPerfectInlineFilter Padding() = %d, want 0", f.Padding())
	}
}

func TestPaletteFilterPadding(t *testing.T) {
	f := NewPaletteFilter()
	if f.Padding() != 0 {
		t.Errorf("PaletteFilter Padding() = %d, want 0", f.Padding())
	}
}

func TestCustomShaderFilterPadding(t *testing.T) {
	f := NewCustomShaderFilter(nil, 5)
	if f.Padding() != 5 {
		t.Errorf("CustomShaderFilter Padding() = %d, want 5", f.Padding())
	}
}

// --- Filter creation ---

func TestColorMatrixFilterIdentity(t *testing.T) {
	f := NewColorMatrixFilter()
	// Identity matrix has 1s on diagonal positions [0], [6], [12], [18].
	if f.Matrix[0] != 1 || f.Matrix[6] != 1 || f.Matrix[12] != 1 || f.Matrix[18] != 1 {
		t.Error("identity matrix diagonal should be all 1s")
	}
	// All other entries should be 0.
	for i, v := range f.Matrix {
		if i == 0 || i == 6 || i == 12 || i == 18 {
			continue
		}
		if v != 0 {
			t.Errorf("Matrix[%d] = %f, want 0", i, v)
		}
	}
}

func TestColorMatrixFilterSetBrightness(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetBrightness(0.5)
	// Offset columns [4], [9], [14] should be 0.5.
	if f.Matrix[4] != 0.5 || f.Matrix[9] != 0.5 || f.Matrix[14] != 0.5 {
		t.Error("brightness offsets should be 0.5")
	}
}

func TestColorMatrixFilterSetContrast(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetContrast(2.0)
	// Diagonal should be 2.0, offsets should be (1-2)/2 = -0.5.
	if f.Matrix[0] != 2.0 || f.Matrix[6] != 2.0 || f.Matrix[12] != 2.0 {
		t.Error("contrast diagonal should be 2.0")
	}
	if f.Matrix[4] != -0.5 || f.Matrix[9] != -0.5 || f.Matrix[14] != -0.5 {
		t.Error("contrast offset should be -0.5")
	}
}

func TestColorMatrixFilterSetSaturation(t *testing.T) {
	f := NewColorMatrixFilter()
	f.SetSaturation(0)
	// At saturation 0, all rows should produce grayscale (same coefficients).
	// R row: [0.299, 0.587, 0.114, 0, 0]
	assertNear(t, "Matrix[0]", f.Matrix[0], 0.299)
	assertNear(t, "Matrix[1]", f.Matrix[1], 0.587)
	assertNear(t, "Matrix[2]", f.Matrix[2], 0.114)
}

// --- Filter chain ---

func TestFilterChainOnNode(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	f1 := NewColorMatrixFilter()
	f2 := NewBlurFilter(4)
	n.Filters = []any{f1, f2}
	if len(n.Filters) != 2 {
		t.Errorf("filter count = %d, want 2", len(n.Filters))
	}
}

func TestFilterChainPadding(t *testing.T) {
	filters := []any{
		NewColorMatrixFilter(),                   // padding 0
		NewBlurFilter(8),                         // padding 8
		NewOutlineFilter(3, ColorWhite),          // padding 3
		NewPixelPerfectOutlineFilter(ColorWhite), // padding 1
	}
	pad := filterChainPadding(filters)
	// Per spec section 13.5: chain padding is cumulative (sum), not max.
	if pad != 12 {
		t.Errorf("chain padding = %d, want 12", pad)
	}
}

func TestFilterChainPaddingEmpty(t *testing.T) {
	pad := filterChainPadding(nil)
	if pad != 0 {
		t.Errorf("empty chain padding = %d, want 0", pad)
	}
}

// --- CustomShaderFilter ---

func TestCustomShaderFilterCreation(t *testing.T) {
	f := NewCustomShaderFilter(nil, 2)
	if f.Shader != nil {
		t.Error("Shader should be nil when created with nil")
	}
	if f.padding != 2 {
		t.Errorf("padding = %d, want 2", f.padding)
	}
	if f.Uniforms == nil {
		t.Error("Uniforms map should be initialized")
	}
}

func TestCustomShaderFilterUniforms(t *testing.T) {
	f := NewCustomShaderFilter(nil, 0)
	f.Uniforms["myFloat"] = float32(1.5)
	f.Uniforms["myVec"] = []float32{1, 2, 3}
	if v, ok := f.Uniforms["myFloat"]; !ok || v != float32(1.5) {
		t.Error("uniform myFloat not stored correctly")
	}
	if v, ok := f.Uniforms["myVec"]; !ok {
		t.Error("uniform myVec not found")
	} else {
		vec := v.([]float32)
		if len(vec) != 3 || vec[0] != 1 || vec[1] != 2 || vec[2] != 3 {
			t.Error("uniform myVec values incorrect")
		}
	}
}

// --- PaletteFilter ---

func TestPaletteFilterDefaultGrayscale(t *testing.T) {
	f := NewPaletteFilter()
	// Entry 0 should be black, entry 255 should be white.
	if f.Palette[0].R() != 0 || f.Palette[0].G() != 0 || f.Palette[0].B() != 0 {
		t.Error("palette[0] should be black")
	}
	if f.Palette[255].R() != 1 || f.Palette[255].G() != 1 || f.Palette[255].B() != 1 {
		t.Error("palette[255] should be white")
	}
}

func TestPaletteFilterSetPalette(t *testing.T) {
	f := NewPaletteFilter()
	f.paletteDirty = false
	var custom [256]Color
	custom[0] = RGBA(1, 0, 0, 1)
	f.SetPalette(custom)
	if !f.paletteDirty {
		t.Error("paletteDirty should be true after SetPalette")
	}
	if f.Palette[0].R() != 1 || f.Palette[0].G() != 0 {
		t.Error("palette not updated")
	}
}

// --- Blur filter ---

func TestNewBlurFilter(t *testing.T) {
	f := NewBlurFilter(12)
	if f.Radius != 12 {
		t.Errorf("Radius = %d, want 12", f.Radius)
	}
}

// --- Outline filter ---

func TestNewOutlineFilter(t *testing.T) {
	c := RGBA(1, 0, 0, 1)
	f := NewOutlineFilter(2, c)
	if f.Thickness != 2 {
		t.Errorf("Thickness = %d, want 2", f.Thickness)
	}
	if f.Color != c {
		t.Error("Color not stored correctly")
	}
}
