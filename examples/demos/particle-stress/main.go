// particle-stress pushes the Willow particle system to 100,000+ simultaneous
// particles across multiple emitters. A live HUD shows the total alive count
// and current FPS. Click to spawn additional burst emitters at the cursor.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"github.com/devthicket/willow"
)

const (
	screenW = 1280
	screenH = 720
)

type demo struct {
	scene    *willow.Scene
	font     *willow.FontFamily
	hud      *willow.Node
	emitters []*willow.Node
	frames   int
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.02, 0.02, 0.04)

	font, err := willow.NewFontFamilyFromFontBundle(willow.GofontBundle)
	if err != nil {
		log.Fatalf("load font: %v", err)
	}

	d := &demo{scene: scene, font: font}
	d.buildEmitters()
	d.buildHUD()

	scene.OnBackgroundClick(func(ctx willow.ClickContext) {
		d.spawnBurst(ctx.GlobalX, ctx.GlobalY)
	})

	scene.SetUpdateFunc(d.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:        "Willow  -  100k Particle Stress Test",
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		Resizable:    true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

func (d *demo) buildEmitters() {
	// Four large fountain emitters across the bottom — each handles 25k particles.
	colors := [][2]willow.Color{
		{willow.RGB(0.3, 0.6, 1.0), willow.RGB(0.7, 0.9, 1.0)}, // blue
		{willow.RGB(1.0, 0.5, 0.1), willow.RGB(1.0, 1.0, 0.3)}, // orange
		{willow.RGB(0.2, 1.0, 0.4), willow.RGB(0.8, 1.0, 0.6)}, // green
		{willow.RGB(1.0, 0.3, 0.6), willow.RGB(1.0, 0.7, 0.9)}, // pink
	}

	xPositions := []float64{160, 440, 720, 1000}

	for i := 0; i < 4; i++ {
		e := willow.NewParticleEmitter(fmt.Sprintf("fountain-%d", i), willow.EmitterConfig{
			MaxParticles: 25_000,
			EmitRate:     12_500,
			Lifetime:     willow.Range{Min: 1.0, Max: 2.0},
			Speed:        willow.Range{Min: 80, Max: 280},
			Angle:        willow.Range{Min: -math.Pi * 0.8, Max: -math.Pi * 0.2},
			StartScale:   willow.Range{Min: 2, Max: 5},
			EndScale:     willow.Range{Min: 0, Max: 1},
			StartAlpha:   willow.Range{Min: 0.7, Max: 1.0},
			EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
			Gravity:      willow.Vec2{X: 0, Y: 200},
			StartColor:   colors[i][0],
			EndColor:     colors[i][1],
			BlendMode:    willow.BlendAdd,
		})
		e.SetPosition(xPositions[i], screenH-40)
		e.Emitter.Start()
		d.scene.Root.AddChild(e)
		d.emitters = append(d.emitters, e)
	}

	// Central radial emitter — another 20k for variety.
	radial := willow.NewParticleEmitter("radial", willow.EmitterConfig{
		MaxParticles: 20_000,
		EmitRate:     10_000,
		Lifetime:     willow.Range{Min: 1.0, Max: 2.0},
		Speed:        willow.Range{Min: 40, Max: 200},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 1, Max: 4},
		EndScale:     willow.Range{Min: 0, Max: 1},
		StartAlpha:   willow.Range{Min: 0.6, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: 30},
		StartColor:   willow.RGB(1.0, 1.0, 0.5),
		EndColor:     willow.RGB(0.6, 0.1, 1.0),
		BlendMode:    willow.BlendAdd,
	})
	radial.SetPosition(screenW/2, screenH/2)
	radial.Emitter.Start()
	d.scene.Root.AddChild(radial)
	d.emitters = append(d.emitters, radial)
}

func (d *demo) buildHUD() {
	bg := willow.NewRect("hud-bg", 220, 30, willow.RGBA(0, 0, 0, 0.7))
	bg.SetPosition(screenW-230, 5)
	d.scene.Root.AddChild(bg)

	d.hud = willow.NewText("hud", "Particles: 0", d.font)
	d.hud.SetPosition(screenW-220, 10)
	d.hud.SetFontSize(20)
	d.hud.SetTextColor(willow.RGB(1, 1, 1))
	d.scene.Root.AddChild(d.hud)
}

func (d *demo) spawnBurst(x, y float64) {
	burst := willow.NewParticleEmitter("burst", willow.EmitterConfig{
		MaxParticles: 5_000,
		EmitRate:     50_000,
		Lifetime:     willow.Range{Min: 0.8, Max: 1.5},
		Speed:        willow.Range{Min: 100, Max: 400},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 2, Max: 6},
		EndScale:     willow.Range{Min: 0, Max: 1},
		StartAlpha:   willow.Range{Min: 0.9, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: 150},
		StartColor:   willow.RGB(1.0, 1.0, 0.8),
		EndColor:     willow.RGB(1.0, 0.2, 0.0),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	burst.SetPosition(x, y)
	burst.Emitter.Start()
	d.scene.Root.AddChild(burst)
	d.emitters = append(d.emitters, burst)
}

func (d *demo) update() error {
	total := 0
	for _, e := range d.emitters {
		total += e.Emitter.AliveCount()
	}
	label := fmt.Sprintf("Particles: %d", total)
	d.hud.TextBlock.Content = label
	d.hud.TextBlock.Invalidate()
	d.frames++
	if d.frames%60 == 0 {
		fmt.Println(label)
	}
	return nil
}
