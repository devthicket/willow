// Particles demonstrates the ParticleEmitter system with three distinct effects
// and two blend modes. A fountain arcs upward with gravity (BlendNormal); a
// campfire burns with layered fire and smoke; a sparkler radiates in all
// directions (BlendAdd). Click anywhere to trigger an explosion burst at the
// cursor position.
package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Particles"
	showFPS     = true
	screenW     = 800
	screenH     = 600
)

type demo struct {
	burst      *willow.Node
	burstTimer float64 // seconds remaining before burst stops emitting
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.04, 0.04, 0.07)

	// --- Fountain (left)  -  BlendNormal, gravity --------------------------------
	// Particles arc upward and fall back down; alpha and scale fade at death.
	fountain := willow.NewParticleEmitter("fountain", willow.EmitterConfig{
		MaxParticles: 300,
		EmitRate:     80,
		Lifetime:     willow.Range{Min: 1.4, Max: 2.2},
		Speed:        willow.Range{Min: 80, Max: 180},
		Angle:        willow.Range{Min: -math.Pi * 0.72, Max: -math.Pi * 0.28},
		StartScale:   willow.Range{Min: 4, Max: 7},
		EndScale:     willow.Range{Min: 1, Max: 2},
		StartAlpha:   willow.Range{Min: 0.8, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.05},
		Gravity:      willow.Vec2{X: 0, Y: 240},
		StartColor:   willow.RGB(0.3, 0.65, 1.0),
		EndColor:     willow.RGB(0.85, 0.95, 1.0),
		BlendMode:    willow.BlendNormal,
	})
	fountain.SetPosition(160, 500)
	fountain.Emitter.Start()
	scene.Root().AddChild(fountain)

	// Decorative base plate under the fountain.
	addBase(scene, 160, 505, willow.RGB(0.2, 0.35, 0.5))

	// --- Campfire (center)  -  fire in BlendAdd, smoke in BlendNormal -----------
	// Fire uses additive blending so overlapping particles brighten naturally.
	fire := willow.NewParticleEmitter("fire", willow.EmitterConfig{
		MaxParticles: 220,
		EmitRate:     130,
		Lifetime:     willow.Range{Min: 0.4, Max: 1.0},
		Speed:        willow.Range{Min: 35, Max: 90},
		Angle:        willow.Range{Min: -math.Pi * 0.68, Max: -math.Pi * 0.32},
		StartScale:   willow.Range{Min: 5, Max: 10},
		EndScale:     willow.Range{Min: 1, Max: 3},
		StartAlpha:   willow.Range{Min: 0.75, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: -15},
		StartColor:   willow.RGB(1.0, 0.75, 0.15),
		EndColor:     willow.RGB(0.9, 0.1, 0.0),
		BlendMode:    willow.BlendAdd,
	})
	fire.SetPosition(400, 500)
	fire.Emitter.Start()
	scene.Root().AddChild(fire)

	// Smoke rises above the fire base; alpha is low so it feels wispy.
	smoke := willow.NewParticleEmitter("smoke", willow.EmitterConfig{
		MaxParticles: 60,
		EmitRate:     12,
		Lifetime:     willow.Range{Min: 2.5, Max: 4.0},
		Speed:        willow.Range{Min: 15, Max: 45},
		Angle:        willow.Range{Min: -math.Pi * 0.75, Max: -math.Pi * 0.25},
		StartScale:   willow.Range{Min: 8, Max: 14},
		EndScale:     willow.Range{Min: 28, Max: 50},
		StartAlpha:   willow.Range{Min: 0.12, Max: 0.22},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 8, Y: -5},
		StartColor:   willow.RGB(0.45, 0.45, 0.45),
		EndColor:     willow.RGB(0.25, 0.25, 0.28),
		BlendMode:    willow.BlendNormal,
	})
	smoke.SetPosition(400, 490)
	smoke.Emitter.Start()
	scene.Root().AddChild(smoke)

	addBase(scene, 400, 505, willow.RGB(0.45, 0.3, 0.1))

	// --- Sparkler (right)  -  BlendAdd, radial ----------------------------------
	// Particles shoot out in all directions with slight gravity; the additive
	// blend makes colors bloom against the dark background.
	sparkler := willow.NewParticleEmitter("sparkler", willow.EmitterConfig{
		MaxParticles: 250,
		EmitRate:     70,
		Lifetime:     willow.Range{Min: 0.6, Max: 1.6},
		Speed:        willow.Range{Min: 25, Max: 130},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 2, Max: 6},
		EndScale:     willow.Range{Min: 0, Max: 1},
		StartAlpha:   willow.Range{Min: 0.8, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: 40},
		StartColor:   willow.RGB(1.0, 0.95, 0.3),
		EndColor:     willow.RGB(0.5, 0.1, 0.9),
		BlendMode:    willow.BlendAdd,
	})
	sparkler.SetPosition(640, 500)
	sparkler.Emitter.Start()
	scene.Root().AddChild(sparkler)

	addBase(scene, 640, 505, willow.RGB(0.4, 0.2, 0.5))

	// --- Burst emitter (triggered on click) -----------------------------------
	// WorldSpace=true keeps particles in world-space after the emitter moves;
	// this way the explosion stays at the click position even if re-triggered.
	burst := willow.NewParticleEmitter("burst", willow.EmitterConfig{
		MaxParticles: 180,
		EmitRate:     900,
		Lifetime:     willow.Range{Min: 0.4, Max: 1.1},
		Speed:        willow.Range{Min: 60, Max: 280},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 3, Max: 8},
		EndScale:     willow.Range{Min: 0, Max: 2},
		StartAlpha:   willow.Range{Min: 0.9, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: 180},
		StartColor:   willow.RGB(1.0, 1.0, 0.6),
		EndColor:     willow.RGB(1.0, 0.15, 0.0),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	scene.Root().AddChild(burst)

	d := &demo{burst: burst}

	// Click anywhere to trigger a burst.
	scene.OnBackgroundClick(func(ctx willow.ClickContext) {
		burst.SetPosition(ctx.GlobalX, ctx.GlobalY)
		burst.Emitter.Reset()
		burst.Emitter.Start()
		d.burstTimer = 0.08 // emit for ~5 frames then stop
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

func (d *demo) update() error {
	if d.burstTimer > 0 {
		d.burstTimer -= 1.0 / float64(ebiten.TPS())
		if d.burstTimer <= 0 {
			d.burst.Emitter.Stop()
		}
	}
	return nil
}

// addBase places a thin colored bar at (cx, y) to mark an emitter's origin.
func addBase(scene *willow.Scene, cx, y float64, c willow.Color) {
	bar := willow.NewRect("base", 60, 5, c)
	bar.SetPosition(cx-30, y)
	scene.Root().AddChild(bar)
}
