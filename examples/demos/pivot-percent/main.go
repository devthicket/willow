// Pivot-percent visually proves two behaviors:
//  1. White-pixel sprite size lives in geometry (SetSize), not Scale, so
//     pivots are interpreted in display-pixel space.
//  2. SetPivotPercent(fx, fy) anchors at a fraction of the sized W×H.
//
// Three rectangles all rotate at the same rate:
//   - Left: default pivot (top-left). Sweeps around its corner.
//   - Center: SetPivot(W/2, H/2). Spins in place at its center.
//   - Right: SetPivotPercent(0.5, 0.5). Spins in place — proving the
//     percent form matches the explicit display-pixel pivot.
package main

import (
	"flag"
	"log"

	"github.com/devthicket/willow"
)

const (
	windowTitle = "Willow  -  Pivot Percent"
	screenW     = 640
	screenH     = 360
	rectW       = 80.0
	rectH       = 50.0
)

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.098, 0.098, 0.137)

	y := float64(screenH) / 2

	cornerPivot := makeRect("corner", willow.RGB(0.95, 0.45, 0.45))
	cornerPivot.SetPosition(screenW*0.2, y)
	scene.Root.AddChild(cornerPivot)

	displayPivot := makeRect("display", willow.RGB(0.45, 0.85, 0.55))
	displayPivot.SetPosition(screenW*0.5, y)
	displayPivot.SetPivot(rectW/2, rectH/2)
	scene.Root.AddChild(displayPivot)

	percentPivot := makeRect("percent", willow.RGB(0.55, 0.65, 0.95))
	percentPivot.SetPosition(screenW*0.8, y)
	percentPivot.SetPivotPercent(0.5, 0.5)
	scene.Root.AddChild(percentPivot)

	angle := 0.0
	scene.SetUpdateFunc(func() error {
		angle += 0.02
		cornerPivot.SetRotation(angle)
		displayPivot.SetRotation(angle)
		percentPivot.SetRotation(angle)
		return nil
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:        windowTitle,
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

func makeRect(name string, c willow.Color) *willow.Node {
	n := willow.NewSprite(name, willow.TextureRegion{})
	n.SetSize(rectW, rectH)
	n.SetColor(c)
	return n
}
