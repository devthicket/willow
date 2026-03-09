package text

import (
	"encoding/json"
	"testing"
)

func TestLoadDistanceFieldFont_Basic(t *testing.T) {
	metrics := sdfMetrics{
		Type:          "sdf",
		Size:          80,
		DistanceRange: 8,
		LineHeight:    96,
		Base:          80,
		Glyphs: []sdfGlyphEntry{
			{ID: 65, X: 0, Y: 0, Width: 50, Height: 60, XOffset: 0, YOffset: -60, XAdvance: 48},
			{ID: 66, X: 50, Y: 0, Width: 48, Height: 60, XOffset: 2, YOffset: -60, XAdvance: 46},
		},
	}
	data, _ := json.Marshal(metrics)

	f, err := LoadDistanceFieldFont(data, 0)
	if err != nil {
		t.Fatalf("LoadDistanceFieldFont: %v", err)
	}

	if f.LineHeight() != 96 {
		t.Errorf("LineHeight = %f, want 96", f.LineHeight())
	}

	g := f.GlyphLookup('A')
	if g == nil {
		t.Fatal("glyph 'A' not found")
	}
	if g.Width != 50 {
		t.Errorf("glyph 'A' width = %d, want 50", g.Width)
	}
}

func TestLoadDistanceFieldFont_MissingLineHeight(t *testing.T) {
	metrics := sdfMetrics{
		Glyphs: []sdfGlyphEntry{
			{ID: 65, X: 0, Y: 0, Width: 10, Height: 10, XAdvance: 10},
		},
	}
	data, _ := json.Marshal(metrics)

	_, err := LoadDistanceFieldFont(data, 0)
	if err == nil {
		t.Error("expected error for missing lineHeight")
	}
}

func TestDistanceFieldFont_MeasureString(t *testing.T) {
	metrics := sdfMetrics{
		Type:          "sdf",
		Size:          80,
		DistanceRange: 8,
		LineHeight:    20,
		Base:          16,
		Glyphs: []sdfGlyphEntry{
			{ID: 72, X: 0, Y: 0, Width: 10, Height: 20, XAdvance: 12}, // H
			{ID: 105, X: 10, Y: 0, Width: 6, Height: 20, XAdvance: 8}, // i
		},
	}
	data, _ := json.Marshal(metrics)

	f, err := LoadDistanceFieldFont(data, 0)
	if err != nil {
		t.Fatalf("LoadDistanceFieldFont: %v", err)
	}

	w, h := f.MeasureString("Hi")
	if w != 20 { // 12 + 8
		t.Errorf("width = %f, want 20", w)
	}
	if h != 20 {
		t.Errorf("height = %f, want 20", h)
	}
}

func TestDistanceFieldFont_Kern(t *testing.T) {
	f := &DistanceFieldFont{}
	if f.Kern('A', 'V') != 0 {
		t.Error("kern with nil map should be 0")
	}
}
