package filter

import (
	"testing"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// oldOutlineApply reproduces the original 9-DrawImage stamp approach.
func oldOutlineApply(thickness int, c types.Color, src, dst *ebiten.Image) {
	t := float64(thickness)
	offsets := [8][2]float64{
		{-t, 0}, {t, 0}, {0, -t}, {0, t},
		{-t, -t}, {t, -t}, {-t, t}, {t, t},
	}
	var op ebiten.DrawImageOptions
	r, g, b, a := c.R(), c.G(), c.B(), c.A()
	for _, off := range offsets {
		op.GeoM.Reset()
		op.ColorScale.Reset()
		op.GeoM.Translate(off[0], off[1])
		op.ColorScale.Scale(float32(r*a), float32(g*a), float32(b*a), float32(a))
		dst.DrawImage(src, &op)
	}
	op.GeoM.Reset()
	op.ColorScale.Reset()
	dst.DrawImage(src, &op)
}

// newOutlineApply runs the new outline filter the way the pipeline does,
// handling MultiPass for separable expansion.
func newOutlineApply(f *OutlineFilter, src *ebiten.Image, a, b *ebiten.Image) {
	passes := f.Passes()
	if passes <= 0 {
		return
	}
	if passes == 1 {
		f.SetPass(0)
		a.Clear()
		f.Apply(src, a)
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

func benchOldOutline(b *testing.B, size, thickness int) {
	c := types.RGBA(1, 0, 0, 1)
	src := ebiten.NewImage(size, size)
	dst := ebiten.NewImage(size+thickness*2, size+thickness*2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst.Clear()
		oldOutlineApply(thickness, c, src, dst)
	}
}

func benchNewOutline(b *testing.B, size, thickness int) {
	f := NewOutlineFilter(thickness, types.RGBA(1, 0, 0, 1))
	src := ebiten.NewImage(size, size)
	a := ebiten.NewImage(size+thickness*2, size+thickness*2)
	bb := ebiten.NewImage(size+thickness*2, size+thickness*2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newOutlineApply(f, src, a, bb)
	}
}

// 64×64, thickness 2 (circular path)
func BenchmarkOutlineOld_64_T2(b *testing.B)  { benchOldOutline(b, 64, 2) }
func BenchmarkOutlineNew_64_T2(b *testing.B)  { benchNewOutline(b, 64, 2) }

// 64×64, thickness 4 (circular path)
func BenchmarkOutlineOld_64_T4(b *testing.B)  { benchOldOutline(b, 64, 4) }
func BenchmarkOutlineNew_64_T4(b *testing.B)  { benchNewOutline(b, 64, 4) }

// 128×128, thickness 8 (separable path)
func BenchmarkOutlineOld_128_T8(b *testing.B) { benchOldOutline(b, 128, 8) }
func BenchmarkOutlineNew_128_T8(b *testing.B) { benchNewOutline(b, 128, 8) }

// 128×128, thickness 16 (separable path)
func BenchmarkOutlineOld_128_T16(b *testing.B) { benchOldOutline(b, 128, 16) }
func BenchmarkOutlineNew_128_T16(b *testing.B) { benchNewOutline(b, 128, 16) }
