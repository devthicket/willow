package particle

import (
	"math"

	"github.com/phanxgames/willow/internal/types"
)

// particle holds per-particle simulation state.
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

// Emitter manages a pool of particles with CPU-based simulation.
type Emitter struct {
	Config    EmitterConfig
	Particles []particle
	Alive     int
	EmitAccum float64
	Active    bool
	// World-space tracking: the emitter's last known world position,
	// set by the update walk so particles can be spawned at world coords.
	WorldX, WorldY float64
}

// NewEmitter creates an Emitter with a preallocated pool.
func NewEmitter(cfg EmitterConfig) *Emitter {
	max := cfg.MaxParticles
	if max <= 0 {
		max = 128
	}
	return &Emitter{
		Config:    cfg,
		Particles: make([]particle, max),
	}
}

// Start begins emitting particles.
func (e *Emitter) Start() {
	e.Active = true
}

// Stop stops emitting new particles. Existing particles continue to live out.
func (e *Emitter) Stop() {
	e.Active = false
}

// Reset stops emitting and kills all alive particles.
func (e *Emitter) Reset() {
	e.Active = false
	e.Alive = 0
	e.EmitAccum = 0
}

// IsActive reports whether the emitter is currently emitting new particles.
func (e *Emitter) IsActive() bool {
	return e.Active
}

// AliveCount returns the number of alive particles.
func (e *Emitter) AliveCount() int {
	return e.Alive
}

// Bounds returns the axis-aligned bounding box of all alive particles in the
// emitter's coordinate space (local for local-space, world for world-space).
// The returned rect includes particle scale so the full rendered extent is
// covered. Returns a zero Rect when no particles are alive.
func (e *Emitter) Bounds() types.Rect {
	if e.Alive == 0 {
		return types.Rect{}
	}
	rw := float64(e.Config.Region.OriginalW)
	rh := float64(e.Config.Region.OriginalH)
	if rw == 0 {
		rw = 1
	}
	if rh == 0 {
		rh = 1
	}

	p := &e.Particles[0]
	halfW := rw * float64(p.scale) * 0.5
	halfH := rh * float64(p.scale) * 0.5
	minX, minY := p.x-halfW, p.y-halfH
	maxX, maxY := p.x+halfW, p.y+halfH

	for i := 1; i < e.Alive; i++ {
		p = &e.Particles[i]
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
	return types.Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// Update advances particle simulation by dt seconds.
func (e *Emitter) Update(dt float64) {
	gx := e.Config.Gravity.X * dt
	gy := e.Config.Gravity.Y * dt

	i := 0
	for i < e.Alive {
		p := &e.Particles[i]
		p.life -= dt
		if p.life <= 0 {
			e.Alive--
			e.Particles[i] = e.Particles[e.Alive]
			continue
		}

		p.vx += gx
		p.vy += gy

		p.x += p.vx * dt
		p.y += p.vy * dt

		t := float32(1.0 - p.life/p.maxLife)
		p.scale = types.Lerp32(p.startScale, p.endScale, t)
		p.alpha = types.Lerp32(p.startAlpha, p.endAlpha, t)
		p.colorR = types.Lerp32(p.startR, p.endR, t)
		p.colorG = types.Lerp32(p.startG, p.endG, t)
		p.colorB = types.Lerp32(p.startB, p.endB, t)

		i++
	}

	if e.Active && e.Config.EmitRate > 0 {
		e.EmitAccum += e.Config.EmitRate * dt
		for e.EmitAccum >= 1.0 {
			e.EmitAccum -= 1.0
			if e.Alive < len(e.Particles) {
				e.spawnParticle()
			}
		}
	}
}

// spawnParticle initializes the particle at slot e.Alive and increments alive.
func (e *Emitter) spawnParticle() {
	p := &e.Particles[e.Alive]

	angle := e.Config.Angle.Random()
	speed := e.Config.Speed.Random()
	p.vx = math.Cos(angle) * speed
	p.vy = math.Sin(angle) * speed

	if e.Config.WorldSpace {
		p.x = e.WorldX
		p.y = e.WorldY
	} else {
		p.x = 0
		p.y = 0
	}

	p.life = e.Config.Lifetime.Random()
	if p.life <= 0 {
		p.life = 1.0
	}
	p.maxLife = p.life

	p.startScale = float32(e.Config.StartScale.Random())
	p.endScale = float32(e.Config.EndScale.Random())
	p.scale = p.startScale

	p.startAlpha = float32(e.Config.StartAlpha.Random())
	p.endAlpha = float32(e.Config.EndAlpha.Random())
	p.alpha = p.startAlpha

	p.startR = float32(e.Config.StartColor.R())
	p.startG = float32(e.Config.StartColor.G())
	p.startB = float32(e.Config.StartColor.B())
	p.endR = float32(e.Config.EndColor.R())
	p.endG = float32(e.Config.EndColor.G())
	p.endB = float32(e.Config.EndColor.B())
	p.colorR = p.startR
	p.colorG = p.startG
	p.colorB = p.startB

	e.Alive++
}

// ParticleScale returns the scale of the particle at the given index.
func (e *Emitter) ParticleScale(i int) float32 { return e.Particles[i].scale }

// ParticleAlpha returns the alpha of the particle at the given index.
func (e *Emitter) ParticleAlpha(i int) float32 { return e.Particles[i].alpha }

// ParticleColor returns the RGB color of the particle at the given index.
func (e *Emitter) ParticleColor(i int) (r, g, b float32) {
	p := &e.Particles[i]
	return p.colorR, p.colorG, p.colorB
}

// ParticlePos returns the position of the particle at the given index.
func (e *Emitter) ParticlePos(i int) (x, y float64) {
	return e.Particles[i].x, e.Particles[i].y
}
