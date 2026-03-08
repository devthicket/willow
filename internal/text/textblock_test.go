package text

import (
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestTextBlock_Invalidate(t *testing.T) {
	tb := &TextBlock{LayoutDirty: false, UniformsDirty: false}
	tb.Invalidate()
	if !tb.LayoutDirty || !tb.UniformsDirty {
		t.Error("Invalidate should set both dirty flags")
	}
}

func TestTextBlock_FontScale_NoFont(t *testing.T) {
	tb := &TextBlock{FontSize: 24}
	if tb.FontScale() != 1.0 {
		t.Errorf("FontScale with nil font = %f, want 1.0", tb.FontScale())
	}
}

func TestTextBlock_MeasureDisplay_NoFont(t *testing.T) {
	tb := &TextBlock{}
	w, h := tb.MeasureDisplay("hello")
	if w != 0 || h != 0 {
		t.Errorf("MeasureDisplay with nil font = (%f, %f), want (0, 0)", w, h)
	}
}

func TestTextBlock_EffectiveLineHeight_Override(t *testing.T) {
	tb := &TextBlock{LineHeight: 30}
	if tb.EffectiveLineHeight() != 30 {
		t.Errorf("EffectiveLineHeight = %f, want 30", tb.EffectiveLineHeight())
	}
}

func TestTextBlock_LayoutSDF_Simple(t *testing.T) {
	// Create a simple glyph lookup for 'A' and 'B'
	glyphs := map[rune]*Glyph{
		'A': {ID: 'A', X: 0, Y: 0, Width: 10, Height: 12, XAdvance: 11},
		'B': {ID: 'B', X: 10, Y: 0, Width: 10, Height: 12, XAdvance: 11},
	}
	lookup := func(r rune) *Glyph { return glyphs[r] }
	kern := func(a, b rune) int16 { return 0 }

	tb := &TextBlock{
		Content:  "AB",
		FontSize: 12,
		Font:     &mockFont{lineHeight: 12},
		Color:    types.RGB(1, 1, 1),
	}
	tb.LayoutSDF(lookup, kern, 0)

	if len(tb.Lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(tb.Lines))
	}
	if len(tb.Lines[0].Glyphs) != 2 {
		t.Errorf("glyphs = %d, want 2", len(tb.Lines[0].Glyphs))
	}
}

func TestTextBlock_LayoutSDF_Newline(t *testing.T) {
	glyphs := map[rune]*Glyph{
		'A': {ID: 'A', X: 0, Y: 0, Width: 10, Height: 12, XAdvance: 11},
	}
	lookup := func(r rune) *Glyph { return glyphs[r] }
	kern := func(a, b rune) int16 { return 0 }

	tb := &TextBlock{
		Content:  "A\nA",
		FontSize: 12,
		Font:     &mockFont{lineHeight: 12},
		Color:    types.RGB(1, 1, 1),
	}
	tb.LayoutSDF(lookup, kern, 0)

	if len(tb.Lines) != 2 {
		t.Errorf("lines = %d, want 2", len(tb.Lines))
	}
}

func TestTextBlock_RebuildLocalVerts(t *testing.T) {
	tb := &TextBlock{
		Lines: []TextLine{
			{
				Glyphs: []GlyphPos{
					{X: 0, Y: 0, Region: types.TextureRegion{X: 0, Y: 0, Width: 10, Height: 12}},
					{X: 11, Y: 0, Region: types.TextureRegion{X: 10, Y: 0, Width: 10, Height: 12}},
				},
				Width: 22,
			},
		},
	}
	tb.RebuildLocalVerts(12)

	if tb.SdfVertCount != 8 { // 2 glyphs * 4 verts
		t.Errorf("SdfVertCount = %d, want 8", tb.SdfVertCount)
	}
	if tb.SdfIndCount != 12 { // 2 glyphs * 6 indices
		t.Errorf("SdfIndCount = %d, want 12", tb.SdfIndCount)
	}
}

// mockFont implements Font for testing.
type mockFont struct {
	lineHeight float64
}

func (f *mockFont) MeasureString(text string) (float64, float64) {
	return float64(len(text)) * 10, f.lineHeight
}

func (f *mockFont) LineHeight() float64 { return f.lineHeight }
