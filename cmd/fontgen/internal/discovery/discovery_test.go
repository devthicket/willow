package discovery

import "testing"

// ---------------------------------------------------------------------------
// classifySubfamily
// ---------------------------------------------------------------------------

func TestClassifySubfamily_Regular(t *testing.T) {
	cases := []string{"Regular", "regular", "", "Book", "Roman"}
	for _, s := range cases {
		if got := classifySubfamily(s); got != StyleRegular {
			t.Errorf("classifySubfamily(%q) = %d, want StyleRegular", s, got)
		}
	}
}

func TestClassifySubfamily_Bold(t *testing.T) {
	cases := []string{"Bold", "bold", "BOLD", "SemiBold", "ExtraBold"}
	for _, s := range cases {
		if got := classifySubfamily(s); got != StyleBold {
			t.Errorf("classifySubfamily(%q) = %d, want StyleBold", s, got)
		}
	}
}

func TestClassifySubfamily_Italic(t *testing.T) {
	cases := []string{"Italic", "italic", "Oblique", "oblique"}
	for _, s := range cases {
		if got := classifySubfamily(s); got != StyleItalic {
			t.Errorf("classifySubfamily(%q) = %d, want StyleItalic", s, got)
		}
	}
}

func TestClassifySubfamily_BoldItalic(t *testing.T) {
	cases := []string{
		"Bold Italic", "bold italic", "BoldItalic", "bolditalic",
		"Bold Oblique", "bold oblique",
	}
	for _, s := range cases {
		if got := classifySubfamily(s); got != StyleBoldItalic {
			t.Errorf("classifySubfamily(%q) = %d, want StyleBoldItalic", s, got)
		}
	}
}

// ---------------------------------------------------------------------------
// searchDirs — smoke test (no assertions on content, just no panic)
// ---------------------------------------------------------------------------

func TestSearchDirs_NoPanic(t *testing.T) {
	dirs := searchDirs()
	if len(dirs) == 0 {
		t.Error("searchDirs returned no directories")
	}
}

// ---------------------------------------------------------------------------
// ByName — graceful behaviour when nothing is found
// ---------------------------------------------------------------------------

func TestByName_UnknownFamily_ReturnsEmptyResults(t *testing.T) {
	// An impossible family name should return empty results without error.
	r, err := ByName("__zzz_no_such_font_family_xyz__")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Results")
	}
	if len(r.Regular)+len(r.Bold)+len(r.Italic)+len(r.BoldItalic) != 0 {
		t.Error("expected no results for unknown family")
	}
}

// ---------------------------------------------------------------------------
// ByPath — error on non-existent file
// ---------------------------------------------------------------------------

func TestByPath_NonExistentFile_Error(t *testing.T) {
	_, err := ByPath("/no/such/font.ttf")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestByPath_NonFontFile_Error(t *testing.T) {
	// Passing a non-font file should return an error.
	_, err := ByPath("/etc/hosts")
	if err == nil {
		t.Error("expected error for non-font file")
	}
}
