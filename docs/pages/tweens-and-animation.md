# Tweens & Animation

<p align="center">
  <img src="gif/tweens.gif" alt="Tweens demo" width="400">
</p>

Willow provides tween-based animation through `TweenGroup`, powered by the [gween](https://github.com/tanema/gween) easing library.

## Tween Functions

Five property tweens are available:

```go
import "github.com/tanema/gween/ease"

// Move to position over 1 second
tween := willow.TweenPosition(node, 300, 200, 1.0, ease.InOutQuad)

// Scale to 2x over 0.5 seconds
tween := willow.TweenScale(node, 2, 2, 0.5, ease.OutBack)

// Fade to 50% alpha over 0.8 seconds
tween := willow.TweenAlpha(node, 0.5, 0.8, ease.Linear)

// Tint to red over 1 second
tween := willow.TweenColor(node, willow.Color{R: 1, G: 0, B: 0, A: 1}, 1.0, ease.InOutQuad)

// Rotate to 90 degrees over 0.5 seconds
tween := willow.TweenRotation(node, math.Pi/2, 0.5, ease.InOutCubic)
```

Each returns a `*TweenGroup`.

## Updating Tweens

There is no global tween manager  -  you call `Update(dt)` yourself:

```go
var tween *willow.TweenGroup

// In your update function:
scene.SetUpdateFunc(func() error {
    dt := float32(1.0 / 60.0)  // or use actual delta time
    if tween != nil && !tween.Done {
        tween.Update(dt)
    }
    return nil
})
```

## TweenGroup

```go
type TweenGroup struct {
    Done bool  // true when all tweens finished or target node disposed
}

func (g *TweenGroup) Update(dt float32)
```

`Done` is automatically set to `true` when:
- The tween reaches its target value
- The target node is disposed (safe  -  no dangling pointer crashes)

## Easing Functions

The `ease` package from gween provides standard easing curves:

| Function | Description |
|----------|-------------|
| `ease.Linear` | Constant speed |
| `ease.InQuad` | Accelerating from zero |
| `ease.OutQuad` | Decelerating to zero |
| `ease.InOutQuad` | Accelerate then decelerate |
| `ease.InCubic` | Cubic ease in |
| `ease.OutCubic` | Cubic ease out |
| `ease.InOutCubic` | Cubic ease in/out |
| `ease.OutBack` | Overshoot then settle |
| `ease.OutBounce` | Bouncing effect |
| `ease.OutElastic` | Elastic spring effect |

See the [gween documentation](https://github.com/tanema/gween) for the full list.

## Chaining Tweens

To sequence tweens, start the next one when the current finishes:

```go
var currentTween *willow.TweenGroup

func startSequence() {
    currentTween = willow.TweenPosition(node, 300, 200, 1.0, ease.InOutQuad)
}

// In update:
if currentTween != nil && currentTween.Done {
    // Start next tween
    currentTween = willow.TweenAlpha(node, 0, 0.5, ease.Linear)
}
```

## Example

```go
scene := willow.NewScene()

box := willow.NewSprite("box", willow.TextureRegion{})
box.Color = willow.Color{R: 0.3, G: 0.7, B: 1, A: 1}
box.ScaleX = 50
box.ScaleY = 50
box.X = 100
box.Y = 200
scene.Root().AddChild(box)

tween := willow.TweenPosition(box, 500, 200, 2.0, ease.InOutQuad)

scene.SetUpdateFunc(func() error {
    if !tween.Done {
        tween.Update(1.0 / 60.0)
    }
    return nil
})

willow.Run(scene, willow.RunConfig{Title: "Tween Demo", Width: 640, Height: 480})
```

## Next Steps

- [Particles](?page=particles)  -  CPU-simulated particle effects
- [Mesh & Distortion](?page=meshes)  -  deformable vertex geometry

## Related

- [Camera & Viewport](?page=camera-and-viewport)  -  `ScrollTo` uses the same easing functions
- [Clipping & Masks](?page=clipping-and-masks)  -  animated masks with tweens
- [Nodes](?page=nodes)  -  node properties that tweens target
