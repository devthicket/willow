// Text-pixelfont demonstrates pixel-perfect bitmap font rendering using
// willow.NewFontFamilyFromPixelFont. Shows basic text, scaling (1x, 2x, 3x),
// word wrapping, alignment, and color tinting.
package main

import (
	"log"
	"os"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	windowTitle = "Willow  -  Pixel Font Example"
	showFPS     = true
	screenW     = 800
	screenH     = 620
)

// The character map for pixelfont1.png (16x16 cells, single row).
// Starts at '!' (0x21) through '~' (0x7E).
const pixelFontChars = "!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

func main() {
	fontImg, _, err := ebitenutil.NewImageFromFile("examples/_assets/pixelfont1.png")
	if err != nil {
		log.Fatalf("load pixelfont1.png: %v", err)
	}

	// One font instance, scale controlled per-label via FontSize.
	// FontSize 16 = 1x (native), 32 = 2x, 48 = 3x (cellH is 16).
	font := willow.NewFontFamilyFromPixelFont(fontImg, 16, 16, pixelFontChars)
	font.TrimCell(0, 4, 0, 4) // trim 4px left/right for tighter spacing

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.08, 0.08, 0.12)

	y := 16.0

	// --- Section: Scale comparison ---
	sectionLabel := willow.NewText("section-scale", "-- Scale --", font)
	sectionLabel.TextBlock.Color = willow.RGB(0.5, 0.5, 0.6)
	sectionLabel.TextBlock.Invalidate()
	sectionLabel.SetPosition(20, y)
	scene.Root.AddChild(sectionLabel)
	y += 22

	label1x := willow.NewText("1x", "Hello, Pixel World! (1x)", font)
	label1x.SetPosition(20, y)
	scene.Root.AddChild(label1x)
	y += 22

	label2x := willow.NewText("2x", "Hello, Pixel World! (2x)", font)
	label2x.TextBlock.FontSize = 32
	label2x.TextBlock.Invalidate()
	label2x.SetPosition(20, y)
	scene.Root.AddChild(label2x)
	y += 40

	label3x := willow.NewText("3x", "Hello! (3x)", font)
	label3x.TextBlock.FontSize = 48
	label3x.TextBlock.Invalidate()
	label3x.SetPosition(20, y)
	scene.Root.AddChild(label3x)
	y += 56

	// --- Section: Color tinting ---
	sectionColor := willow.NewText("section-color", "-- Color Tinting --", font)
	sectionColor.TextBlock.Color = willow.RGB(0.5, 0.5, 0.6)
	sectionColor.TextBlock.Invalidate()
	sectionColor.SetPosition(20, y)
	scene.Root.AddChild(sectionColor)
	y += 22

	labelRed := willow.NewText("red", "Danger!", font)
	labelRed.TextBlock.FontSize = 32
	labelRed.TextBlock.Color = willow.RGB(1, 0.3, 0.3)
	labelRed.TextBlock.Invalidate()
	labelRed.SetPosition(20, y)
	scene.Root.AddChild(labelRed)

	labelGreen := willow.NewText("green", "Success!", font)
	labelGreen.TextBlock.FontSize = 32
	labelGreen.TextBlock.Color = willow.RGB(0.3, 1, 0.3)
	labelGreen.TextBlock.Invalidate()
	labelGreen.SetPosition(280, y)
	scene.Root.AddChild(labelGreen)

	labelBlue := willow.NewText("blue", "Info! (+ Node tint)", font)
	labelBlue.TextBlock.FontSize = 32
	labelBlue.TextBlock.Color = willow.RGB(0.4, 0.6, 1)
	labelBlue.TextBlock.Invalidate()
	labelBlue.SetColor(willow.RGB(0.8, 0.8, 1)) // Node.Color as additional tint
	labelBlue.SetPosition(540, y)
	scene.Root.AddChild(labelBlue)
	y += 44

	// --- Section: Word wrapping ---
	sectionWrap := willow.NewText("section-wrap", "-- Word Wrapping --", font)
	sectionWrap.TextBlock.Color = willow.RGB(0.5, 0.5, 0.6)
	sectionWrap.TextBlock.Invalidate()
	sectionWrap.SetPosition(20, y)
	scene.Root.AddChild(sectionWrap)
	y += 22

	wrapLabel := willow.NewText("wrap", "The quick brown fox jumps over the lazy dog. Pack my box with five dozen liquor jugs.", font)
	wrapLabel.TextBlock.WrapWidth = 500
	wrapLabel.TextBlock.Invalidate()
	wrapLabel.SetPosition(20, y)
	scene.Root.AddChild(wrapLabel)
	y += 56

	// --- Section: Alignment ---
	sectionAlign := willow.NewText("section-align", "-- Alignment --", font)
	sectionAlign.TextBlock.Color = willow.RGB(0.5, 0.5, 0.6)
	sectionAlign.TextBlock.Invalidate()
	sectionAlign.SetPosition(20, y)
	scene.Root.AddChild(sectionAlign)
	y += 24

	guideLine := func(gy float64) {
		g := willow.NewRect("guide", 500, 1, willow.RGB(0.2, 0.2, 0.25))
		g.SetPosition(20, gy)
		scene.Root.AddChild(g)
	}
	guideLine(y - 2)

	alignLeft := willow.NewText("left", "Left aligned text", font)
	alignLeft.TextBlock.WrapWidth = 500
	alignLeft.TextBlock.Align = willow.TextAlignLeft
	alignLeft.TextBlock.Invalidate()
	alignLeft.SetPosition(20, y)
	scene.Root.AddChild(alignLeft)
	y += 20

	alignCenter := willow.NewText("center", "Center aligned text", font)
	alignCenter.TextBlock.WrapWidth = 500
	alignCenter.TextBlock.Align = willow.TextAlignCenter
	alignCenter.TextBlock.Invalidate()
	alignCenter.SetPosition(20, y)
	scene.Root.AddChild(alignCenter)
	y += 20

	alignRight := willow.NewText("right", "Right aligned text", font)
	alignRight.TextBlock.WrapWidth = 500
	alignRight.TextBlock.Align = willow.TextAlignRight
	alignRight.TextBlock.Invalidate()
	alignRight.SetPosition(20, y)
	scene.Root.AddChild(alignRight)
	y += 20

	guideLine(y)
	y += 16

	// --- Section: Full charset ---
	sectionChars := willow.NewText("section-chars", "-- Full Character Set --", font)
	sectionChars.TextBlock.Color = willow.RGB(0.5, 0.5, 0.6)
	sectionChars.TextBlock.Invalidate()
	sectionChars.SetPosition(20, y)
	scene.Root.AddChild(sectionChars)
	y += 22

	allChars := willow.NewText("all",
		"!\"#$%&'()*+,-./  0123456789:;<=>?  @ABCDEFGHIJKLMNOPQRSTUVWXYZ  [\\]^_`abcdefghijklmnopqrstuvwxyz{|}~",
		font)
	allChars.TextBlock.WrapWidth = 760
	allChars.TextBlock.Invalidate()
	allChars.TextBlock.Color = willow.RGB(0.9, 0.85, 0.6)
	allChars.TextBlock.Invalidate()
	allChars.SetPosition(20, y)
	scene.Root.AddChild(allChars)

	if os.Getenv("WILLOW_AUTOTEST") == "1" {
		runner, _ := willow.LoadTestScript([]byte(`{"steps":[
			{"action":"wait","frames":5},
			{"action":"screenshot","label":"text-pixelfont"},
			{"action":"wait","frames":1}
		]}`))
		scene.SetTestRunner(runner)
		scene.ScreenshotDir = "screenshots"
		scene.SetUpdateFunc(func() error {
			if runner.Done() {
				return ebiten.Termination
			}
			return nil
		})
	}

	if err := willow.Run(scene, willow.RunConfig{
		Title:   windowTitle,
		Width:   screenW,
		Height:  screenH,
		ShowFPS: showFPS,
	}); err != nil {
		log.Fatal(err)
	}
}
