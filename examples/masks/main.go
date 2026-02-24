// Masks demonstrates three node-masking techniques in Willow across three
// equal full-height panels:
//   - Star: a rotating, pulsing star polygon mask clips a rainbow tile grid.
//   - Cursor: move the mouse over the centre panel  -  the whelp sprite's
//     alpha channel stamps its shape into bold horizontal stripes.
//   - Scroll: a full-panel rect mask clips smoothly-scrolling colour bars.
//
// NOTE: the mask root node's own transform is ignored by the renderer; only
// the transforms of its *children* are applied.  All animated masks therefore
// use a plain container as the root and put the actual shape one level below.
package main

import (
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Masks"
	showFPS     = true
	screenW     = 900
	screenH     = 480
	panelW      = 300.0 // screenW / 3
)

type demo struct {
	time          float64
	starShape     *willow.Node // child of mask container  -  rotation/scale applied here
	whelpChild    *willow.Node // child of mask container  -  X/Y follows cursor
	scrollContent *willow.Node // Y-scrolled each frame
	scrollY       float64
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())
	d.time += dt

	// Panel 0: star mask  -  rotate and pulse the shape child.
	d.starShape.Rotation = d.time * 0.9
	s := 1.0 + 0.20*math.Sin(d.time*1.8)
	d.starShape.ScaleX = s
	d.starShape.ScaleY = s
	d.starShape.Invalidate()

	// Panel 1: whelp mask  -  move the sprite child to the cursor.
	// p1 is at world (panelW, 0); panel-local cursor = (mx-panelW, my).
	mx, my := ebiten.CursorPosition()
	d.whelpChild.SetPosition(float64(mx)-panelW-128, float64(my)-128)

	// Panel 2: smooth scroll  -  move sub-container up, reset seamlessly.
	const (
		barH  = 60.0
		nBars = 8
	)
	d.scrollY += 60 * dt
	if d.scrollY >= barH*nBars {
		d.scrollY -= barH * nBars
	}
	d.scrollContent.SetY(-d.scrollY)

	return nil
}

func main() {
	f, err := os.Open("examples/_assets/whelp.png")
	if err != nil {
		log.Fatalf("open whelp.png: %v", err)
	}
	defer f.Close()

	whelpImg, _, err := ebitenutil.NewImageFromReader(f)
	if err != nil {
		log.Fatalf("decode whelp.png: %v", err)
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.09, G: 0.09, B: 0.14, A: 1}

	d := &demo{}

	// ---- Panel 0: rotating star mask over a rainbow tile grid ----------------------------------
	// Container at world (0, 0); content fills local (0,0)→(panelW, screenH).

	p0 := willow.NewContainer("star-panel")
	// p0.X = 0, p0.Y = 0 (default)

	// 15×24 grid of 19 px tiles on a 20 px step, filling the panel.
	for row := range 24 {
		for col := range 15 {
			tile := willow.NewSprite("t", willow.TextureRegion{})
			tile.ScaleX = 19
			tile.ScaleY = 19
			tile.X = float64(col) * 20
			tile.Y = float64(row) * 20
			tile.Color = hsvToColor(float64(col+row)/37.0, 0.82, 0.92)
			p0.AddChild(tile)
		}
	}

	// Mask root is a container; star shape is its child so transforms apply.
	maskRoot0 := willow.NewContainer("mask-root-0")
	starShape := willow.NewPolygon("star", starPoints(140, 56, 5))
	starShape.X = panelW / 2  // 150  -  centre of panel
	starShape.Y = screenH / 2 // 240
	starShape.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	maskRoot0.AddChild(starShape)
	p0.SetMask(maskRoot0)

	// Clip container: rect mask hard-clips the entire panel to its bounds.
	clip0 := willow.NewContainer("clip-0")
	clip0Rect := willow.NewPolygon("clip-rect-0", []willow.Vec2{
		{X: 0, Y: 0}, {X: panelW, Y: 0},
		{X: panelW, Y: screenH}, {X: 0, Y: screenH},
	})
	clip0Rect.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	clip0.SetMask(clip0Rect)
	clip0.AddChild(p0)
	scene.Root().AddChild(clip0)
	d.starShape = starShape

	// ---- Panel 1: whelp-alpha mask over bold stripes, follows cursor ----------------------
	// Container at world (panelW, 0); content fills local (0,0)→(panelW,screenH).

	p1 := willow.NewContainer("cursor-panel")
	// p1.X = 0  -  clip1 carries the panelW world offset.

	// Eight bold horizontal rainbow stripes filling the full panel height.
	// Warm palette: crimson → red → orange-red → orange → amber → gold → tan → sienna.
	stripeColors := []willow.Color{
		{R: 0.80, G: 0.08, B: 0.05, A: 1},
		{R: 0.97, G: 0.18, B: 0.08, A: 1},
		{R: 1.00, G: 0.38, B: 0.04, A: 1},
		{R: 1.00, G: 0.58, B: 0.00, A: 1},
		{R: 1.00, G: 0.74, B: 0.00, A: 1},
		{R: 0.94, G: 0.84, B: 0.04, A: 1},
		{R: 0.88, G: 0.54, B: 0.18, A: 1},
		{R: 0.72, G: 0.28, B: 0.10, A: 1},
	}
	const stripeH = 60.0
	for i, c := range stripeColors {
		s := willow.NewSprite("stripe", willow.TextureRegion{})
		s.ScaleX = panelW
		s.ScaleY = stripeH
		s.X = 0
		s.Y = float64(i) * stripeH
		s.Color = c
		p1.AddChild(s)
	}

	// Mask: whelp sprite (scaled 2×) follows the cursor.
	maskRoot1 := willow.NewContainer("mask-root-1")
	whelpChild := willow.NewSprite("whelp-mask", willow.TextureRegion{})
	whelpChild.SetCustomImage(whelpImg)
	whelpChild.ScaleX = 2 // 128 px → 256 px
	whelpChild.ScaleY = 2
	maskRoot1.AddChild(whelpChild)
	p1.SetMask(maskRoot1)

	// Clip container: carries the panel's world offset and hard-clips to panel bounds.
	clip1 := willow.NewContainer("clip-1")
	clip1.X = panelW // 300
	clip1Rect := willow.NewPolygon("clip-rect-1", []willow.Vec2{
		{X: 0, Y: 0}, {X: panelW, Y: 0},
		{X: panelW, Y: screenH}, {X: 0, Y: screenH},
	})
	clip1Rect.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	clip1.SetMask(clip1Rect)
	clip1.AddChild(p1)
	scene.Root().AddChild(clip1)
	d.whelpChild = whelpChild

	// ---- Panel 2: scrolling bars with a static whelp shape erased from them --------
	// clip2 carries the world offset and hard-clips; p2 holds the erase mask.

	p2 := willow.NewContainer("scroll-panel")
	// p2.X = 0  -  clip2 handles the world offset.

	scrollContent := willow.NewContainer("scroll-content")
	p2.AddChild(scrollContent)

	// Cool palette: navy → royal blue → sky blue → cyan → teal → indigo → violet → emerald.
	barColors := []willow.Color{
		{R: 0.05, G: 0.10, B: 0.62, A: 1},
		{R: 0.10, G: 0.32, B: 0.88, A: 1},
		{R: 0.14, G: 0.58, B: 0.94, A: 1},
		{R: 0.00, G: 0.76, B: 0.86, A: 1},
		{R: 0.04, G: 0.64, B: 0.58, A: 1},
		{R: 0.28, G: 0.18, B: 0.80, A: 1},
		{R: 0.54, G: 0.14, B: 0.84, A: 1},
		{R: 0.08, G: 0.72, B: 0.42, A: 1},
	}
	// Two copies for a seamless loop.
	const barH = 60.0
	for copy := range 2 {
		for i, c := range barColors {
			bar := willow.NewSprite("bar", willow.TextureRegion{})
			bar.ScaleX = panelW
			bar.ScaleY = barH - 2 // 2 px gap between bars
			bar.X = 0
			bar.Y = float64(copy*len(barColors)+i) * barH
			bar.Color = c
			scrollContent.AddChild(bar)
		}
	}

	// Erase mask: a white rect (show everything) with the whelp shape stamped out
	// using BlendErase, so the whelp silhouette punches a permanent hole in the bars.
	eraseRoot := willow.NewContainer("erase-root")

	eraseRect := willow.NewPolygon("erase-rect", []willow.Vec2{
		{X: 0, Y: 0}, {X: panelW, Y: 0},
		{X: panelW, Y: screenH}, {X: 0, Y: screenH},
	})
	eraseRect.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	eraseRoot.AddChild(eraseRect)

	whelpStatic := willow.NewSprite("whelp-static", willow.TextureRegion{})
	whelpStatic.SetCustomImage(whelpImg)
	whelpStatic.ScaleX = 2 // 128 px → 256 px
	whelpStatic.ScaleY = 2
	whelpStatic.X = (panelW - 256) / 2        // centred horizontally
	whelpStatic.Y = (screenH - 256) / 2       // centred vertically
	whelpStatic.BlendMode = willow.BlendErase // punch the alpha through the white rect
	eraseRoot.AddChild(whelpStatic)

	p2.SetMask(eraseRoot)

	// Clip container: carries the world offset and hard-clips to panel bounds.
	clip2 := willow.NewContainer("clip-2")
	clip2.X = panelW * 2 // 600
	clip2Rect := willow.NewPolygon("clip-rect-2", []willow.Vec2{
		{X: 0, Y: 0}, {X: panelW, Y: 0},
		{X: panelW, Y: screenH}, {X: 0, Y: screenH},
	})
	clip2Rect.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	clip2.SetMask(clip2Rect)
	clip2.AddChild(p2)
	scene.Root().AddChild(clip2)
	d.scrollContent = scrollContent

	// ---- Dividers ----------------------------------------------------------------------------------------------------------------------------

	for _, divX := range []float64{panelW, panelW * 2} {
		div := willow.NewSprite("div", willow.TextureRegion{})
		div.ScaleX = 2
		div.ScaleY = screenH
		div.X = divX - 1
		div.Color = willow.Color{R: 1, G: 1, B: 1, A: 0.25}
		div.ZIndex = 10
		scene.Root().AddChild(div)
	}

	// ---- Labels --------------------------------------------------------------------------------------------------------------------------------

	for i, lbl := range []string{"Star Mask", "Cursor Mask", "Erase Mask"} {
		x := float64(i)*panelW + panelW/2 - float64(len(lbl)*6)/2
		n := makeLabel(lbl)
		n.X = x
		n.Y = 8
		n.ZIndex = 10
		scene.Root().AddChild(n)
	}

	const hintText = "move cursor over centre panel  |  whelp erased from right panel"
	hint := makeLabel(hintText)
	hint.X = float64(screenW)/2 - float64(len(hintText)*6)/2
	hint.Y = float64(screenH) - 18
	hint.ZIndex = 10
	scene.Root().AddChild(hint)

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

// starPoints returns vertices for a numPoints-pointed star.
// outerR is the tip radius; innerR is the valley radius.
func starPoints(outerR, innerR float64, numPoints int) []willow.Vec2 {
	pts := make([]willow.Vec2, numPoints*2)
	for i := range pts {
		a := -math.Pi/2 + float64(i)*math.Pi/float64(numPoints)
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		pts[i] = willow.Vec2{X: math.Cos(a) * r, Y: math.Sin(a) * r}
	}
	return pts
}

// hsvToColor converts HSV (all in [0, 1]) to willow.Color.
func hsvToColor(h, s, v float64) willow.Color {
	h6 := h * 6
	i := int(h6)
	f := h6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch i % 6 {
	case 0:
		return willow.Color{R: v, G: t, B: p, A: 1}
	case 1:
		return willow.Color{R: q, G: v, B: p, A: 1}
	case 2:
		return willow.Color{R: p, G: v, B: t, A: 1}
	case 3:
		return willow.Color{R: p, G: q, B: v, A: 1}
	case 4:
		return willow.Color{R: t, G: p, B: v, A: 1}
	default:
		return willow.Color{R: v, G: p, B: q, A: 1}
	}
}

// makeLabel renders s into a small sprite using the Ebitengine debug font.
func makeLabel(s string) *willow.Node {
	img := ebiten.NewImage(len(s)*6+4, 16)
	ebitenutil.DebugPrint(img, s)
	n := willow.NewSprite("label", willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}
