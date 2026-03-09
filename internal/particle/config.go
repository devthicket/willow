package particle

import "github.com/phanxgames/willow/internal/types"

// EmitterConfig controls how particles are spawned and behave.
type EmitterConfig struct {
	// MaxParticles is the pool size. New particles are silently dropped when full.
	MaxParticles int
	// EmitRate is the number of particles spawned per second.
	EmitRate float64
	// Lifetime is the range of particle lifetimes in seconds.
	Lifetime types.Range
	// Speed is the range of initial particle speeds in pixels per second.
	Speed types.Range
	// Angle is the range of emission angles in radians.
	Angle types.Range
	// StartScale is the range of scale factors at birth, interpolated to EndScale over lifetime.
	StartScale types.Range
	// EndScale is the range of scale factors at death.
	EndScale types.Range
	// StartAlpha is the range of alpha values at birth, interpolated to EndAlpha over lifetime.
	StartAlpha types.Range
	// EndAlpha is the range of alpha values at death.
	EndAlpha types.Range
	// Gravity is the constant acceleration applied to all particles each frame.
	Gravity types.Vec2
	// StartColor is the tint at birth, interpolated to EndColor over lifetime.
	StartColor types.Color
	// EndColor is the tint at death.
	EndColor types.Color
	// Region is the TextureRegion used to render each particle.
	Region types.TextureRegion
	// BlendMode is the compositing operation for particle rendering.
	BlendMode types.BlendMode
	// WorldSpace, when true, causes particles to keep their world position
	// once emitted rather than following the emitter node.
	WorldSpace bool
}
