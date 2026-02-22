// Underwater demonstrates a layered underwater scene revealed through a circular
// porthole mask that follows the cursor. The surface is an opaque DistortionGrid
// with animated waves; beneath it, three additional water meshes at varying depth,
// seaweed, a swimming whelp, rising bubbles, caustic lighting, and a sandy floor
// create an atmospheric underwater world. Click to spawn a bubble burst.
package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/phanxgames/willow"
)

//go:embed tileset.png
var tilesetPNG []byte

//go:embed whelp.png
var whelpPNG []byte

const (
	windowTitle = "Willow — Underwater"
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
	frame int

	// Porthole tracking — move together with the cursor
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

	// Thumbnail capture
	screenshotDone bool
}

func (d *demo) update() error {
	dt := 1.0 / float64(ebiten.TPS())
	d.time += dt
	t := d.time
	d.frame++

	// Capture thumbnail at frame 60 for docs site.
	if d.frame == 60 && !d.screenshotDone {
		d.scene.ScreenshotDir = "docs/demos/underwater"
		d.scene.Screenshot("thumbnail")
		d.screenshotDone = true
	}

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
	// --- Load embedded assets ---
	tilesetImg, err := png.Decode(bytes.NewReader(tilesetPNG))
	if err != nil {
		log.Fatalf("decode tileset: %v", err)
	}
	tileset := ebiten.NewImageFromImage(tilesetImg)

	whelpRaw, err := png.Decode(bytes.NewReader(whelpPNG))
	if err != nil {
		log.Fatalf("decode whelp: %v", err)
	}
	whelpImg := ebiten.NewImageFromImage(whelpRaw)

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
	scene.ClearColor = willow.Color{R: 0.05, G: 0.12, B: 0.28, A: 1}

	cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	d := &demo{scene: scene}

	// --- Underwater container (masked by porthole circle) ---
	underwaterContainer := willow.NewContainer("underwater")

	maskRoot := willow.NewContainer("mask-root")
	maskChild := willow.NewPolygon("porthole", circlePoints(portholeRadius, portholeSegs))
	maskChild.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
	maskChild.SetPosition(screenW/2, screenH/2)
	maskRoot.AddChild(maskChild)
	underwaterContainer.SetMask(maskRoot)
	d.maskChild = maskChild

	// --- Deep water mesh (ZIndex 0) ---
	deepGrid, deepNode := willow.NewDistortionGrid("deep-water", waterImg, totalCols, totalRows)
	deepNode.SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	deepNode.Color = willow.Color{R: 0.35, G: 0.55, B: 0.75, A: 1.0}
	deepNode.ZIndex = 0
	underwaterContainer.AddChild(deepNode)
	d.deepGrid = deepGrid

	// --- Sandy floor (ZIndex 2) ---
	sandFloor := willow.NewSprite("sand-floor", willow.TextureRegion{})
	sandFloor.SetCustomImage(sandImg)
	sandFloor.Color = willow.Color{R: 0.9, G: 0.75, B: 0.55, A: 1.0}
	sandFloor.SetPosition(0, screenH*0.65)
	sandFloor.ZIndex = 2
	underwaterContainer.AddChild(sandFloor)

	// --- Seaweed (ZIndex 3) ---
	for i := range numSeaweed {
		sw := willow.NewSprite("seaweed", willow.TextureRegion{})
		sw.SetScale(10+float64(i%3)*3, 80+float64(i)*15)
		sw.Color = willow.Color{R: 0.2, G: 0.8 + float64(i)*0.03, B: 0.3, A: 1.0}
		sw.SetPivot(0, 1)
		sw.SetPosition(60+float64(i)*120, screenH*0.65-2)
		sw.ZIndex = 3
		underwaterContainer.AddChild(sw)
		d.seaweed[i] = sw
	}

	// --- Mid water mesh (ZIndex 5) ---
	midGrid, midNode := willow.NewDistortionGrid("mid-water", waterImg, totalCols, totalRows)
	midNode.SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	midNode.Color = willow.Color{R: 0.5, G: 0.7, B: 0.9, A: 0.3}
	midNode.ZIndex = 5
	underwaterContainer.AddChild(midNode)
	d.midGrid = midGrid

	// --- Whelp (ZIndex 16) ---
	whelp := willow.NewSprite("whelp", willow.TextureRegion{})
	whelp.SetCustomImage(whelpImg)
	whelp.SetScale(2, 2)
	whelp.SetPivot(0.5, 0.5)
	whelp.SetPosition(screenW/2, screenH*0.45)
	whelp.ZIndex = 16
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
			StartColor:   willow.Color{R: 0.7, G: 0.9, B: 1.0, A: 1},
			EndColor:     willow.Color{R: 0.5, G: 0.8, B: 1.0, A: 1},
			BlendMode:    willow.BlendAdd,
		})
		b.SetPosition(60+float64(i)*120, screenH*0.65-2)
		b.ZIndex = 15
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
			Color:     willow.Color{R: 0.4, G: 0.7, B: 1.0, A: 1},
			Enabled:   true,
		}
		lightLayer.AddLight(light)
		lights[i] = light
	}
	lightLayer.Node().ZIndex = 18
	underwaterContainer.AddChild(lightLayer.Node())
	d.lightLayer = lightLayer
	d.lights = lights

	// --- Near water mesh (ZIndex 20) ---
	nearGrid, nearNode := willow.NewDistortionGrid("near-water", waterImg, totalCols, totalRows)
	nearNode.SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	nearNode.Color = willow.Color{R: 0.6, G: 0.8, B: 1.0, A: 0.5}
	nearNode.ZIndex = 20
	underwaterContainer.AddChild(nearNode)
	d.nearGrid = nearGrid

	scene.Root().AddChild(underwaterContainer)

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
		StartColor:   willow.Color{R: 0.8, G: 0.95, B: 1.0, A: 1},
		EndColor:     willow.Color{R: 0.3, G: 0.6, B: 1.0, A: 1},
		BlendMode:    willow.BlendAdd,
		WorldSpace:   true,
	})
	burst.ZIndex = 55
	scene.Root().AddChild(burst)
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
		rect.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
		root.AddChild(rect)
		hole := willow.NewPolygon("surf-mask-hole", circlePoints(portholeRadius, portholeSegs))
		hole.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
		hole.BlendMode = willow.BlendErase
		hole.SetPosition(screenW/2, screenH/2)
		root.AddChild(hole)
		return root, hole
	}

	surf1Container := willow.NewContainer("surf1")
	surf1Container.ZIndex = 50
	overlay1Grid, overlay1Node := willow.NewDistortionGrid("overlay1", waterImg, totalCols, totalRows)
	overlay1Node.SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	overlay1Node.Color = willow.Color{R: 0.75, G: 0.92, B: 1.0, A: 1}
	overlay1Node.BlendMode = willow.BlendScreen
	surf1Container.AddChild(overlay1Node)
	mask1, hole1 := makeInvertedMask()
	surf1Container.SetMask(mask1)
	scene.Root().AddChild(surf1Container)
	d.overlayGrid1 = overlay1Grid

	surf2Container := willow.NewContainer("surf2")
	surf2Container.ZIndex = 52
	overlay2Grid, overlay2Node := willow.NewDistortionGrid("overlay2", waterImg2, totalCols, totalRows)
	overlay2Node.SetPosition(float64(-bufTiles*tileSize), float64(-bufTiles*tileSize))
	overlay2Node.Color = willow.Color{R: 2, G: 2, B: 2, A: 0.6}
	overlay2Node.BlendMode = willow.BlendScreen
	surf2Container.AddChild(overlay2Node)
	mask2, hole2 := makeInvertedMask()
	surf2Container.SetMask(mask2)
	scene.Root().AddChild(surf2Container)
	d.overlayGrid2 = overlay2Grid

	d.surfaceEraseChild = hole1
	d.surfaceEraseHole2 = hole2

	// --- Hint label (ZIndex 70) ---
	const hintText = "move cursor to explore  |  click for bubbles"
	hint := makeLabel(hintText)
	hint.SetPosition(float64(screenW)/2-float64(len(hintText)*6)/2, float64(screenH)-20)
	hint.ZIndex = 70
	scene.Root().AddChild(hint)

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
