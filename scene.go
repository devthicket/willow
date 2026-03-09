package willow

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/input"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/render"
)

// EntityStore is the interface for optional ECS integration.
// When set on a Scene via SetEntityStore, interaction events on nodes with
// a non-zero EntityID are forwarded to the ECS.
type EntityStore interface {
	// EmitEvent delivers an interaction event to the ECS.
	EmitEvent(event InteractionEvent)
}

// InteractionEvent carries interaction data for the ECS bridge.
type InteractionEvent struct {
	Type      EventType    // which kind of interaction occurred
	EntityID  uint32       // the ECS entity associated with the hit node
	GlobalX   float64      // pointer X in world coordinates
	GlobalY   float64      // pointer Y in world coordinates
	LocalX    float64      // pointer X in the hit node's local coordinates
	LocalY    float64      // pointer Y in the hit node's local coordinates
	Button    MouseButton  // which mouse button is involved
	Modifiers KeyModifiers // keyboard modifier keys held during the event
	// Drag fields (valid for EventDragStart, EventDrag, EventDragEnd)
	StartX       float64 // world X where the drag began
	StartY       float64 // world Y where the drag began
	DeltaX       float64 // X movement since the previous drag event
	DeltaY       float64 // Y movement since the previous drag event
	ScreenDeltaX float64 // X movement in screen pixels since the previous drag event
	ScreenDeltaY float64 // Y movement in screen pixels since the previous drag event
	// Pinch fields (valid for EventPinch)
	Scale      float64 // cumulative scale factor since pinch start
	ScaleDelta float64 // frame-to-frame scale change
	Rotation   float64 // cumulative rotation in radians since pinch start
	RotDelta   float64 // frame-to-frame rotation change in radians
}

// BatchMode controls how the render pipeline submits draw calls.
type BatchMode = render.BatchMode

const (
	BatchModeCoalesced = render.BatchModeCoalesced
	BatchModeImmediate = render.BatchModeImmediate
)

const defaultCommandCap = 4096

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers.
type Scene struct {
	root  *Node
	store EntityStore
	debug bool

	// transformsReady is set to true after the first updateWorldTransform call.
	// Used by Draw to ensure transforms are computed even if Update hasn't run.
	transformsReady bool

	// ClearColor is the background color used to fill the screen each frame
	// when the scene is run via [Run]. If left at the zero value (transparent
	// black), the screen is not filled, resulting in a black background.
	ClearColor Color

	updateFunc   func() error               // user callback set via SetUpdateFunc
	postDrawFunc func(screen *ebiten.Image) // user callback after Draw, set via SetPostDrawFunc

	// Cameras
	cameras []*Camera

	// Render pipeline — owns command buffer, sort/batch buffers, RT pool,
	// culling state, and batch submission. Replaces the per-field render state
	// that was previously defined directly on Scene.
	pipeline render.Pipeline

	// Input state (Phase 08)
	handlers     handlerRegistry
	captured     [maxPointers]*Node
	pointers     [maxPointers]pointerState
	hitBuf       []*Node
	dragDeadZone float64
	touchMap     [maxPointers]ebiten.TouchID
	touchUsed    [maxPointers]bool
	prevTouchIDs []ebiten.TouchID
	pinch        pinchState

	// Managed tweens (auto-ticked during Update)
	tweens []*TweenGroup

	// Screenshot capture (debug tool)
	screenshotQueue []string
	ScreenshotDir   string

	// Input injection (debug/testing tool)
	injectQueue []syntheticPointerEvent

	// Test runner (automated visual testing)
	testRunner *TestRunner

	// AntiAlias enables anti-aliased edges on DrawTriangles calls (meshes,
	// tilemaps, batched sprites, particles). Toggle at runtime or set via
	// RunConfig for initial setup.
	AntiAlias bool
}

// NewScene creates a new scene with a pre-created root container.
func NewScene() *Scene {
	root := NewContainer("root")
	root.Interactable = true
	s := &Scene{
		root:         root,
		dragDeadZone: defaultDragDeadZone,
		pipeline: render.Pipeline{
			Commands: make([]RenderCommand, 0, defaultCommandCap),
			SortBuf:  make([]RenderCommand, 0, defaultCommandCap),
		},
		ScreenshotDir: "screenshots",
	}
	root.Scene_ = s

	// Wire per-scene input function pointers.
	input.ScreenToWorldFn = func(sx, sy float64) (float64, float64) {
		if len(s.cameras) > 0 {
			return s.cameras[0].ScreenToWorld(sx, sy)
		}
		return sx, sy
	}
	return s
}

// Root returns the scene's root container node. The root node cannot be
// removed or disposed; it always exists for the lifetime of the Scene.
func (s *Scene) Root() *Node {
	return s.root
}

// RunConfig holds optional configuration for [Run].
type RunConfig struct {
	// Title sets the window title. Ignored on platforms without a title bar.
	Title string
	// Width and Height set the window size in device-independent pixels.
	// If zero, defaults to 640x480.
	Width, Height int

	// ShowFPS enables a small FPS/TPS widget in the top-left corner.
	ShowFPS bool

	// AntiAlias enables anti-aliased edges on all DrawTriangles calls.
	AntiAlias bool
}

// SetUpdateFunc registers a callback that is called once per tick before
// [Scene.Update] when the scene is run via [Run]. Use it for game-specific
// logic (movement, spawning, etc.). Pass nil to clear.
func (s *Scene) SetUpdateFunc(fn func() error) {
	s.updateFunc = fn
}

// SetPostDrawFunc registers a callback that is called after [Scene.Draw]
// (and after the FPS widget) when the scene is run via [Run]. Use it for
// screen-space debug overlays (e.g. ebitenutil.DebugPrintAt). Pass nil to clear.
func (s *Scene) SetPostDrawFunc(fn func(screen *ebiten.Image)) {
	s.postDrawFunc = fn
}

// Run is a convenience entry point that creates an Ebitengine game loop around
// the given Scene. It configures the window and calls [ebiten.RunGame].
//
// For full control over the game loop, skip Run and implement [ebiten.Game]
// yourself, calling [Scene.Update] and [Scene.Draw] directly.
func Run(scene *Scene, cfg RunConfig) error {
	w, h := cfg.Width, cfg.Height
	if w == 0 {
		w = 640
	}
	if h == 0 {
		h = 480
	}
	ebiten.SetWindowSize(w, h)
	if cfg.Title != "" {
		ebiten.SetWindowTitle(cfg.Title)
	}
	scene.AntiAlias = cfg.AntiAlias
	g := &gameShell{Scene_: scene, w: w, h: h}
	if cfg.ShowFPS {
		g.fpsWid = NewFPSWidget()
		g.fpsWid.X_, g.fpsWid.Y_ = 8, 8
	}

	return ebiten.RunGame(g)
}

// gameShell implements [ebiten.Game] by delegating to a Scene.
type gameShell struct {
	Scene_ *Scene
	w, h   int
	fpsWid *Node // screen-space FPS overlay (not in scene graph)
}

func (g *gameShell) Update() error {
	if g.Scene_.updateFunc != nil {
		if err := g.Scene_.updateFunc(); err != nil {
			return err
		}
	}
	g.Scene_.Update()
	if g.fpsWid != nil && g.fpsWid.OnUpdate != nil {
		g.fpsWid.OnUpdate(1.0 / float64(ebiten.TPS()))
	}
	return nil
}

func (g *gameShell) Draw(screen *ebiten.Image) {
	if g.Scene_.ClearColor.A() > 0 {
		screen.Fill(colorToRGBA(g.Scene_.ClearColor))
	}
	g.Scene_.Draw(screen)
	// Draw FPS widget in screen space (unaffected by cameras).
	if g.fpsWid != nil && g.fpsWid.CustomImage() != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(g.fpsWid.X_, g.fpsWid.Y_)
		screen.DrawImage(g.fpsWid.CustomImage(), &op)
	}
	if g.Scene_.postDrawFunc != nil {
		g.Scene_.postDrawFunc(screen)
	}
}

func (g *gameShell) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.w, g.h
}

// Update processes input, advances animations, and simulates particles.
func (s *Scene) Update() {
	dt := float32(1.0 / float64(ebiten.TPS()))

	// Refresh world transforms first so camera follow targets and hit testing
	// have accurate positions this frame.
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)
	s.transformsReady = true

	for _, cam := range s.cameras {
		cam.Update(dt)
	}
	updateNodesAndParticles(s.root, float64(dt))
	s.tickTweens(dt)
	if s.testRunner != nil {
		s.testRunner.step(s)
	}
	s.processInput()

	// Re-compute world transforms for any nodes added or moved during
	// OnUpdate callbacks or input processing. Only dirty subtrees are
	// traversed, so this is cheap when nothing changed.
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)
}

func updateNodesAndParticles(n *Node, dt float64) {
	if !n.Visible_ {
		return
	}
	if n.OnUpdate != nil {
		n.OnUpdate(dt)
	}
	if n.Type == NodeTypeParticleEmitter && n.Emitter != nil {
		if n.Emitter.Config.WorldSpace {
			n.Emitter.WorldX = n.WorldTransform[4]
			n.Emitter.WorldY = n.WorldTransform[5]
		}
		n.Emitter.Update(dt)
		// Active particles under a CacheAsTree ancestor must invalidate the
		// cache each frame so the emitter's commands are re-emitted.
		if n.Emitter.Alive > 0 {
			invalidateAncestorCache(n)
		}
	}
	for _, child := range n.Children_ {
		updateNodesAndParticles(child, dt)
	}
}

// Draw traverses the scene tree, emits render commands, sorts them, and submits
// batches to the given screen image.
func (s *Scene) Draw(screen *ebiten.Image) {
	if len(s.cameras) == 0 {
		// No explicit cameras: use implicit identity camera, full screen.
		s.drawWithCamera(screen, nil)
	} else {
		for _, cam := range s.cameras {
			vp := cam.Viewport
			viewportImg := screen.SubImage(image.Rect(
				int(vp.X), int(vp.Y),
				int(vp.X+vp.Width), int(vp.Y+vp.Height),
			)).(*ebiten.Image)
			s.drawWithCamera(viewportImg, cam)
		}
	}

	s.flushScreenshots(screen)
}

// drawWithCamera renders the scene from a camera's perspective.
// If cam is nil, uses identity view (no camera).
func (s *Scene) drawWithCamera(target *ebiten.Image, cam *Camera) {
	// Ensure world transforms are computed if Draw is called before Update
	// (e.g. manual game loop that skips the first Update call).
	if !s.transformsReady {
		updateWorldTransform(s.root, identityTransform, 1.0, false, false)
		s.transformsReady = true
	}

	p := &s.pipeline
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
		p.ViewTransform = identityTransform
		p.CullActive = false
	}

	var stats debugStats
	var t0 time.Time

	if s.debug {
		t0 = time.Now()
	}

	treeOrder := 0
	p.Traverse(s.root, &treeOrder)

	if s.debug {
		stats.traverseTime = time.Since(t0)
		t0 = time.Now()
	}

	if p.CommandsDirtyThisFrame {
		p.Sort()
	}

	if s.debug {
		stats.sortTime = time.Since(t0)
		stats.commandCount = len(p.Commands)
		t0 = time.Now()
	}

	p.SubmitBatches(target)

	if s.debug {
		stats.submitTime = time.Since(t0)
		stats.batchCount = countBatches(p.Commands)
		if p.BatchMode == BatchModeCoalesced {
			stats.drawCallCount = countDrawCallsCoalesced(p.Commands)
		} else {
			stats.drawCallCount = countDrawCalls(p.Commands)
		}
		s.debugLog(stats)
	}

	p.ReleaseDeferred()
}

// NewCamera creates a camera with the given viewport and adds it to the scene.
func (s *Scene) NewCamera(viewport Rect) *Camera {
	cam := camera.NewCamera(viewport)
	s.cameras = append(s.cameras, cam)
	return cam
}

// RemoveCamera removes a camera from the scene.
func (s *Scene) RemoveCamera(cam *Camera) {
	for i, c := range s.cameras {
		if c == cam {
			s.cameras = append(s.cameras[:i], s.cameras[i+1:]...)
			return
		}
	}
}

// Cameras returns the scene's camera list. The returned slice MUST NOT be mutated.
func (s *Scene) Cameras() []*Camera {
	return s.cameras
}

// registerTween adds a tween group to the scene's managed list.
func (s *Scene) registerTween(g *TweenGroup) {
	s.tweens = append(s.tweens, g)
}

// tickTweens advances all managed tweens and compacts out completed ones.
func (s *Scene) tickTweens(dt float32) {
	n := 0
	for _, g := range s.tweens {
		g.tick(dt)
		if !g.Done {
			s.tweens[n] = g
			n++
		}
	}
	// Nil out stale tail entries to avoid retaining references.
	for i := n; i < len(s.tweens); i++ {
		s.tweens[i] = nil
	}
	s.tweens = s.tweens[:n]
}

// SetEntityStore sets the optional ECS bridge.
func (s *Scene) SetEntityStore(store EntityStore) {
	s.store = store
	if store != nil {
		input.EmitInteractionEventFn = func(eventType EventType, n *Node, wx, wy, lx, ly float64,
			button MouseButton, mods KeyModifiers, drag DragContext, pinch PinchContext) {
			s.emitInteractionEvent(eventType, n, wx, wy, lx, ly, button, mods, drag, pinch)
		}
	} else {
		input.EmitInteractionEventFn = nil
	}
}

// SetDebugMode enables or disables debug mode. When enabled, disposed-node
// access panics, tree depth and child count warnings are printed, and per-frame
// timing stats are logged to stderr.
func (s *Scene) SetDebugMode(enabled bool) {
	s.debug = enabled
	globalDebug = enabled
	node.Debug = enabled
}

// SetBatchMode sets the draw-call batching strategy.
func (s *Scene) SetBatchMode(mode BatchMode) { s.pipeline.BatchMode = mode }

// BatchMode returns the current draw-call batching strategy.
func (s *Scene) GetBatchMode() BatchMode { return s.pipeline.BatchMode }

// globalDebug mirrors the most recently set Scene debug flag so that node
// operations (which lack a Scene pointer) can check it cheaply. Only valid
// with a single Scene; multiple Scenes with differing debug modes will
// reflect whichever called SetDebugMode last.
var globalDebug bool

// RegisterPage stores an atlas page image at the given index.
// The render compiler uses these to SubImage sprite regions.
func (s *Scene) RegisterPage(index int, img *ebiten.Image) {
	atlasManager().RegisterPage(index, img)
}

// LoadAtlas parses TexturePacker JSON, registers atlas pages with the scene,
// and returns the Atlas for region lookups. Pages are registered starting at
// the next available page index.
func (s *Scene) LoadAtlas(jsonData []byte, pages []*ebiten.Image) (*Atlas, error) {
	atlas, err := LoadAtlas(jsonData, pages)
	if err != nil {
		return nil, err
	}
	am := atlasManager()
	startIndex := am.NextPage()
	for i, page := range pages {
		am.RegisterPage(startIndex+i, page)
	}
	// Remap region page indices to account for startIndex offset.
	if startIndex > 0 {
		for name, r := range atlas.regions {
			r.Page += uint16(startIndex)
			atlas.regions[name] = r
		}
	}
	return atlas, nil
}
