// 2.5D raycaster — GPU-accelerated with textured walls, floors, and ceilings.
//
// The raycasting engine lives in the raycast sub-package. This file wires
// the engine into a Willow scene and adds game-specific HUD, input handling,
// weapon effects, and a minimap overlay.
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
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/devthicket/willow"
	"github.com/devthicket/willow/examples/demos/3d-raycaster-text/raycast"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tanema/gween/ease"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	screenW   = 800
	screenH   = 600
	moveSpeed = 0.07
	rotSpeed  = 0.045
)

const levelsDir = "examples/demos/3d-raycaster-text/levels"

// ── Game struct ───────────────────────────────────────────────────────────────

type game struct {
	renderer *raycast.Renderer

	health, ammo int
	fired        bool
	kickTween    *willow.TweenGroup

	scene       *willow.Scene
	weaponNode  *willow.Node
	healthText  *willow.Node
	ammoText    *willow.Node
	damageFlash *willow.Node

	minimapBG  *ebiten.Image
	minimapDot *ebiten.Image
}

// ── Constructor ───────────────────────────────────────────────────────────────

func newGame(levelPath string) *game {
	lvl, err := raycast.LoadLevel(levelPath)
	if err != nil {
		log.Fatalf("load level: %v", err)
	}

	g := &game{
		health: 100,
		ammo:   30,
	}

	g.scene = willow.NewScene()
	g.scene.ClearColor = willow.RGB(0, 0, 0)

	// ── Tileset atlas ────────────────────────────────────────────────
	tilesetImg := loadTileset()
	g.scene.RegisterPage(0, tilesetImg)

	// ── Raycaster engine ────────────────────────────────────────────
	cam := lvl.Camera()
	world := lvl.World()

	g.renderer, err = raycast.New(raycast.Config{
		ScreenW: screenW,
		ScreenH: screenH,
	}, cam, world, tilesetImg)
	if err != nil {
		log.Fatalf("init raycaster: %v", err)
	}

	g.scene.Root.AddChild(g.renderer.FloorNode)
	g.scene.Root.AddChild(g.renderer.WallNode)

	// ── HUD ──────────────────────────────────────────────────────────
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
			"W/S: move   A/D: strafe   Q/E or <-/->: turn   SPACE: fire",
			screenW-380, 8)
		g.drawMinimap(screen)
	})

	return g
}

// ── Tileset loader ───────────────────────────────────────────────────────────

func loadTileset() *ebiten.Image {
	f, err := os.Open("examples/_assets/tileset.png")
	if err != nil {
		log.Fatalf("open tileset: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatalf("decode tileset: %v", err)
	}
	return ebiten.NewImageFromImage(img)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (g *game) update() error {
	g.handleInput()
	g.checkWeaponReturn()
	g.renderer.Update()
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
	cam := g.renderer.Camera

	if g.scene.IsKeyPressed(ebiten.KeyLeft) || g.scene.IsKeyPressed(ebiten.KeyQ) {
		cam.Rotate(-rotSpeed)
	}
	if g.scene.IsKeyPressed(ebiten.KeyRight) || g.scene.IsKeyPressed(ebiten.KeyE) {
		cam.Rotate(rotSpeed)
	}

	walkable := g.renderer.World.Walkable
	if g.scene.IsKeyPressed(ebiten.KeyW) || g.scene.IsKeyPressed(ebiten.KeyArrowUp) {
		cam.TryMove(cam.DirX, cam.DirY, moveSpeed, walkable)
	}
	if g.scene.IsKeyPressed(ebiten.KeyS) || g.scene.IsKeyPressed(ebiten.KeyArrowDown) {
		cam.TryMove(cam.DirX, cam.DirY, -moveSpeed, walkable)
	}
	if g.scene.IsKeyPressed(ebiten.KeyA) {
		cam.TryMove(-cam.DirY, cam.DirX, moveSpeed*0.75, walkable)
	}
	if g.scene.IsKeyPressed(ebiten.KeyD) {
		cam.TryMove(cam.DirY, -cam.DirX, moveSpeed*0.75, walkable)
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

// ── Minimap ───────────────────────────────────────────────────────────────────

const (
	mmCell   = 7
	mmOffY   = 28
	mmRadius = 2
)

var minimapColors = [6]color.RGBA{
	{},                                // 0: unused
	{R: 110, G: 110, B: 110, A: 217}, // 1
	{R: 130, G: 70, B: 50, A: 217},   // 2
	{R: 50, G: 80, B: 50, A: 217},    // 3
	{R: 150, G: 140, B: 80, A: 217},  // 4
	{R: 100, G: 40, B: 40, A: 217},   // 5
}

func (g *game) buildMinimapImages() {
	world := g.renderer.World
	mw, mh := world.Width, world.Height
	w := mw * mmCell
	h := mh * mmCell
	g.minimapBG = ebiten.NewImage(w, h)

	tile := ebiten.NewImage(mmCell-1, mmCell-1)
	for my := 0; my < mh; my++ {
		for mx := 0; mx < mw; mx++ {
			v := world.CellAt(mx, my)
			if v > 0 && v < len(minimapColors) {
				tile.Fill(minimapColors[v])
			} else if v > 0 {
				tile.Fill(minimapColors[1])
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
	cam := g.renderer.Camera
	mmOffX := float64(screenW - g.renderer.World.Width*mmCell - 8)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(mmOffX, mmOffY)
	screen.DrawImage(g.minimapBG, op)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(
		mmOffX+cam.PosX*mmCell-mmRadius,
		float64(mmOffY)+cam.PosY*mmCell-mmRadius,
	)
	screen.DrawImage(g.minimapDot, op2)

	ex := mmOffX + (cam.PosX+cam.DirX*2)*mmCell
	ey := float64(mmOffY) + (cam.PosY+cam.DirY*2)*mmCell
	ebitenutil.DrawLine(screen,
		mmOffX+cam.PosX*mmCell, float64(mmOffY)+cam.PosY*mmCell,
		ex, ey,
		color.RGBA{R: 255, G: 255, B: 0, A: 255},
	)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	levelFlag := flag.String("level", filepath.Join(levelsDir, "level1.json"), "path to level JSON file")
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	g := newGame(*levelFlag)

	if *autotest != "" {
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
		Title:   "2.5D Raycaster — Textured Walls, Floors & Ceilings",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}
