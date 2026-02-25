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
	// Outline defines a text stroke rendered behind the fill. Nil means no outline.
	Outline *Outline
	// LineHeight overrides the font's default line height. Zero uses Font.LineHeight().
	LineHeight float64

	// Cached layout (unexported)
	layoutDirty bool
	measuredW   float64
	measuredH   float64
	lines       []textLine // cached line layout

	// SDF rendering fields (unexported)
	TextEffects *TextEffects    // nil = no effects (plain fill)
	sdfImage    *ebiten.Image   // cached SDF render
	sdfPage     int             // page index where sdfImage is registered (-1 = unset)
	sdfDirty    bool            // true when SDF cache needs re-render
	sdfVerts    []ebiten.Vertex // preallocated vertex buffer, grows to high-water mark
	sdfInds     []uint16        // preallocated index buffer
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
// Align, LineHeight, Color, or Outline at runtime.
func (tb *TextBlock) Invalidate() {
	tb.layoutDirty = true
	tb.sdfDirty = true
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

	tb.sdfDirty = true

	switch f := tb.Font.(type) {
	case *SpriteFont:
		tb.layoutSDF(f)
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

// emitSDFTextCommand renders SDF text to a cached image via DrawTrianglesShader
// and emits a single CommandSprite with directImage. The image is only
// re-rendered when the text or effects change (sdfDirty).
func emitSDFTextCommand(tb *TextBlock, n *Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	tb.layout()
	if tb.measuredW == 0 || tb.measuredH == 0 {
		return commands
	}

	f := tb.Font.(*SpriteFont)
	alpha := n.worldAlpha

	// Apply font scale as an affine transform: atlas pixels → display pixels.
	// This is multiplied into the world transform so ScaleX/ScaleY remain independent.
	fontScale := tb.fontScale()
	fst := [6]float64{fontScale, 0, 0, fontScale, 0, 0}
	scaledWT := multiplyAffine(worldTransform, fst)

	// Extract the node's world scale (without fontScale) for smoothing.
	// The SDF image is rendered at atlas resolution; fontScale is applied as a
	// separate transform, so smoothing should not account for it.
	displayScale := math.Sqrt(worldTransform[0]*worldTransform[0] + worldTransform[1]*worldTransform[1])
	if displayScale < 0.05 {
		displayScale = 0.05
	}

	// Compute effect padding in pixels
	effectPad := 0.0
	if tb.TextEffects != nil {
		e := tb.TextEffects
		if e.OutlineWidth > 0 {
			effectPad = e.OutlineWidth * f.distanceRange
		}
		if e.GlowWidth > 0 {
			gpad := (e.OutlineWidth + e.GlowWidth) * f.distanceRange
			if gpad > effectPad {
				effectPad = gpad
			}
		}
		if e.ShadowOffset.X != 0 || e.ShadowOffset.Y != 0 || e.ShadowSoftness > 0 {
			sx := math.Abs(e.ShadowOffset.X) + e.ShadowSoftness*f.distanceRange
			sy := math.Abs(e.ShadowOffset.Y) + e.ShadowSoftness*f.distanceRange
			if sx > effectPad {
				effectPad = sx
			}
			if sy > effectPad {
				effectPad = sy
			}
		}
	}
	pad := int(effectPad + 1)

	imgW := tb.measuredW
	if tb.Align != TextAlignLeft && tb.WrapWidth > 0 {
		fs := tb.fontScale()
		if fs > 0 {
			imgW = tb.WrapWidth / fs
		}
	}
	w := int(imgW) + 2*pad + 1
	h := int(tb.measuredH) + 2*pad + 1

	if tb.sdfDirty || tb.sdfImage == nil {
		tb.sdfDirty = false

		// Reuse or create image
		if tb.sdfImage != nil {
			oldB := tb.sdfImage.Bounds()
			if oldB.Dx() != w || oldB.Dy() != h {
				tb.sdfImage.Deallocate()
				tb.sdfImage = ebiten.NewImage(w, h)
			} else {
				tb.sdfImage.Clear()
			}
		} else {
			tb.sdfImage = ebiten.NewImage(w, h)
		}

		// Get the SDF atlas page image
		am := atlasManager()
		atlasImg := am.Page(int(f.page))
		if atlasImg == nil {
			// Atlas not registered yet; emit nothing
			return commands
		}

		// Build vertex/index buffers for all glyph quads
		lines := tb.lines
		lh := tb.lineHeight()
		glyphCount := 0
		for _, line := range lines {
			glyphCount += len(line.glyphs)
		}

		// Grow vertex/index buffers to high-water mark
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
				// Destination position in the cached image (offset by pad)
				dx := float32(gp.x) + float32(pad)
				dy := float32(gp.y+lineY) + float32(pad)
				dw := float32(gp.region.Width)
				dh := float32(gp.region.Height)

				// Source coordinates in atlas (pixel coordinates for //kage:unit pixels)
				sx := float32(gp.region.X)
				sy := float32(gp.region.Y)
				sw := float32(gp.region.Width)
				sh := float32(gp.region.Height)

				base := uint16(vi)
				// Top-left
				tb.sdfVerts[vi] = ebiten.Vertex{
					DstX: dx, DstY: dy,
					SrcX: sx, SrcY: sy,
					ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
				}
				vi++
				// Top-right
				tb.sdfVerts[vi] = ebiten.Vertex{
					DstX: dx + dw, DstY: dy,
					SrcX: sx + sw, SrcY: sy,
					ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
				}
				vi++
				// Bottom-left
				tb.sdfVerts[vi] = ebiten.Vertex{
					DstX: dx, DstY: dy + dh,
					SrcX: sx, SrcY: sy + sh,
					ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
				}
				vi++
				// Bottom-right
				tb.sdfVerts[vi] = ebiten.Vertex{
					DstX: dx + dw, DstY: dy + dh,
					SrcX: sx + sw, SrcY: sy + sh,
					ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
				}
				vi++

				// Two triangles
				tb.sdfInds[ii] = base
				tb.sdfInds[ii+1] = base + 1
				tb.sdfInds[ii+2] = base + 2
				tb.sdfInds[ii+3] = base + 1
				tb.sdfInds[ii+4] = base + 3
				tb.sdfInds[ii+5] = base + 2
				ii += 6
			}
		}

		// Select shader
		var shader *ebiten.Shader
		if f.multiChannel {
			shader = ensureMSDFShader()
		} else {
			shader = ensureSDFShader()
		}

		// Scale-aware smoothing: the numerator controls the AA band width in
		// screen pixels. 1.5 gives ~3px of anti-aliasing for smooth edges.
		smoothing := 1.5 / (f.distanceRange * displayScale)

		// Below FontSize 24: progressively thicken glyphs to compensate for
		// thin strokes at small display sizes. At 24+ the default threshold
		// and full AA band produce smooth text.
		threshold := 0.5
		if tb.FontSize > 0 && tb.FontSize < 24 {
			t := tb.FontSize / 24     // 0..1 range
			threshold = 0.30 + 0.20*t // 0.30 at tiny sizes, 0.50 at 24
		}

		// Build uniforms
		fillColor := tb.Color
		fillPremul := [4]float32{
			float32(fillColor.R * fillColor.A),
			float32(fillColor.G * fillColor.A),
			float32(fillColor.B * fillColor.A),
			float32(fillColor.A),
		}

		uniforms := map[string]any{
			"Threshold":      float32(threshold),
			"Smoothing":      float32(smoothing),
			"OutlineWidth":   float32(0),
			"OutlineColor":   []float32{0, 0, 0, 0},
			"GlowWidth":      float32(0),
			"GlowColor":      []float32{0, 0, 0, 0},
			"ShadowOffset":   []float32{0, 0},
			"ShadowColor":    []float32{0, 0, 0, 0},
			"ShadowSoftness": float32(0),
			"FillColor":      fillPremul[:],
		}

		if tb.TextEffects != nil {
			e := tb.TextEffects
			// Scale widths to distance-field units (normalized to [0,1] range)
			ow := e.OutlineWidth / f.distanceRange
			uniforms["OutlineWidth"] = float32(ow)
			uniforms["OutlineColor"] = []float32{
				float32(e.OutlineColor.R * e.OutlineColor.A),
				float32(e.OutlineColor.G * e.OutlineColor.A),
				float32(e.OutlineColor.B * e.OutlineColor.A),
				float32(e.OutlineColor.A),
			}
			uniforms["GlowWidth"] = float32(e.GlowWidth / f.distanceRange)
			uniforms["GlowColor"] = []float32{
				float32(e.GlowColor.R * e.GlowColor.A),
				float32(e.GlowColor.G * e.GlowColor.A),
				float32(e.GlowColor.B * e.GlowColor.A),
				float32(e.GlowColor.A),
			}
			uniforms["ShadowOffset"] = []float32{
				float32(e.ShadowOffset.X),
				float32(e.ShadowOffset.Y),
			}
			uniforms["ShadowColor"] = []float32{
				float32(e.ShadowColor.R * e.ShadowColor.A),
				float32(e.ShadowColor.G * e.ShadowColor.A),
				float32(e.ShadowColor.B * e.ShadowColor.A),
				float32(e.ShadowColor.A),
			}
			uniforms["ShadowSoftness"] = float32(e.ShadowSoftness / f.distanceRange)
		}

		opts := &ebiten.DrawTrianglesShaderOptions{
			Uniforms: uniforms,
			Images:   [4]*ebiten.Image{atlasImg},
		}
		tb.sdfImage.DrawTrianglesShader(tb.sdfVerts[:vi], tb.sdfInds[:ii], shader, opts)

		// Allocate a page slot once, reuse on subsequent renders
		if tb.sdfPage < 0 {
			tb.sdfPage = am.AllocPage()
		}
		am.RegisterPage(tb.sdfPage, tb.sdfImage)
	}

	// Emit single sprite command with the cached SDF image.
	// The transform is offset by -pad so the text content aligns with the node position.
	// Uses scaledWT (world * fontScale) so the SDF image renders at display size.
	padF := float64(pad)
	adjustedTransform := composeGlyphTransform(scaledWT, -padF, -padF)

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:      CommandSprite,
		Transform: affine32(adjustedTransform),
		TextureRegion: TextureRegion{
			Page:      uint16(tb.sdfPage),
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

	return commands
}
