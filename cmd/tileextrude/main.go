// Command tileextrude takes a tileset PNG, extrudes each tile's edge pixels
// outward by 1px to prevent texture bleeding, and writes a new tileset.
//
// Usage:
//
//	tileextrude -w 16 -h 16 tileset.png
//
// The output tileset preserves the original tile size. The extrusion lives in
// the spacing and margin, so map editors and loaders reference the same tile
// dimensions. Updated grid parameters are printed to stdout.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
)

func main() {
	tileW := flag.Int("w", 0, "tile width in pixels (required)")
	tileH := flag.Int("h", 0, "tile height in pixels (required)")
	out := flag.String("out", "", "output file path (default: <input>-extruded.png)")
	spacing := flag.Int("spacing", 0, "existing spacing between tiles in the input")
	margin := flag.Int("margin", 0, "existing margin around the input tileset edge")
	flag.Parse()

	if *tileW <= 0 || *tileH <= 0 {
		fmt.Fprintln(os.Stderr, "error: -w and -h are required and must be positive")
		flag.Usage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: input tileset path required")
		flag.Usage()
		os.Exit(1)
	}
	inPath := args[0]

	outPath := *out
	if outPath == "" {
		outPath = strings.TrimSuffix(inPath, ".png") + "-extruded.png"
	}

	src, err := loadPNG(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %s: %v\n", inPath, err)
		os.Exit(1)
	}

	cols, rows, err := validateGrid(src.Bounds().Dx(), src.Bounds().Dy(), *tileW, *tileH, *spacing, *margin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	dst := extrudeTileset(src, cols, rows, *tileW, *tileH, *spacing, *margin)

	if err := writePNG(outPath, dst); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("tile size: %dx%d (unchanged)\n", *tileW, *tileH)
	fmt.Printf("spacing:   2\n")
	fmt.Printf("margin:    1\n")
	fmt.Printf("image:     %s (%dx%d, was %dx%d)\n", outPath, dst.Bounds().Dx(), dst.Bounds().Dy(), src.Bounds().Dx(), src.Bounds().Dy())
}

func validateGrid(imgW, imgH, tileW, tileH, spacing, margin int) (int, int, error) {
	usableW := imgW - 2*margin
	usableH := imgH - 2*margin

	if usableW <= 0 || usableH <= 0 {
		return 0, 0, fmt.Errorf("margin %d is too large for %dx%d image", margin, imgW, imgH)
	}

	cols := (usableW + spacing) / (tileW + spacing)
	rows := (usableH + spacing) / (tileH + spacing)

	if cols <= 0 || rows <= 0 {
		return 0, 0, fmt.Errorf("no tiles fit: %dx%d tiles in %dx%d usable area", tileW, tileH, usableW, usableH)
	}

	expectedW := cols*tileW + (cols-1)*spacing
	expectedH := rows*tileH + (rows-1)*spacing

	if expectedW != usableW {
		return 0, 0, fmt.Errorf("image width %d does not fit tile grid: %dpx tiles, %dpx spacing, %dpx margin leaves %dpx remainder",
			imgW, tileW, spacing, margin, usableW-expectedW)
	}
	if expectedH != usableH {
		return 0, 0, fmt.Errorf("image height %d does not fit tile grid: %dpx tiles, %dpx spacing, %dpx margin leaves %dpx remainder",
			imgH, tileH, spacing, margin, usableH-expectedH)
	}

	return cols, rows, nil
}

func extrudeTileset(src *image.NRGBA, cols, rows, tileW, tileH, spacing, margin int) *image.NRGBA {
	// Output layout: 1px margin, 2px spacing (extrusion only, input spacing/margin discarded).
	const outMargin = 1
	const outSpacing = 2

	outW := 2*outMargin + cols*tileW + (cols-1)*outSpacing
	outH := 2*outMargin + rows*tileH + (rows-1)*outSpacing

	dst := image.NewNRGBA(image.Rect(0, 0, outW, outH))

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			srcX := margin + col*(tileW+spacing)
			srcY := margin + row*(tileH+spacing)
			dstX := outMargin + col*(tileW+outSpacing)
			dstY := outMargin + row*(tileH+outSpacing)

			extrudeTile(dst, src, dstX, dstY, srcX, srcY, tileW, tileH)
		}
	}

	return dst
}

// copyPx copies a single 4-byte NRGBA pixel within an image.
func copyPx(img *image.NRGBA, dstX, dstY, srcX, srcY int) {
	s := img.PixOffset(srcX, srcY)
	d := img.PixOffset(dstX, dstY)
	copy(img.Pix[d:d+4], img.Pix[s:s+4])
}

func extrudeTile(dst, src *image.NRGBA, dstX, dstY, srcX, srcY, tileW, tileH int) {
	// Copy tile pixels into dst.
	for y := 0; y < tileH; y++ {
		s := src.PixOffset(srcX, srcY+y)
		d := dst.PixOffset(dstX, dstY+y)
		copy(dst.Pix[d:d+tileW*4], src.Pix[s:s+tileW*4])
	}

	// Extrude edges: duplicate each border row/column outward by 1px.
	for x := 0; x < tileW; x++ {
		copyPx(dst, dstX+x, dstY-1, dstX+x, dstY)           // top
		copyPx(dst, dstX+x, dstY+tileH, dstX+x, dstY+tileH-1) // bottom
	}
	for y := 0; y < tileH; y++ {
		copyPx(dst, dstX-1, dstY+y, dstX, dstY+y)            // left
		copyPx(dst, dstX+tileW, dstY+y, dstX+tileW-1, dstY+y) // right
	}

	// Fill corners from nearest corner pixel.
	copyPx(dst, dstX-1, dstY-1, dstX, dstY)                         // top-left
	copyPx(dst, dstX+tileW, dstY-1, dstX+tileW-1, dstY)             // top-right
	copyPx(dst, dstX-1, dstY+tileH, dstX, dstY+tileH-1)             // bottom-left
	copyPx(dst, dstX+tileW, dstY+tileH, dstX+tileW-1, dstY+tileH-1) // bottom-right
}

func loadPNG(path string) (*image.NRGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba, nil
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, img.At(x, y))
		}
	}
	return nrgba, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return f.Close()
}
