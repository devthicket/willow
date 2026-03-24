// Tween Recipe demonstrates a coordinated tween combo where twelve circles
// bloom outward from center, rotate and shift color, then converge back
// in a continuous loop. Each ring staggers slightly for a spiral effect.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween/ease"
)

const (
	windowTitle = "Willow  -  Tween Recipe"
	screenW     = 640
	screenH     = 480
	numDots     = 12
	ringRadius  = 140.0
)

var palette = []willow.Color{
	willow.RGB(1.0, 0.3, 0.5),  // rose
	willow.RGB(1.0, 0.6, 0.1),  // amber
	willow.RGB(0.3, 0.9, 0.4),  // mint
	willow.RGB(0.2, 0.6, 1.0),  // sky
	willow.RGB(0.8, 0.3, 1.0),  // violet
	willow.RGB(0.1, 0.95, 0.9), // cyan
}

type demo struct {
	scene  *willow.Scene
	dots   []*willow.Node
	center *willow.Node
	phase  int // 0 = expand, 1 = converge
	tweens []*willow.TweenGroup
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.06, 0.06, 0.1)

	cx, cy := float64(screenW)/2, float64(screenH)/2

	// Central pulsing circle.
	center := willow.NewCircle("center", 18, willow.RGB(1, 1, 1))
	center.SetPosition(cx, cy)
	center.SetPivot(0.5, 0.5)
	center.SetAlpha(0.9)
	scene.Root.AddChild(center)

	// Ring of dots, all starting at center.
	dots := make([]*willow.Node, numDots)
	for i := range dots {
		ci := palette[i%len(palette)]
		dot := willow.NewCircle(fmt.Sprintf("dot-%d", i), 10, ci)
		dot.SetPosition(cx, cy)
		dot.SetPivot(0.5, 0.5)
		dot.SetScale(0.3, 0.3)
		dot.SetAlpha(0)
		scene.Root.AddChild(dot)
		dots[i] = dot
	}

	d := &demo{scene: scene, dots: dots, center: center}
	d.expand()

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
			d.update()
			if runner.Done() {
				fmt.Println("Autotest complete.")
				return ebiten.Termination
			}
			return nil
		})
	} else {
		scene.SetUpdateFunc(d.update)
	}

	if err := willow.Run(scene, willow.RunConfig{
		Title:  windowTitle,
		Width:  screenW,
		Height: screenH,
	}); err != nil {
		log.Fatal(err)
	}
}

func (d *demo) expand() {
	d.phase = 0
	d.tweens = d.tweens[:0]
	cx, cy := float64(screenW)/2, float64(screenH)/2

	// Center: pulse up.
	d.tweens = append(d.tweens,
		willow.TweenScale(d.center, 1.6, 1.6, willow.TweenConfig{Duration: 0.6, Ease: ease.OutBack}),
		willow.TweenColor(d.center, willow.RGB(0.4, 0.8, 1.0), willow.TweenConfig{Duration: 1.2, Ease: ease.InOutSine}),
	)

	// Each dot flies to its ring position with staggered timing via duration offset.
	for i, dot := range d.dots {
		frac := float64(i) / float64(numDots)
		angle := frac * math.Pi * 2
		tx := cx + math.Cos(angle)*ringRadius
		ty := cy + math.Sin(angle)*ringRadius

		// Stagger: earlier dots move slightly faster.
		dur := float32(1.0 + frac*0.4)
		nextColor := palette[(i+2)%len(palette)]

		d.tweens = append(d.tweens,
			willow.TweenPosition(dot, tx, ty, willow.TweenConfig{Duration: dur, Ease: ease.OutCubic}),
			willow.TweenScale(dot, 1.2, 1.2, willow.TweenConfig{Duration: dur, Ease: ease.OutElastic}),
			willow.TweenAlpha(dot, 1.0, willow.TweenConfig{Duration: dur * 0.5, Ease: ease.OutQuad}),
			willow.TweenRotation(dot, math.Pi*2*frac+math.Pi, willow.TweenConfig{Duration: dur, Ease: ease.InOutCubic}),
			willow.TweenColor(dot, nextColor, willow.TweenConfig{Duration: dur, Ease: ease.InOutSine}),
		)
	}
}

func (d *demo) converge() {
	d.phase = 1
	d.tweens = d.tweens[:0]
	cx, cy := float64(screenW)/2, float64(screenH)/2

	// Center: shrink back.
	d.tweens = append(d.tweens,
		willow.TweenScale(d.center, 0.6, 0.6, willow.TweenConfig{Duration: 0.8, Ease: ease.InBack}),
		willow.TweenColor(d.center, willow.RGB(1.0, 0.5, 0.3), willow.TweenConfig{Duration: 1.0, Ease: ease.InOutSine}),
	)

	// Dots spiral back to center.
	for i, dot := range d.dots {
		frac := float64(i) / float64(numDots)
		dur := float32(0.8 + frac*0.5)

		d.tweens = append(d.tweens,
			willow.TweenPosition(dot, cx, cy, willow.TweenConfig{Duration: dur, Ease: ease.InCubic}),
			willow.TweenScale(dot, 0.3, 0.3, willow.TweenConfig{Duration: dur, Ease: ease.InBack}),
			willow.TweenAlpha(dot, 0.0, willow.TweenConfig{Duration: dur, Ease: ease.InQuad}),
			willow.TweenRotation(dot, 0, willow.TweenConfig{Duration: dur, Ease: ease.InOutCubic}),
		)
	}
}

func (d *demo) update() error {
	allDone := true
	for _, tw := range d.tweens {
		if !tw.Done {
			allDone = false
			break
		}
	}

	if allDone && len(d.tweens) > 0 {
		if d.phase == 0 {
			d.converge()
		} else {
			d.expand()
		}
	}

	return nil
}
