// Watermesh demonstrates per-vertex wave animation using a DistortionGrid
// textured with tile 13 (the water tile) from the bundled tileset.
package main

import (
	"image"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Water Mesh"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	tileSize    = 32

	// Core grid dimensions (screen-filling).
	gridCols = screenW / tileSize // 20
	gridRows = screenH / tileSize // 15

	// One extra tile on each edge prevents wave displacement from revealing
	// the background color at screen borders.
	bufTiles  = 1
	totalCols = gridCols + bufTiles*2 // 22
	totalRows = gridRows + bufTiles*2 // 17
)

type game struct {
	grid *willow.DistortionGrid
	time float64
}

func (g *game) update() error {
	g.time += 1.0 / float64(ebiten.TPS())
	t := g.time

	g.grid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		// Two overlapping sine waves travelling in different directions
		// produce an organic rolling water surface.
		wave1 := math.Sin(restX*0.04 + restY*0.03 + t*2.0)
		wave2 := math.Sin(restX*0.03 - restY*0.05 + t*1.5 + 1.2)
		dy = 5.0 * (wave1 + 0.4*wave2)
		dx = 1.5 * math.Sin(restY*0.05+t*1.2)
		return
	})
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

	// Tile 13 (1-indexed from top-left): row 3, col 0 in the 4-wide tileset.
	waterSrc := tileset.SubImage(image.Rect(0, 96, tileSize, 128)).(*ebiten.Image)

	// Pre-render the water tile tiled across a (totalCols × totalRows) image.
	waterImg := ebiten.NewImage(totalCols*tileSize, totalRows*tileSize)
	var op ebiten.DrawImageOptions
	for gy := range totalRows {
		for gx := range totalCols {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(gx*tileSize), float64(gy*tileSize))
			waterImg.DrawImage(waterSrc, &op)
		}
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.03, 0.08, 0.15)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	grid := willow.NewDistortionGrid("water", waterImg, totalCols, totalRows)
	// Shift the mesh so the buffer tiles sit off-screen, centering the
	// screen-filling portion over the viewport.
	grid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	// Subtle blue tint to reinforce the water feel.
	grid.Node().SetColor(willow.RGB(0.75, 0.92, 1.0))
	scene.Root().AddChild(grid.Node())

	g := &game{grid: grid}
	scene.SetUpdateFunc(g.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}
