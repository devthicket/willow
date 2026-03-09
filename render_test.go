package willow

import (
	"math"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/render"
)

// helper to build a scene and run traverse without Draw (no ebiten.Image needed)
func traverseScene(s *Scene) {
	s.Pipeline.Commands = s.Pipeline.Commands[:0]
	s.Pipeline.CommandsDirtyThisFrame = false
	// Compute world transforms first (mirrors Scene.Update), then traverse
	// read-only with identity view.
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)
	s.Pipeline.ViewTransform = identityTransform
	treeOrder := 0
	s.Pipeline.Traverse(s.Root, &treeOrder)
}

// --- Command emission ---

func TestSingleSpriteEmitsOneCommand(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	s.Root.AddChild(sprite)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].Type != CommandSprite {
		t.Errorf("Type = %d, want CommandSprite", s.Pipeline.Commands[0].Type)
	}
}

func TestInvisibleNodeNoCommands(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	sprite.Visible_ = false
	s.Root.AddChild(sprite)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 0 {
		t.Errorf("commands = %d, want 0 for invisible node", len(s.Pipeline.Commands))
	}
}

func TestInvisibleSubtreeSkipped(t *testing.T) {
	s := NewScene()
	parent := NewContainer("parent")
	parent.Visible_ = false
	child := NewSprite("child", TextureRegion{Width: 32, Height: 32})
	parent.AddChild(child)
	s.Root.AddChild(parent)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 0 {
		t.Errorf("commands = %d, want 0 for invisible subtree", len(s.Pipeline.Commands))
	}
}

func TestNonRenderableNodeSkipped(t *testing.T) {
	s := NewScene()
	parent := NewSprite("parent", TextureRegion{Width: 32, Height: 32})
	parent.Renderable_ = false
	child := NewSprite("child", TextureRegion{Width: 16, Height: 16})
	parent.AddChild(child)
	s.Root.AddChild(parent)

	traverseScene(s)

	// Parent not renderable → 0 commands from parent
	// But child is renderable → 1 command
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].TextureRegion.Width != 16 {
		t.Error("command should be from child, not parent")
	}
}

func TestContainerNoCommand(t *testing.T) {
	s := NewScene()
	s.Root.AddChild(NewContainer("empty"))

	traverseScene(s)

	if len(s.Pipeline.Commands) != 0 {
		t.Errorf("containers should not emit commands, got %d", len(s.Pipeline.Commands))
	}
}

func TestTreeOrderAssignment(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	b := NewSprite("b", TextureRegion{Width: 2})
	c := NewSprite("c", TextureRegion{Width: 3})
	s.Root.AddChild(a)
	s.Root.AddChild(b)
	s.Root.AddChild(c)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(s.Pipeline.Commands))
	}
	for i := 1; i < len(s.Pipeline.Commands); i++ {
		if s.Pipeline.Commands[i].TreeOrder <= s.Pipeline.Commands[i-1].TreeOrder {
			t.Errorf("treeOrder not strictly increasing: [%d]=%d, [%d]=%d",
				i-1, s.Pipeline.Commands[i-1].TreeOrder, i, s.Pipeline.Commands[i].TreeOrder)
		}
	}
}

func TestWorldAlphaInCommand(t *testing.T) {
	s := NewScene()
	parent := NewContainer("parent")
	parent.Alpha_ = 0.5
	child := NewSprite("child", TextureRegion{Width: 32, Height: 32})
	child.Alpha_ = 0.8
	parent.AddChild(child)
	s.Root.AddChild(parent)

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	// worldAlpha = 0.5 * 0.8 = 0.4, Color.A = 1.0 * 0.4 = 0.4
	if got := float64(s.Pipeline.Commands[0].Color.A); math.Abs(got-0.4) > 1e-6 {
		t.Errorf("cmd.Color.A = %v, want ~0.4", got)
	}
}

// --- Sorting ---

func TestRenderLayerSorting(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	a.RenderLayer = 1
	b := NewSprite("b", TextureRegion{Width: 2})
	b.RenderLayer = 0
	s.Root.AddChild(a)
	s.Root.AddChild(b)

	traverseScene(s)
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	if s.Pipeline.Commands[0].RenderLayer != 0 {
		t.Errorf("first command should be layer 0, got %d", s.Pipeline.Commands[0].RenderLayer)
	}
	if s.Pipeline.Commands[1].RenderLayer != 1 {
		t.Errorf("second command should be layer 1, got %d", s.Pipeline.Commands[1].RenderLayer)
	}
}

func TestGlobalOrderSorting(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	a.GlobalOrder = 10
	b := NewSprite("b", TextureRegion{Width: 2})
	b.GlobalOrder = 5
	s.Root.AddChild(a)
	s.Root.AddChild(b)

	traverseScene(s)
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	if s.Pipeline.Commands[0].GlobalOrder != 5 {
		t.Errorf("first command GlobalOrder = %d, want 5", s.Pipeline.Commands[0].GlobalOrder)
	}
}

func TestTreeOrderPreservedWithinLayer(t *testing.T) {
	s := NewScene()
	// All same layer, GlobalOrder = 0 → tree order should be preserved
	for i := 0; i < 5; i++ {
		sp := NewSprite("", TextureRegion{Width: uint16(i + 1)})
		s.Root.AddChild(sp)
	}

	traverseScene(s)
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	for i := 0; i < 5; i++ {
		if s.Pipeline.Commands[i].TextureRegion.Width != uint16(i+1) {
			t.Errorf("commands[%d].Width = %d, want %d", i, s.Pipeline.Commands[i].TextureRegion.Width, i+1)
		}
	}
}

// --- ZIndex ---

func TestZIndexSorting(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	b := NewSprite("b", TextureRegion{Width: 2})
	c := NewSprite("c", TextureRegion{Width: 3})
	a.SetZIndex(2)
	b.SetZIndex(0)
	c.SetZIndex(1)
	s.Root.AddChild(a)
	s.Root.AddChild(b)
	s.Root.AddChild(c)

	traverseScene(s)

	// b(z=0) should be first, then c(z=1), then a(z=2)
	if len(s.Pipeline.Commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].TextureRegion.Width != 2 {
		t.Errorf("first should be b (width=2), got width=%d", s.Pipeline.Commands[0].TextureRegion.Width)
	}
	if s.Pipeline.Commands[1].TextureRegion.Width != 3 {
		t.Errorf("second should be c (width=3), got width=%d", s.Pipeline.Commands[1].TextureRegion.Width)
	}
	if s.Pipeline.Commands[2].TextureRegion.Width != 1 {
		t.Errorf("third should be a (width=1), got width=%d", s.Pipeline.Commands[2].TextureRegion.Width)
	}
}

// --- Merge sort ---

func TestMergeSortMatchesStdlib(t *testing.T) {
	s := NewScene()
	// Create commands with varied layers and orders
	cmds := []RenderCommand{
		{RenderLayer: 2, GlobalOrder: 0, TreeOrder: 1},
		{RenderLayer: 0, GlobalOrder: 3, TreeOrder: 2},
		{RenderLayer: 0, GlobalOrder: 1, TreeOrder: 3},
		{RenderLayer: 1, GlobalOrder: 0, TreeOrder: 4},
		{RenderLayer: 0, GlobalOrder: 1, TreeOrder: 5},
		{RenderLayer: 2, GlobalOrder: 0, TreeOrder: 6},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 7},
	}

	// Reference: stdlib stable sort
	ref := make([]RenderCommand, len(cmds))
	copy(ref, cmds)
	sort.SliceStable(ref, func(i, j int) bool {
		a, b := ref[i], ref[j]
		if a.RenderLayer != b.RenderLayer {
			return a.RenderLayer < b.RenderLayer
		}
		if a.GlobalOrder != b.GlobalOrder {
			return a.GlobalOrder < b.GlobalOrder
		}
		return a.TreeOrder < b.TreeOrder
	})

	// Our merge sort
	s.Pipeline.Commands = make([]RenderCommand, len(cmds))
	copy(s.Pipeline.Commands, cmds)
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	for i := range s.Pipeline.Commands {
		a, b := s.Pipeline.Commands[i], ref[i]
		if a.RenderLayer != b.RenderLayer || a.GlobalOrder != b.GlobalOrder || a.TreeOrder != b.TreeOrder {
			t.Errorf("index %d: mergeSort=(%d,%d,%d), stdlib=(%d,%d,%d)",
				i, a.RenderLayer, a.GlobalOrder, a.TreeOrder,
				b.RenderLayer, b.GlobalOrder, b.TreeOrder)
		}
	}
}

func TestMergeSortStable(t *testing.T) {
	s := NewScene()
	// All same layer and GlobalOrder  -  treeOrder should be preserved
	s.Pipeline.Commands = make([]RenderCommand, 100)
	for i := range s.Pipeline.Commands {
		s.Pipeline.Commands[i] = RenderCommand{TreeOrder: i}
	}

	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	for i := range s.Pipeline.Commands {
		if s.Pipeline.Commands[i].TreeOrder != i {
			t.Fatalf("stability broken at index %d: treeOrder=%d", i, s.Pipeline.Commands[i].TreeOrder)
		}
	}
}

func TestMergeSortBufferReuse(t *testing.T) {
	s := NewScene()

	// First sort: allocates buffer
	s.Pipeline.Commands = make([]RenderCommand, 50)
	for i := range s.Pipeline.Commands {
		s.Pipeline.Commands[i] = RenderCommand{TreeOrder: 50 - i}
	}
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)
	bufCap := cap(s.Pipeline.SortBuf)

	// Second sort with smaller input: should not reallocate
	s.Pipeline.Commands = make([]RenderCommand, 30)
	for i := range s.Pipeline.Commands {
		s.Pipeline.Commands[i] = RenderCommand{TreeOrder: 30 - i}
	}
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	if cap(s.Pipeline.SortBuf) != bufCap {
		t.Errorf("sortBuf reallocated: was %d, now %d", bufCap, cap(s.Pipeline.SortBuf))
	}
}

func TestMergeSortEmpty(t *testing.T) {
	s := NewScene()
	s.Pipeline.Commands = nil
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf) // should not panic
}

func TestMergeSortSingleElement(t *testing.T) {
	s := NewScene()
	s.Pipeline.Commands = []RenderCommand{{TreeOrder: 1}}
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf) // should not panic
	if s.Pipeline.Commands[0].TreeOrder != 1 {
		t.Error("single element should remain unchanged")
	}
}

// --- CacheAsTree tests ---

// traverseSceneWithCamera runs traverse with a camera's view transform.
func traverseSceneWithCamera(s *Scene, cam *Camera) {
	s.Pipeline.Commands = s.Pipeline.Commands[:0]
	s.Pipeline.CommandsDirtyThisFrame = false
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)
	if cam != nil {
		s.Pipeline.ViewTransform = cam.ComputeViewMatrix()
	} else {
		s.Pipeline.ViewTransform = identityTransform
	}
	treeOrder := 0
	s.Pipeline.Traverse(s.Root, &treeOrder)
}

func TestCacheAsTree_MatchesUncached(t *testing.T) {
	// Build identical scenes  -  one cached, one not.
	build := func(cached bool) *Scene {
		s := NewScene()
		container := NewContainer("c")
		for i := 0; i < 10; i++ {
			sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
			sp.X_ = float64(i * 40)
			sp.Alpha_ = 0.8
			container.AddChild(sp)
		}
		s.Root.AddChild(container)
		if cached {
			container.SetCacheAsTree(true, CacheTreeManual)
		}
		return s
	}

	uncached := build(false)
	cached := build(true)

	// First traverse builds the cache.
	traverseScene(uncached)
	traverseScene(cached)

	// Second traverse replays from cache.
	traverseScene(cached)

	if len(cached.Pipeline.Commands) != len(uncached.Pipeline.Commands) {
		t.Fatalf("command count: cached=%d, uncached=%d", len(cached.Pipeline.Commands), len(uncached.Pipeline.Commands))
	}
	for i := range cached.Pipeline.Commands {
		a, b := cached.Pipeline.Commands[i], uncached.Pipeline.Commands[i]
		if a.Transform != b.Transform {
			t.Errorf("cmd[%d] transform mismatch: %v vs %v", i, a.Transform, b.Transform)
		}
		if a.Color != b.Color {
			t.Errorf("cmd[%d] color mismatch: %v vs %v", i, a.Color, b.Color)
		}
		if a.TextureRegion != b.TextureRegion {
			t.Errorf("cmd[%d] region mismatch: %v vs %v", i, a.TextureRegion, b.TextureRegion)
		}
	}
}

func TestCacheAsTree_ManualModePersists(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	traverseScene(s) // replay

	// Modify a child via setter  -  manual mode should NOT auto-invalidate.
	sp.SetPosition(100, 100)

	traverseScene(s)
	// Cache should still be valid (manual mode ignores setter bubbling).
	// The command should have the OLD transform (from cache).
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.Pipeline.Commands))
	}
	// Old position was X=0, so transform tx should be 0.
	if s.Pipeline.Commands[0].Transform[4] != 0 {
		t.Errorf("manual mode: expected cached transform tx=0, got %v", s.Pipeline.Commands[0].Transform[4])
	}

	// Now manually invalidate.
	container.InvalidateCacheTree()
	traverseScene(s)
	// Should have new position.
	if s.Pipeline.Commands[0].Transform[4] != 100 {
		t.Errorf("after invalidate: expected tx=100, got %v", s.Pipeline.Commands[0].Transform[4])
	}
}

func TestCacheAsTree_AutoModeBubblesOnSetter(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build
	traverseScene(s) // replay (cache hit)

	// Modify a child  -  auto mode should invalidate.
	sp.SetPosition(50, 50)
	traverseScene(s) // rebuild
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].Transform[4] != 50 {
		t.Errorf("auto mode: expected tx=50 after setter, got %v", s.Pipeline.Commands[0].Transform[4])
	}
}

func TestCacheAsTree_DeltaRemap_CameraPan(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	sp.X_ = 100
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	// Build a reference uncached scene.
	ref := NewScene()
	refContainer := NewContainer("c")
	refSp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	refSp.X_ = 100
	refContainer.AddChild(refSp)
	ref.Root.AddChild(refContainer)

	// First traverse to build cache.
	traverseScene(s)

	// Apply a view transform (camera pan).
	s.Pipeline.ViewTransform = [6]float64{1, 0, 0, 1, -50, -30}
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)
	s.Pipeline.Commands = s.Pipeline.Commands[:0]
	s.Pipeline.CommandsDirtyThisFrame = false
	treeOrder := 0
	s.Pipeline.Traverse(s.Root, &treeOrder)

	ref.Pipeline.ViewTransform = [6]float64{1, 0, 0, 1, -50, -30}
	updateWorldTransform(ref.Root, identityTransform, 1.0, false, false)
	ref.Pipeline.Commands = ref.Pipeline.Commands[:0]
	refTreeOrder := 0
	ref.Pipeline.Traverse(ref.Root, &refTreeOrder)

	if len(s.Pipeline.Commands) != len(ref.Pipeline.Commands) {
		t.Fatalf("command count: cached=%d, ref=%d", len(s.Pipeline.Commands), len(ref.Pipeline.Commands))
	}
	for i := range s.Pipeline.Commands {
		a, b := s.Pipeline.Commands[i], ref.Pipeline.Commands[i]
		for j := 0; j < 6; j++ {
			diff := a.Transform[j] - b.Transform[j]
			if diff > 0.01 || diff < -0.01 {
				t.Errorf("cmd[%d] transform[%d]: cached=%v, ref=%v", i, j, a.Transform[j], b.Transform[j])
			}
		}
	}
}

func TestCacheAsTree_AlphaRemap(t *testing.T) {
	s := NewScene()
	parent := NewContainer("parent")
	parent.Alpha_ = 0.5
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	parent.AddChild(container)
	s.Root.AddChild(parent)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	// Parent alpha is 0.5, sprite alpha is 1.0, so worldAlpha = 0.5
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.Pipeline.Commands))
	}
	if math.Abs(float64(s.Pipeline.Commands[0].Color.A)-0.5) > 0.01 {
		t.Errorf("initial alpha: expected ~0.5, got %v", s.Pipeline.Commands[0].Color.A)
	}

	// Change parent alpha.
	parent.Alpha_ = 0.8
	parent.AlphaDirty = true
	traverseScene(s) // replay with alpha remap
	// New worldAlpha = 0.8 * 1.0 = 0.8
	if math.Abs(float64(s.Pipeline.Commands[0].Color.A)-0.8) > 0.01 {
		t.Errorf("remapped alpha: expected ~0.8, got %v", s.Pipeline.Commands[0].Color.A)
	}
}

func TestCacheAsTree_TextureSwap_SamePage_NoInvalidation(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Page: 0, X: 0, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build

	// Swap texture region (same page).
	sp.SetTextureRegion(TextureRegion{Page: 0, X: 32, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})

	traverseScene(s) // should replay from cache with updated UVs
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].TextureRegion.X != 32 {
		t.Errorf("expected updated UV X=32, got %d", s.Pipeline.Commands[0].TextureRegion.X)
	}
}

func TestCacheAsTree_TextureSwap_PageChange_Invalidates(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Page: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build

	// Swap to a different page  -  should invalidate.
	sp.SetTextureRegion(TextureRegion{Page: 1, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})

	if !getCacheTree(container).Dirty {
		t.Error("page change should have invalidated auto-mode cache")
	}
}

func TestCacheAsTree_TreeOps_Invalidate(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build
	if getCacheTree(container).Dirty {
		t.Error("cache should be clean after build")
	}

	// AddChild should invalidate.
	extra := NewSprite("extra", TextureRegion{Width: 16, Height: 16, OriginalW: 16, OriginalH: 16})
	container.AddChild(extra)
	if !getCacheTree(container).Dirty {
		t.Error("AddChild should invalidate auto cache on self")
	}

	traverseScene(s) // rebuild
	// RemoveChild should invalidate.
	container.RemoveChild(extra)
	if !getCacheTree(container).Dirty {
		t.Error("RemoveChild should invalidate auto cache on self")
	}
}

func TestCacheAsTree_MeshBlocksCache(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	mesh := NewMesh("m", nil, []ebiten.Vertex{{}, {}, {}}, []uint16{0, 1, 2})
	container.AddChild(mesh)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build attempt

	// Mesh should block caching  -  cache stays dirty.
	if !getCacheTree(container).Dirty {
		t.Error("mesh subtree should block cache build")
	}
}

func TestCacheAsTree_SortSkip(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	for i := 0; i < 5; i++ {
		sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
		container.AddChild(sp)
	}
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build (dirty)
	if !s.Pipeline.CommandsDirtyThisFrame {
		t.Error("first traverse should be dirty")
	}

	traverseScene(s) // replay (clean)
	if s.Pipeline.CommandsDirtyThisFrame {
		t.Error("second traverse (all cache hits) should not be dirty")
	}
}

func TestCacheAsTree_Disable(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root.AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	traverseScene(s) // replay

	// Disable.
	container.SetCacheAsTree(false)
	if container.CacheData != nil {
		t.Error("cache should be disabled")
	}

	traverseScene(s) // normal traverse
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command after disable, got %d", len(s.Pipeline.Commands))
	}
}

// --- Benchmarks ---

func buildSpriteScene(count int) *Scene {
	s := NewScene()
	for i := 0; i < count; i++ {
		sp := NewSprite("", TextureRegion{Width: 32, Height: 32})
		s.Root.AddChild(sp)
	}
	return s
}

func BenchmarkTraverse1000(b *testing.B) {
	s := buildSpriteScene(1000)
	// Warm up transforms
	traverseScene(s)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Pipeline.Commands = s.Pipeline.Commands[:0]
		treeOrder := 0
		s.Pipeline.Traverse(s.Root, &treeOrder)
	}
}

func BenchmarkTraverse10000(b *testing.B) {
	s := buildSpriteScene(10000)
	traverseScene(s)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Pipeline.Commands = s.Pipeline.Commands[:0]
		treeOrder := 0
		s.Pipeline.Traverse(s.Root, &treeOrder)
	}
}

func BenchmarkCommandSort10000(b *testing.B) {
	s := NewScene()
	s.Pipeline.Commands = make([]RenderCommand, 10000)
	for i := range s.Pipeline.Commands {
		s.Pipeline.Commands[i] = RenderCommand{
			RenderLayer: uint8(i % 4),
			GlobalOrder: i % 10,
			TreeOrder:   i,
		}
	}
	// Warmup to allocate sortBuf
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reverse to force work
		for i, j := 0, len(s.Pipeline.Commands)-1; i < j; i, j = i+1, j-1 {
			s.Pipeline.Commands[i], s.Pipeline.Commands[j] = s.Pipeline.Commands[j], s.Pipeline.Commands[i]
		}
		render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)
	}
}
