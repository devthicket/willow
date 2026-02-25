# Atlases

An atlas packs many small images onto shared page textures so they can be rendered efficiently. Sprites that share the same atlas page are batched into a single `DrawTriangles32` submission to the GPU.

## Why Use an Atlas?

Willow batches sprites by atlas page. All sprites on the same page with the same blend mode and shader are submitted in one GPU call. Without an atlas, each image becomes its own page, which means its own submission. With many sprites, that adds up.

Pre-packing sprites into an atlas at build time gives you the tightest packing and the best batching with zero runtime cost.

## Atlas

An `Atlas` holds one or more page images and a name-to-region map:

```go
type Atlas struct {
    Pages []*ebiten.Image
}
```

### Loading an Atlas

Willow supports the [TexturePacker](https://www.codeandweb.com/texturepacker) JSON format (both hash and array variants):

```go
jsonData, _ := os.ReadFile("sprites.json")
pageImg, _, _ := ebitenutil.NewImageFromFile("sprites.png")

atlas, err := willow.LoadAtlas(jsonData, []*ebiten.Image{pageImg})
if err != nil {
    log.Fatal(err)
}
```

Or through a scene (which also registers pages with the global atlas manager):

```go
atlas, err := scene.LoadAtlas(jsonData, []*ebiten.Image{pageImg})
```

### Getting Regions

```go
region := atlas.Region("player_idle")
sprite := willow.NewSprite("player", region)
```

If a name isn't found, `Region()` returns a magenta placeholder and logs a warning (in debug mode), so missing sprites are immediately visible.

## TexturePacker Workflow

1. Add your source sprites to TexturePacker
2. Export as **JSON (Hash)** or **JSON (Array)** format
3. Load the JSON and page image(s) at startup
4. Use `atlas.Region("name")` to get regions for sprites

TexturePacker features supported:
- **Trimming**  -  `OffsetX`/`OffsetY` and `OriginalW`/`OriginalH` handle trimmed whitespace
- **Rotation**  -  90-degree CW rotation is handled automatically
- **Multi-page**  -  pass multiple page images to `LoadAtlas`

## Multi-Page Atlases

```go
atlas, err := willow.LoadAtlas(jsonData, []*ebiten.Image{
    page0Img,
    page1Img,
    page2Img,
})
```

Each `TextureRegion` stores its `Page` index, so sprites from different pages render correctly. Sprites on different pages are separate batches.

## Atlas Page Management

All atlas pages are owned by a global `AtlasManager` shared across every scene. You never call the manager directly  -  it works behind the scenes whenever you load an atlas, create a font, or register a page.

```
                  ┌──────────────────────────┐
                  │   Global AtlasManager    │
                  │                          │
                  │  pages[]   refs[]        │
                  │  static[]  nextPage      │
                  └──────────────────────────┘
                       ▲     ▲      ▲
            ┌──────────┘     │      └──────────┐
            │                │                 │
     scene.LoadAtlas    NewFontFromTTF    NewAtlas/Add
     scene.RegisterPage LoadSpriteFont   NewBatchAtlas/Pack
```

Key properties:

- **Pages are global**  -  shared across all scenes. A font loaded once is available everywhere.
- **Reference-counted**  -  `Retain`/`Release` track how many resources use each page. When the count reaches zero, `Cleanup()` can deallocate the page.
- **Static pages persist**  -  pages from `LoadAtlas` and fonts are marked static and are never deallocated by cleanup.
- **Dynamic pages can be cleaned up**  -  pages from `NewAtlas`/`NewBatchAtlas` use reference counting and can be reclaimed when no longer needed.

Developers interact with pages through `scene.LoadAtlas()`, `scene.RegisterPage()`, `NewFontFromTTF()`, and `NewAtlas()`  -  the manager handles the rest automatically.

## Choosing an Approach

| Approach | Batching | VRAM Overhead | Setup | Best For |
|----------|----------|--------------|-------|----------|
| `LoadAtlas` (pre-packed) | Best  -  one page per atlas | Minimal  -  tightly packed offline | Requires build step | Main game assets |
| `NewBatchAtlas` + `Pack` | Good  -  optimal runtime packing | Page-sized allocation | Stage all, pack once | Level init, known set of runtime images |
| `NewAtlas` + `Add` | Good  -  shared runtime pages | Page-sized allocation | None  -  pack at load time | Dynamic content, runtime updates |
| `RegisterPage` per image | Poor  -  one batch per image | Minimal  -  image-sized | None  -  simplest | Tilemaps, bitmap fonts |
| `SetCustomImage` | Poor  -  breaks coalesced batching | Minimal  -  image-sized | None  -  simplest | A few one-off images |

For most games, `LoadAtlas` with TexturePacker or [atlaspack](?page=atlaspack) output covers your needs. Reach for dynamic packing only when you have many runtime-loaded images.

## Next Steps

- [Dynamic Atlas](?page=dynamic-atlas)  -  runtime atlas packing for lazy asset loading
- [Sprite Packing (atlaspack)](?page=atlaspack)  -  pack PNG sprites into an atlas without external tools

## Related

- [Sprites](?page=sprites)  -  creating sprites and TextureRegion basics
- [Scene](?page=scene)  -  atlas loading via `scene.LoadAtlas()`
- [Text & Fonts](?page=text-and-fonts)  -  font atlases use the same page system
