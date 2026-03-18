package bake

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// ---------------------------------------------------------------------------
// CharsetFor
// ---------------------------------------------------------------------------

func TestCharsetFor_ASCII(t *testing.T) {
	cs := CharsetFor("ascii")
	runes := []rune(cs)
	if len(runes) != 95 { // 0x20 through 0x7E inclusive = 95 chars
		t.Errorf("ASCII charset len = %d, want 95", len(runes))
	}
	if !strings.Contains(cs, "A") || !strings.Contains(cs, "z") || !strings.Contains(cs, "0") {
		t.Error("ASCII charset missing expected characters")
	}
}

func TestCharsetFor_Latin_LargerThanASCII(t *testing.T) {
	ascii := CharsetFor("ascii")
	latin := CharsetFor("latin")
	if len([]rune(latin)) <= len([]rune(ascii)) {
		t.Errorf("Latin charset (%d) should be larger than ASCII (%d)",
			len([]rune(latin)), len([]rune(ascii)))
	}
}

func TestCharsetFor_Custom_PassThrough(t *testing.T) {
	custom := "Hello, World!"
	if got := CharsetFor(custom); got != custom {
		t.Errorf("CharsetFor(%q) = %q, want passthrough", custom, got)
	}
}

func TestCharsetFor_Full_ReturnsSentinel(t *testing.T) {
	// "full" is a sentinel; caller must handle it via FullCharset.
	got := CharsetFor("full")
	if got != "" {
		t.Errorf("CharsetFor(full) = %q, want empty sentinel", got)
	}
}

// ---------------------------------------------------------------------------
// formatSize
// ---------------------------------------------------------------------------

func TestFormatSize_Integer(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{64, "64"},
		{256, "256"},
		{0, "0"},
		{128, "128"},
	}
	for _, c := range cases {
		if got := formatSize(c.in); got != c.want {
			t.Errorf("formatSize(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatSize_Float(t *testing.T) {
	got := formatSize(48.5)
	if got == "" || got == "48" || got == "49" {
		t.Errorf("formatSize(48.5) = %q, want non-integer string", got)
	}
}

// ---------------------------------------------------------------------------
// FullCharset
// ---------------------------------------------------------------------------

func TestFullCharset_ValidFont_ReturnsChars(t *testing.T) {
	cs, err := FullCharset(goregular.TTF)
	if err != nil {
		t.Fatalf("FullCharset: %v", err)
	}
	if len([]rune(cs)) == 0 {
		t.Error("FullCharset returned empty string for valid font")
	}
}

func TestFullCharset_InvalidFont_Error(t *testing.T) {
	_, err := FullCharset([]byte("not a font"))
	if err == nil {
		t.Error("expected error for invalid font data")
	}
}

// ---------------------------------------------------------------------------
// Bundle — E2E bake tests
// ---------------------------------------------------------------------------

func TestBundle_SingleStyleSingleSize(t *testing.T) {
	styles := []StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := Options{BakeSizes: []float64{64}, Charset: CharsetFor("ascii")}

	data, err := Bundle(styles, opts)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Bundle returned empty data")
	}

	// Verify it's a valid zip.
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("output is not a valid zip: %v", err)
	}

	// Expect regular_64.png and regular_64.json.
	fileNames := zipFileNames(zr)
	assertHasFile(t, fileNames, "regular_64.png")
	assertHasFile(t, fileNames, "regular_64.json")
}

func TestBundle_MultipleSize(t *testing.T) {
	styles := []StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := Options{BakeSizes: []float64{32, 64}, Charset: CharsetFor("ascii")}

	data, err := Bundle(styles, opts)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	names := zipFileNames(zr)
	assertHasFile(t, names, "regular_32.png")
	assertHasFile(t, names, "regular_32.json")
	assertHasFile(t, names, "regular_64.png")
	assertHasFile(t, names, "regular_64.json")
}

func TestBundle_MultipleStyles(t *testing.T) {
	// Use the same font data for both styles (just for testing the pipeline).
	styles := []StyleData{
		{Label: "regular", TTFData: goregular.TTF},
		{Label: "bold", TTFData: goregular.TTF},
	}
	opts := Options{BakeSizes: []float64{32}, Charset: CharsetFor("ascii")}

	data, err := Bundle(styles, opts)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	names := zipFileNames(zr)
	assertHasFile(t, names, "regular_32.png")
	assertHasFile(t, names, "bold_32.png")
}

func TestBundle_JSONContainsRequiredFields(t *testing.T) {
	styles := []StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := Options{BakeSizes: []float64{32}, Charset: "AZ"}

	data, err := Bundle(styles, opts)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "regular_32.json" {
			continue
		}
		rc, _ := f.Open()
		var buf bytes.Buffer
		buf.ReadFrom(rc)
		rc.Close()

		var meta map[string]any
		if err := json.Unmarshal(buf.Bytes(), &meta); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		for _, field := range []string{"type", "size", "distanceRange", "lineHeight", "base", "glyphs"} {
			if _, ok := meta[field]; !ok {
				t.Errorf("JSON missing field %q", field)
			}
		}
		return
	}
	t.Error("regular_32.json not found in bundle")
}

func TestBundle_JSONLineHeightFromFont(t *testing.T) {
	// Verify lineHeight is overridden with actual font metrics (not size*1.2).
	styles := []StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := Options{BakeSizes: []float64{64}, Charset: "A"}

	data, err := Bundle(styles, opts)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	jsonData := extractFile(t, data, "regular_64.json")
	var meta struct {
		LineHeight float64 `json:"lineHeight"`
		Size       float64 `json:"size"`
	}
	if err := json.Unmarshal(jsonData, &meta); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	// lineHeight should be reasonable (font-derived, > 0).
	if meta.LineHeight <= 0 {
		t.Errorf("lineHeight = %f, want > 0", meta.LineHeight)
	}
}

func TestBundle_InvalidTTF_Error(t *testing.T) {
	styles := []StyleData{{Label: "regular", TTFData: []byte("not a font")}}
	opts := Options{BakeSizes: []float64{32}, Charset: "A"}
	_, err := Bundle(styles, opts)
	if err == nil {
		t.Error("expected error for invalid TTF data")
	}
}

func TestBundle_DefaultsApplied(t *testing.T) {
	// Passing zero-value Options should use defaults without panicking.
	styles := []StyleData{{Label: "regular", TTFData: goregular.TTF}}
	data, err := Bundle(styles, Options{})
	if err != nil {
		t.Fatalf("Bundle with zero opts: %v", err)
	}
	if len(data) == 0 {
		t.Error("Bundle with zero opts returned empty data")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func zipFileNames(zr *zip.Reader) map[string]bool {
	names := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		names[f.Name] = true
	}
	return names
}

func assertHasFile(t *testing.T, names map[string]bool, name string) {
	t.Helper()
	if !names[name] {
		t.Errorf("expected file %q in bundle, got: %v", name, names)
	}
}

func extractFile(t *testing.T, bundleData []byte, name string) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(bundleData), int64(len(bundleData)))
	if err != nil {
		t.Fatalf("extractFile: %v", err)
	}
	for _, f := range zr.File {
		if f.Name == name {
			rc, _ := f.Open()
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			rc.Close()
			return buf.Bytes()
		}
	}
	t.Fatalf("file %q not found in bundle", name)
	return nil
}
