# LLM Quick-Reference

Common pitfalls and patterns that trip up AI code generators. Read this before writing Willow code.

## Camera Is Optional

A camera is **not required** for basic Willow apps. Without a camera, the scene renders with a default identity transform (world origin at top-left, 1:1 pixel mapping). Only create a camera when you need viewport features like culling, scrolling, zooming, or follow.

When creating a camera, you must position it at the center of the viewport  -  not at (0, 0). The camera's X/Y is where it *looks*, so centering it on screen means the world origin aligns with the top-left corner of the window.

```go
cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: screenW, Height: screenH})
cam.X = screenW / 2
cam.Y = screenH / 2
cam.Invalidate()
```

Without this centering, everything renders offset by half the screen.

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

## Callback Setters Auto-Enable Interactable

Callback setter methods (`OnClick`, `OnDrag`, `OnPointerDown`, etc.) automatically set `Interactable = true` on the node. You do not need to enable it manually:

```go
node.OnClick(func(ctx willow.ClickContext) {
    // Interactable is now true automatically
})
```

You can still set `node.Interactable = false` to manually disable hit testing on a node without removing its callbacks.

## Hit Testing and Draggable Sprites

For WhitePixel sprites (created with `willow.TextureRegion{}`), **do not set HitShape** unless you have a specific reason. The default hit test derives an AABB from the node's dimensions automatically.

```go
// Correct  -  let the default AABB handle it:
sp := willow.NewSprite("handle", willow.TextureRegion{})
sp.SetSize(32, 32)
sp.SetPivot(0.5, 0.5)  // 1×1 WhitePixel, so 0.5 is the true center
sp.OnDrag(func(ctx willow.DragContext) {
    sp.SetPosition(sp.X()+ctx.DeltaX, sp.Y()+ctx.DeltaY)
})
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

Create solid-color shapes with an empty TextureRegion. Size them with `SetSize()` and tint with `SetColor()`:

```go
sprite := willow.NewSprite("box", willow.TextureRegion{})
sprite.SetSize(40, 40)                    // width and height in pixels
sprite.SetColor(willow.RGB(1, 0, 0))      // red
```

Do **not** create unique `ebiten.Image` instances for solid-color rectangles.

## Polygons

Create arbitrary polygon shapes with a slice of Vec2 points:

```go
tri := willow.NewPolygon("triangle", []willow.Vec2{
    {X: 0, Y: -40}, {X: 35, Y: 30}, {X: -35, Y: 30},
})
tri.SetColor(willow.RGB(1, 1, 1))  // must set  -  zero color is invisible
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

`NewContainer` returns `*Node`  -  same type as sprites and polygons. Containers have no visual of their own.

Use `container.RemoveChildren()` to detach all children. Use `child.RemoveFromParent()` to detach one child without destroying it.

## Use Setter Methods for Property Changes

Node fields are private. Use setter methods which automatically invalidate the transform:

```go
node.SetPosition(node.X() + dx, node.Y() + dy)
node.SetRotation(node.Rotation() + 0.01)
```

Available setter methods: `SetPosition(x, y)`, `SetX(x)`, `SetY(y)`, `SetScale(sx, sy)`, `SetSize(w, h)`, `SetRotation(r)`, `SetPivot(px, py)`, `SetAlpha(a)`, `SetColor(c)`, `SetVisible(v)`, `SetZIndex(z)`, `SetRenderLayer(l)`, `SetGlobalOrder(o)`, `SetBlendMode(b)`, `SetRenderable(r)`, `SetTextureRegion(r)`.

## Alpha vs Color.A()

`Alpha()` and `Color().A()` are separate properties that multiply together. Use `SetAlpha()` for fading nodes in/out. Use `SetColor()` for tinting.

```go
node.SetAlpha(0.5)                                 // 50% transparent
node.SetColor(willow.RGB(1, 0, 0))                 // red tint at full color alpha
```

`nodeDefaults` initializes both Alpha to 1 and Color to white `{1,1,1,1}`, so new nodes are fully visible.

## Color Zero Value Makes Nodes Invisible

Color is multiplicative. A zero-value color means `{R:0, G:0, B:0, A:0}`  -  fully transparent black. `NewSprite` initializes Color to white automatically, but if you set it to a zero color, the node vanishes:

```go
// WRONG  -  sprite will be invisible:
sprite.SetColor(willow.RGBA(0, 0, 0, 0))  // alpha 0 → invisible

// CORRECT  -  no tint:
sprite.SetColor(willow.RGB(1, 1, 1))
```

## Rotation Is Radians, Not Degrees

```go
// WRONG:
node.SetRotation(90)  // this is ~14 full rotations, not 90°

// CORRECT:
node.SetRotation(math.Pi / 2)  // 90° clockwise
```

## Pivot Is Local Pixels, Not Normalized

PivotX/Y are in **local pixel coordinates**, not normalized 0–1. To center a sprite, use half the image width/height:

```go
// WRONG  -  0.5 pixels, not center:
sprite.SetPivot(0.5, 0.5)

// CORRECT for a 64×64 atlas sprite:
sprite.SetPivot(32, 32)  // half of 64px width/height

// CORRECT for a SetCustomImage sprite:
sprite.SetCustomImage(img)
sprite.SetPivot(float64(img.Bounds().Dx())/2, float64(img.Bounds().Dy())/2)
```

WhitePixel sprites are 1×1, so `SetPivot(0.5, 0.5)` does center them  -  but only by coincidence. For any image larger than 1×1 (atlas regions, custom images), always use `width/2` and `height/2`.

## Visible vs Renderable

`SetVisible(false)` skips the node **and all its children**. If you want to hide only the parent's visual while keeping children visible, use `SetRenderable(false)`:

```go
parent.SetVisible(false)     // hides parent AND all children
parent.SetRenderable(false)  // hides only this node's visual; children still render
```

## Scene Graph Z-Order

Nodes render in tree order (depth-first). Later children render on top of earlier ones. Control z-order by adding nodes in the right sequence or using `SetZIndex()`:

```go
// Ropes first (behind), then handles (on top)
scene.Root().AddChild(ropeNode)
scene.Root().AddChild(handleNode)

// Or use ZIndex for explicit ordering within a parent:
ropeNode.SetZIndex(0)
handleNode.SetZIndex(10)
```

## Input Context Structs

Callbacks receive context structs with useful fields:

```go
node.OnClick(func(ctx willow.ClickContext) {
    ctx.Node      // *Node  -  the clicked node
    ctx.GlobalX   // world-space X
    ctx.GlobalY   // world-space Y
    ctx.LocalX    // node-local X
    ctx.LocalY    // node-local Y
    ctx.Button    // MouseButton
    ctx.Modifiers // KeyModifiers held during click
})

node.OnDrag(func(ctx willow.DragContext) {
    ctx.Node         // *Node  -  the dragged node
    ctx.DeltaX       // world-space movement since last drag event
    ctx.DeltaY       // world-space movement since last drag event
    ctx.ScreenDeltaX // screen-pixel movement (for camera panning)
    ctx.ScreenDeltaY // screen-pixel movement (for camera panning)
    ctx.StartX       // world X where drag began
    ctx.StartY       // world Y where drag began
    ctx.GlobalX      // current world X
    ctx.GlobalY      // current world Y
})
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
node.OnDragEnd(func(ctx willow.DragContext) {
    // snap to grid, validate position, etc.
})
```

`OnUpdate` runs per-frame logic on individual nodes instead of centralizing in `SetUpdateFunc`:

```go
node.OnUpdate = func(dt float64) {  // float64, not float32
    node.SetRotation(node.Rotation() + 0.02)
}
```

Other callbacks: `OnPinch func(PinchContext)`, `OnPointerEnter func(PointerContext)`, `OnPointerLeave func(PointerContext)`.

## Drag Dead Zone

By default, drags require a small movement before firing. Set to 0 for immediate response (useful for camera panning):

```go
scene.SetDragDeadZone(0)
```

## NewRope and NewDistortionGrid Return a Single Value

`NewRope` and `NewDistortionGrid` return a single controller object. Use `.Node()` to get the scene node:

```go
rope := willow.NewRope("cable", img, nil, config)
scene.Root().AddChild(rope.Node())  // .Node() returns the renderable *Node
rope.Update()                        // call methods on the controller

grid := willow.NewDistortionGrid("water", img, 10, 8)
scene.Root().AddChild(grid.Node())
grid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) { ... })
```

## Wrapper Types Need .Node() to Add to Scene

`LightLayer`, `TileMapViewport`, `Rope`, and `DistortionGrid` are not `*Node`. Call `.Node()` to get the renderable node:

```go
ll := willow.NewLightLayer(800, 600, 0.8)
scene.Root().AddChild(ll.Node())  // NOT ll

viewport := willow.NewTileMapViewport("map", 32, 32)
scene.Root().AddChild(viewport.Node())  // NOT viewport

rope := willow.NewRope("cable", img, nil, config)
scene.Root().AddChild(rope.Node())  // NOT rope

grid := willow.NewDistortionGrid("water", img, 10, 8)
scene.Root().AddChild(grid.Node())  // NOT grid
```

## Rope Endpoint Binding (Pointer Stability)

Ropes read their Start/End/Controls positions through pointers. You must ensure the pointers remain valid and that you mutate the **same** Vec2 the Rope references.

**Common mistake:** creating a local Vec2, passing its address to `NewRope`, then copying it into a struct. The Rope still points at the original local  -  your struct copy is a different variable.

```go
// WRONG  -  pointer aliases a loop-local that won't be updated:
for i := range n {
    startPos := willow.Vec2{X: 100, Y: 200}
    r := willow.NewRope("rope", img, nil, willow.RopeConfig{
        Start: &startPos,  // points at loop-local
    })
    ropes[i] = myRope{start: startPos, r: r}  // copies value, Rope still reads loop-local
}
ropes[0].start.X = 999  // Rope never sees this!

// CORRECT  -  allocate the struct first, then point at its fields:
rb := &ropeBinding{
    start: willow.Vec2{X: 100, Y: 200},
    end:   willow.Vec2{X: 400, Y: 200},
}
r := willow.NewRope("rope", img, nil, willow.RopeConfig{
    Start: &rb.start,  // points directly at the struct field
    End:   &rb.end,
})
rb.r = r

// Each frame  -  mutate the same Vec2s the Rope reads:
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

### Dynamic Atlas (Runtime Packing)

For many runtime-loaded images (10+) that should batch together, use `NewAtlas` + `Add` (shelf mode) or `NewBatchAtlas` + `Stage` + `Pack` (batch mode):

```go
// Shelf mode  -  add/free/replace over time:
atlas := willow.NewAtlas(willow.PackerConfig{PageWidth: 512, PageHeight: 512})
region, err := atlas.Add("avatar", avatarImg)
sprite := willow.NewSprite("avatar", region)

// Free and reuse space:
atlas.Free("avatar")                           // reclaims space, clears pixels
atlas.Replace("skin", newSkinImg)              // same-size pixel swap in-place
region, err = atlas.Update("skin", diffImg)    // same size → Replace, different → Free + Add
```

```go
// Batch mode  -  stage all, pack once (better packing efficiency):
atlas := willow.NewBatchAtlas(willow.PackerConfig{PageWidth: 1024, PageHeight: 1024})
atlas.Stage("a", imgA)
atlas.Stage("b", imgB)
atlas.Pack()  // runs MaxRects, immutable after this
sprite := willow.NewSprite("a", atlas.Region("a"))
```

**Do NOT use dynamic atlas for a few images.** `SetCustomImage` or `RegisterPage` is simpler and wastes no VRAM. Dynamic atlas allocates full pages (e.g. 512x512 = 1 MB each). Only use it when per-image batch breaks are a measured performance problem.

Use shelf mode when images arrive over time and may need updating. Use batch mode when all images are known upfront (asset loading, level init)  -  it packs more efficiently.

## BlendMode

Use `node.SetBlendMode()` for compositing effects:

```go
willow.BlendNormal    // standard alpha blending (default)
willow.BlendAdd       // additive  -  overlapping particles brighten
willow.BlendMultiply  // multiply  -  only darkens (used by LightLayer)
willow.BlendScreen    // screen  -  only brightens
willow.BlendErase     // destination-out  -  punches transparent holes
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
    Color:  willow.RGB(1, 0.9, 0.7),
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

**The mask root node's own transform is ignored by the renderer**  -  only the transforms of its *children* are applied. Animated masks must use a container as root and put the actual shape one level below:

```go
maskRoot := willow.NewContainer("mask-root")
shape := willow.NewPolygon("star", starPoints)
shape.SetX(150)  // transforms on children ARE applied
maskRoot.AddChild(shape)
content.SetMask(maskRoot)
```

## Tweens Auto-Tick When Attached to a Scene

Tweens on nodes that are part of a Scene are updated automatically each frame  -  no manual `Update(dt)` call is needed. Just create the tween and it runs:

```go
willow.TweenPosition(node, 200, 300, willow.TweenConfig{Duration: 1.0, Ease: willow.EaseOutCubic})  // starts automatically
```

## SetUpdateFunc Replaces  -  It Does Not Chain

Calling `scene.SetUpdateFunc()` a second time replaces the previous callback. Put all per-frame logic in one function:

```go
// WRONG  -  only the last one runs:
scene.SetUpdateFunc(func() error { lightLayer.Redraw(); return nil })
scene.SetUpdateFunc(func() error { updateEnemies(); return nil })

// CORRECT  -  one function with everything:
scene.SetUpdateFunc(func() error {
    lightLayer.Redraw()
    updateEnemies()
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

Willow has two font types: `*DistanceFieldFont` (SDF-based, smooth scaling) and `*PixelFont` (bitmap, pixel-perfect integer scaling). Both implement the `Font` interface and work with `NewText`.

### DistanceFieldFont (SDF)

The primary constructor is `NewFontFromTTF`:

```go
font, err := willow.NewFontFromTTF(ttfData, 80)  // returns (*DistanceFieldFont, error)
label := willow.NewText("label", "Hello", font)   // returns *Node
label.SetFontSize(24)                              // display size in pixels (default 16)
scene.Root().AddChild(label)
```

`NewFontFromTTF` generates the font atlas and registers the page automatically  -  no `RegisterPage` call needed. Size is the rasterization size (higher = sharper when scaled up). Use 0 for the default (80px).

**FontSize** is applied at render time as a scale factor, independent of `ScaleX()`/`ScaleY()`. This means you can animate node scale without affecting text size, and vice versa. Defaults to 16. Set to 0 to use the font's native atlas size.

**WrapWidth** is in screen pixels. No manual division by scale is needed.

Use `SetTextEffects()` for outline, glow, and shadow. `nil` means plain fill only:

```go
label.SetTextEffects(&willow.TextEffects{
    OutlineWidth: 2.0,
    OutlineColor: willow.RGB(0, 0, 0),
})
```

### Text Properties

Use convenience setters on the node for common text properties:

```go
label.TextBlock.Content = "New text"
label.TextBlock.Invalidate()                         // for Content, use TextBlock directly
label.SetFontSize(24)
label.SetAlign(willow.TextAlignCenter)
label.SetWrapWidth(400)                              // screen pixels
label.SetTextColor(willow.RGB(1, 1, 1))
label.SetTextEffects(&willow.TextEffects{OutlineWidth: 2, OutlineColor: willow.RGBA(0, 0, 0, 1)})
```

Text convenience setters (`SetFontSize`, `SetTextColor`, `SetAlign`, `SetWrapWidth`, `SetLineHeight`, `SetTextEffects`) handle `TextBlock.Invalidate()` automatically. For `TextBlock.Content`, call `TextBlock.Invalidate()` manually.

### Font Measurement

```go
// Native atlas pixels:
w, h := font.MeasureString("Hello World")

// Display pixels (scaled by FontSize):
dw, dh := node.TextBlock.MeasureDisplay("Hello World")
```

### PixelFont (Bitmap)

For pixel art games with monospaced spritesheet fonts. Each character is a fixed cell in a grid. Scaling is integer-only (1x, 2x, 3x) to stay pixel-perfect.

```go
font := willow.NewPixelFont(sheetImg, 16, 16, "!\"#$%&'()*+,-./0123...")
label := willow.NewText("label", "Hello!", font)
scene.Root().AddChild(label)
```

- `sheetImg`: the spritesheet `*ebiten.Image` (registered as an atlas page automatically)
- `cellW, cellH`: pixel dimensions of each character cell
- `chars`: string mapping grid positions to characters (left-to-right, top-to-bottom), starting at the first glyph (typically `!`). Space does not need a glyph  -  it advances the cursor automatically.

**FontSize** controls integer scaling via `TextBlock.FontSize`. The scale factor is `FontSize / cellH`, rounded to the nearest integer (minimum 1x). With a 16px cell: FontSize 16 = 1x, 32 = 2x, 48 = 3x.

```go
label.SetFontSize(32)  // 2x scale for a 16px cell font
```

**TrimCell** trims dead pixels from each side of every cell, tightening character spacing. Parameters follow CSS order: top, right, bottom, left.

```go
font.TrimCell(0, 4, 0, 4)  // trim 4px left/right: 16px cell → 8px advance
```

**WrapWidth** and **Align** work the same as DistanceFieldFont. No shader is used  -  bitmap text renders with `DrawTriangles` and `FilterNearest`.

**Color** works consistently across both font types: `SetTextColor()` sets the fill color, `SetColor()` is a tint multiplier on top (same as how node color tints sprites). The final pixel color is `TextColor * Color`.

**TextEffects** (outline, glow, shadow) are DistanceFieldFont-only and have no effect on PixelFont.

## Filters Is a Slice

The field is `Filters` (plural), type `[]willow.Filter`:

```go
sprite.Filters = []willow.Filter{willow.NewBlurFilter(4)}
```

## CacheAsTree for Static Geometry

Cache a subtree's render commands to avoid re-traversal. Ideal for large tilemaps:

```go
container.SetCacheAsTree(true, willow.CacheTreeManual)
// CacheTreeManual  -  you call container.InvalidateCacheTree() when tiles change
// CacheTreeAuto   -  auto-invalidates when setters on descendants are called (default)
```

## TileMapViewport

Full setup pattern:

```go
viewport := willow.NewTileMapViewport("world", tileW, tileH)
viewport.SetCamera(cam)
scene.Root().AddChild(viewport.Node())

layer := viewport.AddTileLayer(willow.TileLayerConfig{
    Name: "ground", Width: mapW, Height: mapH,
    Data: gidData, Regions: regions, AtlasImage: tilesetImg,
})
layer.Node().SetRenderLayer(0)  // controls draw order within the viewport
```

GID data is `[]uint32`. GID 0 is reserved as empty (not rendered). Valid tile IDs start at 1:

```go
regions := []willow.TextureRegion{
    {},                       // GID 0  -  empty, unused
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
