package main

import (
	"image"
	"image/color"
	"testing"
)

func solidNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestTrimAlpha_FullyOpaque(t *testing.T) {
	img := solidNRGBA(32, 32, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	r := TrimAlpha(img)

	if r.Trimmed {
		t.Error("fully opaque image should not be trimmed")
	}
	if r.Cropped.Bounds().Dx() != 32 || r.Cropped.Bounds().Dy() != 32 {
		t.Errorf("cropped size = %dx%d, want 32x32", r.Cropped.Bounds().Dx(), r.Cropped.Bounds().Dy())
	}
	if r.OffsetX != 0 || r.OffsetY != 0 {
		t.Errorf("offset = (%d, %d), want (0, 0)", r.OffsetX, r.OffsetY)
	}
	if r.OriginalW != 32 || r.OriginalH != 32 {
		t.Errorf("original = %dx%d, want 32x32", r.OriginalW, r.OriginalH)
	}
}

func TestTrimAlpha_TransparentBorder(t *testing.T) {
	// 10x10 image with opaque 6x4 region at (2, 3)
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 3; y < 7; y++ {
		for x := 2; x < 8; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	r := TrimAlpha(img)

	if !r.Trimmed {
		t.Error("should be trimmed")
	}
	if r.OffsetX != 2 || r.OffsetY != 3 {
		t.Errorf("offset = (%d, %d), want (2, 3)", r.OffsetX, r.OffsetY)
	}
	cw, ch := r.Cropped.Bounds().Dx(), r.Cropped.Bounds().Dy()
	if cw != 6 || ch != 4 {
		t.Errorf("cropped size = %dx%d, want 6x4", cw, ch)
	}
	if r.OriginalW != 10 || r.OriginalH != 10 {
		t.Errorf("original = %dx%d, want 10x10", r.OriginalW, r.OriginalH)
	}
	// Verify all pixels in cropped region are opaque.
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			if r.Cropped.NRGBAAt(x, y).A == 0 {
				t.Fatalf("cropped pixel (%d,%d) is transparent", x, y)
			}
		}
	}
}

func TestTrimAlpha_FullyTransparent(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	r := TrimAlpha(img)

	if !r.Trimmed {
		t.Error("fully transparent should report trimmed")
	}
	cw, ch := r.Cropped.Bounds().Dx(), r.Cropped.Bounds().Dy()
	if cw != 1 || ch != 1 {
		t.Errorf("cropped size = %dx%d, want 1x1", cw, ch)
	}
	if r.OriginalW != 16 || r.OriginalH != 16 {
		t.Errorf("original = %dx%d, want 16x16", r.OriginalW, r.OriginalH)
	}
}

func TestTrimAlpha_Asymmetric(t *testing.T) {
	// 20x10 image, single opaque pixel at (15, 2)
	img := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	img.SetNRGBA(15, 2, color.NRGBA{G: 255, A: 128})
	r := TrimAlpha(img)

	if !r.Trimmed {
		t.Error("should be trimmed")
	}
	if r.OffsetX != 15 || r.OffsetY != 2 {
		t.Errorf("offset = (%d, %d), want (15, 2)", r.OffsetX, r.OffsetY)
	}
	cw, ch := r.Cropped.Bounds().Dx(), r.Cropped.Bounds().Dy()
	if cw != 1 || ch != 1 {
		t.Errorf("cropped size = %dx%d, want 1x1", cw, ch)
	}
}

func TestTrimAlpha_1x1Opaque(t *testing.T) {
	img := solidNRGBA(1, 1, color.NRGBA{B: 255, A: 255})
	r := TrimAlpha(img)

	if r.Trimmed {
		t.Error("1x1 opaque should not be trimmed")
	}
	if r.Cropped.Bounds().Dx() != 1 || r.Cropped.Bounds().Dy() != 1 {
		t.Errorf("cropped size = %dx%d, want 1x1", r.Cropped.Bounds().Dx(), r.Cropped.Bounds().Dy())
	}
}
