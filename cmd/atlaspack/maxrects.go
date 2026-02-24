package main

import "fmt"

// PackInput describes a sprite to be placed in the atlas.
type PackInput struct {
	ID     int
	Width  int
	Height int
}

// PackResult describes where a sprite was placed.
type PackResult struct {
	ID      int
	X       int
	Y       int
	Width   int // placed width (may be swapped from input if rotated)
	Height  int // placed height
	Rotated bool
}

type freeRect struct {
	x, y, w, h int
}

// MaxRectsPacker implements the MaxRects bin packing algorithm with
// Best Short Side Fit (BSSF) heuristic.
type MaxRectsPacker struct {
	width, height int
	padding       int
	allowRotation bool
	freeRects     []freeRect
}

// NewMaxRectsPacker creates a packer for the given atlas dimensions.
func NewMaxRectsPacker(w, h, padding int, allowRotation bool) *MaxRectsPacker {
	return &MaxRectsPacker{
		width:         w,
		height:        h,
		padding:       padding,
		allowRotation: allowRotation,
		freeRects:     []freeRect{{0, 0, w, h}},
	}
}

// Pack places all inputs into the atlas. Inputs are sorted largest-area-first
// internally (the original slice is not modified). Returns an error if any
// sprite does not fit.
func (p *MaxRectsPacker) Pack(inputs []PackInput) ([]PackResult, error) {
	// Sort by area descending (copy to avoid mutating caller's slice).
	sorted := make([]PackInput, len(inputs))
	copy(sorted, inputs)
	sortByAreaDesc(sorted)

	results := make([]PackResult, 0, len(inputs))

	for _, in := range sorted {
		pw := in.Width + p.padding
		ph := in.Height + p.padding

		bestIdx := -1
		bestShort := int(^uint(0) >> 1) // max int
		bestLong := bestShort
		bestRotated := false
		bestX, bestY := 0, 0

		for i, fr := range p.freeRects {
			// Try unrotated.
			if pw <= fr.w && ph <= fr.h {
				shortSide := min(fr.w-pw, fr.h-ph)
				longSide := max(fr.w-pw, fr.h-ph)
				if shortSide < bestShort || (shortSide == bestShort && longSide < bestLong) {
					bestIdx = i
					bestShort = shortSide
					bestLong = longSide
					bestRotated = false
					bestX = fr.x
					bestY = fr.y
				}
			}
			// Try rotated (90° CW).
			if p.allowRotation && in.Width != in.Height {
				rw := in.Height + p.padding
				rh := in.Width + p.padding
				if rw <= fr.w && rh <= fr.h {
					shortSide := min(fr.w-rw, fr.h-rh)
					longSide := max(fr.w-rw, fr.h-rh)
					if shortSide < bestShort || (shortSide == bestShort && longSide < bestLong) {
						bestIdx = i
						bestShort = shortSide
						bestLong = longSide
						bestRotated = true
						bestX = fr.x
						bestY = fr.y
					}
				}
			}
		}

		_ = bestIdx
		if bestShort == int(^uint(0)>>1) {
			return nil, fmt.Errorf("sprite %q (%dx%d) does not fit in %dx%d atlas",
				fmt.Sprintf("id:%d", in.ID), in.Width, in.Height, p.width, p.height)
		}

		placedW := in.Width
		placedH := in.Height
		if bestRotated {
			placedW, placedH = in.Height, in.Width
		}

		results = append(results, PackResult{
			ID:      in.ID,
			X:       bestX,
			Y:       bestY,
			Width:   placedW,
			Height:  placedH,
			Rotated: bestRotated,
		})

		// Split free rects around the placed sprite (including padding).
		placed := freeRect{bestX, bestY, placedW + p.padding, placedH + p.padding}
		p.splitFreeRects(placed)
		p.pruneFreeRects()
	}

	return results, nil
}

// splitFreeRects splits all free rectangles that overlap with the placed rect.
func (p *MaxRectsPacker) splitFreeRects(placed freeRect) {
	var newRects []freeRect

	i := 0
	for i < len(p.freeRects) {
		fr := p.freeRects[i]
		if !overlaps(fr, placed) {
			i++
			continue
		}

		// Remove this free rect (swap-remove).
		p.freeRects[i] = p.freeRects[len(p.freeRects)-1]
		p.freeRects = p.freeRects[:len(p.freeRects)-1]

		// Generate up to 4 new free rects from the non-overlapping portions.
		if placed.x > fr.x {
			newRects = append(newRects, freeRect{
				fr.x, fr.y, placed.x - fr.x, fr.h,
			})
		}
		if placed.x+placed.w < fr.x+fr.w {
			newRects = append(newRects, freeRect{
				placed.x + placed.w, fr.y, (fr.x + fr.w) - (placed.x + placed.w), fr.h,
			})
		}
		if placed.y > fr.y {
			newRects = append(newRects, freeRect{
				fr.x, fr.y, fr.w, placed.y - fr.y,
			})
		}
		if placed.y+placed.h < fr.y+fr.h {
			newRects = append(newRects, freeRect{
				fr.x, placed.y + placed.h, fr.w, (fr.y + fr.h) - (placed.y + placed.h),
			})
		}
	}

	p.freeRects = append(p.freeRects, newRects...)
}

// pruneFreeRects removes any free rect that is fully contained by another.
func (p *MaxRectsPacker) pruneFreeRects() {
	n := len(p.freeRects)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if contains(p.freeRects[j], p.freeRects[i]) {
				// i is contained by j — remove i.
				p.freeRects[i] = p.freeRects[n-1]
				n--
				i--
				break
			}
		}
	}
	p.freeRects = p.freeRects[:n]
}

func overlaps(a, b freeRect) bool {
	return a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y
}

func contains(outer, inner freeRect) bool {
	return inner.x >= outer.x && inner.y >= outer.y &&
		inner.x+inner.w <= outer.x+outer.w &&
		inner.y+inner.h <= outer.y+outer.h
}

// sortByAreaDesc sorts pack inputs by area descending using insertion sort
// (simple, no allocations, stable for equal areas).
func sortByAreaDesc(s []PackInput) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		keyArea := key.Width * key.Height
		j := i - 1
		for j >= 0 && s[j].Width*s[j].Height < keyArea {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
