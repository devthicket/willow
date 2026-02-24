# Dynamic Atlas (Runtime Packing)

Optional tool for packing images onto shared atlas pages at runtime, so they batch together in a single `DrawTriangles32` submission.

## When You Need This

Willow batches sprites by atlas page. Sprites on the same page with the same blend mode and shader are drawn in one GPU submission. When you load images at runtime  -  user avatars, procedural sprites, downloaded content  -  each one normally becomes its own page, which means its own submission. With many such images, that adds up.

`NewAtlas` + `Add` packs runtime images onto shared pages using shelf-based bin packing. All images on the same page are one batch.

## When You Do NOT Need This

**For a few images (1-5):** use `SetCustomImage` or `RegisterPage`. The batching overhead of a few extra submissions is negligible, and you avoid allocating a large atlas page.

```go
// Simple approach for one-off images:
sprite := willow.NewSprite("avatar", willow.TextureRegion{})
sprite.SetCustomImage(avatarImg)
```

**For pre-packed atlases:** use `LoadAtlas` with TexturePacker or atlaspack output. Static packing at build time is always more efficient than runtime packing.

**Ebitengine already has internal atlasing.** Small `ebiten.Image` instances are automatically packed onto internal atlas backends. Willow's dynamic atlas operates at a higher level  -  it controls Willow's batch key, not Ebitengine's internal texture layout. If you have few runtime images and don't see batching problems in your profiling, you don't need this.

## API

### Creating a Dynamic Atlas

```go
atlas := willow.NewAtlas()  // 2048x2048 pages, 1px padding
```

With custom page size:

```go
atlas := willow.NewAtlas(willow.PackerConfig{
    PageWidth:  512,
    PageHeight: 512,
    Padding:    2,
})
```

### Adding Images

```go
region, err := atlas.Add("player_skin", skinImg)
if err != nil {
    log.Fatal(err)
}
sprite := willow.NewSprite("player", region)
```

`Add` is idempotent  -  calling it twice with the same name returns the existing region without re-packing.

### Querying

```go
atlas.Has("player_skin")  // true if already added
atlas.RegionCount()        // number of named regions
atlas.Region("player_skin") // returns TextureRegion (magenta placeholder if missing)
```

### Extending a Static Atlas

You can dynamically add images to an atlas created by `LoadAtlas`. New images are packed onto new pages beyond the static ones:

```go
atlas, _ := scene.LoadAtlas(jsonData, []*ebiten.Image{pageImg})
// ... later at runtime:
atlas.Add("dlc_hat", hatImg)
```

### Removing

```go
atlas.Remove("player_skin")
```

Remove deletes the name from the lookup map but does **not** reclaim the pixel space on the atlas page. The dead pixels remain until the page is deallocated.

## Page Size Guidance

Each atlas page allocates `width * height * 4` bytes of VRAM:

| Page Size | VRAM per Page | Good For |
|-----------|--------------|----------|
| 256x256 | 256 KB | A handful of small sprites (icons, UI elements) |
| 512x512 | 1 MB | Moderate set of medium sprites |
| 1024x1024 | 4 MB | Large set of varied sprites |
| 2048x2048 | 16 MB | Many sprites that must all batch together |

**Rule of thumb:** choose the smallest page size that fits your expected image set. If you're adding 8 small icons, a 256x256 page wastes far less VRAM than the 2048x2048 default.

```go
// For a small set of UI icons:
atlas := willow.NewAtlas(willow.PackerConfig{
    PageWidth:  256,
    PageHeight: 256,
})
```

When a page fills up, the packer automatically allocates a new one. Images on different pages are separate batches.

## How It Works

The packer uses a shelf algorithm  -  each page is divided into horizontal strips. Items are placed left-to-right on the shelf whose height best matches the item, minimizing wasted vertical space. When no shelf fits, a new one is opened. When the page is full, a new page is allocated.

Pixels are copied onto the atlas page via `DrawImage` at `Add` time (one-time cost, not per frame). After `Add`, the source image is no longer referenced.

### Padding

Padding inserts a pixel gap between packed images to prevent texture bleeding when the GPU samples between adjacent sprites. The default is 1 pixel. Set to 0 only if you're certain your sprites won't have filtering artifacts:

```go
// Disable padding (edge-to-edge packing):
cfg := willow.PackerConfig{PageWidth: 512, PageHeight: 512}.NoPadding()
atlas := willow.NewAtlas(cfg)
```

## Comparison with Other Approaches

| Approach | Batching | VRAM Overhead | Setup |
|----------|----------|--------------|-------|
| `LoadAtlas` (pre-packed) | Best  -  one page per atlas | Minimal  -  tightly packed offline | Requires build step |
| `NewAtlas` + `Add` | Good  -  shared runtime pages | Page-sized allocation | None  -  pack at load time |
| `RegisterPage` per image | Poor  -  one batch per image | Minimal  -  image-sized | None  -  simplest |
| `SetCustomImage` | Poor  -  breaks coalesced batching | Minimal  -  image-sized | None  -  simplest |

## Next Steps

- [Performance](?page=performance-overview)  -  batching details and optimization strategies
- [CacheAsTree](?page=cache-as-tree)  -  command list caching for semi-static subtrees

## Related

- [Sprites & Atlas](?page=sprites-and-atlas)  -  static atlas loading and TextureRegion basics
- [Sprite Packing (atlaspack)](?page=atlaspack)  -  build-time atlas packing tool
- [Offscreen Rendering](?page=offscreen-rendering)  -  RenderTexture for procedural content
