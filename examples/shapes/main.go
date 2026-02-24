// Shapes demonstrates scene graph hierarchy with polygons and containers.
// A parent container rotates while child shapes inherit the transform.
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Shapes Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480

	numTriangles = 80
	numSquares   = 40
	numPentagons = 20
)

type rotator struct {
	container *willow.Node
	angle     float64
}

func (r *rotator) update() error {
	r.angle += 0.01
	r.container.SetRotation(r.angle)
	return nil
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.098, G: 0.098, B: 0.137, A: 1}

	// Parent container at screen center, rotating around its own origin.
	container := willow.NewContainer("group")
	container.X = screenW / 2
	container.Y = screenH / 2
	container.Interactable = true
	scene.Root().AddChild(container)

	// Initial spawn.
	spawnShapes(container)

	scene.Root().HitShape = willow.HitRect{Width: screenW, Height: screenH}
	scene.Root().OnClick = func(ctx willow.ClickContext) {
		spawnShapes(container)
	}

	r := &rotator{container: container}
	scene.SetUpdateFunc(r.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}

// spawnShapes detaches all children from the container and creates a new set of shapes.
func spawnShapes(container *willow.Node) {
	container.RemoveChildren()

	// Triangles.
	for range numTriangles {
		tri := willow.NewPolygon("triangle", []willow.Vec2{
			{X: 0, Y: -40},
			{X: 35, Y: 30},
			{X: -35, Y: 30},
		})
		tri.Color = willow.Color{R: rand.Float64(), G: rand.Float64(), B: rand.Float64(), A: 1}
		tri.X = (rand.Float64() - 0.5) * screenW * 2
		tri.Y = (rand.Float64() - 0.5) * screenH * 2
		container.AddChild(tri)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.1
		tri.OnUpdate = func(dt float64) {
			tri.Rotation += spin
		}
	}

	// Squares.
	for range numSquares {
		sq := willow.NewPolygon("square", []willow.Vec2{
			{X: -25, Y: -25},
			{X: 25, Y: -25},
			{X: 25, Y: 25},
			{X: -25, Y: 25},
		})
		sq.Color = willow.Color{R: rand.Float64(), G: rand.Float64(), B: rand.Float64(), A: 1}
		sq.X = (rand.Float64() - 0.5) * screenW * 2
		sq.Y = (rand.Float64() - 0.5) * screenH * 2
		container.AddChild(sq)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.05
		sq.OnUpdate = func(dt float64) {
			sq.Rotation += spin
		}
	}

	// Pentagons.
	for range numPentagons {
		pent := willow.NewPolygon("pentagon", pentagonPoints(30))
		pent.Color = willow.Color{R: rand.Float64(), G: rand.Float64(), B: rand.Float64(), A: 1}
		pent.X = (rand.Float64() - 0.5) * screenW * 2
		pent.Y = (rand.Float64() - 0.5) * screenH * 2
		container.AddChild(pent)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.08
		pent.OnUpdate = func(dt float64) {
			pent.Rotation += spin
		}

		// Small orbiting diamond on each pentagon.
		diamond := willow.NewPolygon("diamond", []willow.Vec2{
			{X: 0, Y: -12},
			{X: 10, Y: 0},
			{X: 0, Y: 12},
			{X: -10, Y: 0},
		})
		diamond.Color = willow.Color{R: rand.Float64(), G: rand.Float64(), B: rand.Float64(), A: 1}
		diamond.X = 45
		pent.AddChild(diamond)
	}
}

// pentagonPoints returns a regular pentagon centered at origin with the given radius.
func pentagonPoints(radius float64) []willow.Vec2 {
	pts := make([]willow.Vec2, 5)
	for i := range pts {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		pts[i] = willow.Vec2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		}
	}
	return pts
}
