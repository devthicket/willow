// Filter Showcase demonstrates Willow's filter and shader system with a single
// sprite cycling through dramatic visual effects.
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

	whelp := willow.NewSprite("whelp", willow.TextureRegion{})
	whelp.SetCustomImage(whelpImg)
	whelp.SetScale(1.8, 1.8)
	w := float64(whelpImg.Bounds().Dx()) * 1.8
	h := float64(whelpImg.Bounds().Dy()) * 1.8
	whelp.SetPosition((screenW-w)/2, (screenH-h)/2)
	scene.Root.AddChild(whelp)

	// Filter instances.
	blur := willow.NewBlurFilter(0)
	colorMat := willow.NewColorMatrixFilter()
	outline := willow.NewOutlineFilter(3, willow.RGB(1, 0.2, 0.4))
	palette := willow.NewPaletteFilter()

	var retroPalette [256]willow.Color
	for i := range 256 {
		t := float64(i) / 255.0
		retroPalette[i] = willow.RGB(
			0.5+0.5*math.Cos(2*math.Pi*t+0.5),
			0.5+0.5*math.Cos(2*math.Pi*t+2*math.Pi/3+0.5),
			0.5+0.5*math.Cos(2*math.Pi*t+4*math.Pi/3+0.5),
		)
	}
	palette.SetPalette(retroPalette)

	// Each effect lasts 1 second with a smooth transition feel.
	type effect struct {
		apply func(t float64) // t goes 0→1 over the effect's duration
	}

	effects := []effect{
		// 0: Blur pulse with glow outline
		{apply: func(t float64) {
			r := int(14 * math.Sin(t*math.Pi))
			blur.Radius = r
			outline.Color = willow.RGB(0.3+0.7*math.Sin(t*math.Pi), 0.6*math.Sin(t*math.Pi), 1.0*math.Sin(t*math.Pi))
			outline.Thickness = 2 + int(2*math.Sin(t*math.Pi))
			whelp.Filters = []any{blur, outline}
		}},
		// 1: Hue rotation
		{apply: func(t float64) {
			angle := t * math.Pi * 2
			cos, sin := math.Cos(angle), math.Sin(angle)
			colorMat.Matrix = [20]float64{
				0.213 + cos*0.787 - sin*0.213, 0.715 - cos*0.715 - sin*0.715, 0.072 - cos*0.072 + sin*0.928, 0, 0,
				0.213 - cos*0.213 + sin*0.143, 0.715 + cos*0.285 + sin*0.140, 0.072 - cos*0.072 - sin*0.283, 0, 0,
				0.213 - cos*0.213 - sin*0.787, 0.715 - cos*0.715 + sin*0.715, 0.072 + cos*0.928 + sin*0.072, 0, 0,
				0, 0, 0, 1, 0,
			}
			whelp.Filters = []any{colorMat}
		}},
		// 2: Retro palette with cycling
		{apply: func(t float64) {
			palette.CycleOffset = t * 80
			whelp.Filters = []any{palette}
		}},
		// 3: Invert + neon outline
		{apply: func(t float64) {
			s := 0.5 + 0.5*math.Sin(t*math.Pi*2)
			inv := 1 - 2*s
			colorMat.Matrix = [20]float64{
				inv, 0, 0, 0, s,
				0, inv, 0, 0, s,
				0, 0, inv, 0, s,
				0, 0, 0, 1, 0,
			}
			g := 0.5 + 0.5*math.Sin(t*math.Pi*4)
			outline.Color = willow.RGB(0, g, 1)
			outline.Thickness = 3
			whelp.Filters = []any{colorMat, outline}
		}},
		// 4: Night vision blur combo
		{apply: func(t float64) {
			p := 0.7 + 0.3*math.Sin(t*math.Pi*6)
			colorMat.Matrix = [20]float64{
				0.1 * p, 0.4 * p, 0.1 * p, 0, 0.03,
				0.2 * p, 0.8 * p, 0.2 * p, 0, 0.06,
				0.05 * p, 0.15 * p, 0.05 * p, 0, 0.01,
				0, 0, 0, 1, 0,
			}
			blur.Radius = 2 + int(3*math.Sin(t*math.Pi*3))
			whelp.Filters = []any{colorMat, blur}
		}},
	}

	elapsed := 0.0
	effectDuration := 1.0

	updateFn := func() error {
		elapsed += 1.0 / float64(ebiten.TPS())

		idx := int(elapsed/effectDuration) % len(effects)
		localT := math.Mod(elapsed, effectDuration) / effectDuration

		effects[idx].apply(localT)
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
		Title:   "Willow  -  Filter Showcase",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: false,
	}); err != nil {
		log.Fatal(err)
	}
}
