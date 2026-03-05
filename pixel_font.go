package willow

import (
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// PixelFont is a pixel-perfect bitmap font renderer for monospaced spritesheets.
// Each character occupies a fixed cell in a grid on a single atlas page.
// Scaling is integer-only (1x, 2x, 3x) to preserve pixel-perfect rendering.
type PixelFont struct {
	cellW, cellH       int // cell size in pixels
	padTop, padRight   int // pixels trimmed from each side of each cell
	padBottom, padLeft int
	page               uint16              // atlas page index
	cols               int                 // number of columns in the spritesheet grid
	glyphs             [128]pixelGlyph     // ASCII fast lookup
	glyphSet           [128]bool           // which ASCII entries are populated
	extGlyphs          map[rune]pixelGlyph // extended chars
}

// pixelGlyph stores the atlas position of a single character cell.
type pixelGlyph struct {
	x, y uint16 // pixel position on atlas page
}

// NewPixelFont creates a pixel font from a spritesheet image.
//
//   - img: the spritesheet image (registered as an atlas page)
//   - cellW, cellH: pixel dimensions of each character cell
//   - chars: string mapping grid positions to characters (left-to-right, top-to-bottom)
//
// Example:
//
//	font := willow.NewPixelFont(sheetImg, 8, 12, " !\"#$%&'()*+,-./0123456789...")
func NewPixelFont(img *ebiten.Image, cellW, cellH int, chars string) *PixelFont {
	am := atlasManager()
	pageIndex := am.AllocPage()
	am.RegisterPage(pageIndex, img)

	bounds := img.Bounds()
	cols := bounds.Dx() / cellW

	pf := &PixelFont{
		cellW: cellW,
		cellH: cellH,
		page:  uint16(pageIndex),
		cols:  cols,
	}

	idx := 0
	for i := 0; i < len(chars); {
		r, sz := utf8.DecodeRuneInString(chars[i:])
		i += sz

		col := idx % cols
		row := idx / cols
		gx := uint16(col * cellW)
		gy := uint16(row * cellH)

		pg := pixelGlyph{x: gx, y: gy}

		if r >= 0 && r < 128 {
			pf.glyphs[r] = pg
			pf.glyphSet[r] = true
		} else {
			if pf.extGlyphs == nil {
				pf.extGlyphs = make(map[rune]pixelGlyph)
			}
			pf.extGlyphs[r] = pg
		}

		idx++
	}

	return pf
}

// MeasureString returns the pixel width and height of the rendered text
// in native (unscaled) pixels. Monospaced: each character is cellW wide.
func (pf *PixelFont) MeasureString(text string) (width, height float64) {
	if len(text) == 0 {
		return 0, float64(pf.cellH)
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

	return float64(maxW * pf.advanceW()), float64(lines * pf.advanceH())
}

// LineHeight returns the effective line height in native pixels (after vertical trim).
func (pf *PixelFont) LineHeight() float64 {
	return float64(pf.advanceH())
}

// TrimCell trims pixels from each side of every cell, reducing the rendered
// glyph region and character spacing. Parameters follow CSS order: top, right,
// bottom, left. For example, a 16x16 cell with TrimCell(0, 4, 0, 4) renders
// 8px-wide glyphs spaced 8px apart.
func (pf *PixelFont) TrimCell(top, right, bottom, left int) {
	pf.padTop = top
	pf.padRight = right
	pf.padBottom = bottom
	pf.padLeft = left
}

// advanceW returns the effective character advance width after horizontal trim.
func (pf *PixelFont) advanceW() int {
	w := pf.cellW - pf.padLeft - pf.padRight
	if w < 1 {
		w = 1
	}
	return w
}

// advanceH returns the effective line height after vertical trim.
func (pf *PixelFont) advanceH() int {
	h := pf.cellH - pf.padTop - pf.padBottom
	if h < 1 {
		h = 1
	}
	return h
}

// glyph returns the pixel glyph for the given rune, and whether it was found.
func (pf *PixelFont) glyph(r rune) (pixelGlyph, bool) {
	if r >= 0 && r < 128 {
		if pf.glyphSet[r] {
			return pf.glyphs[r], true
		}
		return pixelGlyph{}, false
	}
	if g, ok := pf.extGlyphs[r]; ok {
		return g, true
	}
	return pixelGlyph{}, false
}
