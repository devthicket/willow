package willow

import (
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
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

// --- Free functions replacing TextBlock methods ---

// textBlockLineHeight returns the effective line height for this text block in atlas pixels.
func textBlockLineHeight(tb *TextBlock) float64 {
	if tb.LineHeight > 0 {
		return tb.LineHeight
	}
	if tb.Font != nil {
		return tb.Font.LineHeight()
	}
	return 0
}

// textBlockFontScale returns the scale factor that maps native atlas pixels to display pixels.
// Returns 1.0 when FontSize is zero/negative or Font is nil.
func textBlockFontScale(tb *TextBlock) float64 {
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

// textBlockLayout recomputes glyph positions if dirty. Returns the cached lines.
func textBlockLayout(tb *TextBlock) []textLine {
	if !tb.LayoutDirty {
		return tb.Lines
	}
	tb.LayoutDirty = false

	if tb.Font == nil {
		tb.Lines = tb.Lines[:0]
		tb.MeasuredW = 0
		tb.MeasuredH = 0
		return tb.Lines
	}

	switch f := tb.Font.(type) {
	case *SpriteFont:
		textBlockLayoutSDF(tb, f)
		textBlockRebuildLocalVerts(tb, f)
		tb.UniformsDirty = true
	case *PixelFont:
		textBlockLayoutPixel(tb, f)
		textBlockRebuildPixelVerts(tb, f)
	default:
		tb.Lines = tb.Lines[:0]
		tb.MeasuredW = 0
		tb.MeasuredH = 0
	}

	return tb.Lines
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

// textBlockLayoutSDF computes glyph positions for an SpriteFont.
func textBlockLayoutSDF(tb *TextBlock, f *SpriteFont) {
	lh := textBlockLineHeight(tb)
	content := tb.Content

	// Convert screen-pixel WrapWidth to atlas pixels.
	fs := textBlockFontScale(tb)
	atlasWrapWidth := 0.0
	if tb.WrapWidth > 0 && fs > 0 {
		atlasWrapWidth = tb.WrapWidth / fs
	}

	tb.Lines = tb.Lines[:0]

	var maxW float64
	var curLine textLine

	var wordStart int
	var wordGlyphs []glyphPos
	var wordWidth float64
	var cursorX float64
	var prevRune rune
	var hasPrev bool

	flush := func() {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
		curLine = textLine{}
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
			X: glyphX,
			Y: glyphY,
			Region: TextureRegion{
				Page:      f.page,
				X:         g.x,
				Y:         g.y,
				Width:     g.width,
				Height:    g.height,
				OriginalW: g.width,
				OriginalH: g.height,
			},
			Page: f.page,
		}

		advance := float64(g.xAdvance) + float64(kern)

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
	for li := range tb.Lines {
		line := &tb.Lines[li]
		var offsetX float64
		switch tb.Align {
		case TextAlignLeft:
		case TextAlignCenter:
			offsetX = (alignW - line.Width) / 2
		case TextAlignRight:
			offsetX = alignW - line.Width
		}
		if offsetX != 0 {
			for gi := range line.Glyphs {
				line.Glyphs[gi].X += offsetX
			}
		}
	}

	tb.MeasuredW = maxW
	tb.MeasuredH = float64(len(tb.Lines)) * lh
}

// textBlockRebuildLocalVerts builds local-space vertex/index buffers from the laid-out
// glyph positions. The atlas glyph regions already include distanceRange+1
// pixels of SDF padding, so no per-glyph expansion is needed  -  the shader
// has sufficient distance-field data for outline, glow, and shadow effects
// within the atlas's distance range.
func textBlockRebuildLocalVerts(tb *TextBlock, f *SpriteFont) {
	lines := tb.Lines
	lh := textBlockLineHeight(tb)

	glyphCount := 0
	for _, line := range lines {
		glyphCount += len(line.Glyphs)
	}

	// Grow buffers to high-water mark.
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
		lineY := float64(li) * lh
		for _, gp := range line.Glyphs {
			// Destination in local atlas-pixel space.
			dx := float32(gp.X)
			dy := float32(gp.Y + lineY)
			dw := float32(gp.Region.Width)
			dh := float32(gp.Region.Height)

			// Source in atlas pixel coords (1:1 mapping, no expansion).
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

// --- Pixel font layout and rendering ---

// textBlockLayoutPixel computes glyph positions for a PixelFont (monospaced grid).
func textBlockLayoutPixel(tb *TextBlock, f *PixelFont) {
	lh := float64(f.advanceH())
	advW := float64(f.advanceW())
	content := tb.Content

	// Trimmed source region dimensions.
	srcX := f.padLeft
	srcY := f.padTop
	srcW := f.advanceW()
	srcH := f.advanceH()

	// Convert screen-pixel WrapWidth to native pixels.
	scale := textBlockFontScale(tb)
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

	tb.Lines = tb.Lines[:0]

	var maxW float64
	var curLine textLine
	var wordGlyphs []glyphPos
	var wordLen int
	var lineCharCount int

	flush := func() {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
		curLine = textLine{}
		lineCharCount = 0
	}

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		i += size

		if r == '\n' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			curLine.Width = float64(lineCharCount+wordLen) * advW
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0
			flush()
			continue
		}

		// Space advances the cursor without needing a glyph in the atlas.
		if r == ' ' {
			curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
			lineCharCount += wordLen
			wordGlyphs = wordGlyphs[:0]
			wordLen = 0

			lineCharCount++
			curLine.Width = float64(lineCharCount) * advW
			continue
		}

		pg, found := f.glyph(r)
		if !found {
			continue
		}

		gp := glyphPos{
			X: float64(lineCharCount+wordLen) * advW,
			Y: 0,
			Region: TextureRegion{
				Page:      f.page,
				X:         pg.x + uint16(srcX),
				Y:         pg.y + uint16(srcY),
				Width:     uint16(srcW),
				Height:    uint16(srcH),
				OriginalW: uint16(srcW),
				OriginalH: uint16(srcH),
			},
			Page: f.page,
		}

		wordGlyphs = append(wordGlyphs, gp)
		wordLen++

		// Word wrap check.
		if maxCharsPerLine > 0 && lineCharCount+wordLen > maxCharsPerLine && lineCharCount > 0 {
			flush()
			// Re-position word glyphs at start of new line.
			for gi := range wordGlyphs {
				wordGlyphs[gi].X = float64(gi) * advW
			}
		}
	}

	// Flush remaining word and line.
	curLine.Glyphs = append(curLine.Glyphs, wordGlyphs...)
	lineCharCount += wordLen
	curLine.Width = float64(lineCharCount) * advW
	if len(curLine.Glyphs) > 0 || len(tb.Lines) == 0 {
		if curLine.Width > maxW {
			maxW = curLine.Width
		}
		tb.Lines = append(tb.Lines, curLine)
	}

	// Alignment.
	alignW := maxW
	if atlasWrapWidth > 0 {
		alignW = atlasWrapWidth
	}
	for li := range tb.Lines {
		line := &tb.Lines[li]
		var offsetX float64
		switch tb.Align {
		case TextAlignLeft:
		case TextAlignCenter:
			offsetX = (alignW - line.Width) / 2
		case TextAlignRight:
			offsetX = alignW - line.Width
		}
		if offsetX != 0 {
			for gi := range line.Glyphs {
				line.Glyphs[gi].X += offsetX
			}
		}
	}

	tb.MeasuredW = maxW
	tb.MeasuredH = float64(len(tb.Lines)) * lh
}

// textBlockRebuildPixelVerts builds local-space vertex/index buffers for pixel font glyphs.
// Reuses the SdfVerts/SdfInds fields (same quad structure, no SDF padding).
func textBlockRebuildPixelVerts(tb *TextBlock, f *PixelFont) {
	lines := tb.Lines
	lh := float64(f.cellH)

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
		lineY := float64(li) * lh
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

// emitPixelTextCommand emits a CommandBitmapText for pixel-perfect bitmap font rendering.
func emitPixelTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	textBlockLayout(tb)
	if tb.MeasuredW == 0 || tb.MeasuredH == 0 {
		return commands
	}

	f := tb.Font.(*PixelFont)

	if tb.SdfVertCount == 0 || len(tb.SdfVerts) == 0 {
		textBlockRebuildPixelVerts(tb, f)
	}
	if tb.SdfVertCount == 0 {
		return commands
	}

	// Apply integer scale as an affine transform.
	s := textBlockFontScale(tb)
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
			R: float32(tb.Color.R() * n.Color_.R()),
			G: float32(tb.Color.G() * n.Color_.G()),
			B: float32(tb.Color.B() * n.Color_.B()),
			A: float32(tb.Color.A() * n.Color_.A() * n.WorldAlpha),
		},
		BlendMode:    n.BlendMode_,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		TreeOrder:    *treeOrder,
		BmpVerts:     tb.SdfVerts,
		BmpInds:      tb.SdfInds,
		BmpVertCount: tb.SdfVertCount,
		BmpIndCount:  tb.SdfIndCount,
		BmpImage:     atlasImg,
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
	sdfShader_  *ebiten.Shader
	msdfShader_ *ebiten.Shader
)

func ensureSDFShader() *ebiten.Shader {
	if sdfShader_ == nil {
		s, err := ebiten.NewShader([]byte(sdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile SDF shader: " + err.Error())
		}
		sdfShader_ = s
	}
	return sdfShader_
}

func ensureMSDFShader() *ebiten.Shader {
	if msdfShader_ == nil {
		s, err := ebiten.NewShader([]byte(msdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile MSDF shader: " + err.Error())
		}
		msdfShader_ = s
	}
	return msdfShader_
}

// textBlockEnsureUniforms builds or updates the cached uniform map for the SDF shader.
// On first call it allocates the map; on subsequent calls it updates values in place.
// Smoothing is always updated since it depends on displayScale.
func textBlockEnsureUniforms(tb *TextBlock, f *SpriteFont, displayScale float64) {
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
		float32(fillColor.R() * fillColor.A()),
		float32(fillColor.G() * fillColor.A()),
		float32(fillColor.B() * fillColor.A()),
		float32(fillColor.A()),
	}

	if tb.SdfUniforms == nil {
		tb.SdfUniforms = make(map[string]any, 11)
		tb.UniformsDirty = true // force full populate
	}

	// Smoothing always changes with camera zoom.
	tb.SdfUniforms["Smoothing"] = smoothing

	if !tb.UniformsDirty {
		return
	}
	tb.UniformsDirty = false

	tb.SdfUniforms["Threshold"] = threshold
	tb.SdfUniforms["FillColor"] = fillPremul[:]
	tb.SdfUniforms["OutlineWidth"] = float32(0)
	tb.SdfUniforms["OutlineColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["GlowWidth"] = float32(0)
	tb.SdfUniforms["GlowColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["ShadowOffset"] = []float32{0, 0}
	tb.SdfUniforms["ShadowColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["ShadowSoftness"] = float32(0)

	if tb.TextEffects != nil {
		e := tb.TextEffects
		tb.SdfUniforms["OutlineWidth"] = float32(e.OutlineWidth / f.distanceRange)
		tb.SdfUniforms["OutlineColor"] = []float32{
			float32(e.OutlineColor.R() * e.OutlineColor.A()),
			float32(e.OutlineColor.G() * e.OutlineColor.A()),
			float32(e.OutlineColor.B() * e.OutlineColor.A()),
			float32(e.OutlineColor.A()),
		}
		tb.SdfUniforms["GlowWidth"] = float32(e.GlowWidth / f.distanceRange)
		tb.SdfUniforms["GlowColor"] = []float32{
			float32(e.GlowColor.R() * e.GlowColor.A()),
			float32(e.GlowColor.G() * e.GlowColor.A()),
			float32(e.GlowColor.B() * e.GlowColor.A()),
			float32(e.GlowColor.A()),
		}
		tb.SdfUniforms["ShadowOffset"] = []float32{
			float32(e.ShadowOffset.X),
			float32(e.ShadowOffset.Y),
		}
		tb.SdfUniforms["ShadowColor"] = []float32{
			float32(e.ShadowColor.R() * e.ShadowColor.A()),
			float32(e.ShadowColor.G() * e.ShadowColor.A()),
			float32(e.ShadowColor.B() * e.ShadowColor.A()),
			float32(e.ShadowColor.A()),
		}
		tb.SdfUniforms["ShadowSoftness"] = float32(e.ShadowSoftness / f.distanceRange)
	}

	// Select shader.
	if f.multiChannel {
		tb.SdfShader = ensureMSDFShader()
	} else {
		tb.SdfShader = ensureSDFShader()
	}
}

// emitSDFTextCommand emits a CommandSDF carrying local-space glyph quads.
// The batch submitter transforms vertices to screen space and calls
// DrawTrianglesShader  -  SDF shader runs at display resolution.
func emitSDFTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	textBlockLayout(tb)
	if tb.MeasuredW == 0 || tb.MeasuredH == 0 {
		return commands
	}

	f := tb.Font.(*SpriteFont)

	// Rebuild local verts if layout changed (layout() clears LayoutDirty and
	// triggers textBlockRebuildLocalVerts via the dirty path).
	if tb.SdfVertCount == 0 || len(tb.SdfVerts) == 0 {
		textBlockRebuildLocalVerts(tb, f)
	}
	if tb.SdfVertCount == 0 {
		return commands
	}

	// Apply font scale as an affine transform: atlas pixels -> display pixels.
	fontScale := textBlockFontScale(tb)
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
	textBlockEnsureUniforms(tb, f, displayScale)

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:         CommandSDF,
		Transform:    affine32(scaledWT),
		Color:        color32{R: float32(n.Color_.R()), G: float32(n.Color_.G()), B: float32(n.Color_.B()), A: float32(n.Color_.A() * n.WorldAlpha)},
		BlendMode:    n.BlendMode_,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		TreeOrder:    *treeOrder,
		SdfVerts:     tb.SdfVerts,
		SdfInds:      tb.SdfInds,
		SdfVertCount: tb.SdfVertCount,
		SdfIndCount:  tb.SdfIndCount,
		SdfShader:    tb.SdfShader,
		SdfAtlasImg:  atlasImg,
		SdfUniforms:  tb.SdfUniforms,
	})

	return commands
}
