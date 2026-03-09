// Rope demonstrates the Rope mesh helper by connecting two draggable nodes
// with a textured rope that sags under gravity. A third "anchor" node orbits
// the midpoint automatically, pulling the rope into a curved path.
// Drag the red or blue endpoint to reshape the rope interactively.
// No external assets are required.
package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Rope Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	handleSize  = 20
	ropeWidth   = 8
)

type demo struct {
	scene   *willow.Scene
	rope    *willow.Rope
	handleA *willow.Node
	handleB *willow.Node
	orbiter *willow.Node
	start   *willow.Vec2 // bound to rope Start
	end     *willow.Vec2 // bound to rope End
	ctrl    *willow.Vec2 // bound to rope Controls[0]
	time    float64
}

func (d *demo) update() error {
	d.time += 1.0 / float64(ebiten.TPS())

	// Orbiter circles the midpoint of the two handles.
	mx := (d.handleA.X() + d.handleB.X()) / 2
	my := (d.handleA.Y() + d.handleB.Y()) / 2
	orbitR := 80.0
	d.orbiter.SetPosition(mx+math.Cos(d.time*1.2)*orbitR, my+math.Sin(d.time*1.2)*orbitR*0.6)
	d.orbiter.Invalidate()

	// Update the bound Vec2s  -  rope.Update() reads these by reference.
	d.start.X = d.handleA.X()
	d.start.Y = d.handleA.Y()
	d.end.X = d.handleB.X()
	d.end.Y = d.handleB.Y()
	d.ctrl.X = d.orbiter.X()
	d.ctrl.Y = d.orbiter.Y() + 40
	d.rope.Update()
	return nil
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.1, 0.1, 0.15)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Create a solid-color rope texture. The image must be large enough to
	// cover the UV range (SrcX = cumulative path length). 512px covers most
	// rope lengths in this demo. The vertical gradient gives a rounded look.
	const ropeTexW = 512
	ropeImg := ebiten.NewImage(ropeTexW, int(ropeWidth))
	for y := range int(ropeWidth) {
		dist := math.Abs(float64(y)-ropeWidth/2) / (ropeWidth / 2)
		bright := 1.0 - dist*0.4
		r := uint8(bright * 180)
		g := uint8(bright * 140)
		b := uint8(bright * 80)
		for x := range ropeTexW {
			ropeImg.Set(x, y, willow.ColorFromRGBA(r, g, b, 255))
		}
	}

	// Positions bound to the rope  -  mutate these, then call rope.Update().
	start := willow.Vec2{X: 160, Y: 200}
	end := willow.Vec2{X: 480, Y: 200}
	ctrl := willow.Vec2{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}

	// Rope mesh.
	rope := willow.NewRope("rope", ropeImg, nil, willow.RopeConfig{
		Width:     ropeWidth,
		JoinMode:  willow.RopeJoinBevel,
		CurveMode: willow.RopeCurveQuadBezier,
		Segments:  20,
		Start:     &start,
		End:       &end,
		Controls:  [2]*willow.Vec2{&ctrl},
	})
	scene.Root.AddChild(rope.Node())

	// Draggable handle A (red).
	handleA := makeHandle("handleA", willow.RGB(0.9, 0.3, 0.3))
	handleA.SetPosition(start.X, start.Y)
	scene.Root.AddChild(handleA)

	// Draggable handle B (blue).
	handleB := makeHandle("handleB", willow.RGB(0.3, 0.5, 0.9))
	handleB.SetPosition(end.X, end.Y)
	scene.Root.AddChild(handleB)

	// Orbiting anchor (green, not draggable).
	orbiter := willow.NewSprite("orbiter", willow.TextureRegion{})
	orbiter.SetScale(12, 12)
	orbiter.SetPivot(0.5, 0.5)
	orbiter.SetColor(willow.RGB(0.3, 0.9, 0.4))
	scene.Root.AddChild(orbiter)

	d := &demo{
		scene:   scene,
		rope:    rope,
		handleA: handleA,
		handleB: handleB,
		orbiter: orbiter,
		start:   &start,
		end:     &end,
		ctrl:    &ctrl,
	}
	scene.SetUpdateFunc(d.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}

// makeHandle creates a draggable square sprite centered on its position.
func makeHandle(name string, c willow.Color) *willow.Node {
	n := willow.NewSprite(name, willow.TextureRegion{})
	n.SetScale(handleSize, handleSize)
	n.SetPivot(0.5, 0.5)
	n.SetColor(c)
	n.OnDrag(func(ctx willow.DragContext) {
		n.SetPosition(n.X()+ctx.DeltaX, n.Y()+ctx.DeltaY)
		n.Invalidate()
	})

	return n
}
