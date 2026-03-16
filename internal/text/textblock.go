package text

import (
	"math"
	"unicode/utf8"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock struct {
	// Content is the text string to render. Supports embedded newlines.
	Content string
	// Font is the font used for measurement and rendering.
	Font Font
	// FontSize is the desired display size in pixels. Applied at render time as a
	// scale factor (FontSize / Font.LineHeight()), independent of ScaleX/ScaleY.
	// Defaults to 16. Set to 0 or negative to use the font's native atlas size.
	FontSize float64
	// Align controls horizontal alignment within the wrap width or measured bounds.
	Align types.TextAlign
	// WrapWidth is the maximum line width in screen pixels before word wrapping.
	// Internally converted to atlas pixels via FontScale. Zero means no wrapping.
	WrapWidth float64
	// Color is the fill color for the text glyphs.
	Color types.Color
	// LineHeight overrides the font's default line height. Zero uses Font.LineHeight().
	LineHeight float64
	// Sharpness tightens the SDF edge smoothstep window. 0.0 = default rendering,
	// 1.0 = sharpest (half the default smoothing width). Values around 0.5–0.7
	// give crisp edges without aliasing on most display densities.
	Sharpness float64

	// Cached layout (unexported)
	LayoutDirty bool
	MeasuredW   float64
	MeasuredH   float64
	Lines       []TextLine // cached line layout

	// SDF rendering fields
	TextEffects   *TextEffects    // nil = no effects (plain fill)
	SdfVerts      []ebiten.Vertex // local-space glyph quads, rebuilt on layout change
	SdfInds       []uint16        // glyph index buffer
	SdfVertCount  int             // active vertex count
	SdfIndCount   int             // active index count
	SdfUniforms   map[string]any  // cached uniform map (allocated once, updated in place)
	SdfShader     *ebiten.Shader  // cached SDF vs MSDF shader selection
	UniformsDirty bool            // triggers uniform rebuild
}

// TextLine stores one line of laid-out glyphs.
type TextLine struct {
	Glyphs []GlyphPos
	Width  float64
}

// GlyphPos is the computed screen position and region for a single glyph.
type GlyphPos struct {
	X, Y   float64
	Region types.TextureRegion
	Page   uint16
}

// Invalidate invalidates the cached layout and SDF image, forcing recomputation
// on the next frame.
func (tb *TextBlock) Invalidate() {
	tb.LayoutDirty = true
	tb.UniformsDirty = true
	tb.SdfVertCount = 0
}

// EffectiveLineHeight returns the effective line height in atlas pixels.
func (tb *TextBlock) EffectiveLineHeight() float64 {
	if tb.LineHeight > 0 {
		return tb.LineHeight
	}
	if tb.Font != nil {
		return tb.Font.LineHeight()
	}
	return 0
}

// FontScale returns the scale factor that maps native atlas pixels to display pixels.
// Returns 1.0 when FontSize is zero/negative or Font is nil.
// pixelFont should be true if the font is a PixelFont (integer scaling).
func (tb *TextBlock) FontScale() float64 {
	if tb.Font == nil {
		return 1.0
	}
	if IsPixelFont(tb.Font) {
		if tb.FontSize <= 0 {
			return 1.0
		}
		native := tb.Font.LineHeight()
		if native <= 0 {
			return 1.0
		}
		s := math.Round(tb.FontSize / native)
		if s < 1 {
			s = 1
		}
		return s
	}
	if tb.FontSize <= 0 {
		return 1.0
	}
	native := tb.Font.LineHeight()
	if native <= 0 {
		return 1.0
	}
	return tb.FontSize / native
}

// MeasureDisplay returns the display-pixel width and height of the rendered text,
// accounting for FontSize scaling.
func (tb *TextBlock) MeasureDisplay(text string) (width, height float64) {
	if tb.Font == nil {
		return 0, 0
	}
	w, h := tb.Font.MeasureString(text)
	fs := tb.FontScale()
	return w * fs, h * fs
}

// Layout recomputes glyph positions if dirty. Returns the cached lines.
// Dispatches to LayoutSDF or LayoutPixel based on font type.
func (tb *TextBlock) Layout() []TextLine {
	if !tb.LayoutDirty {
		return tb.Lines
	}

	if tb.Font == nil {
		tb.LayoutDirty = false
		tb.Lines = tb.Lines[:0]
		tb.MeasuredW = 0
		tb.MeasuredH = 0
		return tb.Lines
	}

	switch f := tb.Font.(type) {
	case *DistanceFieldFont:
		tb.LayoutSDF(f.GlyphLookup, f.Kern, f.Page())
	case *PixelFont:
		advW := f.CellW - f.PadLeft - f.PadRight
		advH := f.CellH - f.PadTop - f.PadBottom
		tb.LayoutPixel(f.GlyphLookup, f.Page, f.CellW, f.CellH, f.PadLeft, f.PadTop, advW, advH)
	}
	tb.LayoutDirty = false

	return tb.Lines
}

// LayoutSDF computes glyph positions for an SDF font (DistanceFieldFont).
func (tb *TextBlock) LayoutSDF(glyphLookup func(r rune) *Glyph, kernLookup func(first, second rune) int16, page uint16) {
	lh := tb.EffectiveLineHeight()
	content := tb.Content

	fs := tb.FontScale()
	atlasWrapWidth := 0.0
	if tb.WrapWidth > 0 && fs > 0 {
		atlasWrapWidth = tb.WrapWidth / fs
	}

	tb.Lines = tb.Lines[:0]

	var maxW float64
	var curLine TextLine

	var wordStart int
	var wordGlyphs []GlyphPos
	var wordWidth float64
	var cursorX float64
	var prevRune rune
	var hasPrev bool

	flush := func() {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
		curLine = TextLine{}
		cursorX = 0
		hasPrev = false
	}

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		i += size

		if r == '\n' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			curLine.Width += wordWidth
			wordGlyphs = wordGlyphs[:0]
			wordWidth = 0
			wordStart = i
			flush()
			prevRune = 0
			hasPrev = false
			continue
		}

		g := glyphLookup(r)
		if g == nil {
			hasPrev = false
			continue
		}

		kern := int16(0)
		if hasPrev {
			kern = kernLookup(prevRune, r)
		}

		glyphX := cursorX + float64(kern) + float64(g.XOffset)
		glyphY := float64(g.YOffset)

		gp := GlyphPos{
			X: glyphX,
			Y: glyphY,
			Region: types.TextureRegion{
				Page:      page,
				X:         g.X,
				Y:         g.Y,
				Width:     g.Width,
				Height:    g.Height,
				OriginalW: g.Width,
				OriginalH: g.Height,
			},
			Page: page,
		}

		advance := float64(g.XAdvance) + float64(kern)

		if r == ' ' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			curLine.Width += wordWidth
			wordGlyphs = wordGlyphs[:0]
			wordWidth = 0
			wordStart = i

			curLine.Glyphs = append(curLine.Glyphs, gp)
			curLine.Width = cursorX + advance
			cursorX += advance
		} else {
			wordGlyphs = append(wordGlyphs, gp)
			wordWidth = cursorX + advance - (cursorX - wordWidth)

			if atlasWrapWidth > 0 && cursorX+advance > atlasWrapWidth && len(curLine.Glyphs) > 0 {
				flush()

				cursorX = 0
				hasPrev = false
				wordGlyphs = wordGlyphs[:0]
				wordWidth = 0

				i = wordStart
				continue
			}
			cursorX += advance
		}

		prevRune = r
		hasPrev = true
	}

	curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
	curLine.Width = cursorX
	if len(curLine.Glyphs) > 0 || len(tb.Lines) == 0 {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
	}

	alignW := maxW
	if atlasWrapWidth > 0 {
		alignW = atlasWrapWidth
	}
	tb.applyAlignment(alignW)

	tb.MeasuredW = maxW
	tb.MeasuredH = float64(len(tb.Lines)) * lh
}

// LayoutPixel computes glyph positions for a pixel font (monospaced grid).
func (tb *TextBlock) LayoutPixel(glyphLookup func(r rune) (x, y uint16, found bool), page uint16, cellW, cellH, padLeft, padTop, advW, advH int) {
	lh := float64(advH)
	content := tb.Content

	srcX := padLeft
	srcY := padTop
	srcW := advW
	srcH := advH

	scale := tb.FontScale()
	atlasWrapWidth := 0.0
	if tb.WrapWidth > 0 && scale > 0 {
		atlasWrapWidth = tb.WrapWidth / scale
	}

	maxCharsPerLine := 0
	if atlasWrapWidth > 0 {
		maxCharsPerLine = int(atlasWrapWidth / float64(advW))
		if maxCharsPerLine < 1 {
			maxCharsPerLine = 1
		}
	}

	tb.Lines = tb.Lines[:0]

	var maxW float64
	var curLine TextLine
	var wordGlyphs []GlyphPos
	var wordLen int
	var lineCharCount int

	flush := func() {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
		curLine = TextLine{}
		lineCharCount = 0
	}

	advWf := float64(advW)

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		i += size

		if r == '\n' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			curLine.Width = float64(lineCharCount+wordLen) * advWf
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0
			flush()
			continue
		}

		if r == ' ' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0

			lineCharCount++
			curLine.Width = float64(lineCharCount) * advWf
			continue
		}

		gx, gy, found := glyphLookup(r)
		if !found {
			continue
		}

		gp := GlyphPos{
			X: float64(lineCharCount+wordLen) * advWf,
			Y: 0,
			Region: types.TextureRegion{
				Page:      page,
				X:         gx + uint16(srcX),
				Y:         gy + uint16(srcY),
				Width:     uint16(srcW),
				Height:    uint16(srcH),
				OriginalW: uint16(srcW),
				OriginalH: uint16(srcH),
			},
			Page: page,
		}

		wordGlyphs = append(wordGlyphs, gp)
		wordLen++

		if maxCharsPerLine > 0 && lineCharCount+wordLen > maxCharsPerLine && lineCharCount > 0 {
			flush()
			for gi := range wordGlyphs {
				wordGlyphs[gi].X = float64(gi) * advWf
			}
		}
	}

	curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
	lineCharCount += wordLen
	curLine.Width = float64(lineCharCount) * advWf
	if len(curLine.Glyphs) > 0 || len(tb.Lines) == 0 {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
	}

	alignW := maxW
	if atlasWrapWidth > 0 {
		alignW = atlasWrapWidth
	}
	tb.applyAlignment(alignW)

	tb.MeasuredW = maxW
	tb.MeasuredH = float64(len(tb.Lines)) * lh
}

// applyAlignment applies horizontal text alignment to all lines.
func (tb *TextBlock) applyAlignment(alignW float64) {
	for li := range tb.Lines {
		line := &tb.Lines[li]
		var offsetX float64
		switch tb.Align {
		case types.TextAlignLeft:
		case types.TextAlignCenter:
			offsetX = (alignW - line.Width) / 2
		case types.TextAlignRight:
			offsetX = alignW - line.Width
		}
		if offsetX != 0 {
			for gi := range line.Glyphs {
				line.Glyphs[gi].X += offsetX
			}
		}
	}
}

// RebuildLocalVerts builds local-space vertex/index buffers from the laid-out
// glyph positions.
func (tb *TextBlock) RebuildLocalVerts(lineHeight float64) {
	lines := tb.Lines

	glyphCount := 0
	for _, line := range lines {
		glyphCount += len(line.Glyphs)
	}

	vertCount := glyphCount * 4
	indCount := glyphCount * 6
	if cap(tb.SdfVerts) < vertCount {
		tb.SdfVerts = make([]ebiten.Vertex, vertCount)
	}
	tb.SdfVerts = tb.SdfVerts[:vertCount]
	if cap(tb.SdfInds) < indCount {
		tb.SdfInds = make([]uint16, indCount)
	}
	tb.SdfInds = tb.SdfInds[:indCount]

	vi := 0
	ii := 0
	for li, line := range lines {
		lineY := float64(li) * lineHeight
		for _, gp := range line.Glyphs {
			dx := float32(gp.X)
			dy := float32(gp.Y + lineY)
			dw := float32(gp.Region.Width)
			dh := float32(gp.Region.Height)

			sx := float32(gp.Region.X)
			sy := float32(gp.Region.Y)
			sw := float32(gp.Region.Width)
			sh := float32(gp.Region.Height)

			base := uint16(vi)
			tb.SdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy,
				SrcX: sx, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.SdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy,
				SrcX: sx + sw, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.SdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy + dh,
				SrcX: sx, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.SdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy + dh,
				SrcX: sx + sw, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++

			tb.SdfInds[ii] = base
			tb.SdfInds[ii+1] = base + 1
			tb.SdfInds[ii+2] = base + 2
			tb.SdfInds[ii+3] = base + 1
			tb.SdfInds[ii+4] = base + 3
			tb.SdfInds[ii+5] = base + 2
			ii += 6
		}
	}
	tb.SdfVertCount = vi
	tb.SdfIndCount = ii
}

// pixelFontMarker is used to identify PixelFont without importing.
type pixelFontMarker interface {
	isPixelFont()
}

// IsPixelFont returns true if the font implements the pixelFontMarker interface.
func IsPixelFont(f Font) bool {
	_, ok := f.(pixelFontMarker)
	return ok
}
