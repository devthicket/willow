package main

import (
	"image"
	"image/color"
	"testing"
)

func TestValidateGrid(t *testing.T) {
	tests := []struct {
		name                   string
		imgW, imgH             int
		tileW, tileH           int
		spacing, margin        int
		wantCols, wantRows     int
		wantErr                bool
	}{
		{"4x4 with 2x2 tiles", 4, 4, 2, 2, 0, 0, 2, 2, false},
		{"16x16 with 16x16 tile", 16, 16, 16, 16, 0, 0, 1, 1, false},
		{"single tile", 8, 8, 8, 8, 0, 0, 1, 1, false},
		{"with spacing", 10, 10, 4, 4, 2, 0, 2, 2, false},
		{"with margin", 6, 6, 2, 2, 0, 1, 2, 2, false},
		{"with spacing and margin", 12, 12, 4, 4, 2, 1, 2, 2, false},
		{"remainder width", 5, 4, 2, 2, 0, 0, 0, 0, true},
		{"remainder height", 4, 5, 2, 2, 0, 0, 0, 0, true},
		{"margin too large", 2, 2, 2, 2, 0, 2, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows, err := validateGrid(tt.imgW, tt.imgH, tt.tileW, tt.tileH, tt.spacing, tt.margin)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got cols=%d rows=%d", cols, rows)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cols != tt.wantCols || rows != tt.wantRows {
				t.Fatalf("got cols=%d rows=%d, want cols=%d rows=%d", cols, rows, tt.wantCols, tt.wantRows)
			}
		})
	}
}

func px(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

func setPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	off := img.PixOffset(x, y)
	img.Pix[off] = c.R
	img.Pix[off+1] = c.G
	img.Pix[off+2] = c.B
	img.Pix[off+3] = c.A
}

func getPixel(img *image.NRGBA, x, y int) color.NRGBA {
	off := img.PixOffset(x, y)
	return color.NRGBA{
		R: img.Pix[off],
		G: img.Pix[off+1],
		B: img.Pix[off+2],
		A: img.Pix[off+3],
	}
}

func TestExtrudeTile(t *testing.T) {
	// 2x2 source tile:
	//   A B     (A=red, B=green, C=blue, D=white)
	//   C D
	A := px(255, 0, 0, 255)
	B := px(0, 255, 0, 255)
	C := px(0, 0, 255, 255)
	D := px(255, 255, 255, 255)

	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	setPixel(src, 0, 0, A)
	setPixel(src, 1, 0, B)
	setPixel(src, 0, 1, C)
	setPixel(src, 1, 1, D)

	// Destination: 4x4 (2+2 extrusion). Tile placed at (1,1).
	dst := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	extrudeTile(dst, src, 1, 1, 0, 0, 2, 2)

	// Expected 4x4:
	//   A A B B
	//   A A B B
	//   C C D D
	//   C C D D
	want := [4][4]color.NRGBA{
		{A, A, B, B},
		{A, A, B, B},
		{C, C, D, D},
		{C, C, D, D},
	}

	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			got := getPixel(dst, x, y)
			if got != want[y][x] {
				t.Errorf("pixel (%d,%d): got %v, want %v", x, y, got, want[y][x])
			}
		}
	}
}

func TestExtrudeTileset(t *testing.T) {
	// 2x2 grid of 2x2 tiles = 4x4 input, no spacing, no margin.
	// Tiles:
	//   T0(red)    T1(green)
	//   T2(blue)   T3(white)
	red := px(255, 0, 0, 255)
	green := px(0, 255, 0, 255)
	blue := px(0, 0, 255, 255)
	white := px(255, 255, 255, 255)

	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	// Fill each 2x2 tile with a solid color.
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			setPixel(src, x, y, red)
			setPixel(src, x+2, y, green)
			setPixel(src, x, y+2, blue)
			setPixel(src, x+2, y+2, white)
		}
	}

	dst := extrudeTileset(src, 2, 2, 2, 2, 0, 0)

	// Output: newMargin=1, newSpacing=2, tileW=2, tileH=2, cols=2, rows=2
	// outW = 2*1 + 2*2 + 1*2 = 8
	// outH = 2*1 + 2*2 + 1*2 = 8
	wantW, wantH := 8, 8
	if dst.Bounds().Dx() != wantW || dst.Bounds().Dy() != wantH {
		t.Fatalf("output size: got %dx%d, want %dx%d", dst.Bounds().Dx(), dst.Bounds().Dy(), wantW, wantH)
	}

	// Tile 0 (red) placed at (1,1), its extruded area is (0,0)-(3,3).
	// Spot-check: center should be red.
	if got := getPixel(dst, 1, 1); got != red {
		t.Errorf("tile0 center (1,1): got %v, want %v", got, red)
	}
	// Extrusion corner at (0,0) should also be red.
	if got := getPixel(dst, 0, 0); got != red {
		t.Errorf("tile0 corner (0,0): got %v, want %v", got, red)
	}

	// Tile 1 (green) placed at (5,1).
	if got := getPixel(dst, 5, 1); got != green {
		t.Errorf("tile1 center (5,1): got %v, want %v", got, green)
	}

	// Tile 2 (blue) placed at (1,5).
	if got := getPixel(dst, 1, 5); got != blue {
		t.Errorf("tile2 center (1,5): got %v, want %v", got, blue)
	}

	// Tile 3 (white) placed at (5,5).
	if got := getPixel(dst, 5, 5); got != white {
		t.Errorf("tile3 center (5,5): got %v, want %v", got, white)
	}
}
