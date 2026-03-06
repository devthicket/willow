// Text demonstrates font rendering with willow.NewText and willow.NewFontFromTTF.
// Shows colors, alignment, word wrapping, multi-line alignment, live content
// updates, and text effects (outline, glow, shadow).
// No external asset files  -  the Go Regular font is sourced from golang.org/x/image.
package main

import (
	"fmt"
	"log"
	// "os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	windowTitle = "Willow  -  Text Example"
	showFPS     = true
	screenW     = 800
	screenH     = 700
)

// Display font sizes (pixels).
const (
	sizeLarge  = 36.0
	sizeMedium = 22.0
	sizeSmall  = 16.0
	sizeEffect = 40.0
)

type demo struct {
	scene    *willow.Scene
	counter  *willow.Node
	frameNum int
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.08, 0.08, 0.1)

	// Try to load a system font by name, fall back to Go Regular.
	ttfData, err := willow.LoadFontFromSystemAsTtf("Arial")
	if err != nil {
		log.Printf("Arial not found, using Go Regular: %v", err)
		ttfData = goregular.TTF
	}
	font, err := willow.NewFontFromTTF(ttfData, 80)
	if err != nil {
		log.Fatalf("SDF font: %v", err)
	}

	root := scene.Root()

	// ---- Title ----------------------------------------------------------------------------------------------------------------------------------
	title := willow.NewText("title", "Willow  -  Text", font)
	title.TextBlock.FontSize = sizeLarge
	title.TextBlock.Color = willow.RGB(1, 1, 1)
	title.TextBlock.Align = willow.TextAlignRight
	title.TextBlock.WrapWidth = screenW - 48
	title.SetPosition(24, 18)
	root.AddChild(title)

	// ---- Colors --------------------------------------------------------------------------------------------------------------------------------
	addLabel(root, font, "Colors", 24, 62)

	colors := []struct {
		label string
		color willow.Color
	}{
		{"White", willow.RGB(1, 1, 1)},
		{"Cyan", willow.RGB(0.4, 0.9, 1)},
		{"Orange", willow.RGB(1, 0.6, 0.2)},
		{"Pink", willow.RGB(1, 0.4, 0.7)},
	}
	x := 24.0
	for _, c := range colors {
		n := willow.NewText("color-"+c.label, c.label, font)
		n.TextBlock.FontSize = sizeMedium
		n.TextBlock.Color = c.color
		n.SetPosition(x, 82)
		root.AddChild(n)
		w, _ := n.TextBlock.MeasureDisplay(c.label)
		x += w + 24
	}

	// ---- Single-line Alignment --------------------------------------------------------------------------------------------------
	addLabel(root, font, "Single-line Alignment", 24, 118)

	for _, a := range []struct {
		name  string
		text  string
		align willow.TextAlign
		y     float64
	}{
		{"left", "Left aligned", willow.TextAlignLeft, 138},
		{"center", "Center aligned", willow.TextAlignCenter, 164},
		{"right", "Right aligned", willow.TextAlignRight, 190},
	} {
		n := willow.NewText("align-"+a.name, a.text, font)
		n.TextBlock.FontSize = sizeMedium
		n.TextBlock.Color = willow.RGB(0.8, 0.8, 0.8)
		n.TextBlock.Align = a.align
		n.TextBlock.WrapWidth = screenW - 48
		n.SetPosition(24, a.y)
		root.AddChild(n)
	}

	// ---- Multi-line Alignment ----------------------------------------------------------------------------------------------------
	const wrapW = 300.0
	const colY = 240.0
	addLabel(root, font, "Multi-line Alignment  (wrap 300px)", 24, 224)

	guideColor := willow.RGB(0.25, 0.3, 0.35)
	colOffsets := []float64{24, 280, 536}
	for _, cx := range colOffsets {
		addGuideLine(root, cx, colY, 80, guideColor)
		addGuideLine(root, cx+wrapW, colY, 80, guideColor)
	}

	multiLine := "Short line.\nA somewhat longer second line here.\nTiny."
	wrapAligns := []struct {
		align willow.TextAlign
		color willow.Color
	}{
		{willow.TextAlignLeft, willow.RGB(0.7, 0.9, 0.7)},
		{willow.TextAlignCenter, willow.RGB(0.7, 0.8, 1)},
		{willow.TextAlignRight, willow.RGB(1, 0.8, 0.7)},
	}
	for i, wa := range wrapAligns {
		n := willow.NewText(fmt.Sprintf("wrap-%d", i), multiLine, font)
		n.TextBlock.FontSize = sizeSmall
		n.TextBlock.Color = wa.color
		n.TextBlock.Align = wa.align
		n.TextBlock.WrapWidth = wrapW
		n.SetPosition(colOffsets[i], colY)
		root.AddChild(n)
	}

	// ---- Text Effects ----------------------------------------------------------------------------------------------------------------
	addLabel(root, font, "Text Effects", 24, 346)

	// Plain
	plain := willow.NewText("fx-plain", "Plain Text", font)
	plain.TextBlock.FontSize = sizeEffect
	plain.TextBlock.Color = willow.RGB(1, 1, 1)
	plain.SetPosition(24, 370)
	root.AddChild(plain)

	// Outline
	outlined := willow.NewText("fx-outline", "Outlined", font)
	outlined.TextBlock.FontSize = sizeEffect
	outlined.TextBlock.Color = willow.RGB(1, 0.95, 0.7)

	outlined.TextBlock.TextEffects = &willow.TextEffects{
		OutlineWidth: 2.0,
		OutlineColor: willow.RGB(0.1, 0.1, 0.3),
	}
	outlined.SetPosition(24, 420)
	root.AddChild(outlined)

	// Glow
	glowing := willow.NewText("fx-glow", "Glowing", font)
	glowing.TextBlock.FontSize = sizeEffect
	glowing.TextBlock.Color = willow.RGB(0.4, 0.9, 1)

	glowing.TextBlock.TextEffects = &willow.TextEffects{
		GlowWidth: 3.0,
		GlowColor: willow.RGBA(0.2, 0.5, 1, 0.6),
	}
	glowing.SetPosition(24, 470)
	glowing.SetCacheAsTexture(true)
	root.AddChild(glowing)

	// Shadow
	shadow := willow.NewText("fx-shadow", "Shadow", font)
	shadow.TextBlock.FontSize = sizeEffect
	shadow.TextBlock.Color = willow.RGB(1, 1, 1)

	shadow.TextBlock.TextEffects = &willow.TextEffects{
		ShadowOffset:   willow.Vec2{X: 3, Y: 3},
		ShadowColor:    willow.RGBA(0, 0, 0, 0.7),
		ShadowSoftness: 1.5,
	}
	shadow.SetPosition(24, 520)
	shadow.SetCacheAsTexture(true)
	root.AddChild(shadow)

	// All effects combined
	allFx := willow.NewText("fx-all", "All Effects", font)
	allFx.TextBlock.FontSize = sizeEffect
	allFx.TextBlock.Color = willow.RGB(1, 0.85, 0.3)

	allFx.TextBlock.TextEffects = &willow.TextEffects{
		OutlineWidth:   1.5,
		OutlineColor:   willow.RGB(0.2, 0, 0),
		GlowWidth:      2.0,
		GlowColor:      willow.RGBA(1, 0.4, 0, 0.4),
		ShadowOffset:   willow.Vec2{X: 2, Y: 2},
		ShadowColor:    willow.RGBA(0, 0, 0, 0.5),
		ShadowSoftness: 1.0,
	}
	allFx.SetPosition(24, 570)
	allFx.SetCacheAsTexture(true)
	root.AddChild(allFx)

	// ---- Live update ----------------------------------------------------------------------------------------------------------------------
	addLabel(root, font, "Live update", 450, 346)

	counter := willow.NewText("counter", "Frame: 0", font)
	counter.TextBlock.FontSize = sizeMedium
	counter.TextBlock.Color = willow.RGB(0.5, 1, 0.5)
	counter.SetPosition(450, 370)
	root.AddChild(counter)

	d := &demo{scene: scene, counter: counter}
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
	d.frameNum++
	d.counter.SetContent(fmt.Sprintf("Frame: %d", d.frameNum))
	//if d.frameNum == 10 {
	//	d.scene.Screenshot("text-rendering")
	//}
	//if d.frameNum == 12 {
	//	os.Exit(0)
	//}
	return nil
}

// addLabel places a small muted section label.
func addLabel(root *willow.Node, font willow.Font, s string, x, y float64) {
	n := willow.NewText("section-"+s, s, font)
	n.TextBlock.FontSize = sizeSmall
	n.TextBlock.Color = willow.RGB(0.4, 0.5, 0.6)
	n.SetPosition(x, y)
	root.AddChild(n)
}

// addGuideLine draws a thin vertical line at (x, y) with the given height.
func addGuideLine(root *willow.Node, x, y, height float64, color willow.Color) {
	line := willow.NewSprite("guide", willow.TextureRegion{})
	line.SetColor(color)
	line.SetPosition(x, y)
	line.SetScale(1, height)
	root.AddChild(line)
}

// init registers an offscreen image so Ebitengine's internal state is
// initialized before we create atlas images in main().
func init() {
	_ = ebiten.NewImage(1, 1)
}
