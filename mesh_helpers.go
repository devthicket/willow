package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/mesh"
	"github.com/phanxgames/willow/internal/node"
)

func init() {
	// Wire NewMeshFn so internal/mesh can create mesh nodes.
	mesh.NewMeshFn = func(name string, img *ebiten.Image, verts []ebiten.Vertex, inds []uint16) *node.Node {
		return NewMesh(name, img, verts, inds)
	}
}

// RopeJoinMode controls how segments join in a Rope mesh.
// Aliased from internal/mesh.RopeJoinMode.
type RopeJoinMode = mesh.RopeJoinMode

// RopeJoin mode constants.
const (
	RopeJoinMiter = mesh.RopeJoinMiter
	RopeJoinBevel = mesh.RopeJoinBevel
)

// RopeCurveMode selects the curve algorithm used by Rope.Update().
// Aliased from internal/mesh.RopeCurveMode.
type RopeCurveMode = mesh.RopeCurveMode

// RopeCurve mode constants.
const (
	RopeCurveLine        = mesh.RopeCurveLine
	RopeCurveCatenary    = mesh.RopeCurveCatenary
	RopeCurveQuadBezier  = mesh.RopeCurveQuadBezier
	RopeCurveCubicBezier = mesh.RopeCurveCubicBezier
	RopeCurveWave        = mesh.RopeCurveWave
	RopeCurveCustom      = mesh.RopeCurveCustom
)

// RopeConfig configures a Rope mesh.
// Aliased from internal/mesh.RopeConfig.
type RopeConfig = mesh.RopeConfig

// Rope generates a ribbon/rope mesh that follows a polyline path.
// Aliased from internal/mesh.Rope.
type Rope = mesh.Rope

// NewRope creates a rope mesh node.
var NewRope = mesh.NewRope

// DistortionGrid provides a grid mesh that can be deformed per-vertex.
// Aliased from internal/mesh.DistortionGrid.
type DistortionGrid = mesh.DistortionGrid

// NewDistortionGrid creates a grid mesh over the given image.
var NewDistortionGrid = mesh.NewDistortionGrid

// NewPolygon creates an untextured polygon mesh from the given vertices.
var NewPolygon = mesh.NewPolygon

// NewRegularPolygon creates an untextured regular polygon.
var NewRegularPolygon = mesh.NewRegularPolygon

// NewStar creates an untextured star polygon.
var NewStar = mesh.NewStar

// NewPolygonTextured creates a textured polygon mesh.
var NewPolygonTextured = mesh.NewPolygonTextured

// SetPolygonPoints updates the polygon's vertices.
var SetPolygonPoints = mesh.SetPolygonPoints
