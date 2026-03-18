package text

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	_ "image/png"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// FontStyle identifies a typographic style variant.
type FontStyle int

const (
	FontStyleRegular FontStyle = iota
	FontStyleBold
	FontStyleItalic
	FontStyleBoldItalic
)

// AtlasEntry holds the raw PNG and JSON bytes for one baked atlas (style + bake size).
type AtlasEntry struct {
	Style    FontStyle
	BakeSize float64
	PNG      []byte
	JSON     []byte
}

// FontFamilyConfig configures a FontFamily created from TTF/OTF data.
type FontFamilyConfig struct {
	Regular    []byte    // required — TTF/OTF data for the regular style
	Bold       []byte    // optional — TTF/OTF data for bold
	Italic     []byte    // optional — TTF/OTF data for italic
	BoldItalic []byte    // optional — TTF/OTF data for bold-italic
	BakeSizes  []float64 // optional — defaults to [64, 256]
}

// fontStyleSlot holds distanceFieldFont entries for a single style, sorted ascending by bake size.
type fontStyleSlot struct {
	entries []fontEntry
}

// fontEntry pairs a bake size with the font rendered at that size.
type fontEntry struct {
	bake float64
	font *distanceFieldFont
	raw  AtlasEntry
}

// resolve returns the best distanceFieldFont for the given display size.
//
// SDF quality depends on the ratio of atlas pixels to display pixels at the
// glyph edge. Each SDF pixel encodes a distance sample; more samples per
// display pixel lets the shader interpolate the edge position with sub-pixel
// precision, producing smoother anti-aliasing. At a 1:1 ratio the shader has
// only one distance sample per pixel, so edge transitions are coarse.
//
// A 4:1 ratio (bake >= displaySize*4) gives 16 samples per display pixel,
// which is enough for visually smooth curves at all common font sizes. Below
// that threshold quality degrades noticeably, especially on diagonal strokes.
//
// Falls back to the largest available bake when no bake meets the threshold.
func (s *fontStyleSlot) resolve(displaySize float64) *distanceFieldFont {
	threshold := displaySize * 4
	for _, e := range s.entries {
		if e.bake >= threshold {
			return e.font
		}
	}
	if len(s.entries) > 0 {
		return s.entries[len(s.entries)-1].font
	}
	return nil
}

// FontFamily holds SDF atlases for multiple styles and bake sizes, or wraps a pixel font.
// It is the only public font type in willow.
type FontFamily struct {
	regular    fontStyleSlot
	bold       *fontStyleSlot
	italic     *fontStyleSlot
	boldItalic *fontStyleSlot
	pixel      *pixelFont
}

// resolveDF returns the best distanceFieldFont for the given display size and style.
func (ff *FontFamily) resolveDF(displaySize float64, bold, italic bool) *distanceFieldFont {
	if bold && italic {
		if ff.boldItalic != nil {
			if f := ff.boldItalic.resolve(displaySize); f != nil {
				return f
			}
		}
		if ff.bold != nil {
			if f := ff.bold.resolve(displaySize); f != nil {
				return f
			}
		}
	} else if bold {
		if ff.bold != nil {
			if f := ff.bold.resolve(displaySize); f != nil {
				return f
			}
		}
	} else if italic {
		if ff.italic != nil {
			if f := ff.italic.resolve(displaySize); f != nil {
				return f
			}
		}
	}
	return ff.regular.resolve(displaySize)
}

func (ff *FontFamily) resolvePixel() *pixelFont { return ff.pixel }
func (ff *FontFamily) isPixelMode() bool        { return ff.pixel != nil }

func (ff *FontFamily) lineHeight(displaySize float64, bold, italic bool) float64 {
	if ff.isPixelMode() {
		return ff.pixel.LineHeight()
	}
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0
	}
	return f.LineHeight()
}

func (ff *FontFamily) measureString(text string, displaySize float64, bold, italic bool) (w, h float64) {
	if ff.isPixelMode() {
		return ff.pixel.MeasureString(text)
	}
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0, 0
	}
	return f.MeasureString(text)
}

// --- Exported measurement methods ---

// LineHeight returns the font's native line height in atlas pixels for the resolved style.
func (ff *FontFamily) LineHeight(displaySize float64, bold, italic bool) float64 {
	return ff.lineHeight(displaySize, bold, italic)
}

// MeasureString returns the width and height of text in atlas pixels for the resolved style.
func (ff *FontFamily) MeasureString(text string, displaySize float64, bold, italic bool) (w, h float64) {
	return ff.measureString(text, displaySize, bold, italic)
}

// TTFData returns the original TTF/OTF byte data for the resolved style variant.
// Returns nil for pixel fonts or if no TTF data was retained.
func (ff *FontFamily) TTFData(bold, italic bool) []byte {
	if ff.isPixelMode() {
		return nil
	}
	// Use a large displaySize so we always get a font (doesn't matter which bake
	// since they all share the same TTF data).
	f := ff.resolveDF(256, bold, italic)
	if f == nil {
		return nil
	}
	return f.TTFData()
}

// --- Exported render accessors (used by internal/render) ---

// IsPixelMode returns true if this FontFamily wraps a pixel spritesheet font.
func (ff *FontFamily) IsPixelMode() bool { return ff.isPixelMode() }

// SDFPage returns the atlas page index for the resolved SDF font.
func (ff *FontFamily) SDFPage(displaySize float64, bold, italic bool) uint16 {
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0
	}
	return f.Page()
}

// SDFLineHeight returns the line height for the resolved SDF font.
func (ff *FontFamily) SDFLineHeight(displaySize float64, bold, italic bool) float64 {
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0
	}
	return f.LineHeight()
}

// SDFDistanceRange returns the distance range for the resolved SDF font.
func (ff *FontFamily) SDFDistanceRange(displaySize float64, bold, italic bool) float64 {
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0
	}
	return f.DistanceRange()
}

// SDFIsMultiChannel returns true if the resolved SDF font is MSDF.
func (ff *FontFamily) SDFIsMultiChannel(displaySize float64, bold, italic bool) bool {
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return false
	}
	return f.IsMultiChannel()
}

// SDFFontSize returns the design bake size for the resolved SDF font.
func (ff *FontFamily) SDFFontSize(displaySize float64, bold, italic bool) float64 {
	f := ff.resolveDF(displaySize, bold, italic)
	if f == nil {
		return 0
	}
	return f.FontSize()
}

// PixelPage returns the atlas page for the pixel font.
func (ff *FontFamily) PixelPage() uint16 {
	if ff.pixel == nil {
		return 0
	}
	return ff.pixel.Page
}

// PixelCellSize returns cellW, cellH for the pixel font.
func (ff *FontFamily) PixelCellSize() (int, int) {
	if ff.pixel == nil {
		return 0, 0
	}
	return ff.pixel.CellW, ff.pixel.CellH
}

// --- Constructors ---

var defaultBakeSizes = []float64{64, 256}

// bakeStyle bakes SDF atlases for each bake size. Page registration is conditional.
func bakeStyle(ttfData []byte, bakeSizes []float64, style FontStyle) (*fontStyleSlot, error) {
	slot := &fontStyleSlot{}
	for _, size := range bakeSizes {
		opts := SDFGenOptions{
			Size:          size,
			DistanceRange: size * 0.125,
		}
		if AllocPageFn != nil {
			opts.PageIndex = uint16(AllocPageFn())
		}
		f, atlasImg, atlasRaw, jsonBytes, err := LoadDistanceFieldFontFromTTF(ttfData, opts)
		if err != nil {
			return nil, fmt.Errorf("bake %.0fpx: %w", size, err)
		}
		f.SetTTFData(ttfData)

		// Encode atlas to PNG bytes for Atlases() access.
		var pngBuf bytes.Buffer
		if err := png.Encode(&pngBuf, atlasRaw); err != nil {
			return nil, fmt.Errorf("bake %.0fpx encode PNG: %w", size, err)
		}

		if AllocPageFn != nil && RegisterPageFn != nil {
			RegisterPageFn(int(opts.PageIndex), atlasImg)
		}

		slot.entries = append(slot.entries, fontEntry{
			bake: size,
			font: f,
			raw:  AtlasEntry{Style: style, BakeSize: size, PNG: pngBuf.Bytes(), JSON: jsonBytes},
		})
	}
	sort.Slice(slot.entries, func(i, j int) bool {
		return slot.entries[i].bake < slot.entries[j].bake
	})
	return slot, nil
}

// NewFontFamilyFromTTF creates a FontFamily from TTF/OTF data.
// All style variants are provided in the config struct. BakeSizes defaults to [64, 256].
func NewFontFamilyFromTTF(cfg FontFamilyConfig) (*FontFamily, error) {
	if len(cfg.Regular) == 0 {
		return nil, fmt.Errorf("fontfamily: Regular TTF data is required")
	}
	bakeSizes := cfg.BakeSizes
	if len(bakeSizes) == 0 {
		bakeSizes = defaultBakeSizes
	}

	reg, err := bakeStyle(cfg.Regular, bakeSizes, FontStyleRegular)
	if err != nil {
		return nil, fmt.Errorf("fontfamily: regular: %w", err)
	}
	ff := &FontFamily{regular: *reg}

	if len(cfg.Bold) > 0 {
		slot, err := bakeStyle(cfg.Bold, bakeSizes, FontStyleBold)
		if err != nil {
			return nil, fmt.Errorf("fontfamily: bold: %w", err)
		}
		ff.bold = slot
	}
	if len(cfg.Italic) > 0 {
		slot, err := bakeStyle(cfg.Italic, bakeSizes, FontStyleItalic)
		if err != nil {
			return nil, fmt.Errorf("fontfamily: italic: %w", err)
		}
		ff.italic = slot
	}
	if len(cfg.BoldItalic) > 0 {
		slot, err := bakeStyle(cfg.BoldItalic, bakeSizes, FontStyleBoldItalic)
		if err != nil {
			return nil, fmt.Errorf("fontfamily: bold-italic: %w", err)
		}
		ff.boldItalic = slot
	}

	return ff, nil
}

// NewFontFamilyFromPixelFont wraps a pixel spritesheet into a FontFamily.
func NewFontFamilyFromPixelFont(img *ebiten.Image, cellW, cellH int, chars string) *FontFamily {
	pageIndex := 0
	if AllocPageFn != nil {
		pageIndex = AllocPageFn()
	}
	if RegisterPageFn != nil {
		RegisterPageFn(pageIndex, img)
	}

	bounds := img.Bounds()
	cols := bounds.Dx() / cellW

	pf := &pixelFont{
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
		pf.AddGlyph(r, uint16(col*cellW), uint16(row*cellH))
		idx++
	}

	return &FontFamily{pixel: pf}
}

// NewFontFamilyFromFontBundle loads a pre-baked .fontbundle archive and returns a FontFamily.
// A .fontbundle is a zip with entries named {style}_{size}.png / {style}_{size}.json.
// For file loading, chain with os.ReadFile:
//
//	ff, err := willow.NewFontFamilyFromFontBundle(must(os.ReadFile("my.fontbundle")))
func NewFontFamilyFromFontBundle(data []byte) (*FontFamily, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("fontbundle: %w", err)
	}

	pngs := make(map[string][]byte)
	jsons := make(map[string][]byte)
	ttfs := make(map[string][]byte) // {style}.ttf entries
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("fontbundle: open %s: %w", f.Name, err)
		}
		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("fontbundle: read %s: %w", f.Name, err)
		}
		switch {
		case strings.HasSuffix(f.Name, ".png"):
			pngs[strings.TrimSuffix(f.Name, ".png")] = buf.Bytes()
		case strings.HasSuffix(f.Name, ".json"):
			jsons[strings.TrimSuffix(f.Name, ".json")] = buf.Bytes()
		case strings.HasSuffix(f.Name, ".ttf"):
			ttfs[strings.TrimSuffix(f.Name, ".ttf")] = buf.Bytes()
		}
	}

	ff := &FontFamily{}
	for base, jsonData := range jsons {
		pngData, ok := pngs[base]
		if !ok {
			return nil, fmt.Errorf("fontbundle: missing PNG for %s.json", base)
		}
		bake, style, err := bundleBaseName(base)
		if err != nil {
			return nil, fmt.Errorf("fontbundle: %w", err)
		}
		rawImg, _, err := image.Decode(bytes.NewReader(pngData))
		if err != nil {
			return nil, fmt.Errorf("fontbundle: decode PNG %s: %w", base, err)
		}
		// Convert grayscale → NRGBA with gray value in alpha (SDF shader reads alpha).
		// Also handles NRGBA PNGs from older bundles.
		nrgba := grayToNRGBA(rawImg)
		eImg := ebiten.NewImageFromImage(nrgba)
		pageIndex := uint16(0)
		if AllocPageFn != nil {
			pageIndex = uint16(AllocPageFn())
		}
		if RegisterPageFn != nil {
			RegisterPageFn(int(pageIndex), eImg)
		}
		f, err := loadDistanceFieldFont(jsonData, pageIndex)
		if err != nil {
			return nil, fmt.Errorf("fontbundle: load font %s: %w", base, err)
		}

		// Set TTF data if the bundle includes it (for offscreen rendering).
		if ttfData, ok := ttfs[style]; ok {
			f.SetTTFData(ttfData)
		}

		fsStyle := FontStyleRegular
		switch style {
		case "bold":
			fsStyle = FontStyleBold
		case "italic":
			fsStyle = FontStyleItalic
		case "bold_italic":
			fsStyle = FontStyleBoldItalic
		}

		entry := fontEntry{
			bake: bake,
			font: f,
			raw:  AtlasEntry{Style: fsStyle, BakeSize: bake, PNG: pngData, JSON: jsonData},
		}
		switch style {
		case "regular":
			ff.regular.entries = append(ff.regular.entries, entry)
		case "bold":
			if ff.bold == nil {
				ff.bold = &fontStyleSlot{}
			}
			ff.bold.entries = append(ff.bold.entries, entry)
		case "italic":
			if ff.italic == nil {
				ff.italic = &fontStyleSlot{}
			}
			ff.italic.entries = append(ff.italic.entries, entry)
		case "bold_italic":
			if ff.boldItalic == nil {
				ff.boldItalic = &fontStyleSlot{}
			}
			ff.boldItalic.entries = append(ff.boldItalic.entries, entry)
		default:
			return nil, fmt.Errorf("fontbundle: unknown style %q in %s", style, base)
		}
	}

	if len(ff.regular.entries) == 0 {
		return nil, fmt.Errorf("fontbundle: no regular style found")
	}
	sortSlot(&ff.regular)
	if ff.bold != nil {
		sortSlot(ff.bold)
	}
	if ff.italic != nil {
		sortSlot(ff.italic)
	}
	if ff.boldItalic != nil {
		sortSlot(ff.boldItalic)
	}
	return ff, nil
}

// TrimCell trims pixels from each side of every pixel-font cell.
// No-op if this FontFamily was not created from a pixel font.
func (ff *FontFamily) TrimCell(top, right, bottom, left int) {
	if ff.pixel != nil {
		ff.pixel.TrimCell(top, right, bottom, left)
	}
}

// Atlases returns all retained raw PNG+JSON atlas entries (for fontgen CLI).
func (ff *FontFamily) Atlases() []AtlasEntry {
	var out []AtlasEntry
	for _, e := range ff.regular.entries {
		out = append(out, e.raw)
	}
	if ff.bold != nil {
		for _, e := range ff.bold.entries {
			out = append(out, e.raw)
		}
	}
	if ff.italic != nil {
		for _, e := range ff.italic.entries {
			out = append(out, e.raw)
		}
	}
	if ff.boldItalic != nil {
		for _, e := range ff.boldItalic.entries {
			out = append(out, e.raw)
		}
	}
	return out
}

// --- Helpers ---

func sortSlot(s *fontStyleSlot) {
	sort.Slice(s.entries, func(i, j int) bool {
		return s.entries[i].bake < s.entries[j].bake
	})
}

// grayToNRGBA converts a decoded PNG to NRGBA with the SDF distance in the alpha channel.
// Grayscale PNGs (from fontgen) have the distance in the gray channel → move to alpha.
// NRGBA PNGs (legacy bundles) already have distance in alpha → pass through.
func grayToNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	switch img := src.(type) {
	case *image.NRGBA:
		return img
	case *image.Gray:
		out := image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				v := img.GrayAt(x, y).Y
				off := out.PixOffset(x, y)
				out.Pix[off+0] = 255 // R
				out.Pix[off+1] = 255 // G
				out.Pix[off+2] = 255 // B
				out.Pix[off+3] = v   // A = distance
			}
		}
		return out
	default:
		// Fallback: use standard conversion.
		out := image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				_, _, _, a := src.At(x, y).RGBA()
				off := out.PixOffset(x, y)
				out.Pix[off+0] = 255
				out.Pix[off+1] = 255
				out.Pix[off+2] = 255
				out.Pix[off+3] = uint8(a >> 8)
			}
		}
		return out
	}
}

func bundleBaseName(base string) (float64, string, error) {
	idx := strings.LastIndex(base, "_")
	if idx < 0 {
		return 0, "", fmt.Errorf("invalid entry name %q (expected {style}_{size})", base)
	}
	size, err := strconv.ParseFloat(base[idx+1:], 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid size in entry name %q: %w", base, err)
	}
	return size, base[:idx], nil
}
