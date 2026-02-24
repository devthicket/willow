# Text & Fonts

All text in Willow renders through **SpriteFont** (Signed Distance Field). SDF fonts store distance-to-edge per pixel, enabling smooth scaling, outlines, glows, and drop shadows via a single shader pass. Willow supports both single-channel SDF and multi-channel MSDF.

## Font Interface

```go
type Font interface {
    MeasureString(text string) (width, height float64)
    LineHeight() float64
}
```

Implemented by `*SpriteFont`.

## Creating an SDF Font

The simplest way to get an SDF font from TTF/OTF data:

```go
font, err := willow.NewFontFromTTF(ttfData, 80)  // 80px rasterization size
```

This generates an SDF atlas at runtime, registers the atlas page, and returns the font ready to use. Higher size = better quality when scaled up. Use 0 for the default (80px).

## Advanced: Low-Level Constructors

### Runtime Generation with Manual Page Registration

```go
font, atlasImg, _, err := willow.LoadSpriteFontFromTTF(ttfData, willow.SDFGenOptions{
    Size:          80,
    DistanceRange: 8,
    PageIndex:     2,
})
scene.RegisterPage(2, atlasImg)
```

### Loading Pre-generated SDF Atlas

Load a pre-generated SDF atlas (from the `sdfgen` CLI tool):

```go
metricsJSON, _ := os.ReadFile("font.json")
font, err := willow.LoadSpriteFont(metricsJSON, 2) // page index 2
scene.RegisterPage(2, atlasImage)
```

### Offline Generation

The `cmd/sdfgen` CLI tool generates SDF atlases offline:

```bash
go run ./cmd/sdfgen -font input.ttf -size 80 -range 8 -out output
```

Produces `output.png` (atlas) and `output.json` (metrics).

## SDF Effects

Set `SDFEffects` on a TextBlock for outline, glow, and shadow:

```go
node.TextBlock.SDFEffects = &willow.SDFEffects{
    OutlineWidth:   2.0,
    OutlineColor:   willow.Color{R: 0, G: 0, B: 0, A: 1},
    GlowWidth:      3.0,
    GlowColor:      willow.Color{R: 0.2, G: 0.5, B: 1, A: 0.6},
    ShadowOffset:   willow.Vec2{X: 3, Y: 3},
    ShadowColor:    willow.Color{R: 0, G: 0, B: 0, A: 0.7},
    ShadowSoftness: 1.5,
}
node.TextBlock.Invalidate()
```

All effects are rendered in a single shader pass — no multi-pass overhead.

## Creating Text Nodes

```go
label := willow.NewText("label", "Hello, Willow!", font)
label.X = 100
label.Y = 50
scene.Root().AddChild(label)
```

## TextBlock

The `TextBlock` struct controls text layout and appearance:

```go
type TextBlock struct {
    Content    string
    Font       Font
    Align      TextAlign    // Left, Center, Right
    WrapWidth  float64      // 0 = no wrapping
    Color      Color
    Outline    *Outline     // nil = no outline
    LineHeight float64      // 0 = use Font.LineHeight()
}
```

Access the text block via the node:

```go
node := willow.NewText("msg", "Initial text", font)
node.TextBlock.Align = willow.TextAlignCenter
node.TextBlock.WrapWidth = 300
node.TextBlock.Color = willow.Color{R: 1, G: 1, B: 0, A: 1}
```

### Updating Text

```go
node.TextBlock.Content = "New text content"
node.TextBlock.Invalidate()
```

Always call `Invalidate()` after changing `TextBlock` properties to trigger re-layout.

## Text Alignment

```go
willow.TextAlignLeft    // default
willow.TextAlignCenter
willow.TextAlignRight
```

Alignment is relative to the node's X position (left-aligned) or within the `WrapWidth` (center/right-aligned).

## Word Wrapping

Set `WrapWidth` to enable automatic line breaking:

```go
node.TextBlock.WrapWidth = 400  // wrap at 400 pixels
node.TextBlock.Invalidate()
```

## Measuring Text

```go
width, height := font.MeasureString("Hello!")
lineH := font.LineHeight()
```

## Next Steps

- [Tilemap Viewport](?page=tilemap-viewport) — tile-based map rendering
- [Polygons](?page=polygons) — point-list polygon shapes

## Related

- [Sprites & Atlas](?page=sprites-and-atlas) — atlas loading and page registration
- [Nodes](?page=nodes) — node types and visual properties
