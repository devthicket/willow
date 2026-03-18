package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/devthicket/willow/cmd/fontgen/internal/bake"
	"github.com/devthicket/willow/cmd/fontgen/internal/extract"
	"github.com/devthicket/willow/cmd/fontgen/internal/info"
)

// ---------------------------------------------------------------------------
// Round-trip: bake → extract → verify files on disk
// ---------------------------------------------------------------------------

func TestE2E_BakeExtract_RoundTrip(t *testing.T) {
	styles := []bake.StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := bake.Options{BakeSizes: []float64{32}, Charset: bake.CharsetFor("ascii")}

	bundleData, err := bake.Bundle(styles, opts)
	if err != nil {
		t.Fatalf("bake.Bundle: %v", err)
	}

	outDir := t.TempDir()
	if err := extract.Bundle(bundleData, outDir); err != nil {
		t.Fatalf("extract.Bundle: %v", err)
	}

	// Verify PNG and JSON were written.
	for _, name := range []string{"regular_32.png", "regular_32.json"} {
		path := filepath.Join(outDir, name)
		if fi, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		} else if fi.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Round-trip: bake → info (PrintBundle)
// ---------------------------------------------------------------------------

func TestE2E_BakeInfo_Bundle(t *testing.T) {
	styles := []bake.StyleData{{Label: "regular", TTFData: goregular.TTF}}
	opts := bake.Options{BakeSizes: []float64{32}, Charset: bake.CharsetFor("ascii")}

	bundleData, err := bake.Bundle(styles, opts)
	if err != nil {
		t.Fatalf("bake.Bundle: %v", err)
	}

	// PrintBundle should not error.
	if err := info.PrintBundle(bundleData, "test.fontbundle"); err != nil {
		t.Errorf("info.PrintBundle: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Info on raw TTF
// ---------------------------------------------------------------------------

func TestE2E_InfoTTF(t *testing.T) {
	if err := info.PrintTTF(goregular.TTF, "goregular.ttf"); err != nil {
		t.Errorf("info.PrintTTF: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Multi-style bundle: bake + verify all style slots in zip
// ---------------------------------------------------------------------------

func TestE2E_MultiStyle_Bundle(t *testing.T) {
	styles := []bake.StyleData{
		{Label: "regular", TTFData: goregular.TTF},
		{Label: "bold", TTFData: goregular.TTF},
		{Label: "italic", TTFData: goregular.TTF},
	}
	opts := bake.Options{BakeSizes: []float64{32}, Charset: "Az"}

	bundleData, err := bake.Bundle(styles, opts)
	if err != nil {
		t.Fatalf("bake.Bundle: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(bundleData), int64(len(bundleData)))
	if err != nil {
		t.Fatalf("bundle not a valid zip: %v", err)
	}

	want := []string{
		"regular_32.png", "regular_32.json",
		"bold_32.png", "bold_32.json",
		"italic_32.png", "italic_32.json",
	}
	names := make(map[string]bool)
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, name := range want {
		if !names[name] {
			t.Errorf("missing %q in bundle", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper: main package helper functions
// ---------------------------------------------------------------------------

func TestIsFilePath_Paths(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"/absolute/path.ttf", true},
		{"./relative.ttf", true},
		{"font.ttf", true},
		{"font.otf", true},
		{"font.fontbundle", true},
		{"arial", false},
		{"Helvetica Neue", false},
	}
	for _, c := range cases {
		if got := isFilePath(c.in); got != c.want {
			t.Errorf("isFilePath(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSizesFlag_Valid(t *testing.T) {
	cases := []struct {
		in   string
		want []float64
	}{
		{"64,256", []float64{64, 256}},
		{"32", []float64{32}},
		{"64, 128, 256", []float64{64, 128, 256}},
	}
	for _, c := range cases {
		got, err := parseSizesFlag(c.in)
		if err != nil {
			t.Errorf("parseSizesFlag(%q): %v", c.in, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("parseSizesFlag(%q) len = %d, want %d", c.in, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseSizesFlag(%q)[%d] = %f, want %f", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestParseSizesFlag_EmptyReturnsDefault(t *testing.T) {
	got, err := parseSizesFlag("")
	if err != nil {
		t.Fatalf("parseSizesFlag(''): %v", err)
	}
	if len(got) == 0 {
		t.Error("expected default sizes for empty string")
	}
}

func TestParseSizesFlag_InvalidValue_Error(t *testing.T) {
	_, err := parseSizesFlag("64,notanumber")
	if err == nil {
		t.Error("expected error for invalid size value")
	}
}

func TestResolveCharset_ASCII(t *testing.T) {
	cs, err := resolveCharset("ascii", goregular.TTF)
	if err != nil {
		t.Fatalf("resolveCharset(ascii): %v", err)
	}
	if !strings.Contains(cs, "A") {
		t.Error("ascii charset missing 'A'")
	}
}

func TestResolveCharset_Latin(t *testing.T) {
	cs, err := resolveCharset("latin", goregular.TTF)
	if err != nil {
		t.Fatalf("resolveCharset(latin): %v", err)
	}
	ascii, _ := resolveCharset("ascii", goregular.TTF)
	if len([]rune(cs)) <= len([]rune(ascii)) {
		t.Error("latin charset should be larger than ascii")
	}
}

func TestResolveCharset_Full(t *testing.T) {
	cs, err := resolveCharset("full", goregular.TTF)
	if err != nil {
		t.Fatalf("resolveCharset(full): %v", err)
	}
	if len([]rune(cs)) == 0 {
		t.Error("full charset returned empty string")
	}
}

func TestResolveCharset_FileMissingFile_Error(t *testing.T) {
	_, err := resolveCharset("file:/no/such/file.txt", goregular.TTF)
	if err == nil {
		t.Error("expected error for missing charset file")
	}
}
