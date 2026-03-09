package willow

import (
	"github.com/phanxgames/willow/internal/core"
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

// updateNodesAndParticles delegates to core.UpdateNodesAndParticles.
func updateNodesAndParticles(n *Node, dt float64) {
	core.UpdateNodesAndParticles(n, dt)
}

// lerp linearly interpolates between a and b by t.
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

