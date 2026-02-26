# Clipping & Masks

<p align="center">
  <img src="gif/masks.gif" alt="Masks demo" width="400">
</p>

Masks let you clip a node's rendering to the alpha channel of another node. The mask node is not part of the scene tree  -  its transforms are relative to the masked node.

## Setting a Mask

```go
// The content to be masked
content := willow.NewSprite("photo", atlas.Region("photo"))
content.X = 100
content.Y = 100
scene.Root().AddChild(content)

// The mask shape (NOT added to scene tree)
mask := willow.NewSprite("circle-mask", atlas.Region("circle"))
mask.PivotX = 64
mask.PivotY = 64

// Apply the mask
content.SetMask(mask)
```

The mask node's alpha channel determines visibility:
- Fully opaque mask pixels = fully visible content
- Fully transparent mask pixels = fully hidden content
- Semi-transparent = partial visibility

## Clearing a Mask

```go
content.ClearMask()
```

## Querying the Mask

```go
maskNode := content.GetMask()  // nil if no mask
```

## Mask Transforms

The mask node's position, scale, rotation, and other transforms are relative to the masked node, not the world. This means moving the masked node moves the mask with it:

```go
mask.X = 32   // offset from masked node's origin
mask.Y = 32
mask.ScaleX = 0.5  // half-size relative to masked node
mask.ScaleY = 0.5
```

## Using Solid-Color Masks

For simple geometric masks, use a WhitePixel sprite:

```go
rectMask := willow.NewSprite("rect-mask", willow.TextureRegion{})
rectMask.ScaleX = 200
rectMask.ScaleY = 100

content.SetMask(rectMask)
```

## Animated Masks

Since the mask is a regular `Node`, you can animate it:

```go
mask.OnUpdate = func(dt float64) {
    mask.Rotation += 0.02
    mask.Invalidate()
}
```

Or use tweens:

```go
tween := willow.TweenScale(mask, 2, 2, willow.TweenConfig{Duration: 1.0, Ease: ease.InOutQuad})
```

## Next Steps

- [Post-Processing Filters](?page=post-processing-filters)  -  shader-based visual effects
- [CacheAsTexture](?page=cache-as-texture)  -  cache static masked content

## Related

- [Solid-Color Sprites](?page=solid-color-sprites)  -  WhitePixel sprites for geometric masks
- [Tweens & Animation](?page=tweens-and-animation)  -  animate mask transforms
- [Nodes](?page=nodes)  -  blend modes including `BlendErase` and `BlendMask`
