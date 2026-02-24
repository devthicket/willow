# Post-Processing Filters

<p align="center">
  <img src="gif/shaders.gif" alt="Shaders demo" width="400">
</p>

Filters are post-processing effects applied to individual nodes. When a node has filters, it is rendered to an offscreen buffer, the filters are applied in sequence (ping-ponging between buffers), and the result is composited back.

## Filter Interface

```go
type Filter interface {
    Apply(src, dst *ebiten.Image)
    Padding() int  // extra pixels needed around source bounds
}
```

## Assigning Filters

```go
sprite.Filters = []willow.Filter{
    willow.NewOutlineFilter(2, willow.Color{R: 1, G: 0, B: 0, A: 1}),
    willow.NewBlurFilter(3),
}
```

Multiple filters chain in order  -  output of one feeds into the next.

## Built-in Filters

### ColorMatrixFilter

4x5 color matrix transformation (Kage shader):

```go
cm := willow.NewColorMatrixFilter()
cm.SetBrightness(0.2)    // [-1, 1]
cm.SetContrast(1.5)      // 1 = normal, 0 = gray
cm.SetSaturation(0)      // 1 = normal, 0 = grayscale
sprite.Filters = []willow.Filter{cm}
```

The `Matrix` field is a `[20]float64` in row-major order for direct manipulation.

### BlurFilter

Kawase iterative blur (downscale/upscale approach):

```go
blur := willow.NewBlurFilter(4)  // radius in pixels
sprite.Filters = []willow.Filter{blur}
```

`Padding()` returns the radius value.

### OutlineFilter

8-direction offset outline at any thickness:

```go
outline := willow.NewOutlineFilter(3, willow.Color{R: 0, G: 0, B: 0, A: 1})
sprite.Filters = []willow.Filter{outline}
```

### PixelPerfectOutlineFilter

1-pixel crisp outline via Kage shader (cardinal neighbor test):

```go
ppOutline := willow.NewPixelPerfectOutlineFilter(willow.Color{R: 1, G: 1, B: 0, A: 1})
sprite.Filters = []willow.Filter{ppOutline}
```

### PixelPerfectInlineFilter

Recolors edge pixels that border transparent areas:

```go
ppInline := willow.NewPixelPerfectInlineFilter(willow.Color{R: 1, G: 0, B: 0, A: 1})
sprite.Filters = []willow.Filter{ppInline}
```

### PaletteFilter

Luminance-based palette remap through a 256-entry lookup table:

```go
pf := willow.NewPaletteFilter()  // default: grayscale palette
pf.CycleOffset = 0.1            // animate palette cycling
sprite.Filters = []willow.Filter{pf}
```

Set a custom palette:

```go
var palette [256]willow.Color
for i := range palette {
    t := float64(i) / 255
    palette[i] = willow.Color{R: t, G: 0, B: 1 - t, A: 1}
}
pf.SetPalette(palette)
```

### CustomShaderFilter

Use your own Kage shader:

```go
shader, err := ebiten.NewShader(kageSource)
csf := willow.NewCustomShaderFilter(shader, 0)  // 0 padding
csf.Uniforms = map[string]any{
    "Time": float32(time),
}
csf.Images[1] = noiseTexture  // Images[0] is auto-filled with source
sprite.Filters = []willow.Filter{csf}
```

## Performance Notes

Filters require offscreen rendering, which adds overhead. For static content, combine filters with [CacheAsTexture](?page=cache-as-texture) to avoid re-applying filters every frame.

## Next Steps

- [CacheAsTree](?page=cache-as-tree)  -  command list caching for semi-static subtrees
- [CacheAsTexture](?page=cache-as-texture)  -  cache filtered content to avoid re-applying every frame

## Related

- [Offscreen Rendering](?page=offscreen-rendering)  -  filters use offscreen buffers internally
- [Clipping & Masks](?page=clipping-and-masks)  -  alpha-based clipping (non-shader approach)
- [Lighting](?page=lighting)  -  another compositing effect
- [Performance](?page=performance-overview)  -  benchmarks, batching, and optimization strategies
