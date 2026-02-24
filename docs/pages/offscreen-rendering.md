# Offscreen Rendering

`RenderTexture` provides an offscreen image buffer for manual drawing operations. Use it for procedural textures, composite effects, minimaps, or any content that needs to be rendered independently and then displayed as a sprite.

## Creating a RenderTexture

```go
rt := willow.NewRenderTexture(256, 256)
```

## Drawing Operations

```go
// Clear to transparent
rt.Clear()

// Fill with a solid color
rt.Fill(willow.Color{R: 0.1, G: 0.1, B: 0.2, A: 1})

// Draw an Ebitengine image with DrawImageOptions
rt.DrawImage(srcImage, &ebiten.DrawImageOptions{})

// Draw at a position with a blend mode
rt.DrawImageAt(srcImage, 50, 30, willow.BlendNormal)

// Draw an atlas sprite
rt.DrawSprite(region, 10, 20, willow.BlendNormal, atlas.Pages)

// Draw with full options (scale, rotation, color, etc.)
rt.DrawSpriteColored(region, willow.RenderTextureDrawOpts{
    X: 100, Y: 100,
    ScaleX: 2, ScaleY: 2,
    Rotation: 0.5,
    Color: willow.Color{R: 1, G: 0.5, B: 0.5, A: 1},
    Alpha: 0.8,
    BlendMode: willow.BlendAdd,
}, atlas.Pages)

// Same for raw images
rt.DrawImageColored(srcImage, willow.RenderTextureDrawOpts{
    X: 50, Y: 50,
    ScaleX: 1, ScaleY: 1,
})
```

### RenderTextureDrawOpts

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `X`, `Y` | `float64` | `0` | Draw position |
| `ScaleX`, `ScaleY` | `float64` | `1.0` (if 0) | Scale multiplier |
| `Rotation` | `float64` | `0` | Radians, clockwise |
| `PivotX`, `PivotY` | `float64` | `0` | Transform origin |
| `Color` | `Color` | white (if zero) | Multiplicative tint |
| `Alpha` | `float64` | `1.0` (if 0) | Opacity |
| `BlendMode` | `BlendMode` | `BlendNormal` | Blend operation |

## Displaying in the Scene

Create a sprite node backed by the render texture:

```go
spriteNode := rt.NewSpriteNode("minimap")
spriteNode.X = 600
spriteNode.Y = 10
scene.Root().AddChild(spriteNode)
```

Or access the raw `*ebiten.Image` for use with other APIs:

```go
img := rt.Image()
```

## Resizing

```go
rt.Resize(512, 512)
```

This creates a new internal image. Previous content is discarded.

## Dimensions

```go
w := rt.Width()
h := rt.Height()
```

## Cleanup

```go
rt.Dispose()
```

## Example: Procedural Background

```go
rt := willow.NewRenderTexture(800, 600)
rt.Fill(willow.Color{R: 0.05, G: 0.05, B: 0.15, A: 1})

// Draw some stars
for i := 0; i < 100; i++ {
    x := rand.Float64() * 800
    y := rand.Float64() * 600
    rt.DrawImageAt(starImage, x, y, willow.BlendAdd)
}

bg := rt.NewSpriteNode("stars")
scene.Root().AddChild(bg)
```

## Next Steps

- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  making nodes interactive
- [Tweens & Animation](?page=tweens-and-animation)  -  animate properties over time

## Related

- [Post-Processing Filters](?page=post-processing-filters)  -  shader effects that use offscreen buffers
- [Lighting](?page=lighting)  -  light layer uses a RenderTexture internally
- [CacheAsTexture](?page=cache-as-texture)  -  cache static offscreen content
