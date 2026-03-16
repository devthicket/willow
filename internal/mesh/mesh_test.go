package mesh

import (
	"testing"

	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestComputeMeshAABB_Empty(t *testing.T) {
	r := ComputeMeshAABB(nil)
	if r.Width != 0 || r.Height != 0 {
		t.Error("empty verts should return zero rect")
	}
}

func TestComputeMeshAABB(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: 10, DstY: 20},
		{DstX: 50, DstY: 80},
		{DstX: 30, DstY: 40},
	}
	r := ComputeMeshAABB(verts)
	if r.X != 10 || r.Y != 20 || r.Width != 40 || r.Height != 60 {
		t.Errorf("AABB = {%f, %f, %f, %f}, want {10, 20, 40, 60}", r.X, r.Y, r.Width, r.Height)
	}
}

func TestTransformVertices_Identity(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 10, DstY: 20, SrcX: 5, SrcY: 5, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	identity := [6]float64{1, 0, 0, 1, 0, 0}
	tint := types.ColorFromRGBA(1, 1, 1, 1)

	TransformVertices(src, dst, identity, tint)

	if dst[0].DstX != 10 || dst[0].DstY != 20 {
		t.Errorf("position = (%f, %f), want (10, 20)", dst[0].DstX, dst[0].DstY)
	}
	if dst[0].SrcX != 5 || dst[0].SrcY != 5 {
		t.Error("SrcX/SrcY should be preserved")
	}
}

func TestTransformVertices_Translate(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 0, DstY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	translate := [6]float64{1, 0, 0, 1, 100, 200}
	tint := types.ColorFromRGBA(1, 1, 1, 1)

	TransformVertices(src, dst, translate, tint)

	if dst[0].DstX != 100 || dst[0].DstY != 200 {
		t.Errorf("position = (%f, %f), want (100, 200)", dst[0].DstX, dst[0].DstY)
	}
}

func TestEarClipIndices_Triangle(t *testing.T) {
	pts := []types.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}
	inds := EarClipIndices(pts)
	if len(inds) != 3 {
		t.Errorf("indices count = %d, want 3", len(inds))
	}
}

func TestEarClipIndices_Quad(t *testing.T) {
	pts := []types.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}
	inds := EarClipIndices(pts)
	if len(inds) != 6 {
		t.Errorf("indices count = %d, want 6", len(inds))
	}
}

func TestEarClipIndices_TooFew(t *testing.T) {
	inds := EarClipIndices([]types.Vec2{{X: 0, Y: 0}, {X: 1, Y: 1}})
	if inds != nil {
		t.Error("should return nil for < 3 points")
	}
}

func TestPoly2DCross(t *testing.T) {
	o := types.Vec2{X: 0, Y: 0}
	a := types.Vec2{X: 1, Y: 0}
	b := types.Vec2{X: 0, Y: 1}
	c := Poly2DCross(o, a, b)
	if c != 1 {
		t.Errorf("cross = %f, want 1", c)
	}
}

func TestPtInTriangle(t *testing.T) {
	a := types.Vec2{X: 0, Y: 0}
	b := types.Vec2{X: 4, Y: 0}
	c := types.Vec2{X: 2, Y: 4}

	if !PtInTriangle(types.Vec2{X: 2, Y: 1}, a, b, c) {
		t.Error("point inside should return true")
	}
	if PtInTriangle(types.Vec2{X: 10, Y: 10}, a, b, c) {
		t.Error("point outside should return false")
	}
}

func TestPerpendicular(t *testing.T) {
	a := types.Vec2{X: 0, Y: 0}
	b := types.Vec2{X: 1, Y: 0}
	nx, ny := Perpendicular(a, b)
	if nx != 0 || ny != 1 {
		t.Errorf("perpendicular = (%f, %f), want (0, 1)", nx, ny)
	}
}

func TestBuildPolygonFan_Untextured(t *testing.T) {
	pts := []types.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}
	verts, inds := BuildPolygonFan(pts, false, nil)
	if len(verts) != 3 {
		t.Errorf("verts = %d, want 3", len(verts))
	}
	if len(inds) != 3 {
		t.Errorf("inds = %d, want 3", len(inds))
	}
	// Untextured should map to center of white pixel
	if verts[0].SrcX != 0.5 || verts[0].SrcY != 0.5 {
		t.Errorf("untextured UV = (%f, %f), want (0.5, 0.5)", verts[0].SrcX, verts[0].SrcY)
	}
}

func TestBuildPolygonFan_TooFew(t *testing.T) {
	verts, inds := BuildPolygonFan([]types.Vec2{{X: 0, Y: 0}}, false, nil)
	if verts != nil || inds != nil {
		t.Error("should return nil for < 3 points")
	}
}

func TestEnsureWhitePixel(t *testing.T) {
	img := EnsureWhitePixel()
	if img == nil {
		t.Error("should return non-nil image")
	}
	b := img.Bounds()
	if b.Dx() != 1 || b.Dy() != 1 {
		t.Errorf("size = %d×%d, want 1×1", b.Dx(), b.Dy())
	}
	// Subsequent call should return same instance
	img2 := EnsureWhitePixel()
	if img2 != img {
		t.Error("should return same singleton")
	}
}
