package willow

import (
	"github.com/phanxgames/willow/internal/text"
)

// --- Type aliases from internal/text ---

// Font is the interface for text measurement and layout.
type Font = text.Font

// TextEffects configures text effects (outline, glow, shadow) rendered in a
// single shader pass.
type TextEffects = text.TextEffects

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock = text.TextBlock

// textLine stores one line of laid-out glyphs.
type textLine = text.TextLine

// glyphPos is the computed screen position and region for a single glyph.
type glyphPos = text.GlyphPos

// Glyph holds glyph metrics and atlas position.
type Glyph = text.Glyph

// --- glyph (internal, used by font.go / pixel_font.go) ---

// glyph holds per-glyph metrics for the root-owned SpriteFont.
type glyph struct {
	id       rune
	x, y     uint16
	width    uint16
	height   uint16
	xOffset  int16
	yOffset  int16
	xAdvance int16
	page     uint16
}

// asciiGlyphCount is the fixed-size fast-path table size.
const asciiGlyphCount = 128

// --- Text rendering helpers ---

// textBlockFontScale returns the scale factor that maps native atlas pixels to display pixels.
// Delegates to TextBlock.FontScale() from internal/text.
func textBlockFontScale(tb *TextBlock) float64 {
	return tb.FontScale()
}

// composeGlyphTransform creates a world transform for a glyph at the given
// local offset relative to the text node's world transform.
// This is: worldTransform * Translate(localX, localY)
func composeGlyphTransform(world [6]float64, localX, localY float64) [6]float64 {
	return [6]float64{
		world[0], world[1], world[2], world[3],
		world[0]*localX + world[2]*localY + world[4],
		world[1]*localX + world[3]*localY + world[5],
	}
}
