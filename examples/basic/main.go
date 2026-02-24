// Basic demonstrates a minimal willow scene with a colored sprite bouncing
// around the window. Uses willow.Run for a zero-boilerplate game loop.
// No external assets are required.
package main

import (
	"log"

	"github.com/phanxgames/willow"
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
	b.node.X += b.dx
	b.node.Y += b.dy
	b.node.Invalidate()

	if b.node.X < 0 || b.node.X+spriteW > screenW {
		b.dx = -b.dx
	}
	if b.node.Y < 0 || b.node.Y+spriteH > screenH {
		b.dy = -b.dy
	}
	return nil
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.118, G: 0.118, B: 0.157, A: 1}

	sprite := willow.NewSprite("box", willow.TextureRegion{})
	sprite.ScaleX = spriteW
	sprite.ScaleY = spriteH
	sprite.Color = willow.Color{R: 80.0 / 255.0, G: 180.0 / 255.0, B: 1, A: 1}
	sprite.X = 100
	sprite.Y = 100
	scene.Root().AddChild(sprite)

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
