// blurcompare renders the old (downscale/upscale) and new (Kawase shader)
// blur side-by-side at several radii and writes a comparison PNG.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	srcSize = 128
	margin  = 16
)

var radii = []int{2, 4, 8, 16, 32}

// --- old blur (verbatim copy of the replaced implementation) ---

func oldBlurApply(radius int, src, dst *ebiten.Image) {
	passes := int(math.Ceil(math.Log2(float64(radius))))
	if passes < 1 {
		passes = 1
	}
	srcBounds := src.Bounds()
	w, h := srcBounds.Dx(), srcBounds.Dy()
	temps := make([]*ebiten.Image, passes)
	var op ebiten.DrawImageOptions

	current := src
	for i := 0; i < passes; i++ {
		w = max(w/2, 1)
		h = max(h/2, 1)
		temps[i] = ebiten.NewImage(w, h)
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Scale(float64(w)/float64(current.Bounds().Dx()), float64(h)/float64(current.Bounds().Dy()))
		op.Filter = ebiten.FilterLinear
		temps[i].DrawImage(current, &op)
		current = temps[i]
	}
	for i := passes - 2; i >= 0; i-- {
		temps[i].Clear()
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Scale(float64(temps[i].Bounds().Dx())/float64(current.Bounds().Dx()),
			float64(temps[i].Bounds().Dy())/float64(current.Bounds().Dy()))
		op.Filter = ebiten.FilterLinear
		temps[i].DrawImage(current, &op)
		current = temps[i]
	}
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.GeoM.Scale(float64(dst.Bounds().Dx())/float64(current.Bounds().Dx()),
		float64(dst.Bounds().Dy())/float64(current.Bounds().Dy()))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(current, &op)
	for _, t := range temps {
		t.Deallocate()
	}
}

// --- new blur (Kawase shader, multi-pass) ---

// We can't import internal/filter, so we compile the shader here.
const kawaseShaderSrc = `//kage:unit pixels
package main

var Offset float

func bilinear(pos vec2) vec4 {
	p := pos - 0.5
	f := fract(p)
	i := floor(p) + 0.5
	return mix(
		mix(imageSrc0At(i), imageSrc0At(i+vec2(1, 0)), f.x),
		mix(imageSrc0At(i+vec2(0, 1)), imageSrc0At(i+vec2(1, 1)), f.x),
		f.y,
	)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	s := bilinear(src + vec2(-Offset, -Offset))
	s += bilinear(src + vec2(Offset, -Offset))
	s += bilinear(src + vec2(-Offset, Offset))
	s += bilinear(src + vec2(Offset, Offset))
	return s / 4.0
}
`

var kawaseShader *ebiten.Shader

func ensureShader() *ebiten.Shader {
	if kawaseShader == nil {
		s, err := ebiten.NewShader([]byte(kawaseShaderSrc))
		if err != nil {
			panic(err)
		}
		kawaseShader = s
	}
	return kawaseShader
}

func kawasePasses(radius int) int {
	if radius <= 0 {
		return 0
	}
	p := int(math.Ceil(math.Sqrt(2.0 * float64(radius))))
	if p < 1 {
		return 1
	}
	return p
}

func newBlurApply(radius int, src, dst *ebiten.Image) {
	shader := ensureShader()
	passes := kawasePasses(radius)
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	a := ebiten.NewImage(w, h)
	b := ebiten.NewImage(w, h)
	defer a.Deallocate()
	defer b.Deallocate()

	offsetBuf := [1]float32{}
	offsetSlice := offsetBuf[:]
	uniforms := map[string]any{"Offset": offsetSlice}
	var shaderOp ebiten.DrawRectShaderOptions

	current, scratch := src, a
	for i := 0; i < passes; i++ {
		offsetBuf[0] = float32(i) + 0.5
		scratch.Clear()
		shaderOp.Images[0] = current
		shaderOp.Uniforms = uniforms
		scratch.DrawRectShader(w, h, shader, &shaderOp)
		current, scratch = scratch, current
	}

	// Copy final result to dst.
	var op ebiten.DrawImageOptions
	dst.DrawImage(current, &op)
}

// --- test pattern ---

func makeTestPattern() *ebiten.Image {
	img := ebiten.NewImage(srcSize, srcSize)
	// Colored rectangles + circles to show blur quality.
	pix := image.NewRGBA(image.Rect(0, 0, srcSize, srcSize))
	// Background
	for y := 0; y < srcSize; y++ {
		for x := 0; x < srcSize; x++ {
			pix.SetRGBA(x, y, color.RGBA{30, 30, 50, 255})
		}
	}
	// Red block
	for y := 20; y < 50; y++ {
		for x := 20; x < 50; x++ {
			pix.SetRGBA(x, y, color.RGBA{220, 40, 40, 255})
		}
	}
	// Green block
	for y := 60; y < 100; y++ {
		for x := 10; x < 60; x++ {
			pix.SetRGBA(x, y, color.RGBA{40, 200, 40, 255})
		}
	}
	// Blue circle
	cx, cy, r := 90.0, 50.0, 25.0
	for y := 0; y < srcSize; y++ {
		for x := 0; x < srcSize; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy < r*r {
				pix.SetRGBA(x, y, color.RGBA{40, 80, 240, 255})
			}
		}
	}
	// White star-burst lines from center
	for a := 0; a < 360; a += 30 {
		rad := float64(a) * math.Pi / 180
		for t := 0.0; t < 40; t += 0.5 {
			px := int(64 + t*math.Cos(rad))
			py := int(64 + t*math.Sin(rad))
			if px >= 0 && px < srcSize && py >= 0 && py < srcSize {
				pix.SetRGBA(px, py, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	img.WritePixels(pix.Pix)
	return img
}

// --- ebitengine game (renders one frame then exits) ---

type game struct {
	done bool
}

func (g *game) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.done {
		return
	}

	src := makeTestPattern()
	cols := len(radii)
	tileW := srcSize + margin
	tileH := srcSize + margin
	totalW := tileW*cols + margin
	totalH := tileH*3 + margin // row 0: original, row 1: old, row 2: new

	canvas := ebiten.NewImage(totalW, totalH)
	// Dark background
	canvas.Fill(color.RGBA{20, 20, 20, 255})

	var op ebiten.DrawImageOptions

	// Row 0: draw all originals first (before any blur touches the atlas).
	for col := range radii {
		x := float64(margin + col*tileW)
		op.GeoM.Reset()
		op.GeoM.Translate(x, float64(margin))
		canvas.DrawImage(src, &op)
	}

	// Row 1 & 2: blur passes (each gets a fresh copy of src to avoid
	// atlas interference from the managed-image batching).
	for col, radius := range radii {
		x := float64(margin + col*tileW)

		// Fresh copy for old blur
		srcCopy := ebiten.NewImage(srcSize, srcSize)
		srcCopy.DrawImage(src, nil)

		oldDst := ebiten.NewImage(srcSize, srcSize)
		oldBlurApply(radius, srcCopy, oldDst)
		op.GeoM.Reset()
		op.GeoM.Translate(x, float64(margin+tileH))
		canvas.DrawImage(oldDst, &op)

		// Fresh copy for new blur
		srcCopy2 := ebiten.NewImage(srcSize, srcSize)
		srcCopy2.DrawImage(src, nil)

		newDst := ebiten.NewImage(srcSize, srcSize)
		newBlurApply(radius, srcCopy2, newDst)
		op.GeoM.Reset()
		op.GeoM.Translate(x, float64(margin+tileH*2))
		canvas.DrawImage(newDst, &op)

		srcCopy.Deallocate()
		srcCopy2.Deallocate()
		oldDst.Deallocate()
		newDst.Deallocate()
	}

	// Read pixels from canvas and write PNG.
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))

	// Read pixels row by row via SubImage to avoid huge single read.
	for y := 0; y < h; y++ {
		sub := canvas.SubImage(image.Rect(0, y, w, y+1)).(*ebiten.Image)
		rowPix := make([]byte, w*4)
		sub.ReadPixels(rowPix)
		copy(rgba.Pix[y*rgba.Stride:], rowPix)
	}

	outPath := "blur_compare.png"
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", outPath, err)
	} else {
		_ = png.Encode(f, rgba)
		f.Close()
		fmt.Printf("wrote %s (%dx%d)\n", outPath, w, h)
		fmt.Println("rows: original | old (downscale/upscale) | new (kawase shader)")
		fmt.Printf("columns: radii %v\n", radii)
	}

	canvas.Deallocate()
	g.done = true
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 240
}

func main() {
	ebiten.SetWindowSize(320, 240)
	ebiten.SetWindowTitle("blur compare")
	if err := ebiten.RunGame(&game{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
