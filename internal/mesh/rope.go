package mesh

import (
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// RopeJoinMode controls how segments join in a Rope mesh.
type RopeJoinMode uint8

const (
	RopeJoinMiter RopeJoinMode = iota
	RopeJoinBevel
)

// RopeCurveMode selects the curve algorithm used by Rope.Update().
type RopeCurveMode uint8

const (
	RopeCurveLine RopeCurveMode = iota
	RopeCurveCatenary
	RopeCurveQuadBezier
	RopeCurveCubicBezier
	RopeCurveWave
	RopeCurveCustom
)

// RopeConfig configures a Rope mesh.
type RopeConfig struct {
	Width     float64
	JoinMode  RopeJoinMode
	CurveMode RopeCurveMode
	Segments  int // number of subdivisions (default 20)
	Start     *types.Vec2
	End       *types.Vec2
	Sag       float64
	Controls  [2]*types.Vec2
	Amplitude float64
	Frequency float64
	Phase     float64
	// Custom callback. Receives a preallocated buffer; must return the slice to use.
	PointsFunc func(buf []types.Vec2) []types.Vec2
}

// Rope generates a ribbon/rope mesh that follows a polyline path.
type Rope struct {
	node_  *node.Node
	config RopeConfig
	cumLen []float64
	ptsBuf []types.Vec2
}

// NewRope creates a rope mesh node that renders a textured ribbon along the given points.
func NewRope(name string, img *ebiten.Image, points []types.Vec2, cfg RopeConfig) *Rope {
	n := NewMeshFn(name, img, nil, nil)
	r := &Rope{node_: n, config: cfg}
	r.SetPoints(points)
	return r
}

// Node returns the underlying mesh node.
func (r *Rope) Node() *node.Node {
	return r.node_
}

// Config returns a pointer to the rope's configuration.
func (r *Rope) Config() *RopeConfig {
	return &r.config
}

// Update recomputes the rope's point path from the current RopeConfig and
// rebuilds the mesh.
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

	if cap(r.ptsBuf) < n {
		r.ptsBuf = make([]types.Vec2, n)
	}
	r.ptsBuf = r.ptsBuf[:n]

	switch r.config.CurveMode {
	case RopeCurveLine:
		s, e := *r.config.Start, *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			r.ptsBuf[i] = types.Vec2{
				X: s.X + (e.X-s.X)*t,
				Y: s.Y + (e.Y-s.Y)*t,
			}
		}

	case RopeCurveCatenary:
		s, e := *r.config.Start, *r.config.End
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			r.ptsBuf[i] = types.Vec2{
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
			r.ptsBuf[i] = types.Vec2{
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
			r.ptsBuf[i] = types.Vec2{
				X: u2*u*a.X + 3*u2*t*c1.X + 3*u*t2*c2.X + t2*t*b.X,
				Y: u2*u*a.Y + 3*u2*t*c1.Y + 3*u*t2*c2.Y + t2*t*b.Y,
			}
		}

	case RopeCurveWave:
		s, e := *r.config.Start, *r.config.End
		dx := e.X - s.X
		dy := e.Y - s.Y
		ln := math.Sqrt(dx*dx + dy*dy)
		var px, py float64
		if ln > 1e-10 {
			px = -dy / ln
			py = dx / ln
		}
		for i := 0; i < n; i++ {
			t := float64(i) / float64(segs)
			off := r.config.Amplitude * math.Sin(r.config.Frequency*2*math.Pi*t+r.config.Phase)
			r.ptsBuf[i] = types.Vec2{
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
func (r *Rope) SetPoints(points []types.Vec2) {
	if len(points) < 2 {
		r.node_.Mesh.Vertices = r.node_.Mesh.Vertices[:0]
		r.node_.Mesh.Indices = r.node_.Mesh.Indices[:0]
		r.node_.Mesh.AabbDirty = true
		return
	}

	n := len(points)
	numVerts := n * 2
	numInds := (n - 1) * 6

	if cap(r.node_.Mesh.Vertices) < numVerts {
		r.node_.Mesh.Vertices = make([]ebiten.Vertex, numVerts)
	}
	r.node_.Mesh.Vertices = r.node_.Mesh.Vertices[:numVerts]

	if cap(r.node_.Mesh.Indices) < numInds {
		r.node_.Mesh.Indices = make([]uint16, numInds)
	}
	r.node_.Mesh.Indices = r.node_.Mesh.Indices[:numInds]

	imgH := float64(0)
	if r.node_.Mesh.Image != nil {
		imgH = float64(r.node_.Mesh.Image.Bounds().Dy())
	}

	halfW := r.config.Width / 2

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
		var nx, ny float64
		if i == 0 {
			nx, ny = Perpendicular(points[0], points[1])
		} else if i == n-1 {
			nx, ny = Perpendicular(points[n-2], points[n-1])
		} else {
			nx0, ny0 := Perpendicular(points[i-1], points[i])
			nx1, ny1 := Perpendicular(points[i], points[i+1])
			nx, ny = nx0+nx1, ny0+ny1
			ln := math.Sqrt(nx*nx + ny*ny)
			if ln > 1e-10 {
				nx /= ln
				ny /= ln
			}
			if r.config.JoinMode == RopeJoinMiter {
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
		r.node_.Mesh.Vertices[vi] = ebiten.Vertex{
			DstX:   float32(points[i].X + nx*halfW),
			DstY:   float32(points[i].Y + ny*halfW),
			SrcX:   srcX,
			SrcY:   0,
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
		r.node_.Mesh.Vertices[vi+1] = ebiten.Vertex{
			DstX:   float32(points[i].X - nx*halfW),
			DstY:   float32(points[i].Y - ny*halfW),
			SrcX:   srcX,
			SrcY:   float32(imgH),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		}
	}

	for i := 0; i < n-1; i++ {
		ii := i * 6
		v := uint16(i * 2)
		r.node_.Mesh.Indices[ii+0] = v
		r.node_.Mesh.Indices[ii+1] = v + 1
		r.node_.Mesh.Indices[ii+2] = v + 2
		r.node_.Mesh.Indices[ii+3] = v + 1
		r.node_.Mesh.Indices[ii+4] = v + 3
		r.node_.Mesh.Indices[ii+5] = v + 2
	}

	r.node_.Mesh.AabbDirty = true
}

// Perpendicular returns the unit left-perpendicular of the segment from a to b.
func Perpendicular(a, b types.Vec2) (float64, float64) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	ln := math.Sqrt(dx*dx + dy*dy)
	if ln < 1e-10 {
		return 0, -1
	}
	return -dy / ln, dx / ln
}
