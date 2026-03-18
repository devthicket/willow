// Package bake converts TTF/OTF font data into a .fontbundle archive.
// It supports multiple styles (regular, bold, italic, bold-italic) and multiple
// bake sizes, producing a zip containing {style}_{size}.png + {style}_{size}.json.
package bake

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/devthicket/willow"
	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/devthicket/willow/cmd/fontgen/internal/kerning"
)

// StyleData holds the TTF bytes and label for one font variant.
type StyleData struct {
	Label   string // "regular", "bold", "italic", "bold_italic"
	TTFData []byte
}

// Options controls atlas generation.
type Options struct {
	BakeSizes     []float64 // e.g. [64, 256]
	Charset       string    // characters to bake
	AtlasWidth    int       // max atlas width in pixels (default 1024)
	DistanceRange float64   // if > 0, overrides auto-scaled distanceRange (size × 0.125)
}

// atlasMeta mirrors internal/text.sdfMetrics for JSON marshaling.
type atlasMeta struct {
	Type          string      `json:"type"`
	Size          float64     `json:"size"`
	DistanceRange float64     `json:"distanceRange"`
	LineHeight    float64     `json:"lineHeight"`
	Base          float64     `json:"base"`
	Glyphs        []glyphMeta `json:"glyphs"`
	Kernings      []kernPair  `json:"kernings"`
}

type glyphMeta struct {
	ID       int `json:"id"`
	X        int `json:"x"`
	Y        int `json:"y"`
	Width    int `json:"width"`
	Height   int `json:"height"`
	XOffset  int `json:"xOffset"`
	YOffset  int `json:"yOffset"`
	XAdvance int `json:"xAdvance"`
}

type kernPair struct {
	First  int   `json:"first"`
	Second int   `json:"second"`
	Amount int16 `json:"amount"`
}

// Bundle bakes all provided styles at all bake sizes and returns the .fontbundle
// archive bytes along with a human-readable summary.
func Bundle(styles []StyleData, opts Options) ([]byte, error) {
	if len(opts.BakeSizes) == 0 {
		opts.BakeSizes = []float64{64, 256}
	}
	if opts.Charset == "" {
		var b []byte
		for c := byte(32); c <= 126; c++ {
			b = append(b, c)
		}
		opts.Charset = string(b)
	}
	if opts.AtlasWidth == 0 {
		opts.AtlasWidth = 1024
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, style := range styles {
		for _, size := range opts.BakeSizes {
			distRange := size * 0.125
			if opts.DistanceRange > 0 {
				distRange = opts.DistanceRange
			}

			atlasImg, metaJSON, err := bakeOne(style.TTFData, size, distRange, opts.Charset, opts.AtlasWidth)
			if err != nil {
				return nil, fmt.Errorf("bake %s %.0fpx: %w", style.Label, size, err)
			}

			baseName := style.Label + "_" + formatSize(size)

			// Write PNG (deflate-compressed).
			pngW, err := zw.CreateHeader(&zip.FileHeader{
				Name:   baseName + ".png",
				Method: zip.Deflate,
			})
			if err != nil {
				return nil, err
			}
			if err := png.Encode(pngW, atlasImg); err != nil {
				return nil, err
			}

			// Write JSON (deflate-compressed).
			jsonW, err := zw.CreateHeader(&zip.FileHeader{
				Name:   baseName + ".json",
				Method: zip.Deflate,
			})
			if err != nil {
				return nil, err
			}
			if _, err := jsonW.Write(metaJSON); err != nil {
				return nil, err
			}

			glyphCount := countGlyphs(metaJSON)
			fmt.Printf("  %-12s %4.0fpx  %dx%d  %d glyphs\n",
				style.Label, size, atlasImg.Bounds().Dx(), atlasImg.Bounds().Dy(), glyphCount)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// bakeOne rasterizes and SDF-encodes a single style at a single bake size.
// Returns the atlas image and metrics JSON (with kerning injected).
func bakeOne(ttfData []byte, size, distRange float64, chars string, atlasWidth int) (image.Image, []byte, error) {
	// Parse the font for line metrics.
	otFont, err := opentype.Parse(ttfData)
	if err != nil {
		return nil, nil, err
	}
	face, err := opentype.NewFace(otFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return nil, nil, err
	}
	defer face.Close()

	faceMetrics := face.Metrics()
	ascent := fixedToFloat(faceMetrics.Ascent)
	lineHeight := fixedToFloat(faceMetrics.Height)
	if lineHeight <= 0 {
		lineHeight = ascent + fixedToFloat(faceMetrics.Descent)
	}

	// Rasterize glyphs.
	glyphs := rasterizeGlyphs(face, chars, faceMetrics)

	// Generate SDF atlas (no kerning yet).
	atlasNRGBA, rawJSON, err := willow.GenerateSDFFromBitmaps(glyphs, willow.SDFGenOptions{
		Size:          size,
		DistanceRange: distRange,
		Chars:         chars,
		AtlasWidth:    atlasWidth,
	})
	if err != nil {
		return nil, nil, err
	}

	// Convert NRGBA → Gray (SDF only uses the alpha channel).
	bounds := atlasNRGBA.Bounds()
	atlasImg := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			off := atlasNRGBA.PixOffset(x, y)
			atlasImg.Pix[(y-bounds.Min.Y)*atlasImg.Stride+(x-bounds.Min.X)] = atlasNRGBA.Pix[off+3] // alpha channel
		}
	}

	// Override lineHeight and base with actual font metrics.
	var meta atlasMeta
	if err := json.Unmarshal(rawJSON, &meta); err != nil {
		return nil, nil, err
	}
	meta.LineHeight = lineHeight
	meta.Base = ascent

	// Extract and inject kerning pairs.
	kpairs, err := kerning.Extract(ttfData, chars, size)
	if err == nil && len(kpairs) > 0 {
		meta.Kernings = make([]kernPair, len(kpairs))
		for i, kp := range kpairs {
			meta.Kernings[i] = kernPair{First: kp.First, Second: kp.Second, Amount: kp.Amount}
		}
	}

	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return atlasImg, metaJSON, nil
}

// rasterizeGlyphs renders each character using the font face and returns GlyphBitmaps.
func rasterizeGlyphs(face xfont.Face, chars string, metrics xfont.Metrics) []willow.GlyphBitmap {
	var glyphs []willow.GlyphBitmap
	for i := 0; i < len(chars); {
		r, sz := utf8.DecodeRuneInString(chars[i:])
		i += sz

		bounds, advance, ok := face.GlyphBounds(r)
		if !ok {
			continue
		}

		x0 := bounds.Min.X.Floor()
		y0 := bounds.Min.Y.Floor()
		gw := bounds.Max.X.Ceil() - x0
		gh := bounds.Max.Y.Ceil() - y0

		if gw <= 0 || gh <= 0 {
			glyphs = append(glyphs, willow.GlyphBitmap{
				ID:       r,
				XAdvance: advance.Ceil(),
			})
			continue
		}

		rgba := image.NewRGBA(image.Rect(0, 0, gw, gh))
		d := xfont.Drawer{
			Dst:  rgba,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{X: -bounds.Min.X, Y: -bounds.Min.Y},
		}
		d.DrawString(string(r))

		alpha := image.NewAlpha(image.Rect(0, 0, gw, gh))
		for py := 0; py < gh; py++ {
			for px := 0; px < gw; px++ {
				off := rgba.PixOffset(px, py)
				alpha.Pix[py*alpha.Stride+px] = rgba.Pix[off+3]
			}
		}

		glyphs = append(glyphs, willow.GlyphBitmap{
			ID:       r,
			Img:      alpha,
			XOffset:  x0,
			YOffset:  y0 + metrics.Ascent.Ceil(),
			XAdvance: advance.Ceil(),
		})
	}
	return glyphs
}

func fixedToFloat(v fixed.Int26_6) float64 {
	return float64(v) / 64.0
}

// formatSize formats a bake size as a clean string: 64.0 → "64", 48.5 → "48.5".
func formatSize(s float64) string {
	if s == math.Trunc(s) {
		return strconv.Itoa(int(s))
	}
	return strconv.FormatFloat(s, 'f', -1, 64)
}

func countGlyphs(jsonData []byte) int {
	var m struct {
		Glyphs []any `json:"glyphs"`
	}
	_ = json.Unmarshal(jsonData, &m)
	return len(m.Glyphs)
}

// CharsetFor returns the rune string for the named preset.
func CharsetFor(name string) string {
	switch name {
	case "ascii":
		var b []byte
		for c := byte(32); c <= 126; c++ {
			b = append(b, c)
		}
		return string(b)
	case "latin":
		return buildLatinCharset()
	case "full":
		return "" // special sentinel: caller handles full-font charset
	default:
		return name // literal chars or file content
	}
}

// buildLatinCharset returns extended Latin characters.
func buildLatinCharset() string {
	// ASCII + Latin-1 Supplement + Latin Extended-A + Latin Extended-B.
	var rs []rune
	for r := rune(32); r <= 126; r++ {
		rs = append(rs, r)
	}
	for r := rune(0xC0); r <= 0x24F; r++ {
		rs = append(rs, r)
	}
	return string(rs)
}

// FullCharset extracts all glyphs present in a TTF file.
func FullCharset(ttfData []byte) (string, error) {
	f, err := sfnt.Parse(ttfData)
	if err != nil {
		return "", err
	}
	n := f.NumGlyphs()
	var b sfnt.Buffer
	var rs []rune
	// Scan common Unicode ranges for glyphs present in the font.
	for r := rune(32); r <= rune(0x10FFFF); r++ {
		if len(rs) >= n {
			break
		}
		g, err := f.GlyphIndex(&b, r)
		if err == nil && g > 0 {
			rs = append(rs, r)
		}
	}
	return string(rs), nil
}
