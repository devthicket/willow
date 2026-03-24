// Lighting Showcase demonstrates Willow's circle light system with multiple
// colored lights orbiting and overlapping to create blended illumination.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 320
	screenH = 240
)

type orbitLight struct {
	light        *willow.Light
	cx, cy       float64 // orbit center
	orbitR       float64 // orbit radius
	speed        float64 // radians per second
	phase        float64 // starting angle
	pulseSpeed   float64
	pulseAmount  float64
	baseRadius   float64
	baseIntesity float64
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.02, 0.02, 0.04)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Textured floor — dark stone tiles to catch the light.
	tileColors := []willow.Color{
		willow.RGB(0.45, 0.42, 0.38),
		willow.RGB(0.40, 0.37, 0.33),
		willow.RGB(0.50, 0.46, 0.40),
		willow.RGB(0.42, 0.39, 0.35),
	}
	for row := range 12 {
		for col := range 16 {
			tile := willow.NewSprite("t", willow.TextureRegion{})
			tile.SetScale(20, 20)
			tile.SetPosition(float64(col)*20, float64(row)*20)
			tile.SetColor(tileColors[(row*17+col)%len(tileColors)])
			scene.Root.AddChild(tile)
		}
	}

	// Scattered objects so the light reveals something interesting.
	objColor := willow.RGB(0.55, 0.48, 0.40)
	objPositions := [][2]float64{
		{40, 30}, {260, 50}, {80, 180}, {200, 190},
		{140, 90}, {50, 120}, {240, 140}, {160, 200},
	}
	for _, p := range objPositions {
		obj := willow.NewSprite("obj", willow.TextureRegion{})
		obj.SetPosition(p[0], p[1])
		obj.SetScale(18, 18)
		obj.SetColor(objColor)
		scene.Root.AddChild(obj)
	}

	// Light layer — dark enough for circles to stand out, bright enough to see overlap.
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.88)

	// Six orbiting lights in complementary pairs, different orbit radii and speeds
	// to create constantly shifting overlap patterns.
	midX, midY := float64(screenW)/2, float64(screenH)/2

	orbits := []orbitLight{
		// Warm trio
		{
			light: &willow.Light{Radius: 90, Intensity: 0.95, Color: willow.RGB(1.0, 0.4, 0.1), Enabled: true},
			cx:    midX, cy: midY, orbitR: 80, speed: 1.2, phase: 0,
			pulseSpeed: 3.0, pulseAmount: 15, baseRadius: 90, baseIntesity: 0.95,
		},
		{
			light: &willow.Light{Radius: 75, Intensity: 0.85, Color: willow.RGB(1.0, 0.75, 0.2), Enabled: true},
			cx:    midX, cy: midY, orbitR: 55, speed: -0.9, phase: math.Pi * 2 / 3,
			pulseSpeed: 2.5, pulseAmount: 10, baseRadius: 75, baseIntesity: 0.85,
		},
		{
			light: &willow.Light{Radius: 70, Intensity: 0.8, Color: willow.RGB(0.9, 0.2, 0.3), Enabled: true},
			cx:    midX, cy: midY, orbitR: 95, speed: 0.7, phase: math.Pi,
			pulseSpeed: 4.0, pulseAmount: 12, baseRadius: 70, baseIntesity: 0.8,
		},
		// Cool trio
		{
			light: &willow.Light{Radius: 85, Intensity: 0.9, Color: willow.RGB(0.2, 0.5, 1.0), Enabled: true},
			cx:    midX, cy: midY, orbitR: 70, speed: -1.0, phase: math.Pi / 3,
			pulseSpeed: 2.8, pulseAmount: 14, baseRadius: 85, baseIntesity: 0.9,
		},
		{
			light: &willow.Light{Radius: 65, Intensity: 0.75, Color: willow.RGB(0.1, 0.8, 0.6), Enabled: true},
			cx:    midX, cy: midY, orbitR: 60, speed: 1.5, phase: math.Pi * 4 / 3,
			pulseSpeed: 3.5, pulseAmount: 8, baseRadius: 65, baseIntesity: 0.75,
		},
		{
			light: &willow.Light{Radius: 80, Intensity: 0.85, Color: willow.RGB(0.6, 0.3, 1.0), Enabled: true},
			cx:    midX, cy: midY, orbitR: 85, speed: -0.6, phase: math.Pi * 5 / 3,
			pulseSpeed: 2.0, pulseAmount: 12, baseRadius: 80, baseIntesity: 0.85,
		},
	}

	for i := range orbits {
		lightLayer.AddLight(orbits[i].light)
	}

	// Dim center anchor light so the middle isn't pitch black.
	center := &willow.Light{
		X: midX, Y: midY,
		Radius: 40, Intensity: 0.3,
		Color:   willow.RGB(0.9, 0.85, 0.7),
		Enabled: true,
	}
	lightLayer.AddLight(center)

	scene.Root.AddChild(lightLayer.Node())

	elapsed := 0.0

	updateFn := func() error {
		elapsed += 1.0 / float64(ebiten.TPS())

		for i := range orbits {
			o := &orbits[i]
			angle := o.phase + elapsed*o.speed
			o.light.X = o.cx + o.orbitR*math.Cos(angle)
			o.light.Y = o.cy + o.orbitR*math.Sin(angle)
			o.light.Radius = o.baseRadius + o.pulseAmount*math.Sin(elapsed*o.pulseSpeed+o.phase)
			o.light.Intensity = o.baseIntesity + 0.1*math.Sin(elapsed*o.pulseSpeed*0.7+o.phase)
		}

		// Gentle center pulse.
		center.Intensity = 0.25 + 0.1*math.Sin(elapsed*1.5)

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
		Title:   "Willow  -  Lighting Showcase",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: false,
	}); err != nil {
		log.Fatal(err)
	}
}
