package willow

import (
	"math"
)

// particle holds per-particle simulation state. Unexported; managed by ParticleEmitter.
type particle struct {
	x, y       float64
	vx, vy     float64
	life       float64 // remaining lifetime in seconds
	maxLife    float64 // initial lifetime (for computing t)
	startScale float32
	endScale   float32
	scale      float32
	startAlpha float32
	endAlpha   float32
	alpha      float32
	startR     float32
	startG     float32
	startB     float32
	endR       float32
	endG       float32
	endB       float32
	colorR     float32
	colorG     float32
	colorB     float32
}

// EmitterConfig controls how particles are spawned and behave.
type EmitterConfig struct {
	// MaxParticles is the pool size. New particles are silently dropped when full.
	MaxParticles int
	// EmitRate is the number of particles spawned per second.
	EmitRate float64
	// Lifetime is the range of particle lifetimes in seconds.
	Lifetime Range
	// Speed is the range of initial particle speeds in pixels per second.
	Speed Range
	// Angle is the range of emission angles in radians.
	Angle Range
	// StartScale is the range of scale factors at birth, interpolated to EndScale over lifetime.
	StartScale Range
	// EndScale is the range of scale factors at death.
	EndScale Range
	// StartAlpha is the range of alpha values at birth, interpolated to EndAlpha over lifetime.
	StartAlpha Range
	// EndAlpha is the range of alpha values at death.
	EndAlpha Range
	// Gravity is the constant acceleration applied to all particles each frame.
	Gravity Vec2
	// StartColor is the tint at birth, interpolated to EndColor over lifetime.
	StartColor Color
	// EndColor is the tint at death.
	EndColor Color
	// Region is the TextureRegion used to render each particle.
	Region TextureRegion
	// BlendMode is the compositing operation for particle rendering.
	BlendMode BlendMode
	// WorldSpace, when true, causes particles to keep their world position
	// once emitted rather than following the emitter node.
	WorldSpace bool
}

// ParticleEmitter manages a pool of particles with CPU-based simulation.
type ParticleEmitter struct {
	config    EmitterConfig
	particles []particle
	alive     int
	emitAccum float64
	active    bool
	// World-space tracking: the emitter's last known world position,
	// set by the update walk so particles can be spawned at world coords.
	worldX, worldY float64
}

// newParticleEmitter creates a ParticleEmitter with a preallocated pool.
func newParticleEmitter(cfg EmitterConfig) *ParticleEmitter {
	max := cfg.MaxParticles
	if max <= 0 {
		max = 128
	}
	return &ParticleEmitter{
		config:    cfg,
		particles: make([]particle, max),
	}
}

// Start begins emitting particles.
func (e *ParticleEmitter) Start() {
	e.active = true
}

// Stop stops emitting new particles. Existing particles continue to live out.
func (e *ParticleEmitter) Stop() {
	e.active = false
}

// Reset stops emitting and kills all alive particles.
func (e *ParticleEmitter) Reset() {
	e.active = false
	e.alive = 0
	e.emitAccum = 0
}

// IsActive reports whether the emitter is currently emitting new particles.
func (e *ParticleEmitter) IsActive() bool {
	return e.active
}

// AliveCount returns the number of alive particles.
func (e *ParticleEmitter) AliveCount() int {
	return e.alive
}

// Config returns a pointer to the emitter's config for live tuning.
func (e *ParticleEmitter) Config() *EmitterConfig {
	return &e.config
}

// bounds returns the axis-aligned bounding box of all alive particles in the
// emitter's coordinate space (local for local-space, world for world-space).
// The returned rect includes particle scale so the full rendered extent is
// covered. Returns a zero Rect when no particles are alive.
func (e *ParticleEmitter) bounds() Rect {
	if e.alive == 0 {
		return Rect{}
	}
	// Region dimensions determine the rendered size of each particle.
	rw := float64(e.config.Region.OriginalW)
	rh := float64(e.config.Region.OriginalH)
	if rw == 0 {
		rw = 1 // WhitePixel fallback
	}
	if rh == 0 {
		rh = 1
	}

	p := &e.particles[0]
	halfW := rw * float64(p.scale) * 0.5
	halfH := rh * float64(p.scale) * 0.5
	minX, minY := p.x-halfW, p.y-halfH
	maxX, maxY := p.x+halfW, p.y+halfH

	for i := 1; i < e.alive; i++ {
		p = &e.particles[i]
		halfW = rw * float64(p.scale) * 0.5
		halfH = rh * float64(p.scale) * 0.5
		px0, py0 := p.x-halfW, p.y-halfH
		px1, py1 := p.x+halfW, p.y+halfH
		if px0 < minX {
			minX = px0
		}
		if py0 < minY {
			minY = py0
		}
		if px1 > maxX {
			maxX = px1
		}
		if py1 > maxY {
			maxY = py1
		}
	}
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// update advances particle simulation by dt seconds.
func (e *ParticleEmitter) update(dt float64) {
	gx := e.config.Gravity.X * dt
	gy := e.config.Gravity.Y * dt

	// Update existing particles, swap-remove dead ones.
	i := 0
	for i < e.alive {
		p := &e.particles[i]
		p.life -= dt
		if p.life <= 0 {
			// Swap with last alive particle.
			e.alive--
			e.particles[i] = e.particles[e.alive]
			continue
		}

		// Apply gravity.
		p.vx += gx
		p.vy += gy

		// Move.
		p.x += p.vx * dt
		p.y += p.vy * dt

		// Interpolate properties.
		t := float32(1.0 - p.life/p.maxLife)
		p.scale = lerp32(p.startScale, p.endScale, t)
		p.alpha = lerp32(p.startAlpha, p.endAlpha, t)
		p.colorR = lerp32(p.startR, p.endR, t)
		p.colorG = lerp32(p.startG, p.endG, t)
		p.colorB = lerp32(p.startB, p.endB, t)

		i++
	}

	// Emit new particles.
	if e.active && e.config.EmitRate > 0 {
		e.emitAccum += e.config.EmitRate * dt
		for e.emitAccum >= 1.0 {
			e.emitAccum -= 1.0
			if e.alive < len(e.particles) {
				e.spawnParticle()
			}
		}
	}
}

// spawnParticle initializes the particle at slot e.alive and increments alive.
func (e *ParticleEmitter) spawnParticle() {
	p := &e.particles[e.alive]

	angle := e.config.Angle.Random()
	speed := e.config.Speed.Random()
	p.vx = math.Cos(angle) * speed
	p.vy = math.Sin(angle) * speed

	if e.config.WorldSpace {
		p.x = e.worldX
		p.y = e.worldY
	} else {
		p.x = 0
		p.y = 0
	}

	p.life = e.config.Lifetime.Random()
	if p.life <= 0 {
		p.life = 1.0
	}
	p.maxLife = p.life

	p.startScale = float32(e.config.StartScale.Random())
	p.endScale = float32(e.config.EndScale.Random())
	p.scale = p.startScale

	p.startAlpha = float32(e.config.StartAlpha.Random())
	p.endAlpha = float32(e.config.EndAlpha.Random())
	p.alpha = p.startAlpha

	p.startR = float32(e.config.StartColor.r)
	p.startG = float32(e.config.StartColor.g)
	p.startB = float32(e.config.StartColor.b)
	p.endR = float32(e.config.EndColor.r)
	p.endG = float32(e.config.EndColor.g)
	p.endB = float32(e.config.EndColor.b)
	p.colorR = p.startR
	p.colorG = p.startG
	p.colorB = p.startB

	e.alive++
}

// lerp linearly interpolates between a and b by t.
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// lerp32 linearly interpolates between a and b by t (float32).
func lerp32(a, b, t float32) float32 {
	return a + (b-a)*t
}

// Range.Random() is defined in internal/types/range.go.
