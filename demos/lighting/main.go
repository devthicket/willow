// Lighting Demo showcases the LightLayer system in a dark dungeon scene.
// Heavy ambient darkness makes lighting dramatic  -  torches flicker on pillars,
// magical wisps drift autonomously, and a lantern follows the cursor.
// Click anywhere to spawn a temporary flash. Click a torch to toggle it.
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
	"github.com/tanema/gween/ease"
)

const (
	windowTitle = "Willow  -  Lighting Demo"
	showFPS     = true
	screenW     = 800
	screenH     = 600
)

// torchEntry bundles a light source with its visible sprite and toggle state.
type torchEntry struct {
	light    *willow.Light
	sprite   *willow.Node
	onColor  willow.Color
	offColor willow.Color
}

// wisp is an autonomous floating light that drifts around the scene.
type wisp struct {
	node  *willow.Node
	light *willow.Light
	tween *willow.TweenGroup
}

// flash is a temporary burst light that fades out.
type flash struct {
	light     *willow.Light
	sprite    *willow.Node
	remaining float64
}

type game struct {
	scene      *willow.Scene
	lightLayer *willow.LightLayer
	torches    []torchEntry
	wisps      []wisp
	flashes    []*flash
	cursor     *willow.Node
	time       float64
}

func (g *game) update() error {
	g.time += 1.0 / float64(ebiten.TPS())
	t := g.time
	dt := float32(1.0 / float64(ebiten.TPS()))

	// Lantern follows cursor.
	mx, my := ebiten.CursorPosition()
	g.cursor.X = float64(mx)
	g.cursor.Y = float64(my)

	// Torch flicker.
	for i := range g.torches {
		tc := &g.torches[i]
		if !tc.light.Enabled {
			continue
		}
		phase := float64(i) * 2.17
		a := 0.82 + 0.15*math.Sin(t*7.3+phase)
		b := 0.92 + 0.08*math.Sin(t*13.7+phase*1.6)
		tc.light.Intensity = a * b
		tc.light.Radius = 120 + 12*math.Sin(t*4.9+phase*0.8)
	}

	// Wisp animation: drift to new positions when tween completes.
	for i := range g.wisps {
		w := &g.wisps[i]
		w.tween.Update(dt)
		if w.tween.Done {
			// Pick a new target within bounds.
			tx := 80 + rand.Float64()*(screenW-160)
			ty := 80 + rand.Float64()*(screenH-160)
			dur := float32(3.0 + rand.Float64()*4.0)
			w.tween = willow.TweenPosition(w.node, tx, ty, dur, ease.InOutSine)
		}
		// Pulse the wisp light.
		phase := float64(i) * 3.14
		w.light.Intensity = 0.6 + 0.3*math.Sin(t*2.5+phase)
		w.light.Radius = 50 + 15*math.Sin(t*1.8+phase*0.7)
	}

	// Flash decay.
	alive := g.flashes[:0]
	for _, f := range g.flashes {
		f.remaining -= 1.0 / float64(ebiten.TPS())
		if f.remaining <= 0 {
			g.lightLayer.RemoveLight(f.light)
			f.sprite.RemoveFromParent()
			continue
		}
		// Fade out.
		frac := f.remaining / 0.6
		f.light.Intensity = frac
		f.light.Radius = 180 * frac
		f.sprite.Alpha = frac
		f.sprite.Invalidate()
		alive = append(alive, f)
	}
	g.flashes = alive

	g.lightLayer.Redraw()
	return nil
}

func (g *game) spawnFlash(x, y float64) {
	color := willow.Color{
		R: 0.8 + rand.Float64()*0.2,
		G: 0.7 + rand.Float64()*0.3,
		B: 0.3 + rand.Float64()*0.4,
		A: 1,
	}

	light := &willow.Light{
		X:         x,
		Y:         y,
		Radius:    180,
		Intensity: 1.0,
		Color:     color,
		Enabled:   true,
	}
	g.lightLayer.AddLight(light)

	// Visual burst sprite.
	sp := willow.NewSprite("flash", willow.TextureRegion{})
	sp.ScaleX = 20
	sp.ScaleY = 20
	sp.PivotX = 0.5
	sp.PivotY = 0.5
	sp.X = x
	sp.Y = y
	sp.Color = color
	// Insert before the light layer node (second to last child).
	g.scene.Root().AddChild(sp)

	g.flashes = append(g.flashes, &flash{
		light:     light,
		sprite:    sp,
		remaining: 0.6,
	})
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.04, G: 0.03, B: 0.03, A: 1}

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// ---- Stone floor --------------------------------------------------------
	tileW, tileH := 60.0, 60.0
	cols := int(screenW/tileW) + 1
	rows := int(screenH/tileH) + 1
	stoneColors := []willow.Color{
		{R: 0.18, G: 0.16, B: 0.14, A: 1},
		{R: 0.15, G: 0.13, B: 0.11, A: 1},
		{R: 0.20, G: 0.17, B: 0.14, A: 1},
		{R: 0.16, G: 0.14, B: 0.12, A: 1},
	}
	for row := range rows {
		for col := range cols {
			tile := willow.NewSprite("tile", willow.TextureRegion{})
			tile.X = float64(col) * tileW
			tile.Y = float64(row) * tileH
			tile.ScaleX = tileW - 1
			tile.ScaleY = tileH - 1
			tile.Color = stoneColors[(row*cols+col+row)%len(stoneColors)]
			scene.Root().AddChild(tile)
		}
	}

	// ---- Walls around edges -------------------------------------------------
	wallColor := willow.Color{R: 0.10, G: 0.09, B: 0.08, A: 1}
	wallThick := 24.0
	walls := [][4]float64{
		{0, 0, screenW, wallThick},                   // top
		{0, screenH - wallThick, screenW, wallThick}, // bottom
		{0, 0, wallThick, screenH},                   // left
		{screenW - wallThick, 0, wallThick, screenH}, // right
	}
	for _, w := range walls {
		wall := willow.NewSprite("wall", willow.TextureRegion{})
		wall.X = w[0]
		wall.Y = w[1]
		wall.ScaleX = w[2]
		wall.ScaleY = w[3]
		wall.Color = wallColor
		scene.Root().AddChild(wall)
	}

	// ---- Pillars ------------------------------------------------------------
	type pillarDef struct {
		x, y float64
	}
	pillarDefs := []pillarDef{
		{180, 160}, {620, 160},
		{180, 440}, {620, 440},
		{400, 300}, // center pillar
	}
	pillarW, pillarH := 40.0, 40.0
	pillarColor := willow.Color{R: 0.12, G: 0.11, B: 0.10, A: 1}
	for _, pd := range pillarDefs {
		p := willow.NewSprite("pillar", willow.TextureRegion{})
		p.X = pd.x - pillarW/2
		p.Y = pd.y - pillarH/2
		p.ScaleX = pillarW
		p.ScaleY = pillarH
		p.Color = pillarColor
		scene.Root().AddChild(p)
	}

	// ---- Treasure and crates ------------------------------------------------
	// Scattered objects to show how lighting reveals scene detail.
	crateColor := willow.Color{R: 0.35, G: 0.22, B: 0.10, A: 1}
	cratePositions := [][2]float64{
		{100, 100}, {700, 100}, {100, 500}, {700, 500},
		{300, 200}, {500, 400}, {350, 480},
	}
	for _, pos := range cratePositions {
		crate := willow.NewSprite("crate", willow.TextureRegion{})
		crate.X = pos[0]
		crate.Y = pos[1]
		crate.ScaleX = 22
		crate.ScaleY = 22
		crate.Color = crateColor
		scene.Root().AddChild(crate)
	}

	// Gem clusters near center.
	gemColors := []willow.Color{
		{R: 0.2, G: 0.8, B: 0.3, A: 1},
		{R: 0.8, G: 0.2, B: 0.3, A: 1},
		{R: 0.3, G: 0.3, B: 0.9, A: 1},
	}
	gemPositions := [][2]float64{{370, 260}, {430, 260}, {400, 340}}
	for i, pos := range gemPositions {
		gem := willow.NewSprite("gem", willow.TextureRegion{})
		gem.X = pos[0]
		gem.Y = pos[1]
		gem.ScaleX = 10
		gem.ScaleY = 10
		gem.Color = gemColors[i%len(gemColors)]
		scene.Root().AddChild(gem)
	}

	// ---- Light layer (heavy darkness) ---------------------------------------
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.96)

	// ---- Torch lights on pillars --------------------------------------------
	type torchDef struct {
		x, y        float64
		lightColor  willow.Color
		spriteColor willow.Color
	}
	torchDefs := []torchDef{
		{180, 160,
			willow.Color{R: 1.0, G: 0.65, B: 0.1, A: 1},
			willow.Color{R: 1.0, G: 0.85, B: 0.4, A: 1}},
		{620, 160,
			willow.Color{R: 0.4, G: 0.6, B: 1.0, A: 1},
			willow.Color{R: 0.6, G: 0.8, B: 1.0, A: 1}},
		{180, 440,
			willow.Color{R: 1.0, G: 0.45, B: 0.05, A: 1},
			willow.Color{R: 1.0, G: 0.65, B: 0.25, A: 1}},
		{620, 440,
			willow.Color{R: 0.75, G: 0.25, B: 1.0, A: 1},
			willow.Color{R: 0.88, G: 0.45, B: 1.0, A: 1}},
		{400, 300,
			willow.Color{R: 0.9, G: 0.9, B: 0.5, A: 1},
			willow.Color{R: 1.0, G: 1.0, B: 0.7, A: 1}},
	}

	offColor := willow.Color{R: 0.15, G: 0.13, B: 0.10, A: 1}
	torches := make([]torchEntry, 0, len(torchDefs))

	for _, td := range torchDefs {
		light := &willow.Light{
			X:         td.x,
			Y:         td.y,
			Radius:    120,
			Intensity: 1.0,
			Color:     td.lightColor,
			Enabled:   true,
		}
		lightLayer.AddLight(light)

		flame := willow.NewSprite("torch", willow.TextureRegion{})
		flame.X = td.x
		flame.Y = td.y
		flame.PivotX = 0.5
		flame.PivotY = 0.5
		flame.ScaleX = 14
		flame.ScaleY = 14
		flame.Color = td.spriteColor
		flame.Interactable = true
		scene.Root().AddChild(flame)

		idx := len(torches)
		torches = append(torches, torchEntry{
			light:    light,
			sprite:   flame,
			onColor:  td.spriteColor,
			offColor: offColor,
		})

		flame.OnClick = func(_ willow.ClickContext) {
			tc := &torches[idx]
			tc.light.Enabled = !tc.light.Enabled
			if tc.light.Enabled {
				tc.sprite.Color = tc.onColor
			} else {
				tc.sprite.Color = tc.offColor
				tc.light.Intensity = 0
			}
			tc.sprite.Invalidate()
		}
	}

	// ---- Wisps (autonomous floating lights) ---------------------------------
	wispColors := []willow.Color{
		{R: 0.3, G: 1.0, B: 0.5, A: 1},
		{R: 0.5, G: 0.8, B: 1.0, A: 1},
		{R: 1.0, G: 0.7, B: 0.3, A: 1},
	}
	wisps := make([]wisp, 3)
	for i := range wisps {
		sx := 150 + rand.Float64()*(screenW-300)
		sy := 150 + rand.Float64()*(screenH-300)

		node := willow.NewSprite("wisp", willow.TextureRegion{})
		node.X = sx
		node.Y = sy
		node.PivotX = 0.5
		node.PivotY = 0.5
		node.ScaleX = 8
		node.ScaleY = 8
		node.Color = wispColors[i]
		scene.Root().AddChild(node)

		light := &willow.Light{
			Radius:    50,
			Intensity: 0.7,
			Color:     wispColors[i],
			Enabled:   true,
			Target:    node,
		}
		lightLayer.AddLight(light)

		tx := 80 + rand.Float64()*(screenW-160)
		ty := 80 + rand.Float64()*(screenH-160)
		dur := float32(3.0 + rand.Float64()*4.0)
		tw := willow.TweenPosition(node, tx, ty, dur, ease.InOutSine)

		wisps[i] = wisp{node: node, light: light, tween: tw}
	}

	// ---- Cursor lantern -----------------------------------------------------
	cursor := willow.NewContainer("cursor")
	cursor.X = screenW / 2
	cursor.Y = screenH / 2
	scene.Root().AddChild(cursor)

	lantern := &willow.Light{
		Radius:    160,
		Intensity: 0.9,
		Color:     willow.Color{R: 1.0, G: 0.95, B: 0.85, A: 1},
		Enabled:   true,
		Target:    cursor,
	}
	lightLayer.AddLight(lantern)

	// Light layer must be added last  -  composites darkness over everything.
	scene.Root().AddChild(lightLayer.Node())

	// ---- HUD labels (added after light layer so they're always visible) -----
	hintText := "Move mouse to explore. Click torches to toggle. Click empty space for a flash."
	hint := makeLabel(hintText)
	hint.X = screenW/2 - float64(len(hintText)*6)/2
	hint.Y = screenH - 20
	scene.Root().AddChild(hint)

	// ---- Click handler for flash spawning -----------------------------------
	g := &game{
		scene:      scene,
		lightLayer: lightLayer,
		torches:    torches,
		wisps:      wisps,
		cursor:     cursor,
	}

	scene.Root().HitShape = willow.HitRect{Width: screenW, Height: screenH}
	scene.Root().Interactable = true
	scene.Root().OnClick = func(ctx willow.ClickContext) {
		g.spawnFlash(ctx.GlobalX, ctx.GlobalY)
	}

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

// makeLabel pre-renders a text string using Ebitengine's debug font.
func makeLabel(s string) *willow.Node {
	w := len(s)*6 + 2
	h := 16
	img := ebiten.NewImage(w, h)
	ebitenutil.DebugPrint(img, s)

	n := willow.NewSprite("label-"+s, willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}
