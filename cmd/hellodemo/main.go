package main

import (
	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/goregular"
)

func main() {
	scene := willow.NewScene()
	rect := willow.NewRect("r", 380, 54, willow.RGBA(0.25, 0.28, 0.35, 1))
	rect.SetPosition(10, 8)
	scene.Root.AddChild(rect)
	font, _ := willow.NewFontFromTTF(goregular.TTF, 80)
	txt := willow.NewText("t", "Hello Willow!", font)
	txt.SetFontSize(36)
	txt.SetPosition(100, 17)
	scene.Root.AddChild(txt)

	frame := 0
	scene.UpdateFunc = func() error {
		frame++
		if frame == 10 {
			scene.Screenshot("hero")
		}
		if frame == 12 {
			return ebiten.Termination
		}
		return nil
	}

	willow.Run(scene, willow.RunConfig{
		Width:      400,
		Height:     70,
		Background: willow.RGBA(0.12, 0.13, 0.17, 1),
	})
}
