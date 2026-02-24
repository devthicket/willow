package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// GID flag bits (same convention as Tiled TMX format).
const (
	tileFlipH    uint32 = 1 << 31 // horizontal flip
	tileFlipV    uint32 = 1 << 30 // vertical flip
	tileFlipD    uint32 = 1 << 29 // diagonal flip (90° rotation)
	tileFlagMask uint32 = tileFlipH | tileFlipV | tileFlipD
)

// maxTilesPerDraw is the maximum number of tiles per DrawTriangles call.
// Limited by uint16 index buffer: 65535 / 4 vertices per tile = 16383.
const maxTilesPerDraw = 16383

// AnimFrame describes a single frame in a tile animation sequence.
type AnimFrame struct {
	GID      uint32 // tile GID for this frame (no flag bits)
	Duration int    // milliseconds
}

// uvOrder defines vertex UV assignment for each combination of flip flags.
// Indexed by 3-bit flag value: (flipH << 2) | (flipV << 1) | flipD.
// Each entry contains 4 corner indices: TL=0, TR=1, BL=2, BR=3.
// The indices map source UV corners to destination vertex positions:
//
//	result[i] is which source corner goes to vertex position i.
var uvOrder = [8][4]int{
	{0, 1, 2, 3}, // no flags
	{2, 0, 3, 1}, // D only (90° CW + H flip)
	{2, 3, 0, 1}, // V flip
	{3, 2, 1, 0}, // V+D (90° CCW)
	{1, 0, 3, 2}, // H flip
	{0, 2, 1, 3}, // H+D (90° CW)
	{3, 2, 1, 0}, // H+V
	{1, 3, 0, 2}, // H+V+D (90° CW + V flip)
}

// TileMapViewport is a scene graph node that manages a viewport into a tilemap.
// It owns an ordered list of TileMapLayer (tile data) and supports interleaved
// regular Node containers for entities (sandwich layers).
type TileMapViewport struct {
	node *Node // container node in the scene graph

	// Tile dimensions in pixels.
	TileWidth  int
	TileHeight int

	// MaxZoomOut is the minimum zoom level the developer expects. Controls
	// buffer sizing. Default 1.0 (no zoom out). E.g. 0.5 means the
	// viewport may show 2x as many tiles.
	MaxZoomOut float64

	// MarginTiles is the number of extra tiles beyond the viewport edge to
	// keep buffered. Prevents pop-in during fast pans. Default 2.
	MarginTiles int

	// Camera binding. nil means use scene.Cameras()[0].
	camera *Camera

	// Direct access to tile layers (avoids type-checking children each frame).
	layers []*TileMapLayer

	// Animation elapsed time in milliseconds.
	animElapsed int
}

// TileMapLayer is a single layer of tile data. It stores the raw integer grid
// and owns a geometry buffer that is rebuilt on tile boundary crossings and
// transformed each frame.
type TileMapLayer struct {
	node   *Node    // container node in the scene graph (created by AddTileLayer)
	data   []uint32 // row-major tile GIDs, len = width * height
	width  int      // map width in tiles
	height int      // map height in tiles

	// Geometry buffer  -  preallocated to max visible tiles.
	vertices  []ebiten.Vertex // 4 vertices per tile, len = bufferCapacity * 4
	indices   []uint16        // 6 indices per tile, len = bufferCapacity * 6
	tileCount int             // number of active (non-empty) tiles in the buffer

	// World-space positions for each tile slot (used for CPU transform each frame).
	worldX []float32 // per-tile-slot world X (len = bufferCapacity)
	worldY []float32 // per-tile-slot world Y (len = bufferCapacity)

	// Tile region lookup: GID -> TextureRegion. Populated from atlas/tileset data.
	regions []TextureRegion // indexed by GID (after masking flags)

	// Tracking which grid columns/rows the buffer currently covers.
	bufStartCol int
	bufStartRow int
	bufCols     int
	bufRows     int
	bufDirty    bool // force rebuild on next update

	// Animation definitions for this layer's tileset.
	anims map[uint32][]AnimFrame // base GID -> animation frames (nil if no animations)

	// The atlas page (ebiten.Image) used for DrawTriangles.
	// All tiles in a layer must come from the same atlas page.
	atlasImage *ebiten.Image

	// Parent viewport reference (for accessing tile dimensions, camera, etc.)
	viewport *TileMapViewport
}

// NewTileMapViewport creates a new tilemap viewport node with the given tile
// dimensions. The viewport is a container node that should be added to the
// scene graph.
func NewTileMapViewport(name string, tileWidth, tileHeight int) *TileMapViewport {
	v := &TileMapViewport{
		node:        NewContainer(name),
		TileWidth:   tileWidth,
		TileHeight:  tileHeight,
		MaxZoomOut:  1.0,
		MarginTiles: 2,
	}
	v.node.OnUpdate = v.update
	return v
}

// Node returns the underlying scene graph node for this viewport.
func (v *TileMapViewport) Node() *Node {
	return v.node
}

// SetCamera binds this viewport to a specific camera. If nil, defaults to
// scene.Cameras()[0] during update.
func (v *TileMapViewport) SetCamera(cam *Camera) {
	v.camera = cam
}

// AddTileLayer creates a tile layer with the given map data and adds it as a
// child of the viewport. All regions must come from the provided atlas image.
// Returns the layer for further configuration (RenderLayer, animations, etc.).
func (v *TileMapViewport) AddTileLayer(name string, w, h int, data []uint32, regions []TextureRegion, atlasImage *ebiten.Image) *TileMapLayer {
	layer := &TileMapLayer{
		node:        NewContainer(name),
		data:        data,
		width:       w,
		height:      h,
		regions:     regions,
		atlasImage:  atlasImage,
		viewport:    v,
		bufDirty:    true,
		bufStartCol: -1, // force initial rebuild
		bufStartRow: -1,
	}

	// Set up customEmit on the layer's node.
	layer.node.customEmit = layer.emitCommands

	v.node.AddChild(layer.node)
	v.layers = append(v.layers, layer)
	return layer
}

// AddChild adds a regular node container as a child of the viewport
// (for sandwich layers: NPCs, items, effects, etc.).
func (v *TileMapViewport) AddChild(child *Node) {
	v.node.AddChild(child)
}

// SetTile updates a single tile in the given layer. If the tile is currently
// visible in the buffer, its vertex UVs are updated immediately.
func (l *TileMapLayer) SetTile(col, row int, newGID uint32) {
	if col < 0 || col >= l.width || row < 0 || row >= l.height {
		return
	}
	l.data[row*l.width+col] = newGID

	// If tile is within the current buffer range, rebuild.
	if col >= l.bufStartCol && col < l.bufStartCol+l.bufCols &&
		row >= l.bufStartRow && row < l.bufStartRow+l.bufRows {
		l.bufDirty = true
	}
}

// SetData replaces the entire tile data array and forces a full buffer rebuild.
func (l *TileMapLayer) SetData(data []uint32, w, h int) {
	l.data = data
	l.width = w
	l.height = h
	l.InvalidateBuffer()
}

// InvalidateBuffer forces a full buffer rebuild on the next frame.
func (l *TileMapLayer) InvalidateBuffer() {
	l.bufDirty = true
	l.bufStartCol = -1
	l.bufStartRow = -1
}

// SetAnimations sets the animation definitions for this layer.
// The map is keyed by base GID (no flag bits).
func (l *TileMapLayer) SetAnimations(anims map[uint32][]AnimFrame) {
	l.anims = anims
}

// Node returns the layer's scene graph node.
func (l *TileMapLayer) Node() *Node {
	return l.node
}

// update is the per-frame update callback registered on the viewport node.
func (v *TileMapViewport) update(dt float64) {
	cam := v.camera
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
		if !layer.node.Visible {
			continue
		}

		// Compute buffer dimensions based on viewport and zoom.
		// +2 accounts for partial tiles visible at both edges when the
		// camera is not tile-aligned.
		bufCols := int(math.Ceil(cam.Viewport.Width/tw/zoom)) + 2 + 2*v.MarginTiles
		bufRows := int(math.Ceil(cam.Viewport.Height/th/zoom)) + 2 + 2*v.MarginTiles

		// Compute visible tile range.
		startCol := int(math.Floor(bounds.X/tw)) - v.MarginTiles
		startRow := int(math.Floor(bounds.Y/th)) - v.MarginTiles

		// Clamp to valid range.
		if startCol < 0 {
			startCol = 0
		}
		if startRow < 0 {
			startRow = 0
		}
		endCol := startCol + bufCols - 1
		endRow := startRow + bufRows - 1
		if endCol >= layer.width {
			endCol = layer.width - 1
			startCol = endCol - bufCols + 1
			if startCol < 0 {
				startCol = 0
			}
		}
		if endRow >= layer.height {
			endRow = layer.height - 1
			startRow = endRow - bufRows + 1
			if startRow < 0 {
				startRow = 0
			}
		}

		// Ensure buffer is large enough.
		layer.ensureBuffer(bufCols, bufRows)

		// Check if tile boundary crossed or dirty.
		if layer.bufDirty || startCol != layer.bufStartCol || startRow != layer.bufStartRow {
			layer.rebuildBuffer(startCol, startRow, bufCols, bufRows)
		}
	}

	// Advance animation timer.
	dtMs := int(dt * 1000)
	if dtMs > 0 {
		v.animElapsed += dtMs
		v.updateAnimations()
	}
}

// ensureBuffer grows the geometry buffer if needed.
func (l *TileMapLayer) ensureBuffer(cols, rows int) {
	cap := cols * rows
	if cap <= len(l.worldX) {
		return
	}

	l.worldX = make([]float32, cap)
	l.worldY = make([]float32, cap)
	l.vertices = make([]ebiten.Vertex, cap*4)

	// Build index buffer (topology never changes).
	l.indices = make([]uint16, cap*6)
	for i := 0; i < cap; i++ {
		base := uint16(i * 4)
		off := i * 6
		l.indices[off+0] = base + 0
		l.indices[off+1] = base + 1
		l.indices[off+2] = base + 2
		l.indices[off+3] = base + 1
		l.indices[off+4] = base + 3
		l.indices[off+5] = base + 2
	}
}

// rebuildBuffer fills the vertex buffer with tile data for the given range.
func (l *TileMapLayer) rebuildBuffer(startCol, startRow, bufCols, bufRows int) {
	l.bufStartCol = startCol
	l.bufStartRow = startRow
	l.bufCols = bufCols
	l.bufRows = bufRows
	l.bufDirty = false

	tw := float32(l.viewport.TileWidth)
	th := float32(l.viewport.TileHeight)

	tileCount := 0
	for br := 0; br < bufRows; br++ {
		row := startRow + br
		if row < 0 || row >= l.height {
			continue
		}
		rowOffset := row * l.width
		for bc := 0; bc < bufCols; bc++ {
			col := startCol + bc
			if col < 0 || col >= l.width {
				continue
			}

			gid := l.data[rowOffset+col]
			if gid == 0 {
				continue // empty tile
			}

			flags := gid & tileFlagMask
			tileID := gid &^ tileFlagMask

			if int(tileID) >= len(l.regions) {
				continue // invalid GID
			}
			region := l.regions[tileID]

			// Store world position for per-frame transform.
			l.worldX[tileCount] = float32(col) * tw
			l.worldY[tileCount] = float32(row) * th

			// Set UV coordinates from TextureRegion with flip flags.
			setTileUVs(l.vertices[tileCount*4:], region, flags, tw, th)

			tileCount++
		}
	}
	l.tileCount = tileCount
}

// setTileUVs sets the UV (SrcX/SrcY) coordinates for 4 vertices of a tile,
// applying flip flags via the lookup table.
func setTileUVs(verts []ebiten.Vertex, region TextureRegion, flags uint32, tw, th float32) {
	// Source UV corners from the TextureRegion.
	sx := float32(region.X)
	sy := float32(region.Y)
	sw := float32(region.Width)
	sh := float32(region.Height)

	// The four UV corners: TL(0), TR(1), BL(2), BR(3).
	uvX := [4]float32{sx, sx + sw, sx, sx + sw}
	uvY := [4]float32{sy, sy, sy + sh, sy + sh}

	// Look up UV reorder based on flip flags.
	flagIdx := 0
	if flags&tileFlipH != 0 {
		flagIdx |= 4
	}
	if flags&tileFlipV != 0 {
		flagIdx |= 2
	}
	if flags&tileFlipD != 0 {
		flagIdx |= 1
	}
	order := uvOrder[flagIdx]

	// Vertex 0 = top-left, 1 = top-right, 2 = bottom-left, 3 = bottom-right.
	verts[0].SrcX = uvX[order[0]]
	verts[0].SrcY = uvY[order[0]]
	verts[1].SrcX = uvX[order[1]]
	verts[1].SrcY = uvY[order[1]]
	verts[2].SrcX = uvX[order[2]]
	verts[2].SrcY = uvY[order[2]]
	verts[3].SrcX = uvX[order[3]]
	verts[3].SrcY = uvY[order[3]]
}

// emitCommands is the customEmit callback for a TileMapLayer node.
// It transforms vertex positions from world to screen space and emits
// CommandTilemap commands into the scene's command pipeline.
func (l *TileMapLayer) emitCommands(s *Scene, treeOrder *int) {
	if l.tileCount == 0 || l.atlasImage == nil {
		return
	}

	// Get the accumulated color from the node's parent chain.
	r := float32(l.node.Color.R * l.node.worldAlpha)
	g := float32(l.node.Color.G * l.node.worldAlpha)
	b := float32(l.node.Color.B * l.node.worldAlpha)
	a := float32(l.node.worldAlpha)

	// Premultiply.
	cr := r * a
	cg := g * a
	cb := b * a

	tw := float32(l.viewport.TileWidth)
	th := float32(l.viewport.TileHeight)

	// View transform for world-to-screen conversion.
	vt := s.viewTransform
	va, vb := float32(vt[0]), float32(vt[1])
	vc, vd := float32(vt[2]), float32(vt[3])
	vtx, vty := float32(vt[4]), float32(vt[5])

	// Compute tile screen dimensions (for axis-aligned cameras).
	tileScreenW := va * tw
	tileScreenH := vd * th

	// Transform all active tile vertices.
	for i := 0; i < l.tileCount; i++ {
		wx := l.worldX[i]
		wy := l.worldY[i]

		// Apply view transform.
		screenX := va*wx + vc*wy + vtx
		screenY := vb*wx + vd*wy + vty

		vi := i * 4
		// Top-left
		l.vertices[vi+0].DstX = screenX
		l.vertices[vi+0].DstY = screenY
		// Top-right
		l.vertices[vi+1].DstX = screenX + tileScreenW
		l.vertices[vi+1].DstY = screenY
		// Bottom-left
		l.vertices[vi+2].DstX = screenX
		l.vertices[vi+2].DstY = screenY + tileScreenH
		// Bottom-right
		l.vertices[vi+3].DstX = screenX + tileScreenW
		l.vertices[vi+3].DstY = screenY + tileScreenH

		// Apply tint color.
		l.vertices[vi+0].ColorR = cr
		l.vertices[vi+0].ColorG = cg
		l.vertices[vi+0].ColorB = cb
		l.vertices[vi+0].ColorA = a
		l.vertices[vi+1].ColorR = cr
		l.vertices[vi+1].ColorG = cg
		l.vertices[vi+1].ColorB = cb
		l.vertices[vi+1].ColorA = a
		l.vertices[vi+2].ColorR = cr
		l.vertices[vi+2].ColorG = cg
		l.vertices[vi+2].ColorB = cb
		l.vertices[vi+2].ColorA = a
		l.vertices[vi+3].ColorR = cr
		l.vertices[vi+3].ColorG = cg
		l.vertices[vi+3].ColorB = cb
		l.vertices[vi+3].ColorA = a
	}

	// Emit commands, splitting at maxTilesPerDraw boundaries.
	totalTiles := l.tileCount
	for offset := 0; offset < totalTiles; offset += maxTilesPerDraw {
		end := offset + maxTilesPerDraw
		if end > totalTiles {
			end = totalTiles
		}
		batchTiles := end - offset

		*treeOrder++
		s.commands = append(s.commands, RenderCommand{
			Type:         CommandTilemap,
			RenderLayer:  l.node.RenderLayer,
			GlobalOrder:  l.node.GlobalOrder,
			treeOrder:    *treeOrder,
			BlendMode:    l.node.BlendMode,
			tilemapVerts: l.vertices[offset*4 : end*4],
			tilemapInds:  l.indices[:batchTiles*6],
			tilemapImage: l.atlasImage,
		})
	}
}

// updateAnimations scans layers for animated tiles and updates their UVs.
func (v *TileMapViewport) updateAnimations() {
	for _, layer := range v.layers {
		if layer.anims == nil || !layer.node.Visible {
			continue
		}

		tw := float32(v.TileWidth)
		th := float32(v.TileHeight)

		for i := 0; i < layer.tileCount; i++ {
			// Recover the grid position from world coordinates.
			col := int(layer.worldX[i] / tw)
			row := int(layer.worldY[i] / th)
			if col < 0 || col >= layer.width || row < 0 || row >= layer.height {
				continue
			}

			gid := layer.data[row*layer.width+col]
			if gid == 0 {
				continue
			}
			flags := gid & tileFlagMask
			baseGID := gid &^ tileFlagMask

			frames, ok := layer.anims[baseGID]
			if !ok || len(frames) == 0 {
				continue
			}

			// Determine current animation frame.
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

			if int(currentGID) < len(layer.regions) {
				region := layer.regions[currentGID]
				setTileUVs(layer.vertices[i*4:], region, flags, tw, th)
			}
		}
	}
}
