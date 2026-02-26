# Camera & Viewport

Cameras define viewport regions on the screen and control what part of the world is visible. A scene can have multiple cameras for split-screen, minimaps, or HUD overlays.

## Creating a Camera

```go
cam := scene.NewCamera(willow.Rect{
    X: 0, Y: 0,
    Width: 800, Height: 600,
})
```

If no cameras are added, the scene uses a default full-screen camera.

## Camera Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `X`, `Y` | `float64` | `0` | World-space center position |
| `Zoom` | `float64` | `1.0` | Scale factor (>1 = zoom in, <1 = zoom out) |
| `Rotation` | `float64` | `0` | Radians, clockwise |
| `Viewport` | `Rect` | *(set at creation)* | Screen-space rectangle |
| `CullEnabled` | `bool` | `false` | Skip nodes outside visible bounds |
| `BoundsEnabled` | `bool` | `false` | Clamp camera to bounds |
| `Bounds` | `Rect` | *(zero)* | World-space clamping rectangle |

## Following a Node

Lock the camera to follow a node each frame:

```go
cam.Follow(playerNode, 0, 0, 0.1)
// offsetX, offsetY: pixel offset from node center
// lerp: smoothing factor (0 = no movement, 1 = instant snap)
```

Stop following:

```go
cam.Unfollow()
```

## Animated Scrolling

Smoothly scroll to a world position:

```go
import "github.com/tanema/gween/ease"

cam.ScrollTo(500, 300, willow.TweenConfig{Duration: 1.0, Ease: ease.InOutQuad})
```

For tile-based scrolling:

```go
cam.ScrollToTile(5, 3, 32, 32, willow.TweenConfig{Duration: 0.5})
```

## Camera Bounds

Prevent the camera from scrolling past world edges:

```go
cam.SetBounds(willow.Rect{X: 0, Y: 0, Width: 2000, Height: 1500})
cam.ClearBounds()
```

## Coordinate Conversion

Convert between screen pixels and world coordinates:

```go
worldX, worldY := cam.ScreenToWorld(mouseX, mouseY)
screenX, screenY := cam.WorldToScreen(entityX, entityY)
```

## Visible Bounds

Get the world-space rectangle currently visible through this camera:

```go
bounds := cam.VisibleBounds()
// bounds.X, bounds.Y, bounds.Width, bounds.Height
```

## Culling

When `CullEnabled` is true, nodes whose world-space bounds don't intersect the camera's visible area are skipped during rendering:

```go
cam.CullEnabled = true
```

This can significantly reduce rendering work for large worlds with many off-screen nodes.

## Multiple Cameras

```go
// Main game view
mainCam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: 600, Height: 600})

// Minimap in the corner
minimap := scene.NewCamera(willow.Rect{X: 610, Y: 10, Width: 180, Height: 180})
minimap.Zoom = 0.1

// Remove a camera
scene.RemoveCamera(minimap)

// List all cameras
cams := scene.Cameras()
```

## Manual Dirty Marking

If you modify camera fields directly, call `Invalidate()` to ensure the view matrix is recomputed:

```go
cam.Zoom = 2.0
cam.Invalidate()
```

The `Follow`, `ScrollTo`, and `SetBounds` methods handle this automatically.

## Next Steps

- [Text & Fonts](?page=text-and-fonts)  -  bitmap and TTF text rendering
- [Tilemap Viewport](?page=tilemap-viewport)  -  tile-based map rendering with camera binding

## Related

- [Transforms](?page=transforms)  -  node-level coordinate conversion (WorldToLocal, LocalToWorld)
- [Tweens & Animation](?page=tweens-and-animation)  -  easing functions used by `ScrollTo`
- [Scene](?page=scene)  -  camera creation via `scene.NewCamera()`
- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  screen-to-world hit testing
