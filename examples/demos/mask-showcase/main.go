// Mask Showcase demonstrates Willow's clipping and masking system with a
// single sprite cycling through five dramatic mask effects, one per second.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenW = 320
	screenH = 240
)

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

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
	scene.ClearColor = willow.RGB(0.04, 0.03, 0.08)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Background: warm-to-cool gradient tile grid visible through mask.
	bg := willow.NewContainer("bg")
	bgColors := []willow.Color{
		willow.RGB(0.10, 0.65, 0.30), // green
		willow.RGB(0.08, 0.55, 0.50), // teal
		willow.RGB(0.10, 0.40, 0.70), // blue
	}
	for row := range 12 {
		for col := range 16 {
			// Blend based on diagonal position.
			t := float64(col+row) / float64(16+12-2)
			idx := t * float64(len(bgColors)-1)
			lo := int(idx)
			hi := lo + 1
			if hi >= len(bgColors) {
				hi = len(bgColors) - 1
			}
			frac := idx - float64(lo)
			c0, c1 := bgColors[lo], bgColors[hi]
			r := c0.R()*(1-frac) + c1.R()*frac
			g := c0.G()*(1-frac) + c1.G()*frac
			b := c0.B()*(1-frac) + c1.B()*frac

			tile := willow.NewSprite("t", willow.TextureRegion{})
			tile.SetScale(20, 20)
			tile.SetPosition(float64(col)*20, float64(row)*20)
			tile.SetColor(willow.RGB(r, g, b))
			bg.AddChild(tile)
		}
	}
	scene.Root.AddChild(bg)

	// Main whelp sprite centered and scaled.
	whelp := willow.NewSprite("whelp", willow.TextureRegion{})
	whelp.SetCustomImage(whelpImg)
	whelp.SetScale(1.8, 1.8)
	w := float64(whelpImg.Bounds().Dx()) * 1.8
	h := float64(whelpImg.Bounds().Dy()) * 1.8
	whelp.SetPosition((screenW-w)/2, (screenH-h)/2)
	scene.Root.AddChild(whelp)

	cx := w / 2 // mask center relative to whelp
	cy := h / 2

	// Pre-build mask roots. Each is a container with a shape child.
	// Effect 0: Rotating star mask
	starRoot := willow.NewContainer("star-root")
	star := willow.NewStar("star", 130, 50, 5)
	star.SetPosition(cx, cy)
	star.SetColor(willow.RGB(1, 1, 1))
	starRoot.AddChild(star)

	// Effect 1: Iris circle expanding/contracting
	circleRoot := willow.NewContainer("circle-root")
	circle := willow.NewCircle("circle", 10, willow.RGB(1, 1, 1))
	circle.SetPosition(cx, cy)
	circleRoot.AddChild(circle)

	// Effect 2: Diamond/rhombus rotating
	diamondRoot := willow.NewContainer("diamond-root")
	diamondPts := []willow.Vec2{
		{X: 0, Y: -120}, {X: 90, Y: 0}, {X: 0, Y: 120}, {X: -90, Y: 0},
	}
	diamond := willow.NewPolygon("diamond", diamondPts)
	diamond.SetPosition(cx, cy)
	diamond.SetColor(willow.RGB(1, 1, 1))
	diamondRoot.AddChild(diamond)

	// Effect 3: Erase mask — full rect with whelp punched out (inverted silhouette)
	eraseRoot := willow.NewContainer("erase-root")
	eraseRect := willow.NewPolygon("erase-rect", []willow.Vec2{
		{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h},
	})
	eraseRect.SetColor(willow.RGB(1, 1, 1))
	eraseRoot.AddChild(eraseRect)
	eraseWhelp := willow.NewSprite("erase-whelp", willow.TextureRegion{})
	eraseWhelp.SetCustomImage(whelpImg)
	eraseScale := 0.5
	eraseWhelp.SetScale(eraseScale, eraseScale)
	eraseWhelp.SetBlendMode(willow.BlendErase)
	eraseRoot.AddChild(eraseWhelp)

	// Effect 4: Multi-star burst
	burstRoot := willow.NewContainer("burst-root")
	burst := willow.NewStar("burst", 130, 20, 12)
	burst.SetPosition(cx, cy)
	burst.SetColor(willow.RGB(1, 1, 1))
	burstRoot.AddChild(burst)

	masks := []*willow.Node{starRoot, circleRoot, diamondRoot, eraseRoot, burstRoot}
	shapes := []*willow.Node{star, circle, diamond, eraseWhelp, burst}

	effectDuration := 1.0
	elapsed := 0.0
	prevIdx := -1

	updateFn := func() error {
		elapsed += 1.0 / float64(ebiten.TPS())
		idx := int(elapsed/effectDuration) % len(masks)
		t := math.Mod(elapsed, effectDuration) / effectDuration

		if idx != prevIdx {
			whelp.SetMask(masks[idx])
			prevIdx = idx
		}

		switch idx {
		case 0: // Rotating star
			star.SetRotation(elapsed * 1.5)
			s := 0.7 + 0.4*math.Sin(t*math.Pi)
			star.SetScale(s, s)
			star.Invalidate()
		case 1: // Iris circle
			r := 10 + 120*math.Sin(t*math.Pi)
			circle.SetScale(r/10, r/10)
			circle.Invalidate()
		case 2: // Diamond spin
			diamond.SetRotation(elapsed * 2.0)
			s := 0.6 + 0.5*math.Sin(t*math.Pi)
			diamond.SetScale(s, s)
			diamond.Invalidate()
		case 3: // Erase whelp orbiting
			angle := t * math.Pi * 2
			ex := cx - float64(whelpImg.Bounds().Dx())*eraseScale/2 + 50*math.Cos(angle)
			ey := cy - float64(whelpImg.Bounds().Dy())*eraseScale/2 + 50*math.Sin(angle)
			eraseWhelp.SetPosition(ex, ey)
			s := 0.4 + 0.3*math.Sin(t*math.Pi*3)
			eraseWhelp.SetScale(s, s)
			eraseWhelp.Invalidate()
		case 4: // Burst star
			burst.SetRotation(elapsed * 3.0)
			s := 0.5 + 0.6*math.Sin(t*math.Pi)
			burst.SetScale(s, s)
			burst.Invalidate()
		}

		// Also tint the background over time for extra flair.
		_ = shapes
		whelp.Invalidate()
		return nil
	}

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
			if err := updateFn(); err != nil {
				return err
			}
			if runner.Done() {
				fmt.Println("Autotest complete.")
				return ebiten.Termination
			}
			return nil
		})
	} else {
		scene.SetUpdateFunc(updateFn)
	}

	if err := willow.Run(scene, willow.RunConfig{
		Title:   "Willow  -  Mask Showcase",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: false,
	}); err != nil {
		log.Fatal(err)
	}
}
