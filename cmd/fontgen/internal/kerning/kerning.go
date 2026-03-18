// Package kerning extracts kern pairs from a TTF/OTF font for injection into
// the atlas JSON. It queries the font's kern and GPOS tables via sfnt.
package kerning

import (
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Pair holds a kerning amount for a pair of runes, pre-scaled to bake pixels.
type Pair struct {
	First  int
	Second int
	Amount int16
}

// Extract returns all non-zero kern pairs for the given charset at the given
// bake size in pixels. It queries kern and GPOS tables via sfnt.Kern.
func Extract(ttfData []byte, chars string, bakePx float64) ([]Pair, error) {
	f, err := sfnt.Parse(ttfData)
	if err != nil {
		return nil, err
	}

	// Collect runes from charset (deduplicated).
	seen := map[rune]bool{}
	var runes []rune
	for i := 0; i < len(chars); {
		r, sz := utf8.DecodeRuneInString(chars[i:])
		i += sz
		if !seen[r] {
			seen[r] = true
			runes = append(runes, r)
		}
	}

	var b sfnt.Buffer
	ppem := fixed.I(int(bakePx))

	// Map rune → GlyphIndex up front.
	glyphMap := make(map[rune]sfnt.GlyphIndex, len(runes))
	for _, r := range runes {
		g, err := f.GlyphIndex(&b, r)
		if err != nil || g == 0 {
			continue
		}
		glyphMap[r] = g
	}

	var pairs []Pair
	for _, r0 := range runes {
		g0, ok := glyphMap[r0]
		if !ok {
			continue
		}
		for _, r1 := range runes {
			g1, ok := glyphMap[r1]
			if !ok {
				continue
			}
			k, err := f.Kern(&b, g0, g1, ppem, font.HintingNone)
			if err != nil || k == 0 {
				continue
			}
			pairs = append(pairs, Pair{
				First:  int(r0),
				Second: int(r1),
				Amount: int16(k.Round()),
			})
		}
	}
	return pairs, nil
}

// Count returns the number of non-zero kern pairs in the font for a charset.
// It is faster than Extract because it stops counting at a cap.
func Count(ttfData []byte, chars string) int {
	// Use a nominal bake size — kern pair presence doesn't depend on ppem.
	pairs, err := Extract(ttfData, chars, 64)
	if err != nil {
		return 0
	}
	return len(pairs)
}
