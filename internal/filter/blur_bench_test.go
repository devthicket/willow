package filter

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// oldBlurApply reproduces the original downscale/upscale blur that was replaced.
// It creates N temp images at halving dimensions and ping-pongs through them.
// Kept here verbatim so the benchmark comparison is apples-to-apples.
func oldBlurApply(radius int, src, dst *ebiten.Image) {
	if radius <= 0 {
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		dst.DrawImage(src, &op)
		return
	}

	passes := int(math.Ceil(math.Log2(float64(radius))))
	if passes < 1 {
		passes = 1
	}

	srcBounds := src.Bounds()
	w, h := srcBounds.Dx(), srcBounds.Dy()

	temps := make([]*ebiten.Image, passes)
	var op ebiten.DrawImageOptions

	// Downscale passes: each half-size
	current := src
	for i := 0; i < passes; i++ {
		w = max(w/2, 1)
		h = max(h/2, 1)
		temps[i] = ebiten.NewImage(w, h)
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		op.GeoM.Scale(float64(w)/sw, float64(h)/sh)
		op.Filter = ebiten.FilterLinear
		temps[i].DrawImage(current, &op)
		current = temps[i]
	}

	// Upscale passes: draw each back up
	for i := passes - 2; i >= 0; i-- {
		temps[i].Clear()
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		tw := float64(temps[i].Bounds().Dx())
		th := float64(temps[i].Bounds().Dy())
		op.GeoM.Scale(tw/sw, th/sh)
		op.Filter = ebiten.FilterLinear
		temps[i].DrawImage(current, &op)
		current = temps[i]
	}

	// Final upscale to dst.
	op.GeoM.Reset()
	op.ColorScale.Reset()
	sw := float64(current.Bounds().Dx())
	sh := float64(current.Bounds().Dy())
	tw := float64(dst.Bounds().Dx())
	th := float64(dst.Bounds().Dy())
	op.GeoM.Scale(tw/sw, th/sh)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(current, &op)

	for _, t := range temps {
		t.Deallocate()
	}
}

// newBlurApply runs the new Kawase shader blur the same way the pipeline does:
// external ping-pong with SetPass + Apply per iteration.
func newBlurApply(f *BlurFilter, src *ebiten.Image, a, b *ebiten.Image) {
	passes := f.Passes()
	if passes <= 0 {
		return
	}
	current, scratch := src, a
	for i := 0; i < passes; i++ {
		f.SetPass(i)
		scratch.Clear()
		f.Apply(current, scratch)
		current, scratch = scratch, current
	}
	_ = current
	_ = b
}

// --- Benchmarks ---
// Use b.N with short benchtime via CLI flag (-benchtime=500ms recommended).

func benchOldBlur(b *testing.B, size, radius int) {
	src := ebiten.NewImage(size, size)
	dst := ebiten.NewImage(size, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst.Clear()
		oldBlurApply(radius, src, dst)
	}
}

func benchNewBlur(b *testing.B, size, radius int) {
	f := NewBlurFilter(radius)
	src := ebiten.NewImage(size, size)
	a := ebiten.NewImage(size, size)
	bb := ebiten.NewImage(size, size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newBlurApply(f, src, a, bb)
	}
}

// 128×128, radius 8
func BenchmarkBlurOld_128_R8(b *testing.B)  { benchOldBlur(b, 128, 8) }
func BenchmarkBlurNew_128_R8(b *testing.B)  { benchNewBlur(b, 128, 8) }

// 256×256, radius 8
func BenchmarkBlurOld_256_R8(b *testing.B)  { benchOldBlur(b, 256, 8) }
func BenchmarkBlurNew_256_R8(b *testing.B)  { benchNewBlur(b, 256, 8) }

// 256×256, radius 16
func BenchmarkBlurOld_256_R16(b *testing.B) { benchOldBlur(b, 256, 16) }
func BenchmarkBlurNew_256_R16(b *testing.B) { benchNewBlur(b, 256, 16) }
