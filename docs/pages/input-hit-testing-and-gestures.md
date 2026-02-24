# Input & Hit Testing

Willow's input system is integrated into the scene graph. Any node can become interactive by enabling hit testing  -  Willow handles pointer dispatch, gesture recognition, and coordinate conversion automatically.

## Making Nodes Interactive

Assign a `HitShape` and register a callback (the setter enables `Interactable` automatically):

```go
button := willow.NewSprite("button", region)
button.HitShape = willow.HitRect{X: 0, Y: 0, Width: 64, Height: 32}

button.OnClick(func(ctx willow.ClickContext) {
    fmt.Println("Button clicked!")
})
// OnClick automatically enables Interactable
```

## Hit Shapes

Three built-in hit shape types:

```go
// Rectangle
willow.HitRect{X: 0, Y: 0, Width: 100, Height: 50}

// Circle
willow.HitCircle{CenterX: 32, CenterY: 32, Radius: 32}

// Convex polygon
willow.HitPolygon{Points: []willow.Vec2{
    {X: 0, Y: 0}, {X: 50, Y: 0}, {X: 25, Y: 50},
}}
```

### Custom Hit Shapes

Implement the `HitShape` interface for custom geometry:

```go
type HitShape interface {
    Contains(x, y float64) bool
}
```

Coordinates passed to `Contains` are in the node's local space.

## How Hit Testing Works

During `scene.Update()`, Willow converts pointer coordinates from screen space to world space (via the active camera), then walks the scene tree in reverse draw order. For each node with `Interactable = true`, it transforms the pointer into the node's local space and calls `HitShape.Contains()`. The first hit wins.

## Mouse Buttons

```go
willow.MouseButtonLeft
willow.MouseButtonRight
willow.MouseButtonMiddle
```

## Key Modifiers

```go
willow.ModShift
willow.ModCtrl
willow.ModAlt
willow.ModMeta
```

Check modifiers in callbacks:

```go
node.OnClick(func(ctx willow.ClickContext) {
    if ctx.Modifiers&willow.ModShift != 0 {
        fmt.Println("Shift+click!")
    }
})
```

## Next Steps

- [Events & Callbacks](?page=events-and-callbacks)  -  node-level and scene-level callback API, context types, drag, and pinch

## Related

- [Nodes](?page=nodes)  -  node properties including `Interactable` and `EntityID`
- [Transforms](?page=transforms)  -  coordinate conversion between local and world space
- [Camera & Viewport](?page=camera-and-viewport)  -  screen-to-world conversion
