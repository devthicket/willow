// SDF Blizzard demonstrates SDF font effects with an icy theme: per-letter
// bouncing, intense glow/outline/shadow effects, and falling ice particles.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowTitle = "Willow  -  SDF Blizzard"
	screenW     = 640
	screenH     = 480
)

// letter holds a container with the character node and optional emitter.
type letter struct {
	container *willow.Node // parent container that gets bounced
	node      *willow.Node // text node (child of container)
	emitter   *willow.Node // optional particle emitter (child of container)
	baseX     float64
	baseY     float64
}

type demo struct {
	scene   *willow.Scene
	font    *willow.FontFamily
	elapsed float64

	// Per-letter rows.
	titleRow []*letter // "FROZEN"
	brrrRow  []*letter // "Brrrrr!"
	frostRow []*letter // "frostbite"
	subRow   []*letter // "the chill never fades"
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.02, 0.03, 0.07)

	font, err := willow.NewFontFamilyFromFontBundle(willow.GofontBundle)
	if err != nil {
		log.Fatalf("font: %v", err)
	}

	d := &demo{scene: scene, font: font}
	d.build()

	scene.SetUpdateFunc(d.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:        windowTitle,
		Width:        screenW,
		Height:       screenH,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}

func (d *demo) build() {
	root := d.scene.Root
	cx := float64(screenW) / 2

	// --- "FROZEN" — big title with heavy outline + glow ---
	d.titleRow = d.makeLetterRow("FROZEN", 72, cx, 60,
		willow.RGB(0.75, 0.9, 1.0),
		&willow.TextEffects{
			OutlineWidth: 3.0,
			OutlineColor: willow.RGB(0.1, 0.2, 0.55),
			GlowWidth:    6.0,
			GlowColor:    willow.RGBA(0.2, 0.55, 1.0, 0.7),
		},
	)

	// --- "Brrrrr!" — medium with strong cyan glow ---
	d.brrrRow = d.makeLetterRow("Brrrrr!", 52, cx, 170,
		willow.RGB(0.8, 0.92, 1.0),
		&willow.TextEffects{
			GlowWidth: 5.0,
			GlowColor: willow.RGBA(0.15, 0.45, 1.0, 0.8),
		},
	)

	// --- "frostbite" — with glow + outline combo ---
	d.frostRow = d.makeLetterRow("frostbite", 46, cx, 275,
		willow.RGB(0.55, 0.85, 1.0),
		&willow.TextEffects{
			OutlineWidth: 1.5,
			OutlineColor: willow.RGB(0.05, 0.15, 0.4),
			GlowWidth:    7.0,
			GlowColor:    willow.RGBA(0.1, 0.35, 0.9, 0.65),
		},
	)

	// --- "the chill never fades" — with shadow + subtle glow ---
	d.subRow = d.makeLetterRow("the chill never fades", 26, cx, 370,
		willow.RGB(0.6, 0.72, 0.9),
		&willow.TextEffects{
			GlowWidth:      3.0,
			GlowColor:      willow.RGBA(0.15, 0.3, 0.8, 0.4),
			ShadowOffset:   willow.Vec2{X: 3, Y: 4},
			ShadowColor:    willow.RGBA(0, 0, 0.1, 0.8),
			ShadowSoftness: 2.5,
		},
	)

	// --- Per-letter ice particle emitters on "FROZEN" ---
	for _, l := range d.titleRow {
		em := willow.NewParticleEmitter("ice", willow.EmitterConfig{
			MaxParticles: 40,
			EmitRate:     12,
			Lifetime:     willow.Range{Min: 1.0, Max: 2.5},
			Speed:        willow.Range{Min: 8, Max: 35},
			Angle:        willow.Range{Min: math.Pi * 0.35, Max: math.Pi * 0.65},
			StartScale:   willow.Range{Min: 2, Max: 5},
			EndScale:     willow.Range{Min: 0, Max: 2},
			StartAlpha:   willow.Range{Min: 0.5, Max: 0.9},
			EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
			Gravity:      willow.Vec2{X: 0, Y: 40},
			StartColor:   willow.RGB(0.6, 0.85, 1.0),
			EndColor:     willow.RGB(0.25, 0.45, 0.9),
			BlendMode:    willow.BlendAdd,
			WorldSpace:   true,
		})
		// Position at bottom-center of the letter (relative to container).
		em.SetPosition(20, 65)
		em.SetZIndex(-1) // behind the letter text
		em.Emitter.Start()
		l.container.AddChild(em)
		l.emitter = em
	}

	// --- Decorative ice bars ---
	topBar := willow.NewRect("bar-top", screenW, 2, willow.RGBA(0.3, 0.6, 1.0, 0.2))
	topBar.SetPosition(0, 25)
	root.AddChild(topBar)

	botBar := willow.NewRect("bar-bot", screenW, 2, willow.RGBA(0.3, 0.6, 1.0, 0.2))
	botBar.SetPosition(0, 450)
	root.AddChild(botBar)
}

// makeLetterRow splits text into individual letter nodes centered at (cx, y).
// Each letter lives inside a container so emitters can be added as siblings.
func (d *demo) makeLetterRow(text string, fontSize, cx, y float64, color willow.Color, fx *willow.TextEffects) []*letter {
	root := d.scene.Root
	runes := []rune(text)

	// Measure each character width.
	tmp := willow.NewText("measure", "", d.font)
	tmp.TextBlock.FontSize = fontSize
	widths := make([]float64, len(runes))
	totalW := 0.0
	for i, r := range runes {
		w, _ := tmp.TextBlock.MeasureDisplay(string(r))
		widths[i] = w
		totalW += w
	}

	letters := make([]*letter, len(runes))
	x := cx - totalW/2
	for i, r := range runes {
		ch := string(r)

		// Container at the letter's position — bouncing moves the whole group.
		ctr := willow.NewContainer(fmt.Sprintf("ctr-%s-%d", text[:1], i))
		ctr.SetPosition(x, y)
		root.AddChild(ctr)

		// Text node at (0,0) relative to container.
		n := willow.NewText(fmt.Sprintf("ch-%s-%d", text[:1], i), ch, d.font)
		n.TextBlock.FontSize = fontSize
		n.TextBlock.Color = color
		if fx != nil {
			fxCopy := *fx
			n.TextBlock.TextEffects = &fxCopy
		}
		n.SetCacheAsTexture(true)
		ctr.AddChild(n)

		letters[i] = &letter{container: ctr, node: n, baseX: x, baseY: y}
		x += widths[i]
	}
	return letters
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())
	d.elapsed += dt

	// "FROZEN" — slow, stately wave.
	bounceRow(d.titleRow, d.elapsed, 1.2, 8.0, 0.25)

	// "Brrrrr!" — fast shivering bounce.
	bounceRow(d.brrrRow, d.elapsed, 3.5, 6.0, 0.15)

	// "frostbite" — medium wave.
	bounceRow(d.frostRow, d.elapsed, 1.8, 5.0, 0.20)

	// "the chill never fades" — gentle drift.
	bounceRow(d.subRow, d.elapsed, 1.0, 3.0, 0.30)

	return nil
}

// bounceRow applies a staggered sine-wave bounce to each letter's container.
// freq controls speed, amp controls height, stagger controls phase offset per letter.
func bounceRow(row []*letter, t, freq, amp, stagger float64) {
	for i, l := range row {
		phase := float64(i) * stagger
		dy := math.Sin((t*freq+phase)*math.Pi*2) * amp
		l.container.SetPosition(l.baseX, l.baseY+dy)
	}
}

func init() {
	_ = ebiten.NewImage(1, 1)
}
