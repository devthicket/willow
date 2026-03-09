// Underwater demonstrates a layered underwater scene revealed through a circular
// porthole mask that follows the cursor. The surface is an opaque DistortionGrid
// with animated waves; beneath it, three additional water meshes at varying depth,
// seaweed, a swimming whelp, rising bubbles, caustic lighting, and a sandy floor
// create an atmospheric underwater world. Click to spawn a bubble burst.
package main

import (
	"image"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

const (
	windowTitle = "Willow  -  Underwater"
	showFPS     = true
	screenW     = 1280
	screenH     = 720
	tileSize    = 32

	gridCols  = screenW / tileSize
	gridRows  = screenH / tileSize
	bufTiles  = 1
	totalCols = gridCols + bufTiles*2
	totalRows = gridRows + bufTiles*2

	portholeRadius = 200.0
	portholeSegs   = 64

	numSeaweed = 6
)

type demo struct {
	scene *willow.Scene
	time  float64

	// Porthole tracking  -  move together with the cursor
	maskChild         *willow.Node // circle inside underwater mask container
	surfaceEraseChild *willow.Node // BlendErase circle inside surface mask layer 1
	surfaceEraseHole2 *willow.Node // BlendErase circle inside surface mask layer 2
	ringContainer     *willow.Node // unused, kept to avoid nil in update

	// Water grids
	deepGrid     *willow.DistortionGrid
	midGrid      *willow.DistortionGrid
	nearGrid     *willow.DistortionGrid
	overlayGrid1 *willow.DistortionGrid
	overlayGrid2 *willow.DistortionGrid

	// Underwater elements
	seaweed    [numSeaweed]*willow.Node
	whelp      *willow.Node
	lightLayer *willow.LightLayer
	lights     [4]*willow.Light

	// Burst emitter
	burst        *willow.Node
	burstTimer   float64
	mouseWasDown bool
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())
	d.time += dt
	t := d.time
	// --- Track cursor for porthole + surface hole + ring ---
	mx, my := ebiten.CursorPosition()
	cx, cy := float64(mx), float64(my)
	d.maskChild.SetPosition(cx, cy)
	d.surfaceEraseChild.SetPosition(cx, cy)
	d.surfaceEraseHole2.SetPosition(cx, cy)
	d.ringContainer.SetPosition(cx, cy)

	// --- Ripple the porthole mask edges ---
	ripple := rippleCircle(portholeRadius, portholeSegs, t)
	willow.SetPolygonPoints(d.maskChild, ripple)
	willow.SetPolygonPoints(d.surfaceEraseChild, ripple)
	willow.SetPolygonPoints(d.surfaceEraseHole2, ripple)

	// --- Animate deep water (slow, muted) ---
	d.deepGrid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		wave1 := math.Sin(restX*0.02 + restY*0.015 + t*0.8)
		wave2 := math.Sin(restX*0.015 - restY*0.025 + t*0.6 + 1.2)
		dy = 5.0 * (wave1 + 0.4*wave2)
		dx = 1.5 * math.Sin(restY*0.025+t*0.48)
		return
	})

	// --- Animate mid water ---
	d.midGrid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		wave1 := math.Sin(restX*0.03 + restY*0.023 + t*1.4)
		wave2 := math.Sin(restX*0.023 - restY*0.038 + t*1.05 + 1.2)
		dy = 5.0 * (wave1 + 0.4*wave2)
		dx = 1.5 * math.Sin(restY*0.038+t*0.84)
		return
	})

	// --- Animate near water (fast, turbulent) ---
	d.nearGrid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		wave1 := math.Sin(restX*0.048 + restY*0.036 + t*2.6)
		wave2 := math.Sin(restX*0.036 - restY*0.06 + t*1.95 + 1.2)
		dy = 5.0 * (wave1 + 0.4*wave2)
		dx = 1.5 * math.Sin(restY*0.06+t*1.56)
		return
	})

	// --- Animate overlay water layers (full-screen, offset from each other) ---
	d.overlayGrid1.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		dx = 10.0 * math.Sin(restY*0.03+t*0.9)
		dy = 3.0 * math.Sin(restX*0.04+t*1.2+1.0)
		return
	})
	d.overlayGrid2.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		dx = 3.0 * math.Sin(restY*0.035+t*1.4+2.5)
		dy = 10.0 * math.Sin(restX*0.025+t*0.7)
		return
	})

	// --- Sway seaweed ---
	for i, sw := range d.seaweed {
		phase := float64(i) * 1.7
		sw.SetRotation(0.12 * math.Sin(t*1.3+phase))
	}

	// --- Bob and drift whelp ---
	d.whelp.SetPosition(screenW*0.5+80*math.Sin(t*0.25), screenH*0.38+10*math.Sin(t*0.9))
	d.whelp.SetRotation(0.08 * math.Sin(t*0.7+0.5))

	// --- Drift caustic lights ---
	for i, light := range d.lights {
		phase := float64(i) * math.Pi * 0.5
		light.X = float64(screenW)*(0.15+float64(i)*0.25) + 60*math.Sin(t*0.4+phase)
		light.Y = float64(screenH)*0.4 + 50*math.Sin(t*0.35+phase*1.3)
		light.Intensity = 0.8 + 0.2*math.Sin(t*1.2+phase)
	}
	d.lightLayer.Redraw()

	// --- Click detection (direct, avoids scene input ordering lag) ---
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if down && !d.mouseWasDown {
		d.burst.SetPosition(cx, cy)
		d.burst.Emitter.Reset()
		d.burst.Emitter.Start()
		d.burstTimer = 0.10
	}
	d.mouseWasDown = down

	// --- Burst timer ---
	if d.burstTimer > 0 {
		d.burstTimer -= dt
		if d.burstTimer <= 0 {
			d.burst.Emitter.Stop()
		}
	}

	return nil
}

func main() {
	// --- Load assets ---
	tf, err := os.Open("examples/_assets/tileset.png")
	if err != nil {
		log.Fatalf("open tileset.png: %v", err)
	}
	defer tf.Close()
	tileset, _, err := ebitenutil.NewImageFromReader(tf)
	if err != nil {
		log.Fatalf("decode tileset.png: %v", err)
	}

	wf, err := os.Open("examples/_assets/whelp.png")
	if err != nil {
		log.Fatalf("open whelp.png: %v", err)
	}
	defer wf.Close()
	whelpImg, _, err := ebitenutil.NewImageFromReader(wf)
	if err != nil {
		log.Fatalf("decode whelp.png: %v", err)
	}

	// Water tile: row 3, col 0
	waterSrc := tileset.SubImage(image.Rect(0, 96, tileSize, 128)).(*ebiten.Image)
	// Sand tile: GID 10 (1-indexed) -> row 2, col 1 in 4-wide tileset.
	sandSrc := tileset.SubImage(image.Rect(32, 64, 64, 96)).(*ebiten.Image)

	// tileWater pre-renders the water tile across the grid area at a given
	// scale factor. scale=1 tiles at native 32px, scale=1.4 tiles larger.
	imgW, imgH := totalCols*tileSize, totalRows*tileSize
	tileWater := func(scale float64) *ebiten.Image {
		img := ebiten.NewImage(imgW, imgH)
		var o ebiten.DrawImageOptions
		ts := float64(tileSize) * scale
		cols := int(math.Ceil(float64(imgW)/ts)) + 1
		rows := int(math.Ceil(float64(imgH)/ts)) + 1
		for gy := range rows {
			for gx := range cols {
				o.GeoM.Reset()
				o.GeoM.Scale(scale, scale)
				o.GeoM.Translate(float64(gx)*ts, float64(gy)*ts)
				img.DrawImage(waterSrc, &o)
			}
		}
		return img
	}

	// Three water images at different tile scales for visual variety.
	waterImg := tileWater(1.0)  // base scale (surface layer 1 + underwater layers)
	waterImg2 := tileWater(1.4) // larger tiles (surface layer 2)

	// Pre-render tiled sand strip (full width, extends past screen bottom)
	sandTilesY := (screenH-int(screenH*0.65))/tileSize + 3
	sandTilesX := screenW/tileSize + 2
	sandImg := ebiten.NewImage(sandTilesX*tileSize, sandTilesY*tileSize)
	var sandOp ebiten.DrawImageOptions
	for gy := range sandTilesY {
		for gx := range sandTilesX {
			sandOp.GeoM.Reset()
			sandOp.GeoM.Translate(float64(gx*tileSize), float64(gy*tileSize))
			sandImg.DrawImage(sandSrc, &sandOp)
		}
	}

	// --- Scene setup ---
	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.05, 0.12, 0.28)

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	d := &demo{scene: scene}

	// --- Underwater container (masked by porthole circle) ---
	underwaterContainer := willow.NewContainer("underwater")

	maskRoot := willow.NewContainer("mask-root")
	maskChild := willow.NewPolygon("porthole", circlePoints(portholeRadius, portholeSegs))
	maskChild.SetColor(willow.RGB(1, 1, 1))
	maskChild.SetPosition(screenW/2, screenH/2)
	maskRoot.AddChild(maskChild)
	underwaterContainer.SetMask(maskRoot)
	d.maskChild = maskChild

	// --- Deep water mesh (ZIndex 0) ---
	deepGrid := willow.NewDistortionGrid("deep-water", waterImg, totalCols, totalRows)
	deepGrid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	deepGrid.Node().SetColor(willow.RGB(0.35, 0.55, 0.75))
	deepGrid.Node().SetZIndex(0)
	underwaterContainer.AddChild(deepGrid.Node())
	d.deepGrid = deepGrid

	// --- Sandy floor (ZIndex 2) ---
	sandFloor := willow.NewSprite("sand-floor", willow.TextureRegion{})
	sandFloor.SetCustomImage(sandImg)
	sandFloor.SetColor(willow.RGB(0.9, 0.75, 0.55))
	sandFloor.SetPosition(0, screenH*0.65)
	sandFloor.SetZIndex(2)
	underwaterContainer.AddChild(sandFloor)

	// --- Seaweed (ZIndex 3) ---
	for i := range numSeaweed {
		sw := willow.NewSprite("seaweed", willow.TextureRegion{})
		sw.SetScale(10+float64(i%3)*3, 80+float64(i)*15)
		sw.SetColor(willow.RGB(0.2, 0.8+float64(i)*0.03, 0.3))
		sw.SetPivot(0, 1)
		sw.SetPosition(60+float64(i)*120, screenH*0.65-2)
		sw.SetZIndex(3)
		underwaterContainer.AddChild(sw)
		d.seaweed[i] = sw
	}

	// --- Mid water mesh (ZIndex 5) ---
	midGrid := willow.NewDistortionGrid("mid-water", waterImg, totalCols, totalRows)
	midGrid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	midGrid.Node().SetColor(willow.RGBA(0.5, 0.7, 0.9, 0.3))
	midGrid.Node().SetZIndex(5)
	underwaterContainer.AddChild(midGrid.Node())
	d.midGrid = midGrid

	// --- Whelp (ZIndex 16) ---
	whelp := willow.NewSprite("whelp", willow.TextureRegion{})
	whelp.SetCustomImage(whelpImg)
	whelp.SetScale(2, 2)
	whelp.SetPivot(0.5, 0.5)
	whelp.SetPosition(screenW/2, screenH*0.45)
	whelp.SetZIndex(16)
	underwaterContainer.AddChild(whelp)
	d.whelp = whelp

	// --- Ambient bubbles at each seaweed base (ZIndex 15) ---
	for i := range numSeaweed {
		b := willow.NewParticleEmitter("bubbles", willow.EmitterConfig{
			MaxParticles: 30,
			EmitRate:     3 + float64(i%3),
			Lifetime:     willow.Range{Min: 2.0, Max: 4.5},
			Speed:        willow.Range{Min: 25, Max: 60},
			Angle:        willow.Range{Min: -math.Pi * 0.65, Max: -math.Pi * 0.35},
			StartScale:   willow.Range{Min: 2, Max: 5},
			EndScale:     willow.Range{Min: 1, Max: 2},
			StartAlpha:   willow.Range{Min: 0.5, Max: 0.7},
			EndAlpha:     willow.Range{Min: 0.0, Max: 0.1},
			Gravity:      willow.Vec2{X: 8, Y: -3},
			StartColor:   willow.RGB(0.7, 0.9, 1.0),
			EndColor:     willow.RGB(0.5, 0.8, 1.0),
			BlendMode:    willow.BlendAdd,
		})
		b.SetPosition(60+float64(i)*120, screenH*0.65-2)
		b.SetZIndex(15)
		b.Emitter.Start()
		underwaterContainer.AddChild(b)
	}

	// --- Caustic light layer (ZIndex 18) ---
	lightLayer := willow.NewLightLayer(screenW, screenH, 0.0)
	var lights [4]*willow.Light
	for i := range 4 {
		light := &willow.Light{
			X:         float64(screenW) * (0.15 + float64(i)*0.25),
			Y:         float64(screenH) * 0.4,
			Radius:    220,
			Intensity: 1.0,
			Color:     willow.RGB(0.4, 0.7, 1.0),
			Enabled:   true,
		}
		lightLayer.AddLight(light)
		lights[i] = light
	}
	lightLayer.Node().SetZIndex(18)
	underwaterContainer.AddChild(lightLayer.Node())
	d.lightLayer = lightLayer
	d.lights = lights

	// --- Near water mesh (ZIndex 20) ---
	nearGrid := willow.NewDistortionGrid("near-water", waterImg, totalCols, totalRows)
	nearGrid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	nearGrid.Node().SetColor(willow.RGBA(0.6, 0.8, 1.0, 0.5))
	nearGrid.Node().SetZIndex(20)
	underwaterContainer.AddChild(nearGrid.Node())
	d.nearGrid = nearGrid

	scene.Root.AddChild(underwaterContainer)

	// --- Burst emitter (ZIndex 55, click-triggered, on root so not clipped) ---
	burst := willow.NewParticleEmitter("burst", willow.EmitterConfig{
		MaxParticles: 150,
		EmitRate:     800,
		Lifetime:     willow.Range{Min: 0.4, Max: 1.2},
		Speed:        willow.Range{Min: 50, Max: 220},
		Angle:        willow.Range{Min: 0, Max: math.Pi * 2},
		StartScale:   willow.Range{Min: 3, Max: 8},
		EndScale:     willow.Range{Min: 0, Max: 2},
		StartAlpha:   willow.Range{Min: 0.9, Max: 1.0},
		EndAlpha:     willow.Range{Min: 0.0, Max: 0.0},
		Gravity:      willow.Vec2{X: 0, Y: -40},
		StartColor:   willow.RGB(0.8, 0.95, 1.0),
		EndColor:     willow.RGB(0.3, 0.6, 1.0),
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	burst.SetZIndex(55)
	scene.Root.AddChild(burst)
	d.burst = burst

	// Dummy ring container (kept so update code doesn't need changes).
	ringContainer := willow.NewContainer("ring")
	d.ringContainer = ringContainer

	// --- Two surface water layers (ZIndex 50, 52) ---
	makeInvertedMask := func() (*willow.Node, *willow.Node) {
		root := willow.NewContainer("surf-mask")
		rect := willow.NewPolygon("surf-mask-rect", []willow.Vec2{
			{X: 0, Y: 0}, {X: screenW, Y: 0},
			{X: screenW, Y: screenH}, {X: 0, Y: screenH},
		})
		rect.SetColor(willow.RGB(1, 1, 1))
		root.AddChild(rect)
		hole := willow.NewPolygon("surf-mask-hole", circlePoints(portholeRadius, portholeSegs))
		hole.SetColor(willow.RGB(1, 1, 1))
		hole.SetBlendMode(willow.BlendErase)
		hole.SetPosition(screenW/2, screenH/2)
		root.AddChild(hole)
		return root, hole
	}

	surf1Container := willow.NewContainer("surf1")
	surf1Container.SetZIndex(50)
	overlay1Grid := willow.NewDistortionGrid("overlay1", waterImg, totalCols, totalRows)
	overlay1Grid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	overlay1Grid.Node().SetColor(willow.RGB(0.75, 0.92, 1.0))
	overlay1Grid.Node().SetBlendMode(willow.BlendScreen)
	surf1Container.AddChild(overlay1Grid.Node())
	mask1, hole1 := makeInvertedMask()
	surf1Container.SetMask(mask1)
	scene.Root.AddChild(surf1Container)
	d.overlayGrid1 = overlay1Grid

	surf2Container := willow.NewContainer("surf2")
	surf2Container.SetZIndex(52)
	overlay2Grid := willow.NewDistortionGrid("overlay2", waterImg2, totalCols, totalRows)
	overlay2Grid.Node().SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	overlay2Grid.Node().SetColor(willow.RGBA(2, 2, 2, 0.6))
	overlay2Grid.Node().SetBlendMode(willow.BlendScreen)
	surf2Container.AddChild(overlay2Grid.Node())
	mask2, hole2 := makeInvertedMask()
	surf2Container.SetMask(mask2)
	scene.Root.AddChild(surf2Container)
	d.overlayGrid2 = overlay2Grid

	d.surfaceEraseChild = hole1
	d.surfaceEraseHole2 = hole2

	// --- Hint label (ZIndex 70) ---
	const hintText = "move cursor to explore  |  click for bubbles"
	hint := makeLabel(hintText)
	hint.SetPosition(float64(screenW)/2-float64(len(hintText)*6)/2, float64(screenH)-20)
	hint.SetZIndex(70)
	scene.Root.AddChild(hint)

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

// circlePoints generates polygon vertices for a filled circle.
func circlePoints(radius float64, segments int) []willow.Vec2 {
	pts := make([]willow.Vec2, segments)
	for i := range segments {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		pts[i] = willow.Vec2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		}
	}
	return pts
}

// rippleCircle generates circle vertices with a time-varying displacement
// that makes the porthole edge wobble organically.
func rippleCircle(radius float64, segments int, t float64) []willow.Vec2 {
	pts := make([]willow.Vec2, segments)
	for i := range segments {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		r := radius +
			4.0*math.Sin(angle*5+t*2.0) +
			2.5*math.Sin(angle*7-t*2.7+1.0) +
			1.5*math.Sin(angle*3+t*1.5+2.3)
		pts[i] = willow.Vec2{
			X: math.Cos(angle) * r,
			Y: math.Sin(angle) * r,
		}
	}
	return pts
}

// makeLabel renders s into a small sprite using the Ebitengine debug font.
func makeLabel(s string) *willow.Node {
	img := ebiten.NewImage(len(s)*6+4, 16)
	ebitenutil.DebugPrint(img, s)
	n := willow.NewSprite("label", willow.TextureRegion{})
	n.SetCustomImage(img)
	return n
}
