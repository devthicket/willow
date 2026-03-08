package willow

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- Debug stats and helpers -----------------------------------------------

// debugStats holds per-frame timing and draw-call metrics.
// Only populated when Scene.debug is true.
type debugStats struct {
	traverseTime  time.Duration
	sortTime      time.Duration
	batchTime     time.Duration
	submitTime    time.Duration
	commandCount  int
	batchCount    int
	drawCallCount int
}

// debugLog prints timing and draw-call stats to stderr.
func (s *Scene) debugLog(stats debugStats) {
	if !s.debug {
		return
	}
	total := stats.traverseTime + stats.sortTime + stats.batchTime + stats.submitTime
	_, _ = fmt.Fprintf(os.Stderr,
		"[willow] traverse: %v | sort: %v | batch: %v | submit: %v | total: %v\n",
		stats.traverseTime, stats.sortTime, stats.batchTime, stats.submitTime, total)
	_, _ = fmt.Fprintf(os.Stderr,
		"[willow] commands: %d | batches: %d | draw calls: %d\n",
		stats.commandCount, stats.batchCount, stats.drawCallCount)
}

// debugCheckDisposed panics with a descriptive message when a disposed node is
// used in a tree operation. Only called when Scene.debug or the node's scene is
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

// countBatches counts contiguous groups of commands sharing the same batchKey.
// This reports how many draw calls a true batching implementation would produce.
func countBatches(commands []RenderCommand) int {
	if len(commands) == 0 {
		return 0
	}
	count := 1
	prev := commandBatchKey(&commands[0])
	for i := 1; i < len(commands); i++ {
		cur := commandBatchKey(&commands[i])
		if cur != prev {
			count++
			prev = cur
		}
	}
	return count
}

// countDrawCalls counts individual draw calls from the command list.
// Meshes and direct-image sprites each count as 1. Particle commands count
// as the number of alive particles.
func countDrawCalls(commands []RenderCommand) int {
	count := 0
	for i := range commands {
		cmd := &commands[i]
		switch cmd.Type {
		case CommandParticle:
			if cmd.emitter != nil {
				count += cmd.emitter.alive
			}
		case CommandSDF:
			count++ // each text node = 1 DrawTrianglesShader call
		default:
			count++
		}
	}
	return count
}

// countDrawCallsCoalesced estimates actual draw calls in coalesced mode.
// Each batch-key run of atlas sprites is 1 DrawTriangles32 call. Each particle
// emitter is 1 call. Direct-image sprites and meshes are 1 call each.
func countDrawCallsCoalesced(commands []RenderCommand) int {
	if len(commands) == 0 {
		return 0
	}
	count := 0
	inSpriteRun := false
	var prevKey batchKey
	for i := range commands {
		cmd := &commands[i]
		switch cmd.Type {
		case CommandSprite:
			if cmd.directImage != nil {
				if inSpriteRun {
					count++ // flush previous run
					inSpriteRun = false
				}
				count++ // this direct-image sprite
			} else {
				key := commandBatchKey(cmd)
				if !inSpriteRun || key != prevKey {
					if inSpriteRun {
						count++ // flush previous run
					}
					inSpriteRun = true
					prevKey = key
				}
			}
		case CommandParticle:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			if cmd.emitter != nil && cmd.emitter.alive > 0 {
				count++ // 1 DrawTriangles32 per emitter
			}
		case CommandMesh:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			count++
		case CommandSDF:
			if inSpriteRun {
				count++
				inSpriteRun = false
			}
			count++ // each text node = 1 DrawTrianglesShader call
		}
	}
	if inSpriteRun {
		count++
	}
	return count
}

// ---- Screenshot ------------------------------------------------------------

// Screenshot queues a labeled screenshot to be captured at the end of the
// current frame's Draw call. The resulting PNG is written to ScreenshotDir
// with a timestamped filename. Safe to call from Update or Draw.
func (s *Scene) Screenshot(label string) {
	s.screenshotQueue = append(s.screenshotQueue, label)
}

// flushScreenshots captures the rendered frame for every queued label and
// writes each as a PNG file. Called at the end of Scene.Draw.
func (s *Scene) flushScreenshots(screen *ebiten.Image) {
	if len(s.screenshotQueue) == 0 {
		return
	}

	if err := os.MkdirAll(s.ScreenshotDir, 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] screenshot: mkdir %s: %v\n", s.ScreenshotDir, err)
		s.screenshotQueue = s.screenshotQueue[:0]
		return
	}

	bounds := screen.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, 4*w*h)
	screen.ReadPixels(pixels)

	// Convert premultiplied RGBA to straight-alpha NRGBA.
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pixels); i += 4 {
		r, g, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
		if a > 0 && a < 255 {
			r = uint8(min(int(r)*255/int(a), 255))
			g = uint8(min(int(g)*255/int(a), 255))
			b = uint8(min(int(b)*255/int(a), 255))
		}
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}

	stamp := time.Now().Format("20060102_150405")

	for _, label := range s.screenshotQueue {
		safe := sanitizeLabel(label)
		path := fmt.Sprintf("%s/%s_%s.png", s.ScreenshotDir, stamp, safe)
		if err := writePNG(path, img); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[willow] screenshot: %v\n", err)
		}
	}

	s.screenshotQueue = s.screenshotQueue[:0]
}

// writePNG encodes an image to a PNG file at the given path.
func writePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return f.Close()
}

// sanitizeLabel replaces characters that are unsafe in file names with
// underscores and falls back to "unlabeled" for empty strings.
func sanitizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "unlabeled"
	}
	var b strings.Builder
	b.Grow(len(label))
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ---- Input injection -------------------------------------------------------

// syntheticPointerEvent represents a single injected pointer event.
// Screen coordinates are used (matching what an AI sees in screenshots)
// and converted to world coordinates via the primary camera, identical
// to real mouse input.
type syntheticPointerEvent struct {
	screenX, screenY float64
	pressed          bool
	button           MouseButton
}

// InjectPress queues a pointer press event at the given screen coordinates
// (left button). The event is consumed on the next frame's processInput call.
func (s *Scene) InjectPress(x, y float64) {
	s.injectQueue = append(s.injectQueue, syntheticPointerEvent{
		screenX: x, screenY: y,
		pressed: true,
		button:  MouseButtonLeft,
	})
}

// InjectMove queues a pointer move event at the given screen coordinates
// with the button held down. Use this between InjectPress and InjectRelease
// to simulate a drag.
func (s *Scene) InjectMove(x, y float64) {
	s.injectQueue = append(s.injectQueue, syntheticPointerEvent{
		screenX: x, screenY: y,
		pressed: true,
		button:  MouseButtonLeft,
	})
}

// InjectRelease queues a pointer release event at the given screen coordinates.
func (s *Scene) InjectRelease(x, y float64) {
	s.injectQueue = append(s.injectQueue, syntheticPointerEvent{
		screenX: x, screenY: y,
		pressed: false,
		button:  MouseButtonLeft,
	})
}

// InjectClick is a convenience that queues a press followed by a release
// at the same screen coordinates. Consumes two frames.
func (s *Scene) InjectClick(x, y float64) {
	s.InjectPress(x, y)
	s.InjectRelease(x, y)
}

// InjectDrag queues a full drag sequence: press at (fromX, fromY),
// linearly interpolated moves over frames-2 intermediate frames, and
// release at (toX, toY). The total sequence consumes `frames` frames.
// Minimum frames is 2 (press + release).
func (s *Scene) InjectDrag(fromX, fromY, toX, toY float64, frames int) {
	if frames < 2 {
		frames = 2
	}
	s.InjectPress(fromX, fromY)
	steps := frames - 2
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps+1)
		x := fromX + (toX-fromX)*t
		y := fromY + (toY-fromY)*t
		s.InjectMove(x, y)
	}
	s.InjectRelease(toX, toY)
}

// processInjectedInput pops one event from the inject queue, converts
// screen→world via the primary camera, and feeds it through processPointer.
// Returns true if an event was consumed (real mouse input should be skipped).
func (s *Scene) processInjectedInput(cam *Camera, mods KeyModifiers) bool {
	if len(s.injectQueue) == 0 {
		return false
	}
	evt := s.injectQueue[0]
	copy(s.injectQueue, s.injectQueue[1:])
	s.injectQueue = s.injectQueue[:len(s.injectQueue)-1]

	wx, wy := screenToWorld(cam, evt.screenX, evt.screenY)
	s.processPointer(0, wx, wy, evt.screenX, evt.screenY, evt.pressed, evt.button, mods)
	return true
}

// ---- Test runner -----------------------------------------------------------

// testStep represents a single action in a test script.
type testStep struct {
	Action string  `json:"action"`
	Label  string  `json:"label,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	FromX  float64 `json:"fromX,omitempty"`
	FromY  float64 `json:"fromY,omitempty"`
	ToX    float64 `json:"toX,omitempty"`
	ToY    float64 `json:"toY,omitempty"`
	Frames int     `json:"frames,omitempty"`
}

// testScript is the top-level JSON structure for a test script.
type testScript struct {
	Steps []testStep `json:"steps"`
}

// TestRunner sequences injected input events and screenshots across frames
// for automated visual testing. Attach to a Scene via SetTestRunner.
type TestRunner struct {
	steps     []testStep
	cursor    int
	waitCount int
	done      bool
}

// LoadTestScript parses a JSON test script and returns a TestRunner ready
// to be attached to a Scene via SetTestRunner.
func LoadTestScript(jsonData []byte) (*TestRunner, error) {
	var script testScript
	if err := json.Unmarshal(jsonData, &script); err != nil {
		return nil, fmt.Errorf("parse test script: %w", err)
	}
	if len(script.Steps) == 0 {
		return nil, fmt.Errorf("parse test script: no steps")
	}
	return &TestRunner{steps: script.Steps}, nil
}

// SetTestRunner attaches a TestRunner to the scene. The runner's step method
// is called from Scene.Update before processInput each frame.
func (s *Scene) SetTestRunner(runner *TestRunner) {
	s.testRunner = runner
}

// Done reports whether all steps in the test script have been executed.
func (r *TestRunner) Done() bool {
	return r.done
}

// step advances the test runner by one frame. Called from Scene.Update.
func (r *TestRunner) step(s *Scene) {
	if r.done {
		return
	}
	// Wait for pending injections to drain before advancing.
	if len(s.injectQueue) > 0 {
		return
	}
	// Count down wait frames.
	if r.waitCount > 0 {
		r.waitCount--
		return
	}
	if r.cursor >= len(r.steps) {
		r.done = true
		return
	}

	st := r.steps[r.cursor]
	r.cursor++

	switch st.Action {
	case "screenshot":
		s.Screenshot(st.Label)
	case "click":
		s.InjectClick(st.X, st.Y)
	case "drag":
		frames := st.Frames
		if frames < 2 {
			frames = 2
		}
		s.InjectDrag(st.FromX, st.FromY, st.ToX, st.ToY, frames)
	case "wait":
		if st.Frames > 0 {
			r.waitCount = st.Frames - 1 // this frame counts as one
		}
	}

	// Check if we've reached the end after executing.
	if r.cursor >= len(r.steps) && r.waitCount == 0 && len(s.injectQueue) == 0 {
		r.done = true
	}
}
