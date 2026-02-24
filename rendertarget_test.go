package willow

import "testing"

// --- nextPowerOfTwo ---

func TestNextPowerOfTwo(t *testing.T) {
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
		got := nextPowerOfTwo(tt.input)
		if got != tt.want {
			t.Errorf("nextPowerOfTwo(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- Pool ---

func TestPoolAcquireReturnsPow2(t *testing.T) {
	var pool renderTexturePool
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

func TestPoolReleaseAndReacquire(t *testing.T) {
	var pool renderTexturePool
	img1 := pool.Acquire(64, 64)
	pool.Release(img1)

	img2 := pool.Acquire(64, 64)
	if img1 != img2 {
		t.Error("expected pool to return the same image after release")
	}
	pool.Release(img2)
}

func TestPoolDifferentSizes(t *testing.T) {
	var pool renderTexturePool
	a := pool.Acquire(32, 32)
	b := pool.Acquire(64, 64)
	if a == b {
		t.Error("different sizes should return different images")
	}
	pool.Release(a)
	pool.Release(b)
}

func TestPoolReleaseNilNoPanic(t *testing.T) {
	var pool renderTexturePool
	pool.Release(nil) // should not panic
}

// --- CacheAsTexture ---

func TestSetCacheAsTextureEnabled(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.SetCacheAsTexture(true)
	if !n.cacheEnabled {
		t.Error("cacheEnabled should be true")
	}
	if !n.cacheDirty {
		t.Error("cacheDirty should be true after enabling")
	}
}

func TestSetCacheAsTextureDisabled(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.SetCacheAsTexture(true)
	n.SetCacheAsTexture(false)
	if n.cacheEnabled {
		t.Error("cacheEnabled should be false after disabling")
	}
	if n.cacheDirty {
		t.Error("cacheDirty should be false after disabling")
	}
	if n.cacheTexture != nil {
		t.Error("cacheTexture should be nil after disabling")
	}
}

func TestSetCacheAsTextureIdempotent(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.SetCacheAsTexture(true)
	n.cacheDirty = false
	n.SetCacheAsTexture(true) // should be no-op
	if n.cacheDirty {
		t.Error("setting cache to same value should not reset dirty flag")
	}
}

func TestInvalidateCache(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.SetCacheAsTexture(true)
	n.cacheDirty = false
	n.InvalidateCache()
	if !n.cacheDirty {
		t.Error("cacheDirty should be true after InvalidateCache")
	}
}

func TestInvalidateCacheNoCacheNoOp(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	n.InvalidateCache() // cache not enabled  -  should be no-op
	if n.cacheDirty {
		t.Error("InvalidateCache on non-cached node should not set dirty")
	}
}

func TestIsCacheEnabled(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 32, Height: 32})
	if n.IsCacheEnabled() {
		t.Error("should be false by default")
	}
	n.SetCacheAsTexture(true)
	if !n.IsCacheEnabled() {
		t.Error("should be true after enabling")
	}
}

// --- Subtree bounds ---

func TestSubtreeBoundsSingleSprite(t *testing.T) {
	n := NewSprite("s", TextureRegion{Width: 40, Height: 30, OriginalW: 40, OriginalH: 30})
	b := subtreeBounds(n)
	if b.Width != 40 || b.Height != 30 {
		t.Errorf("bounds = %v, want 40x30", b)
	}
}

func TestSubtreeBoundsWithChildren(t *testing.T) {
	parent := NewContainer("parent")
	child := NewSprite("child", TextureRegion{Width: 20, Height: 20, OriginalW: 20, OriginalH: 20})
	child.X = 50
	child.Y = 50
	parent.AddChild(child)

	b := subtreeBounds(parent)
	// Child at (50,50) with size 20x20 → bounds should be [50,50,20,20]
	if b.X != 50 || b.Y != 50 || b.Width != 20 || b.Height != 20 {
		t.Errorf("bounds = %v, want {50 50 20 20}", b)
	}
}

func TestSubtreeBoundsMultipleChildren(t *testing.T) {
	parent := NewContainer("parent")
	a := NewSprite("a", TextureRegion{Width: 10, Height: 10, OriginalW: 10, OriginalH: 10})
	b := NewSprite("b", TextureRegion{Width: 10, Height: 10, OriginalW: 10, OriginalH: 10})
	b.X = 100
	b.Y = 100
	parent.AddChild(a)
	parent.AddChild(b)

	bounds := subtreeBounds(parent)
	// a at (0,0)+10x10, b at (100,100)+10x10 → union [0,0,110,110]
	if bounds.X != 0 || bounds.Y != 0 {
		t.Errorf("origin = (%v, %v), want (0, 0)", bounds.X, bounds.Y)
	}
	if bounds.Width != 110 || bounds.Height != 110 {
		t.Errorf("size = (%v, %v), want (110, 110)", bounds.Width, bounds.Height)
	}
}

func TestSubtreeBoundsEmptyContainer(t *testing.T) {
	c := NewContainer("empty")
	b := subtreeBounds(c)
	// No renderable content → zero bounds.
	if b.Width != 0 || b.Height != 0 {
		t.Errorf("empty container bounds = %v, want zero", b)
	}
}

// --- rectUnion ---

func TestRectUnion(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	b := Rect{X: 5, Y: 5, Width: 10, Height: 10}
	u := rectUnion(a, b)
	if u.X != 0 || u.Y != 0 || u.Width != 15 || u.Height != 15 {
		t.Errorf("union = %v, want {0 0 15 15}", u)
	}
}

// --- Benchmarks ---

func BenchmarkPoolAcquireRelease(b *testing.B) {
	var pool renderTexturePool
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
