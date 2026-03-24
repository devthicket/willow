package core

import (
	"fmt"
	"image"
	"os"
	"time"

	"github.com/devthicket/willow/internal/camera"
	"github.com/devthicket/willow/internal/input"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/render"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

const DefaultCommandCap = 4096

// Function pointers wired by the root package for operations that core cannot
// perform directly (atlas management, etc.).
var (
	// RegisterPageFn registers an atlas page image at the given index.
	RegisterPageFn func(index int, img *ebiten.Image)

	// LoadAtlasFn parses TexturePacker JSON and registers atlas pages.
	LoadAtlasFn func(jsonData []byte, pages []*ebiten.Image, startIndex int)
)

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers. It composes render.Pipeline, input.Manager, and cameras.
type Scene struct {
	Root  *node.Node
	Debug bool

	TransformsReady bool

	// ClearColor is the background color used to fill the screen each frame
	// when the scene is run via Run. If left at the zero value (transparent
	// black), the screen is not filled, resulting in a black background.
	ClearColor types.Color

	// Cameras
	Cameras []*camera.Camera

	// Render pipeline — owns command buffer, sort/batch buffers, RT pool,
	// culling state, and batch submission.
	Pipeline render.Pipeline

	// Input state
	Input *input.Manager

	// Managed tweens (auto-ticked during Update)
	Tweens []*TweenGroup

	// Screenshot capture
	ScreenshotQueue []string
	ScreenshotDir   string

	// GIF recording
	GifRecorder *GifRecorder

	// Test runner
	TestRunnerRef *TestRunner

	// Injected keyboard input for test automation.
	InjectedChars []rune       // synthetic characters consumed by AppendInjectedChars
	InjectedKeys  []ebiten.Key // synthetic key presses consumed by IsInjectedKeyPressed

	// Held keys for data-driven autotest (key_down / key_up actions).
	HeldKeys map[ebiten.Key]struct{}

	// User callbacks set via SetUpdateFunc / SetPostDrawFunc / SetOnResize.
	UpdateFunc   func() error
	PostDrawFunc func(screen *ebiten.Image)
	OnResize     func(w, h int)

	// Scene lifecycle hooks for SceneManager.
	OnEnterFn func()
	OnExitFn  func()

	// ECS bridge
	store EntityStore
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

// --- Accessors ---

// RootNode returns the scene's root container node.
func (s *Scene) RootNode() *node.Node {
	return s.Root
}

// GetCameras returns the scene's camera list. The returned slice MUST NOT be mutated.
func (s *Scene) GetCameras() []*camera.Camera {
	return s.Cameras
}

// PrimaryCamera returns the first camera, or nil if there are no cameras.
func (s *Scene) PrimaryCamera() *camera.Camera {
	if len(s.Cameras) > 0 {
		return s.Cameras[0]
	}
	return nil
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

// --- Debug ---

// SetDebugMode enables or disables debug mode. When enabled, disposed-node
// access panics, tree depth and child count warnings are printed, and per-frame
// timing stats are logged to stderr.
func (s *Scene) SetDebugMode(enabled bool) {
	s.Debug = enabled
	node.Debug = enabled
}

// --- Batch mode ---

// SetBatchMode sets the draw-call batching strategy.
func (s *Scene) SetBatchMode(mode render.BatchMode) { s.Pipeline.BatchMode = mode }

// GetBatchMode returns the current draw-call batching strategy.
func (s *Scene) GetBatchMode() render.BatchMode { return s.Pipeline.BatchMode }

// --- User callbacks ---

// SetUpdateFunc registers a callback that is called once per tick before
// Scene.Update when the scene is run via Run.
func (s *Scene) SetUpdateFunc(fn func() error) {
	s.UpdateFunc = fn
}

// SetPostDrawFunc registers a callback that is called after Scene.Draw.
func (s *Scene) SetPostDrawFunc(fn func(*ebiten.Image)) {
	s.PostDrawFunc = fn
}

// SetOnResize registers a callback that is called when the window is resized.
// The callback receives the new logical width and height in pixels.
func (s *Scene) SetOnResize(fn func(w, h int)) {
	s.OnResize = fn
}

// SetOnEnter registers a callback called when this scene becomes active.
func (s *Scene) SetOnEnter(fn func()) {
	s.OnEnterFn = fn
}

// SetOnExit registers a callback called when this scene is being left.
func (s *Scene) SetOnExit(fn func()) {
	s.OnExitFn = fn
}

// --- Atlas / page registration (delegated via function pointers) ---

// RegisterPage stores an atlas page image at the given index.
func (s *Scene) RegisterPage(index int, img *ebiten.Image) {
	if RegisterPageFn != nil {
		RegisterPageFn(index, img)
	}
}

// --- Screenshot ---

// Screenshot queues a labeled screenshot to be captured at the end of Draw.
func (s *Scene) Screenshot(label string) {
	s.ScreenshotQueue = append(s.ScreenshotQueue, label)
}

// --- GIF recording ---

// StartGif begins recording frames as an animated GIF.
func (s *Scene) StartGif(label string) {
	s.StartGifWithConfig(label, GifConfig{})
}

// StartGifWithConfig begins recording with custom settings.
func (s *Scene) StartGifWithConfig(label string, cfg GifConfig) {
	if s.GifRecorder != nil {
		return // already recording
	}
	dir := s.ScreenshotDir
	s.GifRecorder = NewGifRecorder(label, dir, cfg)
}

// StopGif ends recording and writes the GIF file.
func (s *Scene) StopGif() {
	if s.GifRecorder == nil {
		return
	}
	if err := s.GifRecorder.Finish(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[willow] gif: %v\n", err)
	}
	s.GifRecorder = nil
}

// --- Test runner ---

// SetTestRunner attaches a TestRunner to the scene.
func (s *Scene) SetTestRunner(runner *TestRunner) {
	s.TestRunnerRef = runner
}

// --- ECS entity store ---

// SetEntityStore sets the optional ECS bridge.
func (s *Scene) SetEntityStore(store EntityStore) {
	s.store = store
	if store != nil {
		input.EmitInteractionEventFn = func(
			eventType types.EventType, n *node.Node, wx, wy, lx, ly float64,
			button types.MouseButton, mods types.KeyModifiers,
			drag node.DragContext, pinch node.PinchContext,
		) {
			s.emitInteractionEvent(eventType, n, wx, wy, lx, ly, button, mods, drag, pinch)
		}
	} else {
		input.EmitInteractionEventFn = nil
	}
}

func (s *Scene) emitInteractionEvent(
	eventType types.EventType, n *node.Node, wx, wy, lx, ly float64,
	button types.MouseButton, mods types.KeyModifiers,
	drag node.DragContext, pinch node.PinchContext,
) {
	if s.store == nil {
		return
	}
	if eventType != types.EventPinch && (n == nil || n.EntityID == 0) {
		return
	}
	var entityID uint32
	if n != nil {
		entityID = n.EntityID
	}
	s.store.EmitEvent(InteractionEvent{
		Type:         eventType,
		EntityID:     entityID,
		GlobalX:      wx,
		GlobalY:      wy,
		LocalX:       lx,
		LocalY:       ly,
		Button:       button,
		Modifiers:    mods,
		StartX:       drag.StartX,
		StartY:       drag.StartY,
		DeltaX:       drag.DeltaX,
		DeltaY:       drag.DeltaY,
		ScreenDeltaX: drag.ScreenDeltaX,
		ScreenDeltaY: drag.ScreenDeltaY,
		Scale:        pinch.Scale,
		ScaleDelta:   pinch.ScaleDelta,
		Rotation:     pinch.Rotation,
		RotDelta:     pinch.RotDelta,
	})
}

// --- Input: state queries ---

// PointerPosition returns the current pointer position in screen space.
func (s *Scene) PointerPosition() (x, y float64) {
	return s.Input.PointerPosition()
}

// IsPointerDown reports whether the given mouse button is currently pressed.
func (s *Scene) IsPointerDown(button types.MouseButton) bool {
	return s.Input.IsPointerDown(button)
}

// --- Input: pointer capture ---

// CapturePointer routes all events for pointerID to the given node.
func (s *Scene) CapturePointer(pointerID int, n *node.Node) {
	s.Input.CapturePointer(pointerID, n)
}

// ReleasePointer stops routing events for pointerID to a captured node.
func (s *Scene) ReleasePointer(pointerID int) {
	s.Input.ReleasePointer(pointerID)
}

// SetDragDeadZone sets the minimum movement in pixels before a drag starts.
func (s *Scene) SetDragDeadZone(pixels float64) {
	s.Input.SetDragDeadZone(pixels)
}

// --- Input: event registration ---

// CallbackHandle = input.CallbackHandle (re-exported via root alias).

// OnPointerDown registers a scene-level callback for pointer down events.
func (s *Scene) OnPointerDown(fn func(node.PointerContext)) input.CallbackHandle {
	return s.Input.OnPointerDown(fn)
}

// OnPointerUp registers a scene-level callback for pointer up events.
func (s *Scene) OnPointerUp(fn func(node.PointerContext)) input.CallbackHandle {
	return s.Input.OnPointerUp(fn)
}

// OnPointerMove registers a scene-level callback for pointer move events.
func (s *Scene) OnPointerMove(fn func(node.PointerContext)) input.CallbackHandle {
	return s.Input.OnPointerMove(fn)
}

// OnPointerEnter registers a scene-level callback for pointer enter events.
func (s *Scene) OnPointerEnter(fn func(node.PointerContext)) input.CallbackHandle {
	return s.Input.OnPointerEnter(fn)
}

// OnPointerLeave registers a scene-level callback for pointer leave events.
func (s *Scene) OnPointerLeave(fn func(node.PointerContext)) input.CallbackHandle {
	return s.Input.OnPointerLeave(fn)
}

// OnClick registers a scene-level callback for click events.
func (s *Scene) OnClick(fn func(node.ClickContext)) input.CallbackHandle {
	return s.Input.OnClick(fn)
}

// OnBackgroundClick registers a scene-level callback that fires when a click
// lands on empty space.
func (s *Scene) OnBackgroundClick(fn func(node.ClickContext)) input.CallbackHandle {
	return s.Input.OnBackgroundClick(fn)
}

// OnDragStart registers a scene-level callback for drag start events.
func (s *Scene) OnDragStart(fn func(node.DragContext)) input.CallbackHandle {
	return s.Input.OnDragStart(fn)
}

// OnDrag registers a scene-level callback for drag events.
func (s *Scene) OnDrag(fn func(node.DragContext)) input.CallbackHandle {
	return s.Input.OnDrag(fn)
}

// OnDragEnd registers a scene-level callback for drag end events.
func (s *Scene) OnDragEnd(fn func(node.DragContext)) input.CallbackHandle {
	return s.Input.OnDragEnd(fn)
}

// OnPinch registers a scene-level callback for pinch events.
func (s *Scene) OnPinch(fn func(node.PinchContext)) input.CallbackHandle {
	return s.Input.OnPinch(fn)
}

// --- Input injection ---

// InjectPress queues a pointer press event at the given screen coordinates.
func (s *Scene) InjectPress(x, y float64) {
	s.Input.InjectPress(x, y)
}

// InjectMove queues a pointer move event at the given screen coordinates.
func (s *Scene) InjectMove(x, y float64) {
	s.Input.InjectMove(x, y)
}

// InjectRelease queues a pointer release event at the given screen coordinates.
func (s *Scene) InjectRelease(x, y float64) {
	s.Input.InjectRelease(x, y)
}

// InjectHover queues a free pointer move (no button held) to trigger
// OnPointerEnter / OnPointerLeave without pressing a button.
func (s *Scene) InjectHover(x, y float64) {
	s.Input.InjectHover(x, y)
}

// InjectClick queues a press followed by a release at the given coordinates.
func (s *Scene) InjectClick(x, y float64) {
	s.Input.InjectClick(x, y)
}

// InjectDrag queues a full drag sequence over the given number of frames.
func (s *Scene) InjectDrag(fromX, fromY, toX, toY float64, frames int) {
	s.Input.InjectDrag(fromX, fromY, toX, toY, frames)
}

// InjectText queues synthetic character input. Each rune is appended to
// InjectedChars, consumed one batch per frame by AppendInjectedChars.
func (s *Scene) InjectText(text string) {
	s.InjectedChars = append(s.InjectedChars, []rune(text)...)
}

// InjectKey queues a synthetic key press consumed by IsInjectedKeyPressed.
func (s *Scene) InjectKey(key ebiten.Key) {
	s.InjectedKeys = append(s.InjectedKeys, key)
}

// AppendInjectedChars appends all injected characters to buf, then clears
// the queue. Mirrors ebiten.AppendInputChars for synthetic input.
func (s *Scene) AppendInjectedChars(buf []rune) []rune {
	buf = append(buf, s.InjectedChars...)
	s.InjectedChars = s.InjectedChars[:0]
	return buf
}

// IsInjectedKeyPressed checks if key is in the injected keys queue and
// removes it. Returns true if found.
func (s *Scene) IsInjectedKeyPressed(key ebiten.Key) bool {
	for i, k := range s.InjectedKeys {
		if k == key {
			s.InjectedKeys = append(s.InjectedKeys[:i], s.InjectedKeys[i+1:]...)
			return true
		}
	}
	return false
}

// HasInjectedInput returns true if there are any pending injected chars or keys.
func (s *Scene) HasInjectedInput() bool {
	return len(s.InjectedChars) > 0 || len(s.InjectedKeys) > 0
}

// KeyDown marks a key as held.  Used by test runner key_down actions.
func (s *Scene) KeyDown(key ebiten.Key) {
	if s.HeldKeys == nil {
		s.HeldKeys = make(map[ebiten.Key]struct{})
	}
	s.HeldKeys[key] = struct{}{}
}

// KeyUp releases a held key.  Used by test runner key_up actions.
func (s *Scene) KeyUp(key ebiten.Key) {
	delete(s.HeldKeys, key)
}

// IsKeyPressed returns true if a key is pressed on the real keyboard or
// held via the test runner (key_down/key_up).  Demos should use this
// instead of ebiten.IsKeyPressed to support data-driven autotests.
func (s *Scene) IsKeyPressed(key ebiten.Key) bool {
	if _, ok := s.HeldKeys[key]; ok {
		return true
	}
	for _, k := range s.InjectedKeys {
		if k == key {
			return true
		}
	}
	return ebiten.IsKeyPressed(key)
}

// --- Update ---

// Update processes input, advances animations, and simulates particles.
func (s *Scene) Update() {
	dt := float32(1.0 / float64(ebiten.TPS()))

	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	s.TransformsReady = true
	node.AnyTransformDirty = false

	for _, cam := range s.Cameras {
		cam.Update(dt)
	}
	UpdateNodesAndParticles(s.Root, float64(dt))
	s.TickTweens(dt)

	if s.TestRunnerRef != nil {
		s.TestRunnerRef.Step(StepAction{
			Screenshot:  s.Screenshot,
			StartGif:    s.StartGifWithConfig,
			StopGif:     s.StopGif,
			InjectClick: s.Input.InjectClick,
			InjectDrag:  s.Input.InjectDrag,
			InjectMove:  s.Input.InjectHover,
			InjectText:  s.InjectText,
			InjectKey:   func(key string) { s.InjectKey(KeyFromName(key)) },
			KeyDown:     func(key string) { s.KeyDown(KeyFromName(key)) },
			KeyUp:       func(key string) { s.KeyUp(KeyFromName(key)) },
			QueueLen: func() int {
				return len(s.Input.InjectQueue) + len(s.InjectedChars) + len(s.InjectedKeys)
			},
		})
	}

	s.Input.ProcessInput(s.Root)

	if node.AnyTransformDirty {
		node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	}
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

	if s.GifRecorder != nil {
		s.GifRecorder.CaptureFrame(screen)
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
