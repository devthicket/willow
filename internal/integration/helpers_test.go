package integration

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/atlas"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/core"
	"github.com/phanxgames/willow/internal/mesh"
	"github.com/phanxgames/willow/internal/particle"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/types"

	. "github.com/phanxgames/willow"
)

// ---------------------------------------------------------------------------
// Atlas helpers
// ---------------------------------------------------------------------------

const magentaPlaceholderPage = atlas.MagentaPlaceholderPage

func ensureMagentaImage() *ebiten.Image {
	return atlas.EnsureMagentaImage()
}

func atlasManager() *atlas.Manager {
	return atlas.GlobalManager()
}

func resetAtlasManager() {
	atlas.ResetGlobalManager()
}

// ---------------------------------------------------------------------------
// Batch helpers
// ---------------------------------------------------------------------------

type batchKey = render.BatchKey

func commandBatchKey(cmd *RenderCommand) batchKey {
	return render.CommandBatchKey(cmd)
}

var identityTransform32 = [6]float32{1, 0, 0, 1, 0, 0}

type color32 = render.Color32

// ---------------------------------------------------------------------------
// Debug / stats helpers
// ---------------------------------------------------------------------------

type debugStats struct {
	traverseTime  int
	sortTime      int
	batchTime     int
	submitTime    int
	commandCount  int
	batchCount    int
	drawCallCount int
}

func countBatches(commands []RenderCommand) int {
	return core.CountBatches(commands)
}

func countDrawCalls(commands []RenderCommand) int {
	return core.CountDrawCalls(commands)
}

func countDrawCallsCoalesced(commands []RenderCommand) int {
	return core.CountDrawCallsCoalesced(commands)
}

func sanitizeLabel(label string) string {
	return core.SanitizeLabel(label)
}

func stepActionFor(s *Scene) core.StepAction {
	return core.StepAction{
		Screenshot:  s.Screenshot,
		InjectClick: s.Input.InjectClick,
		InjectDrag:  s.Input.InjectDrag,
		QueueLen:    func() int { return len(s.Input.InjectQueue) },
	}
}

const debugMaxTreeDepth = 32
const debugMaxChildCount = 1000

func debugCheckDisposed(n *Node, op string) {
	if n.Disposed {
		panic(fmt.Sprintf("willow debug: %s on disposed node %q (ID was %d)", op, n.Name, n.ID))
	}
}

func debugCheckTreeDepth(n *Node) {
	depth := 0
	for p := n; p != nil; p = p.Parent {
		depth++
	}
	if depth > debugMaxTreeDepth {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] warning: tree depth %d exceeds %d (node %q)\n",
			depth, debugMaxTreeDepth, n.Name)
	}
}

func debugCheckChildCount(n *Node) {
	if len(n.Children_) > debugMaxChildCount {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] warning: node %q has %d children (threshold %d)\n",
			n.Name, len(n.Children_), debugMaxChildCount)
	}
}

// ---------------------------------------------------------------------------
// Particle helpers
// ---------------------------------------------------------------------------

func newParticleEmitter(cfg EmitterConfig) *ParticleEmitter {
	return particle.NewEmitter(cfg)
}

func updateNodesAndParticles(n *Node, dt float64) {
	core.UpdateNodesAndParticles(n, dt)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// ---------------------------------------------------------------------------
// Mesh helpers
// ---------------------------------------------------------------------------

func computeMeshAABB(verts []ebiten.Vertex) Rect {
	return mesh.ComputeMeshAABB(verts)
}

func ensureTransformedVerts(n *Node) []ebiten.Vertex {
	return mesh.EnsureTransformedVerts(n)
}

func recomputeMeshAABB(n *Node) {
	mesh.RecomputeMeshAABB(n)
}

func ensureWhitePixel() *ebiten.Image {
	return mesh.EnsureWhitePixel()
}

func meshWorldAABBOffset(n *Node, transform [6]float64) types.Rect {
	return mesh.MeshWorldAABBOffset(n, transform)
}

// ---------------------------------------------------------------------------
// Camera / culling helpers
// ---------------------------------------------------------------------------

func nodeDimensions(n *Node) (float64, float64) {
	return camera.NodeDimensions(n)
}

func shouldCull(n *Node, viewWorld [6]float64, cullBounds Rect) bool {
	return camera.ShouldCull(n, viewWorld, cullBounds)
}

// ---------------------------------------------------------------------------
// Text helpers
// ---------------------------------------------------------------------------

func textBlockFontScale(tb *TextBlock) float64 {
	return tb.FontScale()
}

func composeGlyphTransform(world [6]float64, localX, localY float64) [6]float64 {
	return [6]float64{
		world[0], world[1], world[2], world[3],
		world[0]*localX + world[2]*localY + world[4],
		world[1]*localX + world[3]*localY + world[5],
	}
}

// ---------------------------------------------------------------------------
// RenderTexture helpers
// ---------------------------------------------------------------------------

func resolvePageImage(region TextureRegion, pages []*ebiten.Image) *ebiten.Image {
	if region.Page == magentaPlaceholderPage {
		return ensureMagentaImage()
	}
	idx := int(region.Page)
	if idx < len(pages) {
		return pages[idx]
	}
	return nil
}

// ---------------------------------------------------------------------------
// Node helpers
// ---------------------------------------------------------------------------

func getCacheTree(n *Node) *render.CacheTreeData {
	return render.GetCacheTreeData(n)
}

func markSubtreeDirty(n *Node) {
	render.InvalidateAncestorCache(n)
	n.TransformDirty = true
	n.AlphaDirty = true
}
