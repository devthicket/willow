// Command fontgen converts a TTF/OTF font into an SDF (Signed Distance Field) atlas.
//
// An SDF atlas stores each glyph not as raw pixels but as a distance field:
// every pixel records how far it is from the nearest edge of the glyph shape.
// This lets text scale to any size without blurring, and enables effects like
// outlines and drop shadows cheaply in a shader. Willow's text renderer
// consumes the two output files this tool produces.
//
// Pipeline:
//  1. Load the font and rasterize each character into a grayscale bitmap.
//  2. Convert each bitmap into a distance field (via willow.GenerateSDFFromBitmaps).
//  3. Pack all distance-field glyphs into a single texture atlas.
//  4. Write the atlas image (.png) and glyph metrics (.json).
//
// Usage:
//
//	fontgen -font input.ttf -size 80 -range 8 -out output [-chars "..."] [-width 1024]
//
// Flags:
//
//	-font    Path to TTF/OTF font file (required).
//	-size    Rasterization size in pixels (default 80). Larger = more detail.
//	-range   SDF spread in pixels (default 8). Controls how far the distance
//	         field extends beyond glyph edges — larger values allow thicker
//	         outlines/shadows but need more atlas space.
//	-out     Output file prefix (default "output"). Produces <prefix>.png and <prefix>.json.
//	-chars   Characters to include (default: printable ASCII 0x20–0x7E).
//	-width   Maximum atlas width in pixels (default 1024).
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"unicode/utf8"

	"github.com/devthicket/willow"
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

	// Step 1: Rasterize each character into a grayscale (alpha-only) bitmap.
	glyphs := rasterizeGlyphs(face, opts.Chars)

	// Step 2–3: Convert bitmaps to distance fields and pack into an atlas.
	atlas, metricsJSON, err := willow.GenerateSDFFromBitmaps(glyphs, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating SDF: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Write output files.
	if err := writeAtlasPNG(*outPrefix+".png", atlas); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	jsonPath := *outPrefix + ".json"
	if err := os.WriteFile(jsonPath, metricsJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", jsonPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s (%dx%d) and %s (%d glyphs)\n",
		*outPrefix+".png", atlas.Bounds().Dx(), atlas.Bounds().Dy(),
		jsonPath, len(glyphs))
}

// rasterizeGlyphs renders each character in the string using the font face and
// returns a slice of GlyphBitmaps containing alpha-only images. The font library
// renders into RGBA; we extract just the alpha channel since that's all the SDF
// generator needs (it only cares about shape coverage, not color).
func rasterizeGlyphs(face font.Face, chars string) []willow.GlyphBitmap {
	fontMetrics := face.Metrics()
	var glyphs []willow.GlyphBitmap

	for i := 0; i < len(chars); {
		char, charSize := utf8.DecodeRuneInString(chars[i:])
		i += charSize

		// GlyphBounds returns coordinates in 26.6 fixed-point (1/64 pixel units).
		bounds, advance, ok := face.GlyphBounds(char)
		if !ok {
			continue
		}

		// Convert fixed-point to integer pixel bounds.
		pixelLeft := bounds.Min.X.Floor()
		pixelTop := bounds.Min.Y.Floor()
		glyphWidth := bounds.Max.X.Ceil() - pixelLeft
		glyphHeight := bounds.Max.Y.Ceil() - pixelTop

		if glyphWidth <= 0 || glyphHeight <= 0 {
			// Whitespace characters (e.g. space) have no visible pixels
			// but still need an advance width for layout.
			glyphs = append(glyphs, willow.GlyphBitmap{
				ID:       char,
				Img:      nil,
				XAdvance: advance.Ceil(),
			})
			continue
		}

		alpha := rasterizeGlyph(face, char, bounds, glyphWidth, glyphHeight)

		glyphs = append(glyphs, willow.GlyphBitmap{
			ID:  char,
			Img: alpha,
			// Offset from the cursor position to where this glyph's bitmap starts.
			// YOffset is relative to the baseline, so we shift by the font's ascent
			// to convert from "top of em square" to "baseline" coordinates.
			XOffset:  pixelLeft,
			YOffset:  pixelTop + fontMetrics.Ascent.Ceil(),
			XAdvance: advance.Ceil(),
		})
	}

	return glyphs
}

// rasterizeGlyph renders a single character and returns its alpha channel.
// The font library only draws into RGBA images, so we render white-on-transparent
// and then copy out the alpha byte for each pixel.
func rasterizeGlyph(face font.Face, char rune, bounds fixed.Rectangle26_6, width, height int) *image.Alpha {
	// Render the glyph as white onto a transparent RGBA image.
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	drawer := font.Drawer{
		Dst:  rgba,
		Src:  image.White,
		Face: face,
		// Negate the bounds origin so the glyph draws at (0,0) in our image.
		Dot: fixed.Point26_6{X: -bounds.Min.X, Y: -bounds.Min.Y},
	}
	drawer.DrawString(string(char))

	// Extract the alpha channel — the SDF generator only needs coverage data.
	alpha := image.NewAlpha(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgbaOffset := rgba.PixOffset(x, y)
			alpha.Pix[y*alpha.Stride+x] = rgba.Pix[rgbaOffset+3] // A byte
		}
	}
	return alpha
}

// writeAtlasPNG encodes the atlas image to a PNG file.
func writeAtlasPNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("error encoding %s: %w", path, err)
	}
	return f.Close()
}
