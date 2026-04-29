package willow

import (
	"fmt"
	"image/color"
	"math"
	"os"

	"github.com/devthicket/willow/internal/atlas"
	"github.com/devthicket/willow/internal/camera"
	"github.com/devthicket/willow/internal/core"
	"github.com/devthicket/willow/internal/filter"
	"github.com/devthicket/willow/internal/input"
	"github.com/devthicket/willow/internal/lighting"
	"github.com/devthicket/willow/internal/mesh"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/particle"
	"github.com/devthicket/willow/internal/render"
	"github.com/devthicket/willow/internal/text"
	"github.com/devthicket/willow/internal/tilemap"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
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

// AnimationSequence defines a single named animation.
type AnimationSequence = node.AnimationSequence

// AnimationPlayer manages multiple named animation sequences on a node.
type AnimationPlayer = node.AnimationPlayer

// ---------------------------------------------------------------------------
// Scene, Camera, Tweens (internal/core, camera)
// ---------------------------------------------------------------------------

// Scene is the top-level object that owns the node tree, cameras, input state,
// and render buffers.
type Scene = core.Scene

// Camera controls the view into the scene: position, zoom, rotation, viewport.
type Camera = camera.Camera

// CameraMode describes the camera's active movement state.
type CameraMode = camera.CameraMode

const (
	CameraModeIdle   = camera.CameraModeIdle
	CameraModeFollow = camera.CameraModeFollow
	CameraModeScroll = camera.CameraModeScroll
)

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

// GifConfig controls GIF recording behaviour.
type GifConfig = core.GifConfig

// SceneManager manages a stack of scenes with optional transitions.
type SceneManager = core.SceneManager

// Transition controls the visual effect between scene changes.
type Transition = core.Transition

// FadeTransition fades through a solid color between scene changes.
type FadeTransition = core.FadeTransition

// BatchMode controls how the render pipeline submits draw calls.
type BatchMode = render.BatchMode

// FXAAConfig holds tunable parameters for the FXAA post-process pass.
// Use DefaultFXAAConfig for sensible defaults.
type FXAAConfig = render.FXAAConfig

// DefaultFXAAConfig returns an FXAAConfig with FXAA 3.11 quality-15 defaults.
var DefaultFXAAConfig = render.DefaultFXAAConfig

// RunConfig holds optional configuration for [Run].
type RunConfig struct {
	Title         string
	Width, Height int
	Background    Color
	ShowFPS       bool
	// FXAA enables full-screen fast approximate anti-aliasing as a post-process
	// pass. Nil disables FXAA. Use DefaultFXAAConfig() for sensible defaults.
	FXAA *FXAAConfig

	// Resizable enables window resizing.
	Resizable bool
	// Decorated controls window chrome (title bar, borders). Nil keeps the
	// default (decorated). Set to a *bool to override.
	Decorated *bool
	// Fullscreen starts the window in fullscreen mode.
	Fullscreen bool
	// VSync controls vertical sync. Nil keeps the default (enabled).
	// Set to a *bool to override.
	VSync *bool
	// TPS overrides the target ticks per second. Zero keeps the default (60).
	TPS int
	// PreDrawFunc is called each frame after the screen is cleared but before
	// the scene is drawn. Use it to render custom content underneath the scene.
	PreDrawFunc func(screen *ebiten.Image)

	// AutoTestPath is an optional path to a JSON autotest script. If empty,
	// Run checks the WILLOW_AUTOTEST environment variable as a fallback.
	// When a script is loaded the test runner is registered on the scene and
	// the process exits automatically once all steps complete.
	AutoTestPath string
}

// ---------------------------------------------------------------------------
// Render types (internal/render)
// ---------------------------------------------------------------------------

// CommandType identifies the kind of render command.
type CommandType = render.CommandType

// RenderCommand is a single draw instruction emitted during scene traversal.
type RenderCommand = render.RenderCommand

// Emitter is the typed view onto the render pipeline passed to a Node's
// custom-emit handler. Use SetCustomEmit to install one.
type Emitter = render.Emitter

// Pipeline is the engine's render pipeline. Exposed for advanced
// integrations (custom emit hooks, low-level renderer extensions), but
// treat it as engine-internal: any field, method, or behavior may change
// between minor versions without notice. Prefer the Emitter facade for
// stable access; reach for Pipeline only when Emitter is missing what you
// need (and please file an issue describing the gap so it can become part
// of the stable surface).
type Pipeline = render.Pipeline

// TrianglesEmit describes a single batch of textured triangles for
// Emitter.AppendTriangles.
type TrianglesEmit = render.TrianglesEmit

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

// FontFamily holds SDF atlases for multiple styles and bake sizes, or wraps a pixel font.
// It is the only public font type in willow.
type FontFamily = text.FontFamily

// FontFamilyConfig configures a FontFamily created from TTF/OTF data.
type FontFamilyConfig = text.FontFamilyConfig

// FontStyle identifies a typographic style variant.
type FontStyle = text.FontStyle

// AtlasEntry holds the raw PNG and JSON bytes for one baked font atlas.
type AtlasEntry = text.AtlasEntry

// TextEffects configures text effects (outline, glow, shadow).
type TextEffects = text.TextEffects

// TextBlock holds text content, formatting, and cached layout state.
type TextBlock = text.TextBlock

// Glyph holds glyph metrics and atlas position.
type Glyph = text.Glyph

// SDFGenOptions configures SDF atlas generation (used by fontgen CLI).
type SDFGenOptions = text.SDFGenOptions

// GlyphBitmap holds a rasterized glyph and its metrics for atlas packing (used by fontgen CLI).
type GlyphBitmap = text.GlyphBitmap

// GenerateSDFFromBitmaps creates an SDF atlas from pre-rasterized glyph bitmaps (used by fontgen CLI).
var GenerateSDFFromBitmaps = text.GenerateSDFFromBitmaps

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

// DrawFilter is an optional interface for per-pixel filters that can be
// applied at draw time without an offscreen render target.
type DrawFilter = filter.DrawFilter

// ColorMatrixFilter applies a 4x5 color matrix transformation.
type ColorMatrixFilter = filter.ColorMatrixFilter

// BlurFilter applies an iterative multi-pass blur.
type BlurFilter = filter.BlurFilter

// OutlineFilter draws a multi-pixel outline around the source.
type OutlineFilter = filter.OutlineFilter

// PixelPerfectOutlineFilter draws a 1-pixel outline via Kage shader.
type PixelPerfectOutlineFilter = filter.PixelPerfectOutlineFilter

// PixelPerfectInlineFilter recolors edge pixels via Kage shader.
type PixelPerfectInlineFilter = filter.PixelPerfectInlineFilter

// PaletteFilter remaps pixel colors through a palette based on luminance.
type PaletteFilter = filter.PaletteFilter

// CustomShaderFilter wraps a user-provided Kage shader (offscreen RT path).
type CustomShaderFilter = filter.CustomShaderFilter

// CustomDrawShaderFilter wraps a user-provided per-pixel Kage shader
// that can be applied at draw time without an offscreen render target.
type CustomDrawShaderFilter = filter.CustomDrawShaderFilter

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

// TileQuery provides a read-only view of tilemap data for external systems.
type TileQuery = tilemap.TileQuery

// Tile GID flip flags (same convention as Tiled TMX format).
const (
	TileFlipH    = tilemap.TileFlipH    // Horizontal flip (bit 31)
	TileFlipV    = tilemap.TileFlipV    // Vertical flip (bit 30)
	TileFlipD    = tilemap.TileFlipD    // Diagonal flip (bit 29)
	TileFlagMask = tilemap.TileFlagMask // All three flags combined
)

// RegionsFromGrid builds a TextureRegion slice from a regular grid tileset.
// Index 0 is a zero region (empty tile); indices 1..count map to tiles in
// row-major order.  Margin is the outer border; spacing is the gap between tiles.
var RegionsFromGrid = tilemap.RegionsFromGrid

// EncodeGID combines a tile ID with flip flags into a single uint32 GID.
var EncodeGID = tilemap.EncodeGID

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

// Font style constants.
const (
	FontStyleRegular    = text.FontStyleRegular
	FontStyleBold       = text.FontStyleBold
	FontStyleItalic     = text.FontStyleItalic
	FontStyleBoldItalic = text.FontStyleBoldItalic
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
	NewCustomDrawShaderFilter    = filter.NewCustomDrawShaderFilter
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

// Animation player.
var NewAnimationPlayer = node.NewAnimationPlayer

// Spatial queries.
var (
	DistanceBetween  = node.DistanceBetween
	DirectionBetween = node.DirectionBetween
)

// Tweens.
var (
	TweenPosition = core.TweenPosition
	TweenScale    = core.TweenScale
	TweenColor    = core.TweenColor
	TweenAlpha    = core.TweenAlpha
	TweenRotation = core.TweenRotation
)

// Scene manager.
var (
	NewSceneManager   = core.NewSceneManager
	NewFadeTransition = core.NewFadeTransition
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

	// Tilemap wiring
	tilemap.NewContainerFn = func(name string) *node.Node {
		return NewContainer(name)
	}
	tilemap.NewLayerEmitFn = func(layer *tilemap.Layer) {
		layer.EmitFn = func(l *tilemap.Layer, eAny any, treeOrder *int) {
			e, ok := eAny.(*render.Emitter)
			if !ok {
				return
			}
			p := e.Pipeline()
			tilemap.EmitTilemapCommands(l, &p.Commands, p.ViewTransform, treeOrder)
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

	// Eagerly compile all built-in shaders so failures panic at startup.
	render.InitShaders()
	filter.InitShaders()
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

// NewTriangle creates a solid-color triangle node from three points.
func NewTriangle(name string, p1, p2, p3 Vec2, c Color) *Node {
	n := NewPolygon(name, []Vec2{p1, p2, p3})
	n.Color_ = c
	return n
}

// NewCircle creates a solid-color circle node with the given radius.
// The circle is approximated with 32 segments.
func NewCircle(name string, radius float64, c Color) *Node {
	const segments = 32
	pts := make([]Vec2, segments)
	for i := range pts {
		angle := float64(i) * 2 * math.Pi / segments
		pts[i] = Vec2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		}
	}
	n := NewPolygon(name, pts)
	n.Color_ = c
	return n
}

// NewLine creates a solid-color line node between two points with a given thickness.
// The line is built as a thin rotated rectangle sprite for efficient batching.
func NewLine(name string, x1, y1, x2, y2, thickness float64, c Color) *Node {
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)
	n := NewSprite(name, TextureRegion{})
	n.SetSize(length, thickness)
	n.SetPivot(0, thickness/2)
	n.SetPosition(x1, y1)
	n.SetRotation(angle)
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

// SetCustomEmit installs a typed custom-emit handler on n. The handler runs
// in place of the node's normal render emit; call e.EmitDefault inside the
// handler to opt back into the default rendering alongside any custom
// commands you append.
//
// Passing fn == nil clears any previously installed handler.
//
// The installed callback panics if invoked with anything other than an
// *Emitter — only the engine's render pipeline should call it.
func SetCustomEmit(n *Node, fn func(e *Emitter, treeOrder *int)) {
	if fn == nil {
		n.CustomEmit = nil
		return
	}
	n.CustomEmit = func(eAny any, treeOrder *int) {
		fn(eAny.(*Emitter), treeOrder)
	}
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
func NewText(name string, content string, font *FontFamily) *Node {
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

// NewFontFamilyFromTTF creates a FontFamily from TTF/OTF data.
// All style variants are provided in the config struct. BakeSizes defaults to [64, 256].
func NewFontFamilyFromTTF(cfg FontFamilyConfig) (*FontFamily, error) {
	return text.NewFontFamilyFromTTF(cfg)
}

// NewFontFamilyFromPixelFont wraps a pixel spritesheet into a FontFamily.
func NewFontFamilyFromPixelFont(img *ebiten.Image, cellW, cellH int, chars string) *FontFamily {
	return text.NewFontFamilyFromPixelFont(img, cellW, cellH, chars)
}

// NewFontFamilyFromFontBundle loads a pre-baked .fontbundle archive and returns a FontFamily.
func NewFontFamilyFromFontBundle(data []byte) (*FontFamily, error) {
	return text.NewFontFamilyFromFontBundle(data)
}

// ---------------------------------------------------------------------------
// Math helpers
// ---------------------------------------------------------------------------

// Deg converts degrees to radians.
func Deg(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// Rad converts radians to degrees.
func Rad(radians float64) float64 {
	return radians * 180 / math.Pi
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
	if cfg.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	if cfg.Decorated != nil {
		ebiten.SetWindowDecorated(*cfg.Decorated)
	}
	if cfg.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	if cfg.VSync != nil {
		ebiten.SetVsyncEnabled(*cfg.VSync)
	}
	if cfg.TPS > 0 {
		ebiten.SetTPS(cfg.TPS)
	}
	if cfg.Background.A() > 0 {
		scene.ClearColor = cfg.Background
	} else if scene.ClearColor.A() == 0 {
		scene.ClearColor = types.RGBA(0.18, 0.20, 0.25, 1)
	}
	g := &gameShell{scene: scene}
	g.w, g.h = w, h
	g.fxaa = cfg.FXAA
	g.preDraw = cfg.PreDrawFunc
	if cfg.FXAA != nil {
		render.EnsureFXAAShader() // compile eagerly so first frame has no stall
	}
	if cfg.ShowFPS {
		g.fpsWid = core.NewFPSWidget()
		g.fpsWid.X_, g.fpsWid.Y_ = 8, 8
	}

	// Autotest: load from RunConfig or WILLOW_AUTOTEST env var.
	atPath := cfg.AutoTestPath
	if atPath == "" {
		atPath = os.Getenv("WILLOW_AUTOTEST")
	}
	if atPath != "" {
		data, err := os.ReadFile(atPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autotest: read %s: %v\n", atPath, err)
			os.Exit(1)
		}
		runner, err := LoadTestScript(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autotest: load script: %v\n", err)
			os.Exit(1)
		}
		scene.SetTestRunner(runner)
		scene.ScreenshotDir = "screenshots"
		g.testRunner = runner
	}

	return ebiten.RunGame(g)
}

// gameBase holds shared fields and methods for both gameShell and managerShell.
type gameBase struct {
	w, h    int
	fpsWid  *Node
	fxaa    *FXAAConfig
	preDraw func(*ebiten.Image)
}

func (b *gameBase) tickFPS() {
	if b.fpsWid != nil && b.fpsWid.OnUpdate != nil {
		b.fpsWid.OnUpdate(1.0 / float64(ebiten.TPS()))
	}
}

func (b *gameBase) drawFPS(screen *ebiten.Image) {
	if b.fpsWid != nil && b.fpsWid.CustomImage() != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(b.fpsWid.X_, b.fpsWid.Y_)
		screen.DrawImage(b.fpsWid.CustomImage(), &op)
	}
}

func (b *gameBase) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	if b.fxaa != nil {
		render.DrawFinalScreenFXAA(screen, offscreen, geoM, *b.fxaa)
		return
	}
	ebiten.DefaultDrawFinalScreen(screen, offscreen, geoM)
}

type gameShell struct {
	gameBase
	scene      *Scene
	testRunner *TestRunner
}

func (g *gameShell) Update() error {
	if g.scene.UpdateFunc != nil {
		if err := g.scene.UpdateFunc(); err != nil {
			return err
		}
	}
	g.scene.Update()
	g.tickFPS()
	if g.testRunner != nil && g.testRunner.Done() {
		fmt.Println("autotest complete")
		return ebiten.Termination
	}
	return nil
}

func (g *gameShell) Draw(screen *ebiten.Image) {
	if g.scene.ClearColor.A() > 0 {
		screen.Fill(render.ColorToRGBA(g.scene.ClearColor))
	}
	if g.preDraw != nil {
		g.preDraw(screen)
	}
	g.scene.Draw(screen)
	g.drawFPS(screen)
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

// ---------------------------------------------------------------------------
// RunWithManager (SceneManager-based game loop)
// ---------------------------------------------------------------------------

// RunWithManager is a convenience entry point that runs a SceneManager.
func RunWithManager(sm *SceneManager, cfg RunConfig) error {
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
	if cfg.Resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	}
	if cfg.Decorated != nil {
		ebiten.SetWindowDecorated(*cfg.Decorated)
	}
	if cfg.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	if cfg.VSync != nil {
		ebiten.SetVsyncEnabled(*cfg.VSync)
	}
	if cfg.TPS > 0 {
		ebiten.SetTPS(cfg.TPS)
	}
	cur := sm.Current()
	if cfg.Background.A() > 0 {
		cur.ClearColor = cfg.Background
	} else if cur.ClearColor.A() == 0 {
		cur.ClearColor = types.RGBA(0.18, 0.20, 0.25, 1)
	}
	g := &managerShell{sm: sm}
	g.w, g.h = w, h
	g.fxaa = cfg.FXAA
	g.preDraw = cfg.PreDrawFunc
	if cfg.FXAA != nil {
		render.EnsureFXAAShader()
	}
	if cfg.ShowFPS {
		g.fpsWid = core.NewFPSWidget()
		g.fpsWid.X_, g.fpsWid.Y_ = 8, 8
	}
	return ebiten.RunGame(g)
}

type managerShell struct {
	gameBase
	sm *SceneManager
}

func (g *managerShell) Update() error {
	cur := g.sm.Current()
	if cur != nil && cur.UpdateFunc != nil {
		if err := cur.UpdateFunc(); err != nil {
			return err
		}
	}
	g.sm.Update()
	g.tickFPS()
	return nil
}

func (g *managerShell) Draw(screen *ebiten.Image) {
	cur := g.sm.Current()
	if cur != nil && cur.ClearColor.A() > 0 {
		screen.Fill(render.ColorToRGBA(cur.ClearColor))
	}
	if g.preDraw != nil {
		g.preDraw(screen)
	}
	g.sm.Draw(screen)
	g.drawFPS(screen)
	if cur != nil && cur.PostDrawFunc != nil {
		cur.PostDrawFunc(screen)
	}
}

func (g *managerShell) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth != g.w || outsideHeight != g.h {
		g.w, g.h = outsideWidth, outsideHeight
		cur := g.sm.Current()
		if cur != nil && cur.OnResize != nil {
			cur.OnResize(outsideWidth, outsideHeight)
		}
	}
	return g.w, g.h
}
