# Text & Fonts

Willow has two font types that both implement the `Font` interface:

- **SpriteFont**  -  SDF (Signed Distance Field) font converted from TTF/OTF, smooth at any size, supports outlines/glows/shadows
- **PixelFont**  -  bitmap spritesheet, pixel-perfect integer scaling, ideal for retro/pixel art games

## Font Interface

```go
type Font interface {
    MeasureString(text string) (width, height float64)  // native pixels
    LineHeight() float64                                  // native pixels
}
```

Both `*SpriteFont` and `*PixelFont` implement this interface. All measurements are in native pixels. Use `TextBlock.MeasureDisplay()` for display-scaled values.

## Creating Text Nodes

Both font types use the same constructor:

```go
label := willow.NewText("label", "Hello, Willow!", font)
label.TextBlock.FontSize = 24
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
    FontSize   float64      // display size in pixels (default 16); 0 = native size
    Align      TextAlign    // Left, Center, Right
    WrapWidth  float64      // screen pixels; 0 = no wrapping
    Color       Color
    LineHeight  float64       // 0 = use Font.LineHeight()
    TextEffects *TextEffects  // nil = no effects (SpriteFont only)
}
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

Set `WrapWidth` to enable automatic line breaking. Values are in screen pixels:

```go
node.TextBlock.WrapWidth = 400  // wrap at 400 screen pixels
node.TextBlock.Invalidate()
```

## Measuring Text

```go
// Native pixels:
width, height := font.MeasureString("Hello!")
lineH := font.LineHeight()

// Display pixels (scaled by FontSize):
dispW, dispH := node.TextBlock.MeasureDisplay("Hello!")
```

---

## SpriteFont (SDF)

SDF stands for **Signed Distance Field**  -  a technique where each pixel in the font atlas stores how far it is from the nearest glyph edge, rather than a simple on/off value. This lets the GPU reconstruct sharp edges at any scale using a single texture lookup, and enables effects like outlines, glows, and drop shadows in the same shader pass.

SpriteFont converts TTF/OTF font files into an SDF glyph atlas (a texture). This conversion can happen at runtime or offline ahead of time using the `fontgen` CLI tool. Either way, the result is the same: a texture that the GPU samples each frame for resolution-independent text rendering.

### Creating a SpriteFont

#### From TTF at Runtime

The simplest path  -  convert TTF/OTF data into an SDF atlas at startup:

```go
font, err := willow.NewFontFromTTF(ttfData, 80)  // 80px rasterization size
```

This generates the SDF glyph atlas, registers it as an atlas page, and returns the font ready to use. Higher size = better quality when scaled up. Use 0 for the default (80px).

#### From a Pre-generated Atlas

For faster startup or to avoid bundling TTF files, pre-generate the atlas offline with the `fontgen` CLI tool and load the result:

```go
metricsJSON, _ := os.ReadFile("font.json")
font, err := willow.LoadSpriteFont(metricsJSON, 2) // page index 2
scene.RegisterPage(2, atlasImage)
```

See [Font Generation (fontgen)](?page=fontgen) for the CLI tool.

### Offline Generation (fontgen)

The `cmd/fontgen` CLI tool converts a TTF/OTF file into an SDF atlas ahead of time:

```bash
go run ./cmd/fontgen -font input.ttf -size 80 -range 8 -out output
```

Produces `output.png` (atlas texture) and `output.json` (glyph metrics). Load these with `LoadSpriteFont` as shown above.

### FontSize

**FontSize** is applied at render time as a scale factor (`FontSize / native line height`), independent of `ScaleX`/`ScaleY`. Defaults to 16.

### Advanced: Runtime Generation with Manual Page Registration

```go
font, atlasImg, _, err := willow.LoadSpriteFontFromTTF(ttfData, willow.SDFGenOptions{
    Size:          80,
    DistanceRange: 8,
    PageIndex:     2,
})
scene.RegisterPage(2, atlasImg)
```

### Text Effects

Set `TextEffects` on a TextBlock for outline, glow, and shadow:

```go
node.TextBlock.TextEffects = &willow.TextEffects{
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

All effects are rendered in a single shader pass  -  no multi-pass overhead. Text effects are SpriteFont-only and have no effect on PixelFont.

---

## PixelFont (Bitmap)

Pixel-perfect bitmap font renderer for monospaced spritesheet fonts. Each character occupies a fixed cell in a grid on a single image. Scaling is integer-only (1x, 2x, 3x) to preserve crisp pixels.

### Creating a PixelFont

```go
font := willow.NewPixelFont(sheetImg, 16, 16, chars)
```

Parameters:

- `sheetImg`  -  the spritesheet `*ebiten.Image` (registered as an atlas page automatically)
- `cellW, cellH`  -  pixel dimensions of each character cell
- `chars`  -  string mapping grid positions to characters, left-to-right, top-to-bottom

The `chars` string starts at the first glyph in the spritesheet. For a font starting at `!`:

```go
const chars = "!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
font := willow.NewPixelFont(sheetImg, 16, 16, chars)
```

Space characters do not need a glyph in the spritesheet  -  they advance the cursor by one cell width automatically.

### Scaling

PixelFont scaling is controlled per-node via `TextBlock.FontSize`, the same as SpriteFont. The scale factor is `FontSize / cellH`, rounded to the nearest integer to stay pixel-perfect (minimum 1x).

```go
// With a 16px cell height:
label.TextBlock.FontSize = 16  // 1x (native size, also the default)
label.TextBlock.FontSize = 32  // 2x
label.TextBlock.FontSize = 48  // 3x
label.TextBlock.Invalidate()
```

Fractional values are rounded: FontSize 20 with cellH 16 rounds to 1x, FontSize 28 rounds to 2x.

### Trimming Cell Padding

Many pixel font spritesheets have dead space around each glyph. `TrimCell` trims pixels from each side of every cell, tightening character spacing and line height. Parameters follow CSS order: top, right, bottom, left.

```go
font.TrimCell(0, 4, 0, 4)  // trim 4px left and right
```

A 16x16 cell with `TrimCell(0, 4, 0, 4)` renders 8px-wide glyphs spaced 8px apart. The source region is cropped to match, so only the inner pixels are drawn.

All four sides can be trimmed independently:

```go
font.TrimCell(2, 4, 2, 4)  // trim 2px top/bottom, 4px left/right
```

### Rendering

PixelFont renders with `DrawTriangles` and `FilterNearest`  -  no shader is used. `TextBlock.Color` is the fill color; `Node.Color` multiplies on top as a tint (same as how Node.Color tints sprites):

```go
label := willow.NewText("msg", "Game Over", font)
label.TextBlock.Color = willow.Color{R: 1, G: 0.3, B: 0.3, A: 1}  // red fill
label.TextBlock.Invalidate()
label.Color = willow.Color{R: 0.8, G: 0.8, B: 0.8, A: 1}          // optional dim tint
```

### What Works

- `TextBlock.FontSize`  -  integer scaling
- `TextBlock.WrapWidth`  -  word wrapping
- `TextBlock.Align`  -  left, center, right alignment
- `TextBlock.Color`  -  fill color (primary text color)
- `Node.Color`  -  tint multiplier on top of fill (same as sprites)

### What Does Not Apply

- `TextBlock.TextEffects`  -  outline, glow, shadow (SpriteFont only)

## Next Steps

- [Tilemap Viewport](?page=tilemap-viewport)  -  tile-based map rendering
- [Polygons](?page=polygons)  -  point-list polygon shapes

## Related

- [Font Generation (fontgen)](?page=fontgen)  -  generate SDF font atlases offline from TTF/OTF files
- [Atlases](?page=atlases)  -  atlas loading and page registration
- [Nodes](?page=nodes)  -  node types and visual properties
