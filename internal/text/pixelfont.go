package text

import "unicode/utf8"

// PixelFont is a pixel-perfect bitmap font renderer for monospaced spritesheets.
// Each character occupies a fixed cell in a grid on a single atlas page.
// Scaling is integer-only (1x, 2x, 3x) to preserve pixel-perfect rendering.
type PixelFont struct {
	CellW, CellH       int // cell size in pixels
	PadTop, PadRight    int // pixels trimmed from each side of each cell
	PadBottom, PadLeft  int
	Page                uint16              // atlas page index
	Cols                int                 // number of columns in the spritesheet grid
	Glyphs              [128]pixelGlyph     // ASCII fast lookup
	GlyphSet            [128]bool           // which ASCII entries are populated
	ExtGlyphs           map[rune]pixelGlyph // extended chars
}

// pixelGlyph stores the atlas position of a single character cell.
type pixelGlyph struct {
	x, y uint16 // pixel position on atlas page
}

// isPixelFont implements the pixelFontMarker interface.
func (pf *PixelFont) isPixelFont() {}

// MeasureString returns the pixel width and height of the rendered text
// in native (unscaled) pixels. Monospaced: each character is cellW wide.
func (pf *PixelFont) MeasureString(text string) (width, height float64) {
	if len(text) == 0 {
		return 0, float64(pf.CellH)
	}

	var maxW int
	var curW int
	lines := 1

	for i := 0; i < len(text); {
		r, sz := utf8.DecodeRuneInString(text[i:])
		i += sz

		if r == '\n' {
			if curW > maxW {
				maxW = curW
			}
			curW = 0
			lines++
			continue
		}
		curW++
	}

	if curW > maxW {
		maxW = curW
	}

	return float64(maxW * pf.AdvanceW()), float64(lines * pf.AdvanceH())
}

// LineHeight returns the effective line height in native pixels (after vertical trim).
func (pf *PixelFont) LineHeight() float64 {
	return float64(pf.AdvanceH())
}

// TrimCell trims pixels from each side of every cell.
func (pf *PixelFont) TrimCell(top, right, bottom, left int) {
	pf.PadTop = top
	pf.PadRight = right
	pf.PadBottom = bottom
	pf.PadLeft = left
}

// AdvanceW returns the effective character advance width after horizontal trim.
func (pf *PixelFont) AdvanceW() int {
	w := pf.CellW - pf.PadLeft - pf.PadRight
	if w < 1 {
		w = 1
	}
	return w
}

// AdvanceH returns the effective line height after vertical trim.
func (pf *PixelFont) AdvanceH() int {
	h := pf.CellH - pf.PadTop - pf.PadBottom
	if h < 1 {
		h = 1
	}
	return h
}

// GlyphLookup returns the atlas position of a glyph, and whether it was found.
func (pf *PixelFont) GlyphLookup(r rune) (x, y uint16, found bool) {
	if r >= 0 && r < 128 {
		if pf.GlyphSet[r] {
			g := pf.Glyphs[r]
			return g.x, g.y, true
		}
		return 0, 0, false
	}
	if g, ok := pf.ExtGlyphs[r]; ok {
		return g.x, g.y, true
	}
	return 0, 0, false
}

// AddGlyph registers a glyph at the given atlas position.
func (pf *PixelFont) AddGlyph(r rune, x, y uint16) {
	pg := pixelGlyph{x: x, y: y}
	if r >= 0 && r < 128 {
		pf.Glyphs[r] = pg
		pf.GlyphSet[r] = true
	} else {
		if pf.ExtGlyphs == nil {
			pf.ExtGlyphs = make(map[rune]pixelGlyph)
		}
		pf.ExtGlyphs[r] = pg
	}
}
