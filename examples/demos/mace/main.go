// Mace demonstrates a medieval flail — a spiked mace head on a chain — swinging
// around the screen using the Rope mesh for the chain and a Star polygon for
// the head. The anchor orbits slowly while the mace head whips on a longer arm.
package main

import (
	"flag"
	"log"
	"math"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowTitle = "Willow  -  Mace"
	screenW     = 320
	screenH     = 240
)

type demo struct {
	scene   *willow.Scene
	elapsed float64

	// Chain.
	rope    *willow.Rope
	start   *willow.Vec2
	end     *willow.Vec2
	control *willow.Vec2

	// Mace head group.
	maceContainer *willow.Node
	maceHead      *willow.Node
	maceBall      *willow.Node

	// Handle.
	handle *willow.Node
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.06, 0.05, 0.08)

	d := &demo{scene: scene}
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

	// --- Chain texture (metallic grey with shading) ---
	const chainTexW = 512
	const chainW = 6.0
	chainImg := ebiten.NewImage(chainTexW, int(chainW))
	for y := range int(chainW) {
		dist := math.Abs(float64(y)-chainW/2) / (chainW / 2)
		bright := 1.0 - dist*0.5
		// Alternating light/dark segments for chain-link look.
		for x := range chainTexW {
			link := float64((x / 8) % 2) // toggle every 8px
			b := bright * (0.55 + link*0.25)
			r := uint8(b * 160)
			g := uint8(b * 155)
			bl := uint8(b * 170)
			chainImg.Set(x, y, willow.ColorFromRGBA(r, g, bl, 255))
		}
	}

	// --- Rope mesh for chain ---
	d.start = &willow.Vec2{X: screenW / 2, Y: screenH / 2}
	d.end = &willow.Vec2{X: screenW / 2, Y: screenH/2 + 60}
	d.control = &willow.Vec2{X: screenW / 2, Y: screenH/2 + 30}

	d.rope = willow.NewRope("chain", chainImg, nil, willow.RopeConfig{
		Width:     chainW,
		JoinMode:  willow.RopeJoinBevel,
		CurveMode: willow.RopeCurveQuadBezier,
		Segments:  16,
		Start:     d.start,
		End:       d.end,
		Controls:  [2]*willow.Vec2{d.control},
	})
	root.AddChild(d.rope.Node())

	// --- Handle (grip at the anchor point) ---
	d.handle = willow.NewRect("handle", 14, 8, willow.RGB(0.45, 0.3, 0.2))
	d.handle.SetPivot(0.5, 0.5)
	d.handle.SetPosition(d.start.X, d.start.Y)
	root.AddChild(d.handle)

	// --- Mace head container ---
	d.maceContainer = willow.NewContainer("mace")
	d.maceContainer.SetPosition(d.end.X, d.end.Y)
	root.AddChild(d.maceContainer)

	// Ball (dark iron core).
	d.maceBall = willow.NewCircle("ball", 16, willow.RGB(0.25, 0.22, 0.28))
	d.maceBall.SetPivot(0.5, 0.5)
	d.maceContainer.AddChild(d.maceBall)

	// Spiked star overlay — thick sharp spikes.
	d.maceHead = willow.NewStar("spikes", 26, 14, 8)
	d.maceHead.SetPivot(0.5, 0.5)
	d.maceHead.SetColor(willow.RGB(0.4, 0.38, 0.45))
	d.maceContainer.AddChild(d.maceHead)

	// Inner ring for depth.
	inner := willow.NewCircle("inner", 10, willow.RGB(0.3, 0.28, 0.35))
	inner.SetPivot(0.5, 0.5)
	d.maceContainer.AddChild(inner)

	// Specular highlight.
	highlight := willow.NewCircle("highlight", 4, willow.RGB(0.55, 0.52, 0.6))
	highlight.SetPivot(0.5, 0.5)
	highlight.SetPosition(-4, -4)
	d.maceContainer.AddChild(highlight)

	// --- Spark trail emitter on the mace ---
	spark := willow.NewParticleEmitter("sparks", willow.EmitterConfig{
		MaxParticles: 80,
		EmitRate:     50,
		Lifetime:     willow.Range{Min: 0.2, Max: 0.6},
		Speed:        willow.Range{Min: 5, Max: 30},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 1, Max: 3},
		EndScale:     willow.Range{Min: 0, Max: 1},
		StartAlpha:   willow.Range{Min: 0.7, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		StartColor:   willow.RGB(1.0, 0.8, 0.4),
		EndColor:     willow.RGB(0.8, 0.3, 0.1),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	spark.SetZIndex(-1)
	d.maceContainer.AddChild(spark)
	spark.Emitter.Start()
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())
	d.elapsed += dt

	t := d.elapsed
	cx, cy := float64(screenW)/2, float64(screenH)/2

	// Anchor drifts in a small figure-8 near top-center.
	ax := cx + math.Cos(t*0.6)*20
	ay := cy*0.45 + math.Sin(t*1.2)*10

	// Mace head on a long chain, whipping wide arcs.
	maceR := 95.0
	maceAngle := t*2.2 + math.Sin(t*1.1)*1.4 + math.Cos(t*0.5)*0.6
	mx := ax + math.Cos(maceAngle)*maceR
	my := ay + math.Sin(maceAngle)*maceR

	// Chain control point: midpoint with heavy sag (gravity droop).
	midX := (ax + mx) / 2
	midY := (ay + my) / 2
	dx := mx - ax
	dy := my - ay
	chainLen := math.Sqrt(dx*dx + dy*dy)
	// More sag when chain is slack, always a decent droop.
	sag := 35.0 * (1.0 - chainLen/200.0)
	if sag < 8 {
		sag = 8
	}
	// Always droop downward plus a slight perpendicular component.
	if chainLen > 0.01 {
		nx := -dy / chainLen
		ny := dx / chainLen
		midX += nx * sag * 0.2
		midY += ny*sag*0.2 + sag
	}

	// Update rope endpoints.
	d.start.X = ax
	d.start.Y = ay
	d.end.X = mx
	d.end.Y = my
	d.control.X = midX
	d.control.Y = midY
	d.rope.Update()

	// Move handle and mace.
	d.handle.SetPosition(ax, ay)
	d.handle.SetRotation(math.Atan2(dy, dx))
	d.maceContainer.SetPosition(mx, my)

	// Spin the mace head based on angular velocity.
	d.maceHead.SetRotation(t * 4.0)

	return nil
}

func init() {
	_ = ebiten.NewImage(1, 1)
}
