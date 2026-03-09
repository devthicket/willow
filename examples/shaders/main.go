// Shaders showcases Willow's built-in filter system by displaying all
// shader effects simultaneously in a 3x3 grid, each applied to a
// pre-rendered tilemap panel with animated parameters.
package main

import (
	"image"
	"log"
	"math"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Shader Showcase"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	tileSize    = 32
	panelTW     = 6 // tiles wide per panel
	panelTH     = 4 // tiles tall per panel
	gridCols    = 3
	gridRows    = 3
	gridGap     = 10
)

// shaderPanel holds one grid cell: a Willow sprite with a filter and its animation.
type shaderPanel struct {
	node   *willow.Node
	update func(t float64)
}

type animator struct {
	panels []*shaderPanel
	time   float64
}

func (a *animator) update() error {
	a.time += 1.0 / float64(ebiten.TPS())
	for _, p := range a.panels {
		p.update(a.time)
	}
	return nil
}

func main() {
	f, err := os.Open("examples/_assets/tileset.png")
	if err != nil {
		log.Fatalf("open tileset: %v", err)
	}
	defer f.Close()

	tileset, _, err := ebitenutil.NewImageFromReader(f)
	if err != nil {
		log.Fatalf("load tileset: %v", err)
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.08, 0.06, 0.12)

	// 1:1 camera: screen pixels map directly to world coordinates.
	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Create one filter instance per panel (ColorMatrix demos each need their own).
	brightness := willow.NewColorMatrixFilter()
	saturation := willow.NewColorMatrixFilter()
	contrast := willow.NewColorMatrixFilter()
	sepia := willow.NewColorMatrixFilter()
	invert := willow.NewColorMatrixFilter()
	nightVision := willow.NewColorMatrixFilter()
	hueShift := willow.NewColorMatrixFilter()
	warmCool := willow.NewColorMatrixFilter()
	blur := willow.NewBlurFilter(0)

	// Define the 9 shader demos.
	type panelDef struct {
		name    string
		filters []any
		update  func(t float64)
	}
	defs := []panelDef{
		{"Brightness", []any{brightness}, func(t float64) {
			brightness.SetBrightness(0.35 * math.Sin(t*3))
		}},
		{"Saturation", []any{saturation}, func(t float64) {
			saturation.SetSaturation(0.5 + 0.5*math.Cos(t*2))
		}},
		{"Contrast", []any{contrast}, func(t float64) {
			contrast.SetContrast(1.2 + 0.8*math.Sin(t*2))
		}},
		{"Sepia", []any{sepia}, func(t float64) {
			// Fade from normal to full sepia and back.
			s := 0.5 + 0.5*math.Sin(t*1.5)
			sepia.Matrix = [20]float64{
				0.393*s + (1 - s), 0.769 * s, 0.189 * s, 0, 0,
				0.349 * s, 0.686*s + (1 - s), 0.168 * s, 0, 0,
				0.272 * s, 0.534 * s, 0.131*s + (1 - s), 0, 0,
				0, 0, 0, 1, 0,
			}
		}},
		{"Invert", []any{invert}, func(t float64) {
			// Smoothly transition between normal and inverted.
			s := 0.5 + 0.5*math.Sin(t*1.2)
			f := 1 - 2*s // +1 (normal) to -1 (inverted)
			o := s       // offset: 0 (normal) to 1 (inverted)
			invert.Matrix = [20]float64{
				f, 0, 0, 0, o,
				0, f, 0, 0, o,
				0, 0, f, 0, o,
				0, 0, 0, 1, 0,
			}
		}},
		{"Night Vision", []any{nightVision}, func(t float64) {
			// Green-tinted with pulsing intensity.
			p := 0.8 + 0.2*math.Sin(t*4)
			nightVision.Matrix = [20]float64{
				0.1 * p, 0.4 * p, 0.1 * p, 0, 0.02,
				0.2 * p, 0.7 * p, 0.2 * p, 0, 0.04,
				0.05 * p, 0.15 * p, 0.05 * p, 0, 0.01,
				0, 0, 0, 1, 0,
			}
		}},
		{"Hue Shift", []any{hueShift}, func(t float64) {
			// Rotate hue through the color wheel.
			angle := t * 0.8
			cos, sin := math.Cos(angle), math.Sin(angle)
			// Hue rotation matrix via luminance-preserving rotation.
			hueShift.Matrix = [20]float64{
				0.213 + cos*0.787 - sin*0.213, 0.715 - cos*0.715 - sin*0.715, 0.072 - cos*0.072 + sin*0.928, 0, 0,
				0.213 - cos*0.213 + sin*0.143, 0.715 + cos*0.285 + sin*0.140, 0.072 - cos*0.072 - sin*0.283, 0, 0,
				0.213 - cos*0.213 - sin*0.787, 0.715 - cos*0.715 + sin*0.715, 0.072 + cos*0.928 + sin*0.072, 0, 0,
				0, 0, 0, 1, 0,
			}
		}},
		{"Warm / Cool", []any{warmCool}, func(t float64) {
			// Cycle between warm (orange tint) and cool (blue tint).
			s := math.Sin(t * math.Pi * 2 / 3) // full cycle every 3 seconds
			warmCool.Matrix = [20]float64{
				1, 0, 0, 0, 0.12 * s,
				0, 1, 0, 0, 0.04 * s,
				0, 0, 1, 0, -0.10 * s,
				0, 0, 0, 1, 0,
			}
		}},
		{"Blur", []any{blur}, func(t float64) {
			blur.Radius = int(12 * (0.5 + 0.5*math.Sin(t*1.5)))
		}},
	}

	// Shuffle so each run shows a different layout.
	for i := len(defs) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		defs[i], defs[j] = defs[j], defs[i]
	}

	// Grid layout.
	pw := panelTW * tileSize
	ph := panelTH * tileSize
	gridW := gridCols*pw + (gridCols-1)*gridGap
	gridH := gridRows*ph + (gridRows-1)*gridGap
	startX := float64(screenW-gridW) / 2
	startY := float64(screenH-gridH) / 2

	panels := make([]*shaderPanel, len(defs))
	for i, def := range defs {
		col := i % gridCols
		row := i / gridCols

		// Pre-render tiles into a single image (no seam gaps).
		panelImg := renderTilePanel(tileset, panelTW, panelTH)

		sprite := willow.NewSprite(def.name, willow.TextureRegion{})
		sprite.SetCustomImage(panelImg)
		sprite.Filters = def.filters

		x := startX + float64(col*(pw+gridGap))
		y := startY + float64(row*(ph+gridGap))
		sprite.SetPosition(x, y)
		scene.Root().AddChild(sprite)

		panels[i] = &shaderPanel{
			node:   sprite,
			update: def.update,
		}

		// Pre-render the label text into a sprite and overlay it on the panel.
		label := makeLabel(def.name)
		label.SetPosition(x+4, y+2)
		scene.Root().AddChild(label)
	}

	a := &animator{panels: panels}
	scene.SetUpdateFunc(a.update)

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
	// Debug font is roughly 6px wide, 16px tall per character.
	w := len(s)*6 + 2
	h := 16
	img := ebiten.NewImage(w, h)
	ebitenutil.DebugPrint(img, s)

	n := willow.NewSprite("label-"+s, willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}

// renderTilePanel composites random tiles into a single seamless image.
func renderTilePanel(tileset *ebiten.Image, cols, rows int) *ebiten.Image {
	w := cols * tileSize
	h := rows * tileSize
	panel := ebiten.NewImage(w, h)

	tilesetW := tileset.Bounds().Dx()
	tilesPerRow := tilesetW / tileSize
	var op ebiten.DrawImageOptions
	for y := range rows {
		for x := range cols {
			idx := rand.IntN(4)
			tx := (idx % tilesPerRow) * tileSize
			ty := (idx / tilesPerRow) * tileSize
			sub := tileset.SubImage(image.Rect(tx, ty, tx+tileSize, ty+tileSize)).(*ebiten.Image)

			op.GeoM.Reset()
			op.GeoM.Translate(float64(x*tileSize), float64(y*tileSize))
			panel.DrawImage(sub, &op)
		}
	}
	return panel
}
