// TileMapViewport demonstrates the geometry-buffer tilemap renderer with
// camera panning, multiple tile layers, and a sandwich entity layer.
// Uses the same 4x4 tileset (32x32 tiles) as the tilemap example.
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  TileMapViewport Example"
	screenW     = 1920
	screenH     = 1080
	tileSize    = 32
	mapWidth    = 1000
	mapHeight   = 1000
	numTiles    = 16 // 4x4 tileset
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
	scene.ClearColor = willow.Color{R: 0.1, G: 0.1, B: 0.1, A: 1}

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = (mapWidth * tileSize) / 2
	cam.Y = (mapHeight * tileSize) / 2
	cam.SetBounds(willow.Rect{X: 0, Y: 0, Width: mapWidth * tileSize, Height: mapHeight * tileSize})
	cam.Invalidate()

	// Build TextureRegion lookup from the tileset image.
	// GID 0 = empty, GIDs 1..16 map to the 16 tiles in the 4x4 sheet.
	tilesPerRow := img.Bounds().Dx() / tileSize
	regions := make([]willow.TextureRegion, numTiles+1)
	for i := 1; i <= numTiles; i++ {
		tx := ((i - 1) % tilesPerRow) * tileSize
		ty := ((i - 1) / tilesPerRow) * tileSize
		regions[i] = willow.TextureRegion{
			X: uint16(tx), Y: uint16(ty),
			Width: tileSize, Height: tileSize,
			OriginalW: tileSize, OriginalH: tileSize,
		}
	}

	// Generate random tile data for two layers.
	groundData := make([]uint32, mapWidth*mapHeight)
	decorData := make([]uint32, mapWidth*mapHeight)
	for i := range groundData {
		groundData[i] = uint32(rand.IntN(4)) + 1 // GIDs 1-4 (first row: ground tiles)
	}
	for i := range decorData {
		if rand.IntN(10) == 0 { // 10% coverage
			decorData[i] = uint32(rand.IntN(4)) + 5 // GIDs 5-8 (second row: decor tiles)
		}
	}

	// Create the TileMapViewport.
	viewport := willow.NewTileMapViewport("world", tileSize, tileSize)
	viewport.SetCamera(cam)
	scene.Root().AddChild(viewport.Node())

	// Ground tile layer (RenderLayer 0).
	ground := viewport.AddTileLayer("ground", mapWidth, mapHeight, groundData, regions, img)
	ground.Node().RenderLayer = 0

	// Decor tile layer (RenderLayer 1).
	decor := viewport.AddTileLayer("decor", mapWidth, mapHeight, decorData, regions, img)
	decor.Node().RenderLayer = 1

	// Sandwich entity layer (RenderLayer 2)  -  a few colored sprites as "NPCs".
	entityLayer := willow.NewContainer("entities")
	entityLayer.RenderLayer = 2
	viewport.AddChild(entityLayer)

	for i := range 20 {
		npc := willow.NewSprite("npc", willow.TextureRegion{})
		npc.ScaleX = 16
		npc.ScaleY = 16
		npc.Color = willow.Color{R: 1, G: 0.3, B: 0.3, A: 1}
		npc.X = float64(rand.IntN(mapWidth*tileSize-tileSize)) + float64(tileSize)/2
		npc.Y = float64(rand.IntN(mapHeight*tileSize-tileSize)) + float64(tileSize)/2
		_ = i
		entityLayer.AddChild(npc)
	}

	// Upper tile layer (RenderLayer 3)  -  sparse overlay that draws on top of entities.
	upperData := make([]uint32, mapWidth*mapHeight)
	for i := range upperData {
		if rand.IntN(20) == 0 { // 5% coverage
			upperData[i] = uint32(rand.IntN(4)) + 13 // GIDs 13-16 (fourth row)
		}
	}
	upper := viewport.AddTileLayer("upper", mapWidth, mapHeight, upperData, regions, img)
	upper.Node().RenderLayer = 3

	// Camera panning via drag.
	scene.SetDragDeadZone(0)
	scene.Root().HitShape = willow.HitRect{X: -1e6, Y: -1e6, Width: 2e6, Height: 2e6}
	scene.Root().OnDrag(func(ctx willow.DragContext) {
		cam.X -= ctx.ScreenDeltaX / cam.Zoom
		cam.Y -= ctx.ScreenDeltaY / cam.Zoom
		cam.ClampToBounds()
		cam.Invalidate()
	})

	// Debug overlay: camera coordinates and tile position at the bottom of the screen.
	scene.SetPostDrawFunc(func(screen *ebiten.Image) {
		tileX := int(cam.X) / tileSize
		tileY := int(cam.Y) / tileSize
		msg := fmt.Sprintf("Camera: %.0f, %.0f  |  Tile: %d, %d", cam.X, cam.Y, tileX, tileY)
		ebitenutil.DebugPrintAt(screen, msg, 4, screenH-16)
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
