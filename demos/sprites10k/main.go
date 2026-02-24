// sprites10k spawns 10,000 whelp sprites that rotate, scale, fade, and
// bounce around the screen simultaneously. A stress test for the Willow
// rendering pipeline.
package main

import (
	"bytes"
	_ "embed"
	"image/png"
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
)

//go:embed whelp.png
var whelpPNG []byte

const (
	screenW = 1280
	screenH = 720
	count   = 10_000
)

type sprite struct {
	node       *willow.Node
	dx, dy     float64
	rotSpeed   float64
	scaleSpeed float64
	scaleBase  float64
	scaleAmp   float64
	alphaSpeed float64
	phase      float64
}

func main() {
	img, err := png.Decode(bytes.NewReader(whelpPNG))
	if err != nil {
		log.Fatalf("decode whelp.png: %v", err)
	}
	whelpImg := ebiten.NewImageFromImage(img)

	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.06, G: 0.06, B: 0.09, A: 1}

	scene.RegisterPage(0, whelpImg)
	region := willow.TextureRegion{
		Page:      0,
		Width:     128,
		Height:    128,
		OriginalW: 128,
		OriginalH: 128,
	}

	sprites := make([]sprite, count)
	root := scene.Root()

	for i := range sprites {
		sp := willow.NewSprite("whelp", region)

		sp.X = rand.Float64() * screenW
		sp.Y = rand.Float64() * screenH

		sp.PivotX = 64
		sp.PivotY = 64

		base := 0.15 + rand.Float64()*0.2
		sp.ScaleX = base
		sp.ScaleY = base

		sp.Color = willow.Color{
			R: 0.5 + rand.Float64()*0.5,
			G: 0.5 + rand.Float64()*0.5,
			B: 0.5 + rand.Float64()*0.5,
			A: 1,
		}

		root.AddChild(sp)

		sprites[i] = sprite{
			node:       sp,
			dx:         (rand.Float64() - 0.5) * 4,
			dy:         (rand.Float64() - 0.5) * 4,
			rotSpeed:   (rand.Float64() - 0.5) * 0.08,
			scaleSpeed: 1 + rand.Float64()*2,
			scaleBase:  base,
			scaleAmp:   0.03 + rand.Float64()*0.07,
			alphaSpeed: 0.5 + rand.Float64()*2,
			phase:      rand.Float64() * math.Pi * 2,
		}
	}

	var frame float64

	scene.SetUpdateFunc(func() error {
		frame++
		t := frame / 60.0

		for i := range sprites {
			s := &sprites[i]
			n := s.node

			n.X += s.dx
			n.Y += s.dy

			size := s.scaleBase * 128
			if n.X < -size/2 {
				n.X = -size / 2
				s.dx = -s.dx
			} else if n.X > screenW-size/2 {
				n.X = screenW - size/2
				s.dx = -s.dx
			}
			if n.Y < -size/2 {
				n.Y = -size / 2
				s.dy = -s.dy
			} else if n.Y > screenH-size/2 {
				n.Y = screenH - size/2
				s.dy = -s.dy
			}

			n.Rotation += s.rotSpeed

			sc := s.scaleBase + s.scaleAmp*math.Sin(t*s.scaleSpeed+s.phase)
			n.ScaleX = sc
			n.ScaleY = sc

			n.Alpha = 0.5 + 0.5*math.Sin(t*s.alphaSpeed+s.phase)

			n.Invalidate()
		}
		return nil
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:   "Willow  -  10k Sprites",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}
