package willow

import "testing"

func TestPacker_SingleItem(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(256, 256, 0)
	pp, x, y, err := sp.pack(32, 32)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if x != 0 || y != 0 {
		t.Errorf("position = (%d, %d), want (0, 0)", x, y)
	}
	if pp == nil {
		t.Fatal("packerPage is nil")
	}
	if pp.pageIdx != 0 {
		t.Errorf("pageIdx = %d, want 0", pp.pageIdx)
	}
}

func TestPacker_TwoItemsSameShelf(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(256, 256, 0)

	pp1, x1, y1, err := sp.pack(32, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	pp2, x2, y2, err := sp.pack(32, 32)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}

	if x1 != 0 || y1 != 0 {
		t.Errorf("item1 = (%d, %d), want (0, 0)", x1, y1)
	}
	if x2 != 32 || y2 != 0 {
		t.Errorf("item2 = (%d, %d), want (32, 0)", x2, y2)
	}
	if pp1.pageIdx != pp2.pageIdx {
		t.Error("items should be on the same page")
	}
}

func TestPacker_NewShelfWhenHorizontalFull(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Page is 64 wide. First item fills horizontal space, second must open a new shelf.
	sp := newShelfPacker(64, 128, 0)

	_, x1, y1, err := sp.pack(64, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	if x1 != 0 || y1 != 0 {
		t.Errorf("item1 = (%d, %d), want (0, 0)", x1, y1)
	}

	_, x2, y2, err := sp.pack(32, 16)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 0 || y2 != 32 {
		t.Errorf("item2 = (%d, %d), want (0, 32)", x2, y2)
	}
}

func TestPacker_BestFitShelf(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Best-fit-height: prefer the shelf whose height is closest to the item.
	sp := newShelfPacker(100, 200, 0)

	// Shelf 0: height 20, place a 40-wide item (remaining 60).
	sp.pack(40, 20)
	// Shelf 1: height 30, place a 60-wide item (remaining 40).
	sp.pack(60, 30)

	// Pack a 20-high item. Both shelves have room.
	// Shelf 0 hWaste=0 (20-20), shelf 1 hWaste=10 (30-20).
	// Best-fit-height picks shelf 0 (less vertical waste).
	_, x, y, err := sp.pack(30, 20)
	if err != nil {
		t.Fatalf("pack best-fit: %v", err)
	}
	// Shelf 0 at y=0, usedW was 40, so x=40.
	if x != 40 || y != 0 {
		t.Errorf("best-fit item = (%d, %d), want (40, 0)", x, y)
	}
}

func TestPacker_BestFitShelf_HeightTiebreaker(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// When two shelves have the same height waste, prefer smaller remaining width.
	sp := newShelfPacker(100, 200, 0)

	// Both shelves are height 20.
	sp.pack(40, 20) // shelf 0: remaining 60
	sp.pack(20, 20) // goes on shelf 0 at x=40, remaining now 40

	// Force a new shelf by packing something taller.
	sp.pack(70, 30) // shelf 1: height 30, remaining 30

	// Open shelf 2 with height 20 (same as shelf 0).
	sp.pack(80, 20) // doesn't fit on shelf 0 (remaining 40 < 80), new shelf 2: remaining 20

	// Now pack 15-wide, 20-high. Fits on shelf 0 (hWaste=0, remain=40) and
	// shelf 2 (hWaste=0, remain=20). Same hWaste → tiebreak on width → shelf 2.
	_, x, y, err := sp.pack(15, 20)
	if err != nil {
		t.Fatalf("pack tiebreaker: %v", err)
	}
	// Shelf 2 starts at y=20+30=50, usedW=80.
	if x != 80 || y != 50 {
		t.Errorf("tiebreaker item = (%d, %d), want (80, 50)", x, y)
	}
}

func TestPacker_PageOverflow(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Tiny page: only fits one 32×32 item.
	sp := newShelfPacker(32, 32, 0)

	pp1, _, _, err := sp.pack(32, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}

	pp2, x2, y2, err := sp.pack(16, 16)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}

	if pp1.pageIdx == pp2.pageIdx {
		t.Error("second item should be on a different page")
	}
	if x2 != 0 || y2 != 0 {
		t.Errorf("second page item = (%d, %d), want (0, 0)", x2, y2)
	}
}

func TestPacker_OversizedError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(64, 64, 0)
	_, _, _, err := sp.pack(128, 32)
	if err == nil {
		t.Error("expected error for oversized image")
	}

	_, _, _, err = sp.pack(32, 128)
	if err == nil {
		t.Error("expected error for oversized image (height)")
	}
}

func TestPacker_ZeroSizeError(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(256, 256, 0)

	_, _, _, err := sp.pack(0, 32)
	if err == nil {
		t.Error("expected error for zero width")
	}
	_, _, _, err = sp.pack(32, 0)
	if err == nil {
		t.Error("expected error for zero height")
	}
	_, _, _, err = sp.pack(-1, 32)
	if err == nil {
		t.Error("expected error for negative width")
	}
}

func TestPacker_PaddingSpacing(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(256, 256, 2)

	_, x1, y1, err := sp.pack(10, 10)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	if x1 != 0 || y1 != 0 {
		t.Errorf("item1 = (%d, %d), want (0, 0)", x1, y1)
	}

	// Second item on same shelf should start at 10+2=12 (item width + padding).
	_, x2, y2, err := sp.pack(10, 10)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 12 || y2 != 0 {
		t.Errorf("item2 = (%d, %d), want (12, 0)", x2, y2)
	}

	// Third item that forces a new shelf should start at y=12 (shelf height 10+2).
	// Page width 20: first item 10+2=12 leaves 8 remaining, next 10+2=12 > 8 → new shelf.
	sp2 := newShelfPacker(20, 256, 2)
	sp2.pack(10, 10) // shelf 0: usedW=12, y=0, h=12
	_, x3, y3, err := sp2.pack(10, 10)
	if err != nil {
		t.Fatalf("pack 3: %v", err)
	}
	if x3 != 0 || y3 != 12 {
		t.Errorf("item3 = (%d, %d), want (0, 12)", x3, y3)
	}
}

func TestPacker_ManyItemsNoOverlaps(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	sp := newShelfPacker(512, 512, 1)

	type rect struct {
		page, x, y, w, h int
	}
	var rects []rect

	for i := 0; i < 120; i++ {
		w := 10 + (i % 20)
		h := 10 + ((i * 3) % 15)
		pp, x, y, err := sp.pack(w, h)
		if err != nil {
			t.Fatalf("pack item %d (%dx%d): %v", i, w, h, err)
		}
		rects = append(rects, rect{pp.pageIdx, x, y, w, h})
	}

	// Verify no overlaps within the same page.
	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			a := rects[i]
			b := rects[j]
			if a.page != b.page {
				continue
			}
			if a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y {
				t.Errorf("overlap: item %d (%d,%d %dx%d) and item %d (%d,%d %dx%d) on page %d",
					i, a.x, a.y, a.w, a.h, j, b.x, b.y, b.w, b.h, a.page)
			}
		}
	}
}

func TestPacker_PaddingOversized(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Image fits without padding but not with it.
	sp := newShelfPacker(64, 64, 2)
	_, _, _, err := sp.pack(63, 63)
	if err == nil {
		t.Error("expected error: 63+2=65 exceeds page size 64")
	}
}

func TestPacker_PageIndexOverflow(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	// Advance the AtlasManager's next page past the uint16 limit.
	am := atlasManager()
	for am.NextPage() <= maxPackerPageIdx {
		am.AllocPage()
	}

	sp := newShelfPacker(32, 32, 0)
	_, _, _, err := sp.pack(16, 16)
	if err == nil {
		t.Error("expected error when page index exceeds uint16 limit")
	}
}
