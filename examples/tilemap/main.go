// Tilemap demonstrates a grid-based tilemap with camera panning.
package main

import (
	"image"
	"log"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Tilemap Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	tileSize    = 32
	mapWidth    = 30
	mapHeight   = 30
)

func main() {
	f, err := os.Open("examples/_assets/tileset.png")
	if err != nil {
		log.Fatalf("failed to open tileset: %v", err)
	}
	defer f.Close()

	img, _, err := ebitenutil.NewImageFromReader(f)
	if err != nil {
		log.Fatalf("failed to load tileset image: %v", err)
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.1, 0.1, 0.1)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = (mapWidth * tileSize) / 2
	cam.Y = (mapHeight * tileSize) / 2
	cam.Invalidate()

	// Cache the tilemap's render commands so camera panning replays them
	// via delta remap instead of re-traversing 900 nodes every frame.
	// Manual mode is used because tiles never change after setup.
	mapContainer := willow.NewContainer("tilemap")
	mapContainer.SetCacheAsTree(true, willow.CacheTreeManual)
	scene.Root.AddChild(mapContainer)

	// Populate the tilemap with random tiles.
	tilesetW := img.Bounds().Dx()
	tilesPerRow := tilesetW / tileSize

	for y := range mapHeight {
		for x := range mapWidth {
			tileIdx := rand.IntN(4)
			tx := (tileIdx % tilesPerRow) * tileSize
			ty := (tileIdx / tilesPerRow) * tileSize

			sub := img.SubImage(image.Rect(tx, ty, tx+tileSize, ty+tileSize)).(*ebiten.Image)

			tile := willow.NewSprite("tile", willow.TextureRegion{})
			tile.SetCustomImage(sub)
			tile.SetPosition(float64(x*tileSize), float64(y*tileSize))
			mapContainer.AddChild(tile)
		}
	}

	// Camera panning via OnDrag with screen-space deltas.
	scene.SetDragDeadZone(0)
	scene.Root.HitShape = willow.HitRect{X: -1e6, Y: -1e6, Width: 2e6, Height: 2e6}
	scene.Root.OnDrag(func(ctx willow.DragContext) {
		cam.X -= ctx.ScreenDeltaX / cam.Zoom
		cam.Y -= ctx.ScreenDeltaY / cam.Zoom
		cam.Invalidate()
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
