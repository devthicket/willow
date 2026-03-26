package raycast

import "github.com/hajimehoshi/ebiten/v2"

// FogShaderSrc is the Kage source for the wall fog post-process.
//
// Wall distance is encoded in vertex alpha: 1.0 = close, low values = far.
// The shader samples the tileset texture, then applies distance fog.
const FogShaderSrc = `
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

// FloorCeilShaderSrc is the Kage source for per-pixel floor/ceiling casting.
//
// Full-screen quad shader. For each pixel it computes the world-space floor
// coordinate via standard raycaster floor-casting math, then samples the
// tileset to produce textured floors and ceilings with distance fog.
const FloorCeilShaderSrc = `
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

// CompileFogShader compiles and returns the wall fog shader.
func CompileFogShader() (*ebiten.Shader, error) {
	return ebiten.NewShader([]byte(FogShaderSrc))
}

// CompileFloorCeilShader compiles and returns the floor/ceiling shader.
func CompileFloorCeilShader() (*ebiten.Shader, error) {
	return ebiten.NewShader([]byte(FloorCeilShaderSrc))
}
