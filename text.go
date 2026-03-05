package willow

import (
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// Font is the interface for text measurement and layout.
// Implemented by SpriteFont.
//
// All measurements are in native atlas pixels. To get display-sized values,
// use TextBlock.MeasureDisplay or scale manually by TextBlock.fontScale().
type Font interface {
	// MeasureString returns the pixel width and height of the rendered text
	// in native atlas pixels, accounting for newlines and the font's line height.
	MeasureString(text string) (width, height float64)
	// LineHeight returns the vertical distance between baselines in native atlas pixels.
	LineHeight() float64
}

// --- TextEffects ---

// TextEffects configures text effects (outline, glow, shadow) rendered in a
// single shader pass. All widths are in distance-field units (relative to the
// font's DistanceRange).
type TextEffects struct {
	OutlineWidth   float64 // 0 = no outline
	OutlineColor   Color
	GlowWidth      float64 // 0 = no glow
	GlowColor      Color
	ShadowOffset   Vec2 // (0,0) = no shadow
	ShadowColor    Color
	ShadowSoftness float64
}

// --- TextBlock ---

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock struct {
	// Content is the text string to render. Supports embedded newlines.
	Content string
	// Font is the SpriteFont used for measurement and rendering.
	Font Font
	// FontSize is the desired display size in pixels. Applied at render time as a
	// scale factor (FontSize / Font.LineHeight()), independent of ScaleX/ScaleY.
	// Defaults to 16. Set to 0 or negative to use the font's native atlas size.
	FontSize float64
	// Align controls horizontal alignment within the wrap width or measured bounds.
	Align TextAlign
	// WrapWidth is the maximum line width in screen pixels before word wrapping.
	// Internally converted to atlas pixels via fontScale. Zero means no wrapping.
	WrapWidth float64
	// Color is the fill color for the text glyphs.
	Color Color
	// LineHeight overrides the font's default line height. Zero uses Font.LineHeight().
	LineHeight float64

	// Cached layout (unexported)
	layoutDirty bool
	measuredW   float64
	measuredH   float64
	lines       []textLine // cached line layout

	// SDF rendering fields (unexported)
	TextEffects   *TextEffects    // nil = no effects (plain fill)
	sdfVerts      []ebiten.Vertex // local-space glyph quads, rebuilt on layout change
	sdfInds       []uint16        // glyph index buffer
	sdfVertCount  int             // active vertex count
	sdfIndCount   int             // active index count
	sdfUniforms   map[string]any  // cached uniform map (allocated once, updated in place)
	sdfShader     *ebiten.Shader  // cached SDF vs MSDF shader selection
	uniformsDirty bool            // triggers uniform rebuild
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

// Invalidate invalidates the cached layout and SDF image, forcing recomputation
// on the next frame. Call this after changing Content, Font, FontSize, WrapWidth,
// Align, LineHeight, Color, or TextEffects at runtime.
func (tb *TextBlock) Invalidate() {
	tb.layoutDirty = true
	tb.uniformsDirty = true
}

// lineHeight returns the effective line height for this text block in atlas pixels.
func (tb *TextBlock) lineHeight() float64 {
	if tb.LineHeight > 0 {
		return tb.LineHeight
	}
	if tb.Font != nil {
		return tb.Font.LineHeight()
	}
	return 0
}

// fontScale returns the scale factor that maps native atlas pixels to display pixels.
// Returns 1.0 when FontSize is zero/negative or Font is nil.
func (tb *TextBlock) fontScale() float64 {
	if _, ok := tb.Font.(*PixelFont); ok {
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
	if tb.FontSize <= 0 || tb.Font == nil {
		return 1.0
	}
	native := tb.Font.LineHeight()
	if native <= 0 {
		return 1.0
	}
	return tb.FontSize / native
}

// MeasureDisplay returns the display-pixel width and height of the rendered text,
// accounting for FontSize scaling. Equivalent to Font.MeasureString scaled by fontScale().
func (tb *TextBlock) MeasureDisplay(text string) (width, height float64) {
	if tb.Font == nil {
		return 0, 0
	}
	w, h := tb.Font.MeasureString(text)
	fs := tb.fontScale()
	return w * fs, h * fs
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

	switch f := tb.Font.(type) {
	case *SpriteFont:
		tb.layoutSDF(f)
		tb.rebuildLocalVerts(f)
		tb.uniformsDirty = true
	case *PixelFont:
		tb.layoutPixel(f)
		tb.rebuildPixelVerts(f)
	default:
		tb.lines = tb.lines[:0]
		tb.measuredW = 0
		tb.measuredH = 0
	}

	return tb.lines
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

const asciiGlyphCount = 128

// --- Text rendering helpers (used by render.go) ---

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

// layoutSDF computes glyph positions for an SpriteFont.
func (tb *TextBlock) layoutSDF(f *SpriteFont) {
	lh := tb.lineHeight()
	content := tb.Content

	// Convert screen-pixel WrapWidth to atlas pixels.
	fs := tb.fontScale()
	atlasWrapWidth := 0.0
	if tb.WrapWidth > 0 && fs > 0 {
		atlasWrapWidth = tb.WrapWidth / fs
	}

	tb.lines = tb.lines[:0]

	var maxW float64
	var curLine textLine

	var wordStart int
	var wordGlyphs []glyphPos
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
			curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
			curLine.width += wordWidth
			wordGlyphs = wordGlyphs[:0]
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
			curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
			curLine.width += wordWidth
			wordGlyphs = wordGlyphs[:0]
			wordWidth = 0
			wordStart = i

			curLine.glyphs = append(curLine.glyphs, gp)
			curLine.width = cursorX + advance
			cursorX += advance
		} else {
			wordGlyphs = append(wordGlyphs, gp)
			wordWidth = cursorX + advance - (cursorX - wordWidth)

			if atlasWrapWidth > 0 && cursorX+advance > atlasWrapWidth && len(curLine.glyphs) > 0 {
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

	curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
	curLine.width = cursorX
	if len(curLine.glyphs) > 0 || len(tb.lines) == 0 {
		if curLine.width > maxW {
			maxW = curLine.width
		}
		tb.lines = append(tb.lines, curLine)
	}

	alignW := maxW
	if atlasWrapWidth > 0 {
		alignW = atlasWrapWidth
	}
	for li := range tb.lines {
		line := &tb.lines[li]
		var offsetX float64
		switch tb.Align {
		case TextAlignLeft:
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

// rebuildLocalVerts builds local-space vertex/index buffers from the laid-out
// glyph positions. The atlas glyph regions already include distanceRange+1
// pixels of SDF padding, so no per-glyph expansion is needed  -  the shader
// has sufficient distance-field data for outline, glow, and shadow effects
// within the atlas's distance range.
func (tb *TextBlock) rebuildLocalVerts(f *SpriteFont) {
	lines := tb.lines
	lh := tb.lineHeight()

	glyphCount := 0
	for _, line := range lines {
		glyphCount += len(line.glyphs)
	}

	// Grow buffers to high-water mark.
	vertCount := glyphCount * 4
	indCount := glyphCount * 6
	if cap(tb.sdfVerts) < vertCount {
		tb.sdfVerts = make([]ebiten.Vertex, vertCount)
	}
	tb.sdfVerts = tb.sdfVerts[:vertCount]
	if cap(tb.sdfInds) < indCount {
		tb.sdfInds = make([]uint16, indCount)
	}
	tb.sdfInds = tb.sdfInds[:indCount]

	vi := 0
	ii := 0
	for li, line := range lines {
		lineY := float64(li) * lh
		for _, gp := range line.glyphs {
			// Destination in local atlas-pixel space.
			dx := float32(gp.x)
			dy := float32(gp.y + lineY)
			dw := float32(gp.region.Width)
			dh := float32(gp.region.Height)

			// Source in atlas pixel coords (1:1 mapping, no expansion).
			sx := float32(gp.region.X)
			sy := float32(gp.region.Y)
			sw := float32(gp.region.Width)
			sh := float32(gp.region.Height)

			base := uint16(vi)
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy,
				SrcX: sx, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy,
				SrcX: sx + sw, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy + dh,
				SrcX: sx, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy + dh,
				SrcX: sx + sw, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++

			tb.sdfInds[ii] = base
			tb.sdfInds[ii+1] = base + 1
			tb.sdfInds[ii+2] = base + 2
			tb.sdfInds[ii+3] = base + 1
			tb.sdfInds[ii+4] = base + 3
			tb.sdfInds[ii+5] = base + 2
			ii += 6
		}
	}
	tb.sdfVertCount = vi
	tb.sdfIndCount = ii
}

// --- Pixel font layout and rendering ---

// layoutPixel computes glyph positions for a PixelFont (monospaced grid).
func (tb *TextBlock) layoutPixel(f *PixelFont) {
	lh := float64(f.advanceH())
	advW := float64(f.advanceW())
	content := tb.Content

	// Trimmed source region dimensions.
	srcX := f.padLeft
	srcY := f.padTop
	srcW := f.advanceW()
	srcH := f.advanceH()

	// Convert screen-pixel WrapWidth to native pixels.
	scale := tb.fontScale()
	atlasWrapWidth := 0.0
	if tb.WrapWidth > 0 && scale > 0 {
		atlasWrapWidth = tb.WrapWidth / scale
	}

	// Maximum characters per line for word wrapping.
	maxCharsPerLine := 0
	if atlasWrapWidth > 0 {
		maxCharsPerLine = int(atlasWrapWidth / advW)
		if maxCharsPerLine < 1 {
			maxCharsPerLine = 1
		}
	}

	tb.lines = tb.lines[:0]

	var maxW float64
	var curLine textLine
	var wordGlyphs []glyphPos
	var wordLen int
	var lineCharCount int

	flush := func() {
		if curLine.width > maxW {
			maxW = curLine.width
		}
		tb.lines = append(tb.lines, curLine)
		curLine = textLine{}
		lineCharCount = 0
	}

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		i += size

		if r == '\n' {
			curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
			curLine.width = float64(lineCharCount+wordLen) * advW
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0
			flush()
			continue
		}

		// Space advances the cursor without needing a glyph in the atlas.
		if r == ' ' {
			curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0

			lineCharCount++
			curLine.width = float64(lineCharCount) * advW
			continue
		}

		pg, found := f.glyph(r)
		if !found {
			continue
		}

		gp := glyphPos{
			x: float64(lineCharCount+wordLen) * advW,
			y: 0,
			region: TextureRegion{
				Page:      f.page,
				X:         pg.x + uint16(srcX),
				Y:         pg.y + uint16(srcY),
				Width:     uint16(srcW),
				Height:    uint16(srcH),
				OriginalW: uint16(srcW),
				OriginalH: uint16(srcH),
			},
			page: f.page,
		}

		wordGlyphs = append(wordGlyphs, gp)
		wordLen++

		// Word wrap check.
		if maxCharsPerLine > 0 && lineCharCount+wordLen > maxCharsPerLine && lineCharCount > 0 {
			flush()
			// Re-position word glyphs at start of new line.
			for gi := range wordGlyphs {
				wordGlyphs[gi].x = float64(gi) * advW
			}
		}
	}

	// Flush remaining word and line.
	curLine.glyphs = append(curLine.glyphs, wordGlyphs...)
	lineCharCount += wordLen
	curLine.width = float64(lineCharCount) * advW
	if len(curLine.glyphs) > 0 || len(tb.lines) == 0 {
		if curLine.width > maxW {
			maxW = curLine.width
		}
		tb.lines = append(tb.lines, curLine)
	}

	// Alignment.
	alignW := maxW
	if atlasWrapWidth > 0 {
		alignW = atlasWrapWidth
	}
	for li := range tb.lines {
		line := &tb.lines[li]
		var offsetX float64
		switch tb.Align {
		case TextAlignLeft:
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

// rebuildPixelVerts builds local-space vertex/index buffers for pixel font glyphs.
// Reuses the sdfVerts/sdfInds fields (same quad structure, no SDF padding).
func (tb *TextBlock) rebuildPixelVerts(f *PixelFont) {
	lines := tb.lines
	lh := float64(f.cellH)

	glyphCount := 0
	for _, line := range lines {
		glyphCount += len(line.glyphs)
	}

	vertCount := glyphCount * 4
	indCount := glyphCount * 6
	if cap(tb.sdfVerts) < vertCount {
		tb.sdfVerts = make([]ebiten.Vertex, vertCount)
	}
	tb.sdfVerts = tb.sdfVerts[:vertCount]
	if cap(tb.sdfInds) < indCount {
		tb.sdfInds = make([]uint16, indCount)
	}
	tb.sdfInds = tb.sdfInds[:indCount]

	vi := 0
	ii := 0
	for li, line := range lines {
		lineY := float64(li) * lh
		for _, gp := range line.glyphs {
			dx := float32(gp.x)
			dy := float32(gp.y + lineY)
			dw := float32(gp.region.Width)
			dh := float32(gp.region.Height)

			sx := float32(gp.region.X)
			sy := float32(gp.region.Y)
			sw := float32(gp.region.Width)
			sh := float32(gp.region.Height)

			base := uint16(vi)
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy,
				SrcX: sx, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy,
				SrcX: sx + sw, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx, DstY: dy + dh,
				SrcX: sx, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++
			tb.sdfVerts[vi] = ebiten.Vertex{
				DstX: dx + dw, DstY: dy + dh,
				SrcX: sx + sw, SrcY: sy + sh,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			vi++

			tb.sdfInds[ii] = base
			tb.sdfInds[ii+1] = base + 1
			tb.sdfInds[ii+2] = base + 2
			tb.sdfInds[ii+3] = base + 1
			tb.sdfInds[ii+4] = base + 3
			tb.sdfInds[ii+5] = base + 2
			ii += 6
		}
	}
	tb.sdfVertCount = vi
	tb.sdfIndCount = ii
}

// emitPixelTextCommand emits a CommandBitmapText for pixel-perfect bitmap font rendering.
func emitPixelTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	tb.layout()
	if tb.measuredW == 0 || tb.measuredH == 0 {
		return commands
	}

	f := tb.Font.(*PixelFont)

	if tb.sdfVertCount == 0 || len(tb.sdfVerts) == 0 {
		tb.rebuildPixelVerts(f)
	}
	if tb.sdfVertCount == 0 {
		return commands
	}

	// Apply integer scale as an affine transform.
	s := tb.fontScale()
	fst := [6]float64{s, 0, 0, s, 0, 0}
	scaledWT := multiplyAffine(worldTransform, fst)

	atlasImg := atlasManager().Page(int(f.page))
	if atlasImg == nil {
		return commands
	}

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:      CommandBitmapText,
		Transform: affine32(scaledWT),
		Color: color32{
			float32(tb.Color.R * n.Color.R),
			float32(tb.Color.G * n.Color.G),
			float32(tb.Color.B * n.Color.B),
			float32(tb.Color.A * n.Color.A * n.worldAlpha),
		},
		BlendMode:    n.BlendMode,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		treeOrder:    *treeOrder,
		bmpVerts:     tb.sdfVerts,
		bmpInds:      tb.sdfInds,
		bmpVertCount: tb.sdfVertCount,
		bmpIndCount:  tb.sdfIndCount,
		bmpImage:     atlasImg,
	})

	return commands
}

// --- SDF Kage shader sources ---

// sdfShaderSrc is the single-channel SDF fragment shader.
// Reads distance from the alpha channel of the atlas glyph.
// Produces fill, outline, glow, and drop shadow in a single pass.
const sdfShaderSrc = `//kage:unit pixels
package main

// Edge threshold for fill (0.5 = exact edge in normalized distance).
var Threshold float
// Smoothing width in distance-field units (controls anti-aliasing sharpness).
var Smoothing float
// Outline width in distance-field units (0 = no outline).
var OutlineWidth float
// Outline color (premultiplied RGBA).
var OutlineColor vec4
// Glow width in distance-field units (0 = no glow).
var GlowWidth float
// Glow color (premultiplied RGBA).
var GlowColor vec4
// Shadow offset in pixels.
var ShadowOffset vec2
// Shadow color (premultiplied RGBA).
var ShadowColor vec4
// Shadow softness in distance-field units.
var ShadowSoftness float
// Fill color (premultiplied RGBA).
var FillColor vec4

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := imageSrc0At(src).a

	// Combine per-pixel derivatives with scale-aware uniform for optimal smoothing.
	// fwidth gives per-texel sharpness; Smoothing provides the scale-aware floor.
	sm := max(fwidth(dist), Smoothing)

	// --- Shadow ---
	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := imageSrc0At(src - ShadowOffset).a
		shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
		result = ShadowColor * shadowAlpha
	}

	// --- Glow ---
	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
		result = mix(result, GlowColor, GlowColor.a * glowAlpha)
	}

	// --- Outline ---
	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
		result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
	}

	// --- Fill ---
	fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
	result = mix(result, FillColor, FillColor.a * fillAlpha)

	// Fade to transparent at the SDF range boundary (dist near 0)
	// to prevent visible rectangles where the distance field runs out.
	// Uses Smoothing*2 instead of fwidth*3 for scale-independent fading:
	// Smoothing already accounts for the atlas-to-screen pixel ratio,
	// so the fade range stays correct regardless of font scale or zoom.
	result *= smoothstep(0, Smoothing*2, dist)

	// Apply vertex color as a tint multiplier (Node.Color).
	result *= color

	return result
}
`

// msdfShaderSrc is the multi-channel SDF fragment shader.
// Computes median(R, G, B) for the distance value, then applies the same
// threshold logic as the single-channel shader.
const msdfShaderSrc = `//kage:unit pixels
package main

var Threshold float
var Smoothing float
var OutlineWidth float
var OutlineColor vec4
var GlowWidth float
var GlowColor vec4
var ShadowOffset vec2
var ShadowColor vec4
var ShadowSoftness float
var FillColor vec4

func median3(a, b, c float) float {
	return max(min(a, b), min(max(a, b), c))
}

func sampleDist(pos vec2) float {
	s := imageSrc0At(pos)
	return median3(s.r, s.g, s.b)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := sampleDist(src)

	// Combine per-pixel derivatives with scale-aware uniform for optimal smoothing.
	sm := max(fwidth(dist), Smoothing)

	// --- Shadow ---
	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := sampleDist(src - ShadowOffset)
		shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
		result = ShadowColor * shadowAlpha
	}

	// --- Glow ---
	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
		result = mix(result, GlowColor, GlowColor.a * glowAlpha)
	}

	// --- Outline ---
	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
		result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
	}

	// --- Fill ---
	fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
	result = mix(result, FillColor, FillColor.a * fillAlpha)

	// Fade to transparent at the SDF range boundary (dist near 0)
	// to prevent visible rectangles where the distance field runs out.
	// Uses Smoothing*2 instead of fwidth*3 for scale-independent fading.
	result *= smoothstep(0, Smoothing*2, dist)

	// Apply vertex color as a tint multiplier (Node.Color).
	result *= color

	return result
}
`

// --- Lazy shader compilation ---

var (
	sdfShader  *ebiten.Shader
	msdfShader *ebiten.Shader
)

func ensureSDFShader() *ebiten.Shader {
	if sdfShader == nil {
		s, err := ebiten.NewShader([]byte(sdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile SDF shader: " + err.Error())
		}
		sdfShader = s
	}
	return sdfShader
}

func ensureMSDFShader() *ebiten.Shader {
	if msdfShader == nil {
		s, err := ebiten.NewShader([]byte(msdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile MSDF shader: " + err.Error())
		}
		msdfShader = s
	}
	return msdfShader
}

// ensureUniforms builds or updates the cached uniform map for the SDF shader.
// On first call it allocates the map; on subsequent calls it updates values in place.
// Smoothing is always updated since it depends on displayScale.
func (tb *TextBlock) ensureUniforms(f *SpriteFont, displayScale float64) {
	// Scale-aware smoothing.
	smoothing := float32(1.5 / (f.distanceRange * displayScale))

	// Threshold: thicken at small sizes.
	threshold := float32(0.5)
	if tb.FontSize > 0 && tb.FontSize < 24 {
		t := tb.FontSize / 24
		threshold = float32(0.30 + 0.20*t)
	}

	fillColor := tb.Color
	fillPremul := [4]float32{
		float32(fillColor.R * fillColor.A),
		float32(fillColor.G * fillColor.A),
		float32(fillColor.B * fillColor.A),
		float32(fillColor.A),
	}

	if tb.sdfUniforms == nil {
		tb.sdfUniforms = make(map[string]any, 11)
		tb.uniformsDirty = true // force full populate
	}

	// Smoothing always changes with camera zoom.
	tb.sdfUniforms["Smoothing"] = smoothing

	if !tb.uniformsDirty {
		return
	}
	tb.uniformsDirty = false

	tb.sdfUniforms["Threshold"] = threshold
	tb.sdfUniforms["FillColor"] = fillPremul[:]
	tb.sdfUniforms["OutlineWidth"] = float32(0)
	tb.sdfUniforms["OutlineColor"] = []float32{0, 0, 0, 0}
	tb.sdfUniforms["GlowWidth"] = float32(0)
	tb.sdfUniforms["GlowColor"] = []float32{0, 0, 0, 0}
	tb.sdfUniforms["ShadowOffset"] = []float32{0, 0}
	tb.sdfUniforms["ShadowColor"] = []float32{0, 0, 0, 0}
	tb.sdfUniforms["ShadowSoftness"] = float32(0)

	if tb.TextEffects != nil {
		e := tb.TextEffects
		tb.sdfUniforms["OutlineWidth"] = float32(e.OutlineWidth / f.distanceRange)
		tb.sdfUniforms["OutlineColor"] = []float32{
			float32(e.OutlineColor.R * e.OutlineColor.A),
			float32(e.OutlineColor.G * e.OutlineColor.A),
			float32(e.OutlineColor.B * e.OutlineColor.A),
			float32(e.OutlineColor.A),
		}
		tb.sdfUniforms["GlowWidth"] = float32(e.GlowWidth / f.distanceRange)
		tb.sdfUniforms["GlowColor"] = []float32{
			float32(e.GlowColor.R * e.GlowColor.A),
			float32(e.GlowColor.G * e.GlowColor.A),
			float32(e.GlowColor.B * e.GlowColor.A),
			float32(e.GlowColor.A),
		}
		tb.sdfUniforms["ShadowOffset"] = []float32{
			float32(e.ShadowOffset.X),
			float32(e.ShadowOffset.Y),
		}
		tb.sdfUniforms["ShadowColor"] = []float32{
			float32(e.ShadowColor.R * e.ShadowColor.A),
			float32(e.ShadowColor.G * e.ShadowColor.A),
			float32(e.ShadowColor.B * e.ShadowColor.A),
			float32(e.ShadowColor.A),
		}
		tb.sdfUniforms["ShadowSoftness"] = float32(e.ShadowSoftness / f.distanceRange)
	}

	// Select shader.
	if f.multiChannel {
		tb.sdfShader = ensureMSDFShader()
	} else {
		tb.sdfShader = ensureSDFShader()
	}
}

// emitSDFTextCommand emits a CommandSDF carrying local-space glyph quads.
// The batch submitter transforms vertices to screen space and calls
// DrawTrianglesShader  -  SDF shader runs at display resolution.
func emitSDFTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	tb.layout()
	if tb.measuredW == 0 || tb.measuredH == 0 {
		return commands
	}

	f := tb.Font.(*SpriteFont)

	// Rebuild local verts if layout changed (layout() clears layoutDirty and
	// triggers rebuildLocalVerts via the dirty path).
	if tb.sdfVertCount == 0 || len(tb.sdfVerts) == 0 {
		tb.rebuildLocalVerts(f)
	}
	if tb.sdfVertCount == 0 {
		return commands
	}

	// Apply font scale as an affine transform: atlas pixels → display pixels.
	fontScale := tb.fontScale()
	fst := [6]float64{fontScale, 0, 0, fontScale, 0, 0}
	scaledWT := multiplyAffine(worldTransform, fst)

	// Resolve atlas image.
	atlasImg := atlasManager().Page(int(f.page))
	if atlasImg == nil {
		return commands
	}

	// Extract the node's world scale (without fontScale) for smoothing.
	displayScale := math.Sqrt(worldTransform[0]*worldTransform[0] + worldTransform[1]*worldTransform[1])
	if displayScale < 0.05 {
		displayScale = 0.05
	}
	tb.ensureUniforms(f, displayScale)

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:         CommandSDF,
		Transform:    affine32(scaledWT),
		Color:        color32{float32(n.Color.R), float32(n.Color.G), float32(n.Color.B), float32(n.Color.A * n.worldAlpha)},
		BlendMode:    n.BlendMode,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		treeOrder:    *treeOrder,
		sdfVerts:     tb.sdfVerts,
		sdfInds:      tb.sdfInds,
		sdfVertCount: tb.sdfVertCount,
		sdfIndCount:  tb.sdfIndCount,
		sdfShader:    tb.sdfShader,
		sdfAtlasImg:  atlasImg,
		sdfUniforms:  tb.sdfUniforms,
	})

	return commands
}
