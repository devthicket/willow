package willow

import (
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/text"
)

// PixelFont is a pixel-perfect bitmap font renderer for monospaced spritesheets.
// Aliased from internal/text.PixelFont.
type PixelFont = text.PixelFont

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
