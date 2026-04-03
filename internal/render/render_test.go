package render

import (
	"math"
	"math/rand"
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
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
	aabb := types.WorldAABB(id, 100, 50)
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
	if types.Clamp01(-0.5) != 0 {
		t.Error("Clamp01(-0.5) should be 0")
	}
	if types.Clamp01(0.5) != 0.5 {
		t.Error("Clamp01(0.5) should be 0.5")
	}
	if types.Clamp01(1.5) != 1 {
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

// --- isSorted tests ---

func TestIsSorted_Empty(t *testing.T) {
	if !isSorted(nil) {
		t.Error("nil slice should be considered sorted")
	}
	if !isSorted([]RenderCommand{}) {
		t.Error("empty slice should be considered sorted")
	}
}

func TestIsSorted_Single(t *testing.T) {
	cmds := []RenderCommand{{TreeOrder: 42}}
	if !isSorted(cmds) {
		t.Error("single-element slice should be sorted")
	}
}

func TestIsSorted_Sorted(t *testing.T) {
	cmds := []RenderCommand{
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 2},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 3},
	}
	if !isSorted(cmds) {
		t.Error("sorted slice should return true")
	}
}

func TestIsSorted_Unsorted(t *testing.T) {
	cmds := []RenderCommand{
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 3},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 2},
	}
	if isSorted(cmds) {
		t.Error("unsorted slice should return false")
	}
}

func TestIsSorted_LayerBreak(t *testing.T) {
	cmds := []RenderCommand{
		{RenderLayer: 1, TreeOrder: 1},
		{RenderLayer: 0, TreeOrder: 2},
	}
	if isSorted(cmds) {
		t.Error("higher layer before lower layer should be unsorted")
	}
}

// --- RadixSort tests ---

func TestRadixSort_AlreadySorted(t *testing.T) {
	cmds := []RenderCommand{
		{TreeOrder: 1},
		{TreeOrder: 2},
		{TreeOrder: 3},
	}
	var buf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmds, &buf, &keys, &keysBuf)
	for i, cmd := range cmds {
		if cmd.TreeOrder != i+1 {
			t.Errorf("cmds[%d].TreeOrder = %d, want %d", i, cmd.TreeOrder, i+1)
		}
	}
}

func TestRadixSort_Reverse(t *testing.T) {
	cmds := []RenderCommand{
		{TreeOrder: 3},
		{TreeOrder: 2},
		{TreeOrder: 1},
	}
	var buf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmds, &buf, &keys, &keysBuf)
	for i, cmd := range cmds {
		if cmd.TreeOrder != i+1 {
			t.Errorf("cmds[%d].TreeOrder = %d, want %d", i, cmd.TreeOrder, i+1)
		}
	}
}

func TestRadixSort_SingleElement(t *testing.T) {
	cmds := []RenderCommand{{TreeOrder: 42}}
	var buf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmds, &buf, &keys, &keysBuf)
	if cmds[0].TreeOrder != 42 {
		t.Errorf("single element should remain unchanged")
	}
}

func TestRadixSort_MultiLayer(t *testing.T) {
	cmds := []RenderCommand{
		{RenderLayer: 2, TreeOrder: 1},
		{RenderLayer: 0, TreeOrder: 2},
		{RenderLayer: 1, TreeOrder: 3},
	}
	var buf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmds, &buf, &keys, &keysBuf)
	if cmds[0].RenderLayer != 0 || cmds[1].RenderLayer != 1 || cmds[2].RenderLayer != 2 {
		t.Errorf("layers out of order: %d, %d, %d", cmds[0].RenderLayer, cmds[1].RenderLayer, cmds[2].RenderLayer)
	}
}

func TestRadixSort_NegativeGlobalOrder(t *testing.T) {
	cmds := []RenderCommand{
		{GlobalOrder: 5, TreeOrder: 1},
		{GlobalOrder: -3, TreeOrder: 2},
		{GlobalOrder: 0, TreeOrder: 3},
	}
	var buf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmds, &buf, &keys, &keysBuf)
	if cmds[0].GlobalOrder != -3 || cmds[1].GlobalOrder != 0 || cmds[2].GlobalOrder != 5 {
		t.Errorf("GlobalOrder not sorted correctly: %d, %d, %d", cmds[0].GlobalOrder, cmds[1].GlobalOrder, cmds[2].GlobalOrder)
	}
}

// TestRadixSort_MatchesMergeSort verifies both sort algorithms produce the same ordering.
func TestRadixSort_MatchesMergeSort(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	n := 200
	cmdsM := make([]RenderCommand, n)
	cmdsR := make([]RenderCommand, n)
	for i := range cmdsM {
		cmdsM[i] = RenderCommand{
			RenderLayer: uint8(rng.Intn(4)),
			GlobalOrder: rng.Intn(20) - 10,
			TreeOrder:   rng.Intn(200),
		}
		cmdsR[i] = cmdsM[i]
	}

	var mBuf []RenderCommand
	MergeSort(cmdsM, &mBuf)

	var rBuf []RenderCommand
	var keys, keysBuf []sortEntry
	RadixSort(cmdsR, &rBuf, &keys, &keysBuf)

	for i := range cmdsM {
		if cmdsM[i].RenderLayer != cmdsR[i].RenderLayer ||
			cmdsM[i].GlobalOrder != cmdsR[i].GlobalOrder ||
			cmdsM[i].TreeOrder != cmdsR[i].TreeOrder {
			t.Errorf("mismatch at [%d]: merge={L:%d G:%d T:%d} radix={L:%d G:%d T:%d}",
				i,
				cmdsM[i].RenderLayer, cmdsM[i].GlobalOrder, cmdsM[i].TreeOrder,
				cmdsR[i].RenderLayer, cmdsR[i].GlobalOrder, cmdsR[i].TreeOrder,
			)
		}
	}
}

// --- packSortKey tests ---

func TestPackSortKey_LayerOrdering(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 0}
	b := RenderCommand{RenderLayer: 1, GlobalOrder: 0, TreeOrder: 0}
	ka := packSortKey(&a)
	kb := packSortKey(&b)
	if ka >= kb {
		t.Errorf("layer 0 key (%d) should be less than layer 1 key (%d)", ka, kb)
	}
}

func TestPackSortKey_NegativeGlobalOrder(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: -5, TreeOrder: 0}
	b := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 0}
	c := RenderCommand{RenderLayer: 0, GlobalOrder: 5, TreeOrder: 0}
	ka := packSortKey(&a)
	kb := packSortKey(&b)
	kc := packSortKey(&c)
	if ka >= kb {
		t.Errorf("negative GlobalOrder key (%d) should be less than zero key (%d)", ka, kb)
	}
	if kb >= kc {
		t.Errorf("zero GlobalOrder key (%d) should be less than positive key (%d)", kb, kc)
	}
}

func TestPackSortKey_NegativeTreeOrder(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: -1}
	b := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 0}
	ka := packSortKey(&a)
	kb := packSortKey(&b)
	if ka >= kb {
		t.Errorf("negative TreeOrder key (%d) should be less than zero key (%d)", ka, kb)
	}
}

func TestPackSortKey_ConsistentWithCommandLessOrEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b RenderCommand
	}{
		{"same layer, different global order",
			RenderCommand{RenderLayer: 0, GlobalOrder: -2, TreeOrder: 10},
			RenderCommand{RenderLayer: 0, GlobalOrder: 3, TreeOrder: 1}},
		{"different layers",
			RenderCommand{RenderLayer: 0, GlobalOrder: 100, TreeOrder: 100},
			RenderCommand{RenderLayer: 1, GlobalOrder: 0, TreeOrder: 0}},
		{"same layer and global, different tree order",
			RenderCommand{RenderLayer: 1, GlobalOrder: 5, TreeOrder: 2},
			RenderCommand{RenderLayer: 1, GlobalOrder: 5, TreeOrder: 7}},
	}
	for _, tc := range cases {
		lessOrEq := CommandLessOrEqual(tc.a, tc.b)
		ka := packSortKey(&tc.a)
		kb := packSortKey(&tc.b)
		if lessOrEq && ka > kb {
			t.Errorf("%s: CommandLessOrEqual says a<=b but key(a)=%d > key(b)=%d", tc.name, ka, kb)
		}
		if !lessOrEq && ka <= kb {
			t.Errorf("%s: CommandLessOrEqual says a>b but key(a)=%d <= key(b)=%d", tc.name, ka, kb)
		}
	}
}

// --- CommandBatchKey tests ---

func TestCommandBatchKey_ExtractsFields(t *testing.T) {
	cmd := RenderCommand{
		TargetID:  3,
		ShaderID:  7,
		BlendMode: types.BlendAdd,
		TextureRegion: types.TextureRegion{
			Page: 42,
		},
	}
	key := CommandBatchKey(&cmd)
	if key.TargetID != 3 {
		t.Errorf("TargetID = %d, want 3", key.TargetID)
	}
	if key.ShaderID != 7 {
		t.Errorf("ShaderID = %d, want 7", key.ShaderID)
	}
	if key.Blend != types.BlendAdd {
		t.Errorf("Blend = %v, want BlendAdd", key.Blend)
	}
	if key.Page != 42 {
		t.Errorf("Page = %d, want 42", key.Page)
	}
}

func TestCommandBatchKey_DifferentBlendModes(t *testing.T) {
	a := RenderCommand{BlendMode: types.BlendNormal}
	b := RenderCommand{BlendMode: types.BlendAdd}
	if CommandBatchKey(&a) == CommandBatchKey(&b) {
		t.Error("different blend modes should produce different batch keys")
	}
}

// --- MultiplyAffine32 tests ---

func TestMultiplyAffine32_Translation(t *testing.T) {
	a := [6]float32{1, 0, 0, 1, 10, 20}
	b := [6]float32{1, 0, 0, 1, 5, 7}
	result := MultiplyAffine32(a, b)
	if result[4] != 15 || result[5] != 27 {
		t.Errorf("translation = (%f, %f), want (15, 27)", result[4], result[5])
	}
}

func TestMultiplyAffine32_ScaleAndTranslation(t *testing.T) {
	parent := [6]float32{2, 0, 0, 2, 0, 0}
	child := [6]float32{1, 0, 0, 1, 5, 10}
	result := MultiplyAffine32(parent, child)
	if result[0] != 2 || result[3] != 2 {
		t.Errorf("scale should remain 2, got (%f, %f)", result[0], result[3])
	}
	if result[4] != 10 || result[5] != 20 {
		t.Errorf("translation = (%f, %f), want (10, 20)", result[4], result[5])
	}
}

// --- InvertAffine32 with translation ---

func TestInvertAffine32_Translation(t *testing.T) {
	m := [6]float32{1, 0, 0, 1, 10, 20}
	inv := InvertAffine32(m)
	result := MultiplyAffine32(m, inv)
	for i := range IdentityTransform32 {
		diff := result[i] - IdentityTransform32[i]
		if diff > 1e-5 || diff < -1e-5 {
			t.Errorf("m*inv[%d] = %f, want %f", i, result[i], IdentityTransform32[i])
		}
	}
}

func TestInvertAffine32_SingularMatrix(t *testing.T) {
	m := [6]float32{0, 0, 0, 0, 5, 10}
	inv := InvertAffine32(m)
	for i := range IdentityTransform32 {
		if inv[i] != IdentityTransform32[i] {
			t.Errorf("singular inv[%d] = %f, want identity %f", i, inv[i], IdentityTransform32[i])
		}
	}
}

// --- Pipeline.Traverse tests ---

func TestTraverse_SimpleSprites(t *testing.T) {
	oldCull := ShouldCullFn
	defer func() { ShouldCullFn = oldCull }()
	ShouldCullFn = nil

	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = true
	root.WorldTransform = node.IdentityTransform
	root.WorldAlpha = 1.0

	s1 := node.NewNode("s1", types.NodeTypeSprite)
	s1.Visible_ = true
	s1.Renderable_ = true
	s1.WorldTransform = node.IdentityTransform
	s1.WorldAlpha = 1.0
	s1.TextureRegion_ = types.TextureRegion{Width: 16, Height: 16, OriginalW: 16, OriginalH: 16, Page: types.MagentaPlaceholderPage}

	s2 := node.NewNode("s2", types.NodeTypeSprite)
	s2.Visible_ = true
	s2.Renderable_ = true
	s2.WorldTransform = node.IdentityTransform
	s2.WorldAlpha = 1.0
	s2.TextureRegion_ = types.TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32, Page: types.MagentaPlaceholderPage}

	root.AddChild(s1)
	root.AddChild(s2)

	p := &Pipeline{
		ViewTransform: node.IdentityTransform,
	}
	treeOrder := 0
	p.Traverse(root, &treeOrder)

	if len(p.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(p.Commands))
	}
	if p.Commands[0].TreeOrder != 1 {
		t.Errorf("first command TreeOrder = %d, want 1", p.Commands[0].TreeOrder)
	}
	if p.Commands[1].TreeOrder != 2 {
		t.Errorf("second command TreeOrder = %d, want 2", p.Commands[1].TreeOrder)
	}
	if p.Commands[0].Type != CommandSprite || p.Commands[1].Type != CommandSprite {
		t.Error("both commands should be CommandSprite")
	}
}

func TestTraverse_InvisibleNodeSkipped(t *testing.T) {
	oldCull := ShouldCullFn
	defer func() { ShouldCullFn = oldCull }()
	ShouldCullFn = nil

	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = true
	root.WorldTransform = node.IdentityTransform

	s1 := node.NewNode("hidden", types.NodeTypeSprite)
	s1.Visible_ = false
	s1.Renderable_ = true
	s1.WorldTransform = node.IdentityTransform
	s1.TextureRegion_ = types.TextureRegion{Width: 16, Height: 16, OriginalW: 16, OriginalH: 16}

	root.AddChild(s1)

	p := &Pipeline{ViewTransform: node.IdentityTransform}
	treeOrder := 0
	p.Traverse(root, &treeOrder)

	if len(p.Commands) != 0 {
		t.Errorf("expected 0 commands for invisible node, got %d", len(p.Commands))
	}
}

func TestTraverse_ZIndexOrdering(t *testing.T) {
	oldCull := ShouldCullFn
	defer func() { ShouldCullFn = oldCull }()
	ShouldCullFn = nil

	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = true
	root.WorldTransform = node.IdentityTransform
	root.WorldAlpha = 1.0

	s1 := node.NewNode("s1", types.NodeTypeSprite)
	s1.Visible_ = true
	s1.Renderable_ = true
	s1.WorldTransform = node.IdentityTransform
	s1.WorldAlpha = 1.0
	s1.ZIndex_ = 10
	s1.TextureRegion_ = types.TextureRegion{Width: 8, Height: 8, OriginalW: 8, OriginalH: 8, Page: types.MagentaPlaceholderPage}

	s2 := node.NewNode("s2", types.NodeTypeSprite)
	s2.Visible_ = true
	s2.Renderable_ = true
	s2.WorldTransform = node.IdentityTransform
	s2.WorldAlpha = 1.0
	s2.ZIndex_ = 1
	s2.TextureRegion_ = types.TextureRegion{Width: 8, Height: 8, OriginalW: 8, OriginalH: 8, Page: types.MagentaPlaceholderPage}

	root.AddChild(s1)
	root.AddChild(s2)

	p := &Pipeline{ViewTransform: node.IdentityTransform}
	treeOrder := 0
	p.Traverse(root, &treeOrder)

	if len(p.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(p.Commands))
	}
	// s2 has lower ZIndex so should be traversed first and get lower TreeOrder.
	if p.Commands[0].TreeOrder >= p.Commands[1].TreeOrder {
		t.Error("first command should have lower TreeOrder than second (ZIndex ordering)")
	}
}

func TestTraverse_NestedChildren(t *testing.T) {
	oldCull := ShouldCullFn
	defer func() { ShouldCullFn = oldCull }()
	ShouldCullFn = nil

	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = true
	root.WorldTransform = node.IdentityTransform
	root.WorldAlpha = 1.0

	container := node.NewNode("container", types.NodeTypeContainer)
	container.Visible_ = true
	container.WorldTransform = node.IdentityTransform
	container.WorldAlpha = 1.0

	s1 := node.NewNode("s1", types.NodeTypeSprite)
	s1.Visible_ = true
	s1.Renderable_ = true
	s1.WorldTransform = node.IdentityTransform
	s1.WorldAlpha = 1.0
	s1.TextureRegion_ = types.TextureRegion{Width: 8, Height: 8, OriginalW: 8, OriginalH: 8}

	container.AddChild(s1)
	root.AddChild(container)

	p := &Pipeline{ViewTransform: node.IdentityTransform}
	treeOrder := 0
	p.Traverse(root, &treeOrder)

	if len(p.Commands) != 1 {
		t.Fatalf("expected 1 command from nested sprite, got %d", len(p.Commands))
	}
	if p.Commands[0].Type != CommandSprite {
		t.Error("command should be CommandSprite")
	}
}

// --- ApplyFiltersAny tests ---

func TestApplyFiltersAny_EmptyReturnsSource(t *testing.T) {
	src := ebiten.NewImage(10, 10)
	var pool RenderTexturePool
	result := ApplyFiltersAny(nil, src, &pool)
	if result != src {
		t.Error("nil filter list should return src unchanged")
	}
	result = ApplyFiltersAny([]any{}, src, &pool)
	if result != src {
		t.Error("empty filter list should return src unchanged")
	}
}

func TestApplyFiltersAny_NonFilterSkipped(t *testing.T) {
	src := ebiten.NewImage(10, 10)
	var pool RenderTexturePool
	result := ApplyFiltersAny([]any{"not a filter", 42}, src, &pool)
	if result != src {
		t.Error("non-filter entries should be skipped and return src")
	}
}

// --- NodeDimensions tests ---

func TestNodeDimensions_Sprite(t *testing.T) {
	n := node.NewNode("sp", types.NodeTypeSprite)
	n.TextureRegion_ = types.TextureRegion{OriginalW: 64, OriginalH: 48}
	w, h := NodeDimensions(n)
	if w != 64 || h != 48 {
		t.Errorf("sprite dimensions = (%f, %f), want (64, 48)", w, h)
	}
}

func TestNodeDimensions_SpriteCustomImage(t *testing.T) {
	n := node.NewNode("sp", types.NodeTypeSprite)
	img := ebiten.NewImage(100, 75)
	n.CustomImage_ = img
	w, h := NodeDimensions(n)
	if w != 100 || h != 75 {
		t.Errorf("custom image dimensions = (%f, %f), want (100, 75)", w, h)
	}
}

func TestNodeDimensions_Container(t *testing.T) {
	n := node.NewNode("c", types.NodeTypeContainer)
	w, h := NodeDimensions(n)
	if w != 0 || h != 0 {
		t.Errorf("container dimensions = (%f, %f), want (0, 0)", w, h)
	}
}

// --- RebuildSortedChildren additional tests ---

func TestRebuildSortedChildren_AlreadySorted(t *testing.T) {
	parent := node.NewNode("p", types.NodeTypeContainer)
	c1 := node.NewNode("c1", types.NodeTypeSprite)
	c1.ZIndex_ = 1
	c2 := node.NewNode("c2", types.NodeTypeSprite)
	c2.ZIndex_ = 2
	c3 := node.NewNode("c3", types.NodeTypeSprite)
	c3.ZIndex_ = 3
	parent.Children_ = []*node.Node{c1, c2, c3}
	parent.ChildrenSorted = false

	RebuildSortedChildren(parent)

	if parent.SortedChildren[0] != c1 || parent.SortedChildren[1] != c2 || parent.SortedChildren[2] != c3 {
		t.Error("already sorted children should remain in order")
	}
}

func TestRebuildSortedChildren_Empty(t *testing.T) {
	parent := node.NewNode("p", types.NodeTypeContainer)
	parent.ChildrenSorted = false
	RebuildSortedChildren(parent)
	if len(parent.SortedChildren) != 0 {
		t.Errorf("empty children should produce empty sorted list, got %d", len(parent.SortedChildren))
	}
	if !parent.ChildrenSorted {
		t.Error("ChildrenSorted should be true after rebuild")
	}
}

func TestRebuildSortedChildren_EqualZIndex(t *testing.T) {
	parent := node.NewNode("p", types.NodeTypeContainer)
	c1 := node.NewNode("c1", types.NodeTypeSprite)
	c1.ZIndex_ = 5
	c2 := node.NewNode("c2", types.NodeTypeSprite)
	c2.ZIndex_ = 5
	c3 := node.NewNode("c3", types.NodeTypeSprite)
	c3.ZIndex_ = 5
	parent.Children_ = []*node.Node{c1, c2, c3}
	parent.ChildrenSorted = false

	RebuildSortedChildren(parent)

	if parent.SortedChildren[0] != c1 || parent.SortedChildren[1] != c2 || parent.SortedChildren[2] != c3 {
		t.Error("equal ZIndex children should maintain original order (stable sort)")
	}
}

// --- Cache-as-tree operation tests ---

func TestSetCacheAsTree_Enable(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true)
	ctd := GetCacheTreeData(n)
	if ctd == nil {
		t.Fatal("CacheTreeData should be allocated after enable")
	}
	if !ctd.Dirty {
		t.Error("newly enabled cache should be dirty")
	}
	if ctd.Mode != types.CacheTreeAuto {
		t.Errorf("default mode should be CacheTreeAuto, got %d", ctd.Mode)
	}
}

func TestSetCacheAsTree_EnableWithMode(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true, types.CacheTreeManual)
	ctd := GetCacheTreeData(n)
	if ctd == nil {
		t.Fatal("CacheTreeData should be allocated")
	}
	if ctd.Mode != types.CacheTreeManual {
		t.Errorf("mode should be CacheTreeManual, got %d", ctd.Mode)
	}
}

func TestSetCacheAsTree_Disable(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true)
	SetCacheAsTree(n, false)
	if GetCacheTreeData(n) != nil {
		t.Error("CacheTreeData should be nil after disable")
	}
}

func TestInvalidateCacheTree(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true)
	ctd := GetCacheTreeData(n)
	ctd.Dirty = false

	InvalidateCacheTree(n)

	if !ctd.Dirty {
		t.Error("InvalidateCacheTree should mark dirty = true")
	}
}

func TestInvalidateCacheTree_NoCache(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	// Should not panic when no cache data exists.
	InvalidateCacheTree(n)
}

func TestInvalidateAncestorCache_SelfAuto(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true, types.CacheTreeAuto)
	ctd := GetCacheTreeData(n)
	ctd.Dirty = false

	InvalidateAncestorCache(n)

	if !ctd.Dirty {
		t.Error("InvalidateAncestorCache should mark auto-mode self dirty")
	}
}

func TestInvalidateAncestorCache_ManualNotDirtied(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeContainer)
	SetCacheAsTree(n, true, types.CacheTreeManual)
	ctd := GetCacheTreeData(n)
	ctd.Dirty = false

	InvalidateAncestorCache(n)

	if ctd.Dirty {
		t.Error("InvalidateAncestorCache should not dirty manual-mode cache")
	}
}

func TestInvalidateAncestorCache_ParentDirtied(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	SetCacheAsTree(parent, true, types.CacheTreeAuto)
	ctd := GetCacheTreeData(parent)
	ctd.Dirty = false

	child := node.NewNode("child", types.NodeTypeSprite)
	parent.AddChild(child)

	InvalidateAncestorCache(child)

	if !ctd.Dirty {
		t.Error("InvalidateAncestorCache should dirty parent's auto-mode cache")
	}
}

// --- WorldAABB additional tests ---

func TestWorldAABB_Translated(t *testing.T) {
	t32 := [6]float64{1, 0, 0, 1, 50, 100}
	aabb := types.WorldAABB(t32, 10, 20)
	if aabb.X != 50 || aabb.Y != 100 || aabb.Width != 10 || aabb.Height != 20 {
		t.Errorf("WorldAABB translated = %+v, want (50,100,10,20)", aabb)
	}
}

func TestWorldAABB_Scaled(t *testing.T) {
	t32 := [6]float64{2, 0, 0, 3, 0, 0}
	aabb := types.WorldAABB(t32, 10, 10)
	if aabb.Width != 20 || aabb.Height != 30 {
		t.Errorf("WorldAABB scaled = (%f, %f), want (20, 30)", aabb.Width, aabb.Height)
	}
}

// --- TransformRect tests ---

func TestTransformRect_Identity(t *testing.T) {
	id := [6]float64{1, 0, 0, 1, 0, 0}
	r := types.Rect{X: 10, Y: 20, Width: 30, Height: 40}
	result := TransformRect(id, r)
	if result.X != 10 || result.Y != 20 || result.Width != 30 || result.Height != 40 {
		t.Errorf("identity TransformRect = %+v, want original", result)
	}
}

func TestTransformRect_Translated(t *testing.T) {
	tr := [6]float64{1, 0, 0, 1, 5, 10}
	r := types.Rect{X: 0, Y: 0, Width: 10, Height: 10}
	result := TransformRect(tr, r)
	if result.X != 5 || result.Y != 10 || result.Width != 10 || result.Height != 10 {
		t.Errorf("translated TransformRect = %+v, want (5,10,10,10)", result)
	}
}

// --- RectUnion additional tests ---

func TestRectUnion_NoOverlap(t *testing.T) {
	a := types.Rect{X: 0, Y: 0, Width: 5, Height: 5}
	b := types.Rect{X: 10, Y: 10, Width: 5, Height: 5}
	u := RectUnion(a, b)
	if u.X != 0 || u.Y != 0 || u.Width != 15 || u.Height != 15 {
		t.Errorf("RectUnion no-overlap = %+v, want (0,0,15,15)", u)
	}
}

func TestRectUnion_OneContainsOther(t *testing.T) {
	a := types.Rect{X: 0, Y: 0, Width: 100, Height: 100}
	b := types.Rect{X: 10, Y: 10, Width: 5, Height: 5}
	u := RectUnion(a, b)
	if u.X != 0 || u.Y != 0 || u.Width != 100 || u.Height != 100 {
		t.Errorf("RectUnion contain = %+v, want (0,0,100,100)", u)
	}
}

// --- AppendSpriteQuad tests ---

func TestAppendSpriteQuad_VertexIndexGrowth(t *testing.T) {
	p := &Pipeline{}
	cmd := RenderCommand{
		Type:      CommandSprite,
		Transform: IdentityTransform32,
		TextureRegion: types.TextureRegion{
			X: 0, Y: 0, Width: 16, Height: 16,
			OriginalW: 16, OriginalH: 16,
		},
		Color: Color32{1, 1, 1, 1},
	}
	p.AppendSpriteQuad(&cmd)
	if len(p.BatchVerts) != 4 {
		t.Errorf("after 1 quad: verts = %d, want 4", len(p.BatchVerts))
	}
	if len(p.BatchInds) != 6 {
		t.Errorf("after 1 quad: inds = %d, want 6", len(p.BatchInds))
	}

	p.AppendSpriteQuad(&cmd)
	if len(p.BatchVerts) != 8 {
		t.Errorf("after 2 quads: verts = %d, want 8", len(p.BatchVerts))
	}
	if len(p.BatchInds) != 12 {
		t.Errorf("after 2 quads: inds = %d, want 12", len(p.BatchInds))
	}

	if p.BatchInds[6] != 4 || p.BatchInds[7] != 5 || p.BatchInds[8] != 6 {
		t.Errorf("second quad indices[0:3] = (%d,%d,%d), want (4,5,6)",
			p.BatchInds[6], p.BatchInds[7], p.BatchInds[8])
	}
}

func TestAppendSpriteQuad_VertexPositions(t *testing.T) {
	p := &Pipeline{}
	cmd := RenderCommand{
		Type:      CommandSprite,
		Transform: [6]float32{1, 0, 0, 1, 10, 20},
		TextureRegion: types.TextureRegion{
			X: 0, Y: 0, Width: 16, Height: 16,
			OriginalW: 16, OriginalH: 16,
		},
		Color: Color32{1, 1, 1, 1},
	}
	p.AppendSpriteQuad(&cmd)

	if p.BatchVerts[0].DstX != 10 || p.BatchVerts[0].DstY != 20 {
		t.Errorf("top-left = (%f, %f), want (10, 20)", p.BatchVerts[0].DstX, p.BatchVerts[0].DstY)
	}
	if p.BatchVerts[3].DstX != 26 || p.BatchVerts[3].DstY != 36 {
		t.Errorf("bottom-right = (%f, %f), want (26, 36)", p.BatchVerts[3].DstX, p.BatchVerts[3].DstY)
	}
}

// --- Pipeline.Sort tests ---

func TestPipelineSort(t *testing.T) {
	p := &Pipeline{
		Commands: []RenderCommand{
			{RenderLayer: 1, TreeOrder: 1},
			{RenderLayer: 0, TreeOrder: 3},
			{RenderLayer: 0, TreeOrder: 1},
		},
	}
	p.Sort()
	if p.Commands[0].RenderLayer != 0 || p.Commands[1].RenderLayer != 0 || p.Commands[2].RenderLayer != 1 {
		t.Error("Pipeline.Sort did not sort correctly by RenderLayer")
	}
}

// --- Pipeline.ReleaseDeferred tests ---

func TestPipelineReleaseDeferred(t *testing.T) {
	p := &Pipeline{}
	img1 := p.RtPool.Acquire(32, 32)
	img2 := p.RtPool.Acquire(32, 32)
	p.RtDeferred = append(p.RtDeferred, img1, img2)

	p.ReleaseDeferred()

	if len(p.RtDeferred) != 0 {
		t.Errorf("RtDeferred should be empty after release, got %d", len(p.RtDeferred))
	}

	reused := p.RtPool.Acquire(32, 32)
	if reused != img1 && reused != img2 {
		t.Error("released images should be reused from pool")
	}
}

// --- DefaultFXAAConfig test ---

func TestDefaultFXAAConfig(t *testing.T) {
	cfg := DefaultFXAAConfig()
	if cfg.SubpixQuality != 0.75 {
		t.Errorf("SubpixQuality = %f, want 0.75", cfg.SubpixQuality)
	}
	if cfg.EdgeThreshold != 0.166 {
		t.Errorf("EdgeThreshold = %f, want 0.166", cfg.EdgeThreshold)
	}
	if cfg.EdgeThresholdMin != 0.0312 {
		t.Errorf("EdgeThresholdMin = %f, want 0.0312", cfg.EdgeThresholdMin)
	}
}

// --- InitShaders test ---

func TestInitShaders_NoPanic(t *testing.T) {
	InitShaders()
}

func TestEnsureSDFShader_ReturnsNonNil(t *testing.T) {
	s := EnsureSDFShader()
	if s == nil {
		t.Error("EnsureSDFShader should return a non-nil shader")
	}
}

func TestEnsureMSDFShader_ReturnsNonNil(t *testing.T) {
	s := EnsureMSDFShader()
	if s == nil {
		t.Error("EnsureMSDFShader should return a non-nil shader")
	}
}

func TestEnsureFXAAShader_ReturnsNonNil(t *testing.T) {
	s := EnsureFXAAShader()
	if s == nil {
		t.Error("EnsureFXAAShader should return a non-nil shader")
	}
}

// --- CommandLessOrEqual additional edge cases ---

func TestCommandLessOrEqual_GlobalOrderPriority(t *testing.T) {
	a := RenderCommand{RenderLayer: 0, GlobalOrder: -5, TreeOrder: 100}
	b := RenderCommand{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 0}
	if !CommandLessOrEqual(a, b) {
		t.Error("negative GlobalOrder should sort before zero")
	}
	if CommandLessOrEqual(b, a) {
		t.Error("zero GlobalOrder should not sort before negative")
	}
}

// --- Affine32 round-trip ---

func TestAffine32_RoundTrip(t *testing.T) {
	m := [6]float64{1.5, 0.25, -0.25, 1.5, 100.5, 200.75}
	f := Affine32(m)
	for i := range m {
		diff := math.Abs(float64(f[i]) - m[i])
		if diff > 1e-5 {
			t.Errorf("Affine32[%d] = %f, want %f (diff %e)", i, float64(f[i]), m[i], diff)
		}
	}
}

// --- ColorToRGBA test ---

func TestColorToRGBA(t *testing.T) {
	c := types.RGBA(1.0, 0.5, 0.0, 1.0)
	rgba := ColorToRGBA(c)
	if rgba.A != 255 {
		t.Errorf("alpha = %d, want 255", rgba.A)
	}
	if rgba.R != 255 {
		t.Errorf("red = %d, want 255", rgba.R)
	}
	if rgba.G < 127 || rgba.G > 128 {
		t.Errorf("green = %d, want ~127", rgba.G)
	}
	if rgba.B != 0 {
		t.Errorf("blue = %d, want 0", rgba.B)
	}
}

// --- MergeSort stability test ---

func TestMergeSort_Stability(t *testing.T) {
	cmds := []RenderCommand{
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1, ShaderID: 10},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1, ShaderID: 20},
		{RenderLayer: 0, GlobalOrder: 0, TreeOrder: 1, ShaderID: 30},
	}
	var buf []RenderCommand
	MergeSort(cmds, &buf)
	if cmds[0].ShaderID != 10 || cmds[1].ShaderID != 20 || cmds[2].ShaderID != 30 {
		t.Error("MergeSort should be stable for equal keys")
	}
}

// --- RenderTexture tests ---

func TestNewRenderTexture(t *testing.T) {
	rt := NewRenderTexture(200, 150)
	if rt.Width() != 200 || rt.Height() != 150 {
		t.Errorf("RenderTexture dimensions = (%d, %d), want (200, 150)", rt.Width(), rt.Height())
	}
	if rt.Image() == nil {
		t.Error("RenderTexture image should not be nil")
	}
}

func TestRenderTexture_Resize(t *testing.T) {
	rt := NewRenderTexture(100, 100)
	rt.Resize(200, 300)
	if rt.Width() != 200 || rt.Height() != 300 {
		t.Errorf("after resize: (%d, %d), want (200, 300)", rt.Width(), rt.Height())
	}
}

func TestRenderTexture_Dispose(t *testing.T) {
	rt := NewRenderTexture(100, 100)
	rt.Dispose()
	if rt.Img != nil {
		t.Error("Dispose should nil the image")
	}
}

// --- RegisterAnimatedInCache test ---

func TestRegisterAnimatedInCache(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	SetCacheAsTree(parent, true)
	ctd := GetCacheTreeData(parent)

	child := node.NewNode("child", types.NodeTypeSprite)
	child.Renderable_ = true
	parent.AddChild(child)

	// Simulate a cached command from the child.
	ctd.Commands = []CachedCmd{
		{Cmd: RenderCommand{}, SourceNodeID: child.ID},
	}
	ctd.Dirty = false

	RegisterAnimatedInCache(child)

	if ctd.Commands[0].Source != child {
		t.Error("RegisterAnimatedInCache should set Source to the child node")
	}
}

// --- ResolvePage test ---

func TestResolvePage_MagentaSentinel(t *testing.T) {
	oldFn := MagentaImageFn
	defer func() { MagentaImageFn = oldFn }()

	sentinel := ebiten.NewImage(2, 2)
	MagentaImageFn = func() *ebiten.Image { return sentinel }

	result := ResolvePage(types.MagentaPlaceholderPage)
	if result != sentinel {
		t.Error("ResolvePage with magenta sentinel should return magenta image")
	}
}

func TestResolvePage_NilFunctions(t *testing.T) {
	oldPage := PageFn
	oldMagenta := MagentaImageFn
	defer func() {
		PageFn = oldPage
		MagentaImageFn = oldMagenta
	}()
	PageFn = nil
	MagentaImageFn = nil

	result := ResolvePage(0)
	if result != nil {
		t.Error("ResolvePage with nil PageFn should return nil")
	}
	result = ResolvePage(types.MagentaPlaceholderPage)
	if result != nil {
		t.Error("ResolvePage magenta with nil MagentaImageFn should return nil")
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
