// 2.5D raycaster — GPU-accelerated variant of the 3d-raycaster demo.
//
// Instead of CPU pixel-fill + WritePixels, this version builds wall geometry
// as screen-space quads and submits them in a single Willow mesh node
// (one DrawTriangles call). A Kage shader applied as a CustomShaderFilter
// handles distance fog on the GPU: wall distance is encoded into vertex
// alpha, and the shader un-premultiplies to recover the base colour and
// blends toward a fog colour. Ceiling and floor are mesh quads with
// vertex-color gradients, so the entire 3D viewport is rendered through
// DrawTriangles with zero pixel-buffer uploads.
//
// Controls:
//
//	W / Up    – move forward
//	S / Down  – move back
//	A / D     – strafe
//	Q / Left  – turn left
//	E / Right – turn right
//	Space     – fire (weapon recoil + damage flash)
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tanema/gween/ease"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	screenW   = 800
	screenH   = 600
	mapW      = 20
	mapH      = 20
	moveSpeed = 0.07
	rotSpeed  = 0.045

	maxCols = screenW // one quad per column worst case
)

// ── Map ──────────────────────────────────────────────────────────────────────

var worldMap = [mapH][mapW]int{
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 2, 2, 0, 0, 1, 0, 0, 0, 3, 3, 3, 3, 0, 0, 0, 0, 0, 1},
	{1, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 3, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 4, 4, 4, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 4, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 4, 0, 0, 0, 0, 1},
	{1, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 3, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0, 1},
	{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

// ── Wall colours ─────────────────────────────────────────────────────────────

type rgb struct{ r, g, b float32 }

var wallColors = [5][2]rgb{
	{}, // 0: unused
	{{110.0 / 255, 110.0 / 255, 110.0 / 255}, {75.0 / 255, 75.0 / 255, 75.0 / 255}}, // 1: stone
	{{130.0 / 255, 70.0 / 255, 50.0 / 255}, {90.0 / 255, 50.0 / 255, 35.0 / 255}},   // 2: brick
	{{50.0 / 255, 80.0 / 255, 50.0 / 255}, {35.0 / 255, 55.0 / 255, 35.0 / 255}},    // 3: moss
	{{150.0 / 255, 140.0 / 255, 80.0 / 255}, {100.0 / 255, 95.0 / 255, 55.0 / 255}}, // 4: iron
}

// ── Fog shader (Kage) ────────────────────────────────────────────────────────
//
// Wall distance is encoded in vertex alpha: 1.0 = close, low values = far.
// Because DrawTriangles writes premultiplied colour, the shader
// un-premultiplies to recover the base wall colour, then blends toward the
// fog colour proportionally.

const fogShaderSrc = `
//kage:unit pixels
package main

func Fragment(dst vec4, src vec2, clr vec4) vec4 {
	c := imageSrc0At(src)
	if c.a < 0.001 {
		return vec4(0)
	}

	// Un-premultiply to recover the original wall colour.
	wall := c.rgb / c.a

	// Alpha encodes closeness: 1 = nearest, ~0.08 = farthest.
	// Invert to get fog strength.
	fog := 1.0 - c.a

	// Blend toward black (dungeon fog).
	fogged := wall * (1.0 - fog * 0.92)
	return vec4(fogged, 1)
}
`

// ── Game struct ───────────────────────────────────────────────────────────────

type game struct {
	posX, posY     float64
	dirX, dirY     float64
	planeX, planeY float64

	health, ammo int
	fired        bool
	kickTween    *willow.TweenGroup

	scene       *willow.Scene
	bgMesh      *willow.Node // ceiling + floor gradient
	wallMesh    *willow.Node // all wall columns, single DrawTriangles
	weaponNode  *willow.Node
	healthText  *willow.Node
	ammoText    *willow.Node
	damageFlash *willow.Node

	// Pre-allocated vertex/index buffers for wall mesh (reused each frame).
	wallVerts []ebiten.Vertex
	wallInds  []uint16

	// Pre-built minimap tiles (static geometry never changes).
	minimapBG  *ebiten.Image // static tile grid, built once
	minimapDot *ebiten.Image // player dot, built once
}

// ── Constructor ───────────────────────────────────────────────────────────────

func newGame() *game {
	g := &game{
		posX:   1.5,
		posY:   1.5,
		dirX:   1.0,
		dirY:   0.0,
		planeX: 0.0,
		planeY: 0.66,
		health: 100,
		ammo:   30,

		wallVerts: make([]ebiten.Vertex, 0, maxCols*4),
		wallInds:  make([]uint16, 0, maxCols*6),
	}

	g.scene = willow.NewScene()
	g.scene.ClearColor = willow.RGB(0, 0, 0)

	// ── Background mesh: ceiling + floor gradients ───────────────────────
	g.bgMesh = buildBackgroundMesh()
	g.scene.Root.AddChild(g.bgMesh)

	// ── Wall mesh: rebuilt each frame ────────────────────────────────────
	g.wallMesh = willow.NewMesh("walls", willow.WhitePixel, nil, nil)
	g.scene.Root.AddChild(g.wallMesh)

	// Compile and attach the fog shader as a post-process filter.
	fogShader, err := ebiten.NewShader([]byte(fogShaderSrc))
	if err != nil {
		log.Fatalf("compile fog shader: %v", err)
	}
	g.wallMesh.Filters = []any{willow.NewCustomShaderFilter(fogShader, 0)}

	// ── HUD (identical to 3d-raycaster) ──────────────────────────────────
	g.damageFlash = willow.NewSprite("flash", willow.TextureRegion{})
	g.damageFlash.SetSize(screenW, screenH)
	g.damageFlash.SetColor(willow.RGBA(1, 0, 0, 0.7))
	g.damageFlash.SetAlpha(0)
	g.scene.Root.AddChild(g.damageFlash)

	gunW, gunH := 120.0, 90.0
	g.weaponNode = willow.NewSprite("gun", willow.TextureRegion{})
	g.weaponNode.SetSize(gunW, gunH)
	g.weaponNode.SetColor(willow.RGB(0.4, 0.4, 0.4))
	g.weaponNode.SetPosition(screenW/2-gunW/2, screenH-gunH)
	g.scene.Root.AddChild(g.weaponNode)

	chH := willow.NewSprite("ch-h", willow.TextureRegion{})
	chH.SetSize(14, 2)
	chH.SetColor(willow.RGBA(1, 1, 1, 0.7))
	chH.SetPosition(screenW/2-7, screenH/2-1)
	g.scene.Root.AddChild(chH)

	chV := willow.NewSprite("ch-v", willow.TextureRegion{})
	chV.SetSize(2, 14)
	chV.SetColor(willow.RGBA(1, 1, 1, 0.7))
	chV.SetPosition(screenW/2-1, screenH/2-7)
	g.scene.Root.AddChild(chV)

	hudBar := willow.NewSprite("hud-bar", willow.TextureRegion{})
	hudBar.SetSize(screenW, 32)
	hudBar.SetColor(willow.RGBA(0, 0, 0, 0.75))
	hudBar.SetPosition(0, screenH-32)
	g.scene.Root.AddChild(hudBar)

	g.healthText = willow.NewText("health", "HP: 100", nil)
	g.healthText.SetFontSize(18)
	g.healthText.SetTextColor(willow.RGB(0.2, 1, 0.3))
	g.healthText.SetPosition(12, screenH-26)
	g.scene.Root.AddChild(g.healthText)

	g.ammoText = willow.NewText("ammo", "AMMO: 30", nil)
	g.ammoText.SetFontSize(18)
	g.ammoText.SetTextColor(willow.RGB(1, 0.85, 0.2))
	g.ammoText.SetPosition(screenW-110, screenH-26)
	g.scene.Root.AddChild(g.ammoText)

	g.buildMinimapImages()

	g.scene.SetUpdateFunc(g.update)

	g.scene.SetPostDrawFunc(func(screen *ebiten.Image) {
		ebitenutil.DebugPrintAt(screen,
			"W/S: move   A/D: strafe   Q/E or ←/→: turn   SPACE: fire",
			screenW-380, 8)
		g.drawMinimap(screen)
	})

	return g
}

// ── Background mesh ──────────────────────────────────────────────────────────
//
// Two quads: ceiling (top of screen → horizon) and floor (horizon → bottom).
// Vertex colours create a smooth gradient without any pixel-buffer work.

func buildBackgroundMesh() *willow.Node {
	half := float32(screenH / 2)

	// Ceiling: darker at top, lighter at horizon.
	ceilTop := rgb{15.0 / 255, 15.0 / 255, 18.0 / 255}
	ceilBot := rgb{40.0 / 255, 40.0 / 255, 44.0 / 255}

	// Floor: dark at horizon, slightly lighter (reddish tint) at bottom.
	floorTop := rgb{20.0 / 255, 20.0 / 255, 20.0 / 255}
	floorBot := rgb{36.0 / 255, 35.0 / 255, 35.0 / 255}

	verts := []ebiten.Vertex{
		// Ceiling quad (indices 0-3)
		{DstX: 0, DstY: 0, SrcX: 0.5, SrcY: 0.5, ColorR: ceilTop.r, ColorG: ceilTop.g, ColorB: ceilTop.b, ColorA: 1},
		{DstX: screenW, DstY: 0, SrcX: 0.5, SrcY: 0.5, ColorR: ceilTop.r, ColorG: ceilTop.g, ColorB: ceilTop.b, ColorA: 1},
		{DstX: screenW, DstY: half, SrcX: 0.5, SrcY: 0.5, ColorR: ceilBot.r, ColorG: ceilBot.g, ColorB: ceilBot.b, ColorA: 1},
		{DstX: 0, DstY: half, SrcX: 0.5, SrcY: 0.5, ColorR: ceilBot.r, ColorG: ceilBot.g, ColorB: ceilBot.b, ColorA: 1},
		// Floor quad (indices 4-7)
		{DstX: 0, DstY: half, SrcX: 0.5, SrcY: 0.5, ColorR: floorTop.r, ColorG: floorTop.g, ColorB: floorTop.b, ColorA: 1},
		{DstX: screenW, DstY: half, SrcX: 0.5, SrcY: 0.5, ColorR: floorTop.r, ColorG: floorTop.g, ColorB: floorTop.b, ColorA: 1},
		{DstX: screenW, DstY: screenH, SrcX: 0.5, SrcY: 0.5, ColorR: floorBot.r, ColorG: floorBot.g, ColorB: floorBot.b, ColorA: 1},
		{DstX: 0, DstY: screenH, SrcX: 0.5, SrcY: 0.5, ColorR: floorBot.r, ColorG: floorBot.g, ColorB: floorBot.b, ColorA: 1},
	}
	inds := []uint16{
		0, 1, 2, 0, 2, 3, // ceiling
		4, 5, 6, 4, 6, 7, // floor
	}
	return willow.NewMesh("background", willow.WhitePixel, verts, inds)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (g *game) update() error {
	g.handleInput()
	g.checkWeaponReturn()
	g.buildWalls()
	g.updateHUD()
	return nil
}

func (g *game) checkWeaponReturn() {
	if g.kickTween != nil && g.kickTween.Done {
		g.kickTween = nil
		restY := screenH - 90.0
		willow.TweenPosition(g.weaponNode,
			g.weaponNode.X(), restY,
			willow.TweenConfig{Duration: 0.15, Ease: ease.OutBounce})
	}
}

func (g *game) handleInput() {
	g.fired = false

	if g.scene.IsKeyPressed(ebiten.KeyLeft) || g.scene.IsKeyPressed(ebiten.KeyQ) {
		g.rotate(-rotSpeed)
	}
	if g.scene.IsKeyPressed(ebiten.KeyRight) || g.scene.IsKeyPressed(ebiten.KeyE) {
		g.rotate(rotSpeed)
	}
	if g.scene.IsKeyPressed(ebiten.KeyW) || g.scene.IsKeyPressed(ebiten.KeyArrowUp) {
		g.tryMove(g.dirX, g.dirY, moveSpeed)
	}
	if g.scene.IsKeyPressed(ebiten.KeyS) || g.scene.IsKeyPressed(ebiten.KeyArrowDown) {
		g.tryMove(g.dirX, g.dirY, -moveSpeed)
	}
	if g.scene.IsKeyPressed(ebiten.KeyA) {
		g.tryMove(-g.dirY, g.dirX, moveSpeed*0.75)
	}
	if g.scene.IsKeyPressed(ebiten.KeyD) {
		g.tryMove(g.dirY, -g.dirX, moveSpeed*0.75)
	}

	if g.scene.IsKeyPressed(ebiten.KeySpace) && !g.fired {
		g.fired = true
		if g.ammo > 0 {
			g.ammo--
			g.triggerShot()
		}
	}
}

func (g *game) triggerShot() {
	restY := screenH - 90.0
	g.kickTween = willow.TweenPosition(g.weaponNode,
		g.weaponNode.X(), restY+30,
		willow.TweenConfig{Duration: 0.07, Ease: ease.OutQuad})

	g.damageFlash.SetAlpha(0.25)
	willow.TweenAlpha(g.damageFlash, 0,
		willow.TweenConfig{Duration: 0.12, Ease: ease.OutQuad})
}

func (g *game) updateHUD() {
	g.healthText.TextBlock.Content = fmt.Sprintf("HP: %d", g.health)
	g.healthText.TextBlock.Invalidate()
	g.ammoText.TextBlock.Content = fmt.Sprintf("AMMO: %d", g.ammo)
	g.ammoText.TextBlock.Invalidate()
}

// ── Movement & rotation ───────────────────────────────────────────────────────

func (g *game) tryMove(mx, my, speed float64) {
	nx := g.posX + mx*speed
	ny := g.posY + my*speed

	const pad = 0.25

	checkX := nx
	if mx >= 0 {
		checkX += pad
	} else {
		checkX -= pad
	}
	if inBounds(int(checkX), int(g.posY)) && worldMap[int(g.posY)][int(checkX)] == 0 {
		g.posX = nx
	}

	checkY := ny
	if my >= 0 {
		checkY += pad
	} else {
		checkY -= pad
	}
	if inBounds(int(g.posX), int(checkY)) && worldMap[int(checkY)][int(g.posX)] == 0 {
		g.posY = ny
	}
}

func (g *game) rotate(angle float64) {
	cos, sin := math.Cos(angle), math.Sin(angle)
	g.dirX, g.dirY = g.dirX*cos-g.dirY*sin, g.dirX*sin+g.dirY*cos
	g.planeX, g.planeY = g.planeX*cos-g.planeY*sin, g.planeX*sin+g.planeY*cos
}

func inBounds(x, y int) bool {
	return x >= 0 && x < mapW && y >= 0 && y < mapH
}

// ── Wall mesh builder ────────────────────────────────────────────────────────
//
// DDA raycasting is identical to the original, but instead of writing pixels
// into a byte buffer, each visible column becomes a screen-space quad (4
// vertices, 6 indices). All columns share a single vertex/index buffer,
// submitted as one DrawTriangles call through the Willow mesh node.
//
// Distance fog is baked into vertex RGB (same formula as the original).

func (g *game) buildWalls() {
	verts := g.wallVerts[:0]
	inds := g.wallInds[:0]

	half := float64(screenH) / 2

	for col := 0; col < screenW; col++ {
		camX := 2.0*float64(col)/float64(screenW) - 1.0
		rayDirX := g.dirX + g.planeX*camX
		rayDirY := g.dirY + g.planeY*camX

		mx, my := int(g.posX), int(g.posY)

		var ddx, ddy float64
		if rayDirX == 0 {
			ddx = 1e18
		} else {
			ddx = math.Abs(1.0 / rayDirX)
		}
		if rayDirY == 0 {
			ddy = 1e18
		} else {
			ddy = math.Abs(1.0 / rayDirY)
		}

		var stepX, stepY int
		var sdx, sdy float64
		if rayDirX < 0 {
			stepX, sdx = -1, (g.posX-float64(mx))*ddx
		} else {
			stepX, sdx = 1, (float64(mx)+1.0-g.posX)*ddx
		}
		if rayDirY < 0 {
			stepY, sdy = -1, (g.posY-float64(my))*ddy
		} else {
			stepY, sdy = 1, (float64(my)+1.0-g.posY)*ddy
		}

		hit, side := 0, 0
		for hit == 0 {
			if sdx < sdy {
				sdx += ddx
				mx += stepX
				side = 0
			} else {
				sdy += ddy
				my += stepY
				side = 1
			}
			if !inBounds(mx, my) {
				break
			}
			hit = worldMap[my][mx]
		}
		if hit == 0 {
			continue
		}

		var dist float64
		if side == 0 {
			dist = sdx - ddx
		} else {
			dist = sdy - ddy
		}
		if dist < 0.01 {
			dist = 0.01
		}

		lineH := screenH / dist
		y0 := half - lineH/2
		y1 := half + lineH/2

		// Wall colour (face-shaded, no fog). Distance encoded in alpha
		// for the fog shader: 1.0 = close, low = far.
		wt := hit
		if wt >= len(wallColors) {
			wt = 1
		}
		c := wallColors[wt][side]
		alpha := float32(1.0 - math.Min(dist/16.0, 0.92))

		x := float32(col)
		fy0 := float32(y0)
		fy1 := float32(y1)

		base := uint16(len(verts))
		verts = append(verts,
			ebiten.Vertex{DstX: x, DstY: fy0, SrcX: 0.5, SrcY: 0.5, ColorR: c.r, ColorG: c.g, ColorB: c.b, ColorA: alpha},
			ebiten.Vertex{DstX: x + 1, DstY: fy0, SrcX: 0.5, SrcY: 0.5, ColorR: c.r, ColorG: c.g, ColorB: c.b, ColorA: alpha},
			ebiten.Vertex{DstX: x + 1, DstY: fy1, SrcX: 0.5, SrcY: 0.5, ColorR: c.r, ColorG: c.g, ColorB: c.b, ColorA: alpha},
			ebiten.Vertex{DstX: x, DstY: fy1, SrcX: 0.5, SrcY: 0.5, ColorR: c.r, ColorG: c.g, ColorB: c.b, ColorA: alpha},
		)
		inds = append(inds, base, base+1, base+2, base, base+2, base+3)
	}

	g.wallVerts = verts
	g.wallInds = inds
	g.wallMesh.SetMeshVertices(verts)
	g.wallMesh.SetMeshIndices(inds)
}

// ── Minimap ───────────────────────────────────────────────────────────────────

const (
	mmCell   = 7
	mmOffX   = screenW - mapW*mmCell - 8
	mmOffY   = 28
	mmRadius = 2
)

// buildMinimapImages pre-renders the static tile grid and player dot once.
func (g *game) buildMinimapImages() {
	w := mapW * mmCell
	h := mapH * mmCell
	g.minimapBG = ebiten.NewImage(w, h)

	tile := ebiten.NewImage(mmCell-1, mmCell-1)
	for my := 0; my < mapH; my++ {
		for mx := 0; mx < mapW; mx++ {
			v := worldMap[my][mx]
			if v > 0 {
				c := wallColors[v][0]
				tile.Fill(color.RGBA{
					R: uint8(c.r * 255), G: uint8(c.g * 255), B: uint8(c.b * 255), A: 217,
				})
			} else {
				tile.Fill(color.RGBA{R: 25, G: 25, B: 25, A: 178})
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(mx*mmCell), float64(my*mmCell))
			g.minimapBG.DrawImage(tile, op)
		}
	}

	g.minimapDot = ebiten.NewImage(mmRadius*2+1, mmRadius*2+1)
	g.minimapDot.Fill(color.RGBA{R: 255, G: 255, B: 0, A: 255})
}

func (g *game) drawMinimap(screen *ebiten.Image) {
	// Blit the pre-built static grid.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(mmOffX, mmOffY)
	screen.DrawImage(g.minimapBG, op)

	// Player dot (only this moves each frame).
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(
		float64(mmOffX)+g.posX*mmCell-mmRadius,
		float64(mmOffY)+g.posY*mmCell-mmRadius,
	)
	screen.DrawImage(g.minimapDot, op2)

	// Direction indicator.
	ex := float64(mmOffX) + (g.posX+g.dirX*2)*mmCell
	ey := float64(mmOffY) + (g.posY+g.dirY*2)*mmCell
	ebitenutil.DrawLine(screen,
		float64(mmOffX)+g.posX*mmCell, float64(mmOffY)+g.posY*mmCell,
		ex, ey,
		color.RGBA{R: 255, G: 255, B: 0, A: 255},
	)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	g := newGame()

	if err := willow.Run(g.scene, willow.RunConfig{
		Title:        "2.5D Raycaster — DrawTriangles + Fog Shader",
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}
