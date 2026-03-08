package text

// Glyph holds glyph metrics and atlas position.
type Glyph struct {
	ID       rune
	X, Y     uint16
	Width    uint16
	Height   uint16
	XOffset  int16
	YOffset  int16
	XAdvance int16
	Page     uint16
}

const AsciiGlyphCount = 128
