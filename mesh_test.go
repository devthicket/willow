package willow

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- transformVertices ---

func TestTransformVerticesIdentity(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 10, DstY: 20, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 30, DstY: 40, SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 2)
	transformVertices(src, dst, identityTransform, Color{1, 1, 1, 1})

	if !approxEqual(float64(dst[0].DstX), 10, epsilon) || !approxEqual(float64(dst[0].DstY), 20, epsilon) {
		t.Errorf("identity: dst[0] = (%f,%f), want (10,20)", dst[0].DstX, dst[0].DstY)
	}
	if !approxEqual(float64(dst[1].DstX), 30, epsilon) || !approxEqual(float64(dst[1].DstY), 40, epsilon) {
		t.Errorf("identity: dst[1] = (%f,%f), want (30,40)", dst[1].DstX, dst[1].DstY)
	}
}

func TestTransformVerticesTranslation(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	// [a=1, b=0, c=0, d=1, tx=100, ty=200]
	transform := [6]float64{1, 0, 0, 1, 100, 200}
	transformVertices(src, dst, transform, Color{1, 1, 1, 1})

	if !approxEqual(float64(dst[0].DstX), 100, epsilon) || !approxEqual(float64(dst[0].DstY), 200, epsilon) {
		t.Errorf("translation: dst[0] = (%f,%f), want (100,200)", dst[0].DstX, dst[0].DstY)
	}
}

func TestTransformVerticesRotation90(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 1, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	// 90° CCW: a=cos90=0, b=sin90=1, c=-sin90=0... wait use our layout
	// [0]=a, [1]=b, [2]=c, [3]=d, [4]=tx, [5]=ty
	// newX = a*x + c*y + tx, newY = b*x + d*y + ty
	// Rotate 90° CCW: a=0, b=1, c=-1, d=0
	transform := [6]float64{0, 1, -1, 0, 0, 0}
	transformVertices(src, dst, transform, Color{1, 1, 1, 1})

	// (1,0) rotated 90° CCW → (0,1)
	if !approxEqual(float64(dst[0].DstX), 0, 0.001) || !approxEqual(float64(dst[0].DstY), 1, 0.001) {
		t.Errorf("rotation90: dst[0] = (%f,%f), want (0,1)", dst[0].DstX, dst[0].DstY)
	}
}

func TestTransformVerticesColorTint(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 0.8, ColorB: 0.5, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	// Tint with color {R:0.5, G:1.0, B:0.8, A:0.6}  -  alpha baked in
	tint := Color{0.5, 1.0, 0.8, 0.6}
	transformVertices(src, dst, identityTransform, tint)

	// ColorR = src.ColorR * tint.R * tint.A = 1.0 * 0.5 * 0.6 = 0.3
	// ColorG = src.ColorG * tint.G * tint.A = 0.8 * 1.0 * 0.6 = 0.48
	// ColorB = src.ColorB * tint.B * tint.A = 0.5 * 0.8 * 0.6 = 0.24
	// ColorA = src.ColorA * tint.A = 1.0 * 0.6 = 0.6
	if !approxEqual(float64(dst[0].ColorR), 0.3, 0.001) {
		t.Errorf("ColorR = %f, want 0.3", dst[0].ColorR)
	}
	if !approxEqual(float64(dst[0].ColorG), 0.48, 0.001) {
		t.Errorf("ColorG = %f, want 0.48", dst[0].ColorG)
	}
	if !approxEqual(float64(dst[0].ColorB), 0.24, 0.001) {
		t.Errorf("ColorB = %f, want 0.24", dst[0].ColorB)
	}
	if !approxEqual(float64(dst[0].ColorA), 0.6, 0.001) {
		t.Errorf("ColorA = %f, want 0.6", dst[0].ColorA)
	}
}

func TestTransformVerticesPreservesUV(t *testing.T) {
	src := []ebiten.Vertex{
		{DstX: 10, DstY: 20, SrcX: 0.25, SrcY: 0.75, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	dst := make([]ebiten.Vertex, 1)
	transform := [6]float64{2, 0, 0, 2, 50, 50}
	transformVertices(src, dst, transform, Color{1, 1, 1, 1})

	if dst[0].SrcX != 0.25 || dst[0].SrcY != 0.75 {
		t.Errorf("UV changed: got (%f,%f), want (0.25, 0.75)", dst[0].SrcX, dst[0].SrcY)
	}
}

func TestTransformVerticesNoDoubleAlpha(t *testing.T) {
	// Ensure alpha is not applied twice: once for premultiply and once for vertex alpha.
	// The tint Color already has worldAlpha baked in via Color.A * worldAlpha.
	src := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 0.5},
	}
	dst := make([]ebiten.Vertex, 1)
	tint := Color{1, 1, 1, 0.8} // worldAlpha already in A
	transformVertices(src, dst, identityTransform, tint)

	// ColorA = src.ColorA * tint.A = 0.5 * 0.8 = 0.4
	// ColorR = src.ColorR * tint.R * tint.A = 1.0 * 1.0 * 0.8 = 0.8
	if !approxEqual(float64(dst[0].ColorA), 0.4, 0.001) {
		t.Errorf("ColorA = %f, want 0.4", dst[0].ColorA)
	}
	if !approxEqual(float64(dst[0].ColorR), 0.8, 0.001) {
		t.Errorf("ColorR = %f, want 0.8 (premultiplied by tint alpha, not vertex alpha)", dst[0].ColorR)
	}
}

// --- computeMeshAABB ---

func TestComputeMeshAABBEmpty(t *testing.T) {
	aabb := computeMeshAABB(nil)
	if aabb.Width != 0 || aabb.Height != 0 {
		t.Errorf("empty AABB = %v, want zero", aabb)
	}
}

func TestComputeMeshAABBTriangle(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: 10, DstY: 20},
		{DstX: 50, DstY: 20},
		{DstX: 30, DstY: 60},
	}
	aabb := computeMeshAABB(verts)
	if !approxEqual(aabb.X, 10, epsilon) || !approxEqual(aabb.Y, 20, epsilon) {
		t.Errorf("AABB origin = (%f,%f), want (10,20)", aabb.X, aabb.Y)
	}
	if !approxEqual(aabb.Width, 40, epsilon) || !approxEqual(aabb.Height, 40, epsilon) {
		t.Errorf("AABB size = (%f,%f), want (40,40)", aabb.Width, aabb.Height)
	}
}

func TestComputeMeshAABBNegativeCoords(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: -10, DstY: -20},
		{DstX: 10, DstY: 20},
	}
	aabb := computeMeshAABB(verts)
	if !approxEqual(aabb.X, -10, epsilon) || !approxEqual(aabb.Y, -20, epsilon) {
		t.Errorf("AABB origin = (%f,%f), want (-10,-20)", aabb.X, aabb.Y)
	}
	if !approxEqual(aabb.Width, 20, epsilon) || !approxEqual(aabb.Height, 40, epsilon) {
		t.Errorf("AABB size = (%f,%f), want (20,40)", aabb.Width, aabb.Height)
	}
}

// --- ensureTransformedVerts ---

func TestEnsureTransformedVertsGrowsToHighWater(t *testing.T) {
	n := NewMesh("test", nil, make([]ebiten.Vertex, 10), nil)
	buf := ensureTransformedVerts(n)
	if len(buf) != 10 {
		t.Errorf("len = %d, want 10", len(buf))
	}
	cap1 := cap(n.transformedVerts)

	// Shrink vertices  -  buffer should not shrink.
	n.Vertices = n.Vertices[:5]
	buf = ensureTransformedVerts(n)
	if len(buf) != 5 {
		t.Errorf("len = %d, want 5", len(buf))
	}
	if cap(n.transformedVerts) != cap1 {
		t.Errorf("cap changed from %d to %d (should keep high-water)", cap1, cap(n.transformedVerts))
	}

	// Grow past high-water.
	n.Vertices = make([]ebiten.Vertex, 20)
	buf = ensureTransformedVerts(n)
	if len(buf) != 20 {
		t.Errorf("len = %d, want 20", len(buf))
	}
}

// --- InvalidateMeshAABB / recomputeMeshAABB ---

func TestMeshAABBDirtyOnNew(t *testing.T) {
	n := NewMesh("test", nil, []ebiten.Vertex{{DstX: 5, DstY: 10}}, nil)
	if !n.meshAABBDirty {
		t.Error("meshAABBDirty should be true after NewMesh")
	}

	n.recomputeMeshAABB()
	if n.meshAABBDirty {
		t.Error("meshAABBDirty should be false after recompute")
	}
	if !approxEqual(n.meshAABB.X, 5, epsilon) || !approxEqual(n.meshAABB.Y, 10, epsilon) {
		t.Errorf("AABB = %v, want origin (5,10)", n.meshAABB)
	}

	n.InvalidateMeshAABB()
	if !n.meshAABBDirty {
		t.Error("meshAABBDirty should be true after Invalidate")
	}
}

// --- Mesh culling ---

func TestMeshCullingWithOffset(t *testing.T) {
	// Mesh vertices centered around (500, 500), not at origin.
	verts := []ebiten.Vertex{
		{DstX: 490, DstY: 490},
		{DstX: 510, DstY: 490},
		{DstX: 510, DstY: 510},
		{DstX: 490, DstY: 510},
	}
	inds := []uint16{0, 1, 2, 0, 2, 3}
	n := NewMesh("m", nil, verts, inds)
	// Set identity world transform.
	n.worldTransform = identityTransform

	// Cull bounds that DON'T overlap (500 area).
	cullBounds := Rect{X: 0, Y: 0, Width: 100, Height: 100}
	if !shouldCull(n, n.worldTransform, cullBounds) {
		t.Error("mesh at (490-510) should be culled by bounds (0-100)")
	}

	// Cull bounds that DO overlap.
	cullBounds = Rect{X: 480, Y: 480, Width: 40, Height: 40}
	if shouldCull(n, n.worldTransform, cullBounds) {
		t.Error("mesh at (490-510) should NOT be culled by bounds (480-520)")
	}
}

// --- Mesh traverse emission ---

func TestMeshTraverseEmitsTransformedVerts(t *testing.T) {
	s := NewScene()
	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 10, DstY: 0, SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 10, DstY: 10, SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	inds := []uint16{0, 1, 2}
	n := NewMesh("m", nil, verts, inds)
	n.X = 50
	n.Y = 100
	s.Root().AddChild(n)

	traverseScene(s)

	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	cmd := &s.commands[0]
	if cmd.Type != CommandMesh {
		t.Fatalf("Type = %d, want CommandMesh", cmd.Type)
	}
	// Vertices should be translated by (50, 100).
	if !approxEqual(float64(cmd.meshVerts[0].DstX), 50, 0.01) {
		t.Errorf("meshVerts[0].DstX = %f, want 50", cmd.meshVerts[0].DstX)
	}
	if !approxEqual(float64(cmd.meshVerts[0].DstY), 100, 0.01) {
		t.Errorf("meshVerts[0].DstY = %f, want 100", cmd.meshVerts[0].DstY)
	}
	if !approxEqual(float64(cmd.meshVerts[1].DstX), 60, 0.01) {
		t.Errorf("meshVerts[1].DstX = %f, want 60", cmd.meshVerts[1].DstX)
	}
}

func TestMeshTraverseEmptySkipped(t *testing.T) {
	s := NewScene()
	n := NewMesh("empty", nil, nil, nil)
	s.Root().AddChild(n)

	traverseScene(s)

	if len(s.commands) != 0 {
		t.Errorf("empty mesh should emit 0 commands, got %d", len(s.commands))
	}
}

func TestMeshTraverseColorTint(t *testing.T) {
	s := NewScene()
	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	inds := []uint16{0}
	n := NewMesh("m", nil, verts, inds)
	n.Color = Color{0.5, 0.8, 1.0, 1.0}
	n.Alpha = 0.5
	s.Root().AddChild(n)

	traverseScene(s)

	if len(s.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.commands))
	}
	v := &s.commands[0].meshVerts[0]
	// worldAlpha = 1.0 * 0.5 = 0.5, tint = {0.5, 0.8, 1.0, 1.0*0.5=0.5}
	// ColorR = 1.0 * 0.5 * 0.5 = 0.25
	// ColorG = 1.0 * 0.8 * 0.5 = 0.40
	// ColorB = 1.0 * 1.0 * 0.5 = 0.50
	// ColorA = 1.0 * 0.5 = 0.50
	if !approxEqual(float64(v.ColorR), 0.25, 0.01) {
		t.Errorf("ColorR = %f, want 0.25", v.ColorR)
	}
	if !approxEqual(float64(v.ColorG), 0.40, 0.01) {
		t.Errorf("ColorG = %f, want 0.40", v.ColorG)
	}
	if !approxEqual(float64(v.ColorB), 0.50, 0.01) {
		t.Errorf("ColorB = %f, want 0.50", v.ColorB)
	}
	if !approxEqual(float64(v.ColorA), 0.50, 0.01) {
		t.Errorf("ColorA = %f, want 0.50", v.ColorA)
	}
}

// --- meshWorldAABBOffset ---

func TestMeshWorldAABBOffset(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: 100, DstY: 200},
		{DstX: 150, DstY: 200},
		{DstX: 150, DstY: 250},
		{DstX: 100, DstY: 250},
	}
	n := NewMesh("m", nil, verts, nil)
	n.worldTransform = identityTransform

	aabb := meshWorldAABBOffset(n, n.worldTransform)
	if !approxEqual(aabb.X, 100, epsilon) || !approxEqual(aabb.Y, 200, epsilon) {
		t.Errorf("AABB origin = (%f,%f), want (100,200)", aabb.X, aabb.Y)
	}
	if !approxEqual(aabb.Width, 50, epsilon) || !approxEqual(aabb.Height, 50, epsilon) {
		t.Errorf("AABB size = (%f,%f), want (50,50)", aabb.Width, aabb.Height)
	}
}

func TestMeshWorldAABBWithScale(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0},
		{DstX: 10, DstY: 0},
		{DstX: 10, DstY: 10},
	}
	n := NewMesh("m", nil, verts, nil)
	// Scale by 2.
	n.worldTransform = [6]float64{2, 0, 0, 2, 0, 0}

	aabb := meshWorldAABBOffset(n, n.worldTransform)
	if !approxEqual(aabb.Width, 20, epsilon) || !approxEqual(aabb.Height, 20, epsilon) {
		t.Errorf("scaled AABB size = (%f,%f), want (20,20)", aabb.Width, aabb.Height)
	}
}

// --- ensureWhitePixel ---

func TestEnsureWhitePixelSingleton(t *testing.T) {
	a := ensureWhitePixel()
	b := ensureWhitePixel()
	if a != b {
		t.Error("ensureWhitePixel should return same image")
	}
	bounds := a.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("white pixel size = %dx%d, want 1x1", bounds.Dx(), bounds.Dy())
	}
}

// --- nodeDimensions for mesh ---

func TestNodeDimensionsMesh(t *testing.T) {
	verts := []ebiten.Vertex{
		{DstX: -5, DstY: -10},
		{DstX: 15, DstY: 30},
	}
	n := NewMesh("m", nil, verts, nil)
	w, h := nodeDimensions(n)
	// AABB: X=-5, Y=-10, W=20, H=40
	if !approxEqual(w, 20, epsilon) {
		t.Errorf("width = %f, want 20", w)
	}
	if !approxEqual(h, 40, epsilon) {
		t.Errorf("height = %f, want 40", h)
	}
}

// --- Dispose cleans up ---

func TestMeshDisposeNilsTransformedVerts(t *testing.T) {
	verts := []ebiten.Vertex{{DstX: 0, DstY: 0}}
	n := NewMesh("m", nil, verts, []uint16{0})
	_ = ensureTransformedVerts(n) // allocate buffer
	if n.transformedVerts == nil {
		t.Fatal("transformedVerts should be allocated")
	}
	n.Dispose()
	if n.transformedVerts != nil {
		t.Error("transformedVerts should be nil after Dispose")
	}
}

// --- Benchmark ---

func BenchmarkTransformVertices1000(b *testing.B) {
	src := make([]ebiten.Vertex, 1000)
	for i := range src {
		src[i] = ebiten.Vertex{
			DstX: float32(i), DstY: float32(i * 2),
			SrcX: 0.5, SrcY: 0.5,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
	}
	dst := make([]ebiten.Vertex, 1000)
	transform := [6]float64{
		math.Cos(0.5), math.Sin(0.5), -math.Sin(0.5), math.Cos(0.5), 100, 200,
	}
	tint := Color{0.8, 0.9, 1.0, 0.7}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		transformVertices(src, dst, transform, tint)
	}
}
