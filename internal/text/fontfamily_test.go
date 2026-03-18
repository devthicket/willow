package text

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// TestMain: wire AllocPageFn / RegisterPageFn for tests that load bundles
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	page := 0
	AllocPageFn = func() int { p := page; page++; return p }
	RegisterPageFn = func(_ int, _ *ebiten.Image) {} // no-op in tests
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// helpers to create minimal distanceFieldFont instances for slot tests
// ---------------------------------------------------------------------------

func minDFF(lineHeight float64) *distanceFieldFont {
	return &distanceFieldFont{lineHeight: lineHeight, fontSize: lineHeight, distanceRange: 8}
}

// ---------------------------------------------------------------------------
// fontStyleSlot.resolve
// ---------------------------------------------------------------------------

func TestFontStyleSlot_ResolvePicksSmallestAboveThreshold(t *testing.T) {
	f64 := minDFF(64)
	f256 := minDFF(256)
	slot := fontStyleSlot{entries: []fontEntry{
		{bake: 64, font: f64},
		{bake: 256, font: f256},
	}}
	if got := slot.resolve(10); got != f64 {
		t.Errorf("expected 64px font, got %p", got)
	}
}

func TestFontStyleSlot_ResolveFallsBackToLargest(t *testing.T) {
	f256 := minDFF(256)
	slot := fontStyleSlot{entries: []fontEntry{
		{bake: 64, font: minDFF(64)},
		{bake: 256, font: f256},
	}}
	if got := slot.resolve(200); got != f256 {
		t.Errorf("expected 256px font (fallback), got %p", got)
	}
}

func TestFontStyleSlot_ResolveExactThreshold(t *testing.T) {
	f64 := minDFF(64)
	slot := fontStyleSlot{entries: []fontEntry{
		{bake: 64, font: f64},
		{bake: 256, font: minDFF(256)},
	}}
	if got := slot.resolve(16); got != f64 {
		t.Errorf("expected 64px font at exact threshold")
	}
}

func TestFontStyleSlot_ResolveEmpty(t *testing.T) {
	if (&fontStyleSlot{}).resolve(10) != nil {
		t.Error("empty slot should resolve to nil")
	}
}

func TestFontStyleSlot_ResolveSingleEntry_AnySize(t *testing.T) {
	f := minDFF(64)
	slot := fontStyleSlot{entries: []fontEntry{{bake: 64, font: f}}}
	for _, ds := range []float64{1, 16, 100, 1000} {
		if got := slot.resolve(ds); got != f {
			t.Errorf("displaySize=%f: expected same font", ds)
		}
	}
}

// ---------------------------------------------------------------------------
// FontFamily.resolveDF — style fallback chain
// ---------------------------------------------------------------------------

func slotWith(f *distanceFieldFont) *fontStyleSlot {
	return &fontStyleSlot{entries: []fontEntry{{bake: 64, font: f}}}
}

func TestFontFamily_ResolveDF_RegularOnly(t *testing.T) {
	reg := minDFF(64)
	ff := &FontFamily{regular: fontStyleSlot{entries: []fontEntry{{bake: 64, font: reg}}}}
	if got := ff.resolveDF(10, false, false); got != reg {
		t.Error("expected regular")
	}
}

func TestFontFamily_ResolveDF_BoldFallsToRegular(t *testing.T) {
	reg := minDFF(64)
	ff := &FontFamily{regular: fontStyleSlot{entries: []fontEntry{{bake: 64, font: reg}}}}
	if got := ff.resolveDF(10, true, false); got != reg {
		t.Error("bold should fall back to regular")
	}
}

func TestFontFamily_ResolveDF_BoldPresent(t *testing.T) {
	bold := minDFF(64)
	ff := &FontFamily{
		regular: fontStyleSlot{entries: []fontEntry{{bake: 64, font: minDFF(64)}}},
		bold:    slotWith(bold),
	}
	if got := ff.resolveDF(10, true, false); got != bold {
		t.Error("expected bold font")
	}
}

func TestFontFamily_ResolveDF_ItalicPresent(t *testing.T) {
	italic := minDFF(64)
	ff := &FontFamily{
		regular: fontStyleSlot{entries: []fontEntry{{bake: 64, font: minDFF(64)}}},
		italic:  slotWith(italic),
	}
	if got := ff.resolveDF(10, false, true); got != italic {
		t.Error("expected italic font")
	}
}

func TestFontFamily_ResolveDF_BoldItalicFallsToBold(t *testing.T) {
	bold := minDFF(64)
	ff := &FontFamily{
		regular: fontStyleSlot{entries: []fontEntry{{bake: 64, font: minDFF(64)}}},
		bold:    slotWith(bold),
		italic:  slotWith(minDFF(64)),
	}
	if got := ff.resolveDF(10, true, true); got != bold {
		t.Error("bold+italic should fall back to bold when no boldItalic slot")
	}
}

func TestFontFamily_ResolveDF_AllSlotsPresent(t *testing.T) {
	reg := minDFF(64)
	bold := minDFF(64)
	italic := minDFF(64)
	bi := minDFF(64)
	ff := &FontFamily{
		regular:    fontStyleSlot{entries: []fontEntry{{bake: 64, font: reg}}},
		bold:       slotWith(bold),
		italic:     slotWith(italic),
		boldItalic: slotWith(bi),
	}
	if got := ff.resolveDF(10, false, false); got != reg {
		t.Error("expected regular")
	}
	if got := ff.resolveDF(10, true, false); got != bold {
		t.Error("expected bold")
	}
	if got := ff.resolveDF(10, false, true); got != italic {
		t.Error("expected italic")
	}
	if got := ff.resolveDF(10, true, true); got != bi {
		t.Error("expected boldItalic")
	}
}

// ---------------------------------------------------------------------------
// bundleBaseName
// ---------------------------------------------------------------------------

func TestBundleBaseName_Regular64(t *testing.T) {
	bake, style, err := bundleBaseName("regular_64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bake != 64 || style != "regular" {
		t.Errorf("got bake=%f style=%q, want 64 regular", bake, style)
	}
}

func TestBundleBaseName_BoldItalic256(t *testing.T) {
	bake, style, err := bundleBaseName("bold_italic_256")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bake != 256 || style != "bold_italic" {
		t.Errorf("got bake=%f style=%q, want 256 bold_italic", bake, style)
	}
}

func TestBundleBaseName_FloatSize(t *testing.T) {
	bake, style, err := bundleBaseName("regular_48.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bake != 48.5 || style != "regular" {
		t.Errorf("got bake=%f style=%q, want 48.5 regular", bake, style)
	}
}

func TestBundleBaseName_NoUnderscore_Error(t *testing.T) {
	if _, _, err := bundleBaseName("regular"); err == nil {
		t.Error("expected error for name with no underscore")
	}
}

func TestBundleBaseName_InvalidSize_Error(t *testing.T) {
	if _, _, err := bundleBaseName("regular_notanumber"); err == nil {
		t.Error("expected error for non-numeric size")
	}
}

// ---------------------------------------------------------------------------
// NewFontFamilyFromFontBundle
// ---------------------------------------------------------------------------

func TestNewFontFamilyFromFontBundle_Basic(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{{"regular", 64}})
	ff, err := NewFontFamilyFromFontBundle(bundle)
	if err != nil {
		t.Fatalf("NewFontFamilyFromFontBundle: %v", err)
	}
	if len(ff.regular.entries) != 1 {
		t.Errorf("regular entries = %d, want 1", len(ff.regular.entries))
	}
	if ff.regular.entries[0].bake != 64 {
		t.Errorf("bake = %f, want 64", ff.regular.entries[0].bake)
	}
}

func TestNewFontFamilyFromFontBundle_AllStyles(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{
		{"regular", 64}, {"bold", 64}, {"italic", 64}, {"bold_italic", 64},
	})
	ff, err := NewFontFamilyFromFontBundle(bundle)
	if err != nil {
		t.Fatalf("NewFontFamilyFromFontBundle: %v", err)
	}
	if ff.bold == nil {
		t.Error("expected bold slot")
	}
	if ff.italic == nil {
		t.Error("expected italic slot")
	}
	if ff.boldItalic == nil {
		t.Error("expected boldItalic slot")
	}
}

func TestNewFontFamilyFromFontBundle_MultipleSizes_SortedAscending(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{{"regular", 256}, {"regular", 64}})
	ff, err := NewFontFamilyFromFontBundle(bundle)
	if err != nil {
		t.Fatalf("NewFontFamilyFromFontBundle: %v", err)
	}
	if len(ff.regular.entries) != 2 {
		t.Fatalf("regular entries = %d, want 2", len(ff.regular.entries))
	}
	if ff.regular.entries[0].bake > ff.regular.entries[1].bake {
		t.Error("entries not sorted ascending by bake size")
	}
}

func TestNewFontFamilyFromFontBundle_NoRegular_Error(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{{"bold", 64}})
	if _, err := NewFontFamilyFromFontBundle(bundle); err == nil {
		t.Error("expected error when no regular style")
	}
}

func TestNewFontFamilyFromFontBundle_MissingPNG_Error(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("regular_64.json")
	w.Write(minAtlasJSON(64))
	zw.Close()

	if _, err := NewFontFamilyFromFontBundle(buf.Bytes()); err == nil {
		t.Error("expected error for missing PNG")
	}
}

func TestNewFontFamilyFromFontBundle_InvalidZip_Error(t *testing.T) {
	if _, err := NewFontFamilyFromFontBundle([]byte("not a zip")); err == nil {
		t.Error("expected error for invalid zip data")
	}
}

func TestNewFontFamilyFromFontBundle_UnknownStyle_Error(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{{"exotic", 64}})
	if _, err := NewFontFamilyFromFontBundle(bundle); err == nil {
		t.Error("expected error for unknown style")
	}
}

func TestNewFontFamilyFromFontBundle_ResolveDF_IntegrationCheck(t *testing.T) {
	bundle := buildBundle(t, []bundleEntry{{"regular", 64}, {"bold", 64}})
	ff, err := NewFontFamilyFromFontBundle(bundle)
	if err != nil {
		t.Fatalf("NewFontFamilyFromFontBundle: %v", err)
	}
	if f := ff.resolveDF(12, false, false); f == nil {
		t.Error("resolveDF(regular) returned nil")
	}
	if f := ff.resolveDF(12, true, false); f == nil {
		t.Error("resolveDF(bold) returned nil")
	}
}

// ---------------------------------------------------------------------------
// NewFontFamilyFromTTF — config validation
// ---------------------------------------------------------------------------

func TestNewFontFamilyFromTTF_EmptyRegular_Error(t *testing.T) {
	_, err := NewFontFamilyFromTTF(FontFamilyConfig{})
	if err == nil {
		t.Error("expected error for empty Regular")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type bundleEntry struct {
	style string
	size  float64
}

func buildBundle(t *testing.T, entries []bundleEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		base := fmt.Sprintf("%s_%.0f", e.style, e.size)
		w, _ := zw.Create(base + ".json")
		w.Write(minAtlasJSON(e.size))
		w2, _ := zw.Create(base + ".png")
		w2.Write(minPNG(t))
	}
	zw.Close()
	return buf.Bytes()
}

func minAtlasJSON(size float64) []byte {
	m := sdfMetrics{
		Type:          "sdf",
		Size:          size,
		DistanceRange: size * 0.125,
		LineHeight:    size * 1.2,
		Base:          size,
		Glyphs:        []sdfGlyphEntry{{ID: 65, Width: 8, Height: 10, XAdvance: 9}},
	}
	data, _ := json.Marshal(m)
	return data
}

func minPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("minPNG: %v", err)
	}
	return buf.Bytes()
}
