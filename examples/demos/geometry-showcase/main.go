// Geometry Showcase demonstrates the breadth of Willow's polygon and custom
// geometry primitives: regular polygons, stars, custom polygons, circles,
// triangles, lines, and runtime vertex updates — all animated with tweens
// and per-frame rotation.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween/ease"
)

const (
	windowTitle = "Willow  -  Geometry Showcase"
	screenW     = 800
	screenH     = 600
)

// palette of saturated colors against the dark background.
var colors = struct {
	rose, amber, mint, sky, violet, cyan, coral, lime willow.Color
}{
	rose:   willow.RGB(1.0, 0.35, 0.5),
	amber:  willow.RGB(1.0, 0.7, 0.15),
	mint:   willow.RGB(0.25, 0.95, 0.55),
	sky:    willow.RGB(0.3, 0.65, 1.0),
	violet: willow.RGB(0.7, 0.35, 1.0),
	cyan:   willow.RGB(0.1, 0.9, 0.85),
	coral:  willow.RGB(1.0, 0.5, 0.35),
	lime:   willow.RGB(0.6, 1.0, 0.2),
}

type demo struct {
	scene  *willow.Scene
	tweens []*willow.TweenGroup

	// Geometry nodes.
	hexRing  [6]*willow.Node // ring of hexagons
	starNode *willow.Node    // morphing star
	morphPts int             // current star point count (toggles 5↔8)

	arrowNode *willow.Node // custom arrow polygon
	gemNode   *willow.Node // custom gem polygon

	circleOrbit [3]*willow.Node // orbiting circles
	orbitAngle  float64

	triStack [4]*willow.Node // stacked triangles

	crossLines [4]*willow.Node // decorative cross lines

	phase int // animation phase counter
}

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.05, 0.05, 0.09)

	d := &demo{scene: scene}
	d.build()
	d.animateIn()

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

	// --- Hexagon ring (upper-left) -------------------------------------------
	hx, hy := 180.0, 180.0
	for i := range 6 {
		hex := willow.NewRegularPolygon("hex", 6, 28)
		angle := float64(i) * math.Pi / 3
		hex.SetPosition(hx+math.Cos(angle)*70, hy+math.Sin(angle)*70)
		hex.SetPivot(0.5, 0.5)
		hex.SetScale(0, 0)
		hex.SetAlpha(0)
		cl := []willow.Color{colors.sky, colors.mint, colors.amber, colors.violet, colors.rose, colors.cyan}
		hex.SetColor(cl[i])
		root.AddChild(hex)
		d.hexRing[i] = hex

		// Gentle individual spin.
		speed := 0.3 + float64(i)*0.1
		hex.OnUpdate = func(dt float64) {
			hex.SetRotation(hex.Rotation() + speed*dt)
		}
	}

	// --- Morphing star (upper-right) -----------------------------------------
	d.morphPts = 5
	d.starNode = willow.NewStar("star", 60, 28, 5)
	d.starNode.SetPosition(620, 180)
	d.starNode.SetPivot(0.5, 0.5)
	d.starNode.SetColor(colors.amber)
	d.starNode.SetScale(0, 0)
	d.starNode.SetAlpha(0)
	d.starNode.OnUpdate = func(dt float64) {
		d.starNode.SetRotation(d.starNode.Rotation() + 0.4*dt)
	}
	root.AddChild(d.starNode)

	// --- Custom arrow polygon (center-left) ----------------------------------
	d.arrowNode = willow.NewPolygon("arrow", arrowPoints())
	d.arrowNode.SetPosition(160, 420)
	d.arrowNode.SetPivot(0.5, 0.5)
	d.arrowNode.SetColor(colors.coral)
	d.arrowNode.SetScale(0, 0)
	d.arrowNode.SetAlpha(0)
	root.AddChild(d.arrowNode)

	// --- Custom gem polygon (center-right) -----------------------------------
	d.gemNode = willow.NewPolygon("gem", gemPoints())
	d.gemNode.SetPosition(640, 420)
	d.gemNode.SetPivot(0.5, 0.5)
	d.gemNode.SetColor(colors.cyan)
	d.gemNode.SetScale(0, 0)
	d.gemNode.SetAlpha(0)
	d.gemNode.OnUpdate = func(dt float64) {
		d.gemNode.SetRotation(d.gemNode.Rotation() - 0.25*dt)
	}
	root.AddChild(d.gemNode)

	// --- Orbiting circles (center) -------------------------------------------
	ox, oy := 400.0, 300.0
	orbitColors := []willow.Color{colors.rose, colors.mint, colors.violet}
	for i := range 3 {
		c := willow.NewCircle(fmt.Sprintf("orb-%d", i), 14, orbitColors[i])
		c.SetPosition(ox, oy)
		c.SetPivot(0.5, 0.5)
		c.SetAlpha(0)
		root.AddChild(c)
		d.circleOrbit[i] = c
	}

	// Center dot.
	dot := willow.NewCircle("center-dot", 6, willow.RGB(1, 1, 1))
	dot.SetPosition(ox, oy)
	dot.SetPivot(0.5, 0.5)
	dot.SetAlpha(0.5)
	root.AddChild(dot)

	// --- Stacked triangles (lower-center) ------------------------------------
	tx, ty := 400.0, 510.0
	triColors := []willow.Color{colors.lime, colors.sky, colors.amber, colors.rose}
	for i := range 4 {
		sz := 40.0 - float64(i)*6
		tri := willow.NewTriangle(fmt.Sprintf("tri-%d", i),
			willow.Vec2{X: 0, Y: -sz},
			willow.Vec2{X: sz * 0.87, Y: sz * 0.5},
			willow.Vec2{X: -sz * 0.87, Y: sz * 0.5},
			triColors[i],
		)
		tri.SetPosition(tx, ty)
		tri.SetPivot(0.5, 0.5)
		tri.SetScale(0, 0)
		tri.SetAlpha(0)
		speed := 0.5 * float64(i+1)
		if i%2 == 1 {
			speed = -speed
		}
		tri.OnUpdate = func(dt float64) {
			tri.SetRotation(tri.Rotation() + speed*dt)
		}
		root.AddChild(tri)
		d.triStack[i] = tri
	}

	// --- Decorative cross lines (background, behind everything) ---------------
	cx, cy := 400.0, 300.0
	lineLen := 260.0
	lineAngles := []float64{0, math.Pi / 4, math.Pi / 2, 3 * math.Pi / 4}
	lineColors := []willow.Color{
		willow.RGBA(1, 1, 1, 0.06),
		willow.RGBA(1, 1, 1, 0.04),
		willow.RGBA(1, 1, 1, 0.06),
		willow.RGBA(1, 1, 1, 0.04),
	}
	for i, a := range lineAngles {
		x1 := cx - math.Cos(a)*lineLen
		y1 := cy - math.Sin(a)*lineLen
		x2 := cx + math.Cos(a)*lineLen
		y2 := cy + math.Sin(a)*lineLen
		ln := willow.NewLine(fmt.Sprintf("line-%d", i), x1, y1, x2, y2, 1.5, lineColors[i])
		ln.SetZIndex(-10)
		root.AddChild(ln)
		d.crossLines[i] = ln
	}
}

func (d *demo) animateIn() {
	d.phase = 0
	d.tweens = d.tweens[:0]

	// Hexagon ring: staggered pop-in.
	for i, hex := range d.hexRing {
		dur := float32(0.6 + float64(i)*0.08)
		d.tweens = append(d.tweens,
			willow.TweenScale(hex, 1, 1, willow.TweenConfig{Duration: dur, Ease: ease.OutBack}),
			willow.TweenAlpha(hex, 0.9, willow.TweenConfig{Duration: dur * 0.6, Ease: ease.OutQuad}),
		)
	}

	// Star: elastic scale-in.
	d.tweens = append(d.tweens,
		willow.TweenScale(d.starNode, 1, 1, willow.TweenConfig{Duration: 0.9, Ease: ease.OutElastic}),
		willow.TweenAlpha(d.starNode, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.OutQuad}),
	)

	// Arrow + gem: slide-scale in.
	d.tweens = append(d.tweens,
		willow.TweenScale(d.arrowNode, 1, 1, willow.TweenConfig{Duration: 0.7, Ease: ease.OutBack}),
		willow.TweenAlpha(d.arrowNode, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.OutQuad}),
		willow.TweenScale(d.gemNode, 1, 1, willow.TweenConfig{Duration: 0.8, Ease: ease.OutBack}),
		willow.TweenAlpha(d.gemNode, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.OutQuad}),
	)

	// Orbiting circles: fade in.
	for _, c := range d.circleOrbit {
		d.tweens = append(d.tweens,
			willow.TweenAlpha(c, 1, willow.TweenConfig{Duration: 0.8, Ease: ease.OutQuad}),
		)
	}

	// Triangle stack: staggered pop.
	for i, tri := range d.triStack {
		dur := float32(0.5 + float64(i)*0.1)
		d.tweens = append(d.tweens,
			willow.TweenScale(tri, 1, 1, willow.TweenConfig{Duration: dur, Ease: ease.OutBack}),
			willow.TweenAlpha(tri, 0.85, willow.TweenConfig{Duration: dur * 0.5, Ease: ease.OutQuad}),
		)
	}
}

func (d *demo) animatePulse() {
	d.phase = 1
	d.tweens = d.tweens[:0]

	// Hexagons: scale pulse outward.
	for i, hex := range d.hexRing {
		dur := float32(0.5 + float64(i)*0.06)
		d.tweens = append(d.tweens,
			willow.TweenScale(hex, 1.4, 1.4, willow.TweenConfig{Duration: dur, Ease: ease.OutCubic}),
		)
	}

	// Star: cycle through point counts and colors on each pulse.
	starCycle := []int{5, 8, 12, 6}
	starColors := []willow.Color{colors.lime, colors.rose, colors.cyan, colors.amber}
	idx := 0
	for i, v := range starCycle {
		if v == d.morphPts {
			idx = (i + 1) % len(starCycle)
			break
		}
	}
	d.morphPts = starCycle[idx]
	willow.SetPolygonPoints(d.starNode, starPoints(d.morphPts))
	d.tweens = append(d.tweens,
		willow.TweenScale(d.starNode, 1.4, 1.4, willow.TweenConfig{Duration: 0.5, Ease: ease.OutElastic}),
		willow.TweenColor(d.starNode, starColors[idx], willow.TweenConfig{Duration: 0.5, Ease: ease.InOutSine}),
	)

	// Arrow: rotate quarter turn.
	d.tweens = append(d.tweens,
		willow.TweenRotation(d.arrowNode, d.arrowNode.Rotation()+math.Pi/2, willow.TweenConfig{Duration: 0.6, Ease: ease.OutBack}),
		willow.TweenScale(d.arrowNode, 1.2, 1.2, willow.TweenConfig{Duration: 0.5, Ease: ease.OutCubic}),
	)

	// Gem: color shift + scale.
	d.tweens = append(d.tweens,
		willow.TweenColor(d.gemNode, colors.violet, willow.TweenConfig{Duration: 0.8, Ease: ease.InOutSine}),
		willow.TweenScale(d.gemNode, 1.3, 1.3, willow.TweenConfig{Duration: 0.6, Ease: ease.OutCubic}),
	)

	// Triangles: scale bump.
	for i, tri := range d.triStack {
		dur := float32(0.4 + float64(i)*0.08)
		d.tweens = append(d.tweens,
			willow.TweenScale(tri, 1.3, 1.3, willow.TweenConfig{Duration: dur, Ease: ease.OutCubic}),
		)
	}
}

func (d *demo) animateSettle() {
	d.phase = 2
	d.tweens = d.tweens[:0]

	// Everything eases back to resting scale.
	for _, hex := range d.hexRing {
		d.tweens = append(d.tweens,
			willow.TweenScale(hex, 1, 1, willow.TweenConfig{Duration: 0.6, Ease: ease.InOutCubic}),
		)
	}

	d.tweens = append(d.tweens,
		willow.TweenScale(d.starNode, 1, 1, willow.TweenConfig{Duration: 0.35, Ease: ease.InOutCubic}),
		willow.TweenScale(d.arrowNode, 1, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.InOutCubic}),
		willow.TweenScale(d.gemNode, 1, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.InOutCubic}),
		willow.TweenColor(d.gemNode, colors.cyan, willow.TweenConfig{Duration: 0.7, Ease: ease.InOutSine}),
	)

	for _, tri := range d.triStack {
		d.tweens = append(d.tweens,
			willow.TweenScale(tri, 1, 1, willow.TweenConfig{Duration: 0.5, Ease: ease.InOutCubic}),
		)
	}
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())

	// Orbit the three circles around center.
	d.orbitAngle += dt * 0.8
	ox, oy := 400.0, 300.0
	for i, c := range d.circleOrbit {
		a := d.orbitAngle + float64(i)*2*math.Pi/3
		r := 55.0 + 15.0*math.Sin(d.orbitAngle*1.5+float64(i))
		c.SetPosition(ox+math.Cos(a)*r, oy+math.Sin(a)*r)
	}

	// Check tween completion.
	allDone := true
	for _, tw := range d.tweens {
		if !tw.Done {
			allDone = false
			break
		}
	}

	if allDone && len(d.tweens) > 0 {
		switch d.phase {
		case 0:
			d.animatePulse()
		case 1:
			d.animateSettle()
		case 2:
			d.animatePulse()
		}
	}

	return nil
}

// --- Geometry helpers --------------------------------------------------------

func arrowPoints() []willow.Vec2 {
	return []willow.Vec2{
		{X: 0, Y: -40},  // tip
		{X: 25, Y: -5},  // right shoulder
		{X: 12, Y: -5},  // right notch
		{X: 12, Y: 35},  // right base
		{X: -12, Y: 35}, // left base
		{X: -12, Y: -5}, // left notch
		{X: -25, Y: -5}, // left shoulder
	}
}

func gemPoints() []willow.Vec2 {
	return []willow.Vec2{
		{X: 0, Y: -50},   // top
		{X: 35, Y: -20},  // upper-right
		{X: 30, Y: 15},   // mid-right
		{X: 0, Y: 50},    // bottom
		{X: -30, Y: 15},  // mid-left
		{X: -35, Y: -20}, // upper-left
	}
}

func starPoints(points int) []willow.Vec2 {
	outerR := 60.0
	innerR := 28.0
	n := points * 2
	pts := make([]willow.Vec2, n)
	for i := range pts {
		a := -math.Pi/2 + float64(i)*math.Pi/float64(points)
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		pts[i] = willow.Vec2{X: math.Cos(a) * r, Y: math.Sin(a) * r}
	}
	return pts
}
