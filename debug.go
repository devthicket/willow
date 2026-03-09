package willow

import (
	"fmt"
	"os"
	"time"

	"github.com/phanxgames/willow/internal/core"
)

// ---- Debug stats and helpers -----------------------------------------------

// debugStats holds per-frame timing and draw-call metrics.
// Only populated when Scene.Debug is true.
type debugStats struct {
	traverseTime  time.Duration
	sortTime      time.Duration
	batchTime     time.Duration
	submitTime    time.Duration
	commandCount  int
	batchCount    int
	drawCallCount int
}

// debugCheckDisposed panics with a descriptive message when a disposed node is
// used in a tree operation. Only called when Scene.Debug or the node's scene is
// in debug mode. In release mode callers skip this entirely.
func debugCheckDisposed(n *Node, op string) {
	if n.Disposed {
		panic(fmt.Sprintf("willow debug: %s on disposed node %q (ID was %d)", op, n.Name, n.ID))
	}
}

// debugCheckTreeDepth warns on stderr if tree depth exceeds the threshold.
const debugMaxTreeDepth = 32

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

// debugCheckChildCount warns on stderr if a node has more than 1000 children.
const debugMaxChildCount = 1000

func debugCheckChildCount(n *Node) {
	if len(n.Children_) > debugMaxChildCount {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] warning: node %q has %d children (threshold %d)\n",
			n.Name, len(n.Children_), debugMaxChildCount)
	}
}

// countBatches delegates to core.CountBatches.
func countBatches(commands []RenderCommand) int {
	return core.CountBatches(commands)
}

// countDrawCalls delegates to core.CountDrawCalls.
func countDrawCalls(commands []RenderCommand) int {
	return core.CountDrawCalls(commands)
}

// countDrawCallsCoalesced delegates to core.CountDrawCallsCoalesced.
func countDrawCallsCoalesced(commands []RenderCommand) int {
	return core.CountDrawCallsCoalesced(commands)
}

// sanitizeLabel delegates to core.SanitizeLabel.
func sanitizeLabel(label string) string {
	return core.SanitizeLabel(label)
}

// ---- Test runner -----------------------------------------------------------

// TestRunner sequences injected input events and screenshots across frames
// for automated visual testing. Attach to a Scene via SetTestRunner.
// Aliased from internal/core.TestRunner.
type TestRunner = core.TestRunner

// LoadTestScript parses a JSON test script and returns a TestRunner ready
// to be attached to a Scene via SetTestRunner.
func LoadTestScript(jsonData []byte) (*TestRunner, error) {
	return core.LoadTestScript(jsonData)
}

// stepActionFor builds a core.StepAction bound to the given scene.
// Used by tests that drive the TestRunner frame-by-frame.
func stepActionFor(s *Scene) core.StepAction {
	return core.StepAction{
		Screenshot:  s.Screenshot,
		InjectClick: s.Input.InjectClick,
		InjectDrag:  s.Input.InjectDrag,
		QueueLen:    func() int { return len(s.Input.InjectQueue) },
	}
}
