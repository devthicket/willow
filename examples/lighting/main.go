// Lighting demonstrates the LightLayer system: a dark dungeon scene lit by
// five colored torches and a warm lantern that follows the mouse cursor.
// The center torch wanders autonomously between the four pillar torches using
// tweened movement. Click any torch sprite to toggle its light on or off.
// No external assets are required.
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
	"github.com/tanema/gween/ease"
)

const (
	windowTitle = "Willow  -  Lighting"
	showFPS     = true
	screenW     = 800
	screenH     = 600
)

// torchEntry bundles a light source with its visible sprite and toggle colors.
type torchEntry struct {
	light    *willow.Light
	sprite   *willow.Node
	onColor  willow.Color
	offColor willow.Color
}

type game struct {
	lightLayer    *willow.LightLayer
	torches       []torchEntry
	cursor        *willow.Node
	wanderer      *willow.Node       // container that the center torch rides on
	wandererTween *willow.TweenGroup // active position tween for the wanderer
	pillarPos     [4][2]float64      // world positions of the four pillar torches
	lastPillar    int                // index of the last pillar visited (avoid repeats)
	time          float64
}

func (g *game) update() error {
	g.time += 1.0 / float64(ebiten.TPS())
	t := g.time

	// Keep the lantern centered on the mouse cursor.
	mx, my := ebiten.CursorPosition()
	g.cursor.X = float64(mx)
	g.cursor.Y = float64(my)

	// Step the wanderer tween; when it finishes pick the next pillar.
	dt := float32(1.0 / float64(ebiten.TPS()))
	if g.wandererTween != nil {
		g.wandererTween.Update(dt)
		if g.wandererTween.Done {
			g.wandererTween = nil
			g.nextWander()
		}
	}

	// Animate per-torch flicker by combining two sine waves at different
	// frequencies so each torch has an organic, independent rhythm.
	for i := range g.torches {
		tc := &g.torches[i]
		if !tc.light.Enabled {
			continue
		}
		phase := float64(i) * 2.17
		a := 0.85 + 0.12*math.Sin(t*7.3+phase)
		b := 0.95 + 0.05*math.Sin(t*13.7+phase*1.6)
		tc.light.Intensity = a * b
		tc.light.Radius = 90 + 8*math.Sin(t*4.9+phase*0.8)
	}

	// Rebuild the light texture before the next Draw call.
	g.lightLayer.Redraw()
	return nil
}

// nextWander picks a random pillar position (different from the last) and
// starts a new TweenPosition to move the wanderer container there.
func (g *game) nextWander() {
	next := rand.IntN(4)
	if next == g.lastPillar {
		next = (next + 1) % 4
	}
	g.lastPillar = next
	pos := g.pillarPos[next]
	duration := float32(2.0 + rand.Float64()*2.0)
	g.wandererTween = willow.TweenPosition(g.wanderer, pos[0], pos[1], duration, ease.InOutCubic)
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.07, G: 0.06, B: 0.05, A: 1}

	// ---- Floor tiles --------------------------------------------------------
	// Cover the screen with alternating stone-colored rectangles so the
	// lighting gradient has something interesting to fall on.
	tileW, tileH := 80.0, 80.0
	cols := int(screenW/tileW) + 1
	rows := int(screenH/tileH) + 1
	stoneColors := []willow.Color{
		{R: 0.22, G: 0.20, B: 0.18, A: 1},
		{R: 0.19, G: 0.17, B: 0.15, A: 1},
		{R: 0.25, G: 0.22, B: 0.19, A: 1},
	}
	for row := range rows {
		for col := range cols {
			tile := willow.NewSprite("tile", willow.TextureRegion{})
			tile.X = float64(col) * tileW
			tile.Y = float64(row) * tileH
			tile.ScaleX = tileW - 1 // 1-pixel grout gap
			tile.ScaleY = tileH - 1
			tile.Color = stoneColors[(row*cols+col)%len(stoneColors)]
			scene.Root().AddChild(tile)
		}
	}

	// ---- Stone pillars ------------------------------------------------------
	// Four pillars mark the corners of the inner chamber.
	pillarW, pillarH := 36.0, 160.0
	pillarPos := [4][2]float64{
		{160, 200}, {640, 200},
		{160, 400}, {640, 400},
	}
	for _, pos := range pillarPos {
		p := willow.NewSprite("pillar", willow.TextureRegion{})
		p.X = pos[0] - pillarW/2
		p.Y = pos[1] - pillarH/2
		p.ScaleX = pillarW
		p.ScaleY = pillarH
		p.Color = willow.Color{R: 0.13, G: 0.12, B: 0.11, A: 1}
		scene.Root().AddChild(p)
	}

	// ---- Light layer --------------------------------------------------------
	// ambientAlpha=0.92 makes the dungeon very dark; lights punch through.
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.92)

	// ---- Four pillar torches ------------------------------------------------
	// One fixed torch per pillar, each a unique color.
	type torchDef struct {
		x, y        float64
		lightColor  willow.Color
		spriteColor willow.Color
	}
	fixedDefs := []torchDef{
		{160, 200, // warm amber
			willow.Color{R: 1.0, G: 0.65, B: 0.1, A: 1},
			willow.Color{R: 1.0, G: 0.85, B: 0.4, A: 1}},
		{640, 200, // cool blue (magical)
			willow.Color{R: 0.4, G: 0.6, B: 1.0, A: 1},
			willow.Color{R: 0.6, G: 0.8, B: 1.0, A: 1}},
		{160, 400, // orange
			willow.Color{R: 1.0, G: 0.45, B: 0.05, A: 1},
			willow.Color{R: 1.0, G: 0.65, B: 0.25, A: 1}},
		{640, 400, // purple (arcane)
			willow.Color{R: 0.75, G: 0.25, B: 1.0, A: 1},
			willow.Color{R: 0.88, G: 0.45, B: 1.0, A: 1}},
	}

	offColor := willow.Color{R: 0.20, G: 0.18, B: 0.15, A: 1}
	torches := make([]torchEntry, 0, len(fixedDefs)+1)

	makeTorch := func(parent *willow.Node, localX, localY float64, d torchDef, idx *int) {
		light := &willow.Light{
			X:         d.x,
			Y:         d.y,
			Radius:    90,
			Intensity: 1.0,
			Color:     d.lightColor,
			Enabled:   true,
		}
		lightLayer.AddLight(light)

		flame := willow.NewSprite("torch", willow.TextureRegion{})
		flame.X = localX
		flame.Y = localY
		flame.PivotX = 0.5
		flame.PivotY = 0.5
		flame.ScaleX = 12
		flame.ScaleY = 12
		flame.Color = d.spriteColor
		flame.Interactable = true
		parent.AddChild(flame)

		i := len(torches)
		torches = append(torches, torchEntry{
			light:    light,
			sprite:   flame,
			onColor:  d.spriteColor,
			offColor: offColor,
		})
		if idx != nil {
			*idx = i
		}

		flame.OnClick = func(_ willow.ClickContext) {
			tc := &torches[i]
			tc.light.Enabled = !tc.light.Enabled
			if tc.light.Enabled {
				tc.sprite.Color = tc.onColor
			} else {
				tc.sprite.Color = tc.offColor
				tc.light.Intensity = 0
			}
		}
	}

	for _, d := range fixedDefs {
		makeTorch(scene.Root(), d.x, d.y, d, nil)
	}

	// ---- Wandering center torch ---------------------------------------------
	// A container node acts as the moving anchor. The flame is its child
	// (local position 0,0) and the light uses Target so LightLayer syncs its
	// world position automatically each Redraw. TweenPosition moves the
	// container between pillar positions.
	wanderer := willow.NewContainer("wanderer")
	wanderer.X = 400
	wanderer.Y = 300
	scene.Root().AddChild(wanderer)

	centerDef := torchDef{
		x: 0, y: 0, // local coords inside wanderer
		lightColor:  willow.Color{R: 1.0, G: 0.88, B: 0.4, A: 1},
		spriteColor: willow.Color{R: 1.0, G: 0.95, B: 0.6, A: 1},
	}
	makeTorch(wanderer, 0, 0, centerDef, nil)

	// The center light follows the wanderer via Target instead of fixed X/Y.
	centerLight := torches[len(torches)-1].light
	centerLight.X = 0
	centerLight.Y = 0
	centerLight.Target = wanderer

	// ---- Player lantern (follows the mouse via Target) ----------------------
	cursor := willow.NewContainer("cursor")
	cursor.X = screenW / 2
	cursor.Y = screenH / 2
	scene.Root().AddChild(cursor)

	lantern := &willow.Light{
		Radius:    140,
		Intensity: 0.85,
		Color:     willow.Color{R: 1.0, G: 0.95, B: 0.85, A: 1},
		Enabled:   true,
		Target:    cursor,
	}
	lightLayer.AddLight(lantern)

	// The light layer node must be added after all scene content so it
	// composites on top via BlendMultiply.
	scene.Root().AddChild(lightLayer.Node())

	g := &game{
		lightLayer: lightLayer,
		torches:    torches,
		cursor:     cursor,
		wanderer:   wanderer,
		pillarPos:  pillarPos,
		lastPillar: -1,
	}
	// Kick off the first wander tween immediately.
	g.nextWander()

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
