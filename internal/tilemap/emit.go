package tilemap

import (
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/types"
)

// EmitTilemapCommands transforms tile vertices into screen space and appends
// one or more CommandTilemap render commands (split at MaxTilesPerDraw) to the
// provided command slice. viewTransform is the camera's view matrix.
func EmitTilemapCommands(layer *Layer, commands *[]render.RenderCommand, viewTransform [6]float64, treeOrder *int) {
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
	for offset := 0; offset < totalTiles; offset += MaxTilesPerDraw {
		end := offset + MaxTilesPerDraw
		if end > totalTiles {
			end = totalTiles
		}
		batchTiles := end - offset

		*treeOrder++
		*commands = append(*commands, render.RenderCommand{
			Type:         render.CommandTilemap,
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

// CameraBoundsAdapter wraps *camera.Camera to satisfy VisibleBoundsProvider.
type CameraBoundsAdapter struct {
	Cam *camera.Camera
}

func (a *CameraBoundsAdapter) VisibleBounds() types.Rect {
	return a.Cam.VisibleBounds()
}

func (a *CameraBoundsAdapter) ViewportWidth() float64 {
	return a.Cam.Viewport.Width
}

func (a *CameraBoundsAdapter) ViewportHeight() float64 {
	return a.Cam.Viewport.Height
}
