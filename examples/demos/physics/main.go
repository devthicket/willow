// physics spawns shapes that fall and collide using Willow's built-in
// Physics* API (a thin shim over jakecoffman/cp/v2). No manual cp.Space,
// no manual Step, no per-body SetPosition/SetRotation write-back -
// EnablePhysics on the subtree and SetBody on each node is all it takes.
//
// See examples/demos/physics_handrolled for the same scene written without
// the shim, as a reference and visual-parity regression target.
//
// Click a shape to fling it with a radial blast.
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

	// Shared seed: physics_handrolled and physics use the same PCG seed so
	// their initial configurations match for visual-parity comparison.
	rngSeed = 0x57494c4c4f57000d
)

type body struct {
	node       *willow.Node
	radius     float64
	baseColor  willow.Color
	flashTimer int
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	rng := rand.New(rand.NewPCG(rngSeed, rngSeed))

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.06, 0.06, 0.09)
	root := scene.Root

	root.EnablePhysics(willow.PhysicsConfig{
		Gravity:    cp.Vector{X: 0, Y: 900},
		Iterations: 10,
	})

	walls := [][2]cp.Vector{
		{{X: 0, Y: 0}, {X: screenW, Y: 0}},
		{{X: 0, Y: screenH}, {X: screenW, Y: screenH}},
		{{X: 0, Y: 0}, {X: 0, Y: screenH}},
		{{X: screenW, Y: 0}, {X: screenW, Y: screenH}},
	}
	for _, w := range walls {
		n := willow.NewContainer("wall")
		root.AddChild(n)
		n.SetBody(willow.PhysicsStatic{
			Shape:      willow.PhysicsSegment{A: w[0], B: w[1]},
			Friction:   0.8,
			Elasticity: 0.5,
		})
	}

	bodies := make([]body, shapeCount)
	for i := range bodies {
		radius := 25.0 + rng.Float64()*15.0

		color := willow.RGB(
			0.3+rng.Float64()*0.7,
			0.3+rng.Float64()*0.7,
			0.3+rng.Float64()*0.7,
		)

		var node *willow.Node
		switch rng.IntN(5) {
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

		x := radius + rng.Float64()*(screenW-2*radius)
		y := radius + rng.Float64()*(screenH/2-radius)
		node.SetPosition(x, y)
		root.AddChild(node)

		node.SetBody(willow.PhysicsDynamic{
			Shape:      willow.PhysicsCircle{Radius: radius},
			Mass:       radius * radius * 0.005,
			Friction:   0.4,
			Elasticity: 0.6,
		})
		node.GetBody().SetVelocityVector(cp.Vector{X: (rng.Float64() - 0.5) * 80, Y: 0})

		idx := i
		node.OnClick(func(ctx willow.ClickContext) {
			explode(bodies[:], idx, rng)
		})

		bodies[i] = body{
			node:      node,
			radius:    radius,
			baseColor: color,
		}
	}

	scene.SetUpdateFunc(func() error {
		// No manual space.Step, no per-body SetPosition/SetRotation.
		// The physics tick + write-back runs inside Scene.Update.
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
		}
		return nil
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:        "Willow  -  Physics",
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

func explode(bodies []body, src int, rng *rand.Rand) {
	srcBody := bodies[src].node.GetBody()
	srcPos := srcBody.Position()

	srcBody.ApplyImpulseAtLocalPoint(
		cp.Vector{X: (rng.Float64() - 0.5) * 200, Y: -blastForce},
		cp.Vector{},
	)
	bodies[src].flashTimer = flashFrames

	for i := range bodies {
		if i == src {
			continue
		}
		b := &bodies[i]
		bb := b.node.GetBody()
		pos := bb.Position()
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

		bb.ApplyImpulseAtLocalPoint(
			cp.Vector{X: nx * strength, Y: (ny - 0.5) * strength},
			cp.Vector{},
		)
		b.flashTimer = int(float64(flashFrames) * falloff)
	}
}
