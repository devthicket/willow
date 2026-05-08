// FullscreenCamera proves that RunConfig.Width/Height define the *logical*
// screen size in fullscreen mode: the camera's viewport, the sprite at
// logical-center coords, and the per-edge markers all line up with the
// visible image on the monitor (with letterbox bars filling any aspect
// mismatch).
//
// Pre-fix, in fullscreen mode, Layout overwrote the logical size with the
// monitor's native resolution. A camera created with viewport
// {0, 0, 640, 480} ended up clipped to a 640×480 box in the top-left
// corner; a sprite at (320, 240) showed up in that corner instead of the
// center. With the fix, this demo shows the sprite at the visual center
// of the monitor regardless of the actual fullscreen resolution.
//
// Controls:
//
//	Esc : quit
//
// Run with:
//
//	go run ./examples/demos/fullscreen-camera
//	go run ./examples/demos/fullscreen-camera -windowed   # disable fullscreen for sanity check
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	logicalW = 640
	logicalH = 480
)

func main() {
	windowed := flag.Bool("windowed", false, "run in a window (default: fullscreen)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.10, 0.12, 0.18)

	// Camera viewport == full logical screen.
	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: logicalW, Height: logicalH})
	// Center camera on logical center of the world. Since the camera's
	// (X, Y) is the world point that maps to viewport center, pinning it
	// to (logicalW/2, logicalH/2) means world coords match logical screen
	// coords 1:1.
	cam.X = logicalW / 2
	cam.Y = logicalH / 2
	cam.Invalidate()

	// Centerpiece: a 60×60 sprite anchored by its top-left at exactly the
	// logical-screen center minus half its size, so its center lands on
	// (logicalW/2, logicalH/2).
	const sz = 60
	center := willow.NewRect("center", sz, sz, willow.RGB(0.95, 0.55, 0.20))
	center.SetPosition(logicalW/2-sz/2, logicalH/2-sz/2)
	scene.Root.AddChild(center)

	// Edge markers: small squares at each corner and the midpoint of each
	// edge, all in logical-screen world coords. If fullscreen letterboxing
	// is correct, these sit exactly at the corners and edge midpoints of
	// the visible (non-bar) region of the monitor.
	addMarker := func(x, y float64, c willow.Color) {
		const m = 16
		mk := willow.NewRect("marker", m, m, c)
		mk.SetPosition(x-m/2, y-m/2)
		scene.Root.AddChild(mk)
	}
	corner := willow.RGB(0.30, 0.85, 0.45)
	addMarker(0, 0, corner)
	addMarker(logicalW, 0, corner)
	addMarker(0, logicalH, corner)
	addMarker(logicalW, logicalH, corner)
	mid := willow.RGB(0.45, 0.65, 1.0)
	addMarker(logicalW/2, 0, mid)
	addMarker(logicalW/2, logicalH, mid)
	addMarker(0, logicalH/2, mid)
	addMarker(logicalW, logicalH/2, mid)

	// OnResize fires once on first Layout call under the new contract.
	// Print what it received so it's easy to confirm at runtime that the
	// callback got cfg.Width × cfg.Height (not the monitor's native size).
	scene.SetOnResize(func(w, h int) {
		fmt.Printf("Scene.OnResize received logical size: %dx%d\n", w, h)
	})

	scene.SetUpdateFunc(func() error {
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			return ebiten.Termination
		}
		return nil
	})

	// PreDraw: paint a 1px outline at the logical screen edges to visually
	// confirm where the logical screen meets the letterbox bars. The line
	// must hug the inner edge of the visible region, not the monitor.
	scene.SetPostDrawFunc(func(screen *ebiten.Image) {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		top := ebiten.NewImage(w, 1)
		top.Fill(color.RGBA{255, 255, 255, 220})
		var op ebiten.DrawImageOptions
		screen.DrawImage(top, &op)
		op.GeoM.Reset()
		op.GeoM.Translate(0, float64(h-1))
		screen.DrawImage(top, &op)

		side := ebiten.NewImage(1, h)
		side.Fill(color.RGBA{255, 255, 255, 220})
		op.GeoM.Reset()
		screen.DrawImage(side, &op)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(w-1), 0)
		screen.DrawImage(side, &op)

		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("logical screen: %dx%d", w, h), 8, 8)
		ebitenutil.DebugPrintAt(screen, "centerpiece sprite at world (320, 240)", 8, 24)
		ebitenutil.DebugPrintAt(screen, "Esc to quit", 8, 40)
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:      "Willow  -  Fullscreen Camera Example",
		Width:      logicalW,
		Height:     logicalH,
		Fullscreen: !*windowed,
		ShowFPS:    true,
	}); err != nil {
		log.Fatal(err)
	}
}
