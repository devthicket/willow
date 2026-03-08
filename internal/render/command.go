package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/particle"
	"github.com/phanxgames/willow/internal/types"
)

// CommandType identifies the kind of render command.
type CommandType uint8

const (
	CommandSprite     CommandType = iota // DrawImage
	CommandMesh                          // DrawTriangles
	CommandParticle                      // particle quads (batches as sprites)
	CommandTilemap                       // DrawTriangles for tilemap layers
	CommandSDF                           // DrawTrianglesShader for SDF text glyphs
	CommandBitmapText                    // DrawTriangles for pixel-perfect bitmap font glyphs
)

// Color32 is a compact RGBA color using float32, for render commands only.
type Color32 struct {
	R, G, B, A float32
}

// BatchMode controls how the render pipeline submits draw calls.
type BatchMode uint8

const (
	// BatchModeCoalesced accumulates vertices and submits one DrawTriangles32 per batch key run.
	BatchModeCoalesced BatchMode = iota
	// BatchModeImmediate submits one DrawImage per sprite (legacy).
	BatchModeImmediate
)

// RenderCommand is a single draw instruction emitted during scene traversal.
type RenderCommand struct {
	Type          CommandType
	Transform     [6]float32
	TextureRegion types.TextureRegion
	Color         Color32
	BlendMode     types.BlendMode
	ShaderID      uint16
	TargetID      uint16
	RenderLayer   uint8
	GlobalOrder   int
	TreeOrder     int // assigned during traversal for stable sort

	// Mesh-only fields (slice headers, not copies of vertex data).
	MeshVerts []ebiten.Vertex
	MeshInds  []uint16
	MeshImage *ebiten.Image

	// DirectImage, when non-nil, is drawn directly instead of looking up an
	// atlas page. Used for cached/filtered/masked node output.
	DirectImage *ebiten.Image

	// Emitter references the particle emitter for CommandParticle commands.
	Emitter            *particle.Emitter
	WorldSpaceParticle bool // particles store world positions; Transform is view-only

	// TransientDirectImage is true when DirectImage points to a pooled RT
	// that is released after the frame.
	TransientDirectImage bool

	// EmittingNodeID is set during CacheAsTree builds to track which Node
	// emitted this command.
	EmittingNodeID uint32

	// Tilemap fields (CommandTilemap only — slice headers, not copies).
	TilemapVerts []ebiten.Vertex
	TilemapInds  []uint16
	TilemapImage *ebiten.Image

	// SDF text fields (CommandSDF only — slice headers, not copies).
	SdfVerts     []ebiten.Vertex // local-space glyph quads (from TextBlock cache)
	SdfInds      []uint16        // glyph indices (from TextBlock cache)
	SdfVertCount int             // active vertex count
	SdfIndCount  int             // active index count
	SdfShader    *ebiten.Shader  // SDF or MSDF shader
	SdfAtlasImg  *ebiten.Image   // font's SDF atlas page
	SdfUniforms  map[string]any  // shader uniforms (threshold, smoothing, effects)

	// Bitmap text fields (CommandBitmapText only — slice headers, not copies).
	BmpVerts     []ebiten.Vertex
	BmpInds      []uint16
	BmpVertCount int
	BmpIndCount  int
	BmpImage     *ebiten.Image
}

// BatchKey groups render commands that can be submitted in a single draw call.
type BatchKey struct {
	TargetID uint16
	ShaderID uint16
	Blend    types.BlendMode
	Page     uint16
}

// CommandBatchKey extracts the batch grouping key from a render command.
func CommandBatchKey(cmd *RenderCommand) BatchKey {
	return BatchKey{
		TargetID: cmd.TargetID,
		ShaderID: cmd.ShaderID,
		Blend:    cmd.BlendMode,
		Page:     cmd.TextureRegion.Page,
	}
}

// IdentityTransform32 is the identity affine matrix as float32.
var IdentityTransform32 = [6]float32{1, 0, 0, 1, 0, 0}

// Affine32 converts a [6]float64 affine matrix to [6]float32.
func Affine32(m [6]float64) [6]float32 {
	return [6]float32{float32(m[0]), float32(m[1]), float32(m[2]), float32(m[3]), float32(m[4]), float32(m[5])}
}

// MultiplyAffine32 multiplies two 2D affine matrices stored as [6]float32.
func MultiplyAffine32(p, c [6]float32) [6]float32 {
	return [6]float32{
		p[0]*c[0] + p[2]*c[1],
		p[1]*c[0] + p[3]*c[1],
		p[0]*c[2] + p[2]*c[3],
		p[1]*c[2] + p[3]*c[3],
		p[0]*c[4] + p[2]*c[5] + p[4],
		p[1]*c[4] + p[3]*c[5] + p[5],
	}
}

// InvertAffine32 computes the inverse of a 2D affine matrix in float32.
func InvertAffine32(m [6]float32) [6]float32 {
	det := m[0]*m[3] - m[2]*m[1]
	if det > -1e-6 && det < 1e-6 {
		return IdentityTransform32
	}
	invDet := 1.0 / det
	a := m[3] * invDet
	b := -m[1] * invDet
	c := -m[2] * invDet
	d := m[0] * invDet
	return [6]float32{
		a, b, c, d,
		-(a*m[4] + c*m[5]),
		-(b*m[4] + d*m[5]),
	}
}

// CommandGeoM converts a command's [6]float32 transform into an ebiten.GeoM.
func CommandGeoM(cmd *RenderCommand) ebiten.GeoM {
	var m ebiten.GeoM
	m.SetElement(0, 0, float64(cmd.Transform[0]))
	m.SetElement(1, 0, float64(cmd.Transform[1]))
	m.SetElement(0, 1, float64(cmd.Transform[2]))
	m.SetElement(1, 1, float64(cmd.Transform[3]))
	m.SetElement(0, 2, float64(cmd.Transform[4]))
	m.SetElement(1, 2, float64(cmd.Transform[5]))
	return m
}
