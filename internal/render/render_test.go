package render

import (
	"testing"

	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

func TestAffine32(t *testing.T) {
	m := [6]float64{1, 2, 3, 4, 5, 6}
	f := Affine32(m)
	for i := range m {
		if float64(f[i]) != m[i] {
			t.Errorf("Affine32[%d] = %f, want %f", i, float64(f[i]), m[i])
		}
	}
}

func TestMultiplyAffine32_Identity(t *testing.T) {
	id := IdentityTransform32
	m := [6]float32{2, 0, 0, 3, 10, 20}
	result := MultiplyAffine32(id, m)
	for i := range m {
		if result[i] != m[i] {
			t.Errorf("id*m[%d] = %f, want %f", i, result[i], m[i])
		}
	}
}

func TestInvertAffine32_Identity(t *testing.T) {
	id := IdentityTransform32
	inv := InvertAffine32(id)
	for i := range id {
		if inv[i] != id[i] {
			t.Errorf("inv(id)[%d] = %f, want %f", i, inv[i], id[i])
		}
	}
}

func TestInvertAffine32_Scale(t *testing.T) {
	m := [6]float32{2, 0, 0, 3, 0, 0}
	inv := InvertAffine32(m)
	result := MultiplyAffine32(m, inv)
	for i := range IdentityTransform32 {
		diff := result[i] - IdentityTransform32[i]
		if diff > 1e-5 || diff < -1e-5 {
			t.Errorf("m*inv[%d] = %f, want %f", i, result[i], IdentityTransform32[i])
		}
	}
}

func TestCommandLessOrEqual(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1}
	b := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 2}
	if !CommandLessOrEqual(a, b) {
		t.Error("a <= b should be true")
	}
	if CommandLessOrEqual(b, a) {
		t.Error("b <= a should be false")
	}
	// Equal
	if !CommandLessOrEqual(a, a) {
		t.Error("a <= a should be true (stability)")
	}
}

func TestCommandLessOrEqual_LayerPriority(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: 100, TreeOrder: 100}
	b := RenderCommand{RenderLayer: 1, GlobalOrder: 0, TreeOrder: 0}
	if !CommandLessOrEqual(a, b) {
		t.Error("lower layer should sort first")
	}
}

func TestMergeSort_AlreadySorted(t *testing.T) {
	cmds := []RenderCommand{
		{TreeOrder: 1},
		{TreeOrder: 2},
		{TreeOrder: 3},
	}
	var buf []RenderCommand
	MergeSort(cmds, &buf)
	for i, cmd := range cmds {
		if cmd.TreeOrder != i+1 {
			t.Errorf("cmds[%d].TreeOrder = %d, want %d", i, cmd.TreeOrder, i+1)
		}
	}
}

func TestMergeSort_Reverse(t *testing.T) {
	cmds := []RenderCommand{
		{TreeOrder: 3},
		{TreeOrder: 1},
		{TreeOrder: 2},
	}
	var buf []RenderCommand
	MergeSort(cmds, &buf)
	expected := []int{1, 2, 3}
	for i, cmd := range cmds {
		if cmd.TreeOrder != expected[i] {
			t.Errorf("cmds[%d].TreeOrder = %d, want %d", i, cmd.TreeOrder, expected[i])
		}
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := [][2]int{{0, 1}, {1, 1}, {2, 2}, {3, 4}, {5, 8}, {127, 128}, {256, 256}}
	for _, tt := range tests {
		got := NextPowerOfTwo(tt[0])
		if got != tt[1] {
			t.Errorf("NextPowerOfTwo(%d) = %d, want %d", tt[0], got, tt[1])
		}
	}
}

func TestRenderTexturePool_AcquireRelease(t *testing.T) {
	var pool RenderTexturePool
	img := pool.Acquire(10, 10)
	if img == nil {
		t.Fatal("Acquire returned nil")
	}
	pool.Release(img)
	img2 := pool.Acquire(10, 10)
	if img2 != img {
		t.Error("expected reuse from pool")
	}
}

func TestBatchKey_Equality(t *testing.T) {
	a := BatchKey{TargetID: 0, ShaderID: 0, Blend: types.BlendNormal, Page: 1}
	b := BatchKey{TargetID: 0, ShaderID: 0, Blend: types.BlendNormal, Page: 1}
	if a != b {
		t.Error("identical batch keys should be equal")
	}
	c := BatchKey{TargetID: 0, ShaderID: 0, Blend: types.BlendNormal, Page: 2}
	if a == c {
		t.Error("different pages should not be equal")
	}
}

func TestColor32_ZeroSentinel(t *testing.T) {
	c := Color32{}
	if c.R != 0 || c.G != 0 || c.B != 0 || c.A != 0 {
		t.Error("zero Color32 should have all zero fields")
	}
}

func TestNodeColor32(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeSprite)
	n.WorldAlpha = 0.5
	c := nodeColor32(n)
	// White color with 0.5 alpha
	if c.R != 1.0 || c.G != 1.0 || c.B != 1.0 {
		t.Errorf("nodeColor32 RGB = (%f,%f,%f), want (1,1,1)", c.R, c.G, c.B)
	}
	if c.A != 0.5 {
		t.Errorf("nodeColor32 A = %f, want 0.5", c.A)
	}
}

func TestRebuildSortedChildren(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	c1 := node.NewNode("c1", types.NodeTypeSprite)
	c1.ZIndex_ = 3
	c2 := node.NewNode("c2", types.NodeTypeSprite)
	c2.ZIndex_ = 1
	c3 := node.NewNode("c3", types.NodeTypeSprite)
	c3.ZIndex_ = 2

	parent.Children_ = []*node.Node{c1, c2, c3}
	parent.ChildrenSorted = false

	RebuildSortedChildren(parent)

	if !parent.ChildrenSorted {
		t.Error("ChildrenSorted should be true after rebuild")
	}
	if len(parent.SortedChildren) != 3 {
		t.Fatalf("SortedChildren len = %d, want 3", len(parent.SortedChildren))
	}
	if parent.SortedChildren[0] != c2 || parent.SortedChildren[1] != c3 || parent.SortedChildren[2] != c1 {
		t.Error("children not sorted by ZIndex")
	}
}

func TestSubtreeBounds_SingleSprite(t *testing.T) {
	n := node.NewNode("sprite", types.NodeTypeSprite)
	n.TextureRegion_ = types.TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32}
	b := SubtreeBounds(n)
	if b.Width != 32 || b.Height != 32 {
		t.Errorf("bounds = (%f, %f), want (32, 32)", b.Width, b.Height)
	}
}

func TestWorldAABB_Identity(t *testing.T) {
	id := node.IdentityTransform
	aabb := WorldAABB(id, 100, 50)
	if aabb.X != 0 || aabb.Y != 0 || aabb.Width != 100 || aabb.Height != 50 {
		t.Errorf("WorldAABB = %+v, want (0,0,100,50)", aabb)
	}
}

func TestRectUnion(t *testing.T) {
	a := types.Rect{X: 0, Y: 0, Width: 10, Height: 10}
	b := types.Rect{X: 5, Y: 5, Width: 10, Height: 10}
	u := RectUnion(a, b)
	if u.X != 0 || u.Y != 0 || u.Width != 15 || u.Height != 15 {
		t.Errorf("RectUnion = %+v, want (0,0,15,15)", u)
	}
}

func TestCacheTreeData_GetSet(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	if GetCacheTreeData(n) != nil {
		t.Error("CacheData should start nil")
	}
	ctd := &CacheTreeData{Dirty: true}
	SetCacheTreeData(n, ctd)
	got := GetCacheTreeData(n)
	if got != ctd {
		t.Error("GetCacheTreeData should return what was set")
	}
	if !got.Dirty {
		t.Error("Dirty should be true")
	}
}

func TestFilterChainPaddingAny_Empty(t *testing.T) {
	pad := filterChainPaddingAny(nil)
	if pad != 0 {
		t.Errorf("padding for nil filters = %d, want 0", pad)
	}
}

func TestCommandGeoM(t *testing.T) {
	cmd := RenderCommand{
		Transform: [6]float32{1, 0, 0, 1, 10, 20},
	}
	m := CommandGeoM(&cmd)
	if m.Element(0, 2) != 10 || m.Element(1, 2) != 20 {
		t.Errorf("GeoM translation = (%f, %f), want (10, 20)", m.Element(0, 2), m.Element(1, 2))
	}
}

func TestClamp01(t *testing.T) {
	if Clamp01(-0.5) != 0 {
		t.Error("Clamp01(-0.5) should be 0")
	}
	if Clamp01(0.5) != 0.5 {
		t.Error("Clamp01(0.5) should be 0.5")
	}
	if Clamp01(1.5) != 1 {
		t.Error("Clamp01(1.5) should be 1")
	}
}

// --- Additional tests migrated from archived rendertarget_test.go ---

func TestNextPowerOfTwo_Thorough(t *testing.T) {
	tests := []struct {
		input, want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{127, 128},
		{128, 128},
		{129, 256},
		{255, 256},
		{256, 256},
		{1000, 1024},
	}
	for _, tt := range tests {
		got := NextPowerOfTwo(tt.input)
		if got != tt.want {
			t.Errorf("NextPowerOfTwo(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPoolAcquireReturnsPow2(t *testing.T) {
	var pool RenderTexturePool
	img := pool.Acquire(100, 50)
	defer pool.Release(img)

	b := img.Bounds()
	if b.Dx() != 128 {
		t.Errorf("width = %d, want 128 (next pow2 of 100)", b.Dx())
	}
	if b.Dy() != 64 {
		t.Errorf("height = %d, want 64 (next pow2 of 50)", b.Dy())
	}
}

func TestPoolDifferentSizes(t *testing.T) {
	var pool RenderTexturePool
	a := pool.Acquire(32, 32)
	b := pool.Acquire(64, 64)
	if a == b {
		t.Error("different sizes should return different images")
	}
	pool.Release(a)
	pool.Release(b)
}

func TestPoolReleaseNilNoPanic(t *testing.T) {
	var pool RenderTexturePool
	pool.Release(nil) // should not panic
}

func TestSubtreeBoundsWithChildren(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	child := node.NewNode("child", types.NodeTypeSprite)
	child.TextureRegion_ = types.TextureRegion{Width: 20, Height: 20, OriginalW: 20, OriginalH: 20}
	child.X_ = 50
	child.Y_ = 50
	parent.AddChild(child)

	b := SubtreeBounds(parent)
	// Child at (50,50) with size 20x20 -> bounds should be [50,50,20,20]
	if b.X != 50 || b.Y != 50 || b.Width != 20 || b.Height != 20 {
		t.Errorf("bounds = %v, want {50 50 20 20}", b)
	}
}

func TestSubtreeBoundsMultipleChildren(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	a := node.NewNode("a", types.NodeTypeSprite)
	a.TextureRegion_ = types.TextureRegion{Width: 10, Height: 10, OriginalW: 10, OriginalH: 10}
	b := node.NewNode("b", types.NodeTypeSprite)
	b.TextureRegion_ = types.TextureRegion{Width: 10, Height: 10, OriginalW: 10, OriginalH: 10}
	b.X_ = 100
	b.Y_ = 100
	parent.AddChild(a)
	parent.AddChild(b)

	bounds := SubtreeBounds(parent)
	// a at (0,0)+10x10, b at (100,100)+10x10 -> union [0,0,110,110]
	if bounds.X != 0 || bounds.Y != 0 {
		t.Errorf("origin = (%v, %v), want (0, 0)", bounds.X, bounds.Y)
	}
	if bounds.Width != 110 || bounds.Height != 110 {
		t.Errorf("size = (%v, %v), want (110, 110)", bounds.Width, bounds.Height)
	}
}

func TestSubtreeBoundsEmptyContainer(t *testing.T) {
	c := node.NewNode("empty", types.NodeTypeContainer)
	b := SubtreeBounds(c)
	// No renderable content -> zero bounds.
	if b.Width != 0 || b.Height != 0 {
		t.Errorf("empty container bounds = %v, want zero", b)
	}
}

// --- Benchmarks ---

func BenchmarkPoolAcquireRelease(b *testing.B) {
	var pool RenderTexturePool
	// Warmup: create the bucket.
	img := pool.Acquire(256, 256)
	pool.Release(img)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		img := pool.Acquire(256, 256)
		pool.Release(img)
	}
}
