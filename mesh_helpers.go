package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Rope ---

// RopeJoinMode controls how segments join in a Rope mesh.
type RopeJoinMode uint8

const (
	// RopeJoinMiter extends segment corners to a sharp point.
	RopeJoinMiter RopeJoinMode = iota
	// RopeJoinBevel flattens corners by inserting extra vertices, avoiding spikes.
	RopeJoinBevel
)

// RopeCurveMode selects the curve algorithm used by Rope.Update().
type RopeCurveMode uint8

const (
	// RopeCurveLine draws a straight line between Start and End.
	RopeCurveLine RopeCurveMode = iota
	// RopeCurveCatenary simulates a drooping rope with gravity sag.
	RopeCurveCatenary
	// RopeCurveQuadBezier uses a quadratic Bézier with one control point.
	RopeCurveQuadBezier
	// RopeCurveCubicBezier uses a cubic Bézier with two control points.
	RopeCurveCubicBezier
	// RopeCurveWave produces a sinusoidal wave along the line.
	RopeCurveWave
	// RopeCurveCustom calls a user-provided PointsFunc callback.
	RopeCurveCustom
)

// RopeConfig configures a Rope mesh.
type RopeConfig struct {
	Width    float64
	JoinMode RopeJoinMode

	CurveMode RopeCurveMode
	Segments  int // number of subdivisions (default 20)

	// Endpoint positions (pointers). Update() dereferences these each call,
	// so you can bind them once and mutate the underlying Vec2 freely.
	Start *Vec2
	End   *Vec2

	// Catenary sag in pixels (downward droop).
	Sag float64

	// Bézier control points (pointers). Quadratic uses Controls[0]; cubic uses both.
	Controls [2]*Vec2

	// Wave parameters.
	Amplitude float64
	Frequency float64 // cycles along the rope length
	Phase     float64 // phase offset in radians

	// Custom callback. Receives a preallocated buffer; must return the slice to use.
	PointsFunc func(buf []Vec2) []Vec2
}

// Rope generates a ribbon/rope mesh that follows a polyline path.
type Rope struct {
	node   *Node
	config RopeConfig
	cumLen []float64 // preallocated cumulative length buffer
	ptsBuf []Vec2    // preallocated points buffer for Update()
}

// NewRope creates a rope mesh node that renders a textured ribbon along the given points.
// The image is tiled along the path (SrcX) and spans the full image height (SrcY).
func NewRope(name string, img *ebiten.Image, points []Vec2, cfg RopeConfig) (*Rope, *Node) {
	n := NewMesh(name, img, nil, nil)
	r := &Rope{node: n, config: cfg}
	r.SetPoints(points)
	return r, n
}

// Node returns the underlying mesh node.
func (r *Rope) Node() *Node {
	return r.node
}

// Config returns a pointer to the rope's configuration so callers can mutate
// fields directly before calling Update().
func (r *Rope) Config() *RopeConfig {
	return &r.config
}

// Update recomputes the rope's point path from the current RopeConfig and
// rebuilds the mesh. Call this in your update loop after changing the bound
// Vec2 values. Start and End must be non-nil (except for RopeCurveCustom).
func (r *Rope) Update() {
	if r.config.CurveMode != RopeCurveCustom {
		if r.config.Start == nil || r.config.End == nil {
			return
		}
	}

	segs := r.config.Segments
	if segs <= 0 {
		segs = 20
	}
	n := segs + 1

	// Grow ptsBuf to high-water mark.
	if cap(r.ptsBuf) < n {
		r.ptsBuf = make([]Vec2, n)
	}
	r.ptsBuf = r.ptsBuf[:n]

	switch r.config.CurveMode {
	case RopeCurveLine:
		s, e := *r.config.Start, *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			r.ptsBuf[i] = Vec2{
				X: s.X + (e.X-s.X)*t,
				Y: s.Y + (e.Y-s.Y)*t,
			}
		}

	case RopeCurveCatenary:
		s, e := *r.config.Start, *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			r.ptsBuf[i] = Vec2{
				X: s.X + (e.X-s.X)*t,
				Y: s.Y + (e.Y-s.Y)*t + r.config.Sag*math.Sin(math.Pi*t),
			}
		}

	case RopeCurveQuadBezier:
		if r.config.Controls[0] == nil {
			return
		}
		a, c, b := *r.config.Start, *r.config.Controls[0], *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			u := 1 - t
			r.ptsBuf[i] = Vec2{
				X: u*u*a.X + 2*u*t*c.X + t*t*b.X,
				Y: u*u*a.Y + 2*u*t*c.Y + t*t*b.Y,
			}
		}

	case RopeCurveCubicBezier:
		if r.config.Controls[0] == nil || r.config.Controls[1] == nil {
			return
		}
		a, c1, c2, b := *r.config.Start, *r.config.Controls[0], *r.config.Controls[1], *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			u := 1 - t
			u2 := u * u
			t2 := t * t
			r.ptsBuf[i] = Vec2{
				X: u2*u*a.X + 3*u2*t*c1.X + 3*u*t2*c2.X + t2*t*b.X,
				Y: u2*u*a.Y + 3*u2*t*c1.Y + 3*u*t2*c2.Y + t2*t*b.Y,
			}
		}

	case RopeCurveWave:
		s, e := *r.config.Start, *r.config.End
		dx := e.X - s.X
		dy := e.Y - s.Y
		ln := math.Sqrt(dx*dx + dy*dy)
		var px, py float64 // perpendicular unit vector
		if ln > 1e-10 {
			px = -dy / ln
			py = dx / ln
		}
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			off := r.config.Amplitude * math.Sin(r.config.Frequency*2*math.Pi*t+r.config.Phase)
			r.ptsBuf[i] = Vec2{
				X: s.X + dx*t + px*off,
				Y: s.Y + dy*t + py*off,
			}
		}

	case RopeCurveCustom:
		if r.config.PointsFunc != nil {
			r.ptsBuf = r.config.PointsFunc(r.ptsBuf)
		}
	}

	r.SetPoints(r.ptsBuf)
}

// SetPoints updates the rope's path. For N points: 2N vertices, 6(N-1) indices.
func (r *Rope) SetPoints(points []Vec2) {
	if len(points) < 2 {
		r.node.Vertices = r.node.Vertices[:0]
		r.node.Indices = r.node.Indices[:0]
		r.node.InvalidateMeshAABB()
		return
	}

	n := len(points)
	numVerts := n * 2
	numInds := (n - 1) * 6

	// Grow vertex/index slices to high-water mark.
	if cap(r.node.Vertices) < numVerts {
		r.node.Vertices = make([]ebiten.Vertex, numVerts)
	}
	r.node.Vertices = r.node.Vertices[:numVerts]

	if cap(r.node.Indices) < numInds {
		r.node.Indices = make([]uint16, numInds)
	}
	r.node.Indices = r.node.Indices[:numInds]

	imgH := float64(0)
	if r.node.MeshImage != nil {
		imgH = float64(r.node.MeshImage.Bounds().Dy())
	}

	halfW := r.config.Width / 2

	// Compute cumulative path length for UV tiling.
	if cap(r.cumLen) < n {
		r.cumLen = make([]float64, n)
	}
	r.cumLen = r.cumLen[:n]
	r.cumLen[0] = 0
	for i := 1; i < n; i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		r.cumLen[i] = r.cumLen[i-1] + math.Sqrt(dx*dx+dy*dy)
	}

	for i := 0; i < n; i++ {
		// Compute perpendicular normal.
		var nx, ny float64
		if i == 0 {
			nx, ny = perpendicular(points[0], points[1])
		} else if i == n-1 {
			nx, ny = perpendicular(points[n-2], points[n-1])
		} else {
			// Average of adjacent segment normals (miter).
			nx0, ny0 := perpendicular(points[i-1], points[i])
			nx1, ny1 := perpendicular(points[i], points[i+1])
			nx, ny = nx0+nx1, ny0+ny1
			ln := math.Sqrt(nx*nx + ny*ny)
			if ln > 1e-10 {
				nx /= ln
				ny /= ln
			}
			if r.config.JoinMode == RopeJoinMiter {
				// Scale to maintain width at the miter, clamped to avoid
				// exaggerated spikes at sharp corners (max 2x extension).
				dot := nx0*nx + ny0*ny
				if dot > 0.1 {
					scale := 1.0 / dot
					if scale > 2.0 {
						scale = 2.0
					}
					nx *= scale
					ny *= scale
				}
			}
		}

		srcX := float32(r.cumLen[i])
		vi := i * 2
		r.node.Vertices[vi] = ebiten.Vertex{
			DstX:   float32(points[i].X + nx*halfW),
			DstY:   float32(points[i].Y + ny*halfW),
			SrcX:   srcX,
			SrcY:   0,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
		r.node.Vertices[vi+1] = ebiten.Vertex{
			DstX:   float32(points[i].X - nx*halfW),
			DstY:   float32(points[i].Y - ny*halfW),
			SrcX:   srcX,
			SrcY:   float32(imgH),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
	}

	// Build indices: two triangles per segment.
	for i := 0; i < n-1; i++ {
		ii := i * 6
		v := uint16(i * 2)
		r.node.Indices[ii+0] = v
		r.node.Indices[ii+1] = v + 1
		r.node.Indices[ii+2] = v + 2
		r.node.Indices[ii+3] = v + 1
		r.node.Indices[ii+4] = v + 3
		r.node.Indices[ii+5] = v + 2
	}

	r.node.InvalidateMeshAABB()
}

// perpendicular returns the unit left-perpendicular of the segment from a to b.
func perpendicular(a, b Vec2) (float64, float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	ln := math.Sqrt(dx*dx + dy*dy)
	if ln < 1e-10 {
		return 0, -1
	}
	return -dy / ln, dx / ln
}

// --- DistortionGrid ---

// DistortionGrid provides a grid mesh that can be deformed per-vertex.
type DistortionGrid struct {
	node    *Node
	cols    int
	rows    int
	restPos []Vec2 // original vertex positions for Reset
}

// NewDistortionGrid creates a grid mesh over the given image. cols and rows
// define the number of cells (vertices = (cols+1) * (rows+1)).
func NewDistortionGrid(name string, img *ebiten.Image, cols, rows int) (*DistortionGrid, *Node) {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	var imgW, imgH float64
	if img != nil {
		b := img.Bounds()
		imgW = float64(b.Dx())
		imgH = float64(b.Dy())
	}

	vcols := cols + 1
	vrows := rows + 1
	numVerts := vcols * vrows
	numInds := cols * rows * 6

	verts := make([]ebiten.Vertex, numVerts)
	inds := make([]uint16, numInds)
	restPos := make([]Vec2, numVerts)

	cellW := imgW / float64(cols)
	cellH := imgH / float64(rows)

	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			x := float64(c) * cellW
			y := float64(r) * cellH
			u := float32(x)
			v := float32(y)
			verts[idx] = ebiten.Vertex{
				DstX: float32(x), DstY: float32(y),
				SrcX: u, SrcY: v,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			}
			restPos[idx] = Vec2{X: x, Y: y}
		}
	}

	// Build indices.
	ii := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tl := uint16(r*vcols + c)
			tr := tl + 1
			bl := uint16((r+1)*vcols + c)
			br := bl + 1
			inds[ii+0] = tl
			inds[ii+1] = bl
			inds[ii+2] = tr
			inds[ii+3] = tr
			inds[ii+4] = bl
			inds[ii+5] = br
			ii += 6
		}
	}

	n := NewMesh(name, img, verts, inds)
	g := &DistortionGrid{node: n, cols: cols, rows: rows, restPos: restPos}
	return g, n
}

// Node returns the underlying mesh node.
func (g *DistortionGrid) Node() *Node {
	return g.node
}

// Cols returns the number of grid columns.
func (g *DistortionGrid) Cols() int { return g.cols }

// Rows returns the number of grid rows.
func (g *DistortionGrid) Rows() int { return g.rows }

// SetVertex offsets a single grid vertex by (dx, dy) from its rest position.
func (g *DistortionGrid) SetVertex(col, row int, dx, dy float64) {
	vcols := g.cols + 1
	idx := row*vcols + col
	rest := g.restPos[idx]
	g.node.Vertices[idx].DstX = float32(rest.X + dx)
	g.node.Vertices[idx].DstY = float32(rest.Y + dy)
	g.node.InvalidateMeshAABB()
}

// SetAllVertices calls fn for each vertex, passing (col, row, restX, restY).
// fn returns the (dx, dy) displacement from the rest position.
func (g *DistortionGrid) SetAllVertices(fn func(col, row int, restX, restY float64) (dx, dy float64)) {
	vcols := g.cols + 1
	vrows := g.rows + 1
	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			rest := g.restPos[idx]
			dx, dy := fn(c, r, rest.X, rest.Y)
			g.node.Vertices[idx].DstX = float32(rest.X + dx)
			g.node.Vertices[idx].DstY = float32(rest.Y + dy)
		}
	}
	g.node.InvalidateMeshAABB()
}

// Reset returns all vertices to their original positions.
func (g *DistortionGrid) Reset() {
	vcols := g.cols + 1
	vrows := g.rows + 1
	for r := 0; r < vrows; r++ {
		for c := 0; c < vcols; c++ {
			idx := r*vcols + c
			rest := g.restPos[idx]
			g.node.Vertices[idx].DstX = float32(rest.X)
			g.node.Vertices[idx].DstY = float32(rest.Y)
		}
	}
	g.node.InvalidateMeshAABB()
}

// --- Polygon ---

// NewPolygon creates an untextured polygon mesh from the given vertices.
// Uses fan triangulation (convex polygons). The polygon is drawn with a shared
// 1x1 white pixel image; color comes from the node's Color field.
func NewPolygon(name string, points []Vec2) *Node {
	white := ensureWhitePixel()
	verts, inds := buildPolygonFan(points, false, nil)
	return NewMesh(name, white, verts, inds)
}

// NewPolygonTextured creates a textured polygon mesh. UVs are mapped to the
// bounding box of the points, so (0,0)→top-left and (imgW,imgH)→bottom-right.
func NewPolygonTextured(name string, img *ebiten.Image, points []Vec2) *Node {
	verts, inds := buildPolygonFan(points, true, img)
	return NewMesh(name, img, verts, inds)
}

// SetPolygonPoints updates the polygon's vertices. Maintains fan triangulation.
// If textured is true and img is non-nil, UVs are mapped to the bounding box.
func SetPolygonPoints(n *Node, points []Vec2) {
	textured := false
	var img *ebiten.Image
	if n.MeshImage != nil && n.MeshImage != ensureWhitePixel() {
		textured = true
		img = n.MeshImage
	}
	verts, inds := buildPolygonFan(points, textured, img)

	// Reuse backing arrays when possible.
	if cap(n.Vertices) >= len(verts) {
		n.Vertices = n.Vertices[:len(verts)]
		copy(n.Vertices, verts)
	} else {
		n.Vertices = verts
	}
	if cap(n.Indices) >= len(inds) {
		n.Indices = n.Indices[:len(inds)]
		copy(n.Indices, inds)
	} else {
		n.Indices = inds
	}

	n.InvalidateMeshAABB()
}

// buildPolygonFan generates vertices and indices for an arbitrary simple polygon.
// Uses ear-clipping triangulation, which handles both convex and concave shapes.
func buildPolygonFan(points []Vec2, textured bool, img *ebiten.Image) ([]ebiten.Vertex, []uint16) {
	n := len(points)
	if n < 3 {
		return nil, nil
	}

	verts := make([]ebiten.Vertex, n)
	inds := earClipIndices(points)

	// Compute bounding box for UV mapping (textured mode).
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
			// Untextured: map to center of white pixel (0.5, 0.5)
			v.SrcX = 0.5
			v.SrcY = 0.5
		}
	}

	return verts, inds
}

// earClipIndices triangulates a simple polygon using ear-clipping.
// Handles both convex and concave polygons. O(n²)  -  fine for small polygons.
func earClipIndices(pts []Vec2) []uint16 {
	n := len(pts)
	if n < 3 {
		return nil
	}
	if n == 3 {
		return []uint16{0, 1, 2}
	}

	// Signed area determines winding. Positive → convex ears have positive cross.
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

			cr := poly2DCross(pts[a], pts[b], pts[c])
			if !((ccw && cr > 0) || (!ccw && cr < 0)) {
				continue // not a convex vertex
			}

			// No other vertex may lie inside triangle (a, b, c).
			inside := false
			for j := 0; j < m; j++ {
				if j == (i+m-1)%m || j == i || j == (i+1)%m {
					continue
				}
				if ptInTriangle(pts[idx[j]], pts[a], pts[b], pts[c]) {
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
			// Degenerate: fan-triangulate the remainder.
			for j := 1; j < len(idx)-1; j++ {
				result = append(result, uint16(idx[0]), uint16(idx[j]), uint16(idx[j+1]))
			}
			return result
		}
	}
	return append(result, uint16(idx[0]), uint16(idx[1]), uint16(idx[2]))
}

// poly2DCross returns (a-o) × (b-o). Positive = CCW turn, negative = CW turn.
func poly2DCross(o, a, b Vec2) float64 {
	return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X)
}

// ptInTriangle reports whether p lies inside or on the boundary of triangle (a, b, c).
func ptInTriangle(p, a, b, c Vec2) bool {
	d1 := poly2DCross(a, b, p)
	d2 := poly2DCross(b, c, p)
	d3 := poly2DCross(c, a, p)
	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(hasNeg && hasPos)
}
