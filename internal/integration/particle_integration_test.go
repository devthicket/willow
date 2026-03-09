package integration

import (
	"math"
	"testing"

	. "github.com/phanxgames/willow"
)

func defaultTestConfig(max int) EmitterConfig {
	return EmitterConfig{
		MaxParticles: max,
		EmitRate:     100,
		Lifetime:     Range{Min: 1.0, Max: 1.0},
		Speed:        Range{Min: 100, Max: 100},
		Angle:        Range{Min: 0, Max: 0},
		StartScale:   Range{Min: 1, Max: 1},
		EndScale:     Range{Min: 0.5, Max: 0.5},
		StartAlpha:   Range{Min: 1, Max: 1},
		EndAlpha:     Range{Min: 0, Max: 0},
		Gravity:      Vec2{X: 0, Y: 0},
		StartColor:   RGBA(1, 1, 1, 1),
		EndColor:     RGBA(0, 0, 0, 1),
		Region:       TextureRegion{Width: 16, Height: 16, OriginalW: 16, OriginalH: 16},
	}
}

func TestEmitterConfigCreatesPool(t *testing.T) {
	cfg := defaultTestConfig(500)
	e := newParticleEmitter(cfg)
	if len(e.Particles) != 500 {
		t.Errorf("pool size = %d, want 500", len(e.Particles))
	}
	if e.Alive != 0 {
		t.Errorf("alive = %d, want 0", e.Alive)
	}
}

func TestEmitterDefaultMaxParticles(t *testing.T) {
	cfg := EmitterConfig{MaxParticles: 0}
	e := newParticleEmitter(cfg)
	if len(e.Particles) != 128 {
		t.Errorf("default pool size = %d, want 128", len(e.Particles))
	}
}

func TestStartStopReset(t *testing.T) {
	cfg := defaultTestConfig(100)
	e := newParticleEmitter(cfg)

	if e.IsActive() {
		t.Error("emitter should not be active initially")
	}

	e.Start()
	if !e.IsActive() {
		t.Error("emitter should be active after Start")
	}

	e.Stop()
	if e.IsActive() {
		t.Error("emitter should not be active after Stop")
	}

	// Start and spawn some particles.
	e.Start()
	e.Update(0.1) // should spawn ~10 particles at rate 100/s
	if e.AliveCount() == 0 {
		t.Fatal("expected particles after update")
	}

	e.Reset()
	if e.IsActive() {
		t.Error("emitter should not be active after Reset")
	}
	if e.AliveCount() != 0 {
		t.Errorf("alive = %d, want 0 after Reset", e.AliveCount())
	}
}

func TestParticleSpawnRate(t *testing.T) {
	cfg := defaultTestConfig(1000)
	cfg.EmitRate = 60
	e := newParticleEmitter(cfg)
	e.Start()

	// 1 second at 60/s -> should spawn 60 particles.
	for i := 0; i < 60; i++ {
		e.Update(1.0 / 60.0)
	}

	alive := e.AliveCount()
	if alive != 60 {
		t.Errorf("alive = %d, want 60", alive)
	}
}

func TestSwapRemoveNoDead(t *testing.T) {
	cfg := defaultTestConfig(100)
	cfg.Lifetime = Range{Min: 0.05, Max: 0.05}
	cfg.EmitRate = 100
	e := newParticleEmitter(cfg)
	e.Start()

	e.Update(0.02)
	before := e.AliveCount()
	if before == 0 {
		t.Fatal("expected particles spawned")
	}

	e.Stop()
	e.Update(0.1)
	if e.AliveCount() != 0 {
		t.Errorf("alive = %d, want 0 after particles expire", e.AliveCount())
	}
}

func TestGravityAffectsVelocity(t *testing.T) {
	cfg := defaultTestConfig(10)
	cfg.Gravity = Vec2{X: 0, Y: 100}
	cfg.Speed = Range{Min: 0, Max: 0}
	cfg.Angle = Range{Min: 0, Max: 0}
	cfg.Lifetime = Range{Min: 10, Max: 10}
	cfg.EmitRate = 10000
	e := newParticleEmitter(cfg)
	e.Start()

	e.Update(0.001)
	e.Stop()
	e.Update(1.0)
	if e.AliveCount() == 0 {
		t.Fatal("expected alive particles")
	}

	_, vy := e.ParticleVelocity(0)
	assertNear(t, "vy", vy, 100.0)
	_, py := e.ParticlePos(0)
	if py < 50 {
		t.Errorf("y = %f, expected > 50 with gravity", py)
	}
}

func TestLifetimeInterpolation(t *testing.T) {
	cfg := defaultTestConfig(1)
	cfg.EmitRate = 1000
	cfg.Lifetime = Range{Min: 1, Max: 1}
	cfg.StartScale = Range{Min: 2, Max: 2}
	cfg.EndScale = Range{Min: 0, Max: 0}
	cfg.StartAlpha = Range{Min: 1, Max: 1}
	cfg.EndAlpha = Range{Min: 0, Max: 0}
	cfg.StartColor = RGBA(1, 0, 0, 1)
	cfg.EndColor = RGBA(0, 1, 0, 1)
	e := newParticleEmitter(cfg)
	e.Start()

	e.Update(0.001)
	e.Stop()
	if e.AliveCount() != 1 {
		t.Fatalf("alive = %d, want 1", e.AliveCount())
	}

	assertNear(t, "scale@t0", float64(e.ParticleScale(0)), 2.0)
	assertNear(t, "alpha@t0", float64(e.ParticleAlpha(0)), 1.0)
	cr, cg, _ := e.ParticleColor(0)
	assertNear(t, "colorR@t0", float64(cr), 1.0)
	assertNear(t, "colorG@t0", float64(cg), 0.0)

	e.Update(0.5)
	t50 := e.ParticleLifeFraction(0)
	assertNear(t, "t~0.5", t50, 0.5)
	assertNear(t, "scale@t0.5", float64(e.ParticleScale(0)), lerp(2, 0, t50))
	assertNear(t, "alpha@t0.5", float64(e.ParticleAlpha(0)), lerp(1, 0, t50))
	cr2, cg2, _ := e.ParticleColor(0)
	assertNear(t, "colorR@t0.5", float64(cr2), lerp(1, 0, t50))
	assertNear(t, "colorG@t0.5", float64(cg2), lerp(0, 1, t50))
}

func TestMaxParticlesCap(t *testing.T) {
	cfg := defaultTestConfig(5)
	cfg.EmitRate = 10000
	e := newParticleEmitter(cfg)
	e.Start()

	e.Update(1.0)
	if e.AliveCount() > 5 {
		t.Errorf("alive = %d, exceeds max 5", e.AliveCount())
	}
}

func TestRangeRandom(t *testing.T) {
	r := Range{Min: 10, Max: 20}
	for i := 0; i < 100; i++ {
		v := r.Random()
		if v < 10 || v > 20 {
			t.Fatalf("Random() = %f, outside [10, 20]", v)
		}
	}

	r2 := Range{Min: 5, Max: 5}
	for i := 0; i < 10; i++ {
		if r2.Random() != 5 {
			t.Fatal("Random() with Min==Max should return Min")
		}
	}
}

func TestLerp(t *testing.T) {
	assertNear(t, "lerp(0,10,0)", lerp(0, 10, 0), 0)
	assertNear(t, "lerp(0,10,0.5)", lerp(0, 10, 0.5), 5)
	assertNear(t, "lerp(0,10,1)", lerp(0, 10, 1), 10)
}

// --- Scene-based integration tests ---

func TestRenderCommandIncludesEmitter(t *testing.T) {
	s := NewScene()
	cfg := defaultTestConfig(100)
	emitterNode := NewParticleEmitter("emitter", cfg)
	emitterNode.Emitter.Start()
	emitterNode.Emitter.Update(0.1)
	s.Root.AddChild(emitterNode)

	traverseScene(s)

	found := false
	for _, cmd := range s.Pipeline.Commands {
		if cmd.Type == CommandParticle {
			found = true
			if cmd.Emitter == nil {
				t.Error("CommandParticle should have non-nil emitter")
			}
			if cmd.Emitter != emitterNode.Emitter {
				t.Error("CommandParticle emitter should match node's emitter")
			}
		}
	}
	if !found {
		t.Error("expected at least one CommandParticle")
	}
}

func TestNoCommandWhenNoAliveParticles(t *testing.T) {
	s := NewScene()
	cfg := defaultTestConfig(100)
	emitterNode := NewParticleEmitter("emitter", cfg)
	s.Root.AddChild(emitterNode)

	traverseScene(s)

	for _, cmd := range s.Pipeline.Commands {
		if cmd.Type == CommandParticle {
			t.Error("should not emit CommandParticle when no particles alive")
		}
	}
}

func TestUpdateParticlesRecursive(t *testing.T) {
	root := NewContainer("root")
	child := NewContainer("child")
	root.AddChild(child)

	cfg := defaultTestConfig(100)
	emitter1 := NewParticleEmitter("e1", cfg)
	emitter1.Emitter.Start()
	root.AddChild(emitter1)

	emitter2 := NewParticleEmitter("e2", cfg)
	emitter2.Emitter.Start()
	child.AddChild(emitter2)

	updateNodesAndParticles(root, 0.1)

	if emitter1.Emitter.AliveCount() == 0 {
		t.Error("emitter1 should have particles after updateParticles")
	}
	if emitter2.Emitter.AliveCount() == 0 {
		t.Error("emitter2 should have particles after updateParticles (nested)")
	}
}

func TestParticleEmitterNodeFields(t *testing.T) {
	cfg := defaultTestConfig(50)
	cfg.Region = TextureRegion{Width: 32, Height: 32, Page: 1}
	cfg.BlendMode = BlendAdd
	n := NewParticleEmitter("p", cfg)

	if n.Type != NodeTypeParticleEmitter {
		t.Error("wrong node type")
	}
	if n.Emitter == nil {
		t.Fatal("Emitter should not be nil")
	}
	if n.TextureRegion_.Width != 32 {
		t.Error("TextureRegion should match config")
	}
	if n.BlendMode_ != BlendAdd {
		t.Error("BlendMode should match config")
	}
}

func TestZeroAllocsDuringUpdate(t *testing.T) {
	cfg := defaultTestConfig(1000)
	cfg.EmitRate = 500
	e := newParticleEmitter(cfg)
	e.Start()

	for i := 0; i < 100; i++ {
		e.Update(1.0 / 60.0)
	}

	allocs := testing.AllocsPerRun(100, func() {
		e.Update(1.0 / 60.0)
	})
	if allocs > 0 {
		t.Errorf("update allocs = %f, want 0", allocs)
	}
}

func TestNodeDimensionsParticleEmitter(t *testing.T) {
	cfg := defaultTestConfig(10)
	cfg.Region = TextureRegion{OriginalW: 64, OriginalH: 48}
	n := NewParticleEmitter("p", cfg)
	w, h := nodeDimensions(n)
	if w != 64 || h != 48 {
		t.Errorf("nodeDimensions = (%f, %f), want (64, 48)", w, h)
	}
}

func TestConfigPointerForLiveTuning(t *testing.T) {
	cfg := defaultTestConfig(100)
	e := newParticleEmitter(cfg)
	e.Config.EmitRate = 999
	if e.Config.EmitRate != 999 {
		t.Error("Config should be mutable for live tuning")
	}
}

func TestParticleMovesWithAngle(t *testing.T) {
	cfg := defaultTestConfig(1)
	cfg.EmitRate = 10000
	cfg.Speed = Range{Min: 100, Max: 100}
	cfg.Angle = Range{Min: math.Pi / 2, Max: math.Pi / 2}
	cfg.Lifetime = Range{Min: 10, Max: 10}
	e := newParticleEmitter(cfg)
	e.Start()

	e.Update(1.0)
	if e.AliveCount() == 0 {
		t.Fatal("expected alive particles")
	}
	vx, vy := e.ParticleVelocity(0)
	assertNear(t, "vx", vx, 0)
	assertNear(t, "vy", vy, 100)
}

// --- Benchmarks ---

func BenchmarkParticleUpdate_1000(b *testing.B) {
	cfg := defaultTestConfig(1000)
	cfg.EmitRate = 500
	e := newParticleEmitter(cfg)
	e.Start()
	for i := 0; i < 200; i++ {
		e.Update(1.0 / 60.0)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Update(1.0 / 60.0)
	}
}

func BenchmarkParticleUpdate_10000(b *testing.B) {
	cfg := defaultTestConfig(10000)
	cfg.EmitRate = 5000
	e := newParticleEmitter(cfg)
	e.Start()
	for i := 0; i < 200; i++ {
		e.Update(1.0 / 60.0)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Update(1.0 / 60.0)
	}
}

func BenchmarkParticleRender_1000(b *testing.B) {
	s := NewScene()
	cfg := defaultTestConfig(1000)
	cfg.EmitRate = 5000
	emitterNode := NewParticleEmitter("e", cfg)
	emitterNode.Emitter.Start()
	for i := 0; i < 200; i++ {
		emitterNode.Emitter.Update(1.0 / 60.0)
	}
	s.Root.AddChild(emitterNode)

	traverseScene(s)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Pipeline.Commands = s.Pipeline.Commands[:0]
		treeOrder := 0
		s.Pipeline.Traverse(s.Root, &treeOrder)
	}
}
