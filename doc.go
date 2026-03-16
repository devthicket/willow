// Package willow is a display-tree 2D rendering layer for [Ebitengine],
// including scene management, batching, cameras, culling, hit detection,
// and special effects. Inspired by [Starling], Flash display lists, and
// [PixiJS], adapted for Go's strengths.
//
// Ebitengine is immediate-mode: every frame you issue draw commands from
// scratch, and nothing persists. Willow adds a retained-mode-inspired layer
// on top - you create a tree of nodes representing your game objects, and
// Willow traverses that tree each frame to produce draw commands for
// Ebitengine. You describe what exists in your scene, not how to render it
// each frame. This is the same pattern used by engines like Unity, Godot,
// and PixiJS: a persistent scene graph (display tree) driving an
// immediate-mode renderer.
//
// It sits between Ebitengine and your game:
//
//	Your Game             - gameplay, content, logic
//	Willow                - scene graph, rendering, interaction
//	Ebitengine            - GPU backend, window, audio, platform
//
// Full documentation, tutorials, and examples are available at:
//
// https://www.devthicket.org/willow
//
// # Quick start
//
// The simplest way to get started is [Run], which creates a window and game
// loop for you:
//
//	scene := willow.NewScene()
//	// ... add nodes ...
//	willow.Run(scene, willow.RunConfig{
//		Title: "My Game", Width: 640, Height: 480,
//	})
//
// For full control, implement [ebiten.Game] yourself and call
// [Scene.Update] and [Scene.Draw] directly:
//
//	type Game struct{ scene *willow.Scene }
//
//	func (g *Game) Update() error         { g.Scene_.Update(); return nil }
//	func (g *Game) Draw(s *ebiten.Image)  { g.Scene_.Draw(s) }
//	func (g *Game) Layout(w, h int) (int, int) { return w, h }
//
// # Scene graph
//
// Every visual element is a [Node]. Nodes form a tree rooted at
// [Scene.Root]. Children inherit their parent's transform and alpha.
//
// Create nodes with typed constructors: [NewContainer], [NewSprite],
// [NewText], [NewParticleEmitter], [NewMesh], [NewPolygon], and others.
//
//	container := willow.NewContainer("ui")
//	scene.Root().AddChild(container)
//
//	sprite := willow.NewSprite("hero", atlas.Region("hero_idle"))
//	sprite.SetPosition(100, 50)
//	container.AddChild(sprite)
//
// For solid-color rectangles, use [NewSprite] with a zero-value
// [TextureRegion] and set color and size:
//
//	box := willow.NewSprite("box", willow.TextureRegion{})
//	box.SetSize(80, 40)
//	box.SetColor(willow.RGBA(0.3, 0.7, 1, 1))
//
// # Finding nodes
//
// For quick tree searches, use [Node.FindChild] (direct children) or
// [Node.FindDescendant] (recursive depth-first). Both support % wildcards:
//
//	bar := enemy.FindChild("health_bar")
//	boss := scene.Root().FindDescendant("boss%")  // starts with "boss"
//
// For repeated lookups or tag-based grouping, use [NodeIndex]:
//
//	idx := willow.NewNodeIndex()
//	idx.Add(enemy, "enemy", "damageable")
//	enemies := idx.FindByTag("enemy")
//	boss := idx.FindByName("boss%")
//
// # Key features
//
// Willow includes cameras with follow/scroll-to/zoom, two text systems
// (SDF-based [DistanceFieldFont] for smooth TTF/OTF scaling with outlines, glows,
// and shadows; pixel-perfect [PixelFont] for bitmap spritesheet fonts with
// integer-only scaling), CPU-simulated particles, mesh/polygon/rope geometry,
// Kage shader filters, texture caching, masking, blend modes, lighting layers,
// and tweens (via [gween]).
//
// All 45 gween easing functions are re-exported as [EaseLinear],
// [EaseOutCubic], [EaseOutBounce], etc. for autocomplete discoverability
// without an extra import.
//
// See the full docs for guides on each feature:
// https://www.devthicket.org/willow
//
// [Ebitengine]: https://ebitengine.org
// [Starling]: https://gamua.com/starling/
// [PixiJS]: https://pixijs.com/
// [gween]: https://github.com/tanema/gween
package willow
