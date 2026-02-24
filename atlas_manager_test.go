package willow

import "testing"

func TestAtlasManagerSingleton(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am1 := atlasManager()
	am2 := atlasManager()
	if am1 != am2 {
		t.Error("atlasManager() should return the same instance")
	}
}

func TestAtlasManagerRegisterAndPage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am := atlasManager()

	// Page out of range returns nil.
	if am.Page(0) != nil {
		t.Error("expected nil for unregistered page 0")
	}
	if am.Page(-1) != nil {
		t.Error("expected nil for negative index")
	}

	// Register a nil page (valid in tests without GPU).
	am.RegisterPage(2, nil)
	if am.PageCount() != 3 {
		t.Errorf("PageCount = %d, want 3", am.PageCount())
	}

	// NextPage should be past registered pages.
	if am.NextPage() != 3 {
		t.Errorf("NextPage = %d, want 3", am.NextPage())
	}
}

func TestAtlasManagerAllocPage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am := atlasManager()

	idx0 := am.AllocPage()
	idx1 := am.AllocPage()
	if idx0 != 0 {
		t.Errorf("first AllocPage = %d, want 0", idx0)
	}
	if idx1 != 1 {
		t.Errorf("second AllocPage = %d, want 1", idx1)
	}
	if am.NextPage() != 2 {
		t.Errorf("NextPage = %d, want 2", am.NextPage())
	}
}

func TestAtlasManagerRetainRelease(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am := atlasManager()
	am.RegisterPage(0, nil)

	am.Retain(0)
	am.Retain(0)
	if am.refs[0] != 2 {
		t.Errorf("refs[0] = %d, want 2", am.refs[0])
	}

	am.Release(0)
	if am.refs[0] != 1 {
		t.Errorf("refs[0] = %d, want 1", am.refs[0])
	}

	am.Release(0)
	if am.refs[0] != 0 {
		t.Errorf("refs[0] = %d, want 0", am.refs[0])
	}

	// Release below zero should not go negative.
	am.Release(0)
	if am.refs[0] != 0 {
		t.Errorf("refs[0] = %d, want 0 (should not go negative)", am.refs[0])
	}

	// Release out of range should not panic.
	am.Release(999)
	am.Release(-1)
}

func TestAtlasManagerSetStatic(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am := atlasManager()
	am.RegisterPage(0, nil)

	am.SetStatic(0)
	if !am.static[0] {
		t.Error("page 0 should be static")
	}

	// SetStatic on unregistered index should grow slices without panic.
	am.SetStatic(5)
	if !am.static[5] {
		t.Error("page 5 should be static")
	}
}

func TestAtlasManagerRetainGrowsSlice(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	am := atlasManager()
	// Retain on an index beyond current refs length should grow without panic.
	am.Retain(3)
	if len(am.refs) < 4 {
		t.Errorf("refs length = %d, want >= 4", len(am.refs))
	}
	if am.refs[3] != 1 {
		t.Errorf("refs[3] = %d, want 1", am.refs[3])
	}
}

func TestAtlasManagerReset(t *testing.T) {
	am := atlasManager()
	am.RegisterPage(0, nil)
	resetAtlasManager()

	am2 := atlasManager()
	if am2.PageCount() != 0 {
		t.Errorf("PageCount after reset = %d, want 0", am2.PageCount())
	}
}
