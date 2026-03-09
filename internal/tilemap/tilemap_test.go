package tilemap

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/types"
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
