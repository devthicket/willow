package willow

import (
	"github.com/phanxgames/willow/internal/particle"
)

// EmitterConfig controls how particles are spawned and behave.
type EmitterConfig = particle.EmitterConfig

// ParticleEmitter manages a pool of particles with CPU-based simulation.
type ParticleEmitter = particle.Emitter

// newParticleEmitter creates a ParticleEmitter with a preallocated pool.
func newParticleEmitter(cfg EmitterConfig) *ParticleEmitter {
	return particle.NewEmitter(cfg)
}

// lerp linearly interpolates between a and b by t.
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// lerp32 linearly interpolates between a and b by t (float32).
func lerp32(a, b, t float32) float32 {
	return a + (b-a)*t
}
