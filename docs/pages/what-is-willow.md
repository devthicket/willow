# What is Willow?

<p align="center">
  <img src="gif/shapes.gif" alt="Shapes demo" width="300">
  <img src="gif/masks.gif" alt="Masks demo" width="300">
  <img src="gif/shaders.gif" alt="Shaders demo" width="300">
  <img src="gif/watermesh.gif" alt="Watermesh demo" width="300">
</p>

**[GitHub](https://github.com/phanxgames/willow)** | **[API Reference](https://pkg.go.dev/github.com/phanxgames/willow)**

Willow is a **retained-mode-inspired** 2D game framework built on [Ebitengine](https://ebitengine.org/) for Go. You build a tree of nodes  -  sprites, text, particles, meshes  -  and Willow traverses that tree each frame to produce optimized render commands for Ebitengine. You describe *what* to render by building a scene tree, not *how* to render by issuing draw commands yourself.

Ebitengine itself is **immediate-mode**  -  every frame you must issue every draw command from scratch, and nothing persists between frames. Willow adds a retained-mode-inspired layer on top: your scene graph is the persistent state. Create a node once, set its properties, and it keeps rendering until you remove it. Under the hood Willow still walks the tree and re-submits draw commands to Ebitengine each frame, but that machinery is hidden from you. This is the same pattern used by engines like Unity, Godot, and PixiJS  -  a retained scene graph driving an immediate-mode renderer.

```
Your Game             - gameplay, content, logic
Willow                - scene graph, rendering, interaction
Ebitengine            - GPU backend, window, audio, platform
```

## The Display Tree

The central concept in Willow is the **display tree**  -  a tree data structure where every visible thing in your game is a node. Nodes can contain children, forming a hierarchy that mirrors how you think about your game:

```
Scene
 └─ Root
     ├─ Background
     ├─ Player
     │   ├─ Sword
     │   └─ HealthBar
     ├─ Enemy
     │   └─ Shadow
     └─ UI
         └─ ScoreText
```

The tree isn't just for organization  -  **parent-child relationships do real work**. When you move a parent, all its children move with it. Hide a parent, and its children disappear. A parent's opacity multiplies into its children. You group things together and the relationships handle the rest. The Player node above carries its Sword and HealthBar along automatically  -  no manual bookkeeping.

This is the core difference between a retained-mode engine and a raw immediate-mode API. Without a display tree, you manually position and draw every sprite every frame. With one, you set properties on a node once and the tree remembers. Move a character? Set `node.X`. The tree persists between frames, so Willow knows what changed and what didn't.

Each frame, Willow walks this tree top-to-bottom, computes final transforms, culls anything off-screen, and submits the result to Ebitengine. You describe the scene; Willow figures out the rendering.

## Key Features

- **Scene graph**  -  tree of nodes with parent-child transforms, visibility, and render ordering
- **Sprites & atlas**  -  TexturePacker JSON loading, automatic batching
- **Camera & viewport**  -  follow, scroll-to, zoom, culling, multi-camera
- **Text**  -  TTF/OTF via signed distance field (SDF) rendering for resolution-independent scaling with outlines, glows, and shadows
- **Input**  -  hit testing, click/drag/pinch gestures with typed callbacks
- **Particles**  -  CPU-simulated emitters with pooled allocation
- **Tweens & animation**  -  frame sequences, easing via gween
- **Filters**  -  Kage shader post-processing (blur, glow, color grading)
- **Meshes**  -  custom vertex geometry, ropes, polygons
- **Tilemaps**  -  viewport-based tile rendering
- **Lighting**  -  2D light layer compositing
- **Caching**  -  `CacheAsTree` and `CacheAsTexture` for static content
- **ECS integration**  -  optional `EntityStore` interface bridges interaction events into your ECS

## How It Works

You own the game loop. Create a `Scene`, call `scene.Update()` and `scene.Draw(screen)` from your Ebitengine game, and Willow handles everything else:

1. **`scene.Update()`**  -  processes input, fires callbacks, updates particles/tweens, recomputes dirty transforms
2. **`scene.Draw(screen)`**  -  traverses the tree, culls off-screen nodes, emits and sorts render commands, submits to Ebitengine

The `willow.Run()` helper wraps this into a one-liner for simple projects, but it's a convenience  -  not a requirement.

## Performance Targets

Willow targets **10,000+ sprites at 120+ FPS** on desktop and **60+ FPS** on mobile with zero heap allocations per frame on the hot path.

## Etymology

Willow's scene graph was inspired by [PixiJS](https://pixijs.com/). Pixies are associated with wisps  -  glowing sprites of light. A display tree full of wisps? That's a willow tree.

## Next Steps

- [Getting Started](?page=getting-started)  -  installation and your first Willow program
- [Architecture](?page=architecture)  -  render pipeline, scene tree structure, and performance design
- [Performance](?page=performance-overview)  -  benchmarks, batching, and optimization strategies
- [Examples](?page=examples)  -  runnable demos with GIFs and source code
