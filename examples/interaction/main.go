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
	scene.ClearColor = willow.Color{R: 0.137, G: 0.118, B: 0.176, A: 1} // dark purple

	colors := []willow.Color{
		{R: 0.9, G: 0.3, B: 0.3, A: 1}, // red
		{R: 0.3, G: 0.7, B: 0.9, A: 1}, // blue
		{R: 0.3, G: 0.9, B: 0.5, A: 1}, // green
	}

	altColors := []willow.Color{
		{R: 1.0, G: 0.7, B: 0.2, A: 1}, // orange
		{R: 0.8, G: 0.3, B: 0.9, A: 1}, // purple
		{R: 0.9, G: 0.9, B: 0.3, A: 1}, // yellow
	}

	for i, c := range colors {
		box := makeBox(fmt.Sprintf("box%d", i), c, altColors[i])
		box.X = float64(120 + i*160)
		box.Y = 200
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
			node.Color = alt
		} else {
			node.Color = primary
		}
		current = !current
	})

	// Move on drag.
	node.OnDrag(func(ctx willow.DragContext) {
		node.SetPosition(node.X+ctx.DeltaX, node.Y+ctx.DeltaY)
	})

	return node
}
