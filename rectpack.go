package willow

import "github.com/hajimehoshi/ebiten/v2"

// MaxRects bin packer for batch atlas mode.
//
// Adapted from cmd/atlaspack/maxrects.go (package main, can't import).
// Uses Best Short Side Fit (BSSF) heuristic with no rotation. Provides
// single-item placement so the caller can do greedy multi-page packing:
// try current page, if it doesn't fit, start a new page.

// rectFreeRect is a free rectangle within a MaxRects packer page.
type rectFreeRect struct {
	x, y, w, h int
}

// rectPacker implements MaxRects bin packing for a single page.
type rectPacker struct {
	freeRects []rectFreeRect
}

// newRectPacker creates a MaxRects packer for one page of size w × h.
func newRectPacker(w, h int) *rectPacker {
	return &rectPacker{
		freeRects: []rectFreeRect{
			{0, 0, w, h},
		},
	}
}

// tryPlace finds the best position for an item of padded size (pw × ph)
// using the BSSF heuristic. Returns position and true, or (0, 0, false)
// if the item does not fit.
func (rp *rectPacker) tryPlace(pw, ph int) (x, y int, ok bool) {
	bestShort := int(^uint(0) >> 1) // max int
	bestLong := bestShort
	bestX, bestY := 0, 0

	for _, fr := range rp.freeRects {
		if pw <= fr.w && ph <= fr.h {
			shortSide := min(fr.w-pw, fr.h-ph)
			longSide := max(fr.w-pw, fr.h-ph)
			if shortSide < bestShort || (shortSide == bestShort && longSide < bestLong) {
				bestShort = shortSide
				bestLong = longSide
				bestX = fr.x
				bestY = fr.y
			}
		}
	}

	if bestShort == int(^uint(0)>>1) {
		return 0, 0, false
	}
	return bestX, bestY, true
}

// place commits a placement at (x, y) with padded size (pw × ph).
// Splits overlapping free rects and prunes contained ones.
func (rp *rectPacker) place(x, y, pw, ph int) {
	placed := rectFreeRect{x, y, pw, ph}
	rp.splitFreeRects(placed)
	rp.pruneFreeRects()
}

// splitFreeRects splits all free rects that overlap with the placed rect.
func (rp *rectPacker) splitFreeRects(placed rectFreeRect) {
	var newRects []rectFreeRect

	i := 0
	for i < len(rp.freeRects) {
		fr := rp.freeRects[i]
		if !rectOverlaps(fr, placed) {
			i++
			continue
		}

		// Remove (swap-remove).
		rp.freeRects[i] = rp.freeRects[len(rp.freeRects)-1]
		rp.freeRects = rp.freeRects[:len(rp.freeRects)-1]

		// Generate up to 4 new free rects from non-overlapping portions.
		if placed.x > fr.x {
			newRects = append(newRects, rectFreeRect{
				fr.x, fr.y, placed.x - fr.x, fr.h,
			})
		}
		if placed.x+placed.w < fr.x+fr.w {
			newRects = append(newRects, rectFreeRect{
				placed.x + placed.w, fr.y, (fr.x + fr.w) - (placed.x + placed.w), fr.h,
			})
		}
		if placed.y > fr.y {
			newRects = append(newRects, rectFreeRect{
				fr.x, fr.y, fr.w, placed.y - fr.y,
			})
		}
		if placed.y+placed.h < fr.y+fr.h {
			newRects = append(newRects, rectFreeRect{
				fr.x, placed.y + placed.h, fr.w, (fr.y + fr.h) - (placed.y + placed.h),
			})
		}
	}

	rp.freeRects = append(rp.freeRects, newRects...)
}

// pruneFreeRects removes any free rect fully contained by another.
func (rp *rectPacker) pruneFreeRects() {
	n := len(rp.freeRects)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if rectContains(rp.freeRects[j], rp.freeRects[i]) {
				// i is contained by j — remove i.
				rp.freeRects[i] = rp.freeRects[n-1]
				n--
				i--
				break
			}
		}
	}
	rp.freeRects = rp.freeRects[:n]
}

func rectOverlaps(a, b rectFreeRect) bool {
	return a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y
}

func rectContains(outer, inner rectFreeRect) bool {
	return inner.x >= outer.x && inner.y >= outer.y &&
		inner.x+inner.w <= outer.x+outer.w &&
		inner.y+inner.h <= outer.y+outer.h
}

// stagedEntry holds a buffered image for batch atlas packing.
type stagedEntry struct {
	name string
	img  *ebiten.Image
	w, h int
}

// sortStagedByArea sorts staged entries by area descending using insertion sort.
func sortStagedByArea(s []stagedEntry) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		keyArea := key.w * key.h
		j := i - 1
		for j >= 0 && s[j].w*s[j].h < keyArea {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
