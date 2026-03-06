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
tween := willow.TweenPosition(node, 300, 200, willow.TweenConfig{Duration: 1.0, Ease: ease.InOutQuad})

// Scale to 2x over 0.5 seconds
tween := willow.TweenScale(node, 2, 2, willow.TweenConfig{Duration: 0.5, Ease: ease.OutBack})

// Fade to 50% alpha over 0.8 seconds
tween := willow.TweenAlpha(node, 0.5, willow.TweenConfig{Duration: 0.8})

// Tint to red over 1 second
tween := willow.TweenColor(node, willow.RGB(1, 0, 0), willow.TweenConfig{Duration: 1.0, Ease: ease.InOutQuad})

// Rotate to 90 degrees over 0.5 seconds
tween := willow.TweenRotation(node, math.Pi/2, willow.TweenConfig{Duration: 0.5, Ease: ease.InOutCubic})
```

Each returns a `*TweenGroup`.

## Auto-Ticking

Tweens on nodes that are part of a scene auto-tick every frame  -  no manual update loop needed:

```go
box := willow.NewSprite("box", willow.TextureRegion{})
scene.Root().AddChild(box)

// This tween runs automatically  -  nothing else to do.
tween := willow.TweenPosition(box, 500, 200, willow.TweenConfig{Duration: 2.0, Ease: ease.InOutQuad})
```

The scene ticks all registered tweens during `Scene.Update()`, removes them when done, and cleans up automatically.

## TweenGroup

```go
type TweenGroup struct {
    Done bool  // true when all tweens finished or target node disposed
}

func (g *TweenGroup) Cancel()  // stop the tween early
```

`Done` is automatically set to `true` when:
- The tween reaches its target value
- The target node is disposed (safe  -  no dangling pointer crashes)
- `Cancel()` is called

## Easing Functions

Willow re-exports all gween easing functions as `willow.Ease*` constants. Type `willow.Ease` in your editor to see every option via autocomplete  -  no extra import needed.

| Function | Description |
|----------|-------------|
| `willow.EaseLinear` | Constant speed |
| `willow.EaseInQuad` | Accelerating from zero |
| `willow.EaseOutQuad` | Decelerating to zero |
| `willow.EaseInOutQuad` | Accelerate then decelerate |
| `willow.EaseInCubic` | Cubic ease in |
| `willow.EaseOutCubic` | Cubic ease out |
| `willow.EaseInOutCubic` | Cubic ease in/out |
| `willow.EaseOutBack` | Overshoot then settle |
| `willow.EaseOutBounce` | Bouncing effect |
| `willow.EaseOutElastic` | Elastic spring effect |

All 45 easing curves from gween are available: `In`, `Out`, `InOut`, and `OutIn` variants for Quad, Cubic, Quart, Quint, Sine, Expo, Circ, Elastic, Back, and Bounce. You can also import `github.com/tanema/gween/ease` directly or pass any `willow.EaseFunc` (custom easing function).

## Chaining Tweens

To sequence tweens, check `Done` in your update function and start the next:

```go
var current *willow.TweenGroup

scene.SetUpdateFunc(func() error {
    if current != nil && current.Done {
        // First tween finished  -  start the next.
        current = willow.TweenAlpha(node, 0, willow.TweenConfig{Duration: 0.5})
    }
    return nil
})

// Kick off the chain.
current = willow.TweenPosition(node, 300, 200, willow.TweenConfig{Duration: 1.0, Ease: ease.InOutQuad})
```

## Example

```go
scene := willow.NewScene()

box := willow.NewSprite("box", willow.TextureRegion{})
box.SetColor(willow.RGBA(0.3, 0.7, 1, 1))
box.SetSize(50, 50)
box.SetPosition(100, 200)
scene.Root().AddChild(box)

// Tween auto-ticks because box is already on the scene.
willow.TweenPosition(box, 500, 200, willow.TweenConfig{Duration: 2.0, Ease: ease.InOutQuad})

willow.Run(scene, willow.RunConfig{Title: "Tween Demo", Width: 640, Height: 480})
```

## Next Steps

- [Particles](?page=particles)  -  CPU-simulated particle effects
- [Mesh & Distortion](?page=meshes)  -  deformable vertex geometry

## Related

- [Camera & Viewport](?page=camera-and-viewport)  -  `ScrollTo` uses the same easing functions
- [Clipping & Masks](?page=clipping-and-masks)  -  animated masks with tweens
- [Nodes](?page=nodes)  -  node properties that tweens target
