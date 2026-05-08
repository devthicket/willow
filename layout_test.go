package willow

import "testing"

// TestLayout_ReturnsCfgRegardlessOfOutside verifies that gameBase.Layout
// always returns the configured logical size and ignores the outside
// (window/monitor) dimensions reported by Ebitengine.
func TestLayout_ReturnsCfgRegardlessOfOutside(t *testing.T) {
	b := &gameBase{cfgW: 640, cfgH: 480}
	cases := []struct{ ow, oh int }{
		{0, 0},
		{640, 480},
		{2560, 1440},
		{8000, 8000},
		{320, 240},
	}
	for _, c := range cases {
		w, h := b.Layout(c.ow, c.oh)
		if w != 640 || h != 480 {
			t.Fatalf("Layout(%d,%d) = (%d,%d); want (640,480)", c.ow, c.oh, w, h)
		}
	}
}

// TestLayout_OnResizeFiresOnceAtStartup verifies that Scene.OnResize is
// invoked exactly once on the first Layout call (zero -> cfg) and not
// again on subsequent Layout calls with different outside dimensions.
func TestLayout_OnResizeFiresOnceAtStartup(t *testing.T) {
	scn := NewScene()
	var calls int
	var lastW, lastH int
	scn.SetOnResize(func(w, h int) {
		calls++
		lastW, lastH = w, h
	})
	b := &gameBase{
		cfgW:         800,
		cfgH:         600,
		currentScene: func() *Scene { return scn },
	}
	b.Layout(1024, 768)
	b.Layout(2560, 1440)
	b.Layout(800, 600)
	b.Layout(0, 0)
	if calls != 1 {
		t.Fatalf("OnResize call count = %d; want 1", calls)
	}
	if lastW != 800 || lastH != 600 {
		t.Fatalf("OnResize received (%d,%d); want (800,600)", lastW, lastH)
	}
}

// TestLayout_ManagerResolverPicksUpSceneSwitch verifies that when the
// current scene changes between Layout calls (manager path), the resolver
// returns the new scene. With the once-per-run OnResize contract, the new
// scene's OnResize must NOT fire on subsequent Layouts; the test confirms
// the resolver indirection works without re-firing.
func TestLayout_ManagerResolverPicksUpSceneSwitch(t *testing.T) {
	a := NewScene()
	c := NewScene()
	var aCalls, cCalls int
	a.SetOnResize(func(w, h int) { aCalls++ })
	c.SetOnResize(func(w, h int) { cCalls++ })

	current := a
	b := &gameBase{
		cfgW:         400,
		cfgH:         300,
		currentScene: func() *Scene { return current },
	}
	b.Layout(1920, 1080)
	if aCalls != 1 || cCalls != 0 {
		t.Fatalf("after first Layout: aCalls=%d cCalls=%d; want 1,0", aCalls, cCalls)
	}
	current = c
	b.Layout(1280, 720)
	b.Layout(800, 600)
	if aCalls != 1 || cCalls != 0 {
		t.Fatalf("after switch: aCalls=%d cCalls=%d; want 1,0 (logical size unchanged)", aCalls, cCalls)
	}
}

// TestLayout_NilSceneIsSafe verifies the resolver may legitimately return
// nil (e.g. SceneManager with no current scene) without panicking.
func TestLayout_NilSceneIsSafe(t *testing.T) {
	b := &gameBase{
		cfgW:         640,
		cfgH:         480,
		currentScene: func() *Scene { return nil },
	}
	w, h := b.Layout(1024, 768)
	if w != 640 || h != 480 {
		t.Fatalf("Layout = (%d,%d); want (640,480)", w, h)
	}
}

// TestLayout_NilResolverIsSafe verifies missing resolver is non-fatal.
func TestLayout_NilResolverIsSafe(t *testing.T) {
	b := &gameBase{cfgW: 640, cfgH: 480}
	w, h := b.Layout(1024, 768)
	if w != 640 || h != 480 {
		t.Fatalf("Layout = (%d,%d); want (640,480)", w, h)
	}
}
