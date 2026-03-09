package atlas

import "testing"

// --- RectPacker tests ---

func TestRectPacker_PlaceAndTryPlace(t *testing.T) {
	rp := NewRectPacker(100, 100)

	x, y, ok := rp.TryPlace(50, 50)
	if !ok {
		t.Fatal("TryPlace should succeed")
	}
	rp.Place(x, y, 50, 50)

	// Should still fit another 50×50.
	x2, y2, ok2 := rp.TryPlace(50, 50)
	if !ok2 {
		t.Fatal("second TryPlace should succeed")
	}
	rp.Place(x2, y2, 50, 50)

	// Two 50×50 placed, should not fit a 60×60.
	_, _, ok3 := rp.TryPlace(60, 60)
	if ok3 {
		t.Error("60×60 should not fit after two 50×50 placements")
	}
}

func TestRectPacker_DoesNotFit(t *testing.T) {
	rp := NewRectPacker(10, 10)
	_, _, ok := rp.TryPlace(11, 5)
	if ok {
		t.Error("11×5 should not fit in 10×10")
	}
}

func TestRectPacker_SingleItem(t *testing.T) {
	rp := NewRectPacker(256, 256)
	x, y, ok := rp.TryPlace(32, 32)
	if !ok {
		t.Fatal("tryPlace failed for single item")
	}
	if x != 0 || y != 0 {
		t.Errorf("position = (%d, %d), want (0, 0)", x, y)
	}
}

func TestRectPacker_MultipleNoOverlaps(t *testing.T) {
	rp := NewRectPacker(256, 256)

	type rect struct {
		x, y, w, h int
	}
	var rects []rect

	sizes := [][2]int{{32, 32}, {64, 48}, {16, 80}, {50, 50}, {100, 30}}
	for _, sz := range sizes {
		x, y, ok := rp.TryPlace(sz[0], sz[1])
		if !ok {
			t.Fatalf("TryPlace(%d, %d) failed", sz[0], sz[1])
		}
		rp.Place(x, y, sz[0], sz[1])
		rects = append(rects, rect{x, y, sz[0], sz[1]})
	}

	// Verify no overlaps.
	for i := 0; i < len(rects); i++ {
		for j := i + 1; j < len(rects); j++ {
			a := rects[i]
			b := rects[j]
			if a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y {
				t.Errorf("overlap: rect %d (%d,%d %dx%d) and rect %d (%d,%d %dx%d)",
					i, a.x, a.y, a.w, a.h, j, b.x, b.y, b.w, b.h)
			}
		}
	}
}

func TestRectPacker_OversizedItem(t *testing.T) {
	rp := NewRectPacker(64, 64)
	_, _, ok := rp.TryPlace(128, 32)
	if ok {
		t.Error("expected TryPlace to fail for oversized item")
	}
}

func TestRectPacker_FreeRectSplitting(t *testing.T) {
	rp := NewRectPacker(100, 100)

	rp.Place(0, 0, 50, 50)

	x, y, ok := rp.TryPlace(50, 50)
	if !ok {
		t.Fatal("TryPlace failed after split")
	}
	rp.Place(x, y, 50, 50)

	x2, y2, ok := rp.TryPlace(50, 50)
	if !ok {
		t.Fatal("TryPlace failed for third item")
	}
	rp.Place(x2, y2, 50, 50)
}

func TestRectPacker_PruneContained(t *testing.T) {
	rp := NewRectPacker(100, 100)

	rp.Place(0, 0, 30, 30)
	rp.Place(30, 0, 30, 30)

	x, y, ok := rp.TryPlace(40, 40)
	if !ok {
		t.Fatal("TryPlace failed after pruning")
	}
	rp.Place(x, y, 40, 40)
}

func TestRectPacker_WithPadding(t *testing.T) {
	rp := NewRectPacker(100, 100)

	x1, y1, ok := rp.TryPlace(50, 50)
	if !ok {
		t.Fatal("TryPlace 1 failed")
	}
	rp.Place(x1, y1, 50, 50)

	x2, y2, ok := rp.TryPlace(50, 50)
	if !ok {
		t.Fatal("TryPlace 2 failed")
	}
	rp.Place(x2, y2, 50, 50)

	if x1 == x2 && y1 == y2 {
		t.Error("two items placed at same position")
	}
}

// --- ShelfPacker tests ---

func TestShelfPacker_SingleItem(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)
	pp, x, y, err := sp.Pack(32, 32)
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

func TestShelfPacker_TwoItemsSameShelf(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	pp1, x1, y1, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	pp2, x2, y2, err := sp.Pack(32, 32)
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

func TestShelfPacker_NewShelfWhenHorizontalFull(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(64, 128, 0)

	_, x1, y1, err := sp.Pack(64, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	if x1 != 0 || y1 != 0 {
		t.Errorf("item1 = (%d, %d), want (0, 0)", x1, y1)
	}

	_, x2, y2, err := sp.Pack(32, 16)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 0 || y2 != 32 {
		t.Errorf("item2 = (%d, %d), want (0, 32)", x2, y2)
	}
}

func TestShelfPacker_BestFitShelf(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(100, 200, 0)

	sp.Pack(40, 20)
	sp.Pack(60, 30)

	_, x, y, err := sp.Pack(30, 20)
	if err != nil {
		t.Fatalf("pack best-fit: %v", err)
	}
	if x != 40 || y != 0 {
		t.Errorf("best-fit item = (%d, %d), want (40, 0)", x, y)
	}
}

func TestShelfPacker_BestFitShelf_HeightTiebreaker(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(100, 200, 0)

	sp.Pack(40, 20) // shelf 0: remaining 60
	sp.Pack(20, 20) // goes on shelf 0 at x=40, remaining now 40
	sp.Pack(70, 30) // shelf 1: height 30, remaining 30
	sp.Pack(80, 20) // new shelf 2: remaining 20

	_, x, y, err := sp.Pack(15, 20)
	if err != nil {
		t.Fatalf("pack tiebreaker: %v", err)
	}
	// Shelf 2 starts at y=20+30=50, usedW=80.
	if x != 80 || y != 50 {
		t.Errorf("tiebreaker item = (%d, %d), want (80, 50)", x, y)
	}
}

func TestShelfPacker_PageOverflow(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(32, 32, 0)

	pp1, _, _, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}

	pp2, x2, y2, err := sp.Pack(16, 16)
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

func TestShelfPacker_OversizedError(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(64, 64, 0)
	_, _, _, err := sp.Pack(128, 32)
	if err == nil {
		t.Error("expected error for oversized image")
	}

	_, _, _, err = sp.Pack(32, 128)
	if err == nil {
		t.Error("expected error for oversized image (height)")
	}
}

func TestShelfPacker_ZeroSizeError(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	_, _, _, err := sp.Pack(0, 32)
	if err == nil {
		t.Error("expected error for zero width")
	}
	_, _, _, err = sp.Pack(32, 0)
	if err == nil {
		t.Error("expected error for zero height")
	}
	_, _, _, err = sp.Pack(-1, 32)
	if err == nil {
		t.Error("expected error for negative width")
	}
}

func TestShelfPacker_PaddingSpacing(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 2)

	_, x1, y1, err := sp.Pack(10, 10)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	if x1 != 0 || y1 != 0 {
		t.Errorf("item1 = (%d, %d), want (0, 0)", x1, y1)
	}

	_, x2, y2, err := sp.Pack(10, 10)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 12 || y2 != 0 {
		t.Errorf("item2 = (%d, %d), want (12, 0)", x2, y2)
	}

	sp2 := NewShelfPacker(20, 256, 2)
	sp2.Pack(10, 10)
	_, x3, y3, err := sp2.Pack(10, 10)
	if err != nil {
		t.Fatalf("pack 3: %v", err)
	}
	if x3 != 0 || y3 != 12 {
		t.Errorf("item3 = (%d, %d), want (0, 12)", x3, y3)
	}
}

func TestShelfPacker_ManyItemsNoOverlaps(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(512, 512, 1)

	type rect struct {
		page, x, y, w, h int
	}
	var rects []rect

	for i := 0; i < 120; i++ {
		w := 10 + (i % 20)
		h := 10 + ((i * 3) % 15)
		pp, x, y, err := sp.Pack(w, h)
		if err != nil {
			t.Fatalf("pack item %d (%dx%d): %v", i, w, h, err)
		}
		rects = append(rects, rect{pp.pageIdx, x, y, w, h})
	}

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

func TestShelfPacker_PaddingOversized(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(64, 64, 2)
	_, _, _, err := sp.Pack(63, 63)
	if err == nil {
		t.Error("expected error: 63+2=65 exceeds page size 64")
	}
}

func TestShelfPacker_FreeSlotReuse(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	pp, x1, y1, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}
	_, x2, _, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}

	sp.Free(pp.pageIdx, x1, y1, 32)

	_, x3, y3, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 3: %v", err)
	}
	if x3 != x1 || y3 != y1 {
		t.Errorf("reused slot = (%d, %d), want (%d, %d)", x3, y3, x1, y1)
	}
	_ = x2
}

func TestShelfPacker_FreeSlotSplitting(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	pp, x1, y1, err := sp.Pack(64, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}

	sp.Free(pp.pageIdx, x1, y1, 64)

	_, x2, y2, err := sp.Pack(20, 32)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 0 || y2 != 0 {
		t.Errorf("item2 = (%d, %d), want (0, 0)", x2, y2)
	}

	_, x3, y3, err := sp.Pack(30, 32)
	if err != nil {
		t.Fatalf("pack 3: %v", err)
	}
	if x3 != 20 || y3 != 0 {
		t.Errorf("item3 = (%d, %d), want (20, 0)", x3, y3)
	}
}

func TestShelfPacker_FreeSmallerItem(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	pp, x1, y1, err := sp.Pack(64, 32)
	if err != nil {
		t.Fatalf("pack 1: %v", err)
	}

	sp.Free(pp.pageIdx, x1, y1, 64)

	_, x2, y2, err := sp.Pack(16, 16)
	if err != nil {
		t.Fatalf("pack 2: %v", err)
	}
	if x2 != 0 || y2 != 0 {
		t.Errorf("smaller item = (%d, %d), want (0, 0)", x2, y2)
	}
}

func TestShelfPacker_FreeNonExistent(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)
	// Freeing on a non-existent page should not panic.
	sp.Free(999, 0, 0, 32)
}

func TestShelfPacker_MultipleFreeAndReuse(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	sp := NewShelfPacker(256, 256, 0)

	type pos struct{ x, y int }
	var positions []pos
	var pp *packerPage
	for i := 0; i < 4; i++ {
		p, x, y, err := sp.Pack(32, 32)
		if err != nil {
			t.Fatalf("pack %d: %v", i, err)
		}
		pp = p
		positions = append(positions, pos{x, y})
	}

	sp.Free(pp.pageIdx, positions[0].x, positions[0].y, 32)
	sp.Free(pp.pageIdx, positions[2].x, positions[2].y, 32)

	_, x5, y5, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 5: %v", err)
	}
	_, x6, y6, err := sp.Pack(32, 32)
	if err != nil {
		t.Fatalf("pack 6: %v", err)
	}

	freed := map[pos]bool{
		positions[0]: true,
		positions[2]: true,
	}
	p5 := pos{x5, y5}
	p6 := pos{x6, y6}
	if !freed[p5] {
		t.Errorf("item 5 at (%d,%d) not in freed set", x5, y5)
	}
	if !freed[p6] {
		t.Errorf("item 6 at (%d,%d) not in freed set", x6, y6)
	}
}

func TestShelfPacker_PageIndexOverflow(t *testing.T) {
	ResetGlobalManager()
	defer ResetGlobalManager()

	am := GlobalManager()
	for am.NextPage() <= MaxPackerPageIdx {
		am.AllocPage()
	}

	sp := NewShelfPacker(32, 32, 0)
	_, _, _, err := sp.Pack(16, 16)
	if err == nil {
		t.Error("expected error when page index exceeds uint16 limit")
	}
}

// --- SortStagedByArea ---

func TestSortStagedByArea(t *testing.T) {
	s := []StagedEntry{
		{Name: "small", W: 2, H: 2},
		{Name: "large", W: 10, H: 10},
		{Name: "medium", W: 5, H: 5},
	}
	SortStagedByArea(s)
	if s[0].Name != "large" || s[1].Name != "medium" || s[2].Name != "small" {
		t.Errorf("sort order = [%s, %s, %s], want [large, medium, small]",
			s[0].Name, s[1].Name, s[2].Name)
	}
}

// --- rectFreeRect helpers ---

func TestRectOverlaps(t *testing.T) {
	a := rectFreeRect{0, 0, 10, 10}
	b := rectFreeRect{5, 5, 10, 10}
	c := rectFreeRect{20, 20, 5, 5}

	if !rectOverlaps(a, b) {
		t.Error("a and b should overlap")
	}
	if rectOverlaps(a, c) {
		t.Error("a and c should not overlap")
	}
}

func TestRectContains(t *testing.T) {
	outer := rectFreeRect{0, 0, 100, 100}
	inner := rectFreeRect{10, 10, 20, 20}
	notInner := rectFreeRect{90, 90, 20, 20}

	if !rectContains(outer, inner) {
		t.Error("outer should contain inner")
	}
	if rectContains(outer, notInner) {
		t.Error("outer should not contain notInner")
	}
}
