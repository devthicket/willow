package tilemap

import (
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTileFlags(t *testing.T) {
	gid := uint32(42) | TileFlipH | TileFlipV
	flags := gid & TileFlagMask
	tileID := gid &^ TileFlagMask

	if tileID != 42 {
		t.Errorf("tileID = %d, want 42", tileID)
	}
	if flags&TileFlipH == 0 {
		t.Error("should have flipH flag")
	}
	if flags&TileFlipV == 0 {
		t.Error("should have flipV flag")
	}
	if flags&TileFlipD != 0 {
		t.Error("should not have flipD flag")
	}
}

func TestSetTileUVs_NoFlags(t *testing.T) {
	verts := make([]ebiten.Vertex, 4)
	region := types.TextureRegion{X: 10, Y: 20, Width: 32, Height: 48}
	SetTileUVs(verts, region, 0, 32, 48)

	// TL
	if verts[0].SrcX != 10 || verts[0].SrcY != 20 {
		t.Errorf("TL = (%f, %f), want (10, 20)", verts[0].SrcX, verts[0].SrcY)
	}
	// TR
	if verts[1].SrcX != 42 || verts[1].SrcY != 20 {
		t.Errorf("TR = (%f, %f), want (42, 20)", verts[1].SrcX, verts[1].SrcY)
	}
	// BL
	if verts[2].SrcX != 10 || verts[2].SrcY != 68 {
		t.Errorf("BL = (%f, %f), want (10, 68)", verts[2].SrcX, verts[2].SrcY)
	}
	// BR
	if verts[3].SrcX != 42 || verts[3].SrcY != 68 {
		t.Errorf("BR = (%f, %f), want (42, 68)", verts[3].SrcX, verts[3].SrcY)
	}
}

func TestSetTileUVs_FlipH(t *testing.T) {
	verts := make([]ebiten.Vertex, 4)
	region := types.TextureRegion{X: 0, Y: 0, Width: 16, Height: 16}
	SetTileUVs(verts, region, TileFlipH, 16, 16)

	// H flip swaps left/right: order = {1, 0, 3, 2}
	if verts[0].SrcX != 16 { // was TR, now TL position
		t.Errorf("flipH TL.SrcX = %f, want 16", verts[0].SrcX)
	}
	if verts[1].SrcX != 0 { // was TL, now TR position
		t.Errorf("flipH TR.SrcX = %f, want 0", verts[1].SrcX)
	}
}

func TestAnimFrame(t *testing.T) {
	frames := []AnimFrame{
		{GID: 1, Duration: 100},
		{GID: 2, Duration: 200},
	}
	if frames[0].GID != 1 || frames[0].Duration != 100 {
		t.Error("frame 0 mismatch")
	}
	if frames[1].GID != 2 || frames[1].Duration != 200 {
		t.Error("frame 1 mismatch")
	}
}

func TestLayer_EnsureBuffer(t *testing.T) {
	l := &Layer{}
	l.EnsureBuffer(3, 3)
	if len(l.WorldX) != 9 {
		t.Errorf("worldX len = %d, want 9", len(l.WorldX))
	}
	if len(l.Vertices) != 36 {
		t.Errorf("vertices len = %d, want 36", len(l.Vertices))
	}
	if len(l.Indices) != 54 {
		t.Errorf("indices len = %d, want 54", len(l.Indices))
	}
}

func TestLayer_EnsureBuffer_NoShrink(t *testing.T) {
	l := &Layer{}
	l.EnsureBuffer(5, 5)
	oldLen := len(l.WorldX)
	l.EnsureBuffer(3, 3)
	if len(l.WorldX) != oldLen {
		t.Error("buffer should not shrink")
	}
}

func TestLayer_SetTile(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 9),
		Width:  3,
		Height: 3,
	}
	l.SetTile(1, 1, 42)
	if l.Data[4] != 42 {
		t.Errorf("data[4] = %d, want 42", l.Data[4])
	}
}

func TestLayer_SetTile_OutOfBounds(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 9),
		Width:  3,
		Height: 3,
	}
	l.SetTile(-1, 0, 1) // should not panic
	l.SetTile(3, 0, 1)  // should not panic
}

func TestLayer_InvalidateBuffer(t *testing.T) {
	l := &Layer{BufStartCol: 5, BufStartRow: 5}
	l.InvalidateBuffer()
	if !l.BufDirty {
		t.Error("should be dirty")
	}
	if l.BufStartCol != -1 || l.BufStartRow != -1 {
		t.Error("should reset start col/row")
	}
}

func TestMaxTilesPerDraw(t *testing.T) {
	if MaxTilesPerDraw != 16383 {
		t.Errorf("MaxTilesPerDraw = %d, want 16383", MaxTilesPerDraw)
	}
}

func TestUVOrder_Length(t *testing.T) {
	if len(UVOrder) != 8 {
		t.Errorf("UVOrder length = %d, want 8", len(UVOrder))
	}
	// Each entry should have 4 values
	for i, order := range UVOrder {
		if len(order) != 4 {
			t.Errorf("UVOrder[%d] length = %d, want 4", i, len(order))
		}
	}
}

// --- SetData tests ---

func TestLayer_SetData(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 4),
		Width:  2,
		Height: 2,
	}

	newData := []uint32{10, 20, 30, 40, 50, 60}
	l.SetData(newData, 3, 2)

	if l.Width != 3 || l.Height != 2 {
		t.Errorf("dimensions = %dx%d, want 3x2", l.Width, l.Height)
	}
	if len(l.Data) != 6 {
		t.Errorf("data len = %d, want 6", len(l.Data))
	}
	for i, want := range []uint32{10, 20, 30, 40, 50, 60} {
		if l.Data[i] != want {
			t.Errorf("data[%d] = %d, want %d", i, l.Data[i], want)
		}
	}
}

func TestLayer_SetData_InvalidatesBuffer(t *testing.T) {
	l := &Layer{
		Data:        make([]uint32, 4),
		Width:       2,
		Height:      2,
		BufStartCol: 5,
		BufStartRow: 5,
		BufDirty:    false,
	}

	l.SetData([]uint32{1, 2, 3, 4}, 2, 2)

	if !l.BufDirty {
		t.Error("SetData should mark buffer dirty")
	}
	if l.BufStartCol != -1 || l.BufStartRow != -1 {
		t.Error("SetData should reset BufStartCol/Row to -1")
	}
}

// --- SetTile additional tests ---

func TestLayer_SetTile_AllCorners(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 9),
		Width:  3,
		Height: 3,
	}

	l.SetTile(0, 0, 1) // top-left
	l.SetTile(2, 0, 2) // top-right
	l.SetTile(0, 2, 3) // bottom-left
	l.SetTile(2, 2, 4) // bottom-right

	if l.Data[0] != 1 {
		t.Errorf("top-left = %d, want 1", l.Data[0])
	}
	if l.Data[2] != 2 {
		t.Errorf("top-right = %d, want 2", l.Data[2])
	}
	if l.Data[6] != 3 {
		t.Errorf("bottom-left = %d, want 3", l.Data[6])
	}
	if l.Data[8] != 4 {
		t.Errorf("bottom-right = %d, want 4", l.Data[8])
	}
}

func TestLayer_SetTile_OutOfBounds_AllDirections(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 9),
		Width:  3,
		Height: 3,
	}
	// None of these should panic
	l.SetTile(-1, 0, 1)
	l.SetTile(0, -1, 1)
	l.SetTile(3, 0, 1)
	l.SetTile(0, 3, 1)
	l.SetTile(-1, -1, 1)
	l.SetTile(3, 3, 1)
	l.SetTile(100, 100, 1)

	// All tiles should remain zero
	for i, v := range l.Data {
		if v != 0 {
			t.Errorf("data[%d] = %d, want 0 (out-of-bounds writes should be no-ops)", i, v)
		}
	}
}

func TestLayer_SetTile_WithFlags(t *testing.T) {
	l := &Layer{
		Data:   make([]uint32, 4),
		Width:  2,
		Height: 2,
	}

	gid := uint32(5) | TileFlipH | TileFlipV
	l.SetTile(1, 0, gid)

	stored := l.Data[1]
	if stored != gid {
		t.Errorf("stored GID = 0x%08X, want 0x%08X", stored, gid)
	}
	tileID := stored &^ TileFlagMask
	if tileID != 5 {
		t.Errorf("tile ID = %d, want 5", tileID)
	}
}

func TestLayer_SetTile_DirtiesBufferWhenInView(t *testing.T) {
	l := &Layer{
		Data:        make([]uint32, 9),
		Width:       3,
		Height:      3,
		BufStartCol: 0,
		BufStartRow: 0,
		BufCols:     3,
		BufRows:     3,
		BufDirty:    false,
	}

	l.SetTile(1, 1, 42)
	if !l.BufDirty {
		t.Error("SetTile within buffer range should mark dirty")
	}
}

func TestLayer_SetTile_DoesNotDirtyBufferWhenOutOfView(t *testing.T) {
	l := &Layer{
		Data:        make([]uint32, 100),
		Width:       10,
		Height:      10,
		BufStartCol: 0,
		BufStartRow: 0,
		BufCols:     3,
		BufRows:     3,
		BufDirty:    false,
	}

	// Tile at (5, 5) is outside the buffer view of (0,0)-(2,2)
	l.SetTile(5, 5, 42)
	if l.BufDirty {
		t.Error("SetTile outside buffer range should not mark dirty")
	}
}

// --- EnsureBuffer additional tests ---

func TestLayer_EnsureBuffer_IndicesCorrect(t *testing.T) {
	l := &Layer{}
	l.EnsureBuffer(2, 2)

	// 4 tiles, each with 6 indices
	want := []uint16{
		0, 1, 2, 1, 3, 2, // tile 0
		4, 5, 6, 5, 7, 6, // tile 1
		8, 9, 10, 9, 11, 10, // tile 2
		12, 13, 14, 13, 15, 14, // tile 3
	}
	if len(l.Indices) != len(want) {
		t.Fatalf("indices len = %d, want %d", len(l.Indices), len(want))
	}
	for i, w := range want {
		if l.Indices[i] != w {
			t.Errorf("indices[%d] = %d, want %d", i, l.Indices[i], w)
		}
	}
}

func TestLayer_EnsureBuffer_GrowsFromExisting(t *testing.T) {
	l := &Layer{}
	l.EnsureBuffer(2, 2) // 4 tiles
	if len(l.WorldX) != 4 {
		t.Fatalf("initial worldX len = %d, want 4", len(l.WorldX))
	}

	l.EnsureBuffer(4, 4) // 16 tiles
	if len(l.WorldX) != 16 {
		t.Errorf("grown worldX len = %d, want 16", len(l.WorldX))
	}
	if len(l.Vertices) != 64 {
		t.Errorf("grown vertices len = %d, want 64", len(l.Vertices))
	}
	if len(l.Indices) != 96 {
		t.Errorf("grown indices len = %d, want 96", len(l.Indices))
	}
}

func TestLayer_EnsureBuffer_SingleTile(t *testing.T) {
	l := &Layer{}
	l.EnsureBuffer(1, 1)
	if len(l.WorldX) != 1 {
		t.Errorf("worldX len = %d, want 1", len(l.WorldX))
	}
	if len(l.Vertices) != 4 {
		t.Errorf("vertices len = %d, want 4", len(l.Vertices))
	}
	if len(l.Indices) != 6 {
		t.Errorf("indices len = %d, want 6", len(l.Indices))
	}
}

// --- RebuildBuffer tests ---

func newTestLayer(w, h int, data []uint32, regions []types.TextureRegion) *Layer {
	v := &Viewport{
		TileWidth:  16,
		TileHeight: 16,
	}
	return &Layer{
		Data:        data,
		Width:       w,
		Height:      h,
		Regions:     regions,
		Viewport_:   v,
		BufDirty:    true,
		BufStartCol: -1,
		BufStartRow: -1,
	}
}

func TestLayer_RebuildBuffer_BasicTiles(t *testing.T) {
	regions := []types.TextureRegion{
		{}, // GID 0 = empty
		{X: 0, Y: 0, Width: 16, Height: 16},
		{X: 16, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{
		1, 2, 0,
		0, 1, 2,
		2, 0, 1,
	}
	l := newTestLayer(3, 3, data, regions)
	l.EnsureBuffer(3, 3)
	l.RebuildBuffer(0, 0, 3, 3)

	// 6 non-zero tiles
	if l.TileCount != 6 {
		t.Errorf("TileCount = %d, want 6", l.TileCount)
	}
	if l.BufDirty {
		t.Error("BufDirty should be false after rebuild")
	}
	if l.BufStartCol != 0 || l.BufStartRow != 0 {
		t.Error("BufStartCol/Row should match rebuild args")
	}
}

func TestLayer_RebuildBuffer_AllEmpty(t *testing.T) {
	regions := []types.TextureRegion{{}}
	data := []uint32{0, 0, 0, 0}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(0, 0, 2, 2)

	if l.TileCount != 0 {
		t.Errorf("TileCount = %d, want 0 for all-empty layer", l.TileCount)
	}
}

func TestLayer_RebuildBuffer_AllFilled(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{1, 1, 1, 1}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(0, 0, 2, 2)

	if l.TileCount != 4 {
		t.Errorf("TileCount = %d, want 4", l.TileCount)
	}
}

func TestLayer_RebuildBuffer_WorldPositions(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{1, 1, 1, 1}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(0, 0, 2, 2)

	// Tile (0,0) -> world (0,0), tile (1,0) -> world (16,0)
	// Tile (0,1) -> world (0,16), tile (1,1) -> world (16,16)
	expectedX := []float32{0, 16, 0, 16}
	expectedY := []float32{0, 0, 16, 16}

	for i := 0; i < l.TileCount; i++ {
		if l.WorldX[i] != expectedX[i] || l.WorldY[i] != expectedY[i] {
			t.Errorf("tile %d world pos = (%f, %f), want (%f, %f)",
				i, l.WorldX[i], l.WorldY[i], expectedX[i], expectedY[i])
		}
	}
}

func TestLayer_RebuildBuffer_PartialView(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	// 4x4 map, all tile 1
	data := make([]uint32, 16)
	for i := range data {
		data[i] = 1
	}
	l := newTestLayer(4, 4, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(1, 1, 2, 2) // view from col=1, row=1, 2x2

	if l.TileCount != 4 {
		t.Errorf("TileCount = %d, want 4", l.TileCount)
	}
	// First tile should be at world (16, 16)
	if l.WorldX[0] != 16 || l.WorldY[0] != 16 {
		t.Errorf("first tile world pos = (%f, %f), want (16, 16)", l.WorldX[0], l.WorldY[0])
	}
}

func TestLayer_RebuildBuffer_SkipsOutOfRangeGID(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	// GID 99 is beyond len(regions)
	data := []uint32{1, 99, 0, 1}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(0, 0, 2, 2)

	// Only GID 1 tiles should be counted (2 of them), GID 99 skipped
	if l.TileCount != 2 {
		t.Errorf("TileCount = %d, want 2 (should skip out-of-range GID)", l.TileCount)
	}
}

func TestLayer_RebuildBuffer_WithFlipFlags(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	flippedGID := uint32(1) | TileFlipH
	data := []uint32{flippedGID, 0, 0, 0}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(2, 2)
	l.RebuildBuffer(0, 0, 2, 2)

	if l.TileCount != 1 {
		t.Fatalf("TileCount = %d, want 1", l.TileCount)
	}

	// With TileFlipH, UVOrder index 4 = {1, 0, 3, 2} means TL gets TR's UV
	v := l.Vertices[0:4]
	if v[0].SrcX != 16 {
		t.Errorf("flipped tile TL.SrcX = %f, want 16", v[0].SrcX)
	}
	if v[1].SrcX != 0 {
		t.Errorf("flipped tile TR.SrcX = %f, want 0", v[1].SrcX)
	}
}

func TestLayer_RebuildBuffer_ClampToLayerBounds(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{1, 1, 1, 1}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(5, 5)
	// Request buffer larger than layer: bufCols=5 but layer is only 2 wide
	l.RebuildBuffer(0, 0, 5, 5)

	// Should only produce tiles for the 2x2 area
	if l.TileCount != 4 {
		t.Errorf("TileCount = %d, want 4 (should clamp to layer bounds)", l.TileCount)
	}
}

func TestLayer_RebuildBuffer_NegativeStartClamps(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{1, 1, 1, 1}
	l := newTestLayer(2, 2, data, regions)
	l.EnsureBuffer(4, 4)
	// Negative start positions: rows/cols before 0 should be skipped
	l.RebuildBuffer(-1, -1, 4, 4)

	if l.TileCount != 4 {
		t.Errorf("TileCount = %d, want 4 (negative start should skip out-of-bounds)", l.TileCount)
	}
}

func TestLayer_RebuildBuffer_UVsSetCorrectly(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 32, Y: 64, Width: 16, Height: 16},
	}
	data := []uint32{1}
	l := newTestLayer(1, 1, data, regions)
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)

	if l.TileCount != 1 {
		t.Fatalf("TileCount = %d, want 1", l.TileCount)
	}

	v := l.Vertices[0:4]
	// No flags: order {0,1,2,3} -> TL, TR, BL, BR
	if v[0].SrcX != 32 || v[0].SrcY != 64 {
		t.Errorf("TL UV = (%f, %f), want (32, 64)", v[0].SrcX, v[0].SrcY)
	}
	if v[1].SrcX != 48 || v[1].SrcY != 64 {
		t.Errorf("TR UV = (%f, %f), want (48, 64)", v[1].SrcX, v[1].SrcY)
	}
	if v[2].SrcX != 32 || v[2].SrcY != 80 {
		t.Errorf("BL UV = (%f, %f), want (32, 80)", v[2].SrcX, v[2].SrcY)
	}
	if v[3].SrcX != 48 || v[3].SrcY != 80 {
		t.Errorf("BR UV = (%f, %f), want (48, 80)", v[3].SrcX, v[3].SrcY)
	}
}

// --- SetTileUVs additional tests ---

func TestSetTileUVs_FlipV(t *testing.T) {
	verts := make([]ebiten.Vertex, 4)
	region := types.TextureRegion{X: 0, Y: 0, Width: 16, Height: 16}
	SetTileUVs(verts, region, TileFlipV, 16, 16)

	// V flip: UVOrder[2] = {2, 3, 0, 1} -> TL gets BL, TR gets BR, BL gets TL, BR gets TR
	if verts[0].SrcY != 16 {
		t.Errorf("flipV TL.SrcY = %f, want 16", verts[0].SrcY)
	}
	if verts[2].SrcY != 0 {
		t.Errorf("flipV BL.SrcY = %f, want 0", verts[2].SrcY)
	}
}

func TestSetTileUVs_FlipD(t *testing.T) {
	verts := make([]ebiten.Vertex, 4)
	region := types.TextureRegion{X: 0, Y: 0, Width: 16, Height: 16}
	SetTileUVs(verts, region, TileFlipD, 16, 16)

	// D flip: UVOrder[1] = {2, 0, 3, 1}
	// vertex 0 gets UV[2] = BL = (0, 16)
	if verts[0].SrcX != 0 || verts[0].SrcY != 16 {
		t.Errorf("flipD TL = (%f, %f), want (0, 16)", verts[0].SrcX, verts[0].SrcY)
	}
	// vertex 1 gets UV[0] = TL = (0, 0)
	if verts[1].SrcX != 0 || verts[1].SrcY != 0 {
		t.Errorf("flipD TR = (%f, %f), want (0, 0)", verts[1].SrcX, verts[1].SrcY)
	}
}

func TestSetTileUVs_FlipHV(t *testing.T) {
	verts := make([]ebiten.Vertex, 4)
	region := types.TextureRegion{X: 0, Y: 0, Width: 16, Height: 16}
	SetTileUVs(verts, region, TileFlipH|TileFlipV, 16, 16)

	// H+V: UVOrder[6] = {3, 2, 1, 0} -> 180 degree rotation
	// vertex 0 gets UV[3] = BR = (16, 16)
	if verts[0].SrcX != 16 || verts[0].SrcY != 16 {
		t.Errorf("flipHV TL = (%f, %f), want (16, 16)", verts[0].SrcX, verts[0].SrcY)
	}
	// vertex 3 gets UV[0] = TL = (0, 0)
	if verts[3].SrcX != 0 || verts[3].SrcY != 0 {
		t.Errorf("flipHV BR = (%f, %f), want (0, 0)", verts[3].SrcX, verts[3].SrcY)
	}
}

func TestSetTileUVs_AllFlagCombinations(t *testing.T) {
	region := types.TextureRegion{X: 0, Y: 0, Width: 16, Height: 16}
	flagCombinations := []uint32{
		0,
		TileFlipD,
		TileFlipV,
		TileFlipV | TileFlipD,
		TileFlipH,
		TileFlipH | TileFlipD,
		TileFlipH | TileFlipV,
		TileFlipH | TileFlipV | TileFlipD,
	}

	for i, flags := range flagCombinations {
		verts := make([]ebiten.Vertex, 4)
		// Should not panic for any combination
		SetTileUVs(verts, region, flags, 16, 16)

		// All UV values should be within the region bounds
		for vi := 0; vi < 4; vi++ {
			if verts[vi].SrcX < 0 || verts[vi].SrcX > 16 {
				t.Errorf("flag combo %d, vert %d: SrcX=%f out of bounds", i, vi, verts[vi].SrcX)
			}
			if verts[vi].SrcY < 0 || verts[vi].SrcY > 16 {
				t.Errorf("flag combo %d, vert %d: SrcY=%f out of bounds", i, vi, verts[vi].SrcY)
			}
		}
	}
}

// --- InvalidateBuffer additional tests ---

func TestLayer_InvalidateBuffer_MultipleCalls(t *testing.T) {
	l := &Layer{BufStartCol: 5, BufStartRow: 5}
	l.InvalidateBuffer()
	l.InvalidateBuffer() // second call should be fine
	if !l.BufDirty {
		t.Error("should still be dirty")
	}
	if l.BufStartCol != -1 || l.BufStartRow != -1 {
		t.Error("should still have -1 start col/row")
	}
}

// --- SetAnimations tests ---

func TestLayer_SetAnimations(t *testing.T) {
	l := &Layer{}
	anims := map[uint32][]AnimFrame{
		1: {{GID: 1, Duration: 100}, {GID: 2, Duration: 200}},
		3: {{GID: 3, Duration: 50}, {GID: 4, Duration: 50}, {GID: 5, Duration: 50}},
	}
	l.SetAnimations(anims)

	if l.Anims == nil {
		t.Fatal("Anims should not be nil after SetAnimations")
	}
	if len(l.Anims[1]) != 2 {
		t.Errorf("Anims[1] len = %d, want 2", len(l.Anims[1]))
	}
	if len(l.Anims[3]) != 3 {
		t.Errorf("Anims[3] len = %d, want 3", len(l.Anims[3]))
	}
}

func TestLayer_SetAnimations_Nil(t *testing.T) {
	l := &Layer{
		Anims: map[uint32][]AnimFrame{1: {{GID: 1, Duration: 100}}},
	}
	l.SetAnimations(nil)
	if l.Anims != nil {
		t.Error("Anims should be nil after SetAnimations(nil)")
	}
}

// --- UpdateAnimations tests ---

func TestViewport_UpdateAnimations_AdvancesFrame(t *testing.T) {
	regions := []types.TextureRegion{
		{},                                   // GID 0
		{X: 0, Y: 0, Width: 16, Height: 16},  // GID 1
		{X: 16, Y: 0, Width: 16, Height: 16}, // GID 2
		{X: 32, Y: 0, Width: 16, Height: 16}, // GID 3
	}
	v := &Viewport{TileWidth: 16, TileHeight: 16}
	l := &Layer{
		Node_:     &node.Node{Visible_: true},
		Data:      []uint32{1},
		Width:     1,
		Height:    1,
		Regions:   regions,
		Viewport_: v,
		Anims: map[uint32][]AnimFrame{
			1: {
				{GID: 1, Duration: 100},
				{GID: 2, Duration: 100},
				{GID: 3, Duration: 100},
			},
		},
	}
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)
	v.layers = []*Layer{l}

	// At elapsed=0, should show frame 0 (GID 1)
	v.animElapsed = 0
	v.UpdateAnimations()
	if l.Vertices[0].SrcX != 0 {
		t.Errorf("frame 0: SrcX = %f, want 0", l.Vertices[0].SrcX)
	}

	// At elapsed=150, should show frame 1 (GID 2)
	v.animElapsed = 150
	v.UpdateAnimations()
	if l.Vertices[0].SrcX != 16 {
		t.Errorf("frame 1: SrcX = %f, want 16", l.Vertices[0].SrcX)
	}

	// At elapsed=250, should show frame 2 (GID 3)
	v.animElapsed = 250
	v.UpdateAnimations()
	if l.Vertices[0].SrcX != 32 {
		t.Errorf("frame 2: SrcX = %f, want 32", l.Vertices[0].SrcX)
	}
}

func TestViewport_UpdateAnimations_Loops(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
		{X: 16, Y: 0, Width: 16, Height: 16},
	}
	v := &Viewport{TileWidth: 16, TileHeight: 16}
	l := &Layer{
		Node_:     &node.Node{Visible_: true},
		Data:      []uint32{1},
		Width:     1,
		Height:    1,
		Regions:   regions,
		Viewport_: v,
		Anims: map[uint32][]AnimFrame{
			1: {
				{GID: 1, Duration: 100},
				{GID: 2, Duration: 100},
			},
		},
	}
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)
	v.layers = []*Layer{l}

	// Total duration = 200ms, at elapsed=250 -> 250 % 200 = 50 -> frame 0 (GID 1)
	v.animElapsed = 250
	v.UpdateAnimations()
	if l.Vertices[0].SrcX != 0 {
		t.Errorf("looped: SrcX = %f, want 0 (should be back to GID 1)", l.Vertices[0].SrcX)
	}
}

func TestViewport_UpdateAnimations_SkipsInvisibleLayer(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
		{X: 16, Y: 0, Width: 16, Height: 16},
	}
	v := &Viewport{TileWidth: 16, TileHeight: 16}
	l := &Layer{
		Node_:     &node.Node{Visible_: false},
		Data:      []uint32{1},
		Width:     1,
		Height:    1,
		Regions:   regions,
		Viewport_: v,
		Anims: map[uint32][]AnimFrame{
			1: {{GID: 1, Duration: 100}, {GID: 2, Duration: 100}},
		},
	}
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)
	v.layers = []*Layer{l}

	// Record UV before animation
	origSrcX := l.Vertices[0].SrcX

	v.animElapsed = 150
	v.UpdateAnimations()

	// UVs should not change since layer is invisible
	if l.Vertices[0].SrcX != origSrcX {
		t.Errorf("invisible layer: SrcX changed from %f to %f", origSrcX, l.Vertices[0].SrcX)
	}
}

func TestViewport_UpdateAnimations_NoAnimsIsNoOp(t *testing.T) {
	v := &Viewport{TileWidth: 16, TileHeight: 16}
	l := &Layer{
		Node_:     &node.Node{Visible_: true},
		Data:      []uint32{1},
		Width:     1,
		Height:    1,
		Regions:   []types.TextureRegion{{}, {X: 0, Y: 0, Width: 16, Height: 16}},
		Viewport_: v,
		Anims:     nil, // no animations
	}
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)
	v.layers = []*Layer{l}

	// Should not panic
	v.animElapsed = 100
	v.UpdateAnimations()
}

func TestViewport_UpdateAnimations_PreservesFlipFlags(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
		{X: 16, Y: 0, Width: 16, Height: 16},
	}
	v := &Viewport{TileWidth: 16, TileHeight: 16}
	flippedGID := uint32(1) | TileFlipH
	l := &Layer{
		Node_:     &node.Node{Visible_: true},
		Data:      []uint32{flippedGID},
		Width:     1,
		Height:    1,
		Regions:   regions,
		Viewport_: v,
		Anims: map[uint32][]AnimFrame{
			1: {
				{GID: 1, Duration: 100},
				{GID: 2, Duration: 100},
			},
		},
	}
	l.EnsureBuffer(1, 1)
	l.RebuildBuffer(0, 0, 1, 1)
	v.layers = []*Layer{l}

	// At elapsed=150, should show GID 2 with flipH applied
	v.animElapsed = 150
	v.UpdateAnimations()

	// GID 2 region starts at X=16, with flipH, TL should get TR UV (X=32)
	if l.Vertices[0].SrcX != 32 {
		t.Errorf("flipped animated tile TL.SrcX = %f, want 32", l.Vertices[0].SrcX)
	}
}

// --- Tile flag constant tests ---

func TestTileFlags_Individual(t *testing.T) {
	if TileFlipH != 1<<31 {
		t.Errorf("TileFlipH = 0x%08X, want 0x%08X", TileFlipH, uint32(1<<31))
	}
	if TileFlipV != 1<<30 {
		t.Errorf("TileFlipV = 0x%08X, want 0x%08X", TileFlipV, uint32(1<<30))
	}
	if TileFlipD != 1<<29 {
		t.Errorf("TileFlipD = 0x%08X, want 0x%08X", TileFlipD, uint32(1<<29))
	}
}

func TestTileFlagMask_CoversAllFlags(t *testing.T) {
	combined := TileFlipH | TileFlipV | TileFlipD
	if TileFlagMask != combined {
		t.Errorf("TileFlagMask = 0x%08X, want 0x%08X", TileFlagMask, combined)
	}
}

func TestTileFlagMask_PreservesTileID(t *testing.T) {
	// Max tile ID that fits below the flag bits (29 bits)
	maxID := uint32((1 << 29) - 1)
	gid := maxID | TileFlipH | TileFlipV | TileFlipD
	tileID := gid &^ TileFlagMask
	if tileID != maxID {
		t.Errorf("tileID = %d, want %d", tileID, maxID)
	}
}

// --- Layer dimensions tests ---

func TestLayer_DimensionsAfterSetData(t *testing.T) {
	l := &Layer{}
	l.SetData(make([]uint32, 20), 5, 4)
	if l.Width != 5 || l.Height != 4 {
		t.Errorf("dims = %dx%d, want 5x4", l.Width, l.Height)
	}
	if len(l.Data) != 20 {
		t.Errorf("data len = %d, want 20", len(l.Data))
	}
}

func TestLayer_TileCountReflectsNonZeroTiles(t *testing.T) {
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := []uint32{0, 1, 0, 0, 0, 1, 1, 0, 0}
	l := newTestLayer(3, 3, data, regions)
	l.EnsureBuffer(3, 3)
	l.RebuildBuffer(0, 0, 3, 3)

	if l.TileCount != 3 {
		t.Errorf("TileCount = %d, want 3", l.TileCount)
	}
}

// --- LateRebuildCheck tests ---

func TestLayer_LateRebuildCheck_NilCamera(t *testing.T) {
	v := &Viewport{
		TileWidth:  16,
		TileHeight: 16,
		camera:     nil,
	}
	l := &Layer{Viewport_: v}
	// Should not panic with nil camera
	l.LateRebuildCheck()
}

func TestLayer_LateRebuildCheck_NilCameraBoundsFn(t *testing.T) {
	v := &Viewport{
		TileWidth:      16,
		TileHeight:     16,
		camera:         "fake",
		CameraBoundsFn: nil,
	}
	l := &Layer{Viewport_: v}
	// Should not panic with nil CameraBoundsFn
	l.LateRebuildCheck()
}

type mockBoundsProvider struct {
	bounds         types.Rect
	viewportWidth  float64
	viewportHeight float64
}

func (m *mockBoundsProvider) VisibleBounds() types.Rect { return m.bounds }
func (m *mockBoundsProvider) ViewportWidth() float64    { return m.viewportWidth }
func (m *mockBoundsProvider) ViewportHeight() float64   { return m.viewportHeight }

func TestLayer_LateRebuildCheck_NoRebuildWhenMatching(t *testing.T) {
	mock := &mockBoundsProvider{
		bounds:         types.Rect{X: 0, Y: 0, Width: 320, Height: 240},
		viewportWidth:  320,
		viewportHeight: 240,
	}
	v := &Viewport{
		TileWidth:   16,
		TileHeight:  16,
		MarginTiles: 2,
		MaxZoomOut:  1.0,
		camera:      "cam",
		CameraBoundsFn: func(cam any) VisibleBoundsProvider {
			return mock
		},
	}
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	data := make([]uint32, 400)
	for i := range data {
		data[i] = 1
	}
	l := &Layer{
		Node_:       &node.Node{Visible_: true},
		Data:        data,
		Width:       20,
		Height:      20,
		Regions:     regions,
		Viewport_:   v,
		BufDirty:    true,
		BufStartCol: -1,
		BufStartRow: -1,
	}
	v.layers = []*Layer{l}

	// Do initial update to set buffer state
	v.update(0)

	savedStartCol := l.BufStartCol
	savedStartRow := l.BufStartRow

	// LateRebuildCheck with same camera should not change anything
	l.LateRebuildCheck()

	if l.BufStartCol != savedStartCol || l.BufStartRow != savedStartRow {
		t.Error("LateRebuildCheck should not rebuild when camera hasn't moved")
	}
}

func TestLayer_LateRebuildCheck_RebuildsWhenCameraMoved(t *testing.T) {
	mock := &mockBoundsProvider{
		bounds:         types.Rect{X: 0, Y: 0, Width: 160, Height: 160},
		viewportWidth:  160,
		viewportHeight: 160,
	}
	v := &Viewport{
		TileWidth:   16,
		TileHeight:  16,
		MarginTiles: 1,
		MaxZoomOut:  1.0,
		camera:      "cam",
		CameraBoundsFn: func(cam any) VisibleBoundsProvider {
			return mock
		},
	}
	regions := []types.TextureRegion{
		{},
		{X: 0, Y: 0, Width: 16, Height: 16},
	}
	// Large map: 100x100 so clamping won't normalize startCol back
	data := make([]uint32, 10000)
	for i := range data {
		data[i] = 1
	}
	l := &Layer{
		Node_:       &node.Node{Visible_: true},
		Data:        data,
		Width:       100,
		Height:      100,
		Regions:     regions,
		Viewport_:   v,
		BufDirty:    true,
		BufStartCol: -1,
		BufStartRow: -1,
	}
	v.layers = []*Layer{l}

	// Initial build
	v.update(0)
	savedStartCol := l.BufStartCol

	// Move camera far enough to change startCol
	mock.bounds = types.Rect{X: 320, Y: 0, Width: 160, Height: 160}

	l.LateRebuildCheck()

	if l.BufStartCol == savedStartCol {
		t.Error("LateRebuildCheck should rebuild when camera moves")
	}
}

func TestLayer_LateRebuildCheck_NilProviderReturn(t *testing.T) {
	v := &Viewport{
		TileWidth:  16,
		TileHeight: 16,
		camera:     "cam",
		CameraBoundsFn: func(cam any) VisibleBoundsProvider {
			return nil
		},
	}
	l := &Layer{Viewport_: v}
	// Should not panic when CameraBoundsFn returns nil
	l.LateRebuildCheck()
}
