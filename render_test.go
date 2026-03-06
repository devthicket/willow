package willow

import (
	"math"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// helper to build a scene and run traverse without Draw (no ebiten.Image needed)
func traverseScene(s *Scene) {
	s.commands = s.commands[:0]
	s.commandsDirtyThisFrame = false
	// Compute world transforms first (mirrors Scene.Update), then traverse
	// read-only with identity view.
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)
	s.viewTransform = identityTransform
	treeOrder := 0
	s.traverse(s.root, &treeOrder)
}

// --- Command emission ---

func TestSingleSpriteEmitsOneCommand(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	s.Root().AddChild(sprite)

	traverseScene(s)

	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	if s.commands[0].Type != CommandSprite {
		t.Errorf("Type = %d, want CommandSprite", s.commands[0].Type)
	}
}

func TestInvisibleNodeNoCommands(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	sprite.visible = false
	s.Root().AddChild(sprite)

	traverseScene(s)

	if len(s.commands) != 0 {
		t.Errorf("commands = %d, want 0 for invisible node", len(s.commands))
	}
}

func TestInvisibleSubtreeSkipped(t *testing.T) {
	s := NewScene()
	parent := NewContainer("parent")
	parent.visible = false
	child := NewSprite("child", TextureRegion{Width: 32, Height: 32})
	parent.AddChild(child)
	s.Root().AddChild(parent)

	traverseScene(s)

	if len(s.commands) != 0 {
		t.Errorf("commands = %d, want 0 for invisible subtree", len(s.commands))
	}
}

func TestNonRenderableNodeSkipped(t *testing.T) {
	s := NewScene()
	parent := NewSprite("parent", TextureRegion{Width: 32, Height: 32})
	parent.renderable = false
	child := NewSprite("child", TextureRegion{Width: 16, Height: 16})
	parent.AddChild(child)
	s.Root().AddChild(parent)

	traverseScene(s)

	// Parent not renderable → 0 commands from parent
	// But child is renderable → 1 command
	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	if s.commands[0].TextureRegion.Width != 16 {
		t.Error("command should be from child, not parent")
	}
}

func TestContainerNoCommand(t *testing.T) {
	s := NewScene()
	s.Root().AddChild(NewContainer("empty"))

	traverseScene(s)

	if len(s.commands) != 0 {
		t.Errorf("containers should not emit commands, got %d", len(s.commands))
	}
}

func TestTreeOrderAssignment(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	b := NewSprite("b", TextureRegion{Width: 2})
	c := NewSprite("c", TextureRegion{Width: 3})
	s.Root().AddChild(a)
	s.Root().AddChild(b)
	s.Root().AddChild(c)

	traverseScene(s)

	if len(s.commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(s.commands))
	}
	for i := 1; i < len(s.commands); i++ {
		if s.commands[i].treeOrder <= s.commands[i-1].treeOrder {
			t.Errorf("treeOrder not strictly increasing: [%d]=%d, [%d]=%d",
				i-1, s.commands[i-1].treeOrder, i, s.commands[i].treeOrder)
		}
	}
}

func TestWorldAlphaInCommand(t *testing.T) {
	s := NewScene()
	parent := NewContainer("parent")
	parent.alpha = 0.5
	child := NewSprite("child", TextureRegion{Width: 32, Height: 32})
	child.alpha = 0.8
	parent.AddChild(child)
	s.Root().AddChild(parent)

	traverseScene(s)

	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	// worldAlpha = 0.5 * 0.8 = 0.4, Color.A = 1.0 * 0.4 = 0.4
	if got := float64(s.commands[0].Color.A); math.Abs(got-0.4) > 1e-6 {
		t.Errorf("cmd.Color.A = %v, want ~0.4", got)
	}
}

// --- Sorting ---

func TestRenderLayerSorting(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	a.renderLayer = 1
	b := NewSprite("b", TextureRegion{Width: 2})
	b.renderLayer = 0
	s.Root().AddChild(a)
	s.Root().AddChild(b)

	traverseScene(s)
	s.mergeSort()

	if s.commands[0].RenderLayer != 0 {
		t.Errorf("first command should be layer 0, got %d", s.commands[0].RenderLayer)
	}
	if s.commands[1].RenderLayer != 1 {
		t.Errorf("second command should be layer 1, got %d", s.commands[1].RenderLayer)
	}
}

func TestGlobalOrderSorting(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{Width: 1})
	a.globalOrder = 10
	b := NewSprite("b", TextureRegion{Width: 2})
	b.globalOrder = 5
	s.Root().AddChild(a)
	s.Root().AddChild(b)

	traverseScene(s)
	s.mergeSort()

	if s.commands[0].GlobalOrder != 5 {
		t.Errorf("first command GlobalOrder = %d, want 5", s.commands[0].GlobalOrder)
	}
}

func TestTreeOrderPreservedWithinLayer(t *testing.T) {
	s := NewScene()
	// All same layer, GlobalOrder = 0 → tree order should be preserved
	for i := 0; i < 5; i++ {
		sp := NewSprite("", TextureRegion{Width: uint16(i + 1)})
		s.Root().AddChild(sp)
	}

	traverseScene(s)
	s.mergeSort()

	for i := 0; i < 5; i++ {
		if s.commands[i].TextureRegion.Width != uint16(i+1) {
			t.Errorf("commands[%d].Width = %d, want %d", i, s.commands[i].TextureRegion.Width, i+1)
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
	s.Root().AddChild(a)
	s.Root().AddChild(b)
	s.Root().AddChild(c)

	traverseScene(s)

	// b(z=0) should be first, then c(z=1), then a(z=2)
	if len(s.commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(s.commands))
	}
	if s.commands[0].TextureRegion.Width != 2 {
		t.Errorf("first should be b (width=2), got width=%d", s.commands[0].TextureRegion.Width)
	}
	if s.commands[1].TextureRegion.Width != 3 {
		t.Errorf("second should be c (width=3), got width=%d", s.commands[1].TextureRegion.Width)
	}
	if s.commands[2].TextureRegion.Width != 1 {
		t.Errorf("third should be a (width=1), got width=%d", s.commands[2].TextureRegion.Width)
	}
}

// --- Merge sort ---

func TestMergeSortMatchesStdlib(t *testing.T) {
	s := NewScene()
	// Create commands with varied layers and orders
	cmds := []RenderCommand{
		{RenderLayer: 2, GlobalOrder: 0, treeOrder: 1},
		{RenderLayer: 0, GlobalOrder: 3, treeOrder: 2},
		{RenderLayer: 0, GlobalOrder: 1, treeOrder: 3},
		{RenderLayer: 1, GlobalOrder: 0, treeOrder: 4},
		{RenderLayer: 0, GlobalOrder: 1, treeOrder: 5},
		{RenderLayer: 2, GlobalOrder: 0, treeOrder: 6},
		{RenderLayer: 0, GlobalOrder: 0, treeOrder: 7},
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
		return a.treeOrder < b.treeOrder
	})

	// Our merge sort
	s.commands = make([]RenderCommand, len(cmds))
	copy(s.commands, cmds)
	s.mergeSort()

	for i := range s.commands {
		a, b := s.commands[i], ref[i]
		if a.RenderLayer != b.RenderLayer || a.GlobalOrder != b.GlobalOrder || a.treeOrder != b.treeOrder {
			t.Errorf("index %d: mergeSort=(%d,%d,%d), stdlib=(%d,%d,%d)",
				i, a.RenderLayer, a.GlobalOrder, a.treeOrder,
				b.RenderLayer, b.GlobalOrder, b.treeOrder)
		}
	}
}

func TestMergeSortStable(t *testing.T) {
	s := NewScene()
	// All same layer and GlobalOrder  -  treeOrder should be preserved
	s.commands = make([]RenderCommand, 100)
	for i := range s.commands {
		s.commands[i] = RenderCommand{treeOrder: i}
	}

	s.mergeSort()

	for i := range s.commands {
		if s.commands[i].treeOrder != i {
			t.Fatalf("stability broken at index %d: treeOrder=%d", i, s.commands[i].treeOrder)
		}
	}
}

func TestMergeSortBufferReuse(t *testing.T) {
	s := NewScene()

	// First sort: allocates buffer
	s.commands = make([]RenderCommand, 50)
	for i := range s.commands {
		s.commands[i] = RenderCommand{treeOrder: 50 - i}
	}
	s.mergeSort()
	bufCap := cap(s.sortBuf)

	// Second sort with smaller input: should not reallocate
	s.commands = make([]RenderCommand, 30)
	for i := range s.commands {
		s.commands[i] = RenderCommand{treeOrder: 30 - i}
	}
	s.mergeSort()

	if cap(s.sortBuf) != bufCap {
		t.Errorf("sortBuf reallocated: was %d, now %d", bufCap, cap(s.sortBuf))
	}
}

func TestMergeSortEmpty(t *testing.T) {
	s := NewScene()
	s.commands = nil
	s.mergeSort() // should not panic
}

func TestMergeSortSingleElement(t *testing.T) {
	s := NewScene()
	s.commands = []RenderCommand{{treeOrder: 1}}
	s.mergeSort() // should not panic
	if s.commands[0].treeOrder != 1 {
		t.Error("single element should remain unchanged")
	}
}

// --- CacheAsTree tests ---

// traverseSceneWithCamera runs traverse with a camera's view transform.
func traverseSceneWithCamera(s *Scene, cam *Camera) {
	s.commands = s.commands[:0]
	s.commandsDirtyThisFrame = false
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)
	if cam != nil {
		s.viewTransform = cam.computeViewMatrix()
	} else {
		s.viewTransform = identityTransform
	}
	treeOrder := 0
	s.traverse(s.root, &treeOrder)
}

func TestCacheAsTree_MatchesUncached(t *testing.T) {
	// Build identical scenes  -  one cached, one not.
	build := func(cached bool) *Scene {
		s := NewScene()
		container := NewContainer("c")
		for i := 0; i < 10; i++ {
			sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
			sp.x = float64(i * 40)
			sp.alpha = 0.8
			container.AddChild(sp)
		}
		s.Root().AddChild(container)
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

	if len(cached.commands) != len(uncached.commands) {
		t.Fatalf("command count: cached=%d, uncached=%d", len(cached.commands), len(uncached.commands))
	}
	for i := range cached.commands {
		a, b := cached.commands[i], uncached.commands[i]
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
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	traverseScene(s) // replay

	// Modify a child via setter  -  manual mode should NOT auto-invalidate.
	sp.SetPosition(100, 100)

	traverseScene(s)
	// Cache should still be valid (manual mode ignores setter bubbling).
	// The command should have the OLD transform (from cache).
	if len(s.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.commands))
	}
	// Old position was X=0, so transform tx should be 0.
	if s.commands[0].Transform[4] != 0 {
		t.Errorf("manual mode: expected cached transform tx=0, got %v", s.commands[0].Transform[4])
	}

	// Now manually invalidate.
	container.InvalidateCacheTree()
	traverseScene(s)
	// Should have new position.
	if s.commands[0].Transform[4] != 100 {
		t.Errorf("after invalidate: expected tx=100, got %v", s.commands[0].Transform[4])
	}
}

func TestCacheAsTree_AutoModeBubblesOnSetter(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build
	traverseScene(s) // replay (cache hit)

	// Modify a child  -  auto mode should invalidate.
	sp.SetPosition(50, 50)
	traverseScene(s) // rebuild
	if len(s.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.commands))
	}
	if s.commands[0].Transform[4] != 50 {
		t.Errorf("auto mode: expected tx=50 after setter, got %v", s.commands[0].Transform[4])
	}
}

func TestCacheAsTree_DeltaRemap_CameraPan(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	sp.x = 100
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	// Build a reference uncached scene.
	ref := NewScene()
	refContainer := NewContainer("c")
	refSp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	refSp.x = 100
	refContainer.AddChild(refSp)
	ref.Root().AddChild(refContainer)

	// First traverse to build cache.
	traverseScene(s)

	// Apply a view transform (camera pan).
	s.viewTransform = [6]float64{1, 0, 0, 1, -50, -30}
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)
	s.commands = s.commands[:0]
	s.commandsDirtyThisFrame = false
	treeOrder := 0
	s.traverse(s.root, &treeOrder)

	ref.viewTransform = [6]float64{1, 0, 0, 1, -50, -30}
	updateWorldTransform(ref.root, identityTransform, 1.0, false, false)
	ref.commands = ref.commands[:0]
	refTreeOrder := 0
	ref.traverse(ref.root, &refTreeOrder)

	if len(s.commands) != len(ref.commands) {
		t.Fatalf("command count: cached=%d, ref=%d", len(s.commands), len(ref.commands))
	}
	for i := range s.commands {
		a, b := s.commands[i], ref.commands[i]
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
	parent.alpha = 0.5
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	parent.AddChild(container)
	s.Root().AddChild(parent)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	// Parent alpha is 0.5, sprite alpha is 1.0, so worldAlpha = 0.5
	if len(s.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.commands))
	}
	if math.Abs(float64(s.commands[0].Color.A)-0.5) > 0.01 {
		t.Errorf("initial alpha: expected ~0.5, got %v", s.commands[0].Color.A)
	}

	// Change parent alpha.
	parent.alpha = 0.8
	parent.alphaDirty = true
	traverseScene(s) // replay with alpha remap
	// New worldAlpha = 0.8 * 1.0 = 0.8
	if math.Abs(float64(s.commands[0].Color.A)-0.8) > 0.01 {
		t.Errorf("remapped alpha: expected ~0.8, got %v", s.commands[0].Color.A)
	}
}

func TestCacheAsTree_TextureSwap_SamePage_NoInvalidation(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Page: 0, X: 0, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build

	// Swap texture region (same page).
	sp.SetTextureRegion(TextureRegion{Page: 0, X: 32, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})

	traverseScene(s) // should replay from cache with updated UVs
	if len(s.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(s.commands))
	}
	if s.commands[0].TextureRegion.X != 32 {
		t.Errorf("expected updated UV X=32, got %d", s.commands[0].TextureRegion.X)
	}
}

func TestCacheAsTree_TextureSwap_PageChange_Invalidates(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Page: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build

	// Swap to a different page  -  should invalidate.
	sp.SetTextureRegion(TextureRegion{Page: 1, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})

	if !container.cacheTree.dirty {
		t.Error("page change should have invalidated auto-mode cache")
	}
}

func TestCacheAsTree_TreeOps_Invalidate(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeAuto)

	traverseScene(s) // build
	if container.cacheTree.dirty {
		t.Error("cache should be clean after build")
	}

	// AddChild should invalidate.
	extra := NewSprite("extra", TextureRegion{Width: 16, Height: 16, OriginalW: 16, OriginalH: 16})
	container.AddChild(extra)
	if !container.cacheTree.dirty {
		t.Error("AddChild should invalidate auto cache on self")
	}

	traverseScene(s) // rebuild
	// RemoveChild should invalidate.
	container.RemoveChild(extra)
	if !container.cacheTree.dirty {
		t.Error("RemoveChild should invalidate auto cache on self")
	}
}

func TestCacheAsTree_MeshBlocksCache(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	mesh := NewMesh("m", nil, []ebiten.Vertex{{}, {}, {}}, []uint16{0, 1, 2})
	container.AddChild(mesh)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build attempt

	// Mesh should block caching  -  cache stays dirty.
	if !container.cacheTree.dirty {
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
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build (dirty)
	if !s.commandsDirtyThisFrame {
		t.Error("first traverse should be dirty")
	}

	traverseScene(s) // replay (clean)
	if s.commandsDirtyThisFrame {
		t.Error("second traverse (all cache hits) should not be dirty")
	}
}

func TestCacheAsTree_Disable(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	sp := NewSprite("sp", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	container.AddChild(sp)
	s.Root().AddChild(container)
	container.SetCacheAsTree(true, CacheTreeManual)

	traverseScene(s) // build
	traverseScene(s) // replay

	// Disable.
	container.SetCacheAsTree(false)
	if container.cacheTree != nil {
		t.Error("cache should be disabled")
	}

	traverseScene(s) // normal traverse
	if len(s.commands) != 1 {
		t.Fatalf("expected 1 command after disable, got %d", len(s.commands))
	}
}

// --- Benchmarks ---

func buildSpriteScene(count int) *Scene {
	s := NewScene()
	for i := 0; i < count; i++ {
		sp := NewSprite("", TextureRegion{Width: 32, Height: 32})
		s.Root().AddChild(sp)
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
		s.commands = s.commands[:0]
		treeOrder := 0
		s.traverse(s.root, &treeOrder)
	}
}

func BenchmarkTraverse10000(b *testing.B) {
	s := buildSpriteScene(10000)
	traverseScene(s)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.commands = s.commands[:0]
		treeOrder := 0
		s.traverse(s.root, &treeOrder)
	}
}

func BenchmarkCommandSort10000(b *testing.B) {
	s := NewScene()
	s.commands = make([]RenderCommand, 10000)
	for i := range s.commands {
		s.commands[i] = RenderCommand{
			RenderLayer: uint8(i % 4),
			GlobalOrder: i % 10,
			treeOrder:   i,
		}
	}
	// Warmup to allocate sortBuf
	s.mergeSort()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reverse to force work
		for i, j := 0, len(s.commands)-1; i < j; i, j = i+1, j-1 {
			s.commands[i], s.commands[j] = s.commands[j], s.commands[i]
		}
		s.mergeSort()
	}
}
