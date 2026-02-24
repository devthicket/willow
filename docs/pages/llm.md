# LLM Quick-Reference

Common pitfalls and patterns that trip up AI code generators. Read this before writing Willow code.

## Camera Setup

When creating a camera, you must position it at the center of the viewport — not at (0, 0). The camera's X/Y is where it *looks*, so centering it on screen means the world origin aligns with the top-left corner of the window.

```go
cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
cam.X = screenW / 2
cam.Y = screenH / 2
cam.Invalidate()
```

Without this, everything renders offset by half the screen.

### Camera Bounds

Use `SetBounds` + `ClampToBounds` to keep the camera within a world region (e.g. a tilemap):

```go
cam.SetBounds(willow.Rect{X: 0, Y: 0, Width: mapPixelW, Height: mapPixelH})
// after moving the camera each frame:
cam.ClampToBounds()
cam.Invalidate()
```

`ClearBounds()` disables clamping.

### Camera Follow Lerp

Lerp = `0` means **no movement** (not instant). Lerp = `1` means instant snap. This is the opposite of some frameworks:

```go
cam.Follow(playerNode, 0, 0, 0.1)  // smooth follow
cam.Follow(playerNode, 0, 0, 1.0)  // instant snap
```

## Interactable Must Be Enabled

Nodes have `Interactable = false` by default. If you set `OnClick`, `OnDrag`, or any pointer callback without enabling it, nothing will happen — the node is excluded from hit testing entirely.

```go
node.Interactable = true   // required for OnClick, OnDrag, etc.
node.OnClick = func(ctx willow.ClickContext) {
    // ...
}
```

## Hit Testing and Draggable Sprites

For WhitePixel sprites (created with `willow.TextureRegion{}`), **do not set HitShape** unless you have a specific reason. The default hit test derives an AABB from the node's dimensions automatically.

```go
// Correct — let the default AABB handle it:
sp := willow.NewSprite("handle", willow.TextureRegion{})
sp.ScaleX = 32
sp.ScaleY = 32
sp.PivotX = 0.5
sp.PivotY = 0.5
sp.Interactable = true
sp.OnDrag = func(ctx willow.DragContext) {
    sp.X += ctx.DeltaX
    sp.Y += ctx.DeltaY
    sp.Invalidate()
}
```

Setting `HitShape = willow.HitRect{X: -0.5, Y: -0.5, Width: 1, Height: 1}` will **break** hit testing because `WorldToLocal` returns coordinates in the node's un-pivoted local space (0 to 1 for a 1×1 white pixel), not centered coordinates.

If you do need an explicit HitShape on a WhitePixel sprite, use:
```go
sp.HitShape = willow.HitRect{Width: 1, Height: 1}  // origin at (0,0), not centered
```

There are two HitShape types:
```go
node.HitShape = willow.HitRect{Width: 64, Height: 64}       // rectangular
node.HitShape = willow.HitCircle{Radius: 32}                 // circular
```

## WhitePixel Sprites (Solid Color Shapes)

Create solid-color shapes with an empty TextureRegion. Size them with ScaleX/Y and tint with Color:

```go
sprite := willow.NewSprite("box", willow.TextureRegion{})
sprite.ScaleX = 40   // width in pixels
sprite.ScaleY = 40   // height in pixels
sprite.Color = willow.Color{R: 1, G: 0, B: 0, A: 1}  // red
```

Do **not** create unique `ebiten.Image` instances for solid-color rectangles.

## Polygons

Create arbitrary polygon shapes with a slice of Vec2 points:

```go
tri := willow.NewPolygon("triangle", []willow.Vec2{
    {X: 0, Y: -40}, {X: 35, Y: 30}, {X: -35, Y: 30},
})
tri.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}  // must set — Color{} is invisible
```

Update vertices at runtime with the free function:
```go
willow.SetPolygonPoints(poly, newPoints)  // NOT poly.SetPoints(newPoints)
```

## Containers

Group nodes under an invisible parent:

```go
group := willow.NewContainer("enemies")
group.AddChild(enemy1)
group.AddChild(enemy2)
scene.Root().AddChild(group)
```

`NewContainer` returns `*Node` — same type as sprites and polygons. Containers have no visual of their own.

Use `container.RemoveChildren()` to detach all children. Use `child.RemoveFromParent()` to detach one child without destroying it.

## Invalidate After Direct Field Mutation

Setting `node.X`, `node.Y`, or any other field directly does **not** invalidate the transform — the node will not move on screen until you call `Invalidate()`. Either call it yourself or use the setter methods which handle it for you:

```go
// Option A: direct fields + manual Invalidate (preferred in hot loops)
node.X += dx
node.Y += dy
node.Rotation += 0.01
node.Invalidate()  // required — without this, nothing visually changes

// Option B: setter methods (call Invalidate internally)
node.SetPosition(node.X + dx, node.Y + dy)
node.SetRotation(node.Rotation + 0.01)
```

Available setter methods: `SetPosition(x, y)`, `SetScale(sx, sy)`, `SetRotation(r)`, `SetPivot(px, py)`, `SetAlpha(a)`, `SetZIndex(z)`.

This is the most common mistake: assigning `node.X = 100` and wondering why the node didn't move.

## Alpha vs Color.A

`Alpha` and `Color.A` are separate fields that multiply together. Use `Alpha` for fading nodes in/out. Use `Color` for tinting.

```go
node.Alpha = 0.5                                  // 50% transparent
node.SetAlpha(0.5)                                 // same, with auto-invalidate
node.Color = willow.Color{R: 1, G: 0, B: 0, A: 1} // red tint at full color alpha
```

`nodeDefaults` initializes both `Alpha = 1` and `Color = {1,1,1,1}`, so new nodes are fully visible.

## Color Zero Value Makes Nodes Invisible

`Color` is multiplicative. The zero value `Color{}` means `{R:0, G:0, B:0, A:0}` — fully transparent black. `NewSprite` initializes Color to `{1,1,1,1}` automatically, but if you manually set `Color` to a zero struct, the node vanishes:

```go
// WRONG — sprite will be invisible:
sprite.Color = willow.Color{}  // alpha 0 → invisible

// CORRECT — no tint:
sprite.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
```

## Rotation Is Radians, Not Degrees

```go
// WRONG:
node.Rotation = 90  // this is ~14 full rotations, not 90°

// CORRECT:
node.Rotation = math.Pi / 2  // 90° clockwise
```

## Pivot Is Local Pixels for Atlas Sprites

For atlas sprites, PivotX/Y are in local pixel coordinates, not normalized 0–1. A 64×64 sprite centered needs `PivotX = 32`, not `PivotX = 0.5`:

```go
// WRONG for a 64×64 atlas sprite:
sprite.PivotX = 0.5  // 0.5 pixels, not center

// CORRECT:
sprite.PivotX = 32  // half of 64px width
sprite.PivotY = 32
```

WhitePixel sprites are 1×1, so `PivotX = 0.5` does center them — but only by coincidence.

## Visible vs Renderable

`Visible = false` skips the node **and all its children**. If you want to hide only the parent's visual while keeping children visible, use `Renderable = false`:

```go
parent.Visible = false     // hides parent AND all children
parent.Renderable = false  // hides only this node's visual; children still render
```

## Scene Graph Z-Order

Nodes render in tree order (depth-first). Later children render on top of earlier ones. Control z-order by adding nodes in the right sequence or setting `ZIndex`:

```go
// Ropes first (behind), then handles (on top)
scene.Root().AddChild(ropeNode)
scene.Root().AddChild(handleNode)

// Or use ZIndex for explicit ordering within a parent:
ropeNode.ZIndex = 0
handleNode.ZIndex = 10
```

You can use `node.ZIndex = n` (direct field, requires `Invalidate`) or `node.SetZIndex(n)` (auto-invalidates).

## Input Context Structs

Callbacks receive context structs with useful fields:

```go
node.OnClick = func(ctx willow.ClickContext) {
    ctx.Node      // *Node — the clicked node
    ctx.GlobalX   // world-space X
    ctx.GlobalY   // world-space Y
    ctx.LocalX    // node-local X
    ctx.LocalY    // node-local Y
    ctx.Button    // MouseButton
    ctx.Modifiers // KeyModifiers held during click
}

node.OnDrag = func(ctx willow.DragContext) {
    ctx.Node         // *Node — the dragged node
    ctx.DeltaX       // world-space movement since last drag event
    ctx.DeltaY       // world-space movement since last drag event
    ctx.ScreenDeltaX // screen-pixel movement (for camera panning)
    ctx.ScreenDeltaY // screen-pixel movement (for camera panning)
    ctx.StartX       // world X where drag began
    ctx.StartY       // world Y where drag began
    ctx.GlobalX      // current world X
    ctx.GlobalY      // current world Y
}
```

Use `DeltaX/DeltaY` when dragging nodes in world space. Use `ScreenDeltaX/ScreenDeltaY` when panning a camera (divide by `cam.Zoom`):

```go
// Camera panning:
cam.X -= ctx.ScreenDeltaX / cam.Zoom
cam.Y -= ctx.ScreenDeltaY / cam.Zoom
cam.ClampToBounds()
cam.Invalidate()
```

## OnDragEnd and OnUpdate Callbacks

`OnDragEnd` fires when the user releases a drag:

```go
node.OnDragEnd = func(ctx willow.DragContext) {
    // snap to grid, validate position, etc.
}
```

`OnUpdate` runs per-frame logic on individual nodes instead of centralizing in `SetUpdateFunc`:

```go
node.OnUpdate = func(dt float64) {  // float64, not float32
    node.Rotation += 0.02
}
```

Other callbacks: `OnPinch func(PinchContext)`, `OnPointerEnter func(PointerContext)`, `OnPointerLeave func(PointerContext)`.

## Drag Dead Zone

By default, drags require a small movement before firing. Set to 0 for immediate response (useful for camera panning):

```go
scene.SetDragDeadZone(0)
```

## Constructors That Return Two Values

`NewRope` and `NewDistortionGrid` return `(controller, node)` — not just a node. You need both:

```go
rope, ropeNode := willow.NewRope("cable", img, nil, config)
scene.Root().AddChild(ropeNode)  // add the NODE to scene
rope.Update()                     // call methods on the ROPE controller

grid, gridNode := willow.NewDistortionGrid("water", img, 10, 8)
scene.Root().AddChild(gridNode)
grid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) { ... })
```

## Wrapper Types Need .Node() to Add to Scene

`LightLayer` and `TileMapViewport` are not `*Node`. Call `.Node()` to get the renderable node:

```go
ll := willow.NewLightLayer(800, 600, 0.8)
scene.Root().AddChild(ll.Node())  // NOT ll

viewport := willow.NewTileMapViewport("map", 32, 32)
scene.Root().AddChild(viewport.Node())  // NOT viewport
```

## Rope Endpoint Binding (Pointer Stability)

Ropes read their Start/End/Controls positions through pointers. You must ensure the pointers remain valid and that you mutate the **same** Vec2 the Rope references.

**Common mistake:** creating a local Vec2, passing its address to `NewRope`, then copying it into a struct. The Rope still points at the original local — your struct copy is a different variable.

```go
// WRONG — pointer aliases a loop-local that won't be updated:
for i := range n {
    startPos := willow.Vec2{X: 100, Y: 200}
    r, rn := willow.NewRope("rope", img, nil, willow.RopeConfig{
        Start: &startPos,  // points at loop-local
    })
    ropes[i] = myRope{start: startPos, r: r}  // copies value, Rope still reads loop-local
}
ropes[0].start.X = 999  // Rope never sees this!

// CORRECT — allocate the struct first, then point at its fields:
rb := &ropeBinding{
    start: willow.Vec2{X: 100, Y: 200},
    end:   willow.Vec2{X: 400, Y: 200},
}
r, rn := willow.NewRope("rope", img, nil, willow.RopeConfig{
    Start: &rb.start,  // points directly at the struct field
    End:   &rb.end,
})
rb.r = r

// Each frame — mutate the same Vec2s the Rope reads:
rb.start.X = newX
rb.r.Update()
```

### Rope Curve Modes

```go
willow.RopeConfig{
    CurveMode: willow.RopeCurveQuadBezier,  // smooth bezier (needs Controls)
    JoinMode:  willow.RopeJoinBevel,
}
willow.RopeConfig{
    CurveMode: willow.RopeCurveCatenary,    // hanging cable
    Sag:       25,                           // droop amount (catenary only)
    JoinMode:  willow.RopeJoinBevel,
}
```

## Loading Atlas Sprites

Use `scene.LoadAtlas` to load a TexturePacker JSON atlas. It auto-registers pages:

```go
atlas, err := scene.LoadAtlas(jsonData, []*ebiten.Image{atlasPage})
sprite := willow.NewSprite("hero", atlas.Region("hero"))
```

Missing region names return a magenta placeholder. Enable `scene.SetDebugMode(true)` to log warnings for missing regions.

### Loading Standalone Images

Use `SetCustomImage` for one-off images not in an atlas, or register them as atlas pages:

```go
// Option A: RegisterPage + TextureRegion
scene.RegisterPage(0, img)
region := willow.TextureRegion{Page: 0, Width: 128, Height: 128, OriginalW: 128, OriginalH: 128}
sprite := willow.NewSprite("name", region)

// Option B: SetCustomImage (simpler for previews/one-offs)
sprite := willow.NewSprite("preview", willow.TextureRegion{})
sprite.SetCustomImage(img)
```

## BlendMode

Set `node.BlendMode` for compositing effects:

```go
willow.BlendNormal    // standard alpha blending (default)
willow.BlendAdd       // additive — overlapping particles brighten
willow.BlendMultiply  // multiply — only darkens (used by LightLayer)
willow.BlendScreen    // screen — only brightens
willow.BlendErase     // destination-out — punches transparent holes
willow.BlendMask      // clips destination to source alpha
```

## LightLayer

### Setup

```go
lightLayer := willow.NewLightLayer(screenW, screenH, ambientAlpha)
scene.Root().AddChild(lightLayer.Node())
```

Ambient alpha is inverted: `0.0 = fully lit` (additive-only), `1.0 = fully dark`. Texture lights need `lightLayer.SetPages(atlas.Pages)`.

### Requires Redraw() Every Frame

The light layer does not auto-update:

```go
scene.SetUpdateFunc(func() error {
    lightLayer.Redraw()
    return nil
})
```

### Light.Target for Auto-Following

A light can follow a node automatically:

```go
light := &willow.Light{
    Target: playerNode,  // syncs world position from node on Redraw()
    Radius: 200,
    Color:  willow.Color{R: 1, G: 0.9, B: 0.7, A: 1},
}
lightLayer.AddLight(light)
```

## Mask Nodes Are NOT Part of the Scene Tree

Do not `AddChild` a mask node. Masks live outside the scene tree:

```go
mask := willow.NewSprite("circle-mask", atlas.Region("circle"))
// Do NOT call scene.Root().AddChild(mask)
content.SetMask(mask)
```

**The mask root node's own transform is ignored by the renderer** — only the transforms of its *children* are applied. Animated masks must use a container as root and put the actual shape one level below:

```go
maskRoot := willow.NewContainer("mask-root")
shape := willow.NewPolygon("star", starPoints)
shape.X = 150  // transforms on children ARE applied
maskRoot.AddChild(shape)
content.SetMask(maskRoot)
```

## Tweens Have No Auto-Tick

Willow has no global tween manager. You must call `Update(dt)` yourself every frame, and `dt` is `float32`:

```go
dt := float32(1.0 / float64(ebiten.TPS()))  // use TPS(), not hardcoded 60

scene.SetUpdateFunc(func() error {
    if tween != nil && !tween.Done {
        tween.Update(dt)
    }
    return nil
})
```

## SetUpdateFunc Replaces — It Does Not Chain

Calling `scene.SetUpdateFunc()` a second time replaces the previous callback. Put all per-frame logic in one function:

```go
// WRONG — only the last one runs:
scene.SetUpdateFunc(func() error { lightLayer.Redraw(); return nil })
scene.SetUpdateFunc(func() error { tween.Update(dt); return nil })

// CORRECT — one function with everything:
scene.SetUpdateFunc(func() error {
    lightLayer.Redraw()
    if !tween.Done { tween.Update(dt) }
    return nil
})
```

`SetPostDrawFunc(func(screen *ebiten.Image))` also exists for overlaying debug info after the scene renders.

## Particle Emitters

### Control Is on .Emitter, Not the Node

`Start()`, `Stop()`, `Reset()` live on the `.Emitter` sub-object:

```go
emitter.Emitter.Start()
emitter.Emitter.Stop()
count := emitter.Emitter.AliveCount()
```

### Burst Emitters: Reset() Before Start()

When re-triggering a burst emitter (e.g. on click), always call `Reset()` first:

```go
burst.Emitter.Reset()
burst.Emitter.Start()
```

Without `Reset()`, the emitter may have stale particle state from the previous burst.

### WorldSpace for Positionally-Stable Bursts

When particles should stay at their spawn position even if the emitter node moves:

```go
willow.EmitterConfig{
    WorldSpace: true,
    BlendMode:  willow.BlendAdd,
    // ...
}
```

## Text

All fonts are `*SpriteFont`. The primary constructor is `NewFontFromTTF`:

```go
font, err := willow.NewFontFromTTF(ttfData, 80)  // returns (*SpriteFont, error)
label := willow.NewText("label", "Hello", font)   // returns *Node
scene.Root().AddChild(label)
```

`NewFontFromTTF` generates the SDF atlas and registers the page automatically — no `RegisterPage` call needed. Size is the rasterization size (higher = sharper when scaled up). Use 0 for the default (80px).

Set `SDFEffects` on the `TextBlock` (not the node) for outline, glow, and shadow. `nil` SDFEffects means plain fill only:

```go
label.TextBlock.SDFEffects = &willow.SDFEffects{
    OutlineWidth: 2.0,
    OutlineColor: willow.Color{R: 0, G: 0, B: 0, A: 1},
}
label.TextBlock.Invalidate()
```

### TextBlock Properties

```go
label.TextBlock.Content = "New text"
label.TextBlock.Align = willow.TextAlignCenter
label.TextBlock.WrapWidth = 400
label.TextBlock.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}
label.TextBlock.Outline = &willow.Outline{Color: willow.Color{A: 1}, Thickness: 2}
label.TextBlock.Invalidate()  // NOT label.Invalidate()
```

Changing `TextBlock` properties requires `node.TextBlock.Invalidate()` — not `node.Invalidate()`.

### Font Measurement

```go
w, h := font.MeasureString("Hello World")  // returns (float64, float64)
```

## Filters Is a Slice

The field is `Filters` (plural), type `[]willow.Filter`:

```go
sprite.Filters = []willow.Filter{willow.NewBlurFilter(4)}
```

## CacheAsTree for Static Geometry

Cache a subtree's render commands to avoid re-traversal. Ideal for large tilemaps:

```go
container.SetCacheAsTree(true, willow.CacheTreeManual)
// CacheTreeManual — you call container.InvalidateCacheTree() when tiles change
// CacheTreeAuto  — auto-invalidates when setters on descendants are called (default)
```

## TileMapViewport

Full setup pattern:

```go
viewport := willow.NewTileMapViewport("world", tileW, tileH)
viewport.SetCamera(cam)
scene.Root().AddChild(viewport.Node())

layer := viewport.AddTileLayer("ground", mapW, mapH, gidData, regions, tilesetImg)
layer.Node().RenderLayer = 0  // controls draw order within the viewport
```

GID data is `[]uint32`. GID 0 is reserved as empty (not rendered). Valid tile IDs start at 1:

```go
regions := []willow.TextureRegion{
    {},                       // GID 0 — empty, unused
    atlas.Region("grass"),    // GID 1
    atlas.Region("dirt"),     // GID 2
}
```

Sandwich entity layers between tile layers using `viewport.AddChild(entityLayer)` with matching `RenderLayer` values.

## Dispose Is Permanent and Recursive

`Dispose()` permanently destroys a node **and all its children**. Disposed nodes cannot be reused. If you want to temporarily remove a node, use `RemoveFromParent()` instead.

## Node Name Is a Debug Label

The `name` parameter in `NewSprite("box", ...)`, `NewContainer("group")`, `NewPolygon("tri", ...)` is just a debug label. It does not need to be unique.

## Build and Test

```bash
go build ./...
go vet ./...
go test ./...
cd ecs && go test ./...
```
