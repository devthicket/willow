// physics spawns random shapes with gravity, collisions, and click-to-explode.
// All shapes are procedural (no textures)  -  compiled to WASM for the docs site.
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/phanxgames/willow"
)

const (
	screenW     = 1280
	screenH     = 720
	shapeCount  = 100
	gravity     = 0.12
	restitution = 0.25
	damping     = 0.98
	maxVel      = 12.0

	// Explosion settings
	blastRadius = 350.0
	blastForce  = 50.0

	// Flash animation
	flashFrames = 12

	// Collision solver iterations  -  more passes = less clipping
	solverPasses = 3
)

type body struct {
	node       *willow.Node
	radius     float64
	mass       float64
	vx, vy     float64
	baseColor  willow.Color
	flashTimer int
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.06, 0.06, 0.09)

	root := scene.Root
	bodies := make([]body, shapeCount)

	for i := range bodies {
		radius := 25.0 + rand.Float64()*15.0
		mass := radius / 20.0
		color := willow.RGB(
			0.3+rand.Float64()*0.7,
			0.3+rand.Float64()*0.7,
			0.3+rand.Float64()*0.7,
		)

		var node *willow.Node
		shapeType := rand.IntN(5)

		switch shapeType {
		case 0: // Circle (24-gon)
			node = willow.NewRegularPolygon("circle", 24, radius)
		case 1: // Square
			node = willow.NewRegularPolygon("square", 4, radius)
		case 2: // Triangle
			node = willow.NewRegularPolygon("triangle", 3, radius)
		case 3: // Pentagon
			node = willow.NewRegularPolygon("pentagon", 5, radius)
		case 4: // Hexagon
			node = willow.NewRegularPolygon("hexagon", 6, radius)
		}

		node.SetColor(color)

		// Place shapes spread across the full screen area
		node.SetPosition(
			radius+rand.Float64()*(screenW-2*radius),
			radius+rand.Float64()*(screenH-2*radius),
		)

		node.HitShape = willow.HitCircle{Radius: radius}

		idx := i
		node.OnClick(func(ctx willow.ClickContext) {
			explode(bodies[:], idx)
		})

		root.AddChild(node)

		bodies[i] = body{
			node:      node,
			radius:    radius,
			mass:      mass,
			vx:        (rand.Float64() - 0.5) * 2,
			vy:        0,
			baseColor: color,
		}
	}

	scene.SetUpdateFunc(func() error {
		// Apply gravity and damping
		for i := range bodies {
			b := &bodies[i]
			b.vy += gravity * b.mass
			b.vx *= damping
			b.vy *= damping

			// Clamp velocity to prevent tunneling
			b.vx = clamp(b.vx, -maxVel, maxVel)
			b.vy = clamp(b.vy, -maxVel, maxVel)

			b.node.SetPosition(b.node.X()+b.vx, b.node.Y()+b.vy)
		}

		// Multiple solver passes to resolve overlaps without jitter
		for pass := 0; pass < solverPasses; pass++ {
			// Wall/floor/ceiling collisions
			for i := range bodies {
				b := &bodies[i]
				x, y, r := b.node.X(), b.node.Y(), b.radius

				if x-r < 0 {
					b.node.SetX(r)
					b.vx = math.Abs(b.vx) * restitution
				} else if x+r > screenW {
					b.node.SetX(screenW - r)
					b.vx = -math.Abs(b.vx) * restitution
				}

				if y-r < 0 {
					b.node.SetY(r)
					b.vy = math.Abs(b.vy) * restitution
				} else if y+r > screenH {
					b.node.SetY(screenH - r)
					b.vy = -math.Abs(b.vy) * restitution
				}
			}

			// Circle-circle collision resolution
			for i := 0; i < len(bodies); i++ {
				a := &bodies[i]
				for j := i + 1; j < len(bodies); j++ {
					b := &bodies[j]

					dx := b.node.X() - a.node.X()
					dy := b.node.Y() - a.node.Y()
					distSq := dx*dx + dy*dy
					minDist := a.radius + b.radius

					if distSq >= minDist*minDist || distSq < 0.001 {
						continue
					}

					dist := math.Sqrt(distSq)
					nx := dx / dist
					ny := dy / dist

					// Positional correction  -  push apart proportional to mass
					overlap := minDist - dist
					totalMass := a.mass + b.mass
					a.node.SetPosition(
						a.node.X()-nx*overlap*(b.mass/totalMass),
						a.node.Y()-ny*overlap*(b.mass/totalMass),
					)
					b.node.SetPosition(
						b.node.X()+nx*overlap*(a.mass/totalMass),
						b.node.Y()+ny*overlap*(a.mass/totalMass),
					)

					// Only apply velocity impulse on the first pass
					if pass > 0 {
						continue
					}

					// Relative velocity along collision normal
					dvx := a.vx - b.vx
					dvy := a.vy - b.vy
					dvn := dvx*nx + dvy*ny

					if dvn > 0 {
						impulse := (1 + restitution) * dvn / totalMass
						a.vx -= impulse * b.mass * nx
						a.vy -= impulse * b.mass * ny
						b.vx += impulse * a.mass * nx
						b.vy += impulse * a.mass * ny
					}
				}
			}
		}

		// Animate flash and scale punch
		for i := range bodies {
			b := &bodies[i]
			if b.flashTimer > 0 {
				b.flashTimer--
				t := float64(b.flashTimer) / flashFrames
				b.node.SetColor(willow.RGB(
					b.baseColor.R()+(1-b.baseColor.R())*t,
					b.baseColor.G()+(1-b.baseColor.G())*t,
					b.baseColor.B()+(1-b.baseColor.B())*t,
				))
				scale := 1.0 + 0.4*t*t
				b.node.SetScale(scale, scale)
			}

			b.node.Invalidate()
		}

		return nil
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:   "Willow  -  Physics Shapes",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// explode applies a radial blast from the clicked body, flinging nearby shapes
// outward with force that falls off with distance. Lighter shapes fly further.
func explode(bodies []body, src int) {
	cx := bodies[src].node.X()
	cy := bodies[src].node.Y()

	bodies[src].vy = -blastForce * (1.0 / bodies[src].mass)
	bodies[src].vx += (rand.Float64() - 0.5) * 10
	bodies[src].flashTimer = flashFrames

	for i := range bodies {
		if i == src {
			continue
		}
		b := &bodies[i]
		dx := b.node.X() - cx
		dy := b.node.Y() - cy
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > blastRadius || dist < 0.1 {
			continue
		}

		strength := blastForce * (1.0 - dist/blastRadius) / b.mass
		nx := dx / dist
		ny := dy / dist

		b.vx += nx * strength
		b.vy += (ny - 0.5) * strength
		b.flashTimer = int(float64(flashFrames) * (1.0 - dist/blastRadius))
	}
}
