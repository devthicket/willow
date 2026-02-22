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

## WhitePixel Sprites (Solid Color Shapes)

Create solid-color shapes with an empty TextureRegion. Size them with ScaleX/Y and tint with Color:

```go
sprite := willow.NewSprite("box", willow.TextureRegion{})
sprite.ScaleX = 40   // width in pixels
sprite.ScaleY = 40   // height in pixels
sprite.Color = willow.Color{R: 1, G: 0, B: 0, A: 1}  // red
```

Do **not** create unique `ebiten.Image` instances for solid-color rectangles.

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

This is the most common mistake: assigning `node.X = 100` and wondering why the node didn't move.

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

## Scene Graph Z-Order

Nodes render in tree order (depth-first). Later children render on top of earlier ones. There is no `MoveToFront` method — control z-order by adding nodes in the right sequence:

```go
// Ropes first (behind), then handles (on top)
scene.Root().AddChild(ropeNode)
scene.Root().AddChild(handleNode)
```

Or use `SetZIndex(z)` for explicit ordering within a parent.

## Loading Standalone Images as Sprites

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

## Color Zero Value Makes Nodes Invisible

`Color` is multiplicative. The zero value `Color{}` means `{R:0, G:0, B:0, A:0}` — fully transparent black. Atlas sprites (non-WhitePixel) need `{1,1,1,1}` for no tint:

```go
// WRONG — sprite will be invisible:
sprite := willow.NewSprite("hero", atlas.Region("hero"))
// sprite.Color is Color{} by default → alpha 0 → invisible

// CORRECT:
sprite.Color = willow.Color{R: 1, G: 1, B: 1, A: 1}  // no tint
```

Note: `NewSprite` likely initializes Color to `{1,1,1,1}` — but if you manually set `Color` to a zero struct, the node vanishes.

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

## LightLayer Requires Redraw() Every Frame

The light layer does not auto-update. You must call `Redraw()` each frame:

```go
scene.SetUpdateFunc(func() error {
    lightLayer.Redraw()
    return nil
})
```

Also: ambient alpha is inverted — `0 = fully lit`, `1 = fully dark`. And texture lights need `lightLayer.SetPages(atlas.Pages)`.

## Mask Nodes Are NOT Part of the Scene Tree

Do not `AddChild` a mask node. Masks live outside the scene tree, and their transforms are relative to the masked node:

```go
mask := willow.NewSprite("circle-mask", atlas.Region("circle"))
// Do NOT call scene.Root().AddChild(mask)
content.SetMask(mask)
mask.X = 10  // offset relative to content, not world
```

## Tweens Have No Auto-Tick

Willow has no global tween manager. You must call `Update(dt)` yourself every frame, and `dt` is `float32`:

```go
scene.SetUpdateFunc(func() error {
    if tween != nil && !tween.Done {
        tween.Update(float32(1.0 / 60.0))  // float32, not float64
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

## Particle Emitter Control Is on .Emitter, Not the Node

`Start()`, `Stop()`, `Reset()` live on the `.Emitter` sub-object, not the node itself:

```go
emitter.Emitter.Start()
emitter.Emitter.Stop()
count := emitter.Emitter.AliveCount()
```

## BitmapFont Page Index Collisions

BitmapFont defaults to page index 0. If your atlas also uses page 0, they conflict. Use `LoadBitmapFontPage` to pick a different index:

```go
font, err := willow.LoadBitmapFontPage(fntData, 3)
scene.RegisterPage(3, fontPageImage)
```

## TextBlock Has Its Own Invalidate

Changing `TextBlock` properties (`Content`, `Align`, `WrapWidth`) requires `node.TextBlock.Invalidate()` — not `node.Invalidate()`:

```go
node.TextBlock.Content = "New text"
node.TextBlock.Invalidate()  // NOT node.Invalidate()
```

## Polygon Points Use a Free Function

Updating polygon vertices uses a package-level function, not a method:

```go
willow.SetPolygonPoints(poly, newPoints)  // NOT poly.SetPoints(newPoints)
```

## Filters Is a Slice

The field is `Filters` (plural), type `[]willow.Filter`:

```go
sprite.Filters = []willow.Filter{willow.NewBlurFilter(4)}
```

## Dispose Is Permanent and Recursive

`Dispose()` permanently destroys a node **and all its children**. Disposed nodes cannot be reused. If you want to temporarily remove a node, use `RemoveFromParent()` instead.

## Camera Follow Lerp

Lerp = `0` means **no movement** (not instant). Lerp = `1` means instant snap. This is the opposite of some frameworks:

```go
cam.Follow(playerNode, 0, 0, 0.1)  // smooth follow
cam.Follow(playerNode, 0, 0, 1.0)  // instant snap
```

## Tilemap GID 0 Is Empty

GID 0 is reserved as empty (not rendered). Valid tile IDs start at 1:

```go
regions := []willow.TextureRegion{
    {},                       // GID 0 — empty, unused
    atlas.Region("grass"),    // GID 1
    atlas.Region("dirt"),     // GID 2
}
```

## Build and Test

```bash
go build ./...
go vet ./...
go test ./...
cd ecs && go test ./...
```
