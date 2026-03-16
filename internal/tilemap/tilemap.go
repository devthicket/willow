package tilemap

import (
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// Function pointers wired by root to break dependency on Camera, Scene, etc.
var (
	// NewContainerFn creates a container node. Set by root package.
	NewContainerFn func(name string) *node.Node

	// NewLayerEmitFn, if set, is called after AddTileLayer creates a Layer so
	// root can inject the render-command emission logic as Layer.EmitFn.
	NewLayerEmitFn func(layer *Layer)
)

// GID flag bits (same convention as Tiled TMX format).
const (
	TileFlipH    uint32 = 1 << 31
	TileFlipV    uint32 = 1 << 30
	TileFlipD    uint32 = 1 << 29
	TileFlagMask uint32 = TileFlipH | TileFlipV | TileFlipD
)

// MaxTilesPerDraw is the maximum number of tiles per DrawTriangles call.
const MaxTilesPerDraw = 16383

// AnimFrame describes a single frame in a tile animation sequence.
type AnimFrame struct {
	GID      uint32 // tile GID for this frame (no flag bits)
	Duration int    // milliseconds
}

// UVOrder defines vertex UV assignment for each combination of flip flags.
var UVOrder = [8][4]int{
	{0, 1, 2, 3}, // no flags
	{2, 0, 3, 1}, // D only
	{2, 3, 0, 1}, // V flip
	{3, 2, 1, 0}, // V+D
	{1, 0, 3, 2}, // H flip
	{0, 2, 1, 3}, // H+D
	{3, 2, 1, 0}, // H+V
	{1, 3, 0, 2}, // H+V+D
}

// VisibleBoundsProvider is the interface for camera visible bounds.
type VisibleBoundsProvider interface {
	VisibleBounds() types.Rect
	ViewportWidth() float64
	ViewportHeight() float64
}

// EmitCommandsFn is the callback type for emitting tilemap render commands.
// The any parameter is the opaque *Scene, matching Node.CustomEmit signature.
type EmitCommandsFn func(layer *Layer, sceneAny any, treeOrder *int)

// Viewport is a scene graph node that manages a viewport into a tilemap.
type Viewport struct {
	node_ *node.Node

	TileWidth   int
	TileHeight  int
	MaxZoomOut  float64
	MarginTiles int

	camera any // opaque camera reference
	// CameraBoundsFn resolves the camera to a VisibleBoundsProvider.
	CameraBoundsFn func(cam any) VisibleBoundsProvider

	layers []*Layer

	animElapsed int
}

// Layer is a single layer of tile data.
type Layer struct {
	Node_       *node.Node
	Data        []uint32
	Width       int
	Height      int
	Vertices    []ebiten.Vertex
	Indices     []uint16
	TileCount   int
	WorldX      []float32
	WorldY      []float32
	Regions     []types.TextureRegion
	BufStartCol int
	BufStartRow int
	BufCols     int
	BufRows     int
	BufDirty    bool
	Anims       map[uint32][]AnimFrame
	AtlasImage  *ebiten.Image
	Viewport_   *Viewport

	// EmitFn is called during rendering to emit tilemap commands.
	EmitFn EmitCommandsFn
}

// LayerConfig holds the parameters for creating a tile layer.
type LayerConfig struct {
	Name       string
	Width      int
	Height     int
	Data       []uint32
	Regions    []types.TextureRegion
	AtlasImage *ebiten.Image
}

// NewViewport creates a new tilemap viewport node.
func NewViewport(name string, tileWidth, tileHeight int) *Viewport {
	v := &Viewport{
		node_:       NewContainerFn(name),
		TileWidth:   tileWidth,
		TileHeight:  tileHeight,
		MaxZoomOut:  1.0,
		MarginTiles: 2,
	}
	v.node_.OnUpdate = v.update
	return v
}

// Node returns the underlying scene graph node.
func (v *Viewport) Node() *node.Node {
	return v.node_
}

// SetCamera binds this viewport to a specific camera (opaque).
func (v *Viewport) SetCamera(cam any) {
	v.camera = cam
}

// AddTileLayer creates a tile layer and adds it as a child of the viewport.
func (v *Viewport) AddTileLayer(cfg LayerConfig) *Layer {
	layer := &Layer{
		Node_:       NewContainerFn(cfg.Name),
		Data:        cfg.Data,
		Width:       cfg.Width,
		Height:      cfg.Height,
		Regions:     cfg.Regions,
		AtlasImage:  cfg.AtlasImage,
		Viewport_:   v,
		BufDirty:    true,
		BufStartCol: -1,
		BufStartRow: -1,
	}

	layer.Node_.CustomEmit = func(sceneAny any, treeOrder *int) {
		if layer.EmitFn != nil {
			layer.EmitFn(layer, sceneAny, treeOrder)
		}
	}

	if NewLayerEmitFn != nil {
		NewLayerEmitFn(layer)
	}

	v.node_.AddChild(layer.Node_)
	v.layers = append(v.layers, layer)
	return layer
}

// AddChild adds a regular node container as a child (sandwich layers).
func (v *Viewport) AddChild(child *node.Node) {
	v.node_.AddChild(child)
}

// Layers returns the tile layers.
func (v *Viewport) Layers() []*Layer {
	return v.layers
}

// SetTile updates a single tile in the given layer.
func (l *Layer) SetTile(col, row int, newGID uint32) {
	if col < 0 || col >= l.Width || row < 0 || row >= l.Height {
		return
	}
	l.Data[row*l.Width+col] = newGID

	if col >= l.BufStartCol && col < l.BufStartCol+l.BufCols &&
		row >= l.BufStartRow && row < l.BufStartRow+l.BufRows {
		l.BufDirty = true
	}
}

// SetData replaces the entire tile data array and forces a full buffer rebuild.
func (l *Layer) SetData(data []uint32, w, h int) {
	l.Data = data
	l.Width = w
	l.Height = h
	l.InvalidateBuffer()
}

// InvalidateBuffer forces a full buffer rebuild on the next frame.
func (l *Layer) InvalidateBuffer() {
	l.BufDirty = true
	l.BufStartCol = -1
	l.BufStartRow = -1
}

// SetAnimations sets the animation definitions for this layer.
func (l *Layer) SetAnimations(anims map[uint32][]AnimFrame) {
	l.Anims = anims
}

// Node returns the layer's scene graph node.
func (l *Layer) Node() *node.Node {
	return l.Node_
}

func (v *Viewport) update(dt float64) {
	if v.camera == nil || v.CameraBoundsFn == nil {
		return
	}
	cam := v.CameraBoundsFn(v.camera)
	if cam == nil {
		return
	}

	bounds := cam.VisibleBounds()
	tw := float64(v.TileWidth)
	th := float64(v.TileHeight)
	zoom := v.MaxZoomOut
	if zoom <= 0 {
		zoom = 1.0
	}

	for _, layer := range v.layers {
		if !layer.Node_.Visible_ {
			continue
		}

		bufCols := int(math.Ceil(cam.ViewportWidth()/tw/zoom)) + 2 + 2*v.MarginTiles
		bufRows := int(math.Ceil(cam.ViewportHeight()/th/zoom)) + 2 + 2*v.MarginTiles

		startCol := int(math.Floor(bounds.X/tw)) - v.MarginTiles
		startRow := int(math.Floor(bounds.Y/th)) - v.MarginTiles

		if startCol < 0 {
			startCol = 0
		}
		if startRow < 0 {
			startRow = 0
		}
		endCol := startCol + bufCols - 1
		endRow := startRow + bufRows - 1
		if endCol >= layer.Width {
			endCol = layer.Width - 1
			startCol = endCol - bufCols + 1
			if startCol < 0 {
				startCol = 0
			}
		}
		if endRow >= layer.Height {
			endRow = layer.Height - 1
			startRow = endRow - bufRows + 1
			if startRow < 0 {
				startRow = 0
			}
		}

		layer.EnsureBuffer(bufCols, bufRows)

		if layer.BufDirty || startCol != layer.BufStartCol || startRow != layer.BufStartRow {
			layer.RebuildBuffer(startCol, startRow, bufCols, bufRows)
		}
	}

	dtMs := int(dt * 1000)
	if dtMs > 0 {
		v.animElapsed += dtMs
		v.UpdateAnimations()
	}
}

// EnsureBuffer grows the geometry buffer if needed.
func (l *Layer) EnsureBuffer(cols, rows int) {
	cap_ := cols * rows
	if cap_ <= len(l.WorldX) {
		return
	}

	l.WorldX = make([]float32, cap_)
	l.WorldY = make([]float32, cap_)
	l.Vertices = make([]ebiten.Vertex, cap_*4)

	l.Indices = make([]uint16, cap_*6)
	for i := 0; i < cap_; i++ {
		base := uint16(i * 4)
		off := i * 6
		l.Indices[off+0] = base + 0
		l.Indices[off+1] = base + 1
		l.Indices[off+2] = base + 2
		l.Indices[off+3] = base + 1
		l.Indices[off+4] = base + 3
		l.Indices[off+5] = base + 2
	}
}

// RebuildBuffer fills the vertex buffer with tile data for the given range.
func (l *Layer) RebuildBuffer(startCol, startRow, bufCols, bufRows int) {
	l.BufStartCol = startCol
	l.BufStartRow = startRow
	l.BufCols = bufCols
	l.BufRows = bufRows
	l.BufDirty = false

	tw := float32(l.Viewport_.TileWidth)
	th := float32(l.Viewport_.TileHeight)

	tileCount := 0
	for br := 0; br < bufRows; br++ {
		row := startRow + br
		if row < 0 || row >= l.Height {
			continue
		}
		rowOffset := row * l.Width
		for bc := 0; bc < bufCols; bc++ {
			col := startCol + bc
			if col < 0 || col >= l.Width {
				continue
			}

			gid := l.Data[rowOffset+col]
			if gid == 0 {
				continue
			}

			flags := gid & TileFlagMask
			tileID := gid &^ TileFlagMask

			if int(tileID) >= len(l.Regions) {
				continue
			}
			region := l.Regions[tileID]

			l.WorldX[tileCount] = float32(col) * tw
			l.WorldY[tileCount] = float32(row) * th

			SetTileUVs(l.Vertices[tileCount*4:], region, flags, tw, th)

			tileCount++
		}
	}
	l.TileCount = tileCount
}

// SetTileUVs sets the UV (SrcX/SrcY) coordinates for 4 vertices of a tile.
func SetTileUVs(verts []ebiten.Vertex, region types.TextureRegion, flags uint32, tw, th float32) {
	sx := float32(region.X)
	sy := float32(region.Y)
	sw := float32(region.Width)
	sh := float32(region.Height)

	uvX := [4]float32{sx, sx + sw, sx, sx + sw}
	uvY := [4]float32{sy, sy, sy + sh, sy + sh}

	flagIdx := 0
	if flags&TileFlipH != 0 {
		flagIdx |= 4
	}
	if flags&TileFlipV != 0 {
		flagIdx |= 2
	}
	if flags&TileFlipD != 0 {
		flagIdx |= 1
	}
	order := UVOrder[flagIdx]

	verts[0].SrcX = uvX[order[0]]
	verts[0].SrcY = uvY[order[0]]
	verts[1].SrcX = uvX[order[1]]
	verts[1].SrcY = uvY[order[1]]
	verts[2].SrcX = uvX[order[2]]
	verts[2].SrcY = uvY[order[2]]
	verts[3].SrcX = uvX[order[3]]
	verts[3].SrcY = uvY[order[3]]
}

// LateRebuildCheck detects whether the camera moved after the Update pass.
func (l *Layer) LateRebuildCheck() {
	v := l.Viewport_
	if v.camera == nil || v.CameraBoundsFn == nil {
		return
	}
	cam := v.CameraBoundsFn(v.camera)
	if cam == nil {
		return
	}

	bounds := cam.VisibleBounds()
	tw := float64(v.TileWidth)
	th := float64(v.TileHeight)

	startCol := int(math.Floor(bounds.X/tw)) - v.MarginTiles
	startRow := int(math.Floor(bounds.Y/th)) - v.MarginTiles
	if startCol < 0 {
		startCol = 0
	}
	if startRow < 0 {
		startRow = 0
	}

	if startCol != l.BufStartCol || startRow != l.BufStartRow {
		v.update(0)
	}
}

// UpdateAnimations scans layers for animated tiles and updates their UVs.
func (v *Viewport) UpdateAnimations() {
	for _, layer := range v.layers {
		if layer.Anims == nil || !layer.Node_.Visible_ {
			continue
		}

		tw := float32(v.TileWidth)
		th := float32(v.TileHeight)

		for i := 0; i < layer.TileCount; i++ {
			col := int(layer.WorldX[i] / tw)
			row := int(layer.WorldY[i] / th)
			if col < 0 || col >= layer.Width || row < 0 || row >= layer.Height {
				continue
			}

			gid := layer.Data[row*layer.Width+col]
			if gid == 0 {
				continue
			}
			flags := gid & TileFlagMask
			baseGID := gid &^ TileFlagMask

			frames, ok := layer.Anims[baseGID]
			if !ok || len(frames) == 0 {
				continue
			}

			totalDuration := 0
			for _, f := range frames {
				totalDuration += f.Duration
			}
			if totalDuration == 0 {
				continue
			}

			elapsed := v.animElapsed % totalDuration
			currentGID := frames[0].GID
			acc := 0
			for _, f := range frames {
				acc += f.Duration
				if elapsed < acc {
					currentGID = f.GID
					break
				}
			}

			if int(currentGID) < len(layer.Regions) {
				region := layer.Regions[currentGID]
				SetTileUVs(layer.Vertices[i*4:], region, flags, tw, th)
			}
		}
	}
}
