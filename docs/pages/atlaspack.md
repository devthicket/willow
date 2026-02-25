# atlaspack

`atlaspack` is a CLI tool that packs sprite PNGs into a Willow-compatible `.png` + `.json` atlas. It replaces external tools like TexturePacker for projects that want a self-contained build pipeline.

## Installation

```bash
go install github.com/phanxgames/willow/cmd/atlaspack@latest
```

Or run directly from the repo:

```bash
go run ./cmd/atlaspack [flags] <paths...>
```

## Usage

```bash
atlaspack [flags] <paths...>
```

Positional args are auto-detected:

- **Directory** — recursive scan for `.png` files
- **Contains `*`** — glob expansion
- **Ends in `.json`** — parsed as manifest
- **Ends in `.png`** — explicit file

Multiple types can be mixed:

```bash
atlaspack -out atlas sprites/ extra/boss.png overlay/*.png
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-out` | `"atlas"` | Output prefix — produces `<out>.png` + `<out>.json` |
| `-width` | `1024` | Atlas width in pixels |
| `-height` | `1024` | Atlas height in pixels |
| `-padding` | `1` | Pixels between sprites |
| `-no-trim` | `false` | Disable transparent whitespace trimming |
| `-no-rotate` | `false` | Disable 90° CW rotation for tighter packing |

## Sprite Naming

Names are derived from the file path relative to the input source, extension stripped, forward slashes normalized:

| Input | Sprite name |
|-------|-------------|
| `sprites/` containing `enemies/boss.png` | `enemies/boss` |
| Explicit `boss.png` | `boss` |
| Glob `overlay/*.png` matching `overlay/hud.png` | `hud` |

Duplicate names across inputs are an error (both paths are shown in the message).

## Output

The tool produces two files:

- **`<out>.png`** — the packed atlas image (NRGBA)
- **`<out>.json`** — TexturePacker hash-format JSON matching Willow's `LoadAtlas` schema

```json
{
  "frames": {
    "enemies/boss": {
      "frame": {"x": 0, "y": 0, "w": 64, "h": 64},
      "rotated": false,
      "trimmed": true,
      "spriteSourceSize": {"x": 2, "y": 3, "w": 60, "h": 58},
      "sourceSize": {"w": 64, "h": 64}
    }
  },
  "meta": {
    "image": "atlas.png",
    "size": {"w": 1024, "h": 1024}
  }
}
```

Load the output in Willow:

```go
jsonData, _ := os.ReadFile("atlas.json")
pageImg, _, _ := ebitenutil.NewImageFromFile("atlas.png")
atlas, err := scene.LoadAtlas(jsonData, []*ebiten.Image{pageImg})

sprite := willow.NewSprite("boss", atlas.Region("enemies/boss"))
```

## Manifest

A `.json` manifest encodes flags and paths in a single file, useful for repeatable builds:

```json
{
  "out": "atlas",
  "width": 2048,
  "height": 2048,
  "padding": 2,
  "trim": true,
  "rotate": true,
  "paths": [
    "sprites/",
    "extra/boss.png",
    "overlay/*.png"
  ]
}
```

Each path entry gets the same auto-detection as CLI args. CLI flags override manifest values when both are provided.

```bash
atlaspack build.json
atlaspack -padding 4 build.json extra_sprites/
```

## Packing Algorithm

`atlaspack` uses the **MaxRects Best Short Side Fit (BSSF)** algorithm:

1. Sprites are sorted by area (largest first)
2. Each sprite is placed in the free rectangle that leaves the smallest short-side remainder
3. When rotation is enabled, both orientations are tried and the tighter fit wins
4. Free rectangles are split and pruned after each placement

Trimming removes transparent whitespace before packing, so sprites occupy only the pixels they need. Trim offsets are preserved in the JSON so Willow restores the original sprite bounds at render time.

## Next Steps

- [Font Generation (fontgen)](?page=fontgen) — generate SDF font atlases from TTF/OTF files

## Related

- [Atlases](?page=atlases) — loading atlases and using regions in Willow
- [Text & Fonts](?page=text-and-fonts) — font atlas loading (uses the same page registration)
