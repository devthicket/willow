package text

import (
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewPixelFont creates a pixel font from a spritesheet image.
func NewPixelFont(img *ebiten.Image, cellW, cellH int, chars string) *PixelFont {
	pageIndex := AllocPageFn()
	RegisterPageFn(pageIndex, img)

	bounds := img.Bounds()
	cols := bounds.Dx() / cellW

	pf := &PixelFont{
		CellW: cellW,
		CellH: cellH,
		Page:  uint16(pageIndex),
		Cols:  cols,
	}

	idx := 0
	for i := 0; i < len(chars); {
		r, sz := utf8.DecodeRuneInString(chars[i:])
		i += sz
		col := idx % cols
		row := idx / cols
		gx := uint16(col * cellW)
		gy := uint16(row * cellH)
		pf.AddGlyph(r, gx, gy)
		idx++
	}

	return pf
}
