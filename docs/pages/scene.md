# Scene

The `Scene` is the top-level manager in Willow. It owns the root node, cameras, input state, and the render pipeline. Every Willow application creates at least one `Scene`.

## Creating a Scene

```go
scene := willow.NewScene()
```

This creates a scene with a root container node, default settings, and no cameras (a default full-screen camera is used automatically if none are added).

## Scene Fields

| Field | Type | Description |
|-------|------|-------------|
| `ClearColor` | `Color` | Background fill color when using `Run()` |
| `ScreenshotDir` | `string` | Output folder for screenshots (default: `"screenshots"`) |

## The Root Node

Every scene has a root container node:

```go
root := scene.Root()
root.AddChild(mySprite)
```

All visible content must be added to the root (or a descendant of the root) to be rendered.

## RunConfig

When using the `willow.Run()` helper, these fields configure the window:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Title` | `string` | `""` | Window title |
| `Width` | `int` | `640` | Window width |
| `Height` | `int` | `480` | Window height |
| `ShowFPS` | `bool` | `false` | Show FPS/TPS counter overlay |

When `ShowFPS` is true, an FPS widget is added at `RenderLayer` 255 (always on top).

## Batch Mode

| Mode | Description |
|------|-------------|
| `BatchModeCoalesced` | Default. Accumulates vertices and submits one `DrawTriangles32` per batch run. Best performance. |
| `BatchModeImmediate` | One `DrawImage` per sprite. Useful for debugging. |

```go
scene.SetBatchMode(willow.BatchModeImmediate)  // switch to per-sprite rendering
```

## Next Steps

- [Nodes](?page=nodes)  -  node types, visual properties, and tree manipulation
- [Camera & Viewport](?page=camera-and-viewport)  -  camera creation, follow, zoom, and culling
- [Sprites & Atlas](?page=sprites-and-atlas)  -  atlas loading with `scene.LoadAtlas()`

## Related

- [Getting Started](?page=getting-started)  -  game loop integration, `willow.Run()`, and `SetUpdateFunc`
- [Debug & Testing](?page=debug-and-testing)  -  debug overlays with `scene.SetDebugMode()`
- [ECS Integration](?page=ecs-integration)  -  connecting an entity store with `scene.SetEntityStore()`
- [Architecture](?page=architecture)  -  render pipeline and performance design
