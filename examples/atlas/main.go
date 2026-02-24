// Atlas demonstrates the TexturePacker atlas system and dynamic page registration.
//
// A 4×2 atlas page is built programmatically (no external assets required):
// eight 32×32 colored tiles are packed into a 128×64 image, described with
// TexturePacker hash-format JSON, and loaded via scene.LoadAtlas.
//
// Layout:
//   - Eight named regions displayed at 80×80 on screen (2.5× of the 32×32 tiles).
//   - A ninth sprite starts as the "dragon" region  -  not in the atlas  -  so it
//     renders as a 1×1 magenta placeholder. With debug mode on, a warning is
//     logged to stderr.
//   - Click the magenta tile to dynamically register whelp.png as atlas page 1,
//     swap its TextureRegion, and reveal the whelp sprite.
//   - Bottom-right: the raw 128×64 atlas page at 2× so you can see the source.
//
// Click any other tile to toggle its alpha between 1.0 and 0.25.
package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Atlas Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480

	tileSize     = 32                     // each tile in the atlas is 32×32 px
	atlasW       = 128                    // 4 columns × tileSize
	atlasH       = 64                     // 2 rows  × tileSize
	displaySize  = 80.0                   // rendered size on screen per tile
	displayScale = displaySize / tileSize // 2.5×
	gridCols     = 4
	gridPad      = 24.0 // gap between tiles
)

var (
	spriteNames = [8]string{
		"fire", "water", "earth", "wind",
		"light", "shadow", "star", "moon",
	}
	tileColors = [8]color.RGBA{
		{R: 255, G: 79, B: 40, A: 255},   // fire
		{R: 40, G: 120, B: 255, A: 255},  // water
		{R: 99, G: 181, B: 61, A: 255},   // earth
		{R: 220, G: 220, B: 79, A: 255},  // wind
		{R: 255, G: 255, B: 199, A: 255}, // light
		{R: 61, G: 40, B: 99, A: 255},    // shadow
		{R: 199, G: 160, B: 255, A: 255}, // star
		{R: 181, G: 200, B: 232, A: 255}, // moon
	}
)

// buildAtlasPage creates a 128×64 atlas image with 8 colored 32×32 tiles.
func buildAtlasPage() *ebiten.Image {
	img := ebiten.NewImage(atlasW, atlasH)
	for i, c := range tileColors {
		col := i % gridCols
		row := i / gridCols
		sub := img.SubImage(image.Rect(
			col*tileSize, row*tileSize,
			(col+1)*tileSize, (row+1)*tileSize,
		)).(*ebiten.Image)
		sub.Fill(c)
	}
	return img
}

// buildAtlasJSON returns TexturePacker hash-format JSON for the 8 tiles.
func buildAtlasJSON() []byte {
	var b strings.Builder
	b.WriteString(`{"frames":{`)
	for i, name := range spriteNames {
		col := i % gridCols
		row := i / gridCols
		x := col * tileSize
		y := row * tileSize
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b,
			`%q:{"frame":{"x":%d,"y":%d,"w":%d,"h":%d},"rotated":false,"trimmed":false,"spriteSourceSize":{"x":0,"y":0,"w":%d,"h":%d},"sourceSize":{"w":%d,"h":%d}}`,
			name, x, y, tileSize, tileSize, tileSize, tileSize, tileSize, tileSize,
		)
	}
	b.WriteString(`}}`)
	return []byte(b.String())
}

func main() {
	atlasPage := buildAtlasPage()
	jsonData := buildAtlasJSON()

	// Load whelp.png for the dynamic reveal. The atlas starts with only the
	// synthetic page; whelp is registered as page 1 on first click.
	whelpImg, _, err := ebitenutil.NewImageFromFile("examples/_assets/whelp.png")
	if err != nil {
		log.Fatalf("load whelp.png: %v", err)
	}

	scene := willow.NewScene()
	scene.SetDebugMode(true) // unknown region names log a warning to stderr
	scene.ClearColor = willow.Color{R: 0.1, G: 0.1, B: 0.15, A: 1}

	// LoadAtlas parses the JSON, registers atlas pages with the scene, and
	// returns an Atlas for named region lookups.
	atlas, err := scene.LoadAtlas(jsonData, []*ebiten.Image{atlasPage})
	if err != nil {
		log.Fatalf("LoadAtlas: %v", err)
	}

	// Grid layout: 4 columns, centered.
	gridW := gridCols*displaySize + (gridCols-1)*gridPad
	startX := (screenW - gridW) / 2.0
	startY := 40.0

	// Spawn the 8 named sprites.
	for i, name := range spriteNames {
		col := float64(i % gridCols)
		row := float64(i / gridCols)
		sp := willow.NewSprite(name, atlas.Region(name))
		sp.X = startX + col*(displaySize+gridPad)
		sp.Y = startY + row*(displaySize+gridPad)
		sp.ScaleX = displayScale
		sp.ScaleY = displayScale
		sp.HitShape = willow.HitRect{Width: tileSize, Height: tileSize}
		sp.OnClick(func(ctx willow.ClickContext) {
			if ctx.Node.Alpha > 0.5 {
				ctx.Node.SetAlpha(0.25)
			} else {
				ctx.Node.SetAlpha(1)
			}
		})
		scene.Root().AddChild(sp)
	}

	// "dragon" is not in the atlas. atlas.Region logs a warning (debug mode) and
	// returns a 1×1 magenta placeholder, scaled up to 80×80 here.
	// Click it to dynamically register whelp.png as atlas page 1 and reveal the sprite.
	const missingName = "dragon"
	missing := willow.NewSprite(missingName, atlas.Region(missingName))
	missing.X = startX
	missing.Y = startY + 2*(displaySize+gridPad)
	missing.ScaleX = displaySize // scale 1px → 80px
	missing.ScaleY = displaySize
	missing.HitShape = willow.HitRect{Width: 1, Height: 1}
	missing.OnClick(func(ctx willow.ClickContext) {
		// Append whelp.png as the next atlas page in both the scene and the Atlas.
		newPageIdx := len(atlas.Pages) // == 1 after initial load
		scene.RegisterPage(newPageIdx, whelpImg)
		atlas.Pages = append(atlas.Pages, whelpImg)

		// Swap the TextureRegion to point at the new page.
		const whelpW, whelpH = 128, 128
		ctx.Node.TextureRegion = willow.TextureRegion{
			Page:      uint16(newPageIdx),
			Width:     whelpW,
			Height:    whelpH,
			OriginalW: whelpW,
			OriginalH: whelpH,
		}
		// Rescale: whelp is 128×128, we still want 80×80 on screen.
		ctx.Node.ScaleX = displaySize / whelpW
		ctx.Node.ScaleY = displaySize / whelpH
		ctx.Node.Invalidate()

		ctx.Node.OnClick(nil) // one-shot
	})
	scene.Root().AddChild(missing)

	// Atlas page preview: raw 128×64 source shown at 2× alongside the placeholder.
	const previewScale = 2.0
	preview := willow.NewSprite("preview", willow.TextureRegion{})
	preview.SetCustomImage(atlasPage)
	preview.X = startX + displaySize + gridPad
	preview.Y = startY + 2*(displaySize+gridPad)
	preview.ScaleX = previewScale
	preview.ScaleY = previewScale
	scene.Root().AddChild(preview)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}
