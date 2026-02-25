# Nodes

The `Node` is the fundamental building block of Willow's scene graph. Every visible element  -  sprites, text, particles, meshes  -  is a `Node` added to the scene tree.

## Node Types

| Type | Constructor | Description |
|------|-------------|-------------|
| `NodeTypeContainer` | `NewContainer(name)` | Invisible grouping node  -  organizes children, applies transforms |
| `NodeTypeSprite` | `NewSprite(name, region)` | Renders a texture region (or solid color) |
| `NodeTypeMesh` | `NewMesh(name, img, verts, indices)` | Custom vertex geometry |
| `NodeTypeParticleEmitter` | `NewParticleEmitter(name, cfg)` | CPU-simulated particle system |
| `NodeTypeText` | `NewText(name, content, font)` | Text rendered with bitmap or TTF font |

## Creating Nodes

```go
// Container (invisible grouping)
group := willow.NewContainer("enemies")

// Sprite from atlas region
player := willow.NewSprite("player", atlas.Region("player_idle"))

// Text
label := willow.NewText("score", "Score: 0", myFont)
label.TextBlock.FontSize = 16

// Particle emitter
emitter := willow.NewParticleEmitter("sparks", emitterConfig)

// Mesh
mesh := willow.NewMesh("grid", img, vertices, indices)
```

## Visual Properties

| Field | Type | Description |
|-------|------|-------------|
| `Visible` | `bool` | Whether the node (and children) are rendered |
| `Renderable` | `bool` | Whether this specific node renders (children still can) |
| `Alpha` | `float64` | Opacity `[0,1]`, multiplied with parent's alpha |
| `Color` | `Color` | Multiplicative tint; `{1,1,1,1}` = no tint |
| `BlendMode` | `BlendMode` | Blend operation (see below) |
| `RenderLayer` | `uint8` | Primary sort key; lower values draw first |
| `GlobalOrder` | `int` | Secondary sort key within same RenderLayer |

## Blend Modes

| Mode | Description |
|------|-------------|
| `BlendNormal` | Standard alpha blending (source-over) |
| `BlendAdd` | Additive / lighter |
| `BlendMultiply` | Multiply (only darkens) |
| `BlendScreen` | Screen (only brightens) |
| `BlendErase` | Destination-out (punch transparent holes) |
| `BlendMask` | Clip destination to source alpha |
| `BlendBelow` | Destination-over (draw behind) |
| `BlendNone` | Opaque copy (skip blending) |

## Identity Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Human-readable label |
| `ID` | `uint32` | Auto-assigned unique identifier |
| `ZIndex` | `int` | Draw order among siblings; higher = on top |
| `EntityID` | `uint32` | ECS entity bridge |
| `UserData` | `any` | Arbitrary user data |

## Tree Manipulation

```go
// Add children
parent.AddChild(child)
parent.AddChildAt(child, 0)  // insert at index

// Remove children
parent.RemoveChild(child)
removed := parent.RemoveChildAt(2)
child.RemoveFromParent()
parent.RemoveChildren()  // remove all

// Query
children := parent.Children()   // read-only slice
count := parent.NumChildren()
first := parent.ChildAt(0)
```

## ZIndex and Draw Order

Siblings are drawn in order of their `ZIndex` (lower first). Nodes with equal `ZIndex` draw in the order they were added:

```go
background.SetZIndex(-1)  // draws behind
foreground.SetZIndex(10)  // draws on top
```

Use `SetChildIndex(child, index)` to reorder children directly.

## Update Callback

Attach per-node logic that runs each frame during `scene.Update()`:

```go
node.OnUpdate = func(dt float64) {
    node.X += speed * dt
}
```

## Interaction Callbacks

Register interaction callbacks with setter methods. Each setter automatically enables `Interactable` on the node:

```go
node.OnClick(func(ctx willow.ClickContext) {
    // handle click
})

node.OnDrag(func(ctx willow.DragContext) {
    node.X += ctx.DeltaX
    node.Y += ctx.DeltaY
    node.Invalidate()
})
```

Available setters: `OnClick`, `OnPointerDown`, `OnPointerUp`, `OnPointerMove`, `OnPointerEnter`, `OnPointerLeave`, `OnDragStart`, `OnDrag`, `OnDragEnd`, `OnPinch`. Pass `nil` to remove a callback.

See [Events & Callbacks](?page=events-and-callbacks) for the full API including scene-level handlers.

## Custom Images

For nodes that need a manually-managed image instead of an atlas region:

```go
node.SetCustomImage(myImage)
img := node.CustomImage()
```

## Disposal

```go
node.Dispose()           // removes from parent, releases resources
node.IsDisposed()        // check if disposed
```

Disposing a node also disposes all its children. Disposed nodes cannot be reused.

## Next Steps

- [Transforms](?page=transforms)  -  position, scale, rotation, pivot, and dirty flags
- [Solid-Color Sprites](?page=solid-color-sprites)  -  creating shapes without textures
- [Sprites](?page=sprites)  -  TextureRegion and image display
- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  making nodes interactive

## Related

- [Scene](?page=scene)  -  the scene that owns the root node
- [Particles](?page=particles)  -  CPU-simulated particle emitters
- [Mesh & Distortion](?page=meshes)  -  custom vertex geometry and distortion grids
- [Text & Fonts](?page=text-and-fonts)  -  bitmap and TTF text rendering
