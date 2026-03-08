package mesh

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// NewPolygon creates an untextured polygon mesh from the given vertices.
func NewPolygon(name string, points []types.Vec2) *node.Node {
	white := EnsureWhitePixel()
	verts, inds := BuildPolygonFan(points, false, nil)
	return NewMeshFn(name, white, verts, inds)
}

// NewRegularPolygon creates an untextured regular polygon with the given number
// of sides, centered at the origin with the specified radius.
func NewRegularPolygon(name string, sides int, radius float64) *node.Node {
	if sides < 3 {
		sides = 3
	}
	pts := make([]types.Vec2, sides)
	for i := range pts {
		angle := float64(i)*2*math.Pi/float64(sides) - math.Pi/2
		pts[i] = types.Vec2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		}
	}
	return NewPolygon(name, pts)
}

// NewStar creates an untextured star polygon.
func NewStar(name string, outerR, innerR float64, points int) *node.Node {
	if points < 2 {
		points = 2
	}
	pts := make([]types.Vec2, points*2)
	for i := range pts {
		a := -math.Pi/2 + float64(i)*math.Pi/float64(points)
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		pts[i] = types.Vec2{X: math.Cos(a) * r, Y: math.Sin(a) * r}
	}
	return NewPolygon(name, pts)
}

// NewPolygonTextured creates a textured polygon mesh.
func NewPolygonTextured(name string, img *ebiten.Image, points []types.Vec2) *node.Node {
	verts, inds := BuildPolygonFan(points, true, img)
	return NewMeshFn(name, img, verts, inds)
}

// SetPolygonPoints updates the polygon's vertices.
func SetPolygonPoints(n *node.Node, points []types.Vec2) {
	textured := false
	var img *ebiten.Image
	if n.Mesh.Image != nil && n.Mesh.Image != EnsureWhitePixel() {
		textured = true
		img = n.Mesh.Image
	}
	verts, inds := BuildPolygonFan(points, textured, img)

	if cap(n.Mesh.Vertices) >= len(verts) {
		n.Mesh.Vertices = n.Mesh.Vertices[:len(verts)]
		copy(n.Mesh.Vertices, verts)
	} else {
		n.Mesh.Vertices = verts
	}
	if cap(n.Mesh.Indices) >= len(inds) {
		n.Mesh.Indices = n.Mesh.Indices[:len(inds)]
		copy(n.Mesh.Indices, inds)
	} else {
		n.Mesh.Indices = inds
	}

	n.Mesh.AabbDirty = true
}

// BuildPolygonFan generates vertices and indices for an arbitrary simple polygon.
func BuildPolygonFan(points []types.Vec2, textured bool, img *ebiten.Image) ([]ebiten.Vertex, []uint16) {
	n := len(points)
	if n < 3 {
		return nil, nil
	}

	verts := make([]ebiten.Vertex, n)
	inds := EarClipIndices(points)

	var minX, minY, maxX, maxY float64
	var imgW, imgH float64
	if textured && img != nil {
		minX, minY = points[0].X, points[0].Y
		maxX, maxY = minX, minY
		for i := 1; i < n; i++ {
			if points[i].X < minX {
				minX = points[i].X
			}
			if points[i].X > maxX {
				maxX = points[i].X
			}
			if points[i].Y < minY {
				minY = points[i].Y
			}
			if points[i].Y > maxY {
				maxY = points[i].Y
			}
		}
		b := img.Bounds()
		imgW = float64(b.Dx())
		imgH = float64(b.Dy())
	}

	for i, p := range points {
		v := &verts[i]
		v.DstX = float32(p.X)
		v.DstY = float32(p.Y)
		v.ColorR = 1
		v.ColorG = 1
		v.ColorB = 1
		v.ColorA = 1

		if textured && img != nil {
			bbW := maxX - minX
			bbH := maxY - minY
			var u, vv float64
			if bbW > 0 {
				u = (p.X - minX) / bbW * imgW
			}
			if bbH > 0 {
				vv = (p.Y - minY) / bbH * imgH
			}
			v.SrcX = float32(u)
			v.SrcY = float32(vv)
		} else {
			v.SrcX = 0.5
			v.SrcY = 0.5
		}
	}

	return verts, inds
}

// EarClipIndices triangulates a simple polygon using ear-clipping.
func EarClipIndices(pts []types.Vec2) []uint16 {
	n := len(pts)
	if n < 3 {
		return nil
	}
	if n == 3 {
		return []uint16{0, 1, 2}
	}

	var area float64
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	ccw := area > 0

	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}

	result := make([]uint16, 0, (n-2)*3)
	for len(idx) > 3 {
		m := len(idx)
		earFound := false
		for i := 0; i < m; i++ {
			a := idx[(i+m-1)%m]
			b := idx[i]
			c := idx[(i+1)%m]

			cr := Poly2DCross(pts[a], pts[b], pts[c])
			if !((ccw && cr > 0) || (!ccw && cr < 0)) {
				continue
			}

			inside := false
			for j := 0; j < m; j++ {
				if j == (i+m-1)%m || j == i || j == (i+1)%m {
					continue
				}
				if PtInTriangle(pts[idx[j]], pts[a], pts[b], pts[c]) {
					inside = true
					break
				}
			}
			if inside {
				continue
			}

			result = append(result, uint16(a), uint16(b), uint16(c))
			idx = append(idx[:i], idx[i+1:]...)
			earFound = true
			break
		}
		if !earFound {
			for j := 1; j < len(idx)-1; j++ {
				result = append(result, uint16(idx[0]), uint16(idx[j]), uint16(idx[j+1]))
			}
			return result
		}
	}
	return append(result, uint16(idx[0]), uint16(idx[1]), uint16(idx[2]))
}

// Poly2DCross returns (a-o) × (b-o). Positive = CCW turn.
func Poly2DCross(o, a, b types.Vec2) float64 {
	return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
}

// PtInTriangle reports whether p lies inside triangle (a, b, c).
func PtInTriangle(p, a, b, c types.Vec2) bool {
	d1 := Poly2DCross(a, b, p)
	d2 := Poly2DCross(b, c, p)
	d3 := Poly2DCross(c, a, p)
	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(hasNeg && hasPos)
}
