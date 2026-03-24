// Lightning Showcase demonstrates a dramatic lightning strike effect combining
// the lighting system, particle sparks, and screen flash.
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

// bolt holds the geometry for a single lightning bolt rendered as sprites.
type bolt struct {
	nodes []*willow.Node
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.03, 0.03, 0.06)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Stormy sky — brighter so the light layer has something to darken.
	skyColors := []willow.Color{
		willow.RGB(0.25, 0.22, 0.40),
		willow.RGB(0.30, 0.26, 0.45),
		willow.RGB(0.22, 0.20, 0.38),
	}
	for row := range 12 {
		for col := range 16 {
			tile := willow.NewSprite("sky", willow.TextureRegion{})
			tile.SetScale(20, 20)
			tile.SetPosition(float64(col)*20, float64(row)*20)
			idx := row / 4
			if idx >= len(skyColors) {
				idx = len(skyColors) - 1
			}
			tile.SetColor(skyColors[idx])
			scene.Root.AddChild(tile)
		}
	}

	// Ground strip.
	for col := range 16 {
		g := willow.NewSprite("ground", willow.TextureRegion{})
		g.SetScale(20, 40)
		g.SetPosition(float64(col)*20, screenH-40)
		g.SetColor(willow.RGB(0.30, 0.35, 0.20))
		scene.Root.AddChild(g)
	}

	// Lighting layer — darkens the scene, strike light punches through.
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.65)
	strikeLight := &willow.Light{
		X: screenW / 2, Y: screenH / 2,
		Radius: 10, Intensity: 0,
		Color:   willow.RGB(0.8, 0.8, 1.0),
		Enabled: true,
	}
	lightLayer.AddLight(strikeLight)

	ambientLight := &willow.Light{
		X: screenW / 2, Y: screenH / 2,
		Radius: 300, Intensity: 0.15,
		Color:   willow.RGB(0.4, 0.4, 0.7),
		Enabled: true,
	}
	lightLayer.AddLight(ambientLight)
	scene.Root.AddChild(lightLayer.Node())

	// Everything below renders ABOVE the light layer so it isn't darkened.

	// Container for bolt segments.
	boltContainer := willow.NewContainer("bolts")
	scene.Root.AddChild(boltContainer)

	// Spark particle emitter.
	sparkNode := willow.NewParticleEmitter("sparks", willow.EmitterConfig{
		MaxParticles: 400,
		EmitRate:     1200,
		Lifetime:     willow.Range{Min: 0.3, Max: 0.8},
		Speed:        willow.Range{Min: 80, Max: 300},
		Angle:        willow.Range{Min: -math.Pi, Max: math.Pi},
		StartScale:   willow.Range{Min: 2, Max: 5},
		EndScale:     willow.Range{Min: 0, Max: 1},
		StartAlpha:   willow.Range{Min: 0.9, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: 300},
		StartColor:   willow.RGB(1.0, 0.95, 0.8),
		EndColor:     willow.RGB(0.6, 0.4, 1.0),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	scene.Root.AddChild(sparkNode)

	// Ember emitter — slower rising glow.
	emberNode := willow.NewParticleEmitter("embers", willow.EmitterConfig{
		MaxParticles: 200,
		EmitRate:     600,
		Lifetime:     willow.Range{Min: 0.5, Max: 1.2},
		Speed:        willow.Range{Min: 20, Max: 100},
		Angle:        willow.Range{Min: -math.Pi, Max: math.Pi},
		StartScale:   willow.Range{Min: 3, Max: 8},
		EndScale:     willow.Range{Min: 1, Max: 2},
		StartAlpha:   willow.Range{Min: 0.6, Max: 0.8},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: -40},
		StartColor:   willow.RGB(0.5, 0.3, 1.0),
		EndColor:     willow.RGB(0.2, 0.1, 0.5),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	scene.Root.AddChild(emberNode)

	// Screen flash overlay.
	flash := willow.NewSprite("flash", willow.TextureRegion{})
	flash.SetScale(screenW, screenH)
	flash.SetColor(willow.RGB(0.8, 0.85, 1.0))
	flash.SetAlpha(0)
	flash.SetBlendMode(willow.BlendAdd)
	scene.Root.AddChild(flash)

	// State.
	elapsed := 0.0
	flashAlpha := 0.0
	strikeCooldown := 0.5
	var activeBolt *bolt

	generateBolt := func(targetX, targetY float64) *bolt {
		b := &bolt{}
		x := targetX + (rand.Float64()-0.5)*40
		y := 0.0
		segments := 12 + rand.IntN(8)
		stepY := targetY / float64(segments)

		for i := range segments {
			nextX := x + (rand.Float64()-0.5)*40
			nextY := y + stepY

			dx := nextX - x
			dy := nextY - y
			length := math.Sqrt(dx*dx + dy*dy)
			angle := math.Atan2(dy, dx)
			thickness := 3.0 - float64(i)*0.1
			if thickness < 1.2 {
				thickness = 1.2
			}

			// Main bolt segment.
			seg := willow.NewSprite("bolt-seg", willow.TextureRegion{})
			seg.SetScale(length, thickness)
			seg.SetPosition(x, y)
			seg.SetRotation(angle)
			seg.SetColor(willow.RGB(0.9, 0.9, 1.0))
			seg.SetBlendMode(willow.BlendAdd)
			boltContainer.AddChild(seg)
			b.nodes = append(b.nodes, seg)

			// Glow halo.
			glow := willow.NewSprite("bolt-glow", willow.TextureRegion{})
			glow.SetScale(length, thickness*5)
			glow.SetPosition(x, y)
			glow.SetRotation(angle)
			glow.SetColor(willow.RGB(0.5, 0.5, 1.0))
			glow.SetAlpha(0.35)
			glow.SetBlendMode(willow.BlendAdd)
			boltContainer.AddChild(glow)
			b.nodes = append(b.nodes, glow)

			// Random branch.
			if rand.Float64() < 0.35 {
				branchAngle := angle + (rand.Float64()-0.5)*1.2
				branchLen := length * (0.3 + rand.Float64()*0.5)
				br := willow.NewSprite("branch", willow.TextureRegion{})
				br.SetScale(branchLen, 1.2)
				br.SetPosition(nextX, nextY)
				br.SetRotation(branchAngle)
				br.SetColor(willow.RGB(0.7, 0.7, 1.0))
				br.SetAlpha(0.6)
				br.SetBlendMode(willow.BlendAdd)
				boltContainer.AddChild(br)
				b.nodes = append(b.nodes, br)
			}

			x = nextX
			y = nextY
		}
		return b
	}

	clearBolt := func() {
		if activeBolt != nil {
			for _, n := range activeBolt.nodes {
				n.RemoveFromParent()
			}
			activeBolt = nil
		}
	}

	sparkBurstTimer := 0.0
	emberBurstTimer := 0.0
	boltFadeTimer := 0.0

	triggerStrike := func() {
		clearBolt()

		targetX := 60 + rand.Float64()*(screenW-120)
		targetY := screenH - 40 + rand.Float64()*10

		activeBolt = generateBolt(targetX, targetY)
		flashAlpha = 0.8 + rand.Float64()*0.2
		sparkBurstTimer = 0.12
		emberBurstTimer = 0.20
		boltFadeTimer = 0.35

		sparkNode.SetPosition(targetX, targetY)
		sparkNode.Emitter.Reset()
		sparkNode.Emitter.Start()

		emberNode.SetPosition(targetX, targetY-10)
		emberNode.Emitter.Reset()
		emberNode.Emitter.Start()

		strikeLight.X = targetX
		strikeLight.Y = targetY
		strikeLight.Radius = 350
		strikeLight.Intensity = 1.0
	}

	updateFn := func() error {
		dt := 1.0 / float64(ebiten.TPS())
		elapsed += dt

		strikeCooldown -= dt
		if strikeCooldown <= 0 {
			triggerStrike()
			strikeCooldown = 0.7 + rand.Float64()*0.8
		}

		if sparkBurstTimer > 0 {
			sparkBurstTimer -= dt
			if sparkBurstTimer <= 0 {
				sparkNode.Emitter.Stop()
			}
		}
		if emberBurstTimer > 0 {
			emberBurstTimer -= dt
			if emberBurstTimer <= 0 {
				emberNode.Emitter.Stop()
			}
		}

		// Fade bolt.
		if boltFadeTimer > 0 {
			boltFadeTimer -= dt
			if boltFadeTimer <= 0 {
				clearBolt()
			} else if activeBolt != nil {
				a := boltFadeTimer / 0.35
				for _, n := range activeBolt.nodes {
					n.SetAlpha(a)
					n.Invalidate()
				}
			}
		}

		// Flash decay.
		if flashAlpha > 0 {
			flashAlpha -= dt * 3.0
			if flashAlpha < 0 {
				flashAlpha = 0
			}
			flash.SetAlpha(flashAlpha)
			flash.Invalidate()
		}

		// Strike light decay.
		if strikeLight.Intensity > 0 {
			strikeLight.Intensity -= dt * 3.0
			if strikeLight.Intensity < 0 {
				strikeLight.Intensity = 0
			}
			strikeLight.Radius *= 0.96
		}

		ambientLight.Intensity = 0.12 + 0.05*math.Sin(elapsed*8)
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
		Title:   "Willow  -  Lightning",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: false,
	}); err != nil {
		log.Fatal(err)
	}
}
