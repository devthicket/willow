package raycast

import (
	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

// Config holds initialization parameters for the Renderer.
type Config struct {
	ScreenW int
	ScreenH int
}

// Renderer manages the raycaster's mesh nodes and per-frame updates.
// The caller adds FloorNode and WallNode to the scene tree.
type Renderer struct {
	Camera *Camera
	World  *World

	// FloorNode is the full-screen quad for floor/ceiling shader rendering.
	FloorNode *willow.Node
	// WallNode is the dynamic mesh rebuilt each frame with wall column quads.
	WallNode *willow.Node

	floorFilter *willow.CustomShaderFilter
	wallVerts   []ebiten.Vertex
	wallInds    []uint16
	cfg         Config
}

// New creates a Renderer, compiles shaders, and builds the floor and wall
// mesh nodes. The tilesetImg is the atlas page image that both meshes sample.
// The caller is responsible for adding FloorNode and WallNode to the scene
// tree and for registering tilesetImg as an atlas page.
func New(cfg Config, cam *Camera, world *World, tilesetImg *ebiten.Image) (*Renderer, error) {
	r := &Renderer{
		Camera:    cam,
		World:     world,
		cfg:       cfg,
		wallVerts: make([]ebiten.Vertex, 0, cfg.ScreenW*4),
		wallInds:  make([]uint16, 0, cfg.ScreenW*6),
	}

	// ── Floor/ceiling: full-screen quad with raycasting shader ───────
	floorShader, err := CompileFloorCeilShader()
	if err != nil {
		return nil, err
	}
	r.floorFilter = willow.NewCustomShaderFilter(floorShader, 0)

	floorU, floorV := world.TileUV(world.FloorTile)
	ceilU, ceilV := world.TileUV(world.CeilTile)
	r.floorFilter.Uniforms["ScreenW"] = float32(cfg.ScreenW)
	r.floorFilter.Uniforms["ScreenH"] = float32(cfg.ScreenH)
	r.floorFilter.Uniforms["FloorTileU"] = floorU
	r.floorFilter.Uniforms["FloorTileV"] = floorV
	r.floorFilter.Uniforms["CeilTileU"] = ceilU
	r.floorFilter.Uniforms["CeilTileV"] = ceilV
	r.floorFilter.Uniforms["TileSize"] = float32(world.TileSize - 1)
	r.floorFilter.Uniforms["PosX"] = float32(cam.PosX)
	r.floorFilter.Uniforms["PosY"] = float32(cam.PosY)
	r.floorFilter.Uniforms["DirX"] = float32(cam.DirX)
	r.floorFilter.Uniforms["DirY"] = float32(cam.DirY)
	r.floorFilter.Uniforms["PlaneX"] = float32(cam.PlaneX)
	r.floorFilter.Uniforms["PlaneY"] = float32(cam.PlaneY)

	// Render the tileset 1:1 into the top-left of a screen-covering mesh.
	// The filter's imageSrc0 will contain the tileset at its native pixel
	// coordinates, so the shader can sample with tileset-space UVs directly.
	tw := float32(tilesetImg.Bounds().Dx())
	th := float32(tilesetImg.Bounds().Dy())
	floorVerts := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: tw, DstY: 0, SrcX: tw, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: tw, DstY: th, SrcX: tw, SrcY: th, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 0, DstY: th, SrcX: 0, SrcY: th, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		// Extend the mesh bounding box to cover the full screen (transparent).
		{DstX: float32(cfg.ScreenW), DstY: float32(cfg.ScreenH), SrcX: 0.5, SrcY: 0.5, ColorR: 0, ColorG: 0, ColorB: 0, ColorA: 0},
	}
	floorInds := []uint16{0, 1, 2, 0, 2, 3}
	r.FloorNode = willow.NewMesh("floor-ceil", tilesetImg, floorVerts, floorInds)
	r.FloorNode.Filters = []any{r.floorFilter}

	// ── Wall mesh: rebuilt each frame ────────────────────────────────
	fogShader, err := CompileFogShader()
	if err != nil {
		return nil, err
	}
	r.WallNode = willow.NewMesh("walls", tilesetImg, nil, nil)
	r.WallNode.Filters = []any{willow.NewCustomShaderFilter(fogShader, 0)}

	return r, nil
}

// Update rebuilds wall geometry and refreshes floor shader uniforms.
// Call once per frame from the game's update function.
func (r *Renderer) Update() {
	r.wallVerts, r.wallInds = BuildWalls(
		r.Camera, r.World,
		r.cfg.ScreenW, r.cfg.ScreenH,
		r.wallVerts[:0], r.wallInds[:0],
	)
	r.WallNode.SetMeshVertices(r.wallVerts)
	r.WallNode.SetMeshIndices(r.wallInds)

	r.floorFilter.Uniforms["PosX"] = float32(r.Camera.PosX)
	r.floorFilter.Uniforms["PosY"] = float32(r.Camera.PosY)
	r.floorFilter.Uniforms["DirX"] = float32(r.Camera.DirX)
	r.floorFilter.Uniforms["DirY"] = float32(r.Camera.DirY)
	r.floorFilter.Uniforms["PlaneX"] = float32(r.Camera.PlaneX)
	r.floorFilter.Uniforms["PlaneY"] = float32(r.Camera.PlaneY)
}
