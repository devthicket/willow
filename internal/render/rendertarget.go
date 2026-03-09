package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// RenderTexturePool manages reusable offscreen ebiten.Images keyed by
// power-of-two dimensions. After warmup, Acquire/Release are zero-alloc.
type RenderTexturePool struct {
	buckets map[uint64][]*ebiten.Image
}

// poolKey packs power-of-two width and height into a single uint64.
func poolKey(w, h int) uint64 {
	return uint64(w)<<32 | uint64(h)
}

// Acquire returns a cleared offscreen image with at least (w, h) pixels.
// Dimensions are rounded up to the next power of two.
func (p *RenderTexturePool) Acquire(w, h int) *ebiten.Image {
	pw := NextPowerOfTwo(w)
	ph := NextPowerOfTwo(h)
	key := poolKey(pw, ph)

	if p.buckets != nil {
		if stack := p.buckets[key]; len(stack) > 0 {
			img := stack[len(stack)-1]
			p.buckets[key] = stack[:len(stack)-1]
			img.Clear()
			return img
		}
	}

	return ebiten.NewImageWithOptions(
		image.Rect(0, 0, pw, ph),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
}

// Release returns an image to the pool for reuse.
func (p *RenderTexturePool) Release(img *ebiten.Image) {
	if img == nil {
		return
	}
	b := img.Bounds()
	key := poolKey(b.Dx(), b.Dy())

	if p.buckets == nil {
		p.buckets = make(map[uint64][]*ebiten.Image)
	}
	p.buckets[key] = append(p.buckets[key], img)
}

// NextPowerOfTwo returns the smallest power of two >= n (minimum 1).
func NextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << int(math.Ceil(math.Log2(float64(n))))
}
