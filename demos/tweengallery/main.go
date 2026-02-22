// Tween Gallery showcases every easing function available in Willow's tween
// system. Five whelp sprites each demonstrate a different tween property
// (position, scale, rotation, alpha, color). Users click easing buttons to
// replay all animations with the selected curve.
package main

import (
	"bytes"
	_ "embed"
	"image/png"
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
	"github.com/tanema/gween/ease"
)

//go:embed whelp.png
var whelpPNG []byte

const (
	windowTitle = "Willow — Tween Gallery"
	showFPS     = true
	screenW     = 800
	screenH     = 600
)

// tweenType labels the five property categories.
type tweenType int

const (
	tweenPosition tweenType = iota
	tweenScale
	tweenRotation
	tweenAlpha
	tweenColor
)

var tweenLabels = [5]string{"Position", "Scale", "Rotation", "Alpha", "Color"}
var tweenColors = [5]willow.Color{
	{R: 0.4, G: 0.8, B: 1.0, A: 1}, // cyan
	{R: 1.0, G: 0.6, B: 0.2, A: 1}, // orange
	{R: 0.6, G: 1.0, B: 0.4, A: 1}, // green
	{R: 0.9, G: 0.9, B: 0.3, A: 1}, // yellow
	{R: 1.0, G: 0.4, B: 0.7, A: 1}, // pink
}

// easingEntry pairs a display name with its gween function.
type easingEntry struct {
	label string
	fn    ease.TweenFunc

	// UI nodes
	bg *willow.Node
}

func main() {
	// Decode embedded whelp image.
	whelpRaw, err := png.Decode(bytes.NewReader(whelpPNG))
	if err != nil {
		log.Fatalf("decode whelp: %v", err)
	}
	whelpImg := ebiten.NewImageFromImage(whelpRaw)

	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.08, G: 0.06, B: 0.12, A: 1}

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	// Five whelp sprites — one per tween property.
	sprites := make([]*willow.Node, 5)
	spacing := 140.0
	startX := (screenW - 4*spacing) / 2
	spriteY := 160.0

	for i := 0; i < 5; i++ {
		sp := willow.NewSprite("whelp-"+tweenLabels[i], willow.TextureRegion{})
		sp.SetCustomImage(whelpImg)
		sp.ScaleX = 1.5
		sp.ScaleY = 1.5
		sp.PivotX = 64
		sp.PivotY = 64
		sp.X = startX + float64(i)*spacing
		sp.Y = spriteY
		sp.Invalidate()
		scene.Root().AddChild(sp)
		sprites[i] = sp
	}

	// Property labels below each sprite.
	for i := 0; i < 5; i++ {
		label := makeLabel(tweenLabels[i])
		lw := float64(len(tweenLabels[i]) * 6)
		label.X = startX + float64(i)*spacing - lw/2
		label.Y = spriteY + 80
		scene.Root().AddChild(label)

		// Colored indicator dot.
		dot := willow.NewSprite("dot", willow.TextureRegion{})
		dot.ScaleX = lw + 8
		dot.ScaleY = 3
		dot.Color = tweenColors[i]
		dot.X = startX + float64(i)*spacing - lw/2 - 4
		dot.Y = spriteY + 96
		scene.Root().AddChild(dot)
	}

	// Status label showing current easing.
	statusLabel := makeLabel("Select an easing function")
	statusLabel.X = screenW/2 - float64(len("Select an easing function")*6)/2
	statusLabel.Y = 290
	scene.Root().AddChild(statusLabel)

	// Define easing functions.
	easings := []*easingEntry{
		{label: "Linear", fn: ease.Linear},
		{label: "InQuad", fn: ease.InQuad},
		{label: "OutQuad", fn: ease.OutQuad},
		{label: "InOutQuad", fn: ease.InOutQuad},
		{label: "InCubic", fn: ease.InCubic},
		{label: "OutCubic", fn: ease.OutCubic},
		{label: "InOutCubic", fn: ease.InOutCubic},
		{label: "OutBack", fn: ease.OutBack},
		{label: "OutBounce", fn: ease.OutBounce},
		{label: "OutElastic", fn: ease.OutElastic},
		{label: "InOutSine", fn: ease.InOutSine},
		{label: "InExpo", fn: ease.InExpo},
	}

	// Tween state.
	var tweens []*willow.TweenGroup
	selectedIdx := -1

	// Home positions for reset.
	homeX := make([]float64, 5)
	for i := 0; i < 5; i++ {
		homeX[i] = startX + float64(i)*spacing
	}

	resetSprites := func() {
		for i, sp := range sprites {
			sp.X = homeX[i]
			sp.Y = spriteY
			sp.ScaleX = 1.5
			sp.ScaleY = 1.5
			sp.Rotation = 0
			sp.Alpha = 1
			sp.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
			sp.Invalidate()
		}
	}

	startTweens := func(fn ease.TweenFunc) {
		resetSprites()
		tweens = []*willow.TweenGroup{
			// Position: bounce vertically.
			willow.TweenPosition(sprites[0], homeX[0], spriteY-70+(rand.Float64()*40), 1.8, fn),
			// Scale: grow then shrink back.
			willow.TweenScale(sprites[1], 2.8, 2.8, 1.8, fn),
			// Rotation: full spin.
			willow.TweenRotation(sprites[2], math.Pi*2, 1.8, fn),
			// Alpha: fade to near-invisible.
			willow.TweenAlpha(sprites[3], 0.1, 1.8, fn),
			// Color: tint to random bright color.
			willow.TweenColor(sprites[4], randomBrightColor(), 1.8, fn),
		}
	}

	updateStatusLabel := func(text string) {
		statusLabel.RemoveFromParent()
		statusLabel = makeLabel(text)
		statusLabel.X = screenW/2 - float64(len(text)*6)/2
		statusLabel.Y = 290
		scene.Root().AddChild(statusLabel)
	}

	// Button grid: 3 rows x 4 cols.
	btnW := 120.0
	btnH := 30.0
	btnGap := 8.0
	cols := 4
	gridW := float64(cols)*btnW + float64(cols-1)*btnGap
	gridStartX := (screenW - gridW) / 2
	gridStartY := 340.0

	activeColor := willow.Color{R: 0.15, G: 0.4, B: 0.2, A: 1}
	inactiveColor := willow.Color{R: 0.2, G: 0.2, B: 0.25, A: 1}

	for i, entry := range easings {
		col := i % cols
		row := i / cols

		bx := gridStartX + float64(col)*(btnW+btnGap)
		by := gridStartY + float64(row)*(btnH+btnGap)

		// Button background.
		bg := willow.NewSprite("btn-"+entry.label, willow.TextureRegion{})
		bg.ScaleX = btnW
		bg.ScaleY = btnH
		bg.Color = inactiveColor
		bg.X = bx
		bg.Y = by
		bg.Invalidate()
		scene.Root().AddChild(bg)
		entry.bg = bg

		// Label.
		label := makeLabel(entry.label)
		lw := float64(len(entry.label) * 6)
		label.X = bx + (btnW-lw)/2
		label.Y = by + 7
		scene.Root().AddChild(label)

		// Click handler.
		idx := i
		e := entry
		bg.Interactable = true
		bg.OnClick = func(ctx willow.ClickContext) {
			// Deselect previous.
			if selectedIdx >= 0 && selectedIdx < len(easings) {
				easings[selectedIdx].bg.Color = inactiveColor
				easings[selectedIdx].bg.Invalidate()
			}

			selectedIdx = idx
			e.bg.Color = activeColor
			e.bg.Invalidate()

			updateStatusLabel("Easing: " + e.label)
			startTweens(e.fn)
		}
	}

	// "Replay" instruction hint.
	hint := makeLabel("Click a button to see each easing curve applied to all 5 tween properties")
	hint.X = screenW/2 - float64(len("Click a button to see each easing curve applied to all 5 tween properties")*6)/2
	hint.Y = screenH - 40
	scene.Root().AddChild(hint)

	// Update loop: advance tweens, auto-restart when all done.
	scene.SetUpdateFunc(func() error {
		dt := float32(1.0 / float64(ebiten.TPS()))

		if len(tweens) == 0 {
			return nil
		}

		allDone := true
		for _, tw := range tweens {
			tw.Update(dt)
			if !tw.Done {
				allDone = false
			}
		}

		// When finished, reset and replay with same easing.
		if allDone && selectedIdx >= 0 {
			startTweens(easings[selectedIdx].fn)
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

func randomBrightColor() willow.Color {
	return willow.Color{
		R: 0.3 + rand.Float64()*0.7,
		G: 0.3 + rand.Float64()*0.7,
		B: 0.3 + rand.Float64()*0.7,
		A: 1,
	}
}
