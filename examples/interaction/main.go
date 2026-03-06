// Interaction demonstrates draggable colored rectangles with click callbacks
// using willow.Run for a minimal game loop.
// Click a rectangle to change its color; drag any rectangle to move it.
// No external assets are required.
package main

import (
	"fmt"
	"log"

	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Interaction Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	boxSize     = 60
)

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.137, 0.118, 0.176) // dark purple

	colors := []willow.Color{
		willow.RGB(0.9, 0.3, 0.3), // red
		willow.RGB(0.3, 0.7, 0.9), // blue
		willow.RGB(0.3, 0.9, 0.5), // green
	}

	altColors := []willow.Color{
		willow.RGB(1.0, 0.7, 0.2), // orange
		willow.RGB(0.8, 0.3, 0.9), // purple
		willow.RGB(0.9, 0.9, 0.3), // yellow
	}

	for i, c := range colors {
		box := makeBox(fmt.Sprintf("box%d", i), c, altColors[i])
		box.SetPosition(float64(120+i*160), 200)
		scene.Root().AddChild(box)
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

// makeBox creates a draggable, clickable sprite node with a solid color.
func makeBox(name string, primary, alt willow.Color) *willow.Node {
	node := willow.NewRect(name, boxSize, boxSize, primary)
	// Toggle color on click.
	current := true
	node.OnClick(func(ctx willow.ClickContext) {
		if current {
			node.SetColor(alt)
		} else {
			node.SetColor(primary)
		}
		current = !current
	})

	// Move on drag.
	node.OnDrag(func(ctx willow.DragContext) {
		node.SetPosition(node.X()+ctx.DeltaX, node.Y()+ctx.DeltaY)
	})

	return node
}
