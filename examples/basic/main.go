// Basic demonstrates a minimal willow scene with a colored sprite bouncing
// around the window. Uses willow.Run for a zero-boilerplate game loop.
// No external assets are required.
package main

import (
	"log"

	"github.com/devthicket/willow"
)

const (
	windowTitle = "Willow  -  Basic Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	spriteW     = 40
	spriteH     = 40
)

type bouncer struct {
	node   *willow.Node
	dx, dy float64
}

func (b *bouncer) update() error {
	b.node.SetPosition(b.node.X()+b.dx, b.node.Y()+b.dy)

	if b.node.X() < 0 || b.node.X()+spriteW > screenW {
		b.dx = -b.dx
	}
	if b.node.Y() < 0 || b.node.Y()+spriteH > screenH {
		b.dy = -b.dy
	}
	return nil
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.118, 0.118, 0.157)

	sprite := willow.NewRect("box", spriteW, spriteH, willow.RGB(80.0/255.0, 180.0/255.0, 1))
	sprite.SetPosition(100, 100)
	scene.Root.AddChild(sprite)

	b := &bouncer{node: sprite, dx: 2, dy: 1.5}
	scene.SetUpdateFunc(b.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}
