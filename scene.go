package willow

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/core"
	"github.com/phanxgames/willow/internal/input"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/render"
)

// BatchMode controls how the render pipeline submits draw calls.
type BatchMode = render.BatchMode

const (
	BatchModeCoalesced = render.BatchModeCoalesced
	BatchModeImmediate = render.BatchModeImmediate
)

// EntityStore is the interface for optional ECS integration.
// Aliased from internal/core.EntityStore.
type EntityStore = core.EntityStore

// InteractionEvent carries interaction data for the ECS bridge.
// Aliased from internal/core.InteractionEvent.
type InteractionEvent = core.InteractionEvent

// CallbackHandle allows removing a registered scene-level callback.
// Aliased from internal/input.CallbackHandle.
type CallbackHandle = input.CallbackHandle

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers. Aliased from internal/core.Scene; all methods live there.
type Scene = core.Scene

// NewScene creates a new scene with a pre-created root container and wires
// all necessary function pointers.
func NewScene() *Scene {
	root := NewContainer("root")
	root.Interactable = true
	s := core.NewScene(root)
	root.Scene_ = s

	// Wire the ScreenToWorld function pointer so the input manager can
	// convert screen coords to world coords via the primary camera.
	input.ScreenToWorldFn = func(sx, sy float64) (float64, float64) {
		if len(s.Cameras) > 0 {
			return s.Cameras[0].ScreenToWorld(sx, sy)
		}
		return sx, sy
	}

	// Wire the RegisterPage function pointer so core.Scene.RegisterPage works.
	core.RegisterPageFn = func(index int, img *ebiten.Image) {
		atlasManager().RegisterPage(index, img)
	}

	return s
}

// Root returns the scene's root container node. The root node cannot be
// removed or disposed; it always exists for the lifetime of the Scene.
// This is a convenience shim to preserve the Root() method name.
func Root(s *Scene) *node.Node {
	return s.Root
}

// Cameras returns the scene's camera list. The returned slice MUST NOT be mutated.
// This is a convenience shim to preserve the Cameras() method name.
func Cameras(s *Scene) []*camera.Camera {
	return s.Cameras
}

// LoadSceneAtlas parses TexturePacker JSON, registers atlas pages with the scene,
// and returns the Atlas for region lookups. Pages are registered starting at
// the next available page index.
func LoadSceneAtlas(s *Scene, jsonData []byte, pages []*ebiten.Image) (*Atlas, error) {
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
		for name, r := range atlas.Regions {
			r.Page += uint16(startIndex)
			atlas.Regions[name] = r
		}
	}
	return atlas, nil
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
	if g.Scene_.UpdateFunc != nil {
		if err := g.Scene_.UpdateFunc(); err != nil {
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
		screen.Fill(render.ColorToRGBA(g.Scene_.ClearColor))
	}
	g.Scene_.Draw(screen)
	// Draw FPS widget in screen space (unaffected by cameras).
	if g.fpsWid != nil && g.fpsWid.CustomImage() != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(g.fpsWid.X_, g.fpsWid.Y_)
		screen.DrawImage(g.fpsWid.CustomImage(), &op)
	}
	if g.Scene_.PostDrawFunc != nil {
		g.Scene_.PostDrawFunc(screen)
	}
}

func (g *gameShell) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.w, g.h
}
