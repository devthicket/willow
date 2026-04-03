package atlas

// Shelf-based bin packer for runtime atlas page allocation.
//
// This packer is used internally by Atlas.Add to arrange dynamically loaded
// images onto shared atlas pages. It divides each page into horizontal strips
// (shelves). Items are placed left-to-right on the best-fitting shelf, or a
// new shelf is opened if none fits. When no page has room, a new page is
// allocated via the global Manager.
//
// Algorithm: shelf packing with best-fit-height heuristic (O(shelves) per
// insertion). Same family used by Firefox, Mapbox, and freetype-gl. Typical
// packing efficiency is 85-90% for varied sprite sizes. Can be replaced with
// guillotine or skyline packing later without changing the public Atlas API.

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// MaxPackerPageIdx is the highest page index the packer may allocate.
// Page 0xFFFF is reserved as the magenta placeholder sentinel.
const MaxPackerPageIdx = int(MagentaPlaceholderPage) - 1

// shelf is a horizontal strip within an atlas page used by the shelf packer.
type shelf struct {
	y     int // top Y coordinate of this shelf within the page
	h     int // height of the shelf (set by the first item placed)
	usedW int // how far right items have been placed
}

// freeSlot is a rectangular hole left by a freed region on a shelf.
type freeSlot struct {
	x, y int // top-left corner
	w, h int // w = padded item width that was freed, h = shelf height
}

// packerPage wraps an atlas page image with shelf packing metadata.
type packerPage struct {
	img       *ebiten.Image
	pageIdx   int        // index in the Manager
	shelves   []shelf    // horizontal shelves allocated so far
	usedH     int        // total vertical space consumed (sum of shelf heights)
	freeSlots []freeSlot // holes left by freed regions
}

// tryPackFreeSlot scans free slots for the best fit (smallest area that fits
// pw × ph). If placed, the slot is consumed and any remaining horizontal
// space is split into a new free slot. Returns position and true on success.
func (pp *packerPage) tryPackFreeSlot(pw, ph int) (x, y int, ok bool) {
	bestIdx := -1
	bestArea := 0

	for i, fs := range pp.freeSlots {
		if pw > fs.w || ph > fs.h {
			continue
		}
		area := fs.w * fs.h
		if bestIdx < 0 || area < bestArea {
			bestArea = area
			bestIdx = i
		}
	}

	if bestIdx < 0 {
		return 0, 0, false
	}

	fs := pp.freeSlots[bestIdx]
	x, y = fs.x, fs.y

	// Split remaining horizontal space into a new free slot.
	remainder := fs.w - pw
	if remainder > 0 {
		pp.freeSlots[bestIdx] = freeSlot{
			x: fs.x + pw,
			y: fs.y,
			w: remainder,
			h: fs.h,
		}
	} else {
		// Exact fit — remove slot (swap-remove).
		last := len(pp.freeSlots) - 1
		pp.freeSlots[bestIdx] = pp.freeSlots[last]
		pp.freeSlots = pp.freeSlots[:last]
	}

	return x, y, true
}

// addFreeSlot records a freed region as a reusable slot. The shelf at the
// given Y coordinate determines the slot's full height.
func (pp *packerPage) addFreeSlot(x, y, pw int) {
	// Find the shelf at this Y coordinate to get its height.
	shelfH := 0
	for i := range pp.shelves {
		if pp.shelves[i].y == y {
			shelfH = pp.shelves[i].h
			break
		}
	}
	if shelfH == 0 {
		// Fallback: shouldn't happen, but avoid zero-height slots.
		return
	}
	pp.freeSlots = append(pp.freeSlots, freeSlot{
		x: x,
		y: y,
		w: pw,
		h: shelfH,
	})
}

// tryPack attempts to place a padded item of size (pw × ph) on this page.
// pageW and pageH are the page dimensions. Returns the top-left position
// and true on success, or (0, 0, false) if the item does not fit.
//
// Best-fit-height: picks the shelf whose height is closest to the item
// height, using remaining width as a tiebreaker. This minimises wasted
// vertical space within shelves when item heights vary.
func (pp *packerPage) tryPack(pw, ph, pageW, pageH int) (x, y int, ok bool) {
	bestIdx := -1
	bestHWaste := pageH + 1 // vertical waste (shelf.h - ph)
	bestRemain := pageW + 1 // horizontal remainder (tiebreaker)

	for i := range pp.shelves {
		s := &pp.shelves[i]
		if ph > s.h {
			continue // item taller than shelf
		}
		remain := pageW - s.usedW
		if pw > remain {
			continue // not enough horizontal room
		}
		hWaste := s.h - ph
		if hWaste < bestHWaste || (hWaste == bestHWaste && remain < bestRemain) {
			bestHWaste = hWaste
			bestRemain = remain
			bestIdx = i
		}
	}

	if bestIdx >= 0 {
		s := &pp.shelves[bestIdx]
		x = s.usedW
		y = s.y
		s.usedW += pw
		return x, y, true
	}

	// No existing shelf fits — try opening a new one.
	if pp.usedH+ph <= pageH {
		newShelf := shelf{
			y:     pp.usedH,
			h:     ph,
			usedW: pw,
		}
		pp.shelves = append(pp.shelves, newShelf)
		x = 0
		y = pp.usedH
		pp.usedH += ph
		return x, y, true
	}

	return 0, 0, false
}

// ShelfPacker manages shelf-based bin packing across multiple atlas pages.
type ShelfPacker struct {
	pages   []*packerPage
	pageW   int
	pageH   int
	Padding int
}

// NewShelfPacker creates a shelf packer with the given page dimensions and padding.
func NewShelfPacker(pageW, pageH, padding int) *ShelfPacker {
	return &ShelfPacker{
		pageW:   pageW,
		pageH:   pageH,
		Padding: padding,
	}
}

// Pack finds space for an image of size (w × h) on an existing page or
// allocates a new one. Returns the page, top-left position, and any error.
func (sp *ShelfPacker) Pack(w, h int) (*packerPage, int, int, error) {
	if w <= 0 || h <= 0 {
		return nil, 0, 0, fmt.Errorf("willow: cannot pack zero-size image (%d×%d)", w, h)
	}

	// Add padding to the item dimensions.
	pw := w + sp.Padding
	ph := h + sp.Padding

	if pw > sp.pageW || ph > sp.pageH {
		return nil, 0, 0, fmt.Errorf(
			"willow: image %d×%d (padded %d×%d) exceeds page size %d×%d",
			w, h, pw, ph, sp.pageW, sp.pageH,
		)
	}

	// Try free slots first (across all pages).
	for _, pp := range sp.pages {
		if x, y, ok := pp.tryPackFreeSlot(pw, ph); ok {
			return pp, x, y, nil
		}
	}

	// Try each existing page (normal shelf packing).
	for _, pp := range sp.pages {
		if x, y, ok := pp.tryPack(pw, ph, sp.pageW, sp.pageH); ok {
			return pp, x, y, nil
		}
	}

	// No room on any existing page — allocate a new one.
	pp, err := sp.allocPage()
	if err != nil {
		return nil, 0, 0, err
	}
	x, y, ok := pp.tryPack(pw, ph, sp.pageW, sp.pageH)
	if !ok {
		// Should not happen — we checked dimensions above.
		return nil, 0, 0, fmt.Errorf("willow: internal error: item does not fit on fresh page")
	}
	return pp, x, y, nil
}

// Free marks the region at (x, y) with padded width pw on the given page
// as a reusable free slot. The shelf height at that Y coordinate determines
// the slot's vertical extent.
func (sp *ShelfPacker) Free(pageIdx, x, y, pw int) {
	for _, pp := range sp.pages {
		if pp.pageIdx == pageIdx {
			pp.addFreeSlot(x, y, pw)
			return
		}
	}
}

// allocPage creates a new atlas page via the global Manager and
// registers it, then wraps it in a packerPage. Returns an error if the
// page index would exceed the uint16 range (0xFFFE).
func (sp *ShelfPacker) allocPage() (*packerPage, error) {
	am := GlobalManager()
	idx := am.AllocPage()
	if idx > MaxPackerPageIdx {
		return nil, fmt.Errorf("willow: atlas page index %d exceeds maximum %d", idx, MaxPackerPageIdx)
	}
	img := ebiten.NewImage(sp.pageW, sp.pageH)
	am.RegisterPage(idx, img)

	pp := &packerPage{
		img:     img,
		pageIdx: idx,
	}
	sp.pages = append(sp.pages, pp)
	return pp, nil
}
