package willow

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/atlas"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/core"
	"github.com/phanxgames/willow/internal/filter"
	"github.com/phanxgames/willow/internal/input"
	"github.com/phanxgames/willow/internal/lighting"
	"github.com/phanxgames/willow/internal/mesh"
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/particle"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/text"
	"github.com/phanxgames/willow/internal/tilemap"
	"github.com/phanxgames/willow/internal/types"
	"github.com/tanema/gween/ease"
)

// ---------------------------------------------------------------------------
// Foundation types (internal/types)
// ---------------------------------------------------------------------------

// Color represents an RGBA color with components in [0, 1].
type Color = types.Color

// Vec2 is a 2D vector.
type Vec2 = types.Vec2

// Rect is an axis-aligned rectangle (origin top-left, Y down).
type Rect = types.Rect

// Range is a general-purpose min/max range.
type Range = types.Range

// BlendMode selects a compositing operation.
type BlendMode = types.BlendMode

// NodeType distinguishes rendering behavior for a Node.
type NodeType = types.NodeType

// EventType identifies a kind of interaction event.
type EventType = types.EventType

// MouseButton identifies a mouse button.
type MouseButton = types.MouseButton

// KeyModifiers is a bitmask of keyboard modifier keys.
type KeyModifiers = types.KeyModifiers

// TextAlign controls horizontal text alignment within a TextBlock.
type TextAlign = types.TextAlign

// TextureRegion describes a sub-rectangle within an atlas page.
type TextureRegion = types.TextureRegion

// CacheTreeMode controls how a cached subtree invalidates.
type CacheTreeMode = types.CacheTreeMode

// HitShape is implemented by custom hit-test shapes attached to a Node.
type HitShape = types.HitShape

// TweenConfig holds the duration and easing function for a tween.
type TweenConfig = types.TweenConfig

// EaseFunc is the signature for easing functions used by TweenConfig.
type EaseFunc = ease.TweenFunc

// ---------------------------------------------------------------------------
// Node types (internal/node)
// ---------------------------------------------------------------------------

// Node is the fundamental scene graph element.
type Node = node.Node

// PointerContext carries pointer event data.
type PointerContext = node.PointerContext

// ClickContext carries click event data.
type ClickContext = node.ClickContext

// DragContext carries drag event data.
type DragContext = node.DragContext

// PinchContext carries two-finger pinch/rotate gesture data.
type PinchContext = node.PinchContext

// NodeIndex is an opt-in registry for looking up nodes by name or tag.
type NodeIndex = node.NodeIndex

// ---------------------------------------------------------------------------
// Scene, Camera, Tweens (internal/core, camera)
// ---------------------------------------------------------------------------

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers.
type Scene = core.Scene

// Camera controls the view into the scene: position, zoom, rotation, viewport.
type Camera = camera.Camera

// TweenGroup animates up to 4 float64 fields on a Node simultaneously.
type TweenGroup = core.TweenGroup

// EntityStore is the interface for optional ECS integration.
type EntityStore = core.EntityStore

// InteractionEvent carries interaction data for the ECS bridge.
type InteractionEvent = core.InteractionEvent

// CallbackHandle allows removing a registered scene-level callback.
type CallbackHandle = input.CallbackHandle

// TestRunner sequences injected input events and screenshots across frames.
type TestRunner = core.TestRunner

// BatchMode controls how the render pipeline submits draw calls.
type BatchMode = render.BatchMode

// RunConfig holds optional configuration for [Run].
type RunConfig struct {
	Title         string
	Width, Height int
	ShowFPS       bool
	AntiAlias     bool
}

// ---------------------------------------------------------------------------
// Render types (internal/render)
// ---------------------------------------------------------------------------

// CommandType identifies the kind of render command.
type CommandType = render.CommandType

// RenderCommand is a single draw instruction emitted during scene traversal.
type RenderCommand = render.RenderCommand

// RenderTexture is a persistent offscreen canvas.
type RenderTexture = render.RenderTexture

// RenderTextureDrawOpts controls how an image or sprite is drawn onto a RenderTexture.
type RenderTextureDrawOpts = render.RenderTextureDrawOpts

// ---------------------------------------------------------------------------
// Atlas (internal/atlas)
// ---------------------------------------------------------------------------

// Atlas holds one or more atlas page images and a map of named regions.
type Atlas = atlas.Atlas

// PackerConfig controls the dynamic atlas packer.
type PackerConfig = atlas.PackerConfig

// ---------------------------------------------------------------------------
// Text (internal/text)
// ---------------------------------------------------------------------------

// Font is the interface for text measurement and layout.
type Font = text.Font

// DistanceFieldFont renders text from a pre-generated SDF or MSDF atlas.
type DistanceFieldFont = text.DistanceFieldFont

// PixelFont is a pixel-perfect bitmap font renderer.
type PixelFont = text.PixelFont

// TextEffects configures text effects (outline, glow, shadow).
type TextEffects = text.TextEffects

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock = text.TextBlock

// Glyph holds glyph metrics and atlas position.
type Glyph = text.Glyph

// SDFGenOptions configures SDF atlas generation.
type SDFGenOptions = text.SDFGenOptions

// GlyphBitmap holds a rasterized glyph and its metrics for atlas packing.
type GlyphBitmap = text.GlyphBitmap

// ---------------------------------------------------------------------------
// Particle (internal/particle)
// ---------------------------------------------------------------------------

// EmitterConfig controls how particles are spawned and behave.
type EmitterConfig = particle.EmitterConfig

// ParticleEmitter manages a pool of particles with CPU-based simulation.
type ParticleEmitter = particle.Emitter

// ---------------------------------------------------------------------------
// Filter (internal/filter)
// ---------------------------------------------------------------------------

// Filter is the interface for visual effects applied to a node.
type Filter = filter.Filter

// ColorMatrixFilter applies a 4x5 color matrix transformation.
type ColorMatrixFilter = filter.ColorMatrixFilter

// BlurFilter applies a Kawase iterative blur.
type BlurFilter = filter.BlurFilter

// OutlineFilter draws a multi-pixel outline around the source.
type OutlineFilter = filter.OutlineFilter

// PixelPerfectOutlineFilter draws a 1-pixel outline via Kage shader.
type PixelPerfectOutlineFilter = filter.PixelPerfectOutlineFilter

// PixelPerfectInlineFilter recolors edge pixels via Kage shader.
type PixelPerfectInlineFilter = filter.PixelPerfectInlineFilter

// PaletteFilter remaps pixel colors through a palette based on luminance.
type PaletteFilter = filter.PaletteFilter

// CustomShaderFilter wraps a user-provided Kage shader.
type CustomShaderFilter = filter.CustomShaderFilter

// ---------------------------------------------------------------------------
// Tilemap (internal/tilemap)
// ---------------------------------------------------------------------------

// TileMapLayer is a single layer of tile data.
type TileMapLayer = tilemap.Layer

// TileMapViewport is a scene graph node that manages a viewport into a tilemap.
type TileMapViewport = tilemap.Viewport

// TileLayerConfig holds the parameters for creating a tile layer.
type TileLayerConfig = tilemap.LayerConfig

// AnimFrame describes a single frame in a tile animation sequence.
type AnimFrame = tilemap.AnimFrame

// ---------------------------------------------------------------------------
// Mesh (internal/mesh)
// ---------------------------------------------------------------------------

// Rope generates a ribbon/rope mesh that follows a polyline path.
type Rope = mesh.Rope

// RopeConfig configures a Rope mesh.
type RopeConfig = mesh.RopeConfig

// RopeJoinMode controls how segments join in a Rope mesh.
type RopeJoinMode = mesh.RopeJoinMode

// RopeCurveMode selects the curve algorithm used by Rope.Update().
type RopeCurveMode = mesh.RopeCurveMode

// DistortionGrid provides a grid mesh that can be deformed per-vertex.
type DistortionGrid = mesh.DistortionGrid

// ---------------------------------------------------------------------------
// Lighting (internal/lighting)
// ---------------------------------------------------------------------------

// Light represents a light source in a LightLayer.
type Light = lighting.Light

// LightLayer provides a convenient 2D lighting effect using erase blending.
type LightLayer = lighting.LightLayer

// ---------------------------------------------------------------------------
// Input shapes (internal/input)
// ---------------------------------------------------------------------------

// HitRect is an axis-aligned rectangular hit area in local coordinates.
type HitRect = input.HitRect

// HitCircle is a circular hit area in local coordinates.
type HitCircle = input.HitCircle

// HitPolygon is a convex polygon hit area in local coordinates.
type HitPolygon = input.HitPolygon

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// Blend modes.
const (
	BlendNormal   = types.BlendNormal
	BlendAdd      = types.BlendAdd
	BlendMultiply = types.BlendMultiply
	BlendScreen   = types.BlendScreen
	BlendErase    = types.BlendErase
	BlendMask     = types.BlendMask
	BlendBelow    = types.BlendBelow
	BlendNone     = types.BlendNone
)

// Node types.
const (
	NodeTypeContainer       = types.NodeTypeContainer
	NodeTypeSprite          = types.NodeTypeSprite
	NodeTypeMesh            = types.NodeTypeMesh
	NodeTypeParticleEmitter = types.NodeTypeParticleEmitter
	NodeTypeText            = types.NodeTypeText
)

// Event types.
const (
	EventPointerDown  = types.EventPointerDown
	EventPointerUp    = types.EventPointerUp
	EventPointerMove  = types.EventPointerMove
	EventClick        = types.EventClick
	EventDragStart    = types.EventDragStart
	EventDrag         = types.EventDrag
	EventDragEnd      = types.EventDragEnd
	EventPinch        = types.EventPinch
	EventPointerEnter = types.EventPointerEnter
	EventPointerLeave = types.EventPointerLeave
)

// Mouse buttons.
const (
	MouseButtonLeft   = types.MouseButtonLeft
	MouseButtonRight  = types.MouseButtonRight
	MouseButtonMiddle = types.MouseButtonMiddle
)

// Modifier keys.
const (
	ModShift = types.ModShift
	ModCtrl  = types.ModCtrl
	ModAlt   = types.ModAlt
	ModMeta  = types.ModMeta
)

// Text alignment.
const (
	TextAlignLeft   = types.TextAlignLeft
	TextAlignCenter = types.TextAlignCenter
	TextAlignRight  = types.TextAlignRight
)

// Cache tree modes.
const (
	CacheTreeManual = types.CacheTreeManual
	CacheTreeAuto   = types.CacheTreeAuto
)

// Batch modes.
const (
	BatchModeCoalesced = render.BatchModeCoalesced
	BatchModeImmediate = render.BatchModeImmediate
)

// Command types.
const (
	CommandSprite     = render.CommandSprite
	CommandMesh       = render.CommandMesh
	CommandParticle   = render.CommandParticle
	CommandTilemap    = render.CommandTilemap
	CommandSDF        = render.CommandSDF
	CommandBitmapText = render.CommandBitmapText
)

// Rope join modes.
const (
	RopeJoinMiter = mesh.RopeJoinMiter
	RopeJoinBevel = mesh.RopeJoinBevel
)

// Rope curve modes.
const (
	RopeCurveLine        = mesh.RopeCurveLine
	RopeCurveCatenary    = mesh.RopeCurveCatenary
	RopeCurveQuadBezier  = mesh.RopeCurveQuadBezier
	RopeCurveCubicBezier = mesh.RopeCurveCubicBezier
	RopeCurveWave        = mesh.RopeCurveWave
	RopeCurveCustom      = mesh.RopeCurveCustom
)

// ---------------------------------------------------------------------------
// Var re-exports
// ---------------------------------------------------------------------------

// Color constructors and constants.
var (
	RGB              = types.RGB
	ColorFromRGBA    = types.ColorFromRGBA
	ColorFromHSV     = types.ColorFromHSV
	ColorWhite       = types.ColorWhite
	ColorBlack       = types.ColorBlack
	ColorTransparent = types.ColorTransparent
)

// RGBA creates a Color from red, green, blue, alpha components in [0, 1].
func RGBA(r, g, b, a float64) Color { return types.RGBA(r, g, b, a) }

// Atlas constructors.
var (
	NewAtlas      = atlas.New
	NewBatchAtlas = atlas.NewBatch
	LoadAtlas     = atlas.LoadAtlas
)

// Text / Font constructors.
var (
	LoadDistanceFieldFont        = text.LoadDistanceFieldFont
	LoadDistanceFieldFontFromTTF = text.LoadDistanceFieldFontFromTTF
	GenerateSDFFromBitmaps       = text.GenerateSDFFromBitmaps
)

// Filter constructors.
var (
	NewColorMatrixFilter         = filter.NewColorMatrixFilter
	NewBlurFilter                = filter.NewBlurFilter
	NewOutlineFilter             = filter.NewOutlineFilter
	NewPixelPerfectOutlineFilter = filter.NewPixelPerfectOutlineFilter
	NewPixelPerfectInlineFilter  = filter.NewPixelPerfectInlineFilter
	NewPaletteFilter             = filter.NewPaletteFilter
	NewCustomShaderFilter        = filter.NewCustomShaderFilter
)

// Mesh constructors.
var (
	NewRope            = mesh.NewRope
	NewDistortionGrid  = mesh.NewDistortionGrid
	NewPolygon         = mesh.NewPolygon
	NewRegularPolygon  = mesh.NewRegularPolygon
	NewStar            = mesh.NewStar
	NewPolygonTextured = mesh.NewPolygonTextured
	SetPolygonPoints   = mesh.SetPolygonPoints
)

// Lighting.
var NewLightLayer = lighting.NewLightLayer

// Render.
var NewRenderTexture = render.NewRenderTexture

// Node index.
var NewNodeIndex = node.NewNodeIndex

// Tweens.
var (
	TweenPosition = core.TweenPosition
	TweenScale    = core.TweenScale
	TweenColor    = core.TweenColor
	TweenAlpha    = core.TweenAlpha
	TweenRotation = core.TweenRotation
)

// Test runner.
var LoadTestScript = core.LoadTestScript

// Render to texture.
var ToTexture = core.ToTexture

// ---------------------------------------------------------------------------
// Easing re-exports
// ---------------------------------------------------------------------------

var (
	EaseLinear       EaseFunc = ease.Linear
	EaseInQuad       EaseFunc = ease.InQuad
	EaseOutQuad      EaseFunc = ease.OutQuad
	EaseInOutQuad    EaseFunc = ease.InOutQuad
	EaseOutInQuad    EaseFunc = ease.OutInQuad
	EaseInCubic      EaseFunc = ease.InCubic
	EaseOutCubic     EaseFunc = ease.OutCubic
	EaseInOutCubic   EaseFunc = ease.InOutCubic
	EaseOutInCubic   EaseFunc = ease.OutInCubic
	EaseInQuart      EaseFunc = ease.InQuart
	EaseOutQuart     EaseFunc = ease.OutQuart
	EaseInOutQuart   EaseFunc = ease.InOutQuart
	EaseOutInQuart   EaseFunc = ease.OutInQuart
	EaseInQuint      EaseFunc = ease.InQuint
	EaseOutQuint     EaseFunc = ease.OutQuint
	EaseInOutQuint   EaseFunc = ease.InOutQuint
	EaseOutInQuint   EaseFunc = ease.OutInQuint
	EaseInSine       EaseFunc = ease.InSine
	EaseOutSine      EaseFunc = ease.OutSine
	EaseInOutSine    EaseFunc = ease.InOutSine
	EaseOutInSine    EaseFunc = ease.OutInSine
	EaseInExpo       EaseFunc = ease.InExpo
	EaseOutExpo      EaseFunc = ease.OutExpo
	EaseInOutExpo    EaseFunc = ease.InOutExpo
	EaseOutInExpo    EaseFunc = ease.OutInExpo
	EaseInCirc       EaseFunc = ease.InCirc
	EaseOutCirc      EaseFunc = ease.OutCirc
	EaseInOutCirc    EaseFunc = ease.InOutCirc
	EaseOutInCirc    EaseFunc = ease.OutInCirc
	EaseInElastic    EaseFunc = ease.InElastic
	EaseOutElastic   EaseFunc = ease.OutElastic
	EaseInOutElastic EaseFunc = ease.InOutElastic
	EaseOutInElastic EaseFunc = ease.OutInElastic
	EaseInBack       EaseFunc = ease.InBack
	EaseOutBack      EaseFunc = ease.OutBack
	EaseInOutBack    EaseFunc = ease.InOutBack
	EaseOutInBack    EaseFunc = ease.OutInBack
	EaseInBounce     EaseFunc = ease.InBounce
	EaseOutBounce    EaseFunc = ease.OutBounce
	EaseInOutBounce  EaseFunc = ease.InOutBounce
	EaseOutInBounce  EaseFunc = ease.OutInBounce
)

// ---------------------------------------------------------------------------
// Globals
// ---------------------------------------------------------------------------

// WhitePixel is a 1x1 white image used by default for solid color sprites.
var WhitePixel *ebiten.Image

// eventBgClick is the internal background click event type.
var eventBgClick = types.EventBgClick

// ---------------------------------------------------------------------------
// init — function pointer wiring
// ---------------------------------------------------------------------------

func init() {
	// WhitePixel
	WhitePixel = ebiten.NewImage(1, 1)
	WhitePixel.Fill(color.White)

	// Node → render cache wiring
	node.WhitePixelImage = WhitePixel
	node.InvalidateAncestorCacheFn = render.InvalidateAncestorCache
	node.RegisterAnimatedInCacheFn = render.RegisterAnimatedInCache
	node.SetCacheAsTreeFn = render.SetCacheAsTree
	node.InvalidateCacheTreeFn = render.InvalidateCacheTree
	node.IsCacheAsTreeEnabledFn = func(n *Node) bool { return n.CacheData != nil }

	// Node → core wiring
	node.PropagateSceneFn = core.PropagateScene

	// Render pipeline → atlas wiring
	render.AtlasPageFn = func(pageIdx int) *ebiten.Image {
		return atlas.GlobalManager().Page(pageIdx)
	}
	render.EnsureMagentaImageFn = atlas.EnsureMagentaImage
	render.ShouldCullFn = camera.ShouldCull
	render.BlendMaskFn = func() types.BlendMode { return types.BlendMask }

	// Render texture function pointers
	render.PageFn = func(pageIdx int) *ebiten.Image {
		return atlas.GlobalManager().Page(pageIdx)
	}
	render.MagentaImageFn = atlas.EnsureMagentaImage
	render.NewSpriteFn = NewSprite

	// Input wiring
	input.NodeDimensionsFn = camera.NodeDimensions
	input.RebuildSortedChildrenFn = render.RebuildSortedChildren

	// Mesh wiring
	mesh.NewMeshFn = func(name string, img *ebiten.Image, verts []ebiten.Vertex, inds []uint16) *node.Node {
		return NewMesh(name, img, verts, inds)
	}

	// Lighting wiring
	lighting.NewRenderTextureFn = func(w, h int) any {
		return render.NewRenderTexture(w, h)
	}
	lighting.RenderTextureImageFn = func(rt any) *ebiten.Image {
		return rt.(*render.RenderTexture).Image()
	}
	lighting.RenderTextureNewSpriteFn = func(rt any, name string) *Node {
		return rt.(*render.RenderTexture).NewSpriteNode(name)
	}
	lighting.RenderTextureDisposeFn = func(rt any) {
		rt.(*render.RenderTexture).Dispose()
	}
	lighting.EnsureMagentaImageFn = atlas.EnsureMagentaImage
	lighting.Clamp01Fn = render.Clamp01

	// Tilemap wiring
	tilemap.NewContainerFn = func(name string) *node.Node {
		return NewContainer(name)
	}
	tilemap.NewLayerEmitFn = func(layer *tilemap.Layer) {
		layer.EmitFn = func(l *tilemap.Layer, sAny any, treeOrder *int) {
			// Extract Pipeline — sAny is *core.Scene which has Pipeline field
			s, ok := sAny.(*core.Scene)
			if !ok {
				return
			}
			tilemap.EmitTilemapCommands(l, &s.Pipeline.Commands, s.Pipeline.ViewTransform, treeOrder)
		}
	}

	// Core → atlas page registration
	core.RegisterPageFn = func(index int, img *ebiten.Image) {
		atlas.GlobalManager().RegisterPage(index, img)
	}

	// Text → atlas page registration
	text.RegisterPageFn = func(idx int, img *ebiten.Image) {
		atlas.GlobalManager().RegisterPage(idx, img)
	}
	text.NextPageFn = func() int {
		return atlas.GlobalManager().NextPage()
	}
	text.AllocPageFn = func() int {
		return atlas.GlobalManager().AllocPage()
	}
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewScene creates a new scene with a pre-created root container.
func NewScene() *Scene {
	root := NewContainer("root")
	root.Interactable = true
	s := core.NewScene(root)
	root.Scene_ = s

	input.ScreenToWorldFn = func(sx, sy float64) (float64, float64) {
		if len(s.Cameras) > 0 {
			return s.Cameras[0].ScreenToWorld(sx, sy)
		}
		return sx, sy
	}

	return s
}

// NewContainer creates a container node with no visual representation.
func NewContainer(name string) *Node {
	return node.NewNode(name, NodeTypeContainer)
}

// NewSprite creates a sprite node that renders a texture region.
func NewSprite(name string, region TextureRegion) *Node {
	n := node.NewNode(name, NodeTypeSprite)
	n.TextureRegion_ = region
	if region == (TextureRegion{}) {
		n.CustomImage_ = WhitePixel
	}
	return n
}

// NewRect creates a solid-color rectangle node.
func NewRect(name string, w, h float64, c Color) *Node {
	n := NewSprite(name, TextureRegion{})
	n.SetSize(w, h)
	n.Color_ = c
	return n
}

// NewMesh creates a mesh node that uses DrawTriangles for rendering.
func NewMesh(name string, img *ebiten.Image, vertices []ebiten.Vertex, indices []uint16) *Node {
	n := node.NewNode(name, NodeTypeMesh)
	n.Mesh = &node.MeshData{
		Vertices:  vertices,
		Indices:   indices,
		Image:     img,
		AabbDirty: true,
	}
	return n
}

// NewParticleEmitter creates a particle emitter node with a preallocated pool.
func NewParticleEmitter(name string, cfg EmitterConfig) *Node {
	n := node.NewNode(name, NodeTypeParticleEmitter)
	n.TextureRegion_ = cfg.Region
	n.BlendMode_ = cfg.BlendMode
	n.Emitter = particle.NewEmitter(cfg)
	if cfg.Region == (TextureRegion{}) {
		n.CustomImage_ = WhitePixel
	}
	return n
}

// NewText creates a text node that renders the given string using font.
func NewText(name string, content string, font Font) *Node {
	n := node.NewNode(name, NodeTypeText)
	n.TextBlock = &TextBlock{
		Content:       content,
		Font:          font,
		FontSize:      16,
		Color:         RGBA(1, 1, 1, 1),
		LayoutDirty:   true,
		UniformsDirty: true,
	}
	return n
}

// NewCamera creates a standalone camera with the given viewport.
func NewCamera(viewport Rect) *Camera {
	return camera.NewCamera(viewport)
}

// NewTileMapViewport creates a new tilemap viewport node.
func NewTileMapViewport(name string, tileWidth, tileHeight int) *TileMapViewport {
	v := tilemap.NewViewport(name, tileWidth, tileHeight)
	v.CameraBoundsFn = func(cam any) tilemap.VisibleBoundsProvider {
		c, ok := cam.(*camera.Camera)
		if !ok || c == nil {
			return nil
		}
		return &tilemap.CameraBoundsAdapter{Cam: c}
	}
	return v
}

// NewPixelFont creates a pixel font from a spritesheet image.
func NewPixelFont(img *ebiten.Image, cellW, cellH int, chars string) *PixelFont {
	return text.NewPixelFont(img, cellW, cellH, chars)
}

// NewFontFromTTF generates an SDF font from TTF/OTF data, registers the atlas
// page, and returns the font ready to use.
func NewFontFromTTF(ttfData []byte, size float64) (*DistanceFieldFont, error) {
	return text.NewFontFromTTF(ttfData, size)
}

// LoadFontFromPathAsTtf reads a TTF/OTF file from disk.
func LoadFontFromPathAsTtf(path string) ([]byte, error) {
	return text.LoadFontFromPath(path)
}

// LoadFontFromSystemAsTtf searches OS system font directories for a font by name.
func LoadFontFromSystemAsTtf(name string) ([]byte, error) {
	return text.LoadFontFromSystem(name)
}

// ---------------------------------------------------------------------------
// Scene helpers
// ---------------------------------------------------------------------------

// Root returns the scene's root container node.
func Root(s *Scene) *Node {
	return s.Root
}

// Cameras returns the scene's camera list.
func Cameras(s *Scene) []*Camera {
	return s.Cameras
}

// LoadSceneAtlas parses TexturePacker JSON, registers atlas pages with the scene,
// and returns the Atlas for region lookups.
func LoadSceneAtlas(s *Scene, jsonData []byte, pages []*ebiten.Image) (*Atlas, error) {
	a, err := LoadAtlas(jsonData, pages)
	if err != nil {
		return nil, err
	}
	am := atlas.GlobalManager()
	startIndex := am.NextPage()
	for i, page := range pages {
		am.RegisterPage(startIndex+i, page)
	}
	if startIndex > 0 {
		for name, r := range a.Regions {
			r.Page += uint16(startIndex)
			a.Regions[name] = r
		}
	}
	return a, nil
}

// ---------------------------------------------------------------------------
// Run (ebiten.Game adapter)
// ---------------------------------------------------------------------------

// Run is a convenience entry point that creates an Ebitengine game loop around
// the given Scene.
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
	g := &gameShell{scene: scene, w: w, h: h}
	if cfg.ShowFPS {
		g.fpsWid = core.NewFPSWidget()
		g.fpsWid.X_, g.fpsWid.Y_ = 8, 8
	}
	return ebiten.RunGame(g)
}

type gameShell struct {
	scene  *Scene
	w, h   int
	fpsWid *Node
}

func (g *gameShell) Update() error {
	if g.scene.UpdateFunc != nil {
		if err := g.scene.UpdateFunc(); err != nil {
			return err
		}
	}
	g.scene.Update()
	if g.fpsWid != nil && g.fpsWid.OnUpdate != nil {
		g.fpsWid.OnUpdate(1.0 / float64(ebiten.TPS()))
	}
	return nil
}

func (g *gameShell) Draw(screen *ebiten.Image) {
	if g.scene.ClearColor.A() > 0 {
		screen.Fill(render.ColorToRGBA(g.scene.ClearColor))
	}
	g.scene.Draw(screen)
	if g.fpsWid != nil && g.fpsWid.CustomImage() != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(g.fpsWid.X_, g.fpsWid.Y_)
		screen.DrawImage(g.fpsWid.CustomImage(), &op)
	}
	if g.scene.PostDrawFunc != nil {
		g.scene.PostDrawFunc(screen)
	}
}

func (g *gameShell) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth != g.w || outsideHeight != g.h {
		g.w, g.h = outsideWidth, outsideHeight
		if g.scene.OnResize != nil {
			g.scene.OnResize(outsideWidth, outsideHeight)
		}
	}
	return g.w, g.h
}
