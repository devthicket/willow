package willow

import "testing"

func TestRectPacker_SingleItem(t *testing.T) {
	rp := newRectPacker(256, 256)
	x, y, ok := rp.tryPlace(32, 32)
	if !ok {
		t.Fatal("tryPlace failed for single item")
	}
	if x != 0 || y != 0 {
		t.Errorf("position = (%d, %d), want (0, 0)", x, y)
	}
}

func TestRectPacker_MultipleNoOverlaps(t *testing.T) {
	rp := newRectPacker(256, 256)

	type rect struct {
		x, y, w, h int
	}
	var rects []rect

	sizes := [][2]int{{32, 32}, {64, 48}, {16, 80}, {50, 50}, {100, 30}}
	for _, sz := range sizes {
		x, y, ok := rp.tryPlace(sz[0], sz[1])
		if !ok {
			t.Fatalf("tryPlace(%d, %d) failed", sz[0], sz[1])
		}
		rp.place(x, y, sz[0], sz[1])
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

func TestRectPacker_BestFitHeuristic(t *testing.T) {
	rp := newRectPacker(100, 100)

	// Place a 60x60 item in the top-left.
	rp.place(0, 0, 60, 60)
	// Now free rects should include:
	// - right strip: (60, 0, 40, 100)
	// - bottom strip: (0, 60, 100, 40)

	// Place a 39x39 item. BSSF should prefer the right strip (40x100) over
	// the bottom strip (100x40) because the short side leftover is smaller.
	x, y, ok := rp.tryPlace(39, 39)
	if !ok {
		t.Fatal("tryPlace failed")
	}
	// Should be at (60, 0) in the right strip — short side = min(1, 61) = 1
	// Bottom strip would give short side = min(61, 1) = 1 too, but long side
	// would be 61 vs 61, so either is acceptable.
	if x != 60 && x != 0 {
		t.Errorf("unexpected x = %d", x)
	}
	_ = y
}

func TestRectPacker_OversizedItem(t *testing.T) {
	rp := newRectPacker(64, 64)
	_, _, ok := rp.tryPlace(128, 32)
	if ok {
		t.Error("expected tryPlace to fail for oversized item")
	}
}

func TestRectPacker_FreeRectSplitting(t *testing.T) {
	rp := newRectPacker(100, 100)

	// Place at (0,0) size 50x50.
	rp.place(0, 0, 50, 50)

	// Should be able to place items in remaining space.
	x, y, ok := rp.tryPlace(50, 50)
	if !ok {
		t.Fatal("tryPlace failed after split")
	}
	rp.place(x, y, 50, 50)

	// There should still be room for more.
	x2, y2, ok := rp.tryPlace(50, 50)
	if !ok {
		t.Fatal("tryPlace failed for third item")
	}
	rp.place(x2, y2, 50, 50)
}

func TestRectPacker_PruneContained(t *testing.T) {
	rp := newRectPacker(100, 100)

	// Place an item that will create overlapping free rects needing pruning.
	rp.place(0, 0, 30, 30)
	rp.place(30, 0, 30, 30)

	// Verify the packer still works correctly after pruning.
	x, y, ok := rp.tryPlace(40, 40)
	if !ok {
		t.Fatal("tryPlace failed after pruning")
	}
	rp.place(x, y, 40, 40)
}

func TestRectPacker_WithPadding(t *testing.T) {
	// Caller handles padding by adding it to pw/ph before tryPlace/place.
	rp := newRectPacker(100, 100)

	x1, y1, ok := rp.tryPlace(50, 50) // 48 + 2 padding
	if !ok {
		t.Fatal("tryPlace 1 failed")
	}
	rp.place(x1, y1, 50, 50)

	x2, y2, ok := rp.tryPlace(50, 50)
	if !ok {
		t.Fatal("tryPlace 2 failed")
	}
	rp.place(x2, y2, 50, 50)

	// The two items should not overlap.
	if x1 == x2 && y1 == y2 {
		t.Error("two items placed at same position")
	}
}

func TestSortStagedByArea(t *testing.T) {
	entries := []stagedEntry{
		{name: "small", w: 10, h: 10},
		{name: "large", w: 100, h: 100},
		{name: "medium", w: 50, h: 50},
	}
	sortStagedByArea(entries)

	if entries[0].name != "large" {
		t.Errorf("first = %q, want large", entries[0].name)
	}
	if entries[1].name != "medium" {
		t.Errorf("second = %q, want medium", entries[1].name)
	}
	if entries[2].name != "small" {
		t.Errorf("third = %q, want small", entries[2].name)
	}
}

func TestRectFreeRect_Overlaps(t *testing.T) {
	a := rectFreeRect{0, 0, 10, 10}
	b := rectFreeRect{5, 5, 10, 10}
	c := rectFreeRect{20, 20, 10, 10}

	if !rectOverlaps(a, b) {
		t.Error("a and b should overlap")
	}
	if rectOverlaps(a, c) {
		t.Error("a and c should not overlap")
	}
}

func TestRectFreeRect_Contains(t *testing.T) {
	outer := rectFreeRect{0, 0, 100, 100}
	inner := rectFreeRect{10, 10, 20, 20}
	outside := rectFreeRect{90, 90, 20, 20}

	if !rectContains(outer, inner) {
		t.Error("outer should contain inner")
	}
	if rectContains(outer, outside) {
		t.Error("outer should not contain outside")
	}
}
