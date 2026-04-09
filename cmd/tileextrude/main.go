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
	spacing := flag.Int("spacing", 0, "existing spacing between tiles")
	margin := flag.Int("margin", 0, "margin around the tileset edge")
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

	newSpacing := *spacing + 2
	newMargin := *margin + 1
	fmt.Printf("tile size: %dx%d (unchanged)\n", *tileW, *tileH)
	fmt.Printf("spacing:   %d (was %d)\n", newSpacing, *spacing)
	fmt.Printf("margin:    %d (was %d)\n", newMargin, *margin)
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
	newMargin := margin + 1
	newSpacing := spacing + 2

	outW := 2*newMargin + cols*tileW + (cols-1)*newSpacing
	outH := 2*newMargin + rows*tileH + (rows-1)*newSpacing

	dst := image.NewNRGBA(image.Rect(0, 0, outW, outH))

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			srcX := margin + col*(tileW+spacing)
			srcY := margin + row*(tileH+spacing)
			dstX := newMargin + col*(tileW+newSpacing)
			dstY := newMargin + row*(tileH+newSpacing)

			extrudeTile(dst, src, dstX, dstY, srcX, srcY, tileW, tileH)
		}
	}

	return dst
}

func extrudeTile(dst, src *image.NRGBA, dstX, dstY, srcX, srcY, tileW, tileH int) {
	// Phase 1: Copy tile center.
	for y := 0; y < tileH; y++ {
		sOff := src.PixOffset(srcX, srcY+y)
		dOff := dst.PixOffset(dstX, dstY+y)
		copy(dst.Pix[dOff:dOff+tileW*4], src.Pix[sOff:sOff+tileW*4])
	}

	// Phase 2: Extrude edges.
	// Top edge: copy row dstY to dstY-1.
	sOff := dst.PixOffset(dstX, dstY)
	dOff := dst.PixOffset(dstX, dstY-1)
	copy(dst.Pix[dOff:dOff+tileW*4], dst.Pix[sOff:sOff+tileW*4])

	// Bottom edge: copy row dstY+tileH-1 to dstY+tileH.
	sOff = dst.PixOffset(dstX, dstY+tileH-1)
	dOff = dst.PixOffset(dstX, dstY+tileH)
	copy(dst.Pix[dOff:dOff+tileW*4], dst.Pix[sOff:sOff+tileW*4])

	// Left edge: copy column dstX to dstX-1.
	for y := 0; y < tileH; y++ {
		sOff = dst.PixOffset(dstX, dstY+y)
		dOff = dst.PixOffset(dstX-1, dstY+y)
		copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])
	}

	// Right edge: copy column dstX+tileW-1 to dstX+tileW.
	for y := 0; y < tileH; y++ {
		sOff = dst.PixOffset(dstX+tileW-1, dstY+y)
		dOff = dst.PixOffset(dstX+tileW, dstY+y)
		copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])
	}

	// Phase 3: Fill corners.
	// Top-left.
	sOff = dst.PixOffset(dstX, dstY)
	dOff = dst.PixOffset(dstX-1, dstY-1)
	copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])

	// Top-right.
	sOff = dst.PixOffset(dstX+tileW-1, dstY)
	dOff = dst.PixOffset(dstX+tileW, dstY-1)
	copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])

	// Bottom-left.
	sOff = dst.PixOffset(dstX, dstY+tileH-1)
	dOff = dst.PixOffset(dstX-1, dstY+tileH)
	copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])

	// Bottom-right.
	sOff = dst.PixOffset(dstX+tileW-1, dstY+tileH-1)
	dOff = dst.PixOffset(dstX+tileW, dstY+tileH)
	copy(dst.Pix[dOff:dOff+4], dst.Pix[sOff:sOff+4])
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
