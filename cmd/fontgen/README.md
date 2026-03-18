# fontgen

CLI tool that bakes TTF/OTF fonts into Signed Distance Field (SDF) atlas bundles for use with Willow.

## Usage

```sh
# Bake a system font by name
fontgen arial --sizes=64,256 -o arial.fontbundle

# Bake from a TTF file
fontgen /path/to/arial.ttf --sizes=64,256 -o arial.fontbundle

# Headless (no prompts)
fontgen arial --sizes=64,256 --charset=latin -o arial.fontbundle --auto

# Inspect bundle contents
fontgen arial.fontbundle --info

# Unpack bundle to PNG + JSON files
fontgen arial.fontbundle --extract
fontgen arial.fontbundle --extract=./output
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-o` | `<name>.fontbundle` | Output path |
| `--sizes` | `64,256` | Comma-separated bake sizes in pixels |
| `--charset` | `ascii` | `ascii`, `latin`, `full`, `"chars"`, or `file:path` |
| `--range` | `size×0.125` | Override SDF distance range for all sizes |
| `--bold` | | Path to bold TTF |
| `--italic` | | Path to italic TTF |
| `--bolditalic` | | Path to bold-italic TTF |
| `--regular` | | Alternative to positional arg for regular TTF |
| `--auto` / `-y` | | Skip all prompts |
| `--watch` | | Re-bake on TTF file change |
| `--width` | `1024` | Max atlas width in pixels |
| `--info` | | Print metadata without baking |
| `--extract[=dir]` | | Unpack `.fontbundle` to PNG + JSON |

## Bake sizes and the 64/256 split

When you bake with `--sizes=64,256` the bundle contains two SDF atlas sets — one at 64px and one at 256px. Willow picks between them at runtime using this rule:

> Use the smallest bake whose size is ≥ the current display size.

This means:
- **64-bake** is used for display sizes ≤ 64 px — small and medium text stays crisp because the SDF field is dense relative to the rendered glyphs.
- **256-bake** is used for display sizes > 64 px — large headings need the extra resolution so the SDF field doesn't get stretched.

Using only a 256-bake for everything would make small text blurry (the SDF field is too coarse relative to the output size). Using only a 64-bake for everything would make large text blurry (the atlas is scaled up). Two sizes cover the full range cleanly.

## `gofont.fontbundle` (built-in)

`assets/fonts/gofont.fontbundle` is the Go Regular font family (Regular, Bold, Italic, Bold-Italic) baked at **64 and 256** and embedded directly into the `willow` package as `willow.GofontBundle`.

To regenerate it after a fontgen change, the source TTFs live in `assets/fonts/input/`:

```sh
fontgen \
  --regular ../../assets/fonts/input/goregular.ttf \
  --bold ../../assets/fonts/input/gobold.ttf \
  --italic ../../assets/fonts/input/goitalic.ttf \
  --bolditalic ../../assets/fonts/input/gobolditalic.ttf \
  --sizes=64,256 \
  --charset=ascii \
  --width=4096 \
  --auto \
  -o ../../assets/fonts/gofont.fontbundle

# then re-extract the PNGs/JSON for inspection
fontgen --extract=../../assets/fonts/extracted/gofont ../../assets/fonts/gofont.fontbundle
```

Run from `cmd/fontgen/`.

The `--charset=ascii` and `--width=4096` combination keeps the 256px atlas under 4096px tall, which is Ebitengine's per-image limit. A latin charset at 256px with 1024px width produces atlases ~18 000px tall and will panic at runtime.
