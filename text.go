package willow

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Font is the interface for text measurement and layout.
// Implemented by BitmapFont and TTFFont.
type Font interface {
	// MeasureString returns the pixel width and height of the rendered text,
	// accounting for newlines and the font's line height.
	MeasureString(text string) (width, height float64)
	// LineHeight returns the vertical distance between baselines in pixels.
	LineHeight() float64
}

// --- Outline ---

// Outline defines a text stroke rendered behind the fill.
type Outline struct {
	Color     Color
	Thickness float64
}

// --- TextBlock ---

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock struct {
	// Content is the text string to render. Supports embedded newlines.
	Content string
	// Font is the BitmapFont or TTFFont used for measurement and rendering.
	Font Font
	// Align controls horizontal alignment within the wrap width or measured bounds.
	Align TextAlign
	// WrapWidth is the maximum line width in pixels before word wrapping.
	// Zero means no wrapping.
	WrapWidth float64
	// Color is the fill color for the text glyphs.
	Color Color
	// Outline defines a text stroke rendered behind the fill. Nil means no outline.
	Outline *Outline
	// LineHeight overrides the font's default line height. Zero uses Font.LineHeight().
	LineHeight float64

	// Cached layout (unexported)
	layoutDirty bool
	measuredW   float64
	measuredH   float64
	lines       []textLine // cached line layout
	wordGlyphs  []glyphPos // preallocated word buffer for layoutBitmap

	// TTF rendering cache (unexported)
	ttfImage   *ebiten.Image // cached rendered TTF text
	ttfPage    int           // page index where ttfImage is registered (-1 = unset)
	ttfDirty   bool          // true when TTF cache needs re-render
	ttfWrapped string        // content with word-wrap newlines injected
}

// textLine stores one line of laid-out glyphs.
type textLine struct {
	glyphs []glyphPos
	width  float64
}

// glyphPos is the computed screen position and region for a single glyph.
type glyphPos struct {
	x, y   float64
	region TextureRegion
	page   uint16
}

// Invalidate invalidates the cached layout and TTF image, forcing recomputation
// on the next frame. Call this after changing Content, Font, WrapWidth, Align,
// LineHeight, Color, or Outline at runtime.
func (tb *TextBlock) Invalidate() {
	tb.layoutDirty = true
	tb.ttfDirty = true
}

// lineHeight returns the effective line height for this text block.
func (tb *TextBlock) lineHeight() float64 {
	if tb.LineHeight > 0 {
		return tb.LineHeight
	}
	if tb.Font != nil {
		return tb.Font.LineHeight()
	}
	return 0
}

// layout recomputes glyph positions if dirty. Returns the cached lines.
func (tb *TextBlock) layout() []textLine {
	if !tb.layoutDirty {
		return tb.lines
	}
	tb.layoutDirty = false

	if tb.Font == nil {
		tb.lines = tb.lines[:0]
		tb.measuredW = 0
		tb.measuredH = 0
		return tb.lines
	}

	tb.ttfDirty = true // any layout recompute invalidates TTF cache

	switch f := tb.Font.(type) {
	case *BitmapFont:
		tb.layoutBitmap(f)
	case *TTFFont:
		tb.layoutTTF(f)
	default:
		tb.lines = tb.lines[:0]
		tb.measuredW = 0
		tb.measuredH = 0
	}

	return tb.lines
}

// layoutBitmap computes glyph positions for a BitmapFont.
func (tb *TextBlock) layoutBitmap(f *BitmapFont) {
	lh := tb.lineHeight()
	content := tb.Content

	// Reuse lines slice
	tb.lines = tb.lines[:0]

	var maxW float64
	var curLine textLine

	// Word wrapping state
	var wordStart int
	tb.wordGlyphs = tb.wordGlyphs[:0]
	var wordWidth float64
	var cursorX float64
	var prevRune rune
	var hasPrev bool

	flush := func() {
		if curLine.width > maxW {
			maxW = curLine.width
		}
		tb.lines = append(tb.lines, curLine)
		curLine = textLine{}
		cursorX = 0
		hasPrev = false
	}

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		i += size

		if r == '\n' {
			// Flush word then line
			curLine.glyphs = append(curLine.glyphs, tb.wordGlyphs...)
			curLine.width += wordWidth
			tb.wordGlyphs = tb.wordGlyphs[:0]
			wordWidth = 0
			wordStart = i
			flush()
			prevRune = 0
			hasPrev = false
			continue
		}

		g := f.glyph(r)
		if g == nil {
			hasPrev = false
			continue
		}

		kern := int16(0)
		if hasPrev {
			kern = f.kern(prevRune, r)
		}

		glyphX := cursorX + float64(kern) + float64(g.xOffset)
		glyphY := float64(g.yOffset)

		gp := glyphPos{
			x: glyphX,
			y: glyphY,
			region: TextureRegion{
				Page:      f.page,
				X:         g.x,
				Y:         g.y,
				Width:     g.width,
				Height:    g.height,
				OriginalW: g.width,
				OriginalH: g.height,
			},
			page: f.page,
		}

		advance := float64(g.xAdvance) + float64(kern)

		if r == ' ' {
			// Space: flush word into current line
			curLine.glyphs = append(curLine.glyphs, tb.wordGlyphs...)
			curLine.width += wordWidth
			tb.wordGlyphs = tb.wordGlyphs[:0]
			wordWidth = 0
			wordStart = i

			// Add space glyph
			curLine.glyphs = append(curLine.glyphs, gp)
			curLine.width = cursorX + advance
			cursorX += advance
		} else {
			// Accumulate in word
			tb.wordGlyphs = append(tb.wordGlyphs, gp)
			wordWidth = cursorX + advance - (cursorX - wordWidth)

			// Check wrap
			if tb.WrapWidth > 0 && cursorX+advance > tb.WrapWidth && len(curLine.glyphs) > 0 {
				// Wrap: flush current line without this word
				flush()

				// Recompute word glyph positions from cursor 0
				cursorX = 0
				hasPrev = false
				tb.wordGlyphs = tb.wordGlyphs[:0]
				wordWidth = 0

				// Re-layout from wordStart
				i = wordStart
				continue
			}
			cursorX += advance
		}

		prevRune = r
		hasPrev = true
	}

	// Flush remaining word and line
	curLine.glyphs = append(curLine.glyphs, tb.wordGlyphs...)
	curLine.width = cursorX
	if len(curLine.glyphs) > 0 || len(tb.lines) == 0 {
		if curLine.width > maxW {
			maxW = curLine.width
		}
		tb.lines = append(tb.lines, curLine)
	}

	// Apply text alignment offsets. Use WrapWidth as the reference width when
	// set; otherwise fall back to the widest line (maxW).
	alignW := maxW
	if tb.WrapWidth > 0 {
		alignW = tb.WrapWidth
	}
	for li := range tb.lines {
		line := &tb.lines[li]
		var offsetX float64
		switch tb.Align {
		case TextAlignLeft:
			// No offset needed for left alignment.
		case TextAlignCenter:
			offsetX = (alignW - line.width) / 2
		case TextAlignRight:
			offsetX = alignW - line.width
		}
		if offsetX != 0 {
			for gi := range line.glyphs {
				line.glyphs[gi].x += offsetX
			}
		}
	}

	tb.measuredW = maxW
	tb.measuredH = float64(len(tb.lines)) * lh
}

// layoutTTF computes measured dimensions for a TTFFont, applying word wrapping
// when WrapWidth > 0. The wrapped content (with injected newlines) is stored in
// ttfWrapped for use by emitTTFTextCommand. TTF text renders as a single cached
// image rather than per-glyph sprite commands.
func (tb *TextBlock) layoutTTF(f *TTFFont) {
	tb.lines = tb.lines[:0]

	if tb.WrapWidth <= 0 {
		tb.ttfWrapped = tb.Content
		w, h := f.MeasureString(tb.Content)
		tb.measuredW = w
		tb.measuredH = h
		return
	}

	// Word-wrap: split each paragraph into lines that fit within WrapWidth,
	// preserving original whitespace between words.
	var out strings.Builder
	paragraphs := strings.Split(tb.Content, "\n")
	for pi, para := range paragraphs {
		if pi > 0 {
			out.WriteByte('\n')
		}
		if len(para) == 0 {
			continue
		}
		// Find word spans preserving original spacing.
		type span struct{ start, end int }
		var words []span
		i := 0
		for i < len(para) {
			// Skip spaces.
			for i < len(para) && para[i] == ' ' {
				i++
			}
			if i >= len(para) {
				break
			}
			ws := i
			for i < len(para) && para[i] != ' ' {
				i++
			}
			words = append(words, span{ws, i})
		}
		if len(words) == 0 {
			out.WriteString(para) // all spaces — preserve them
			continue
		}
		lineStart := 0
		for wi := range words {
			// Candidate uses original text to preserve spacing.
			candidate := para[words[lineStart].start:words[wi].end]
			cw, _ := f.MeasureString(candidate)
			if cw > tb.WrapWidth && wi > lineStart {
				// Emit line up to previous word end.
				out.WriteString(para[words[lineStart].start:words[wi-1].end])
				out.WriteByte('\n')
				lineStart = wi
			}
		}
		// Emit remaining words.
		out.WriteString(para[words[lineStart].start:words[len(words)-1].end])
		// Preserve any trailing spaces.
		lastWordEnd := words[len(words)-1].end
		if lastWordEnd < len(para) {
			out.WriteString(para[lastWordEnd:])
		}
	}

	wrapped := out.String()
	tb.ttfWrapped = wrapped
	w, h := f.MeasureString(wrapped)
	tb.measuredW = w
	tb.measuredH = h
}

// --- glyph (internal) ---

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

// --- BitmapFont ---

const asciiGlyphCount = 128

// BitmapFont renders text from pre-rasterized glyph atlases in BMFont format.
type BitmapFont struct {
	lineHeight float64
	base       float64
	page       uint16 // atlas page index

	asciiGlyphs [asciiGlyphCount]glyph // fixed array for ASCII, zero-alloc lookup
	asciiSet    [asciiGlyphCount]bool  // which ASCII entries are populated
	extGlyphs   map[rune]*glyph        // extended Unicode (pointer avoids per-lookup alloc)

	kernings map[[2]rune]int16
}

// MeasureString returns the width and height of the rendered text.
func (f *BitmapFont) MeasureString(s string) (width, height float64) {
	var maxW float64
	var cursorX float64
	var prevRune rune
	var hasPrev bool
	lines := 1

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		if r == '\n' {
			if cursorX > maxW {
				maxW = cursorX
			}
			cursorX = 0
			lines++
			hasPrev = false
			continue
		}

		g := f.glyph(r)
		if g == nil {
			hasPrev = false
			continue
		}

		if hasPrev {
			cursorX += float64(f.kern(prevRune, r))
		}
		cursorX += float64(g.xAdvance)
		prevRune = r
		hasPrev = true
	}

	if cursorX > maxW {
		maxW = cursorX
	}
	return maxW, float64(lines) * f.lineHeight
}

// LineHeight returns the vertical distance between baselines.
func (f *BitmapFont) LineHeight() float64 {
	return f.lineHeight
}

// glyph returns the glyph for the given rune, or nil if not found.
func (f *BitmapFont) glyph(r rune) *glyph {
	if r >= 0 && r < asciiGlyphCount {
		if f.asciiSet[r] {
			return &f.asciiGlyphs[r]
		}
		return nil
	}
	if g, ok := f.extGlyphs[r]; ok {
		return g
	}
	return nil
}

// kern returns the kerning amount for the given rune pair.
func (f *BitmapFont) kern(first, second rune) int16 {
	if f.kernings == nil {
		return 0
	}
	return f.kernings[[2]rune{first, second}]
}

// LoadBitmapFont parses BMFont .fnt text-format data. The page index defaults
// to 0. Register the atlas page image on the Scene via Scene.RegisterPage.
func LoadBitmapFont(fntData []byte) (*BitmapFont, error) {
	return LoadBitmapFontPage(fntData, 0)
}

// LoadBitmapFontPage parses BMFont .fnt text-format data with an explicit page index.
func LoadBitmapFontPage(fntData []byte, pageIndex uint16) (*BitmapFont, error) {
	f := &BitmapFont{
		page: pageIndex,
	}

	scanner := bufio.NewScanner(bytes.NewReader(fntData))
	var charCount int

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		tag, rest := splitTag(line)
		fields := parseFields(rest)

		switch tag {
		case "common":
			if v, ok := fields["lineHeight"]; ok {
				f.lineHeight, _ = strconv.ParseFloat(v, 64)
			}
			if v, ok := fields["base"]; ok {
				f.base, _ = strconv.ParseFloat(v, 64)
			}

		case "char":
			charCount++
			g := glyph{}
			if v, ok := fields["id"]; ok {
				id, _ := strconv.Atoi(v)
				g.id = rune(id)
			}
			if v, ok := fields["x"]; ok {
				val, _ := strconv.Atoi(v)
				g.x = uint16(val)
			}
			if v, ok := fields["y"]; ok {
				val, _ := strconv.Atoi(v)
				g.y = uint16(val)
			}
			if v, ok := fields["width"]; ok {
				val, _ := strconv.Atoi(v)
				g.width = uint16(val)
			}
			if v, ok := fields["height"]; ok {
				val, _ := strconv.Atoi(v)
				g.height = uint16(val)
			}
			if v, ok := fields["xoffset"]; ok {
				val, _ := strconv.Atoi(v)
				g.xOffset = int16(val)
			}
			if v, ok := fields["yoffset"]; ok {
				val, _ := strconv.Atoi(v)
				g.yOffset = int16(val)
			}
			if v, ok := fields["xadvance"]; ok {
				val, _ := strconv.Atoi(v)
				g.xAdvance = int16(val)
			}
			if v, ok := fields["page"]; ok {
				val, _ := strconv.Atoi(v)
				g.page = uint16(val)
			}

			if g.id >= 0 && g.id < asciiGlyphCount {
				f.asciiGlyphs[g.id] = g
				f.asciiSet[g.id] = true
			} else {
				if f.extGlyphs == nil {
					f.extGlyphs = make(map[rune]*glyph)
				}
				g := g // copy for heap allocation
				f.extGlyphs[g.id] = &g
			}

		case "kerning":
			var first, second rune
			var amount int16
			if v, ok := fields["first"]; ok {
				val, _ := strconv.Atoi(v)
				first = rune(val)
			}
			if v, ok := fields["second"]; ok {
				val, _ := strconv.Atoi(v)
				second = rune(val)
			}
			if v, ok := fields["amount"]; ok {
				val, _ := strconv.Atoi(v)
				amount = int16(val)
			}
			if f.kernings == nil {
				f.kernings = make(map[[2]rune]int16)
			}
			f.kernings[[2]rune{first, second}] = amount
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("willow: error reading .fnt data: %w", err)
	}

	if f.lineHeight == 0 {
		return nil, fmt.Errorf("willow: .fnt data missing common lineHeight")
	}
	if charCount == 0 {
		return nil, fmt.Errorf("willow: .fnt data has no char definitions")
	}

	return f, nil
}

// splitTag splits a BMFont line into its tag and the rest of the line.
func splitTag(line string) (string, string) {
	idx := strings.IndexByte(line, ' ')
	if idx == -1 {
		return line, ""
	}
	return line[:idx], line[idx+1:]
}

// parseFields parses "key=value key=value ..." into a map.
func parseFields(s string) map[string]string {
	fields := make(map[string]string)
	for _, part := range strings.Fields(s) {
		eq := strings.IndexByte(part, '=')
		if eq == -1 {
			continue
		}
		key := part[:eq]
		val := part[eq+1:]
		// Strip quotes from values like face="Arial"
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		fields[key] = val
	}
	return fields
}

// --- TTFFont ---

// TTFFont wraps Ebitengine's text/v2 for TrueType font rendering.
type TTFFont struct {
	face   *text.GoTextFace
	source *text.GoTextFaceSource
	size   float64
	lh     float64 // cached line height
}

// LoadTTFFont loads a TrueType font from raw TTF/OTF data at the given size.
func LoadTTFFont(ttfData []byte, size float64) (*TTFFont, error) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(ttfData))
	if err != nil {
		return nil, fmt.Errorf("willow: failed to parse TTF data: %w", err)
	}

	face := &text.GoTextFace{
		Source: source,
		Size:   size,
	}

	// Compute line height from metrics
	m := face.Metrics()
	lh := m.HAscent + m.HDescent + m.HLineGap

	return &TTFFont{
		face:   face,
		source: source,
		size:   size,
		lh:     lh,
	}, nil
}

// MeasureString returns the width and height of the rendered text.
func (f *TTFFont) MeasureString(s string) (width, height float64) {
	w, h := text.Measure(s, f.face, f.lh)
	return w, h
}

// LineHeight returns the vertical distance between baselines.
func (f *TTFFont) LineHeight() float64 {
	return f.lh
}

// Face returns the underlying GoTextFace for direct Ebitengine text/v2 rendering.
func (f *TTFFont) Face() *text.GoTextFace {
	return f.face
}

// --- Text rendering helpers (used by render.go) ---

// emitBitmapTextCommands emits CommandSprite per glyph for a BitmapFont text node.
// worldTransform is the coordinate-space transform for glyph positioning
// (view*world in the main traverse, or an explicit local transform in subtree rendering).
func emitBitmapTextCommands(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	lines := tb.layout()
	if len(lines) == 0 {
		return commands
	}

	lh := tb.lineHeight()
	alpha := n.worldAlpha
	color := color32{
		R: float32(tb.Color.R * n.Color.R),
		G: float32(tb.Color.G * n.Color.G),
		B: float32(tb.Color.B * n.Color.B),
		A: float32(tb.Color.A * n.Color.A * alpha),
	}

	// Outline pass: render glyphs offset in 8 directions with outline color
	if tb.Outline != nil && tb.Outline.Thickness > 0 {
		outColor := color32{
			R: float32(tb.Outline.Color.R * n.Color.R),
			G: float32(tb.Outline.Color.G * n.Color.G),
			B: float32(tb.Outline.Color.B * n.Color.B),
			A: float32(tb.Outline.Color.A * n.Color.A * alpha),
		}
		t := tb.Outline.Thickness
		offsets := [8][2]float64{
			{-t, 0}, {t, 0}, {0, -t}, {0, t},
			{-t, -t}, {t, -t}, {-t, t}, {t, t},
		}
		for _, off := range offsets {
			for li, line := range lines {
				lineY := float64(li) * lh
				for _, gp := range line.glyphs {
					*treeOrder++
					// Compose glyph-local offset into world transform
					glyphTransform := composeGlyphTransform(worldTransform, gp.x+off[0], gp.y+lineY+off[1])
					commands = append(commands, RenderCommand{
						Type:          CommandSprite,
						Transform:     affine32(glyphTransform),
						TextureRegion: gp.region,
						Color:         outColor,
						BlendMode:     n.BlendMode,
						RenderLayer:   n.RenderLayer,
						GlobalOrder:   n.GlobalOrder,
						treeOrder:     *treeOrder,
					})
				}
			}
		}
	}

	// Fill pass: render glyphs at actual positions
	for li, line := range lines {
		lineY := float64(li) * lh
		for _, gp := range line.glyphs {
			*treeOrder++
			glyphTransform := composeGlyphTransform(worldTransform, gp.x, gp.y+lineY)
			commands = append(commands, RenderCommand{
				Type:          CommandSprite,
				Transform:     affine32(glyphTransform),
				TextureRegion: gp.region,
				Color:         color,
				BlendMode:     n.BlendMode,
				RenderLayer:   n.RenderLayer,
				GlobalOrder:   n.GlobalOrder,
				treeOrder:     *treeOrder,
			})
		}
	}

	return commands
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

// emitTTFTextCommand renders TTF text to a cached image and emits a single
// CommandSprite. The image is only re-rendered when the text content changes
// (ttfDirty). May allocate on first render (Ebitengine's internal glyph cache).
func emitTTFTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int, pages []*ebiten.Image, nextPage *int) ([]RenderCommand, []*ebiten.Image) {
	tb.layout() // ensure measured dims are computed
	if tb.measuredW == 0 || tb.measuredH == 0 {
		return commands, pages
	}

	f := tb.Font.(*TTFFont)
	alpha := n.worldAlpha

	// For non-left alignment, use WrapWidth (or measuredW) as the image width
	// so each line can be independently aligned within the image.
	imgW := tb.measuredW
	if tb.Align != TextAlignLeft && tb.WrapWidth > 0 {
		imgW = tb.WrapWidth
	}
	w := int(imgW) + 1
	h := int(tb.measuredH) + 1

	// Re-render only when TTF cache is dirty (content/font/layout changed)
	if tb.ttfDirty || tb.ttfImage == nil {
		tb.ttfDirty = false

		// Reuse or create image
		if tb.ttfImage != nil {
			oldB := tb.ttfImage.Bounds()
			if oldB.Dx() != w || oldB.Dy() != h {
				tb.ttfImage.Deallocate()
				tb.ttfImage = ebiten.NewImage(w, h)
			} else {
				tb.ttfImage.Clear()
			}
		} else {
			tb.ttfImage = ebiten.NewImage(w, h)
		}

		op := &text.DrawOptions{}
		op.ColorScale.Scale(
			float32(tb.Color.R),
			float32(tb.Color.G),
			float32(tb.Color.B),
			float32(tb.Color.A),
		)
		op.LineSpacing = f.lh

		// Use Ebitengine's PrimaryAlign for per-line alignment within the image.
		switch tb.Align {
		case TextAlignCenter:
			op.PrimaryAlign = text.AlignCenter
			op.GeoM.Translate(imgW/2, 0)
		case TextAlignRight:
			op.PrimaryAlign = text.AlignEnd
			op.GeoM.Translate(imgW, 0)
		}

		text.Draw(tb.ttfImage, tb.ttfWrapped, f.face, op)

		// Allocate a page slot once, reuse on subsequent renders
		if tb.ttfPage < 0 {
			tb.ttfPage = *nextPage
			*nextPage = tb.ttfPage + 1
		}
		for len(pages) <= tb.ttfPage {
			pages = append(pages, nil)
		}
		pages[tb.ttfPage] = tb.ttfImage
	}

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:      CommandSprite,
		Transform: affine32(worldTransform),
		TextureRegion: TextureRegion{
			Page:      uint16(tb.ttfPage),
			X:         0,
			Y:         0,
			Width:     uint16(w),
			Height:    uint16(h),
			OriginalW: uint16(w),
			OriginalH: uint16(h),
		},
		Color:       color32{float32(n.Color.R), float32(n.Color.G), float32(n.Color.B), float32(n.Color.A * alpha)},
		BlendMode:   n.BlendMode,
		RenderLayer: n.RenderLayer,
		GlobalOrder: n.GlobalOrder,
		treeOrder:   *treeOrder,
	})

	return commands, pages
}
