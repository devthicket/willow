// Wolfenstein-style dungeon crawler testbed for the Willow engine.
//
// The 3D view is a classic DDA raycaster that writes directly into an
// ebiten.Image pixel buffer, displayed via SetCustomImage on a full-screen
// sprite. Everything else – the HUD, weapon sprite, damage flash, and text –
// are Willow nodes, demonstrating that the engine handles the game layer while
// the raycaster owns the pixel work.
//
// Controls:
//   W / Up    – move forward
//   S / Down  – move back
//   A / D     – strafe
//   Q / Left  – turn left
//   E / Right – turn right
//   Space     – fire (weapon recoil + damage flash)
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/devthicket/willow"
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
)

// ── Map ──────────────────────────────────────────────────────────────────────
//
// 0 = open floor
// 1 = stone wall (grey)
// 2 = brick wall (red-brown)
// 3 = dungeon wall (green-black)
// 4 = iron door frame (yellow-grey)

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
// Each wall type has two faces: [0] = N/S face (brighter), [1] = E/W face (darker).

type rgb struct{ r, g, b uint8 }

var wallColors = [5][2]rgb{
	{},                                         // 0: unused
	{{110, 110, 110}, {75, 75, 75}},            // 1: stone grey
	{{130, 70, 50}, {90, 50, 35}},              // 2: brick red-brown
	{{50, 80, 50}, {35, 55, 35}},               // 3: dungeon moss-green
	{{150, 140, 80}, {100, 95, 55}},            // 4: iron door frame
}

// ── Game struct ───────────────────────────────────────────────────────────────

type game struct {
	// Player state
	posX, posY     float64 // world position (in tile units)
	dirX, dirY     float64 // unit direction vector
	planeX, planeY float64 // camera plane (perpendicular, length ≈ tan(FOV/2))

	// Health / ammo
	health, ammo int
	fired        bool // did we fire this frame?
	kickTween    *willow.TweenGroup // tracks weapon recoil so we can chain the return

	// Pixel buffer for the raycaster view
	pixels   []byte
	framebuf *ebiten.Image

	// Willow scene and nodes
	scene       *willow.Scene
	viewport    *willow.Node // full-screen sprite showing framebuf
	weaponNode  *willow.Node // coloured rectangle standing in for a gun sprite
	healthText  *willow.Node
	ammoText    *willow.Node
	damageFlash *willow.Node // full-screen red overlay, alpha-tweened on hit

	// Autotest autopilot
	autopilot     bool
	autopilotTick int
}

// ── Constructor ───────────────────────────────────────────────────────────────

func newGame() *game {
	g := &game{
		posX:   1.5,
		posY:   1.5,
		dirX:   1.0,
		dirY:   0.0,
		planeX: 0.0,
		planeY: 0.66, // ~66° horizontal FOV
		health: 100,
		ammo:   30,
		pixels: make([]byte, screenW*screenH*4),
	}

	// --- pixel buffer ---------------------------------------------------------
	g.framebuf = ebiten.NewImage(screenW, screenH)

	// --- scene ----------------------------------------------------------------
	g.scene = willow.NewScene()
	g.scene.ClearColor = willow.RGB(0, 0, 0)

	// Full-screen viewport sprite (z = 0, rendered first)
	g.viewport = willow.NewSprite("viewport", willow.TextureRegion{})
	g.viewport.SetCustomImage(g.framebuf)
	g.scene.Root.AddChild(g.viewport)

	// Damage flash overlay (rendered above viewport, below weapon/text)
	g.damageFlash = willow.NewSprite("flash", willow.TextureRegion{})
	g.damageFlash.SetSize(screenW, screenH)
	g.damageFlash.SetColor(willow.RGBA(1, 0, 0, 0.7))
	g.damageFlash.SetAlpha(0)
	g.scene.Root.AddChild(g.damageFlash)

	// Weapon placeholder: a simple dark grey rectangle at screen centre-bottom.
	// In a real game this would be a sprite sheet animated via SetTextureRegion.
	gunW, gunH := 120.0, 90.0
	g.weaponNode = willow.NewSprite("gun", willow.TextureRegion{})
	g.weaponNode.SetSize(gunW, gunH)
	g.weaponNode.SetColor(willow.RGB(0.4, 0.4, 0.4))
	g.weaponNode.SetPosition(screenW/2-gunW/2, screenH-gunH)
	g.scene.Root.AddChild(g.weaponNode)

	// Crosshair (two tiny overlapping sprites)
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

	// HUD bar at bottom
	hudBar := willow.NewSprite("hud-bar", willow.TextureRegion{})
	hudBar.SetSize(screenW, 32)
	hudBar.SetColor(willow.RGBA(0, 0, 0, 0.75))
	hudBar.SetPosition(0, screenH-32)
	g.scene.Root.AddChild(hudBar)

	// Health and ammo text – these are willow.Node objects that live in the scene
	// graph and are tweened / updated just like any other node.
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

	// Per-frame update
	g.scene.SetUpdateFunc(g.update)

	// Debug / controls overlay drawn after the scene via SetPostDrawFunc.
	g.scene.SetPostDrawFunc(func(screen *ebiten.Image) {
		ebitenutil.DebugPrint(screen,
			"W/S: move   A/D: strafe   Q/E or ←/→: turn   SPACE: fire")
		g.drawMinimap(screen)
	})

	return g
}

// ── Update ────────────────────────────────────────────────────────────────────

func (g *game) update() error {
	if g.autopilot {
		g.autopilotTick++
		g.rotate(rotSpeed * 0.3)
		g.tryMove(g.dirX, g.dirY, moveSpeed*0.5)
		// Fire occasionally
		if g.autopilotTick%40 == 0 && g.ammo > 0 {
			g.ammo--
			g.triggerShot()
		}
		g.checkWeaponReturn()
		g.renderFrame()
		g.updateHUD()
		if g.autopilotTick >= 180 {
			g.scene.StopGif()
			return ebiten.Termination
		}
		return nil
	}
	g.handleInput()
	g.checkWeaponReturn()
	g.renderFrame()
	g.updateHUD()
	return nil
}

// checkWeaponReturn starts the weapon return tween once the kick tween finishes.
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

	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.rotate(-rotSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyE) {
		g.rotate(rotSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.tryMove(g.dirX, g.dirY, moveSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.tryMove(g.dirX, g.dirY, -moveSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		// strafe: perpendicular = (-dirY, dirX)
		g.tryMove(-g.dirY, g.dirX, moveSpeed*0.75)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.tryMove(g.dirY, -g.dirX, moveSpeed*0.75)
	}

	// Fire: trigger recoil tween on weapon node + flash on damage overlay.
	if ebiten.IsKeyPressed(ebiten.KeySpace) && !g.fired {
		g.fired = true
		if g.ammo > 0 {
			g.ammo--
			g.triggerShot()
		}
	}
}

// triggerShot fires weapon animations using Willow tweens.
// TweenConfig has no OnComplete, so we track the kick tween and start the
// return tween in update() once it finishes.
func (g *game) triggerShot() {
	restY := screenH - 90.0
	// Kick down.
	g.kickTween = willow.TweenPosition(g.weaponNode,
		g.weaponNode.X(), restY+30,
		willow.TweenConfig{Duration: 0.07, Ease: ease.OutQuad})

	// Muzzle flash: briefly show the damage overlay.
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

	// Axis-separated collision with a small padding so the player
	// doesn't clip inside wall geometry.
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

// ── Raycaster ─────────────────────────────────────────────────────────────────

func (g *game) renderFrame() {
	px := g.pixels
	half := screenH / 2

	// Ceiling (dark stone overhead) and floor (darker ground).
	for y := 0; y < screenH; y++ {
		var r, gr, b uint8
		if y < half {
			// Ceiling: subtle gradient from lighter at horizon to dark overhead.
			t := float64(half-y) / float64(half)
			v := uint8(40 - uint8(t*25))
			r, gr, b = v, v, uint8(float64(v)*1.1)
		} else {
			// Floor: very dark with a faint reddish tint.
			t := float64(y-half) / float64(half)
			v := uint8(20 + uint8(t*15))
			r, gr, b = uint8(float64(v)*1.05), v, v
		}
		for x := 0; x < screenW; x++ {
			i := (y*screenW + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = r, gr, b, 255
		}
	}

	// One ray per screen column.
	for col := 0; col < screenW; col++ {
		// Camera-space x: −1 (left edge) … +1 (right edge).
		camX := 2.0*float64(col)/float64(screenW) - 1.0
		rayDirX := g.dirX + g.planeX*camX
		rayDirY := g.dirY + g.planeY*camX

		mx, my := int(g.posX), int(g.posY)

		// Step sizes (distance between consecutive grid intersections).
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

		// Initial side-distances and step directions.
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

		// DDA march.
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

		// Perpendicular (fish-eye-free) distance to wall.
		var dist float64
		if side == 0 {
			dist = sdx - ddx
		} else {
			dist = sdy - ddy
		}
		if dist < 0.01 {
			dist = 0.01
		}

		lineH := int(float64(screenH) / dist)
		y0 := half - lineH/2
		y1 := half + lineH/2
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= screenH {
			y1 = screenH - 1
		}

		// Colour with distance fog and face shading.
		wt := hit
		if wt >= len(wallColors) {
			wt = 1
		}
		c := wallColors[wt][side]
		fog := 1.0 - math.Min(dist/16.0, 0.92)
		wr := uint8(float64(c.r) * fog)
		wg := uint8(float64(c.g) * fog)
		wb := uint8(float64(c.b) * fog)

		for y := y0; y <= y1; y++ {
			i := (y*screenW + col) * 4
			px[i], px[i+1], px[i+2], px[i+3] = wr, wg, wb, 255
		}
	}

	g.framebuf.WritePixels(px)
}

// ── Minimap ───────────────────────────────────────────────────────────────────
//
// Drawn in SetPostDrawFunc so it floats above the scene without needing a
// dedicated node hierarchy.

func (g *game) drawMinimap(screen *ebiten.Image) {
	const (
		cell   = 7
		offX   = 8
		offY   = 20 // below the debug text line
		radius = 2  // player dot half-size
	)

	// Scratch image for each cell (reuse via Fill + DrawImage).
	tile := ebiten.NewImage(cell-1, cell-1)

	for my := 0; my < mapH; my++ {
		for mx := 0; mx < mapW; mx++ {
			x0 := float64(offX + mx*cell)
			y0 := float64(offY + my*cell)

			v := worldMap[my][mx]
			if v > 0 {
				c := wallColors[v][0]
				tile.Fill(color.RGBA{R: c.r, G: c.g, B: c.b, A: 217})
			} else {
				tile.Fill(color.RGBA{R: 25, G: 25, B: 25, A: 178})
			}

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x0, y0)
			screen.DrawImage(tile, op)
		}
	}

	// Player dot.
	dot := ebiten.NewImage(radius*2+1, radius*2+1)
	dot.Fill(color.RGBA{R: 255, G: 255, B: 0, A: 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(
		float64(offX)+g.posX*cell-radius,
		float64(offY)+g.posY*cell-radius,
	)
	screen.DrawImage(dot, op)

	// Direction indicator.
	ex := float64(offX) + (g.posX+g.dirX*2)*cell
	ey := float64(offY) + (g.posY+g.dirY*2)*cell
	ebitenutil.DrawLine(screen,
		float64(offX)+g.posX*cell, float64(offY)+g.posY*cell,
		ex, ey,
		color.RGBA{R: 255, G: 255, B: 0, A: 255},
	)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	g := newGame()

	if *autotest != "" {
		g.autopilot = true
		scriptData, err := os.ReadFile(*autotest)
		if err != nil {
			log.Fatalf("read test script: %v", err)
		}
		runner, err := willow.LoadTestScript(scriptData)
		if err != nil {
			log.Fatalf("parse test script: %v", err)
		}
		g.scene.SetTestRunner(runner)
		g.scene.ScreenshotDir = "screenshots"
		origUpdate := g.update
		g.scene.SetUpdateFunc(func() error {
			if err := origUpdate(); err != nil {
				return err
			}
			if runner.Done() {
				fmt.Println("Autotest complete.")
				return ebiten.Termination
			}
			return nil
		})
	}

	if err := willow.Run(g.scene, willow.RunConfig{
		Title:   "Dungeon Crawler — Willow Testbed",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}
