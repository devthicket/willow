// Package willow is a retained-mode 2D game framework for [Ebitengine].
//
// Ebitengine is immediate-mode: every frame you issue draw commands from
// scratch, and nothing persists. Willow adds a retained-mode layer on top -
// you build a persistent scene graph of nodes, and Willow traverses it each
// frame to produce draw commands for Ebitengine.
//
// Willow provides the scene graph, transform hierarchy, sprite batching, input
// handling, camera viewports, text rendering, particle systems, and more that
// every non-trivial 2D game needs.
//
// Full documentation, tutorials, and examples are available at:
//
// https://phanxgames.github.io/willow/
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
//	func (g *Game) Update() error         { g.scene.Update(); return nil }
//	func (g *Game) Draw(s *ebiten.Image)  { g.scene.Draw(s) }
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
//	sprite.X, sprite.Y = 100, 50
//	container.AddChild(sprite)
//
// For solid-color rectangles, use [NewSprite] with a zero-value
// [TextureRegion] and set [Node.Color] and [Node.ScaleX]/[Node.ScaleY]:
//
//	box := willow.NewSprite("box", willow.TextureRegion{})
//	box.ScaleX, box.ScaleY = 80, 40
//	box.Color = willow.Color{R: 0.3, G: 0.7, B: 1, A: 1}
//
// # Key features
//
// Willow includes cameras with follow/scroll-to/zoom, bitmap and TTF text
// rendering (TTF via signed distance fields for resolution-independent
// scaling with GPU-accelerated effects), CPU-simulated particles,
// mesh/polygon/rope geometry, Kage shader filters, texture caching,
// masking, blend modes, lighting layers, tweens (via [gween]), and ECS
// integration (via [Donburi] adapter in willow/ecs).
//
// See the full docs for guides on each feature:
// https://phanxgames.github.io/willow/
//
// [Ebitengine]: https://ebitengine.org
// [gween]: https://github.com/tanema/gween
// [Donburi]: https://github.com/yohamta/donburi
package willow
