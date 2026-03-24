// Fireflies demonstrates a nighttime meadow scene with glowing fireflies
// drifting on tweened paths, each casting a warm light. Ambient pollen
// particles float upward through the scene. Combines Willow's lighting,
// particle, tween, and shape systems into a single atmospheric demo.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 320
	screenH = 240
)

type firefly struct {
	sprite *willow.Node
	light  *willow.Light
	tween  *willow.TweenGroup
	phase  float64 // for pulse offset
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.02, 0.03, 0.06)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH/2 + 20
	cam.Zoom = 1.8
	cam.Invalidate()

	// Night sky gradient — darker at top, slightly lighter at horizon.
	for row := range 12 {
		for col := range 16 {
			t := float64(row) / 11.0
			r := 0.08 + 0.12*t
			g := 0.10 + 0.16*t
			b := 0.20 + 0.22*t
			tile := willow.NewSprite("sky", willow.TextureRegion{})
			tile.SetScale(20, 20)
			tile.SetPosition(float64(col)*20, float64(row)*20)
			tile.SetColor(willow.RGB(r, g, b))
			scene.Root.AddChild(tile)
		}
	}

	// Moon — a soft circle in the upper right.
	moon := willow.NewCircle("moon", 18, willow.RGB(0.85, 0.85, 0.75))
	moon.SetPosition(screenW-55, 30)
	moon.SetAlpha(0.7)
	scene.Root.AddChild(moon)

	// Moon glow.
	moonGlow := willow.NewCircle("moon-glow", 35, willow.RGB(0.4, 0.4, 0.35))
	moonGlow.SetPosition(screenW-55, 30)
	moonGlow.SetAlpha(0.15)
	moonGlow.SetBlendMode(willow.BlendAdd)
	scene.Root.AddChild(moonGlow)

	// Rolling hills silhouette — overlapping dark polygons.
	hill1 := willow.NewPolygon("hill1", []willow.Vec2{
		{X: -20, Y: screenH}, {X: -20, Y: 180},
		{X: 60, Y: 155}, {X: 140, Y: 165}, {X: 200, Y: 150},
		{X: 260, Y: 170}, {X: 340, Y: 160}, {X: 340, Y: screenH},
	})
	hill1.SetColor(willow.RGB(0.12, 0.18, 0.08))
	scene.Root.AddChild(hill1)

	hill2 := willow.NewPolygon("hill2", []willow.Vec2{
		{X: -20, Y: screenH}, {X: -20, Y: 195},
		{X: 50, Y: 175}, {X: 120, Y: 190}, {X: 180, Y: 172},
		{X: 250, Y: 185}, {X: 310, Y: 178}, {X: 340, Y: 188}, {X: 340, Y: screenH},
	})
	hill2.SetColor(willow.RGB(0.10, 0.15, 0.06))
	scene.Root.AddChild(hill2)

	// Foreground grass.
	grass := willow.NewPolygon("grass", []willow.Vec2{
		{X: -20, Y: screenH}, {X: -20, Y: 210},
		{X: 40, Y: 205}, {X: 100, Y: 212}, {X: 160, Y: 204},
		{X: 220, Y: 210}, {X: 280, Y: 206}, {X: 340, Y: 208}, {X: 340, Y: screenH},
	})
	grass.SetColor(willow.RGB(0.08, 0.14, 0.06))
	scene.Root.AddChild(grass)

	// Tree silhouettes — simple triangles on trunks.
	trees := [][2]float64{{30, 185}, {90, 178}, {230, 172}, {285, 180}}
	for i, pos := range trees {
		// Trunk.
		trunk := willow.NewSprite("trunk", willow.TextureRegion{})
		trunk.SetScale(4, 30+float64(i)*3)
		trunk.SetPosition(pos[0]-2, pos[1])
		trunk.SetColor(willow.RGB(0.10, 0.12, 0.06))
		scene.Root.AddChild(trunk)

		// Canopy — triangle.
		h := 35.0 + float64(i%3)*10
		w := 22.0 + float64(i%2)*8
		canopy := willow.NewPolygon(fmt.Sprintf("canopy-%d", i), []willow.Vec2{
			{X: 0, Y: -h}, {X: -w, Y: 0}, {X: w, Y: 0},
		})
		canopy.SetPosition(pos[0], pos[1])
		canopy.SetColor(willow.RGB(0.08, 0.14, 0.06))
		scene.Root.AddChild(canopy)
	}

	// Pollen/dust particles floating upward.
	pollenNode := willow.NewParticleEmitter("pollen", willow.EmitterConfig{
		MaxParticles: 80,
		EmitRate:     8,
		Lifetime:     willow.Range{Min: 3.0, Max: 6.0},
		Speed:        willow.Range{Min: 5, Max: 15},
		Angle:        willow.Range{Min: -math.Pi/2 - 0.4, Max: -math.Pi/2 + 0.4},
		StartScale:   willow.Range{Min: 1, Max: 2},
		EndScale:     willow.Range{Min: 0.5, Max: 1},
		StartAlpha:   willow.Range{Min: 0.15, Max: 0.35},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 8, Y: -3},
		StartColor:   willow.RGB(0.7, 0.75, 0.5),
		EndColor:     willow.RGB(0.5, 0.6, 0.4),
		BlendMode:    willow.BlendAdd,
	})
	pollenNode.SetPosition(screenW/2, screenH-20)
	pollenNode.SetScale(screenW, 1)
	pollenNode.Emitter.Start()
	scene.Root.AddChild(pollenNode)

	// Lighting layer.
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.70)

	// Moon light — dim wide glow.
	moonLight := &willow.Light{
		X: screenW - 55, Y: 30,
		Radius: 180, Intensity: 0.25,
		Color:   willow.RGB(0.6, 0.6, 0.8),
		Enabled: true,
	}
	lightLayer.AddLight(moonLight)

	// Fireflies — small sprites with tracked lights.
	numFireflies := 12
	flies := make([]firefly, numFireflies)
	flyColors := []willow.Color{
		willow.RGB(1.0, 0.85, 0.3), // warm gold
		willow.RGB(0.8, 1.0, 0.3),  // yellow-green
		willow.RGB(1.0, 0.7, 0.2),  // amber
		willow.RGB(0.6, 1.0, 0.4),  // green
	}

	for i := range numFireflies {
		sx := 30 + rand.Float64()*(screenW-60)
		sy := 80 + rand.Float64()*120
		c := flyColors[i%len(flyColors)]

		sprite := willow.NewSprite("fly", willow.TextureRegion{})
		sprite.SetScale(2, 2)
		sprite.SetPosition(sx, sy)
		sprite.SetColor(c)
		sprite.SetBlendMode(willow.BlendAdd)
		scene.Root.AddChild(sprite)

		// Glow halo around the firefly.
		glow := willow.NewSprite("glow", willow.TextureRegion{})
		glow.SetScale(8, 8)
		glow.SetPivot(0.5, 0.5)
		glow.SetPosition(sx+1, sy+1)
		glow.SetColor(c)
		glow.SetAlpha(0.15)
		glow.SetBlendMode(willow.BlendAdd)
		sprite.AddChild(glow)

		light := &willow.Light{
			Radius:    30 + rand.Float64()*20,
			Intensity: 0.5 + rand.Float64()*0.3,
			Color:     c,
			Enabled:   true,
			Target:    sprite,
			OffsetX:   1, OffsetY: 1,
		}
		lightLayer.AddLight(light)

		// Initial tween to a random position.
		tx := 20 + rand.Float64()*(screenW-40)
		ty := 70 + rand.Float64()*130
		dur := float32(2.5 + rand.Float64()*4.0)
		tw := willow.TweenPosition(sprite, tx, ty, willow.TweenConfig{
			Duration: dur,
			Ease:     willow.EaseInOutSine,
		})

		flies[i] = firefly{
			sprite: sprite,
			light:  light,
			tween:  tw,
			phase:  rand.Float64() * math.Pi * 2,
		}
	}

	scene.Root.AddChild(lightLayer.Node())

	elapsed := 0.0

	updateFn := func() error {
		dt := 1.0 / float64(ebiten.TPS())
		elapsed += dt

		for i := range flies {
			f := &flies[i]

			// Pulse the light — gentle breathing glow.
			pulse := 0.5 + 0.4*math.Sin(elapsed*2.5+f.phase)
			f.light.Intensity = pulse * 0.7
			f.light.Radius = 25 + 20*pulse

			// Pulse sprite alpha to match.
			f.sprite.SetAlpha(0.4 + 0.6*pulse)
			f.sprite.Invalidate()

			// When tween completes, pick a new destination.
			if f.tween.Done {
				tx := 20 + rand.Float64()*(screenW-40)
				ty := 70 + rand.Float64()*130
				dur := float32(2.0 + rand.Float64()*4.0)
				f.tween = willow.TweenPosition(f.sprite, tx, ty, willow.TweenConfig{
					Duration: dur,
					Ease:     willow.EaseInOutSine,
				})
			}
		}

		// Moon light gentle pulse.
		moonLight.Intensity = 0.22 + 0.05*math.Sin(elapsed*0.8)

		lightLayer.Redraw()
		return nil
	}

	if *autotest != "" {
		scriptData, err := os.ReadFile(*autotest)
		if err != nil {
			log.Fatalf("read test script: %v", err)
		}
		runner, err := willow.LoadTestScript(scriptData)
		if err != nil {
			log.Fatalf("parse test script: %v", err)
		}
		scene.SetTestRunner(runner)
		scene.ScreenshotDir = "screenshots"
		scene.SetUpdateFunc(func() error {
			if err := updateFn(); err != nil {
				return err
			}
			if runner.Done() {
				fmt.Println("Autotest complete.")
				return ebiten.Termination
			}
			return nil
		})
	} else {
		scene.SetUpdateFunc(updateFn)
	}

	if err := willow.Run(scene, willow.RunConfig{
		Title:   "Willow  -  Fireflies",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: false,
	}); err != nil {
		log.Fatal(err)
	}
}
