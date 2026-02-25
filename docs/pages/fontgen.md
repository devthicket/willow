# fontgen

`fontgen` is a CLI tool that converts a TTF/OTF font into an SDF (Signed Distance Field) atlas for use with Willow's text renderer. SDF atlases store distance-from-edge data instead of raw pixels, enabling sharp text at any scale and cheap shader-based effects like outlines, glows, and drop shadows.

## Installation

```bash
go install github.com/phanxgames/willow/cmd/fontgen@latest
```

Or run directly from the repo:

```bash
go run ./cmd/fontgen [flags]
```

## Usage

```bash
fontgen -font input.ttf -size 80 -range 8 -out myfont
```

Produces `myfont.png` (atlas image) and `myfont.json` (glyph metrics).

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-font` | *(required)* | Path to TTF/OTF font file |
| `-size` | `80` | Rasterization size in pixels. Larger values capture more detail for high-resolution scaling. |
| `-range` | `8` | SDF spread in pixels. Controls how far the distance field extends beyond glyph edges — larger values allow thicker outlines and shadows but need more atlas space. |
| `-out` | `"output"` | Output file prefix — produces `<out>.png` + `<out>.json` |
| `-chars` | printable ASCII | Characters to include. Defaults to `0x20`–`0x7E`. Pass a string to override. |
| `-width` | `1024` | Maximum atlas width in pixels |

## Output

The tool produces two files:

- **`<out>.png`** — grayscale atlas image where each pixel encodes distance from the nearest glyph edge
- **`<out>.json`** — glyph metrics (advance widths, offsets, atlas coordinates) consumed by `willow.LoadSpriteFont`

## Loading in Willow

```go
metricsJSON, _ := os.ReadFile("myfont.json")
atlasImg, _, _ := ebitenutil.NewImageFromFile("myfont.png")

font, err := willow.LoadSpriteFont(metricsJSON, 0)  // page index 0
scene.RegisterPage(0, atlasImg)

label := willow.NewText("title", "Hello!", font)
label.TextBlock.FontSize = 24
scene.Root().AddChild(label)
```

For convenience, `NewFontFromTTF` does the same thing at runtime without the CLI tool — but offline generation is faster at startup and keeps the atlas out of the binary.

## Choosing Parameters

**`-size`**: The rasterization resolution. A size of 80 works well for most body text. Use 120+ for fonts that will be displayed very large or need fine detail at high zoom.

**`-range`**: The distance field spread. A range of 8 is a good default. Increase to 12–16 if you plan to use thick outlines or wide drop shadows, since the effect radius is limited by this value.

**`-chars`**: By default, all printable ASCII characters are included. For games with localized text, pass the full character set your game uses:

```bash
fontgen -font game.ttf -chars "ABCDEFabcdef0123456789!?,." -out game_font
```

## Next Steps

- [LLM Guide](?page=llm) — quick reference for AI code generators working with Willow

## Related

- [Text & Fonts](?page=text-and-fonts) — using fonts in Willow, text effects, alignment, and wrapping
- [Atlases](?page=atlases) — atlas loading and page registration
- [Sprite Packing (atlaspack)](?page=atlaspack) — pack sprite images into a texture atlas
