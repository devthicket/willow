// 2.5D raycaster — GPU-accelerated with textured walls, floors, and ceilings.
//
// Walls are rendered as screen-space quads in a Willow mesh node with UV
// coordinates mapped into a shared tileset. A fog shader handles distance
// attenuation on the GPU. Floors and ceilings are rendered via a separate
// Kage shader that performs per-pixel floor-casting entirely on the GPU,
// receiving camera state through uniforms and sampling the same tileset.
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
	"math"
	"os"

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

	// Tileset layout: 128×128 image, 4×4 grid of 32px tiles.
	tileSize = 32
	tileCols = 4
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
	{1, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 5, 0, 1},
	{1, 0, 3, 0, 0, 0, 2, 2, 2, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 1},
	{1, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

// ── Tile indices (row * tileCols + col) into the 4×4 tileset ────────────────
//
// Wall tiles: indexed by wall type (1–5), with [0] = X side, [1] = Y side.
// Using different tiles for X/Y gives a subtle face-shading effect.
var wallTiles = [6][2]int{
	{},       // 0: unused
	{4, 5},   // 1: tiles 5 & 6
	{5, 6},   // 2: tiles 6 & 7
	{6, 7},   // 3: tiles 7 & 8
	{7, 8},   // 4: tiles 8 & 9
	{8, 15},  // 5: tiles 9 & 16
}

// Floor and ceiling tile indices (1-indexed input: floor=10, ceiling=2).
const (
	floorTile   = 1 // tile 2  (row 0, col 1)
	ceilingTile = 9 // tile 10 (row 2, col 1)
)

// tileUV returns the top-left pixel coordinate of a tile in the tileset.
func tileUV(index int) (float32, float32) {
	col := index % tileCols
	row := index / tileCols
	return float32(col * tileSize), float32(row * tileSize)
}

// ── Fog shader (Kage) ────────────────────────────────────────────────────────
//
// Wall distance is encoded in vertex alpha: 1.0 = close, low values = far.
// The shader samples the tileset texture, then applies distance fog.

const fogShaderSrc = `
//kage:unit pixels
package main

func Fragment(dst vec4, src vec2, clr vec4) vec4 {
	c := imageSrc0At(src)
	if c.a < 0.001 {
		return vec4(0)
	}

	// Vertex alpha encodes closeness (1 = near, low = far).
	// Texture colour is the base; vertex RGB is 1,1,1.
	wall := c.rgb / clr.a

	fog := 1.0 - clr.a
	fogged := wall * (1.0 - fog * 0.92)
	return vec4(fogged, 1)
}
`

// ── Floor/ceiling shader (Kage) ──────────────────────────────────────────────
//
// Full-screen quad shader. For each pixel, it computes the world-space floor
// coordinate via standard raycaster floor-casting math, then samples the
// tileset to produce textured floors and ceilings with distance fog.

const floorCeilShaderSrc = `
//kage:unit pixels
package main

var PosX float
var PosY float
var DirX float
var DirY float
var PlaneX float
var PlaneY float
var ScreenW float
var ScreenH float

// Tile UV offsets (top-left corner in tileset pixel coords).
var FloorTileU float
var FloorTileV float
var CeilTileU float
var CeilTileV float
var TileSize float

func Fragment(dst vec4, src vec2, clr vec4) vec4 {
	half := ScreenH / 2.0
	y := dst.y
	x := dst.x

	// Ceiling: mirror the row above horizon.
	isCeiling := 0.0
	row := y - half
	if row < 0.5 {
		row = half - y
		isCeiling = 1.0
	}
	if row < 0.5 {
		return vec4(0)
	}

	// Horizontal distance from camera to this scanline's floor plane.
	rowDist := half / row

	// Direction of the leftmost and rightmost rays for this row.
	rayDirX0 := DirX - PlaneX
	rayDirY0 := DirY - PlaneY
	rayDirX1 := DirX + PlaneX
	rayDirY1 := DirY + PlaneY

	// Interpolate based on the screen x position.
	t := x / ScreenW
	floorX := PosX + rowDist*(rayDirX0+(rayDirX1-rayDirX0)*t)
	floorY := PosY + rowDist*(rayDirY0+(rayDirY1-rayDirY0)*t)

	// Fractional part gives us the position within one tile [0,1).
	u := floorX - floor(floorX)
	v := floorY - floor(floorY)

	// Map to tileset pixel coordinates.
	tileU := FloorTileU
	tileV := FloorTileV
	if isCeiling > 0.5 {
		tileU = CeilTileU
		tileV = CeilTileV
	}
	texX := tileU + u*TileSize
	texY := tileV + v*TileSize

	texel := imageSrc0At(vec2(texX, texY))

	// Distance fog (same curve as walls).
	maxDist := 16.0
	fog := min(rowDist/maxDist, 0.92)
	fogged := texel.rgb * (1.0 - fog)

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
	floorMesh   *willow.Node // full-screen quad for floor/ceiling shader
	wallMesh    *willow.Node // all wall columns, single DrawTriangles
	weaponNode  *willow.Node
	healthText  *willow.Node
	ammoText    *willow.Node
	damageFlash *willow.Node

	tileset     *ebiten.Image
	floorFilter *willow.CustomShaderFilter

	// Pre-allocated vertex/index buffers for wall mesh (reused each frame).
	wallVerts []ebiten.Vertex
	wallInds  []uint16

	// Pre-built minimap tiles (static geometry never changes).
	minimapBG  *ebiten.Image
	minimapDot *ebiten.Image
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

	// ── Load tileset ────────────────────────────────────────────────────
	g.tileset = loadTileset()

	g.scene = willow.NewScene()
	g.scene.ClearColor = willow.RGB(0, 0, 0)

	// ── Floor/ceiling: full-screen quad with raycasting shader ───────
	floorShader, err := ebiten.NewShader([]byte(floorCeilShaderSrc))
	if err != nil {
		log.Fatalf("compile floor shader: %v", err)
	}
	g.floorFilter = willow.NewCustomShaderFilter(floorShader, 0)

	floorU, floorV := tileUV(floorTile)
	ceilU, ceilV := tileUV(ceilingTile)
	g.floorFilter.Uniforms["ScreenW"] = float32(screenW)
	g.floorFilter.Uniforms["ScreenH"] = float32(screenH)
	g.floorFilter.Uniforms["FloorTileU"] = floorU
	g.floorFilter.Uniforms["FloorTileV"] = floorV
	g.floorFilter.Uniforms["CeilTileU"] = ceilU
	g.floorFilter.Uniforms["CeilTileV"] = ceilV
	g.floorFilter.Uniforms["TileSize"] = float32(tileSize - 1)
	g.floorFilter.Uniforms["PosX"] = float32(g.posX)
	g.floorFilter.Uniforms["PosY"] = float32(g.posY)
	g.floorFilter.Uniforms["DirX"] = float32(g.dirX)
	g.floorFilter.Uniforms["DirY"] = float32(g.dirY)
	g.floorFilter.Uniforms["PlaneX"] = float32(g.planeX)
	g.floorFilter.Uniforms["PlaneY"] = float32(g.planeY)

	// Render the tileset 1:1 into the top-left of a screen-covering mesh.
	// The filter's imageSrc0 will contain the tileset at its native pixel
	// coordinates, so the shader can sample with tileset-space UVs directly.
	floorVerts := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 128, DstY: 0, SrcX: 128, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 128, DstY: 128, SrcX: 128, SrcY: 128, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: 128, SrcX: 0, SrcY: 128, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		// Extend the mesh bounding box to cover the full screen (transparent).
		{DstX: screenW, DstY: screenH, SrcX: 0.5, SrcY: 0.5, ColorR: 0, ColorG: 0, ColorB: 0, ColorA: 0},
	}
	floorInds := []uint16{0, 1, 2, 0, 2, 3}
	g.floorMesh = willow.NewMesh("floor-ceil", g.tileset, floorVerts, floorInds)
	g.floorMesh.Filters = []any{g.floorFilter}
	g.scene.Root.AddChild(g.floorMesh)

	// ── Wall mesh: rebuilt each frame ────────────────────────────────────
	g.wallMesh = willow.NewMesh("walls", g.tileset, nil, nil)
	g.scene.Root.AddChild(g.wallMesh)

	// Compile and attach the fog shader as a post-process filter.
	fogShader, err := ebiten.NewShader([]byte(fogShaderSrc))
	if err != nil {
		log.Fatalf("compile fog shader: %v", err)
	}
	g.wallMesh.Filters = []any{willow.NewCustomShaderFilter(fogShader, 0)}

	// ── HUD ──────────────────────────────────────────────────────────────
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
	g.buildWalls()
	g.updateFloorUniforms()
	g.updateHUD()
	return nil
}

func (g *game) updateFloorUniforms() {
	g.floorFilter.Uniforms["PosX"] = float32(g.posX)
	g.floorFilter.Uniforms["PosY"] = float32(g.posY)
	g.floorFilter.Uniforms["DirX"] = float32(g.dirX)
	g.floorFilter.Uniforms["DirY"] = float32(g.dirY)
	g.floorFilter.Uniforms["PlaneX"] = float32(g.planeX)
	g.floorFilter.Uniforms["PlaneY"] = float32(g.planeY)
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
// DDA raycasting builds one screen-space quad per visible column. Each quad's
// SrcX/SrcY coordinates are mapped into the tileset based on the wall type
// and the fractional hit position along the wall face.

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

		// Compute wallX — the fractional position where the ray hit the wall face [0,1).
		var wallX float64
		if side == 0 {
			wallX = g.posY + dist*rayDirY
		} else {
			wallX = g.posX + dist*rayDirX
		}
		wallX -= math.Floor(wallX)

		// Look up tile UV in the tileset.
		wt := hit
		if wt >= len(wallTiles) {
			wt = 1
		}
		tileIdx := wallTiles[wt][side]
		tileU, tileV := tileUV(tileIdx)

		// Map wallX to the horizontal pixel within the tile.
		srcX := tileU + float32(wallX)*float32(tileSize-1)

		// Clamp to screen bounds and adjust texture V to match.
		// Without clamping, close walls produce huge DstY ranges that
		// blow up Ebitengine's atlas allocation.
		clampY0 := math.Max(y0, 0)
		clampY1 := math.Min(y1, screenH)
		if clampY0 >= clampY1 {
			continue
		}
		t0 := (clampY0 - y0) / (y1 - y0)
		t1 := (clampY1 - y0) / (y1 - y0)
		srcY0 := tileV + float32(t0)*float32(tileSize-1)
		srcY1 := tileV + float32(t1)*float32(tileSize-1)

		// Distance encoded in alpha for the fog shader.
		alpha := float32(1.0 - math.Min(dist/16.0, 0.92))

		x := float32(col)
		fy0 := float32(clampY0)
		fy1 := float32(clampY1)

		base := uint16(len(verts))
		verts = append(verts,
			ebiten.Vertex{DstX: x, DstY: fy0, SrcX: srcX, SrcY: srcY0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
			ebiten.Vertex{DstX: x + 1, DstY: fy0, SrcX: srcX + 1, SrcY: srcY0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
			ebiten.Vertex{DstX: x + 1, DstY: fy1, SrcX: srcX + 1, SrcY: srcY1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
			ebiten.Vertex{DstX: x, DstY: fy1, SrcX: srcX, SrcY: srcY1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
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

var minimapColors = [6]color.RGBA{
	{},                                // 0: unused
	{R: 110, G: 110, B: 110, A: 217}, // 1
	{R: 130, G: 70, B: 50, A: 217},   // 2
	{R: 50, G: 80, B: 50, A: 217},    // 3
	{R: 150, G: 140, B: 80, A: 217},  // 4
	{R: 100, G: 40, B: 40, A: 217},   // 5
}

func (g *game) buildMinimapImages() {
	w := mapW * mmCell
	h := mapH * mmCell
	g.minimapBG = ebiten.NewImage(w, h)

	tile := ebiten.NewImage(mmCell-1, mmCell-1)
	for my := 0; my < mapH; my++ {
		for mx := 0; mx < mapW; mx++ {
			v := worldMap[my][mx]
			if v > 0 {
				tile.Fill(minimapColors[v])
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
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(mmOffX, mmOffY)
	screen.DrawImage(g.minimapBG, op)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(
		float64(mmOffX)+g.posX*mmCell-mmRadius,
		float64(mmOffY)+g.posY*mmCell-mmRadius,
	)
	screen.DrawImage(g.minimapDot, op2)

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
