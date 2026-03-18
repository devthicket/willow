package info

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// ---------------------------------------------------------------------------
// parseBase
// ---------------------------------------------------------------------------

func TestParseBase_Regular64(t *testing.T) {
	style, size, ok := parseBase("regular_64")
	if !ok {
		t.Fatal("parseBase returned not ok")
	}
	if style != "regular" || size != 64 {
		t.Errorf("got style=%q size=%f, want regular 64", style, size)
	}
}

func TestParseBase_BoldItalic256(t *testing.T) {
	style, size, ok := parseBase("bold_italic_256")
	if !ok {
		t.Fatal("parseBase returned not ok")
	}
	if style != "bold_italic" || size != 256 {
		t.Errorf("got style=%q size=%f, want bold_italic 256", style, size)
	}
}

func TestParseBase_FloatSize(t *testing.T) {
	style, size, ok := parseBase("regular_48.5")
	if !ok {
		t.Fatal("parseBase returned not ok")
	}
	if style != "regular" || size != 48.5 {
		t.Errorf("got style=%q size=%f, want regular 48.5", style, size)
	}
}

func TestParseBase_NoUnderscore_NotOK(t *testing.T) {
	_, _, ok := parseBase("regular")
	if ok {
		t.Error("expected not ok for name with no underscore")
	}
}

func TestParseBase_InvalidSize_NotOK(t *testing.T) {
	_, _, ok := parseBase("regular_notanumber")
	if ok {
		t.Error("expected not ok for non-numeric size")
	}
}

// ---------------------------------------------------------------------------
// sortEntries
// ---------------------------------------------------------------------------

func TestSortEntries_StylePriority(t *testing.T) {
	entries := []entryInfo{
		{style: "bold_italic", size: 64},
		{style: "regular", size: 64},
		{style: "italic", size: 64},
		{style: "bold", size: 64},
	}
	sortEntries(entries)
	want := []string{"regular", "bold", "italic", "bold_italic"}
	for i, e := range entries {
		if e.style != want[i] {
			t.Errorf("entries[%d].style = %q, want %q", i, e.style, want[i])
		}
	}
}

func TestSortEntries_SizeAscendingWithinStyle(t *testing.T) {
	entries := []entryInfo{
		{style: "regular", size: 256},
		{style: "regular", size: 64},
	}
	sortEntries(entries)
	if entries[0].size != 64 {
		t.Errorf("entries[0].size = %f, want 64", entries[0].size)
	}
}

// ---------------------------------------------------------------------------
// PrintTTF — smoke tests
// ---------------------------------------------------------------------------

func TestPrintTTF_ValidFont_NoPanic(t *testing.T) {
	err := PrintTTF(goregular.TTF, "goregular.ttf")
	if err != nil {
		t.Errorf("PrintTTF: %v", err)
	}
}

func TestPrintTTF_InvalidData_Error(t *testing.T) {
	err := PrintTTF([]byte("not a font"), "bad.ttf")
	if err == nil {
		t.Error("expected error for invalid TTF data")
	}
}

// ---------------------------------------------------------------------------
// PrintBundle — smoke tests
// ---------------------------------------------------------------------------

func TestPrintBundle_Basic_NoPanic(t *testing.T) {
	bundle := makeTestBundle(t)
	err := PrintBundle(bundle, "test.fontbundle")
	if err != nil {
		t.Errorf("PrintBundle: %v", err)
	}
}

func TestPrintBundle_InvalidZip_Error(t *testing.T) {
	err := PrintBundle([]byte("not a zip"), "bad.fontbundle")
	if err == nil {
		t.Error("expected error for invalid zip")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeTestBundle(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, _ := zw.Create("regular_64.json")
	meta := map[string]any{
		"type": "sdf", "size": 64.0, "distanceRange": 8.0,
		"lineHeight": 76.8, "base": 64.0,
		"glyphs": []map[string]any{
			{"id": 65, "x": 0, "y": 0, "width": 8, "height": 10,
				"xOffset": 0, "yOffset": 0, "xAdvance": 9},
		},
	}
	data, _ := json.Marshal(meta)
	w.Write(data)

	w2, _ := zw.Create("regular_64.png")
	img := image.NewNRGBA(image.Rect(0, 0, 8, 10))
	png.Encode(w2, img)

	zw.Close()
	return buf.Bytes()
}
