package mesh

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

func init() {
	// Wire function pointer needed by NewRope, NewPolygon, NewDistortionGrid.
	NewMeshFn = func(name string, img *ebiten.Image, verts []ebiten.Vertex, inds []uint16) *node.Node {
		n := node.NewNode(name, types.NodeTypeMesh)
		n.Mesh = &node.MeshData{
			Image:    img,
			Vertices: verts,
			Indices:  inds,
		}
		return n
	}
}

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

// --- Rope ---

func TestRopeBevelJoinMode(t *testing.T) {
	// L-shaped path: bevel mode should not scale normals at the join.
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4, JoinMode: RopeJoinBevel})
	n := r.Node()
	if len(n.Mesh.Vertices) != 6 {
		t.Errorf("vertices = %d, want 6", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 12 {
		t.Errorf("indices = %d, want 12", len(n.Mesh.Indices))
	}
}

func TestRopeVertexAndIndexCounts(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 20, Y: 0}, {X: 30, Y: 0}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4})
	n := r.Node()
	// 4 points -> 8 vertices, 3 segments -> 18 indices
	if len(n.Mesh.Vertices) != 8 {
		t.Errorf("vertices = %d, want 8", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 18 {
		t.Errorf("indices = %d, want 18", len(n.Mesh.Indices))
	}
}

func TestRopeTwoPoints(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4})
	n := r.Node()
	// 2 points -> 4 vertices, 1 segment -> 6 indices
	if len(n.Mesh.Vertices) != 4 {
		t.Errorf("vertices = %d, want 4", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 6 {
		t.Errorf("indices = %d, want 6", len(n.Mesh.Indices))
	}

	// For horizontal segment left->right (dx=10,dy=0), left-perpendicular is (0, 1).
	if !approxEqual(float64(n.Mesh.Vertices[0].DstY), 2, 0.01) {
		t.Errorf("top vertex Y = %f, want 2", n.Mesh.Vertices[0].DstY)
	}
	if !approxEqual(float64(n.Mesh.Vertices[1].DstY), -2, 0.01) {
		t.Errorf("bottom vertex Y = %f, want -2", n.Mesh.Vertices[1].DstY)
	}
}

func TestRopeUVTiling(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 20, Y: 0}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4})
	n := r.Node()
	if !approxEqual(float64(n.Mesh.Vertices[0].SrcX), 0, 0.01) {
		t.Errorf("SrcX[0] = %f, want 0", n.Mesh.Vertices[0].SrcX)
	}
	if !approxEqual(float64(n.Mesh.Vertices[2].SrcX), 10, 0.01) {
		t.Errorf("SrcX[2] = %f, want 10", n.Mesh.Vertices[2].SrcX)
	}
	if !approxEqual(float64(n.Mesh.Vertices[4].SrcX), 20, 0.01) {
		t.Errorf("SrcX[4] = %f, want 20", n.Mesh.Vertices[4].SrcX)
	}
}

func TestRopeSetPointsReusesBuffer(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 20, Y: 0}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4})
	n := r.Node()
	vertCap := cap(n.Mesh.Vertices)
	indCap := cap(n.Mesh.Indices)

	// Set fewer points - should not reallocate.
	r.SetPoints([]types.Vec2{{X: 0, Y: 0}, {X: 5, Y: 0}})
	if cap(n.Mesh.Vertices) != vertCap {
		t.Errorf("vertex cap changed from %d to %d", vertCap, cap(n.Mesh.Vertices))
	}
	if cap(n.Mesh.Indices) != indCap {
		t.Errorf("index cap changed from %d to %d", indCap, cap(n.Mesh.Indices))
	}
	if len(n.Mesh.Vertices) != 4 {
		t.Errorf("vertices = %d, want 4", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 6 {
		t.Errorf("indices = %d, want 6", len(n.Mesh.Indices))
	}
}

func TestRopeSinglePointNoMesh(t *testing.T) {
	r := NewRope("rope", nil, []types.Vec2{{X: 0, Y: 0}}, RopeConfig{Width: 4})
	n := r.Node()
	if len(n.Mesh.Vertices) != 0 || len(n.Mesh.Indices) != 0 {
		t.Error("single point rope should have no vertices/indices")
	}
}

func TestRopeInvalidatesMeshAABB(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}}
	r := NewRope("rope", nil, points, RopeConfig{Width: 4})
	n := r.Node()
	RecomputeMeshAABB(n)
	if n.Mesh.AabbDirty {
		t.Error("AABB should not be dirty after recompute")
	}

	r.SetPoints([]types.Vec2{{X: 0, Y: 0}, {X: 20, Y: 0}})
	if !n.Mesh.AabbDirty {
		t.Error("AABB should be dirty after SetPoints")
	}
}

// --- Rope Curve Modes ---

func TestRopeUpdateLine(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveLine,
		Segments:  10,
		Start:     &start,
		End:       &end,
	})
	r.Update()
	n := r.Node()

	// 11 points -> 22 vertices, 10 segments -> 60 indices.
	if len(n.Mesh.Vertices) != 22 {
		t.Errorf("vertices = %d, want 22", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 60 {
		t.Errorf("indices = %d, want 60", len(n.Mesh.Indices))
	}

	// First point at start, last point at end.
	if !approxEqual(float64(n.Mesh.Vertices[0].DstX), 0, 0.1) {
		t.Errorf("first vertex X = %f, want ~0", n.Mesh.Vertices[0].DstX)
	}
	lastV := len(n.Mesh.Vertices) - 2 // top vertex of last point
	if !approxEqual(float64(n.Mesh.Vertices[lastV].DstX), 100, 0.1) {
		t.Errorf("last vertex X = %f, want ~100", n.Mesh.Vertices[lastV].DstX)
	}
}

func TestRopeUpdateCatenary(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveCatenary,
		Segments:  20,
		Start:     &start,
		End:       &end,
		Sag:       50,
	})
	r.Update()

	// Midpoint should sag downward (positive Y).
	midIdx := 10 * 2 // vertex index for midpoint (top vertex)
	midY := float64(r.Node().Mesh.Vertices[midIdx].DstY)
	if midY < 40 {
		t.Errorf("catenary midpoint Y = %f, want > 40 (sag=50)", midY)
	}
}

func TestRopeUpdateQuadBezier(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	ctrl := types.Vec2{X: 50, Y: 80}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveQuadBezier,
		Segments:  10,
		Start:     &start,
		End:       &end,
		Controls:  [2]*types.Vec2{&ctrl},
	})
	r.Update()
	n := r.Node()

	if len(n.Mesh.Vertices) != 22 {
		t.Errorf("vertices = %d, want 22", len(n.Mesh.Vertices))
	}

	// Midpoint (t=0.5): 0.25*0 + 2*0.25*50 + 0.25*100 = 50 (X)
	midIdx := 5 * 2
	if !approxEqual(float64(n.Mesh.Vertices[midIdx].DstX), 50, 5) {
		t.Errorf("quad bezier midpoint X = %f, want ~50", n.Mesh.Vertices[midIdx].DstX)
	}
}

func TestRopeUpdateCubicBezier(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	c1 := types.Vec2{X: 33, Y: 60}
	c2 := types.Vec2{X: 66, Y: 60}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveCubicBezier,
		Segments:  10,
		Start:     &start,
		End:       &end,
		Controls:  [2]*types.Vec2{&c1, &c2},
	})
	r.Update()
	n := r.Node()

	if len(n.Mesh.Vertices) != 22 {
		t.Errorf("vertices = %d, want 22", len(n.Mesh.Vertices))
	}

	// Endpoints should match Start and End (within halfW perpendicular offset).
	if !approxEqual(float64(n.Mesh.Vertices[0].DstX), 0, 3) {
		t.Errorf("start X = %f, want ~0", n.Mesh.Vertices[0].DstX)
	}
	lastV := len(n.Mesh.Vertices) - 2
	if !approxEqual(float64(n.Mesh.Vertices[lastV].DstX), 100, 3) {
		t.Errorf("end X = %f, want ~100", n.Mesh.Vertices[lastV].DstX)
	}
}

func TestRopeUpdateWave(t *testing.T) {
	start := types.Vec2{X: 0, Y: 100}
	end := types.Vec2{X: 200, Y: 100}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveWave,
		Segments:  40,
		Start:     &start,
		End:       &end,
		Amplitude: 30,
		Frequency: 2,
	})
	r.Update()

	// For a horizontal line with wave, some vertices should deviate from Y=100.
	maxDev := 0.0
	for i := 0; i < len(r.Node().Mesh.Vertices); i += 2 {
		dev := math.Abs(float64(r.Node().Mesh.Vertices[i].DstY) - 100)
		if dev > maxDev {
			maxDev = dev
		}
	}
	if maxDev < 20 {
		t.Errorf("wave max deviation = %f, want > 20 (amplitude=30)", maxDev)
	}
}

func TestRopeUpdateCustom(t *testing.T) {
	customPts := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 10}, {X: 20, Y: 0}}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveCustom,
		PointsFunc: func(buf []types.Vec2) []types.Vec2 {
			return customPts
		},
	})
	r.Update()
	n := r.Node()

	// 3 points -> 6 vertices.
	if len(n.Mesh.Vertices) != 6 {
		t.Errorf("vertices = %d, want 6", len(n.Mesh.Vertices))
	}
}

func TestRopeUpdateDefaultSegments(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveLine,
		Start:     &start,
		End:       &end,
		// Segments is 0 -> should default to 20.
	})
	r.Update()
	n := r.Node()

	// 21 points -> 42 vertices.
	if len(n.Mesh.Vertices) != 42 {
		t.Errorf("vertices = %d, want 42 (default 20 segments)", len(n.Mesh.Vertices))
	}
}

func TestRopeUpdateBufferReuse(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveLine,
		Segments:  10,
		Start:     &start,
		End:       &end,
	})
	r.Update()

	n := r.Node()
	vertCapBefore := cap(n.Mesh.Vertices)
	indCapBefore := cap(n.Mesh.Indices)

	r.Config().Segments = 5
	r.Update()

	if cap(n.Mesh.Vertices) != vertCapBefore {
		t.Errorf("vertex backing array reallocated: was %d, now %d", vertCapBefore, cap(n.Mesh.Vertices))
	}
	if cap(n.Mesh.Indices) != indCapBefore {
		t.Errorf("index backing array reallocated: was %d, now %d", indCapBefore, cap(n.Mesh.Indices))
	}
}

func TestRopeUpdateByRefMutation(t *testing.T) {
	start := types.Vec2{X: 0, Y: 0}
	end := types.Vec2{X: 100, Y: 0}
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveLine,
		Segments:  10,
		Start:     &start,
		End:       &end,
	})
	r.Update()

	// Mutate the bound Vec2 directly - Update() should pick it up.
	end.X = 200
	r.Update()

	lastV := len(r.Node().Mesh.Vertices) - 2
	if !approxEqual(float64(r.Node().Mesh.Vertices[lastV].DstX), 200, 1) {
		t.Errorf("after ref mutation, end X = %f, want ~200", r.Node().Mesh.Vertices[lastV].DstX)
	}
}

func TestRopeUpdateNilStartEnd(t *testing.T) {
	r := NewRope("rope", nil, nil, RopeConfig{
		Width:     4,
		CurveMode: RopeCurveLine,
		Segments:  10,
		// Start and End are nil.
	})
	r.Update()
	n := r.Node()

	// Should be a no-op - no vertices generated.
	if len(n.Mesh.Vertices) != 0 {
		t.Errorf("nil start/end should produce no vertices, got %d", len(n.Mesh.Vertices))
	}
}

// --- DistortionGrid ---

func TestDistortionGridDimensions(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 4, 3)
	n := g.Node()
	// 4 cols, 3 rows -> (5*4)=20 vertices, (4*3*6)=72 indices
	if len(n.Mesh.Vertices) != 20 {
		t.Errorf("vertices = %d, want 20", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 72 {
		t.Errorf("indices = %d, want 72", len(n.Mesh.Indices))
	}
}

func TestDistortionGridSetVertex(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 2, 2)
	n := g.Node()
	g.SetVertex(1, 1, 5.0, 10.0)
	idx := 1*3 + 1 // row=1, vcols=3, col=1
	v := n.Mesh.Vertices[idx]
	if !approxEqual(float64(v.DstX), 5, 0.01) || !approxEqual(float64(v.DstY), 10, 0.01) {
		t.Errorf("SetVertex(1,1, 5,10): got (%f,%f), want (5,10)", v.DstX, v.DstY)
	}
}

func TestDistortionGridSetAllVertices(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 2, 2)
	n := g.Node()

	g.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
		return float64(col) * 3, float64(row) * 7
	})

	// Check vertex at (2, 1): rest=(0,0) since nil image, offset=(2*3, 1*7)=(6,7)
	vcols := 3
	idx := 1*vcols + 2
	v := n.Mesh.Vertices[idx]
	if !approxEqual(float64(v.DstX), 6, 0.01) || !approxEqual(float64(v.DstY), 7, 0.01) {
		t.Errorf("SetAllVertices: vertex(%d) = (%f,%f), want (6,7)", idx, v.DstX, v.DstY)
	}
}

func TestDistortionGridReset(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 2, 2)
	n := g.Node()
	g.SetVertex(0, 0, 99, 99)
	g.Reset()

	// After reset, vertex (0,0) should be back at rest position (0,0).
	v := n.Mesh.Vertices[0]
	if !approxEqual(float64(v.DstX), 0, 0.01) || !approxEqual(float64(v.DstY), 0, 0.01) {
		t.Errorf("Reset: vertex(0) = (%f,%f), want (0,0)", v.DstX, v.DstY)
	}
}

func TestDistortionGridUVStability(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 2, 2)
	n := g.Node()
	// Store original UVs.
	origUVs := make([][2]float32, len(n.Mesh.Vertices))
	for i, v := range n.Mesh.Vertices {
		origUVs[i] = [2]float32{v.SrcX, v.SrcY}
	}

	// Deform and reset.
	g.SetVertex(1, 1, 10, 10)
	g.Reset()

	// UVs should not change.
	for i, v := range n.Mesh.Vertices {
		if v.SrcX != origUVs[i][0] || v.SrcY != origUVs[i][1] {
			t.Errorf("UV changed at %d: (%f,%f) vs (%f,%f)", i, v.SrcX, v.SrcY, origUVs[i][0], origUVs[i][1])
		}
	}
}

func TestDistortionGridInvalidatesAABB(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 2, 2)
	n := g.Node()
	RecomputeMeshAABB(n)
	g.SetVertex(0, 0, 5, 5)
	if !n.Mesh.AabbDirty {
		t.Error("SetVertex should invalidate AABB")
	}

	RecomputeMeshAABB(n)
	g.SetAllVertices(func(c, r int, rx, ry float64) (float64, float64) { return 0, 0 })
	if !n.Mesh.AabbDirty {
		t.Error("SetAllVertices should invalidate AABB")
	}

	RecomputeMeshAABB(n)
	g.Reset()
	if !n.Mesh.AabbDirty {
		t.Error("Reset should invalidate AABB")
	}
}

func TestDistortionGridColsRows(t *testing.T) {
	g := NewDistortionGrid("grid", nil, 5, 3)
	if g.Cols() != 5 {
		t.Errorf("Cols = %d, want 5", g.Cols())
	}
	if g.Rows() != 3 {
		t.Errorf("Rows = %d, want 3", g.Rows())
	}
}

// --- Polygon ---

func TestPolygonFanTriangulation(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	n := NewPolygon("poly", points)
	// 4 vertices, 2 triangles -> 6 indices
	if len(n.Mesh.Vertices) != 4 {
		t.Errorf("vertices = %d, want 4", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 6 {
		t.Errorf("indices = %d, want 6", len(n.Mesh.Indices))
	}
	// Ear-clip: first ear is vertex 0 (triangle 3,0,1), remainder is 1,2,3
	expected := []uint16{3, 0, 1, 1, 2, 3}
	for i, idx := range expected {
		if n.Mesh.Indices[i] != idx {
			t.Errorf("Indices[%d] = %d, want %d", i, n.Mesh.Indices[i], idx)
		}
	}
}

func TestPolygonTriangle(t *testing.T) {
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}
	n := NewPolygon("tri", points)
	if len(n.Mesh.Vertices) != 3 {
		t.Errorf("vertices = %d, want 3", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 3 {
		t.Errorf("indices = %d, want 3", len(n.Mesh.Indices))
	}
}

func TestPolygonPentagon(t *testing.T) {
	// 5-sided polygon -> 5 verts, 3 triangles -> 9 indices
	var points []types.Vec2
	for i := 0; i < 5; i++ {
		angle := float64(i) * 2 * math.Pi / 5
		points = append(points, types.Vec2{X: math.Cos(angle) * 10, Y: math.Sin(angle) * 10})
	}
	n := NewPolygon("pent", points)
	if len(n.Mesh.Vertices) != 5 {
		t.Errorf("vertices = %d, want 5", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 9 {
		t.Errorf("indices = %d, want 9", len(n.Mesh.Indices))
	}
}

func TestPolygonWhitePixelSharing(t *testing.T) {
	a := NewPolygon("a", []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}})
	b := NewPolygon("b", []types.Vec2{{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 10, Y: 20}})
	if a.Mesh.Image != b.Mesh.Image {
		t.Error("untextured polygons should share the same white pixel image")
	}
}

func TestPolygonUntexturedUV(t *testing.T) {
	n := NewPolygon("poly", []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}})
	// Untextured: all UVs should be at center of white pixel (0.5, 0.5).
	for i, v := range n.Mesh.Vertices {
		if v.SrcX != 0.5 || v.SrcY != 0.5 {
			t.Errorf("vertex %d UV = (%f,%f), want (0.5,0.5)", i, v.SrcX, v.SrcY)
		}
	}
}

func TestPolygonTooFewPoints(t *testing.T) {
	n := NewPolygon("tiny", []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}})
	if len(n.Mesh.Vertices) != 0 || len(n.Mesh.Indices) != 0 {
		t.Error("polygon with <3 points should have no vertices/indices")
	}
}

func TestSetPolygonPointsFunc(t *testing.T) {
	n := NewPolygon("poly", []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}})
	vertCap := cap(n.Mesh.Vertices)

	// Update with fewer points - should reuse backing array.
	SetPolygonPoints(n, []types.Vec2{{X: 0, Y: 0}, {X: 20, Y: 0}, {X: 10, Y: 20}})
	if len(n.Mesh.Vertices) != 3 {
		t.Errorf("vertices = %d, want 3", len(n.Mesh.Vertices))
	}
	if cap(n.Mesh.Vertices) != vertCap {
		t.Errorf("vertex backing array reallocated: was %d, now %d", vertCap, cap(n.Mesh.Vertices))
	}
	if !n.Mesh.AabbDirty {
		t.Error("SetPolygonPoints should invalidate AABB")
	}
}

func TestNewPolygonTextured(t *testing.T) {
	img := ebiten.NewImage(32, 32)
	points := []types.Vec2{{X: 0, Y: 0}, {X: 32, Y: 0}, {X: 32, Y: 32}, {X: 0, Y: 32}}
	n := NewPolygonTextured("texPoly", img, points)

	if len(n.Mesh.Vertices) != 4 {
		t.Errorf("vertices = %d, want 4", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 6 {
		t.Errorf("indices = %d, want 6", len(n.Mesh.Indices))
	}
	if n.Mesh.Image != img {
		t.Error("MeshImage should be the provided image")
	}

	// Bottom-right vertex should have UVs mapped to image dimensions.
	v := n.Mesh.Vertices[2] // point (32,32)
	if v.SrcX != 32 || v.SrcY != 32 {
		t.Errorf("textured UV at (32,32) = (%f,%f), want (32,32)", v.SrcX, v.SrcY)
	}
}

func TestSetPolygonPointsGrows(t *testing.T) {
	n := NewPolygon("poly", []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}})
	// Grow to 5 points - may need new backing array.
	points := []types.Vec2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 20, Y: 5}, {X: 15, Y: 15}, {X: 0, Y: 10}}
	SetPolygonPoints(n, points)
	if len(n.Mesh.Vertices) != 5 {
		t.Errorf("vertices = %d, want 5", len(n.Mesh.Vertices))
	}
	if len(n.Mesh.Indices) != 9 {
		t.Errorf("indices = %d, want 9", len(n.Mesh.Indices))
	}
}
