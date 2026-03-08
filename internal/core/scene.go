package core

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/input"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/types"
)

const DefaultCommandCap = 4096

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers. It composes render.Pipeline, input.Manager, and cameras.
type Scene struct {
	Root  *node.Node
	Debug bool

	TransformsReady bool

	ClearColor types.Color

	// Cameras
	Cameras []*camera.Camera

	// Render
	Pipeline render.Pipeline

	// Input
	Input *input.Manager

	// Managed tweens
	Tweens []*TweenGroup

	// Screenshot
	ScreenshotQueue []string
	ScreenshotDir   string

	// Test runner
	TestRunnerRef *TestRunner

	// AntiAlias enables anti-aliased edges on DrawTriangles calls.
	AntiAlias bool
}

// NewScene creates a new Scene with default settings.
// The root node must be provided by the caller (root package creates it).
func NewScene(root *node.Node) *Scene {
	return &Scene{
		Root:          root,
		Input:         input.NewManager(),
		ScreenshotDir: "screenshots",
		Pipeline: render.Pipeline{
			Commands: make([]render.RenderCommand, 0, DefaultCommandCap),
			SortBuf:  make([]render.RenderCommand, 0, DefaultCommandCap),
		},
	}
}

// --- Camera management ---

// NewCamera creates a camera with the given viewport and adds it to the scene.
func (s *Scene) NewCamera(viewport types.Rect) *camera.Camera {
	cam := camera.NewCamera(viewport)
	s.Cameras = append(s.Cameras, cam)
	return cam
}

// RemoveCamera removes a camera from the scene.
func (s *Scene) RemoveCamera(cam *camera.Camera) {
	for i, c := range s.Cameras {
		if c == cam {
			s.Cameras = append(s.Cameras[:i], s.Cameras[i+1:]...)
			return
		}
	}
}

// --- Tween management ---

// RegisterTween adds a tween group to the scene's managed list.
func (s *Scene) RegisterTween(g *TweenGroup) {
	s.Tweens = append(s.Tweens, g)
}

// TickTweens advances all managed tweens and compacts out completed ones.
func (s *Scene) TickTweens(dt float32) {
	n := 0
	for _, g := range s.Tweens {
		g.Tick(dt)
		if !g.Done {
			s.Tweens[n] = g
			n++
		}
	}
	for i := n; i < len(s.Tweens); i++ {
		s.Tweens[i] = nil
	}
	s.Tweens = s.Tweens[:n]
}

// --- Screenshot ---

// Screenshot queues a labeled screenshot to be captured at the end of Draw.
func (s *Scene) Screenshot(label string) {
	s.ScreenshotQueue = append(s.ScreenshotQueue, label)
}

// --- Update ---

// Update processes input, advances animations, and simulates particles.
func (s *Scene) Update() {
	dt := float32(1.0 / float64(ebiten.TPS()))

	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	s.TransformsReady = true

	for _, cam := range s.Cameras {
		cam.Update(dt)
	}
	UpdateNodesAndParticles(s.Root, float64(dt))
	s.TickTweens(dt)

	if s.TestRunnerRef != nil {
		s.TestRunnerRef.Step(StepAction{
			Screenshot:  s.Screenshot,
			InjectClick: s.Input.InjectClick,
			InjectDrag:  s.Input.InjectDrag,
			QueueLen:    func() int { return len(s.Input.InjectQueue) },
		})
	}

	s.Input.ProcessInput(s.Root)

	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
}

// --- Draw ---

// Draw traverses the scene tree, emits render commands, sorts them, and submits
// batches to the given screen image.
func (s *Scene) Draw(screen *ebiten.Image) {
	if len(s.Cameras) == 0 {
		s.drawWithCamera(screen, nil)
	} else {
		for _, cam := range s.Cameras {
			vp := cam.Viewport
			viewportImg := screen.SubImage(image.Rect(
				int(vp.X), int(vp.Y),
				int(vp.X+vp.Width), int(vp.Y+vp.Height),
			)).(*ebiten.Image)
			s.drawWithCamera(viewportImg, cam)
		}
	}

	FlushScreenshots(screen, s.ScreenshotQueue, s.ScreenshotDir)
	s.ScreenshotQueue = s.ScreenshotQueue[:0]
}

func (s *Scene) drawWithCamera(target *ebiten.Image, cam *camera.Camera) {
	if !s.TransformsReady {
		node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
		s.TransformsReady = true
	}

	p := &s.Pipeline
	p.Commands = p.Commands[:0]
	p.CommandsDirtyThisFrame = false
	p.AntiAlias = s.AntiAlias

	if cam != nil {
		p.ViewTransform = cam.ComputeViewMatrix()
		p.CullActive = cam.CullEnabled
		if cam.CullEnabled {
			p.CullBounds = cam.Viewport
		}
	} else {
		p.ViewTransform = node.IdentityTransform
		p.CullActive = false
	}

	var stats DebugStats
	var t0 time.Time

	if s.Debug {
		t0 = time.Now()
	}

	treeOrder := 0
	p.Traverse(s.Root, &treeOrder)

	if s.Debug {
		stats.TraverseTime = time.Since(t0)
		t0 = time.Now()
	}

	if p.CommandsDirtyThisFrame {
		p.Sort()
	}

	if s.Debug {
		stats.SortTime = time.Since(t0)
		stats.CommandCount = len(p.Commands)
		t0 = time.Now()
	}

	if p.BatchMode == render.BatchModeCoalesced {
		p.SubmitBatchesCoalesced(target)
	} else {
		p.SubmitBatches(target)
	}

	if s.Debug {
		stats.SubmitTime = time.Since(t0)
		stats.BatchCount = CountBatches(p.Commands)
		if p.BatchMode == render.BatchModeCoalesced {
			stats.DrawCallCount = CountDrawCallsCoalesced(p.Commands)
		} else {
			stats.DrawCallCount = CountDrawCalls(p.Commands)
		}
		DebugLog(stats)
	}

	p.ReleaseDeferred()
}
