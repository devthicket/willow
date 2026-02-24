package main

import "image"

// TrimResult holds the result of alpha-trimming a sprite image.
type TrimResult struct {
	Cropped   *image.NRGBA // the cropped sub-image (1x1 transparent if fully transparent)
	OffsetX   int          // horizontal offset of cropped region within original
	OffsetY   int          // vertical offset of cropped region within original
	OriginalW int          // original image width before trimming
	OriginalH int          // original image height before trimming
	Trimmed   bool         // true if any transparent pixels were removed
}

// TrimAlpha scans for the tightest bounding box of non-transparent pixels
// (alpha > 0) and returns the cropped image plus metadata. Uses direct Pix
// slice access for speed. Fully transparent images return a 1x1 image.
func TrimAlpha(img *image.NRGBA) TrimResult {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	minX, minY := w, h
	maxX, maxY := -1, -1

	for y := 0; y < h; y++ {
		rowStart := (bounds.Min.Y+y-img.Rect.Min.Y)*img.Stride + (bounds.Min.X-img.Rect.Min.X)*4
		for x := 0; x < w; x++ {
			a := img.Pix[rowStart+x*4+3]
			if a > 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// Fully transparent — return 1x1 transparent image.
	if maxX < 0 {
		cropped := image.NewNRGBA(image.Rect(0, 0, 1, 1))
		return TrimResult{
			Cropped:   cropped,
			OffsetX:   0,
			OffsetY:   0,
			OriginalW: w,
			OriginalH: h,
			Trimmed:   w != 1 || h != 1,
		}
	}

	cropW := maxX - minX + 1
	cropH := maxY - minY + 1
	trimmed := minX != 0 || minY != 0 || cropW != w || cropH != h

	// Copy the cropped region into a new tight image.
	cropped := image.NewNRGBA(image.Rect(0, 0, cropW, cropH))
	for row := 0; row < cropH; row++ {
		srcOff := img.PixOffset(bounds.Min.X+minX, bounds.Min.Y+minY+row)
		dstOff := cropped.PixOffset(0, row)
		copy(cropped.Pix[dstOff:dstOff+cropW*4], img.Pix[srcOff:srcOff+cropW*4])
	}

	return TrimResult{
		Cropped:   cropped,
		OffsetX:   minX,
		OffsetY:   minY,
		OriginalW: w,
		OriginalH: h,
		Trimmed:   trimmed,
	}
}
