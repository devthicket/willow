package text

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// SpriteFont renders text from a pre-generated SDF or MSDF atlas.
// It implements the Font interface for measurement and layout.
type SpriteFont struct {
	lineHeight    float64
	base          float64
	fontSize      float64 // design size SDF was generated at
	distanceRange float64 // pixel range of distance field (typically 4-8)
	multiChannel  bool    // true = MSDF, false = SDF
	page          uint16  // atlas page index

	asciiGlyphs [AsciiGlyphCount]Glyph // fixed array for ASCII, zero-alloc lookup
	asciiSet    [AsciiGlyphCount]bool   // which ASCII entries are populated
	extGlyphs   map[rune]*Glyph        // extended Unicode

	kernings map[[2]rune]int16

	ttfData []byte // original TTF data (retained for offscreen text rendering)
}

// MeasureString returns the pixel width and height of the rendered text
// in native atlas pixels.
func (f *SpriteFont) MeasureString(s string) (width, height float64) {
	var maxW float64
	var cursorX float64
	var prevRune rune
	var hasPrev bool
	lines := 1

	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		i += sz

		if r == '\n' {
			if cursorX > maxW {
				maxW = cursorX
			}
			cursorX = 0
			lines++
			hasPrev = false
			continue
		}

		g := f.GlyphLookup(r)
		if g == nil {
			hasPrev = false
			continue
		}

		if hasPrev {
			cursorX += float64(f.Kern(prevRune, r))
		}
		cursorX += float64(g.XAdvance)
		prevRune = r
		hasPrev = true
	}

	if cursorX > maxW {
		maxW = cursorX
	}
	return maxW, float64(lines) * f.lineHeight
}

// LineHeight returns the vertical distance between baselines in native atlas pixels.
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

// TTFData returns the original TTF/OTF byte data, or nil if the font was not
// created from TTF.
func (f *SpriteFont) TTFData() []byte {
	return f.ttfData
}

// SetTTFData sets the TTF data on the font.
func (f *SpriteFont) SetTTFData(data []byte) {
	f.ttfData = data
}

// FontSize returns the design size (in pixels) that the SDF was generated at.
func (f *SpriteFont) FontSize() float64 {
	return f.fontSize
}

// Page returns the atlas page index for this font.
func (f *SpriteFont) Page() uint16 {
	return f.page
}

// GlyphLookup returns the glyph for the given rune, or nil if not found.
func (f *SpriteFont) GlyphLookup(r rune) *Glyph {
	if r >= 0 && r < AsciiGlyphCount {
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

// Kern returns the kerning amount for the given rune pair.
func (f *SpriteFont) Kern(first, second rune) int16 {
	if f.kernings == nil {
		return 0
	}
	return f.kernings[[2]rune{first, second}]
}

// --- SDF metrics JSON types ---

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

// LoadSpriteFont parses SDF font metrics JSON and returns a SpriteFont.
// The pageIndex specifies which atlas page the SDF glyphs are on.
func LoadSpriteFont(jsonData []byte, pageIndex uint16) (*SpriteFont, error) {
	var m sdfMetrics
	if err := json.Unmarshal(jsonData, &m); err != nil {
		return nil, fmt.Errorf("text: failed to parse SDF metrics JSON: %w", err)
	}

	if m.LineHeight == 0 {
		return nil, fmt.Errorf("text: SDF metrics missing lineHeight")
	}
	if len(m.Glyphs) == 0 {
		return nil, fmt.Errorf("text: SDF metrics has no glyph definitions")
	}
	if m.DistanceRange <= 0 {
		m.DistanceRange = 6
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
		g := Glyph{
			ID:       rune(ge.ID),
			X:        uint16(ge.X),
			Y:        uint16(ge.Y),
			Width:    uint16(ge.Width),
			Height:   uint16(ge.Height),
			XOffset:  int16(ge.XOffset),
			YOffset:  int16(ge.YOffset),
			XAdvance: int16(ge.XAdvance),
			Page:     pageIndex,
		}
		if g.ID >= 0 && g.ID < AsciiGlyphCount {
			f.asciiGlyphs[g.ID] = g
			f.asciiSet[g.ID] = true
		} else {
			if f.extGlyphs == nil {
				f.extGlyphs = make(map[rune]*Glyph)
			}
			g := g
			f.extGlyphs[g.ID] = &g
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
