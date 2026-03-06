// Tweens demonstrates the tween animation system using tiles from the shared
// tileset. Five tiles animate continuously, each showcasing a different tween
// type: position, scale, rotation, alpha, and color. Click anywhere to restart
// all animations with fresh random targets.
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
	"github.com/tanema/gween/ease"
)

const (
	windowTitle = "Willow  -  Tweens Example"
	showFPS     = true
	screenW     = 640
	screenH     = 480
	tileSize    = 32
)

// demo holds the five animated tiles and their active tween groups.
type demo struct {
	scene *willow.Scene

	posNode   *willow.Node
	scaleNode *willow.Node
	rotNode   *willow.Node
	alphaNode *willow.Node
	colorNode *willow.Node

	tweens []*willow.TweenGroup
}

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
	scene.ClearColor = willow.RGB(0.1, 0.1, 0.15)

	// Pick five distinct tiles from the tileset (4x4 grid of 32×32 tiles).
	tileIndices := []int{0, 3, 5, 9, 14}
	tilesPerRow := img.Bounds().Dx() / tileSize
	tiles := make([]*ebiten.Image, len(tileIndices))
	for i, idx := range tileIndices {
		tx := (idx % tilesPerRow) * tileSize
		ty := (idx / tilesPerRow) * tileSize
		tiles[i] = img.SubImage(image.Rect(tx, ty, tx+tileSize, ty+tileSize)).(*ebiten.Image)
	}

	// Helper to create a tile sprite at a position, scaled up 2× for visibility.
	makeSprite := func(name string, tile *ebiten.Image, x, y float64) *willow.Node {
		n := willow.NewSprite(name, willow.TextureRegion{})
		n.SetCustomImage(tile)
		n.SetPosition(x, y)
		n.SetScale(2, 2)
		n.SetPivot(0.5, 0.5)
		scene.Root().AddChild(n)
		return n
	}

	d := &demo{scene: scene}

	// Row layout: five tiles spread across the screen.
	cx := float64(screenW) / 2
	cy := float64(screenH) / 2
	spacing := 110.0
	startX := cx - 2*spacing

	d.posNode = makeSprite("position", tiles[0], startX, cy)
	d.scaleNode = makeSprite("scale", tiles[1], startX+spacing, cy)
	d.rotNode = makeSprite("rotation", tiles[2], startX+2*spacing, cy)
	d.alphaNode = makeSprite("alpha", tiles[3], startX+3*spacing, cy)
	d.colorNode = makeSprite("color", tiles[4], startX+4*spacing, cy)

	// Labels via small colored indicators below each tile.
	addLabel(scene, startX, cy+50, willow.RGB(0.4, 0.8, 1.0))           // position: cyan
	addLabel(scene, startX+spacing, cy+50, willow.RGB(1.0, 0.6, 0.2))   // scale: orange
	addLabel(scene, startX+2*spacing, cy+50, willow.RGB(0.6, 1.0, 0.4)) // rotation: green
	addLabel(scene, startX+3*spacing, cy+50, willow.RGB(0.9, 0.9, 0.3)) // alpha: yellow
	addLabel(scene, startX+4*spacing, cy+50, willow.RGB(1.0, 0.4, 0.7)) // color: pink

	d.startTweens()

	// Click anywhere to restart all tweens.
	scene.OnBackgroundClick(func(ctx willow.ClickContext) {
		d.resetPositions()
		d.startTweens()
	})

	scene.SetUpdateFunc(d.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}

func (d *demo) resetPositions() {
	cx := float64(screenW) / 2
	cy := float64(screenH) / 2
	spacing := 110.0
	startX := cx - 2*spacing

	d.posNode.SetPosition(startX, cy)
	d.scaleNode.SetScale(2, 2)
	d.rotNode.SetRotation(0)
	d.alphaNode.SetAlpha(1)
	d.colorNode.SetColor(willow.RGB(1, 1, 1))
}

func (d *demo) startTweens() {
	cx := float64(screenW) / 2
	cy := float64(screenH) / 2
	spacing := 110.0
	startX := cx - 2*spacing

	// Position: bounce to a random nearby spot and back.
	targetY := cy + (rand.Float64()-0.5)*120
	d.tweens = []*willow.TweenGroup{
		willow.TweenPosition(d.posNode, startX, targetY, willow.TweenConfig{Duration: 1.5, Ease: ease.OutBounce}),
		willow.TweenScale(d.scaleNode, 3.5, 3.5, willow.TweenConfig{Duration: 1.2, Ease: ease.OutElastic}),
		willow.TweenRotation(d.rotNode, math.Pi*2, willow.TweenConfig{Duration: 2.0, Ease: ease.InOutCubic}),
		willow.TweenAlpha(d.alphaNode, 0.1, willow.TweenConfig{Duration: 1.5, Ease: ease.InOutSine}),
		willow.TweenColor(d.colorNode, randomBrightColor(), willow.TweenConfig{Duration: 1.5, Ease: ease.InOutQuad}),
	}
}

func (d *demo) update() error {
	allDone := true
	for _, tw := range d.tweens {
		if !tw.Done {
			allDone = false
		}
	}

	// When all tweens finish, restart with new targets.
	if allDone && len(d.tweens) > 0 {
		d.resetPositions()
		d.startTweens()
	}

	return nil
}

// addLabel places a small colored dot below a tile as a visual indicator.
func addLabel(scene *willow.Scene, x, y float64, c willow.Color) {
	dot := willow.NewRect("label", 8, 4, c)
	dot.SetPosition(x-4, y) // center the 8px-wide dot under the tile
	scene.Root().AddChild(dot)
}

func randomBrightColor() willow.Color {
	return willow.RGB(
		0.3+rand.Float64()*0.7,
		0.3+rand.Float64()*0.7,
		0.3+rand.Float64()*0.7,
	)
}
