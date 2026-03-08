package atlas

import "testing"

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
