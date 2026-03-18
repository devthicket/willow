package integration

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"testing"

	"github.com/devthicket/willow/internal/node"

	. "github.com/devthicket/willow"
)

// --- TextBlock color tint ---

func TestTextBlock_ColorTint(t *testing.T) {
	tb := &TextBlock{
		Content:     "A",
		Color:       RGBA(0.5, 0.8, 1.0, 0.9),
		LayoutDirty: true,
	}

	// Verify the color is preserved on the TextBlock
	if tb.Color.R() != 0.5 || tb.Color.G() != 0.8 || tb.Color.B() != 1.0 || tb.Color.A() != 0.9 {
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
	if n.TextBlock.Color != RGBA(1, 1, 1, 1) {
		t.Errorf("TextBlock.Color = %+v, want white", n.TextBlock.Color)
	}
	if !n.TextBlock.LayoutDirty {
		t.Error("LayoutDirty should be true initially")
	}
}

func TestNewText_DefaultFontSize(t *testing.T) {
	n := NewText("label", "Hello", nil)
	if n.TextBlock.FontSize != 16 {
		t.Errorf("FontSize = %f, want 16", n.TextBlock.FontSize)
	}
	if n.ScaleX_ != 1 || n.ScaleY_ != 1 {
		t.Errorf("Scale = (%f, %f), want (1, 1)", n.ScaleX_, n.ScaleY_)
	}
}

func TestFontScale_NilFont(t *testing.T) {
	tb := &TextBlock{FontSize: 24}
	if textBlockFontScale(tb) != 1.0 {
		t.Errorf("textBlockFontScale with nil Font = %f, want 1.0", textBlockFontScale(tb))
	}
}

func TestFontScale_ZeroFontSize(t *testing.T) {
	ff := testFontFamily(t, 80)
	tb := &TextBlock{FontSize: 0, Font: ff}
	if textBlockFontScale(tb) != 1.0 {
		t.Errorf("textBlockFontScale with FontSize 0 = %f, want 1.0", textBlockFontScale(tb))
	}
}

func TestFontScale_NegativeFontSize(t *testing.T) {
	ff := testFontFamily(t, 80)
	tb := &TextBlock{FontSize: -1, Font: ff}
	if textBlockFontScale(tb) != 1.0 {
		t.Errorf("textBlockFontScale with negative FontSize = %f, want 1.0", textBlockFontScale(tb))
	}
}

func TestFontScale_Computed(t *testing.T) {
	ff := testFontFamily(t, 80)
	tb := &TextBlock{FontSize: 24, Font: ff}
	expected := 24.0 / (80 * 1.2) // lineHeight = size * 1.2 from minAtlasJSON
	got := textBlockFontScale(tb)
	if math.Abs(got-expected) > 1e-6 {
		t.Errorf("textBlockFontScale = %f, want %f", got, expected)
	}
}

// --- composeGlyphTransform ---

func TestComposeGlyphTransform_Identity(t *testing.T) {
	world := node.IdentityTransform
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

// testFontFamily creates a minimal FontFamily from a font bundle for testing.
func testFontFamily(t *testing.T, size float64) *FontFamily {
	t.Helper()
	bundle := testBundle(t, size)
	ff, err := NewFontFamilyFromFontBundle(bundle)
	if err != nil {
		t.Fatalf("testFontFamily: %v", err)
	}
	return ff
}

func testBundle(t *testing.T, size float64) []byte {
	t.Helper()
	type sdfGlyphEntry struct {
		ID       int `json:"id"`
		Width    int `json:"width"`
		Height   int `json:"height"`
		XAdvance int `json:"xAdvance"`
	}
	type sdfMetrics struct {
		Type          string          `json:"type"`
		Size          float64         `json:"size"`
		DistanceRange float64         `json:"distanceRange"`
		LineHeight    float64         `json:"lineHeight"`
		Base          float64         `json:"base"`
		Glyphs        []sdfGlyphEntry `json:"glyphs"`
	}
	m := sdfMetrics{
		Type:          "sdf",
		Size:          size,
		DistanceRange: size * 0.125,
		LineHeight:    size * 1.2,
		Base:          size,
		Glyphs:        []sdfGlyphEntry{{ID: 65, Width: 8, Height: 10, XAdvance: 9}},
	}
	jsonData, _ := json.Marshal(m)
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	base := fmt.Sprintf("regular_%.0f", size)
	w, _ := zw.Create(base + ".json")
	w.Write(jsonData)
	w2, _ := zw.Create(base + ".png")
	w2.Write(pngBuf.Bytes())
	zw.Close()
	return buf.Bytes()
}
