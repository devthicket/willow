package willow

import (
	"encoding/json"
	"fmt"
	"image"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// SDFEffects configures distance-field text effects rendered by the SDF shader.
// All widths are in distance-field units (relative to the font's DistanceRange).
type SDFEffects struct {
	OutlineWidth   float64 // 0 = no outline
	OutlineColor   Color
	GlowWidth      float64 // 0 = no glow
	GlowColor      Color
	ShadowOffset   Vec2 // (0,0) = no shadow
	ShadowColor    Color
	ShadowSoftness float64
}

// SpriteFont renders text from a pre-generated SDF or MSDF atlas.
// It implements the Font interface for measurement and layout.
type SpriteFont struct {
	lineHeight    float64
	base          float64
	fontSize      float64 // design size SDF was generated at
	distanceRange float64 // pixel range of distance field (typically 4-8)
	multiChannel  bool    // true = MSDF, false = SDF
	page          uint16  // atlas page index

	asciiGlyphs [asciiGlyphCount]glyph // fixed array for ASCII, zero-alloc lookup
	asciiSet    [asciiGlyphCount]bool  // which ASCII entries are populated
	extGlyphs   map[rune]*glyph        // extended Unicode

	kernings map[[2]rune]int16
}

// MeasureString returns the pixel width and height of the rendered text.
func (f *SpriteFont) MeasureString(s string) (width, height float64) {
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

// LineHeight returns the vertical distance between baselines in pixels.
func (f *SpriteFont) LineHeight() float64 {
	return f.lineHeight
}

// DistanceRange returns the pixel range of the distance field.
func (f *SpriteFont) DistanceRange() float64 {
	return f.distanceRange
}

// IsMultiChannel returns true if this is an MSDF font (multi-channel SDF).
func (f *SpriteFont) IsMultiChannel() bool {
	return f.multiChannel
}

// glyph returns the glyph for the given rune, or nil if not found.
func (f *SpriteFont) glyph(r rune) *glyph {
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
func (f *SpriteFont) kern(first, second rune) int16 {
	if f.kernings == nil {
		return 0
	}
	return f.kernings[[2]rune{first, second}]
}

// --- SDF metrics JSON loading ---

// sdfMetrics is the JSON structure for SDF font metrics.
type sdfMetrics struct {
	Type          string          `json:"type"`
	Size          float64         `json:"size"`
	DistanceRange float64         `json:"distanceRange"`
	LineHeight    float64         `json:"lineHeight"`
	Base          float64         `json:"base"`
	Glyphs        []sdfGlyphEntry `json:"glyphs"`
	Kernings      []sdfKernEntry  `json:"kernings"`
}

type sdfGlyphEntry struct {
	ID       int `json:"id"`
	X        int `json:"x"`
	Y        int `json:"y"`
	Width    int `json:"width"`
	Height   int `json:"height"`
	XOffset  int `json:"xOffset"`
	YOffset  int `json:"yOffset"`
	XAdvance int `json:"xAdvance"`
}

type sdfKernEntry struct {
	First  int   `json:"first"`
	Second int   `json:"second"`
	Amount int16 `json:"amount"`
}

// LoadSpriteFont parses SDF font metrics JSON and returns an SpriteFont.
// The pageIndex specifies which atlas page the SDF glyphs are on.
// Register the atlas page image on the Scene via Scene.RegisterPage.
func LoadSpriteFont(jsonData []byte, pageIndex uint16) (*SpriteFont, error) {
	var m sdfMetrics
	if err := json.Unmarshal(jsonData, &m); err != nil {
		return nil, fmt.Errorf("willow: failed to parse SDF metrics JSON: %w", err)
	}

	if m.LineHeight == 0 {
		return nil, fmt.Errorf("willow: SDF metrics missing lineHeight")
	}
	if len(m.Glyphs) == 0 {
		return nil, fmt.Errorf("willow: SDF metrics has no glyph definitions")
	}
	if m.DistanceRange <= 0 {
		m.DistanceRange = 6 // sensible default
	}

	f := &SpriteFont{
		lineHeight:    m.LineHeight,
		base:          m.Base,
		fontSize:      m.Size,
		distanceRange: m.DistanceRange,
		multiChannel:  m.Type == "msdf",
		page:          pageIndex,
	}

	for _, ge := range m.Glyphs {
		g := glyph{
			id:       rune(ge.ID),
			x:        uint16(ge.X),
			y:        uint16(ge.Y),
			width:    uint16(ge.Width),
			height:   uint16(ge.Height),
			xOffset:  int16(ge.XOffset),
			yOffset:  int16(ge.YOffset),
			xAdvance: int16(ge.XAdvance),
			page:     pageIndex,
		}
		if g.id >= 0 && g.id < asciiGlyphCount {
			f.asciiGlyphs[g.id] = g
			f.asciiSet[g.id] = true
		} else {
			if f.extGlyphs == nil {
				f.extGlyphs = make(map[rune]*glyph)
			}
			g := g
			f.extGlyphs[g.id] = &g
		}
	}

	for _, ke := range m.Kernings {
		if f.kernings == nil {
			f.kernings = make(map[[2]rune]int16)
		}
		f.kernings[[2]rune{rune(ke.First), rune(ke.Second)}] = ke.Amount
	}

	return f, nil
}

// LoadSpriteFontFromTTF generates an SDF font atlas at runtime from TTF/OTF data.
// Uses pure-Go rasterization (golang.org/x/image/font/opentype) so it can be
// called before the Ebitengine game loop starts. Returns the SpriteFont, the atlas
// as an ebiten.Image (caller must register via Scene.RegisterPage), the raw
// atlas as an *image.NRGBA (for saving to disk), and any error.
func LoadSpriteFontFromTTF(ttfData []byte, opts SDFGenOptions) (*SpriteFont, *ebiten.Image, *image.NRGBA, error) {
	opts.Defaults()

	// Parse font with golang.org/x/image/font/opentype (pure Go, no GPU needed).
	otFont, err := opentype.Parse(ttfData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("willow: failed to parse TTF data: %w", err)
	}

	face, err := opentype.NewFace(otFont, &opentype.FaceOptions{
		Size:    opts.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("willow: failed to create font face: %w", err)
	}
	defer face.Close()

	faceMetrics := face.Metrics()
	ascent := fixedToFloat64(faceMetrics.Ascent)
	descent := fixedToFloat64(faceMetrics.Descent)
	lineHeight := fixedToFloat64(faceMetrics.Height)
	if lineHeight <= 0 {
		lineHeight = ascent + descent
	}

	// Rasterize each glyph using pure-Go font.Drawer → image.RGBA.
	var glyphs []GlyphBitmap
	for i := 0; i < len(opts.Chars); {
		r, sz := utf8.DecodeRuneInString(opts.Chars[i:])
		i += sz

		bounds, advance, ok := face.GlyphBounds(r)
		if !ok {
			continue
		}

		x0 := bounds.Min.X.Floor()
		y0 := bounds.Min.Y.Floor()
		x1 := bounds.Max.X.Ceil()
		y1 := bounds.Max.Y.Ceil()
		gw := x1 - x0
		gh := y1 - y0

		if gw <= 0 || gh <= 0 {
			// Whitespace or empty glyph — record advance only.
			glyphs = append(glyphs, GlyphBitmap{
				ID:       r,
				Img:      nil,
				XOffset:  0,
				YOffset:  0,
				XAdvance: advance.Ceil(),
			})
			continue
		}

		// Rasterize to RGBA using font.Drawer (CPU only, no Ebitengine).
		rgba := image.NewRGBA(image.Rect(0, 0, gw, gh))
		d := font.Drawer{
			Dst:  rgba,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{X: -bounds.Min.X, Y: -bounds.Min.Y},
		}
		d.DrawString(string(r))

		// Extract alpha channel directly from RGBA pixel data.
		alphaImg := image.NewAlpha(image.Rect(0, 0, gw, gh))
		for py := 0; py < gh; py++ {
			for px := 0; px < gw; px++ {
				off := rgba.PixOffset(px, py)
				alphaImg.Pix[py*alphaImg.Stride+px] = rgba.Pix[off+3]
			}
		}

		glyphs = append(glyphs, GlyphBitmap{
			ID:       r,
			Img:      alphaImg,
			XOffset:  x0,
			YOffset:  y0 + faceMetrics.Ascent.Ceil(),
			XAdvance: advance.Ceil(),
		})
	}

	atlas, metricsJSON, err := GenerateSDFFromBitmaps(glyphs, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("willow: SDF generation failed: %w", err)
	}

	// Override lineHeight and base with actual font metrics.
	var m sdfMetrics
	_ = json.Unmarshal(metricsJSON, &m)
	m.LineHeight = lineHeight
	m.Base = ascent
	metricsJSON, _ = json.Marshal(m)

	sdfFont, err := LoadSpriteFont(metricsJSON, opts.PageIndex)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("willow: failed to load generated SDF font: %w", err)
	}

	// Convert image.NRGBA → ebiten.Image (WritePixels works before game start).
	atlasImg := ebiten.NewImage(atlas.Bounds().Dx(), atlas.Bounds().Dy())
	atlasImg.WritePixels(atlas.Pix)

	return sdfFont, atlasImg, atlas, nil
}

// NewFontFromTTF generates an SDF font from TTF/OTF data, registers the atlas
// page, and returns the font ready to use:
//
//	font, err := willow.NewFontFromTTF(ttfData, 80)
//	label := willow.NewText("label", "Hello!", font)
//
// Size is the rasterization size in pixels (higher = better quality when scaled up).
// Use 0 for the default (80px).
func NewFontFromTTF(ttfData []byte, size float64) (*SpriteFont, error) {
	am := atlasManager()
	pageIndex := am.AllocPage()
	sdfFont, atlasImg, _, err := LoadSpriteFontFromTTF(ttfData, SDFGenOptions{
		Size:      size,
		PageIndex: uint16(pageIndex),
	})
	if err != nil {
		return nil, err
	}
	am.RegisterPage(pageIndex, atlasImg)
	return sdfFont, nil
}

func fixedToFloat64(f fixed.Int26_6) float64 {
	return float64(f) / 64.0
}
