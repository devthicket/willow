// Outline demonstrates outline and inline filters applied to a sprite with
// alpha transparency (whelp.png). Three copies are shown side by side:
// left has an outline, center is unfiltered, and right has an inline.
// Click anywhere to cycle the outline/inline color.
package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Outline Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
)

var outlineColors = []willow.Color{
	{R: 1, G: 0.85, B: 0.2, A: 1}, // gold
	{R: 0.3, G: 1, B: 0.5, A: 1},  // green
	{R: 0.4, G: 0.7, B: 1, A: 1},  // sky blue
	{R: 1, G: 0.3, B: 0.4, A: 1},  // red
	{R: 0.8, G: 0.4, B: 1, A: 1},  // purple
	{R: 1, G: 1, B: 1, A: 1},      // white
}

func main() {
	f, err := os.Open("examples/_assets/whelp.png")
	if err != nil {
		log.Fatalf("failed to open whelp.png: %v", err)
	}
	defer f.Close()

	whelpImg, _, err := ebitenutil.NewImageFromReader(f)
	if err != nil {
		log.Fatalf("failed to load whelp image: %v", err)
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.08, G: 0.08, B: 0.12, A: 1}

	colorIdx := 0
	c := outlineColors[colorIdx]

	outline := willow.NewOutlineFilter(3, c)
	inline := willow.NewPixelPerfectInlineFilter(c)

	type column struct {
		label   string
		filters []willow.Filter
	}
	cols := []column{
		{"Outline", []willow.Filter{outline}},
		{"Original", nil},
		{"Inline", []willow.Filter{inline}},
	}

	whelpW := float64(whelpImg.Bounds().Dx())
	whelpH := float64(whelpImg.Bounds().Dy())
	labelH := 24.0 // gap (8) + label height (16)
	cy := (float64(screenH) - labelH) / 2

	var filteredNodes []*willow.Node
	for i, col := range cols {
		x := screenW * float64(i+1) / float64(len(cols)+1)

		sp := willow.NewSprite("whelp", willow.TextureRegion{})
		sp.SetCustomImage(whelpImg)
		sp.Filters = col.filters
		sp.X = x
		sp.Y = cy
		sp.PivotX = whelpW / 2
		sp.PivotY = whelpH / 2
		scene.Root().AddChild(sp)
		if col.filters != nil {
			filteredNodes = append(filteredNodes, sp)
		}

		label := makeLabel(col.label)
		label.X = x - float64(len(col.label)*6)/2
		label.Y = cy + whelpH/2 + 8
		scene.Root().AddChild(label)
	}

	// Click to cycle outline/inline color.
	scene.OnBackgroundClick(func(ctx willow.ClickContext) {
		colorIdx = (colorIdx + 1) % len(outlineColors)
		c = outlineColors[colorIdx]
		outline.Color = c
		inline.Color = c
		for _, n := range filteredNodes {
			n.Invalidate()
		}
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}

// makeLabel pre-renders a text string into a sprite using Ebitengine's debug font.
func makeLabel(s string) *willow.Node {
	w := len(s)*6 + 2
	h := 16
	img := ebiten.NewImage(w, h)
	ebitenutil.DebugPrint(img, s)

	n := willow.NewSprite("label-"+s, willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}
