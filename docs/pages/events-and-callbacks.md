# Events & Callbacks

Willow uses typed callback functions for input events. Callbacks can be attached to individual nodes or registered at the scene level to observe all interactions.

## Node-Level Callbacks

Assign callback functions directly to nodes:

```go
node.OnPointerDown  = func(ctx willow.PointerContext) { /* ... */ }
node.OnPointerUp    = func(ctx willow.PointerContext) { /* ... */ }
node.OnPointerMove  = func(ctx willow.PointerContext) { /* ... */ }
node.OnPointerEnter = func(ctx willow.PointerContext) { /* ... */ }
node.OnPointerLeave = func(ctx willow.PointerContext) { /* ... */ }
node.OnClick        = func(ctx willow.ClickContext)   { /* ... */ }
node.OnDragStart    = func(ctx willow.DragContext)    { /* ... */ }
node.OnDrag         = func(ctx willow.DragContext)    { /* ... */ }
node.OnDragEnd      = func(ctx willow.DragContext)    { /* ... */ }
node.OnPinch        = func(ctx willow.PinchContext)   { /* ... */ }
```

## Scene-Level Handlers

Register handlers that fire for *any* node interaction. Scene-level handlers fire before per-node callbacks:

```go
handle := scene.OnClick(func(ctx willow.ClickContext) {
    fmt.Printf("Click on node: %s\n", ctx.Node.Name)
})

// Remove when no longer needed
handle.Remove()
```

All scene-level registration methods return a `CallbackHandle`:

```go
scene.OnPointerDown(fn)
scene.OnPointerUp(fn)
scene.OnPointerMove(fn)
scene.OnPointerEnter(fn)
scene.OnPointerLeave(fn)
scene.OnClick(fn)
scene.OnDragStart(fn)
scene.OnDrag(fn)
scene.OnDragEnd(fn)
scene.OnPinch(fn)
```

## Context Types

### PointerContext

```go
type PointerContext struct {
    Node      *Node
    EntityID  uint32
    UserData  any
    GlobalX, GlobalY float64  // world coordinates
    LocalX, LocalY   float64  // node-local coordinates
    Button    MouseButton
    PointerID int
    Modifiers KeyModifiers
}
```

### ClickContext

Same fields as `PointerContext`.

### DragContext

```go
type DragContext struct {
    Node              *Node
    EntityID          uint32
    UserData          any
    GlobalX, GlobalY  float64  // current world position
    LocalX, LocalY    float64  // current node-local position
    StartX, StartY    float64  // drag start position
    DeltaX, DeltaY    float64  // world-space delta since last frame
    ScreenDeltaX, ScreenDeltaY float64  // screen-space delta
    Button            MouseButton
    PointerID         int
    Modifiers         KeyModifiers
}
```

### PinchContext

```go
type PinchContext struct {
    CenterX, CenterY  float64  // pinch center
    Scale, ScaleDelta  float64  // cumulative and per-frame scale
    Rotation, RotDelta float64  // cumulative and per-frame rotation
}
```

## Drag Dead Zone

Configure how many pixels the pointer must move before a drag starts (default: 4):

```go
scene.SetDragDeadZone(8.0)
```

## Pointer Capture

Force all pointer events to go to a specific node, regardless of hit testing:

```go
scene.CapturePointer(0, myNode)   // pointerID 0 = primary mouse
scene.ReleasePointer(0)
```

## Event Types

The `EventType` enum is used primarily with ECS integration for `InteractionEvent.Type`:

```go
willow.EventPointerDown
willow.EventPointerUp
willow.EventPointerMove
willow.EventClick
willow.EventDragStart
willow.EventDrag
willow.EventDragEnd
willow.EventPinch
willow.EventPointerEnter
willow.EventPointerLeave
```

## Next Steps

- [Tweens & Animation](?page=tweens-and-animation)  -  animate nodes in response to input
- [Particles](?page=particles)  -  CPU-simulated particle effects

## Related

- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  making nodes interactive and hit shape types
- [ECS Integration](?page=ecs-integration)  -  forward interaction events to an entity store
- [Debug & Testing](?page=debug-and-testing)  -  input injection for automated testing
