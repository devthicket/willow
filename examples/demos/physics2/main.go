// physics2 spawns shapes that fall and collide using the Chipmunk2D-based
// jakecoffman/cp/v2 library. Click a shape to fling it with a radial blast.
package main

import (
	"flag"
	"log"
	"math"
	"math/rand/v2"

	"github.com/devthicket/willow"
	"github.com/jakecoffman/cp/v2"
)

const (
	screenW    = 1280
	screenH    = 720
	shapeCount = 100

	blastRadius = 350.0
	blastForce  = 1200.0

	flashFrames = 12
)

type body struct {
	node       *willow.Node
	body       *cp.Body
	radius     float64
	baseColor  willow.Color
	flashTimer int
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.06, 0.06, 0.09)
	root := scene.Root

	space := cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0, Y: 900})
	space.Iterations = 10

	// Static walls: floor, ceiling, left, right.
	walls := [][2]cp.Vector{
		{{X: 0, Y: 0}, {X: screenW, Y: 0}},
		{{X: 0, Y: screenH}, {X: screenW, Y: screenH}},
		{{X: 0, Y: 0}, {X: 0, Y: screenH}},
		{{X: screenW, Y: 0}, {X: screenW, Y: screenH}},
	}
	for _, w := range walls {
		seg := cp.NewSegment(space.StaticBody, w[0], w[1], 0)
		seg.SetElasticity(0.5)
		seg.SetFriction(0.8)
		space.AddShape(seg)
	}

	bodies := make([]body, shapeCount)
	for i := range bodies {
		radius := 25.0 + rand.Float64()*15.0
		mass := radius * radius * 0.005

		color := willow.RGB(
			0.3+rand.Float64()*0.7,
			0.3+rand.Float64()*0.7,
			0.3+rand.Float64()*0.7,
		)

		var node *willow.Node
		switch rand.IntN(5) {
		case 0:
			node = willow.NewRegularPolygon("circle", 24, radius)
		case 1:
			node = willow.NewRegularPolygon("square", 4, radius)
		case 2:
			node = willow.NewRegularPolygon("triangle", 3, radius)
		case 3:
			node = willow.NewRegularPolygon("pentagon", 5, radius)
		case 4:
			node = willow.NewRegularPolygon("hexagon", 6, radius)
		}
		node.SetColor(color)
		node.HitShape = willow.HitCircle{Radius: radius}

		x := radius + rand.Float64()*(screenW-2*radius)
		y := radius + rand.Float64()*(screenH/2-radius)
		node.SetPosition(x, y)

		cpBody := cp.NewBody(mass, cp.MomentForCircle(mass, 0, radius, cp.Vector{}))
		cpBody.SetPosition(cp.Vector{X: x, Y: y})
		cpBody.SetVelocityVector(cp.Vector{X: (rand.Float64() - 0.5) * 80, Y: 0})
		space.AddBody(cpBody)

		shape := cp.NewCircle(cpBody, radius, cp.Vector{})
		shape.SetElasticity(0.6)
		shape.SetFriction(0.4)
		space.AddShape(shape)

		idx := i
		node.OnClick(func(ctx willow.ClickContext) {
			explode(bodies[:], idx)
		})
		root.AddChild(node)

		bodies[i] = body{
			node:      node,
			body:      cpBody,
			radius:    radius,
			baseColor: color,
		}
	}

	scene.SetUpdateFunc(func() error {
		space.Step(1.0 / 60.0)

		for i := range bodies {
			b := &bodies[i]
			pos := b.body.Position()
			b.node.SetPosition(pos.X, pos.Y)
			b.node.SetRotation(b.body.Angle())

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
		Title:        "Willow  -  Physics (cp/Chipmunk2D)",
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

func explode(bodies []body, src int) {
	srcPos := bodies[src].body.Position()

	bodies[src].body.ApplyImpulseAtLocalPoint(
		cp.Vector{X: (rand.Float64() - 0.5) * 200, Y: -blastForce},
		cp.Vector{},
	)
	bodies[src].flashTimer = flashFrames

	for i := range bodies {
		if i == src {
			continue
		}
		b := &bodies[i]
		pos := b.body.Position()
		dx := pos.X - srcPos.X
		dy := pos.Y - srcPos.Y
		distSq := dx*dx + dy*dy
		if distSq > blastRadius*blastRadius || distSq < 0.01 {
			continue
		}
		dist := math.Sqrt(distSq)
		falloff := 1.0 - dist/blastRadius
		strength := blastForce * falloff
		nx := dx / dist
		ny := dy / dist

		b.body.ApplyImpulseAtLocalPoint(
			cp.Vector{X: nx * strength, Y: (ny - 0.5) * strength},
			cp.Vector{},
		)
		b.flashTimer = int(float64(flashFrames) * falloff)
	}
}