// Shapes demonstrates scene graph hierarchy with polygons and containers.
// A parent container rotates while child shapes inherit the transform.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
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
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.098, 0.098, 0.137)

	// Parent container at screen center, rotating around its own origin.
	container := willow.NewContainer("group")
	container.SetPosition(screenW/2, screenH/2)
	container.Interactable = true
	scene.Root.AddChild(container)

	// Initial spawn.
	spawnShapes(container)

	scene.OnBackgroundClick(func(ctx willow.ClickContext) {
		spawnShapes(container)
	})

	r := &rotator{container: container}

	if *autotest != "" {
		scriptData, err := os.ReadFile(*autotest)
		if err != nil {
			log.Fatalf("read test script: %v", err)
		}
		runner, err := willow.LoadTestScript(scriptData)
		if err != nil {
			log.Fatalf("parse test script: %v", err)
		}
		scene.SetTestRunner(runner)
		scene.ScreenshotDir = "screenshots"
		scene.SetUpdateFunc(func() error {
			if err := r.update(); err != nil {
				return err
			}
			if runner.Done() {
				fmt.Println("Autotest complete.")
				return ebiten.Termination
			}
			return nil
		})
	} else {
		scene.SetUpdateFunc(r.update)
	}

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
		tri.SetColor(willow.RGB(rand.Float64(), rand.Float64(), rand.Float64()))
		tri.SetPosition((rand.Float64()-0.5)*screenW*2, (rand.Float64()-0.5)*screenH*2)
		container.AddChild(tri)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.1
		tri.OnUpdate = func(dt float64) {
			tri.SetRotation(tri.Rotation() + spin)
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
		sq.SetColor(willow.RGB(rand.Float64(), rand.Float64(), rand.Float64()))
		sq.SetPosition((rand.Float64()-0.5)*screenW*2, (rand.Float64()-0.5)*screenH*2)
		container.AddChild(sq)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.05
		sq.OnUpdate = func(dt float64) {
			sq.SetRotation(sq.Rotation() + spin)
		}
	}

	// Pentagons.
	for range numPentagons {
		pent := willow.NewRegularPolygon("pentagon", 5, 30)
		pent.SetColor(willow.RGB(rand.Float64(), rand.Float64(), rand.Float64()))
		pent.SetPosition((rand.Float64()-0.5)*screenW*2, (rand.Float64()-0.5)*screenH*2)
		container.AddChild(pent)

		// Independent rotation.
		spin := (rand.Float64() - 0.5) * 0.08
		pent.OnUpdate = func(dt float64) {
			pent.SetRotation(pent.Rotation() + spin)
		}

		// Small orbiting diamond on each pentagon.
		diamond := willow.NewPolygon("diamond", []willow.Vec2{
			{X: 0, Y: -12},
			{X: 10, Y: 0},
			{X: 0, Y: 12},
			{X: -10, Y: 0},
		})
		diamond.SetColor(willow.RGB(rand.Float64(), rand.Float64(), rand.Float64()))
		diamond.SetX(45)
		pent.AddChild(diamond)
	}
}
