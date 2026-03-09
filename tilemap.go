package willow

import (
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/tilemap"
	"github.com/phanxgames/willow/internal/types"
)

// AnimFrame describes a single frame in a tile animation sequence.
// Aliased from internal/tilemap.AnimFrame.
type AnimFrame = tilemap.AnimFrame

// TileLayerConfig holds the parameters for creating a tile layer.
// Aliased from internal/tilemap.LayerConfig.
type TileLayerConfig = tilemap.LayerConfig

// TileMapLayer is a single layer of tile data.
// Aliased from internal/tilemap.Layer.
type TileMapLayer = tilemap.Layer

// TileMapViewport is a scene graph node that manages a viewport into a tilemap.
// Aliased from internal/tilemap.Viewport.
type TileMapViewport = tilemap.Viewport

func init() {
	// Wire the NewContainerFn so internal/tilemap can create container nodes.
	tilemap.NewContainerFn = func(name string) *Node {
		return NewContainer(name)
	}

	// Wire the layer emit function so render commands are emitted correctly.
	tilemap.NewLayerEmitFn = func(layer *tilemap.Layer) {
		layer.EmitFn = emitTilemapCommands
	}
}

// cameraBoundsAdapter wraps *camera.Camera to satisfy tilemap.VisibleBoundsProvider.
type cameraBoundsAdapter struct {
	cam *camera.Camera
}

func (a *cameraBoundsAdapter) VisibleBounds() types.Rect {
	return a.cam.VisibleBounds()
}

func (a *cameraBoundsAdapter) ViewportWidth() float64 {
	return a.cam.Viewport.Width
}

func (a *cameraBoundsAdapter) ViewportHeight() float64 {
	return a.cam.Viewport.Height
}

// NewTileMapViewport creates a new tilemap viewport node with the given tile
// dimensions, wiring all necessary function pointers.
func NewTileMapViewport(name string, tileWidth, tileHeight int) *TileMapViewport {
	v := tilemap.NewViewport(name, tileWidth, tileHeight)
	v.CameraBoundsFn = func(cam any) tilemap.VisibleBoundsProvider {
		c, ok := cam.(*camera.Camera)
		if !ok || c == nil {
			return nil
		}
		return &cameraBoundsAdapter{cam: c}
	}
	return v
}

// emitTilemapCommands is wired as the EmitFn for each TileMapLayer.
func emitTilemapCommands(layer *tilemap.Layer, sAny any, treeOrder *int) {
	// Accept either *render.Pipeline (internal path) or *Scene (legacy path).
	var commands *[]RenderCommand
	var viewTransform [6]float64
	switch v := sAny.(type) {
	case *render.Pipeline:
		commands = &v.Commands
		viewTransform = v.ViewTransform
	case *Scene:
		commands = &v.Pipeline.Commands
		viewTransform = v.Pipeline.ViewTransform
	default:
		return
	}

	layer.LateRebuildCheck()

	if layer.TileCount == 0 || layer.AtlasImage == nil {
		return
	}

	// Get the accumulated color from the node's parent chain.
	n := layer.Node_
	r := float32(n.Color_.R() * n.WorldAlpha)
	g := float32(n.Color_.G() * n.WorldAlpha)
	b := float32(n.Color_.B() * n.WorldAlpha)
	a := float32(n.WorldAlpha)

	// Premultiply.
	cr := r * a
	cg := g * a
	cb := b * a

	tw := float32(layer.Viewport_.TileWidth)
	th := float32(layer.Viewport_.TileHeight)

	// View transform for world-to-screen conversion.
	vt := viewTransform
	va, vb := float32(vt[0]), float32(vt[1])
	vc, vd := float32(vt[2]), float32(vt[3])
	vtx, vty := float32(vt[4]), float32(vt[5])

	// Compute tile screen dimensions (for axis-aligned cameras).
	tileScreenW := va * tw
	tileScreenH := vd * th

	// Transform all active tile vertices.
	for i := 0; i < layer.TileCount; i++ {
		wx := layer.WorldX[i]
		wy := layer.WorldY[i]

		screenX := va*wx + vc*wy + vtx
		screenY := vb*wx + vd*wy + vty

		vi := i * 4
		layer.Vertices[vi+0].DstX = screenX
		layer.Vertices[vi+0].DstY = screenY
		layer.Vertices[vi+1].DstX = screenX + tileScreenW
		layer.Vertices[vi+1].DstY = screenY
		layer.Vertices[vi+2].DstX = screenX
		layer.Vertices[vi+2].DstY = screenY + tileScreenH
		layer.Vertices[vi+3].DstX = screenX + tileScreenW
		layer.Vertices[vi+3].DstY = screenY + tileScreenH

		layer.Vertices[vi+0].ColorR = cr
		layer.Vertices[vi+0].ColorG = cg
		layer.Vertices[vi+0].ColorB = cb
		layer.Vertices[vi+0].ColorA = a
		layer.Vertices[vi+1].ColorR = cr
		layer.Vertices[vi+1].ColorG = cg
		layer.Vertices[vi+1].ColorB = cb
		layer.Vertices[vi+1].ColorA = a
		layer.Vertices[vi+2].ColorR = cr
		layer.Vertices[vi+2].ColorG = cg
		layer.Vertices[vi+2].ColorB = cb
		layer.Vertices[vi+2].ColorA = a
		layer.Vertices[vi+3].ColorR = cr
		layer.Vertices[vi+3].ColorG = cg
		layer.Vertices[vi+3].ColorB = cb
		layer.Vertices[vi+3].ColorA = a
	}

	// Emit commands, splitting at MaxTilesPerDraw boundaries.
	totalTiles := layer.TileCount
	for offset := 0; offset < totalTiles; offset += tilemap.MaxTilesPerDraw {
		end := offset + tilemap.MaxTilesPerDraw
		if end > totalTiles {
			end = totalTiles
		}
		batchTiles := end - offset

		*treeOrder++
		*commands = append(*commands, RenderCommand{
			Type:         CommandTilemap,
			RenderLayer:  n.RenderLayer,
			GlobalOrder:  n.GlobalOrder,
			TreeOrder:    *treeOrder,
			BlendMode:    n.BlendMode_,
			TilemapVerts: layer.Vertices[offset*4 : end*4],
			TilemapInds:  layer.Indices[:batchTiles*6],
			TilemapImage: layer.AtlasImage,
		})
	}
}
