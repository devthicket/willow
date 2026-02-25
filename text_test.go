package willow

import (
	"math"
	"testing"
)

// --- TextBlock color tint ---

func TestTextBlock_ColorTint(t *testing.T) {
	tb := &TextBlock{
		Content:     "A",
		Color:       Color{0.5, 0.8, 1.0, 0.9},
		layoutDirty: true,
	}

	// Verify the color is preserved on the TextBlock
	if tb.Color.R != 0.5 || tb.Color.G != 0.8 || tb.Color.B != 1.0 || tb.Color.A != 0.9 {
		t.Errorf("TextBlock color = %+v, want {0.5 0.8 1.0 0.9}", tb.Color)
	}
}

// --- NewText constructor ---

func TestNewText_SetsTextBlock(t *testing.T) {
	// Use a nil font  -  constructor should still set fields correctly.
	n := NewText("label", "Hello", nil)

	if n.Type != NodeTypeText {
		t.Errorf("Type = %d, want NodeTypeText", n.Type)
	}
	if n.TextBlock == nil {
		t.Fatal("TextBlock is nil")
	}
	if n.TextBlock.Content != "Hello" {
		t.Errorf("Content = %q, want \"Hello\"", n.TextBlock.Content)
	}
	if n.TextBlock.Color != (Color{1, 1, 1, 1}) {
		t.Errorf("TextBlock.Color = %+v, want white", n.TextBlock.Color)
	}
	if !n.TextBlock.layoutDirty {
		t.Error("layoutDirty should be true initially")
	}
}

func TestNewText_DefaultFontSize(t *testing.T) {
	n := NewText("label", "Hello", nil)
	if n.TextBlock.FontSize != 16 {
		t.Errorf("FontSize = %f, want 16", n.TextBlock.FontSize)
	}
	if n.ScaleX != 1 || n.ScaleY != 1 {
		t.Errorf("Scale = (%f, %f), want (1, 1)", n.ScaleX, n.ScaleY)
	}
}

func TestFontScale_NilFont(t *testing.T) {
	tb := &TextBlock{FontSize: 24}
	if tb.fontScale() != 1.0 {
		t.Errorf("fontScale with nil Font = %f, want 1.0", tb.fontScale())
	}
}

func TestFontScale_ZeroFontSize(t *testing.T) {
	tb := &TextBlock{FontSize: 0, Font: &SpriteFont{lineHeight: 80}}
	if tb.fontScale() != 1.0 {
		t.Errorf("fontScale with FontSize 0 = %f, want 1.0", tb.fontScale())
	}
}

func TestFontScale_NegativeFontSize(t *testing.T) {
	tb := &TextBlock{FontSize: -1, Font: &SpriteFont{lineHeight: 80}}
	if tb.fontScale() != 1.0 {
		t.Errorf("fontScale with negative FontSize = %f, want 1.0", tb.fontScale())
	}
}

func TestFontScale_Computed(t *testing.T) {
	tb := &TextBlock{FontSize: 24, Font: &SpriteFont{lineHeight: 80}}
	expected := 24.0 / 80.0
	if math.Abs(tb.fontScale()-expected) > 1e-10 {
		t.Errorf("fontScale = %f, want %f", tb.fontScale(), expected)
	}
}

// --- composeGlyphTransform ---

func TestComposeGlyphTransform_Identity(t *testing.T) {
	world := identityTransform
	result := composeGlyphTransform(world, 10, 20)

	if result[4] != 10 || result[5] != 20 {
		t.Errorf("translate = (%f, %f), want (10, 20)", result[4], result[5])
	}
	if result[0] != 1 || result[3] != 1 {
		t.Error("scale should remain identity")
	}
}

func TestComposeGlyphTransform_Scaled(t *testing.T) {
	world := [6]float64{2, 0, 0, 2, 50, 100} // 2x scale, translate(50,100)
	result := composeGlyphTransform(world, 10, 20)

	// tx = 2*10 + 0*20 + 50 = 70
	// ty = 0*10 + 2*20 + 100 = 140
	if math.Abs(result[4]-70) > 0.01 {
		t.Errorf("tx = %f, want 70", result[4])
	}
	if math.Abs(result[5]-140) > 0.01 {
		t.Errorf("ty = %f, want 140", result[5])
	}
}
