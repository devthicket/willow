// Filter Gallery showcases every built-in Willow filter applied to a whelp
// sprite. Users toggle filters on/off via clickable buttons, and multiple
// filters can stack on the same sprite simultaneously.
package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"strings"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	windowTitle = "Willow  -  Filter Gallery"
	showFPS     = true
	screenW     = 800
	screenH     = 600
)

// filterEntry holds the state for one toggle button + filter pair.
type filterEntry struct {
	label    string
	filter   willow.Filter
	active   bool
	animated bool
	update   func(t float64) // per-frame update for animated filters

	// UI nodes
	bg       *willow.Node
	checkImg [2]*ebiten.Image // [0]=unchecked, [1]=checked
	checkSp  *willow.Node
}

func main() {
	f, err := os.Open("examples/_assets/whelp.png")
	if err != nil {
		log.Fatalf("open whelp.png: %v", err)
	}
	defer f.Close()
	whelpImg, _, err := ebitenutil.NewImageFromReader(f)
	if err != nil {
		log.Fatalf("decode whelp.png: %v", err)
	}

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.08, 0.06, 0.12)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Whelp sprite: centered in upper area, scaled 2x.
	whelp := willow.NewSprite("whelp", willow.TextureRegion{})
	whelp.SetCustomImage(whelpImg)
	whelp.SetScale(2, 2)
	whelp.SetPivot(64, 64) // half of 128px
	whelp.SetPosition(screenW/2, 180)
	whelp.Invalidate()
	scene.Root.AddChild(whelp)

	// Status label showing active filters.
	statusLabel := makeLabel("No filters active")
	statusLabel.SetPosition(screenW/2-float64(len("No filters active")*6)/2, 320)
	scene.Root.AddChild(statusLabel)

	// Create filter instances.
	blur := willow.NewBlurFilter(6)

	sepia := willow.NewColorMatrixFilter()
	sepia.Matrix = [20]float64{
		0.393, 0.769, 0.189, 0, 0,
		0.349, 0.686, 0.168, 0, 0,
		0.272, 0.534, 0.131, 0, 0,
		0, 0, 0, 1, 0,
	}

	outline := willow.NewOutlineFilter(3, willow.RGB(1, 0.85, 0.2))
	inline := willow.NewPixelPerfectInlineFilter(willow.RGB(0, 1, 1))
	ppOutline := willow.NewPixelPerfectOutlineFilter(willow.RGB(1, 0.3, 0.3))

	brightness := willow.NewColorMatrixFilter()
	grayscale := willow.NewColorMatrixFilter()
	grayscale.SetSaturation(0)

	contrast := willow.NewColorMatrixFilter()
	contrast.SetContrast(1.8)

	hueShift := willow.NewColorMatrixFilter()

	palette := willow.NewPaletteFilter()
	// Set a retro palette: warm reds/oranges/yellows/greens/cyans/blues.
	var retroPalette [256]willow.Color
	for i := 0; i < 256; i++ {
		t := float64(i) / 255.0
		retroPalette[i] = willow.RGB(
			0.5+0.5*math.Cos(2*math.Pi*t),
			0.5+0.5*math.Cos(2*math.Pi*t+2*math.Pi/3),
			0.5+0.5*math.Cos(2*math.Pi*t+4*math.Pi/3),
		)
	}
	palette.SetPalette(retroPalette)

	// Define all 10 filter entries.
	entries := []*filterEntry{
		{label: "Blur", filter: blur},
		{label: "Sepia", filter: sepia},
		{label: "Outline", filter: outline},
		{label: "Inline", filter: inline},
		{label: "PP Outline", filter: ppOutline},
		{label: "Brightness", filter: brightness, animated: true, update: func(t float64) {
			brightness.SetBrightness(0.3 * math.Sin(t*3))
		}},
		{label: "Grayscale", filter: grayscale},
		{label: "Contrast", filter: contrast},
		{label: "Hue Shift", filter: hueShift, animated: true, update: func(t float64) {
			angle := t * 0.8
			cos, sin := math.Cos(angle), math.Sin(angle)
			hueShift.Matrix = [20]float64{
				0.213 + cos*0.787 - sin*0.213, 0.715 - cos*0.715 - sin*0.715, 0.072 - cos*0.072 + sin*0.928, 0, 0,
				0.213 - cos*0.213 + sin*0.143, 0.715 + cos*0.285 + sin*0.140, 0.072 - cos*0.072 - sin*0.283, 0, 0,
				0.213 - cos*0.213 - sin*0.787, 0.715 - cos*0.715 + sin*0.715, 0.072 + cos*0.928 + sin*0.072, 0, 0,
				0, 0, 0, 1, 0,
			}
		}},
		{label: "Palette", filter: palette, animated: true, update: func(t float64) {
			palette.CycleOffset = t * 20
		}},
	}

	// Pre-render checkbox images.
	uncheckedImg := ebiten.NewImage(14, 14)
	uncheckedImg.Fill(color.RGBA{R: 77, G: 77, B: 77, A: 255})
	innerUnchecked := ebiten.NewImage(10, 10)
	innerUnchecked.Fill(color.RGBA{R: 38, G: 38, B: 38, A: 255})
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(2, 2)
	uncheckedImg.DrawImage(innerUnchecked, &op)

	checkedImg := ebiten.NewImage(14, 14)
	checkedImg.Fill(color.RGBA{R: 77, G: 204, B: 77, A: 255})
	innerChecked := ebiten.NewImage(10, 10)
	innerChecked.Fill(color.RGBA{R: 51, G: 153, B: 51, A: 255})
	op.GeoM.Reset()
	op.GeoM.Translate(2, 2)
	checkedImg.DrawImage(innerChecked, &op)

	// Button grid: 2 rows x 5 cols, centered at bottom.
	btnW := 130.0
	btnH := 30.0
	btnGap := 10.0
	cols := 5
	gridW := float64(cols)*btnW + float64(cols-1)*btnGap
	startX := (screenW - gridW) / 2
	startY := 420.0

	// rebuildFilters collects all active filters and updates the whelp and status.
	rebuildFilters := func() {
		var active []any
		var names []string
		for _, e := range entries {
			if e.active {
				active = append(active, e.filter)
				names = append(names, e.label)
			}
		}
		whelp.Filters = active
		whelp.Invalidate()

		// Update status label.
		statusLabel.RemoveFromParent()
		text := "No filters active"
		if len(names) > 0 {
			text = fmt.Sprintf("Active: %s", strings.Join(names, ", "))
		}
		statusLabel = makeLabel(text)
		statusLabel.SetPosition(screenW/2-float64(len(text)*6)/2, 320)
		scene.Root.AddChild(statusLabel)
	}

	for i, entry := range entries {
		col := i % cols
		row := i / cols

		bx := startX + float64(col)*(btnW+btnGap)
		by := startY + float64(row)*(btnH+btnGap)

		// Background button.
		bg := willow.NewRect("btn-bg-"+entry.label, btnW, btnH, willow.RGB(0.2, 0.2, 0.25))
		bg.SetPosition(bx, by)
		scene.Root.AddChild(bg)
		entry.bg = bg

		// Checkbox indicator.
		checkSp := willow.NewSprite("check-"+entry.label, willow.TextureRegion{})
		checkSp.SetCustomImage(uncheckedImg)
		checkSp.SetPosition(bx+6, by+8)
		checkSp.Invalidate()
		scene.Root.AddChild(checkSp)
		entry.checkSp = checkSp
		entry.checkImg[0] = uncheckedImg
		entry.checkImg[1] = checkedImg

		// Label text.
		label := makeLabel(entry.label)
		label.SetPosition(bx+24, by+7)
		scene.Root.AddChild(label)

		// Click handler.
		e := entry // capture loop variable
		bg.OnClick(func(ctx willow.ClickContext) {
			e.active = !e.active
			if e.active {
				e.bg.SetColor(willow.RGB(0.15, 0.4, 0.2))
				e.checkSp.SetCustomImage(e.checkImg[1])
			} else {
				e.bg.SetColor(willow.RGB(0.2, 0.2, 0.25))
				e.checkSp.SetCustomImage(e.checkImg[0])
			}
			e.bg.Invalidate()
			e.checkSp.Invalidate()
			rebuildFilters()
		})
	}

	// Animate filters that support it.
	elapsed := 0.0
	scene.SetUpdateFunc(func() error {
		elapsed += 1.0 / float64(ebiten.TPS())

		needInvalidate := false
		for _, e := range entries {
			if e.active && e.animated && e.update != nil {
				e.update(elapsed)
				needInvalidate = true
			}
		}
		if needInvalidate {
			whelp.Invalidate()
		}

		return nil
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}

// makeLabel pre-renders a text string into a sprite using Ebitengine's debug font.
func makeLabel(s string) *willow.Node {
	w := len(s)*6 + 2
	h := 16
	img := ebiten.NewImage(w, h)
	ebitenutil.DebugPrint(img, s)

	n := willow.NewSprite("label-"+s, willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}
