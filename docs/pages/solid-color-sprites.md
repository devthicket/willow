# Solid-Color Sprites

Willow lets you create solid-color rectangles without loading any textures. Pass an empty `TextureRegion{}` to `NewSprite` and it uses `willow.WhitePixel`  -  a shared 1x1 white image  -  internally.

## Basic Usage

```go
rect := willow.NewSprite("rect", willow.TextureRegion{})
rect.Color = willow.Color{R: 0, G: 0.5, B: 1, A: 1}  // blue
rect.ScaleX = 200  // width in pixels
rect.ScaleY = 50   // height in pixels
scene.Root().AddChild(rect)
```

The `Color` field tints the white pixel to produce the fill color, and `ScaleX`/`ScaleY` control the size. This is the standard way to draw solid-color shapes in Willow.

### NewRect Shorthand

`NewRect` combines all three steps (create sprite, set color, set size) into one call:

```go
rect := willow.NewRect("rect", 200, 50, willow.Color{R: 0, G: 0.5, B: 1, A: 1})
scene.Root().AddChild(rect)
```

This is equivalent to creating a `NewSprite` with `TextureRegion{}`, then setting `ScaleX`, `ScaleY`, and `Color`.

## Why Not Create Images Directly?

Never create unique `*ebiten.Image` instances for solid colors. Each unique image increases texture atlas fragmentation and prevents batching. Using `WhitePixel` means all solid-color sprites share the same source texture, which keeps rendering efficient.

## Common Patterns

### Background panel

```go
bg := willow.NewSprite("bg", willow.TextureRegion{})
bg.Color = willow.Color{R: 0.1, G: 0.1, B: 0.15, A: 0.9}
bg.ScaleX = 400
bg.ScaleY = 300
```

### Debug visualization

```go
hitbox := willow.NewSprite("hitbox", willow.TextureRegion{})
hitbox.Color = willow.Color{R: 1, G: 0, B: 0, A: 0.3}
hitbox.ScaleX = float64(width)
hitbox.ScaleY = float64(height)
entity.AddChild(hitbox)
```

### Color from `Node.Color`

Since `Color` is a field on `Node`, you can change it at any time  -  no need to rebuild or swap textures:

```go
// Flash red on hit
sprite.Color = willow.Color{R: 1, G: 0.2, B: 0.2, A: 1}
```

## Color Conversions

`willow.Color` uses float64 components in [0, 1]. Conversion helpers bridge to other color representations:

```go
// From 8-bit RGBA (0-255)
c := willow.ColorFromRGBA(255, 128, 0, 255)  // orange

// From HSV (all in [0, 1])  -  useful for procedural palettes and color cycling
c := willow.ColorFromHSV(0.6, 0.8, 1.0)  // bright blue
```

`willow.Color` also satisfies Go's `image/color.Color` interface, so you can pass it directly to `image.Set()` or any API that accepts `color.Color`:

```go
img.Set(x, y, willow.ColorFromRGBA(r, g, b, 255))
```

## Next Steps

- [Sprites](?page=sprites)  -  loading textures and TextureRegion
- [Camera & Viewport](?page=camera-and-viewport)  -  viewport setup and scrolling

## Related

- [Nodes](?page=nodes)  -  node types, visual properties, blend modes
- [Transforms](?page=transforms)  -  position, scale, rotation, pivot
