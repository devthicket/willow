// MultiCamera proves multiple cameras with non-overlapping viewports
// each render their own clipped slice of the world.
//
// Layout:
//   - Left half  (0..screenW/2)        : camera A follows the blue player (WASD)
//   - Right half (screenW/2..screenW)  : camera B follows the orange player (arrows)
//
// Each camera's Viewport is its on-screen slice. Scene.Draw uses
// screen.SubImage(viewport) to clip to that slice, then applies the
// camera's view matrix on top — so the same scene tree renders twice,
// each from a different vantage.
//
// Controls:
//
//	WASD  : move blue player (camera A target)
//	Arrows: move orange player (camera B target)
//	F11   : toggle fullscreen
//	Esc   : quit
package main

import (
	"image/color"
	"log"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	windowTitle = "Willow  -  Multi-Camera Example"
	screenW     = 1280
	screenH     = 720
	worldW      = 3000.0
	worldH      = 2000.0
	playerSize  = 32.0
	playerSpeed = 4.0
)

type player struct {
	node       *willow.Node
	upK, dnK   ebiten.Key
	leftK, rtK ebiten.Key
}

func (p *player) update() {
	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(p.leftK) {
		dx -= playerSpeed
	}
	if ebiten.IsKeyPressed(p.rtK) {
		dx += playerSpeed
	}
	if ebiten.IsKeyPressed(p.upK) {
		dy -= playerSpeed
	}
	if ebiten.IsKeyPressed(p.dnK) {
		dy += playerSpeed
	}
	x := clamp(p.node.X()+dx, 0, worldW-playerSize)
	y := clamp(p.node.Y()+dy, 0, worldH-playerSize)
	p.node.SetPosition(x, y)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.08, 0.08, 0.10)

	// World content: a grid of reference squares so each camera's view is
	// visually distinguishable from the others.
	for gy := 0; gy < int(worldH); gy += 200 {
		for gx := 0; gx < int(worldW); gx += 200 {
			tile := willow.NewRect("tile", 180, 180,
				willow.RGB(0.18+0.02*float64((gx/200+gy/200)%2),
					0.20+0.02*float64((gx/200+gy/200)%2),
					0.30))
			tile.SetPosition(float64(gx)+10, float64(gy)+10)
			scene.Root.AddChild(tile)
		}
	}
	// World-edge marker: a red border on the world bounds so the minimap
	// shows the extent and the per-player cameras can hit edges.
	border := willow.NewRect("border", worldW, 8, willow.RGB(0.9, 0.2, 0.2))
	border.SetPosition(0, 0)
	scene.Root.AddChild(border)
	border = willow.NewRect("border", worldW, 8, willow.RGB(0.9, 0.2, 0.2))
	border.SetPosition(0, worldH-8)
	scene.Root.AddChild(border)
	border = willow.NewRect("border", 8, worldH, willow.RGB(0.9, 0.2, 0.2))
	border.SetPosition(0, 0)
	scene.Root.AddChild(border)
	border = willow.NewRect("border", 8, worldH, willow.RGB(0.9, 0.2, 0.2))
	border.SetPosition(worldW-8, 0)
	scene.Root.AddChild(border)

	// Two players.
	blue := willow.NewRect("blue", playerSize, playerSize, willow.RGB(0.30, 0.55, 1.0))
	blue.SetPosition(worldW*0.25-playerSize/2, worldH*0.5-playerSize/2)
	scene.Root.AddChild(blue)

	orange := willow.NewRect("orange", playerSize, playerSize, willow.RGB(1.0, 0.55, 0.20))
	orange.SetPosition(worldW*0.75-playerSize/2, worldH*0.5-playerSize/2)
	scene.Root.AddChild(orange)

	// Three cameras with disjoint viewports.
	camLeft := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW / 2, Height: screenH})
	camLeft.Follow(blue, playerSize/2, playerSize/2, 0.15)
	camLeft.SetBounds(willow.Rect{X: 0, Y: 0, Width: worldW, Height: worldH})

	camRight := scene.NewCamera(willow.Rect{X: screenW / 2, Y: 0, Width: screenW / 2, Height: screenH})
	camRight.Follow(orange, playerSize/2, playerSize/2, 0.15)
	camRight.SetBounds(willow.Rect{X: 0, Y: 0, Width: worldW, Height: worldH})

	bluePlayer := &player{node: blue, upK: ebiten.KeyW, dnK: ebiten.KeyS, leftK: ebiten.KeyA, rtK: ebiten.KeyD}
	orangePlayer := &player{node: orange, upK: ebiten.KeyUp, dnK: ebiten.KeyDown, leftK: ebiten.KeyLeft, rtK: ebiten.KeyRight}
	scene.SetUpdateFunc(func() error {
		bluePlayer.update()
		orangePlayer.update()
		if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
			ebiten.SetFullscreen(!ebiten.IsFullscreen())
		}
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			return ebiten.Termination
		}
		return nil
	})

	// HUD overlays drawn in logical screen pixels (PostDrawFunc receives a
	// screen sized cfg.Width × cfg.Height).
	scene.SetPostDrawFunc(func(screen *ebiten.Image) {
		// Vertical seam between left and right viewports.
		seam := ebiten.NewImage(2, screenH)
		seam.Fill(color.RGBA{40, 40, 60, 255})
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(screenW/2-1, 0)
		screen.DrawImage(seam, &op)

		ebitenutil.DebugPrintAt(screen, "Cam A (WASD) — blue", 8, 8)
		ebitenutil.DebugPrintAt(screen, "Cam B (Arrows) — orange", screenW/2+8, 8)
		ebitenutil.DebugPrintAt(screen, "F11: toggle fullscreen   Esc: quit", 8, screenH-20)
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}
