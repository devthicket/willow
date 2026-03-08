package filter

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// BlurFilter applies a Kawase iterative blur using downscale/upscale passes.
// No Kage shader needed — bilinear filtering during DrawImage does the work.
type BlurFilter struct {
	Radius int
	temps  []*ebiten.Image
	imgOp  ebiten.DrawImageOptions
}

// NewBlurFilter creates a blur filter with the given radius (in pixels).
func NewBlurFilter(radius int) *BlurFilter {
	if radius < 0 {
		radius = 0
	}
	return &BlurFilter{Radius: radius}
}

// Apply renders a Kawase blur from src into dst using iterative downscale/upscale.
func (f *BlurFilter) Apply(src, dst *ebiten.Image) {
	if f.Radius <= 0 {
		f.imgOp.GeoM.Reset()
		f.imgOp.ColorScale.Reset()
		f.imgOp.Filter = ebiten.FilterNearest
		dst.DrawImage(src, &f.imgOp)
		return
	}

	passes := int(math.Ceil(math.Log2(float64(f.Radius))))
	if passes < 1 {
		passes = 1
	}

	srcBounds := src.Bounds()
	w, h := srcBounds.Dx(), srcBounds.Dy()

	needed := passes
	for len(f.temps) < needed {
		f.temps = append(f.temps, nil)
	}
	for i := needed; i < len(f.temps); i++ {
		if f.temps[i] != nil {
			f.temps[i].Deallocate()
			f.temps[i] = nil
		}
	}
	f.temps = f.temps[:needed]

	op := &f.imgOp

	// Downscale passes: each half-size
	current := src
	for i := 0; i < passes; i++ {
		w = max(w/2, 1)
		h = max(h/2, 1)
		if f.temps[i] == nil || f.temps[i].Bounds().Dx() != w || f.temps[i].Bounds().Dy() != h {
			if f.temps[i] != nil {
				f.temps[i].Deallocate()
			}
			f.temps[i] = ebiten.NewImage(w, h)
		} else {
			f.temps[i].Clear()
		}
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		op.GeoM.Scale(float64(w)/sw, float64(h)/sh)
		op.Filter = ebiten.FilterLinear
		f.temps[i].DrawImage(current, op)
		current = f.temps[i]
	}

	// Upscale passes: draw each back up
	for i := passes - 2; i >= 0; i-- {
		f.temps[i].Clear()
		op.GeoM.Reset()
		op.ColorScale.Reset()
		sw := float64(current.Bounds().Dx())
		sh := float64(current.Bounds().Dy())
		tw := float64(f.temps[i].Bounds().Dx())
		th := float64(f.temps[i].Bounds().Dy())
		op.GeoM.Scale(tw/sw, th/sh)
		op.Filter = ebiten.FilterLinear
		f.temps[i].DrawImage(current, op)
		current = f.temps[i]
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
	dst.DrawImage(current, op)
}

// Padding returns the blur radius; the offscreen buffer is expanded to avoid clipping.
func (f *BlurFilter) Padding() int { return f.Radius }
