package raycast

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// BuildWalls performs DDA raycasting and fills vertex/index buffers with one
// screen-space quad per visible wall column. The caller should pass pre-
// allocated slices (sliced to zero length); they are returned potentially
// grown so the caller can reuse the backing array next frame.
func BuildWalls(
	cam *Camera,
	world *World,
	screenW, screenH int,
	verts []ebiten.Vertex,
	inds []uint16,
) ([]ebiten.Vertex, []uint16) {
	half := float64(screenH) / 2

	for col := 0; col < screenW; col++ {
		camX := 2.0*float64(col)/float64(screenW) - 1.0
		rayDirX := cam.DirX + cam.PlaneX*camX
		rayDirY := cam.DirY + cam.PlaneY*camX

		mx, my := int(cam.PosX), int(cam.PosY)

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
			stepX, sdx = -1, (cam.PosX-float64(mx))*ddx
		} else {
			stepX, sdx = 1, (float64(mx)+1.0-cam.PosX)*ddx
		}
		if rayDirY < 0 {
			stepY, sdy = -1, (cam.PosY-float64(my))*ddy
		} else {
			stepY, sdy = 1, (float64(my)+1.0-cam.PosY)*ddy
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
			if !world.InBounds(mx, my) {
				break
			}
			hit = world.Grid[my][mx]
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

		lineH := float64(screenH) / dist
		y0 := half - lineH/2
		y1 := half + lineH/2

		// Fractional position where the ray hit the wall face [0,1).
		var wallX float64
		if side == 0 {
			wallX = cam.PosY + dist*rayDirY
		} else {
			wallX = cam.PosX + dist*rayDirX
		}
		wallX -= math.Floor(wallX)

		// Look up tile UV in the tileset.
		wt := hit
		if wt >= len(world.WallTiles) {
			wt = 1
		}
		tileIdx := world.WallTiles[wt]
		tileU, tileV := world.TileUV(tileIdx)

		// Map wallX to the horizontal pixel within the tile.
		srcX := tileU + float32(wallX)*float32(world.TileSize-1)

		// Clamp to screen bounds and adjust texture V to match.
		clampY0 := math.Max(y0, 0)
		clampY1 := math.Min(y1, float64(screenH))
		if clampY0 >= clampY1 {
			continue
		}
		t0 := (clampY0 - y0) / (y1 - y0)
		t1 := (clampY1 - y0) / (y1 - y0)
		srcY0 := tileV + float32(t0)*float32(world.TileSize-1)
		srcY1 := tileV + float32(t1)*float32(world.TileSize-1)

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

	return verts, inds
}
