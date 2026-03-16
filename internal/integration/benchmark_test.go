package integration

import (
	"image"
	"math"
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/render"
	"github.com/hajimehoshi/ebiten/v2"

	. "github.com/devthicket/willow"
)

// setupBenchScene creates a Scene with n sprite nodes for benchmark use.
// Each sprite gets a magenta placeholder TextureRegion (no atlas page needed).
func setupBenchScene(n int) *Scene {
	s := NewScene()
	root := s.Root
	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     32,
		Height:    32,
		OriginalW: 32,
		OriginalH: 32,
	}
	for i := 0; i < n; i++ {
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%100) * 40
		sp.Y_ = float64(i/100) * 40
		root.AddChild(sp)
	}
	return s
}

// --- Sprite Rendering Benchmarks ---

func BenchmarkDraw_10000Sprites_Static(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)

	// Warm up: first draw populates sortBuf etc.
	s.Draw(screen)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_Rotating(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Dirty all transforms by rotating every sprite.
		for _, child := range children {
			child.Rotation_ += 0.01
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_AlphaVarying(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j, child := range children {
			child.Alpha_ = 0.5 + 0.5*math.Sin(float64(i+j)*0.001)
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_AlphaOnly(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j, child := range children {
			child.SetAlpha(0.5 + 0.5*math.Sin(float64(i+j)*0.001))
		}
		s.Draw(screen)
	}
}

// --- Sprite Rendering Benchmarks (Coalesced) ---

func BenchmarkDraw_10000Sprites_Static_Coalesced(b *testing.B) {
	s := setupBenchScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_Rotating_Coalesced(b *testing.B) {
	s := setupBenchScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, child := range children {
			child.Rotation_ += 0.01
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_AlphaVarying_Coalesced(b *testing.B) {
	s := setupBenchScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j, child := range children {
			child.Alpha_ = 0.5 + 0.5*math.Sin(float64(i+j)*0.001)
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_10000Sprites_AlphaOnly_Coalesced(b *testing.B) {
	s := setupBenchScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j, child := range children {
			child.SetAlpha(0.5 + 0.5*math.Sin(float64(i+j)*0.001))
		}
		s.Draw(screen)
	}
}

// --- 100K Sprite Rendering Benchmarks ---

func BenchmarkDraw_100000Sprites_Static(b *testing.B) {
	s := setupBenchScene(100000)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_100000Sprites_Rotating(b *testing.B) {
	s := setupBenchScene(100000)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, child := range children {
			child.Rotation_ += 0.01
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_100000Sprites_Static_Coalesced(b *testing.B) {
	s := setupBenchScene(100000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_100000Sprites_Rotating_Coalesced(b *testing.B) {
	s := setupBenchScene(100000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	children := s.Root.Children()
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, child := range children {
			child.Rotation_ += 0.01
			child.TransformDirty = true
		}
		s.Draw(screen)
	}
}

// --- CacheAsTree Benchmarks ---

// setupCacheAsTreeBenchScene creates a scene with a single container
// (CacheAsTree enabled) holding n sprites.
func setupCacheAsTreeBenchScene(n int, mode CacheTreeMode) *Scene {
	s := NewScene()
	root := s.Root
	container := NewContainer("cached")
	root.AddChild(container)
	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     32,
		Height:    32,
		OriginalW: 32,
		OriginalH: 32,
	}
	for i := 0; i < n; i++ {
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%100) * 40
		sp.Y_ = float64(i/100) * 40
		container.AddChild(sp)
	}
	container.SetCacheAsTree(true, mode)
	return s
}

func BenchmarkDraw_CacheAsTree_Manual_Static(b *testing.B) {
	s := setupCacheAsTreeBenchScene(10000, CacheTreeManual)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup: builds cache

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_CacheAsTree_Manual_CameraScroll(b *testing.B) {
	s := setupCacheAsTreeBenchScene(10000, CacheTreeManual)
	cam := s.NewCamera(Rect{X: 0, Y: 0, Width: 1280, Height: 720})
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cam.X = float64(i % 500)
		cam.Y = float64(i % 300)
		s.Draw(screen)
	}
}

func BenchmarkDraw_CacheAsTree_Manual_TextureSwap(b *testing.B) {
	s := setupCacheAsTreeBenchScene(10000, CacheTreeManual)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup: builds cache

	// Pick 100 tiles to animate (same page, different UVs).
	container := s.Root.ChildAt(0)
	animated := container.Children()[:100]
	frames := [2]TextureRegion{
		{Page: magentaPlaceholderPage, X: 0, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32},
		{Page: magentaPlaceholderPage, X: 32, Y: 0, Width: 32, Height: 32, OriginalW: 32, OriginalH: 32},
	}
	// First swap to register as animated.
	for _, tile := range animated {
		tile.SetTextureRegion(frames[1])
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		frame := frames[i%2]
		for _, tile := range animated {
			tile.SetTextureRegion(frame)
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_CacheAsTree_Auto_1PctMoving(b *testing.B) {
	s := setupCacheAsTreeBenchScene(10000, CacheTreeAuto)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	// 1% of sprites move each frame (triggers auto-invalidation).
	container := s.Root.ChildAt(0)
	movers := container.Children()[:100]

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, m := range movers {
			m.SetPosition(m.X_+1, m.Y_)
		}
		s.Draw(screen)
	}
}

func BenchmarkDraw_CacheAsTree_None_Baseline(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_CacheAsTree_Manual_ContainerMove(b *testing.B) {
	s := setupCacheAsTreeBenchScene(10000, CacheTreeManual)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	container := s.Root.ChildAt(0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		container.X_ = float64(i % 100)
		container.TransformDirty = true
		s.Draw(screen)
	}
}

// --- Particle Draw Benchmarks (Immediate vs Coalesced) ---

func setupParticleDrawScene(mode BatchMode) *Scene {
	s := NewScene()
	s.SetBatchMode(mode)
	cfg := EmitterConfig{
		MaxParticles: 1000,
		EmitRate:     100000,
		Lifetime:     Range{Min: 10, Max: 10},
		Speed:        Range{Min: 10, Max: 50},
		Angle:        Range{Min: 0, Max: 2 * math.Pi},
		StartScale:   Range{Min: 1, Max: 1},
		EndScale:     Range{Min: 0.1, Max: 0.1},
		StartAlpha:   Range{Min: 1, Max: 1},
		EndAlpha:     Range{Min: 0, Max: 0},
		StartColor:   RGBA(1, 1, 1, 1),
		EndColor:     RGBA(1, 0, 0, 1),
		Region: TextureRegion{
			Page:      magentaPlaceholderPage,
			Width:     8,
			Height:    8,
			OriginalW: 8,
			OriginalH: 8,
		},
	}
	emitterNode := NewParticleEmitter("particles", cfg)
	emitterNode.Emitter.Start()
	for emitterNode.Emitter.Alive < cfg.MaxParticles {
		emitterNode.Emitter.Update(1.0 / 60.0)
	}
	s.Root.AddChild(emitterNode)
	return s
}

func BenchmarkParticle_Draw_Immediate(b *testing.B) {
	s := setupParticleDrawScene(BatchModeImmediate)
	screen := ebiten.NewImage(640, 480)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkParticle_Draw_Coalesced(b *testing.B) {
	s := setupParticleDrawScene(BatchModeCoalesced)
	screen := ebiten.NewImage(640, 480)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

// --- Transform Benchmarks ---

func BenchmarkTransform_10000Dirty(b *testing.B) {
	s := setupBenchScene(10000)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Mark all transforms dirty.
		markSubtreeDirty(s.Root)
		node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	}
}

func BenchmarkTransform_10000Clean(b *testing.B) {
	s := setupBenchScene(10000)
	// Pre-compute so nothing is dirty.
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	}
}

func BenchmarkTransform_10000AlphaOnly(b *testing.B) {
	s := setupBenchScene(10000)
	// Pre-compute all transforms so they're clean.
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Mark only alpha dirty on every node.
		children := s.Root.Children()
		for _, child := range children {
			child.AlphaDirty = true
		}
		s.Root.AlphaDirty = true
		node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	}
}

// --- Command Sort Benchmark ---

func BenchmarkCommandSort_10000(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)

	// Generate commands via a real traversal.
	s.Draw(screen)

	// Save commands for reset.
	saved := make([]RenderCommand, len(s.Pipeline.Commands))
	copy(saved, s.Pipeline.Commands)

	// Warm up sortBuf to high-water mark.
	render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Restore unsorted commands.
		s.Pipeline.Commands = s.Pipeline.Commands[:len(saved)]
		copy(s.Pipeline.Commands, saved)
		render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)
	}
}

// --- Culling Benchmarks ---

func BenchmarkCulling_On_10000(b *testing.B) {
	s := setupBenchScene(10000)
	cam := s.NewCamera(Rect{X: 0, Y: 0, Width: 640, Height: 480})
	cam.CullEnabled = true
	screen := ebiten.NewImage(640, 480)

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkCulling_Off_10000(b *testing.B) {
	s := setupBenchScene(10000)
	cam := s.NewCamera(Rect{X: 0, Y: 0, Width: 640, Height: 480})
	cam.CullEnabled = false
	screen := ebiten.NewImage(640, 480)

	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

// --- Hit Testing Benchmark ---

func BenchmarkHitTest_1000Interactable(b *testing.B) {
	s := NewScene()
	for i := 0; i < 1000; i++ {
		n := NewSprite("n", TextureRegion{OriginalW: 10, OriginalH: 10})
		n.Interactable = true
		n.X_ = float64(i%100) * 12
		n.Y_ = float64(i/100) * 12
		s.Root.AddChild(n)
	}
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Input.HitTest(s.Root, 500, 50)
	}
}

// --- Particle Benchmarks ---

func BenchmarkParticle_10000Particles(b *testing.B) {
	cfg := EmitterConfig{
		MaxParticles: 10000,
		EmitRate:     100000, // emit fast to fill pool
		Lifetime:     Range{Min: 10, Max: 10},
		Speed:        Range{Min: 10, Max: 50},
		Angle:        Range{Min: 0, Max: 2 * math.Pi},
		StartScale:   Range{Min: 1, Max: 1},
		EndScale:     Range{Min: 0.1, Max: 0.1},
		StartAlpha:   Range{Min: 1, Max: 1},
		EndAlpha:     Range{Min: 0, Max: 0},
		Gravity:      Vec2{X: 0, Y: 100},
		StartColor:   RGBA(1, 1, 1, 1),
		EndColor:     RGBA(1, 0, 0, 1),
		Region: TextureRegion{
			Page:      magentaPlaceholderPage,
			Width:     8,
			Height:    8,
			OriginalW: 8,
			OriginalH: 8,
		},
	}
	emitter := newParticleEmitter(cfg)
	emitter.Start()

	// Fill to capacity.
	for emitter.Alive < cfg.MaxParticles {
		emitter.Update(1.0 / 60.0)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		emitter.Update(1.0 / 60.0)
	}
}

func BenchmarkParticle_MaxEmitters(b *testing.B) {
	const numEmitters = 10
	emitters := make([]*ParticleEmitter, numEmitters)
	cfg := EmitterConfig{
		MaxParticles: 1000,
		EmitRate:     100000,
		Lifetime:     Range{Min: 10, Max: 10},
		Speed:        Range{Min: 10, Max: 50},
		Angle:        Range{Min: 0, Max: 2 * math.Pi},
		StartScale:   Range{Min: 1, Max: 1},
		EndScale:     Range{Min: 0.1, Max: 0.1},
		StartAlpha:   Range{Min: 1, Max: 1},
		EndAlpha:     Range{Min: 0, Max: 0},
		Gravity:      Vec2{X: 0, Y: 100},
		StartColor:   RGBA(1, 1, 1, 1),
		EndColor:     RGBA(1, 0, 0, 1),
		Region: TextureRegion{
			Page:      magentaPlaceholderPage,
			Width:     8,
			Height:    8,
			OriginalW: 8,
			OriginalH: 8,
		},
	}
	for i := range emitters {
		emitters[i] = newParticleEmitter(cfg)
		emitters[i].Start()
		for emitters[i].Alive < cfg.MaxParticles {
			emitters[i].Update(1.0 / 60.0)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, e := range emitters {
			e.Update(1.0 / 60.0)
		}
	}
}

// --- Filter Benchmarks ---

func BenchmarkFilter_BlurOutline(b *testing.B) {
	s := NewScene()
	root := s.Root
	region := TextureRegion{
		Page:      magentaPlaceholderPage,
		Width:     32,
		Height:    32,
		OriginalW: 32,
		OriginalH: 32,
	}
	for i := 0; i < 100; i++ {
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%10) * 40
		sp.Y_ = float64(i/10) * 40
		sp.Filters = []any{NewBlurFilter(4), NewOutlineFilter(2, ColorWhite)}
		root.AddChild(sp)
	}
	screen := ebiten.NewImage(640, 480)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

// --- Lighting Benchmark ---

func BenchmarkLighting_MultipleLights(b *testing.B) {
	ll := NewLightLayer(512, 512, 0.7)
	defer ll.Dispose()

	for i := 0; i < 50; i++ {
		ll.AddLight(&Light{
			X:         float64(i%10) * 50,
			Y:         float64(i/10) * 50,
			Radius:    30,
			Intensity: 0.8,
			Enabled:   true,
		})
	}
	ll.Redraw() // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ll.Redraw()
	}
}

// --- Realistic Multi-Page / Mixed Benchmarks ---

func setupMultiPageScene(n int) *Scene {
	s := NewScene()
	pages := [4]*ebiten.Image{
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
	}
	for i, p := range pages {
		s.RegisterPage(i, p)
	}
	root := s.Root
	for i := 0; i < n; i++ {
		region := TextureRegion{
			Page:      uint16(i % 4),
			X:         0,
			Y:         0,
			Width:     32,
			Height:    32,
			OriginalW: 32,
			OriginalH: 32,
		}
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%100) * 40
		sp.Y_ = float64(i/100) * 40
		root.AddChild(sp)
	}
	return s
}

func setupMixedScene(nSprites, nEmitters int) *Scene {
	s := NewScene()
	pages := [2]*ebiten.Image{
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
	}
	for i, p := range pages {
		s.RegisterPage(i, p)
	}
	root := s.Root

	spritesPerGroup := nSprites / (nEmitters + 1)
	spriteIdx := 0

	addSprites := func(count int) {
		for j := 0; j < count && spriteIdx < nSprites; j++ {
			region := TextureRegion{
				Page:      uint16(spriteIdx % 2),
				X:         0,
				Y:         0,
				Width:     32,
				Height:    32,
				OriginalW: 32,
				OriginalH: 32,
			}
			sp := NewSprite("sp", region)
			sp.X_ = float64(spriteIdx%100) * 40
			sp.Y_ = float64(spriteIdx/100) * 40
			if spriteIdx%10 == 0 {
				sp.BlendMode_ = BlendAdd
			}
			root.AddChild(sp)
			spriteIdx++
		}
	}

	for e := 0; e < nEmitters; e++ {
		addSprites(spritesPerGroup)
		cfg := EmitterConfig{
			MaxParticles: 200,
			EmitRate:     100000,
			Lifetime:     Range{Min: 10, Max: 10},
			Speed:        Range{Min: 10, Max: 50},
			Angle:        Range{Min: 0, Max: 2 * math.Pi},
			StartScale:   Range{Min: 1, Max: 1},
			EndScale:     Range{Min: 0.1, Max: 0.1},
			StartAlpha:   Range{Min: 1, Max: 1},
			EndAlpha:     Range{Min: 0, Max: 0},
			StartColor:   RGBA(1, 1, 1, 1),
			EndColor:     RGBA(1, 0, 0, 1),
			Region: TextureRegion{
				Page:      uint16(e % 2),
				Width:     8,
				Height:    8,
				OriginalW: 8,
				OriginalH: 8,
			},
		}
		emitterNode := NewParticleEmitter("particles", cfg)
		emitterNode.Emitter.Start()
		for emitterNode.Emitter.Alive < cfg.MaxParticles {
			emitterNode.Emitter.Update(1.0 / 60.0)
		}
		root.AddChild(emitterNode)
	}
	addSprites(nSprites - spriteIdx)

	return s
}

func setupWorstCaseScene(n int) *Scene {
	s := NewScene()
	pages := [2]*ebiten.Image{
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
	}
	for i, p := range pages {
		s.RegisterPage(i, p)
	}
	root := s.Root
	for i := 0; i < n; i++ {
		region := TextureRegion{
			Page:      uint16(i % 2),
			X:         0,
			Y:         0,
			Width:     32,
			Height:    32,
			OriginalW: 32,
			OriginalH: 32,
		}
		sp := NewSprite("sp", region)
		sp.X_ = float64(i%100) * 40
		sp.Y_ = float64(i/100) * 40
		root.AddChild(sp)
	}
	return s
}

func BenchmarkDraw_MultiPage_Immediate(b *testing.B) {
	s := setupMultiPageScene(10000)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_MultiPage_Coalesced(b *testing.B) {
	s := setupMultiPageScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_Mixed_Immediate(b *testing.B) {
	s := setupMixedScene(5000, 5)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_Mixed_Coalesced(b *testing.B) {
	s := setupMixedScene(5000, 5)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_WorstCase_Immediate(b *testing.B) {
	s := setupWorstCaseScene(10000)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_WorstCase_Coalesced(b *testing.B) {
	s := setupWorstCaseScene(10000)
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func setupRealWorldAtlasScene(runLen, runs int) *Scene {
	s := NewScene()
	pages := [2]*ebiten.Image{
		ebiten.NewImage(4096, 4096),
		ebiten.NewImage(4096, 4096),
	}
	for i, p := range pages {
		s.RegisterPage(i, p)
	}
	root := s.Root
	total := runLen * runs
	for i := 0; i < total; i++ {
		run := i / runLen
		page := uint16(run % 2)
		tileX := uint16((i % 64) * 64)
		tileY := uint16((i / 64 % 64) * 64)
		region := TextureRegion{
			Page:      page,
			X:         tileX,
			Y:         tileY,
			Width:     64,
			Height:    64,
			OriginalW: 64,
			OriginalH: 64,
		}
		sp := NewSprite("tile", region)
		sp.X_ = float64(i%100) * 64
		sp.Y_ = float64(i/100) * 64
		root.AddChild(sp)
	}
	return s
}

func BenchmarkDraw_RealWorldAtlas_Immediate(b *testing.B) {
	s := setupRealWorldAtlasScene(1000, 10) // 10K sprites, 10 page swaps
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

func BenchmarkDraw_RealWorldAtlas_Coalesced(b *testing.B) {
	s := setupRealWorldAtlasScene(1000, 10) // 10K sprites, 10 page swaps
	s.SetBatchMode(BatchModeCoalesced)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // warmup

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Draw(screen)
	}
}

// --- Tween Tick Benchmarks ---

func BenchmarkTweenTick_100(b *testing.B) {
	benchmarkTweenTick(b, 100)
}

func BenchmarkTweenTick_1000(b *testing.B) {
	benchmarkTweenTick(b, 1000)
}

func benchmarkTweenTick(b *testing.B, count int) {
	b.Helper()
	tweens := make([]*TweenGroup, count)
	for i := range tweens {
		n := NewSprite("t", TextureRegion{OriginalW: 8, OriginalH: 8})
		tweens[i] = TweenPosition(n, 100, 200, TweenConfig{Duration: 2.0})
		tweens[i].Managed = false // tick manually
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, tw := range tweens {
			tw.Tick(1.0 / 60.0)
		}
	}
}

// --- UpdateNodesAndParticles Benchmarks ---

func BenchmarkUpdateNodes_10000_NoCallbacks(b *testing.B) {
	s := setupBenchScene(10000)
	// Pre-compute transforms so the tree is ready.
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, true, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		updateNodesAndParticles(s.Root, 1.0/60.0)
	}
}

func BenchmarkUpdateNodes_10000_WithCallbacks(b *testing.B) {
	s := setupBenchScene(10000)
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, true, true)

	// Attach a lightweight OnUpdate to every sprite.
	for _, child := range s.Root.Children() {
		child.OnUpdate = func(dt float64) {}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		updateNodesAndParticles(s.Root, 1.0/60.0)
	}
}

// --- Command Sort: Already-Sorted Fast Path ---

func BenchmarkCommandSort_10000_PreSorted(b *testing.B) {
	s := setupBenchScene(10000)
	screen := ebiten.NewImage(1280, 720)
	s.Draw(screen) // generate and sort commands

	// Commands are now sorted from the Draw call above.
	sorted := make([]RenderCommand, len(s.Pipeline.Commands))
	copy(sorted, s.Pipeline.Commands)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Pipeline.Commands = s.Pipeline.Commands[:len(sorted)]
		copy(s.Pipeline.Commands, sorted)
		render.MergeSort(s.Pipeline.Commands, &s.Pipeline.SortBuf)
	}
}

// =============================================================================
// Raw Ebitengine baselines
// =============================================================================

type rawSprite struct {
	sub *ebiten.Image
	op  ebiten.DrawImageOptions
}

type rawBatch struct {
	verts []ebiten.Vertex
	inds  []uint32
	page  *ebiten.Image
}

func buildRawSprites_SinglePage(n int) ([]rawSprite, *ebiten.Image) {
	page := ebiten.NewImage(128, 128)
	sub := page.SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image)
	sprites := make([]rawSprite, n)
	for i := range sprites {
		sprites[i].sub = sub
		sprites[i].op.GeoM.Translate(float64(i%100)*40, float64(i/100)*40)
		sprites[i].op.ColorScale.Scale(1, 1, 1, 1)
	}
	return sprites, page
}

func buildRawBatches_SinglePage(n int) ([]rawBatch, *ebiten.Image) {
	page := ebiten.NewImage(128, 128)
	verts := make([]ebiten.Vertex, 0, n*4)
	inds := make([]uint32, 0, n*6)
	for i := 0; i < n; i++ {
		x := float32(i%100) * 40
		y := float32(i/100) * 40
		base := uint32(len(verts))
		verts = append(verts,
			ebiten.Vertex{DstX: x, DstY: y, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x + 32, DstY: y, SrcX: 32, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x, DstY: y + 32, SrcX: 0, SrcY: 32, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x + 32, DstY: y + 32, SrcX: 32, SrcY: 32, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		)
		inds = append(inds, base, base+1, base+2, base+1, base+3, base+2)
	}
	return []rawBatch{{verts: verts, inds: inds, page: page}}, page
}

func buildRawSprites_RealWorld(runLen, runs int) []rawSprite {
	pages := [2]*ebiten.Image{
		ebiten.NewImage(4096, 4096),
		ebiten.NewImage(4096, 4096),
	}
	total := runLen * runs
	sprites := make([]rawSprite, total)
	for i := 0; i < total; i++ {
		run := i / runLen
		p := pages[run%2]
		tileX := (i % 64) * 64
		tileY := (i / 64 % 64) * 64
		sprites[i].sub = p.SubImage(image.Rect(tileX, tileY, tileX+64, tileY+64)).(*ebiten.Image)
		sprites[i].op.GeoM.Translate(float64(i%100)*64, float64(i/100)*64)
		sprites[i].op.ColorScale.Scale(1, 1, 1, 1)
	}
	return sprites
}

func buildRawBatches_RealWorld(runLen, runs int) []rawBatch {
	pages := [2]*ebiten.Image{
		ebiten.NewImage(4096, 4096),
		ebiten.NewImage(4096, 4096),
	}
	total := runLen * runs
	batches := make([]rawBatch, 0, runs)
	var curVerts []ebiten.Vertex
	var curInds []uint32
	curPage := -1

	flush := func(page *ebiten.Image) {
		if len(curVerts) > 0 {
			batches = append(batches, rawBatch{verts: curVerts, inds: curInds, page: page})
		}
	}

	for i := 0; i < total; i++ {
		run := i / runLen
		pageIdx := run % 2
		if pageIdx != curPage {
			if curPage >= 0 {
				flush(pages[curPage])
			}
			curVerts = make([]ebiten.Vertex, 0, runLen*4)
			curInds = make([]uint32, 0, runLen*6)
			curPage = pageIdx
		}
		x := float32(i%100) * 64
		y := float32(i/100) * 64
		tileX := float32((i % 64) * 64)
		tileY := float32((i / 64 % 64) * 64)
		base := uint32(len(curVerts))
		curVerts = append(curVerts,
			ebiten.Vertex{DstX: x, DstY: y, SrcX: tileX, SrcY: tileY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x + 64, DstY: y, SrcX: tileX + 64, SrcY: tileY, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x, DstY: y + 64, SrcX: tileX, SrcY: tileY + 64, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			ebiten.Vertex{DstX: x + 64, DstY: y + 64, SrcX: tileX + 64, SrcY: tileY + 64, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		)
		curInds = append(curInds, base, base+1, base+2, base+1, base+3, base+2)
	}
	flush(pages[curPage])
	return batches
}

func buildRawSprites_Mixed(nSprites, nEmitters, particlesPerEmitter int) []rawSprite {
	pages := [2]*ebiten.Image{
		ebiten.NewImage(128, 128),
		ebiten.NewImage(128, 128),
	}
	spriteSub := [2]*ebiten.Image{
		pages[0].SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image),
		pages[1].SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image),
	}
	particleSub := [2]*ebiten.Image{
		pages[0].SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image),
		pages[1].SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image),
	}
	total := nSprites + nEmitters*particlesPerEmitter
	sprites := make([]rawSprite, 0, total)
	spritesPerGroup := nSprites / (nEmitters + 1)
	spriteIdx := 0

	addSprites := func(count int) {
		for j := 0; j < count && spriteIdx < nSprites; j++ {
			var rs rawSprite
			rs.sub = spriteSub[spriteIdx%2]
			rs.op.GeoM.Translate(float64(spriteIdx%100)*40, float64(spriteIdx/100)*40)
			rs.op.ColorScale.Scale(1, 1, 1, 1)
			if spriteIdx%10 == 0 {
				rs.op.Blend = BlendAdd.EbitenBlend()
			}
			sprites = append(sprites, rs)
			spriteIdx++
		}
	}

	for e := 0; e < nEmitters; e++ {
		addSprites(spritesPerGroup)
		for p := 0; p < particlesPerEmitter; p++ {
			var rs rawSprite
			rs.sub = particleSub[e%2]
			rs.op.GeoM.Translate(float64(p%50)*10, float64(p/50)*10)
			rs.op.ColorScale.Scale(1, 0.5, 0.5, 0.8)
			sprites = append(sprites, rs)
		}
	}
	addSprites(nSprites - spriteIdx)
	return sprites
}

// --- Raw Ebitengine: Single Page (10K sprites) ---

func BenchmarkRaw_SinglePage_DrawImage(b *testing.B) {
	sprites, _ := buildRawSprites_SinglePage(10000)
	screen := ebiten.NewImage(1280, 720)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range sprites {
			screen.DrawImage(sprites[j].sub, &sprites[j].op)
		}
	}
}

func BenchmarkRaw_SinglePage_DrawTriangles32(b *testing.B) {
	batches, _ := buildRawBatches_SinglePage(10000)
	screen := ebiten.NewImage(1280, 720)
	var triOp ebiten.DrawTrianglesOptions
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range batches {
			screen.DrawTriangles32(batches[j].verts, batches[j].inds, batches[j].page, &triOp)
		}
	}
}

// --- Raw Ebitengine: Real-World Atlas ---

func BenchmarkRaw_RealWorldAtlas_DrawImage(b *testing.B) {
	sprites := buildRawSprites_RealWorld(1000, 10)
	screen := ebiten.NewImage(1280, 720)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range sprites {
			screen.DrawImage(sprites[j].sub, &sprites[j].op)
		}
	}
}

func BenchmarkRaw_RealWorldAtlas_DrawTriangles32(b *testing.B) {
	batches := buildRawBatches_RealWorld(1000, 10)
	screen := ebiten.NewImage(1280, 720)
	var triOp ebiten.DrawTrianglesOptions
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range batches {
			screen.DrawTriangles32(batches[j].verts, batches[j].inds, batches[j].page, &triOp)
		}
	}
}

// --- Raw Ebitengine: Mixed ---

func BenchmarkRaw_Mixed_DrawImage(b *testing.B) {
	sprites := buildRawSprites_Mixed(5000, 5, 200)
	screen := ebiten.NewImage(1280, 720)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range sprites {
			screen.DrawImage(sprites[j].sub, &sprites[j].op)
		}
	}
}

// --- Raw Ebitengine: Particles ---

func BenchmarkRaw_Particles_DrawImage(b *testing.B) {
	page := ebiten.NewImage(128, 128)
	sub := page.SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image)
	sprites := make([]rawSprite, 1000)
	for i := range sprites {
		sprites[i].sub = sub
		sprites[i].op.GeoM.Translate(float64(i%50)*10, float64(i/50)*10)
		sprites[i].op.GeoM.Scale(0.5, 0.5)
		sprites[i].op.ColorScale.Scale(0.8, 0.4, 0.4, 0.7)
	}
	screen := ebiten.NewImage(640, 480)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := range sprites {
			screen.DrawImage(sprites[j].sub, &sprites[j].op)
		}
	}
}

func BenchmarkRaw_Particles_DrawTriangles32(b *testing.B) {
	page := ebiten.NewImage(128, 128)
	verts := make([]ebiten.Vertex, 0, 1000*4)
	inds := make([]uint32, 0, 1000*6)
	for i := 0; i < 1000; i++ {
		x := float32(i%50) * 10 * 0.5
		y := float32(i/50) * 10 * 0.5
		base := uint32(len(verts))
		verts = append(verts,
			ebiten.Vertex{DstX: x, DstY: y, SrcX: 0, SrcY: 0, ColorR: 0.56, ColorG: 0.28, ColorB: 0.28, ColorA: 0.7},
			ebiten.Vertex{DstX: x + 4, DstY: y, SrcX: 8, SrcY: 0, ColorR: 0.56, ColorG: 0.28, ColorB: 0.28, ColorA: 0.7},
			ebiten.Vertex{DstX: x, DstY: y + 4, SrcX: 0, SrcY: 8, ColorR: 0.56, ColorG: 0.28, ColorB: 0.28, ColorA: 0.7},
			ebiten.Vertex{DstX: x + 4, DstY: y + 4, SrcX: 8, SrcY: 8, ColorR: 0.56, ColorG: 0.28, ColorB: 0.28, ColorA: 0.7},
		)
		inds = append(inds, base, base+1, base+2, base+1, base+3, base+2)
	}
	screen := ebiten.NewImage(640, 480)
	var triOp ebiten.DrawTrianglesOptions
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		screen.DrawTriangles32(verts, inds, page, &triOp)
	}
}
