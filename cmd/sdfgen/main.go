// Command sdfgen generates SDF font atlases from TTF/OTF files.
//
// Usage:
//
//	sdfgen -font input.ttf -size 80 -range 8 -out output [-chars "..."] [-width 1024]
//
// Produces output.png (atlas) and output.json (metrics).
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"unicode/utf8"

	"github.com/phanxgames/willow"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func main() {
	fontPath := flag.String("font", "", "path to TTF/OTF font file (required)")
	size := flag.Float64("size", 80, "font size in pixels")
	distRange := flag.Float64("range", 8, "distance field range in pixels")
	outPrefix := flag.String("out", "output", "output file prefix (.png and .json)")
	chars := flag.String("chars", "", "characters to include (default: printable ASCII)")
	atlasWidth := flag.Int("width", 1024, "maximum atlas width in pixels")
	flag.Parse()

	if *fontPath == "" {
		fmt.Fprintln(os.Stderr, "error: -font is required")
		flag.Usage()
		os.Exit(1)
	}

	ttfData, err := os.ReadFile(*fontPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading font: %v\n", err)
		os.Exit(1)
	}

	opts := willow.SDFGenOptions{
		Size:          *size,
		DistanceRange: *distRange,
		Chars:         *chars,
		AtlasWidth:    *atlasWidth,
	}
	opts.Defaults()

	// Parse font
	otFont, err := opentype.Parse(ttfData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing font: %v\n", err)
		os.Exit(1)
	}

	face, err := opentype.NewFace(otFont, &opentype.FaceOptions{
		Size:    opts.Size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating font face: %v\n", err)
		os.Exit(1)
	}
	defer face.Close()

	metrics := face.Metrics()

	// Rasterize each glyph
	var glyphs []willow.GlyphBitmap
	for i := 0; i < len(opts.Chars); {
		r, sz := utf8.DecodeRuneInString(opts.Chars[i:])
		i += sz

		bounds, advance, ok := face.GlyphBounds(r)
		if !ok {
			continue
		}

		// Convert fixed-point bounds to pixels
		x0 := bounds.Min.X.Floor()
		y0 := bounds.Min.Y.Floor()
		x1 := bounds.Max.X.Ceil()
		y1 := bounds.Max.Y.Ceil()
		gw := x1 - x0
		gh := y1 - y0

		if gw <= 0 || gh <= 0 {
			// Whitespace or empty glyph
			glyphs = append(glyphs, willow.GlyphBitmap{
				ID:       r,
				Img:      nil,
				XOffset:  0,
				YOffset:  0,
				XAdvance: advance.Ceil(),
			})
			continue
		}

		// Rasterize to RGBA, then extract alpha
		rgba := image.NewRGBA(image.Rect(0, 0, gw, gh))
		d := font.Drawer{
			Dst:  rgba,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{X: -bounds.Min.X, Y: -bounds.Min.Y},
		}
		d.DrawString(string(r))

		// Extract alpha channel
		dst := image.NewAlpha(image.Rect(0, 0, gw, gh))
		for py := 0; py < gh; py++ {
			for px := 0; px < gw; px++ {
				off := rgba.PixOffset(px, py)
				dst.Pix[py*dst.Stride+px] = rgba.Pix[off+3]
			}
		}

		glyphs = append(glyphs, willow.GlyphBitmap{
			ID:       r,
			Img:      dst,
			XOffset:  x0,
			YOffset:  y0 + metrics.Ascent.Ceil(),
			XAdvance: advance.Ceil(),
		})
	}

	atlas, metricsJSON, err := willow.GenerateSDFFromBitmaps(glyphs, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating SDF: %v\n", err)
		os.Exit(1)
	}

	// Write atlas PNG
	pngPath := *outPrefix + ".png"
	pngFile, err := os.Create(pngPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", pngPath, err)
		os.Exit(1)
	}
	if err := png.Encode(pngFile, atlas); err != nil {
		pngFile.Close()
		fmt.Fprintf(os.Stderr, "error encoding PNG: %v\n", err)
		os.Exit(1)
	}
	pngFile.Close()

	// Write metrics JSON
	jsonPath := *outPrefix + ".json"
	if err := os.WriteFile(jsonPath, metricsJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", jsonPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s (%dx%d) and %s (%d glyphs)\n",
		pngPath, atlas.Bounds().Dx(), atlas.Bounds().Dy(),
		jsonPath, len(glyphs))
}
