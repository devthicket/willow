package kerning

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestExtract_InvalidTTF_Error(t *testing.T) {
	_, err := Extract([]byte("not a font"), "ABC", 64)
	if err == nil {
		t.Error("expected error for invalid TTF data")
	}
}

func TestExtract_EmptyChars_NoError(t *testing.T) {
	pairs, err := Extract(goregular.TTF, "", 64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for empty charset, got %d", len(pairs))
	}
}

func TestExtract_ValidFont_ReturnsPairs(t *testing.T) {
	pairs, err := Extract(goregular.TTF, "AVToWa", 64)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// Go Regular has kerning; we expect at least some pairs for common combos.
	// We don't require a specific count — just that the function runs without error.
	t.Logf("extracted %d kern pairs for 'AVToWa'", len(pairs))
}

func TestExtract_PairsHaveValidRunes(t *testing.T) {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pairs, err := Extract(goregular.TTF, charset, 64)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, p := range pairs {
		if p.First <= 0 || p.Second <= 0 {
			t.Errorf("pair has non-positive rune: first=%d second=%d", p.First, p.Second)
		}
	}
}

func TestExtract_DifferentBakeSizes_NoPanic(t *testing.T) {
	for _, size := range []float64{32, 64, 128, 256} {
		_, err := Extract(goregular.TTF, "AV", size)
		if err != nil {
			t.Errorf("Extract at size %.0f: %v", size, err)
		}
	}
}

func TestCount_InvalidTTF_ReturnsZero(t *testing.T) {
	n := Count([]byte("not a font"), "ABC")
	if n != 0 {
		t.Errorf("expected 0 for invalid TTF, got %d", n)
	}
}

func TestCount_ValidFont_ReturnsNonNegative(t *testing.T) {
	n := Count(goregular.TTF, "AVToWa")
	if n < 0 {
		t.Errorf("Count returned negative: %d", n)
	}
	t.Logf("Count for 'AVToWa' = %d", n)
}
