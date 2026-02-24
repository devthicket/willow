// Text demonstrates SDF font rendering with willow.NewText and willow.NewFontFromTTF.
// Shows colors, alignment, word wrapping, multi-line alignment, live content
// updates, and SDF text effects (outline, glow, shadow).
// No external asset files — the Go Regular font is sourced from golang.org/x/image.
package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	windowTitle = "Willow — SDF Text Example"
	showFPS     = true
	screenW     = 800
	screenH     = 700
)

type demo struct {
	scene    *willow.Scene
	counter  *willow.Node
	frameNum int
}

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.08, G: 0.08, B: 0.1, A: 1}

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

	// Scale helpers: 80px atlas → display sizes.
	const large = 0.45  // ~36px
	const medium = 0.28 // ~22px
	const small = 0.20  // ~16px

	// ── Title ─────────────────────────────────────────────────────────────────
	title := willow.NewText("title", "Willow — SDF Text", font)
	title.TextBlock.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	title.TextBlock.Align = willow.TextAlignRight
	title.TextBlock.WrapWidth = (screenW - 48) / large
	title.ScaleX = large
	title.ScaleY = large
	title.X = 24
	title.Y = 18
	root.AddChild(title)

	// ── Colors ────────────────────────────────────────────────────────────────
	addLabel(root, font, small, "Colors", 24, 62)

	colors := []struct {
		label string
		color willow.Color
	}{
		{"White", willow.Color{R: 1, G: 1, B: 1, A: 1}},
		{"Cyan", willow.Color{R: 0.4, G: 0.9, B: 1, A: 1}},
		{"Orange", willow.Color{R: 1, G: 0.6, B: 0.2, A: 1}},
		{"Pink", willow.Color{R: 1, G: 0.4, B: 0.7, A: 1}},
	}
	x := 24.0
	for _, c := range colors {
		n := willow.NewText("color-"+c.label, c.label, font)
		n.TextBlock.Color = c.color
		n.ScaleX = medium
		n.ScaleY = medium
		n.X = x
		n.Y = 82
		root.AddChild(n)
		w, _ := font.MeasureString(c.label)
		x += w*medium + 24
	}

	// ── Single-line Alignment ─────────────────────────────────────────────────
	addLabel(root, font, small, "Single-line Alignment", 24, 118)

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
		n.TextBlock.Color = willow.Color{R: 0.8, G: 0.8, B: 0.8, A: 1}
		n.TextBlock.Align = a.align
		n.TextBlock.WrapWidth = (screenW - 48) / medium
		n.ScaleX = medium
		n.ScaleY = medium
		n.X = 24
		n.Y = a.y
		root.AddChild(n)
	}

	// ── Multi-line Alignment ──────────────────────────────────────────────────
	const wrapW = 300.0
	const colY = 240.0
	addLabel(root, font, small, "Multi-line Alignment  (wrap 300px)", 24, 224)

	guideColor := willow.Color{R: 0.25, G: 0.3, B: 0.35, A: 1}
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
		{willow.TextAlignLeft, willow.Color{R: 0.7, G: 0.9, B: 0.7, A: 1}},
		{willow.TextAlignCenter, willow.Color{R: 0.7, G: 0.8, B: 1, A: 1}},
		{willow.TextAlignRight, willow.Color{R: 1, G: 0.8, B: 0.7, A: 1}},
	}
	for i, wa := range wrapAligns {
		n := willow.NewText(fmt.Sprintf("wrap-%d", i), multiLine, font)
		n.TextBlock.Color = wa.color
		n.TextBlock.Align = wa.align
		n.TextBlock.WrapWidth = wrapW / small
		n.ScaleX = small
		n.ScaleY = small
		n.X = colOffsets[i]
		n.Y = colY
		root.AddChild(n)
	}

	// ── SDF Text Effects ─────────────────────────────────────────────────────
	addLabel(root, font, small, "SDF Text Effects", 24, 346)

	const effectScale = 0.5

	// Plain
	sdfPlain := willow.NewText("sdf-plain", "Plain Text", font)
	sdfPlain.TextBlock.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	sdfPlain.ScaleX = effectScale
	sdfPlain.ScaleY = effectScale
	sdfPlain.X = 24
	sdfPlain.Y = 370
	root.AddChild(sdfPlain)

	// Outline
	sdfOutlined := willow.NewText("sdf-outline", "Outlined", font)
	sdfOutlined.TextBlock.Color = willow.Color{R: 1, G: 0.95, B: 0.7, A: 1}
	sdfOutlined.TextBlock.SDFEffects = &willow.SDFEffects{
		OutlineWidth: 2.0,
		OutlineColor: willow.Color{R: 0.1, G: 0.1, B: 0.3, A: 1},
	}
	sdfOutlined.ScaleX = effectScale
	sdfOutlined.ScaleY = effectScale
	sdfOutlined.X = 24
	sdfOutlined.Y = 420
	root.AddChild(sdfOutlined)

	// Glow
	sdfGlow := willow.NewText("sdf-glow", "Glowing", font)
	sdfGlow.TextBlock.Color = willow.Color{R: 0.4, G: 0.9, B: 1, A: 1}
	sdfGlow.TextBlock.SDFEffects = &willow.SDFEffects{
		GlowWidth: 3.0,
		GlowColor: willow.Color{R: 0.2, G: 0.5, B: 1, A: 0.6},
	}
	sdfGlow.ScaleX = effectScale
	sdfGlow.ScaleY = effectScale
	sdfGlow.X = 24
	sdfGlow.Y = 470
	root.AddChild(sdfGlow)

	// Shadow
	sdfShadow := willow.NewText("sdf-shadow", "Shadow", font)
	sdfShadow.TextBlock.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	sdfShadow.TextBlock.SDFEffects = &willow.SDFEffects{
		ShadowOffset:   willow.Vec2{X: 3, Y: 3},
		ShadowColor:    willow.Color{R: 0, G: 0, B: 0, A: 0.7},
		ShadowSoftness: 1.5,
	}
	sdfShadow.ScaleX = effectScale
	sdfShadow.ScaleY = effectScale
	sdfShadow.X = 24
	sdfShadow.Y = 520
	root.AddChild(sdfShadow)

	// All effects combined
	sdfAll := willow.NewText("sdf-all", "All Effects", font)
	sdfAll.TextBlock.Color = willow.Color{R: 1, G: 0.85, B: 0.3, A: 1}
	sdfAll.TextBlock.SDFEffects = &willow.SDFEffects{
		OutlineWidth:   1.5,
		OutlineColor:   willow.Color{R: 0.2, G: 0, B: 0, A: 1},
		GlowWidth:      2.0,
		GlowColor:      willow.Color{R: 1, G: 0.4, B: 0, A: 0.4},
		ShadowOffset:   willow.Vec2{X: 2, Y: 2},
		ShadowColor:    willow.Color{R: 0, G: 0, B: 0, A: 0.5},
		ShadowSoftness: 1.0,
	}
	sdfAll.ScaleX = effectScale
	sdfAll.ScaleY = effectScale
	sdfAll.X = 24
	sdfAll.Y = 570
	root.AddChild(sdfAll)

	// ── Live update ───────────────────────────────────────────────────────────
	addLabel(root, font, small, "Live update", 450, 346)

	counter := willow.NewText("counter", "Frame: 0", font)
	counter.TextBlock.Color = willow.Color{R: 0.5, G: 1, B: 0.5, A: 1}
	counter.ScaleX = medium
	counter.ScaleY = medium
	counter.X = 450
	counter.Y = 370
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
	d.counter.TextBlock.Content = fmt.Sprintf("Frame: %d", d.frameNum)
	d.counter.TextBlock.Invalidate()
	return nil
}

// addLabel places a small muted section label.
func addLabel(root *willow.Node, font willow.Font, scale float64, s string, x, y float64) {
	n := willow.NewText("section-"+s, s, font)
	n.TextBlock.Color = willow.Color{R: 0.4, G: 0.5, B: 0.6, A: 1}
	n.ScaleX = scale
	n.ScaleY = scale
	n.X = x
	n.Y = y
	root.AddChild(n)
}

// addGuideLine draws a thin vertical line at (x, y) with the given height.
func addGuideLine(root *willow.Node, x, y, height float64, color willow.Color) {
	line := willow.NewSprite("guide", willow.TextureRegion{})
	line.Color = color
	line.X = x
	line.Y = y
	line.ScaleX = 1
	line.ScaleY = height
	root.AddChild(line)
}

// init registers an offscreen image so Ebitengine's internal state is
// initialized before we create atlas images in main().
func init() {
	_ = ebiten.NewImage(1, 1)
}
