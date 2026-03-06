// sprites10k spawns 10,000 whelp sprites that rotate, scale, fade, and
// bounce around the screen simultaneously. A stress test for the Willow
// rendering pipeline.
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

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
	scene.ClearColor = willow.RGB(0.06, 0.06, 0.09)

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

		sp.SetPosition(rand.Float64()*screenW, rand.Float64()*screenH)

		sp.SetPivot(64, 64)

		base := 0.15 + rand.Float64()*0.2
		sp.SetScale(base, base)

		sp.SetColor(willow.RGB(
			0.5+rand.Float64()*0.5,
			0.5+rand.Float64()*0.5,
			0.5+rand.Float64()*0.5,
		))

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

			nx := n.X() + s.dx
			ny := n.Y() + s.dy

			size := s.scaleBase * 128
			if nx < -size/2 {
				nx = -size / 2
				s.dx = -s.dx
			} else if nx > screenW-size/2 {
				nx = screenW - size/2
				s.dx = -s.dx
			}
			if ny < -size/2 {
				ny = -size / 2
				s.dy = -s.dy
			} else if ny > screenH-size/2 {
				ny = screenH - size/2
				s.dy = -s.dy
			}
			n.SetPosition(nx, ny)

			n.SetRotation(n.Rotation() + s.rotSpeed)

			sc := s.scaleBase + s.scaleAmp*math.Sin(t*s.scaleSpeed+s.phase)
			n.SetScale(sc, sc)

			n.SetAlpha(0.5 + 0.5*math.Sin(t*s.alphaSpeed+s.phase))

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
