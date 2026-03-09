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
	willow.RGB(1, 0.85, 0.2), // gold
	willow.RGB(0.3, 1, 0.5),  // green
	willow.RGB(0.4, 0.7, 1),  // sky blue
	willow.RGB(1, 0.3, 0.4),  // red
	willow.RGB(0.8, 0.4, 1),  // purple
	willow.RGB(1, 1, 1),      // white
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
	scene.ClearColor = willow.RGB(0.08, 0.08, 0.12)

	colorIdx := 0
	c := outlineColors[colorIdx]

	outline := willow.NewOutlineFilter(3, c)
	inline := willow.NewPixelPerfectInlineFilter(c)

	type column struct {
		label   string
		filters []any
	}
	cols := []column{
		{"Outline", []any{outline}},
		{"Original", nil},
		{"Inline", []any{inline}},
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
		sp.SetPosition(x, cy)
		sp.SetPivot(whelpW/2, whelpH/2)
		scene.Root.AddChild(sp)
		if col.filters != nil {
			filteredNodes = append(filteredNodes, sp)
		}

		label := makeLabel(col.label)
		label.SetPosition(x-float64(len(col.label)*6)/2, cy+whelpH/2+8)
		scene.Root.AddChild(label)
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
