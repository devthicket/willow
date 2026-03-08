package particle

import (
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestNewEmitter_DefaultPool(t *testing.T) {
	e := NewEmitter(EmitterConfig{})
	if len(e.Particles) != 128 {
		t.Errorf("default pool = %d, want 128", len(e.Particles))
	}
}

func TestNewEmitter_CustomPool(t *testing.T) {
	e := NewEmitter(EmitterConfig{MaxParticles: 64})
	if len(e.Particles) != 64 {
		t.Errorf("pool = %d, want 64", len(e.Particles))
	}
}

func TestEmitter_StartStop(t *testing.T) {
	e := NewEmitter(EmitterConfig{MaxParticles: 10, EmitRate: 100})
	if e.IsActive() {
		t.Error("should not be active before Start")
	}
	e.Start()
	if !e.IsActive() {
		t.Error("should be active after Start")
	}
	e.Stop()
	if e.IsActive() {
		t.Error("should not be active after Stop")
	}
}

func TestEmitter_Update_SpawnsParticles(t *testing.T) {
	e := NewEmitter(EmitterConfig{
		MaxParticles: 100,
		EmitRate:     60,
		Lifetime:     types.Range{Min: 1, Max: 1},
		Speed:        types.Range{Min: 10, Max: 10},
		StartScale:   types.Range{Min: 1, Max: 1},
		EndScale:     types.Range{Min: 1, Max: 1},
		StartAlpha:   types.Range{Min: 1, Max: 1},
		EndAlpha:     types.Range{Min: 0, Max: 0},
		StartColor:   types.RGB(1, 1, 1),
		EndColor:     types.RGB(1, 1, 1),
	})
	e.Start()
	// At 60 particles/sec and dt=1/60, should spawn 1 particle per frame.
	e.Update(1.0 / 60.0)
	if e.AliveCount() != 1 {
		t.Errorf("alive = %d, want 1", e.AliveCount())
	}
}

func TestEmitter_Reset(t *testing.T) {
	e := NewEmitter(EmitterConfig{
		MaxParticles: 100,
		EmitRate:     1000,
		Lifetime:     types.Range{Min: 1, Max: 1},
		Speed:        types.Range{Min: 0, Max: 0},
		StartScale:   types.Range{Min: 1, Max: 1},
		EndScale:     types.Range{Min: 1, Max: 1},
		StartAlpha:   types.Range{Min: 1, Max: 1},
		EndAlpha:     types.Range{Min: 1, Max: 1},
		StartColor:   types.RGB(1, 1, 1),
		EndColor:     types.RGB(1, 1, 1),
	})
	e.Start()
	e.Update(0.1)
	if e.AliveCount() == 0 {
		t.Fatal("expected some particles after update")
	}
	e.Reset()
	if e.AliveCount() != 0 {
		t.Errorf("alive after reset = %d, want 0", e.AliveCount())
	}
	if e.IsActive() {
		t.Error("should not be active after reset")
	}
}

func TestEmitter_Bounds_Empty(t *testing.T) {
	e := NewEmitter(EmitterConfig{})
	b := e.Bounds()
	if b.Width != 0 || b.Height != 0 {
		t.Errorf("bounds of empty emitter = %v, want zero", b)
	}
}

func TestEmitter_ParticlesDie(t *testing.T) {
	e := NewEmitter(EmitterConfig{
		MaxParticles: 10,
		EmitRate:     100,
		Lifetime:     types.Range{Min: 0.1, Max: 0.1},
		Speed:        types.Range{Min: 0, Max: 0},
		StartScale:   types.Range{Min: 1, Max: 1},
		EndScale:     types.Range{Min: 1, Max: 1},
		StartAlpha:   types.Range{Min: 1, Max: 1},
		EndAlpha:     types.Range{Min: 1, Max: 1},
		StartColor:   types.RGB(1, 1, 1),
		EndColor:     types.RGB(1, 1, 1),
	})
	e.Start()
	e.Update(0.05) // spawn some
	alive := e.AliveCount()
	if alive == 0 {
		t.Fatal("expected particles to spawn")
	}
	e.Stop()
	// Advance past lifetime
	e.Update(0.2)
	if e.AliveCount() != 0 {
		t.Errorf("alive after lifetime = %d, want 0", e.AliveCount())
	}
}
