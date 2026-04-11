// Demo: loading a Tiled (.tmx) map into Willow's tilemap system via go-tiled.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	tiled "github.com/lafriks/go-tiled"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	// Generate the tileset PNG the TMX references.
	assetsDir := filepath.Join("examples", "_assets")
	generateTilesetPNG(filepath.Join(assetsDir, "terrain.png"))

	// Load the TMX map via go-tiled.
	tmxPath := filepath.Join(assetsDir, "demo.tmx")
	gameMap, err := tiled.LoadFile(tmxPath)
	if err != nil {
		log.Fatalf("load tmx: %v", err)
	}

	fmt.Printf("Loaded map: %dx%d tiles, tile size %dx%d\n",
		gameMap.Width, gameMap.Height, gameMap.TileWidth, gameMap.TileHeight)

	// Build tileset atlas image from the tileset PNG.
	atlasImage, regions := buildAtlas(gameMap)

	// Set up Willow scene.
	mapW := float64(gameMap.Width * gameMap.TileWidth)
	mapH := float64(gameMap.Height * gameMap.TileHeight)
	scene := willow.NewScene()
	cam := scene.NewCamera(willow.Rect{
		X: 0, Y: 0,
		Width:  mapW,
		Height: mapH,
	})
	cam.X = mapW / 2
	cam.Y = mapH / 2

	viewport := willow.NewTileMapViewport("tilemap",
		gameMap.TileWidth, gameMap.TileHeight)
	viewport.SetCamera(cam)
	willow.Root(scene).AddChild(viewport.Node())

	// Convert each TMX tile layer into a Willow tile layer.
	for _, tmxLayer := range gameMap.Layers {
		data := convertLayerData(gameMap, tmxLayer)
		viewport.AddTileLayer(willow.TileLayerConfig{
			Name:       tmxLayer.Name,
			Width:      gameMap.Width,
			Height:     gameMap.Height,
			Data:       data,
			Regions:    regions,
			AtlasImage: atlasImage,
		})
		fmt.Printf("  layer %q: %d×%d\n", tmxLayer.Name, gameMap.Width, gameMap.Height)
	}

	fmt.Println("Running demo — close window to exit.")

	if err := willow.Run(scene, willow.RunConfig{
		Title:        "Tiled Loader Demo",
		Width:        gameMap.Width * gameMap.TileWidth,
		Height:       gameMap.Height * gameMap.TileHeight,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

// buildAtlas loads the first tileset's image and creates TextureRegion entries
// for every tile GID (index 0 is reserved as empty).
func buildAtlas(gameMap *tiled.Map) (*ebiten.Image, []willow.TextureRegion) {
	ts := gameMap.Tilesets[0]

	imgPath := filepath.Join("examples", "_assets", ts.Image.Source)
	f, err := os.Open(imgPath)
	if err != nil {
		log.Fatalf("open tileset image: %v", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		log.Fatalf("decode tileset image: %v", err)
	}
	atlasImage := ebiten.NewImageFromImage(img)
	regions := willow.RegionsFromGrid(ts.Columns, ts.TileWidth, ts.TileHeight,
		ts.Margin, ts.Spacing, ts.TileCount)

	return atlasImage, regions
}

// convertLayerData turns a go-tiled Layer into the []uint32 GID array that
// Willow expects.  Flip flags are re-encoded into the high bits.
func convertLayerData(gameMap *tiled.Map, layer *tiled.Layer) []uint32 {
	w, h := gameMap.Width, gameMap.Height
	data := make([]uint32, w*h)

	for i, tile := range layer.Tiles {
		if tile.IsNil() {
			continue
		}
		data[i] = willow.EncodeGID(
			tile.ID+tile.Tileset.FirstGID,
			tile.HorizontalFlip, tile.VerticalFlip, tile.DiagonalFlip,
		)
	}
	return data
}

// generateTilesetPNG creates a simple 128×64 tileset (4 columns × 2 rows,
// 32×32 tiles) with distinct colors so we can visually verify the map loads.
//
// GID 1 = dark green  (grass)
// GID 2 = brown       (dirt border)
// GID 3 = tan         (dirt fill)
// GID 4 = dark grey   (stone border)
// GID 5 = light grey  (stone fill)
// GID 6 = blue        (water)
// GID 7 = red         (marker)
// GID 8 = yellow      (highlight)
func generateTilesetPNG(path string) {
	colors := []color.RGBA{
		{34, 139, 34, 255},   // 1: grass
		{139, 90, 43, 255},   // 2: dirt border
		{210, 180, 140, 255}, // 3: dirt fill
		{90, 90, 90, 255},    // 4: stone border
		{170, 170, 170, 255}, // 5: stone fill
		{30, 100, 200, 255},  // 6: water
		{200, 40, 40, 255},   // 7: marker
		{240, 220, 60, 255},  // 8: highlight
	}

	img := image.NewRGBA(image.Rect(0, 0, 128, 64))
	for idx, c := range colors {
		col := idx % 4
		row := idx / 4
		for dy := 0; dy < 32; dy++ {
			for dx := 0; dx < 32; dx++ {
				img.SetRGBA(col*32+dx, row*32+dy, c)
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("create tileset png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatalf("encode tileset png: %v", err)
	}
}
