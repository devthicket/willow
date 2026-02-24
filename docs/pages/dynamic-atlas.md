# Dynamic Atlas (Runtime Packing)

Optional tool for packing images onto shared atlas pages at runtime, so they batch together in a single `DrawTriangles32` submission.

## When You Need This

Willow batches sprites by atlas page. Sprites on the same page with the same blend mode and shader are drawn in one GPU submission. When you load images at runtime  -  user avatars, procedural sprites, downloaded content  -  each one normally becomes its own page, which means its own submission. With many such images, that adds up.

Two modes are available:

- **Shelf mode (streaming):** `NewAtlas` + `Add`. Add, free, and replace images over time. Space is reused when freed. Best for dynamic content that changes at runtime.
- **Batch mode (rect pack):** `NewBatchAtlas` + `Stage` + `Pack`. Stage all images, then finalize with optimal MaxRects packing. No additions after packing. Best when all images are known upfront.

## When You Do NOT Need This

**For a few images (1-5):** use `SetCustomImage` or `RegisterPage`. The batching overhead of a few extra submissions is negligible, and you avoid allocating a large atlas page.

```go
// Simple approach for one-off images:
sprite := willow.NewSprite("avatar", willow.TextureRegion{})
sprite.SetCustomImage(avatarImg)
```

**For pre-packed atlases:** use `LoadAtlas` with TexturePacker or atlaspack output. Static packing at build time is always more efficient than runtime packing.

**Ebitengine already has internal atlasing.** Small `ebiten.Image` instances are automatically packed onto internal atlas backends. Willow's dynamic atlas operates at a higher level  -  it controls Willow's batch key, not Ebitengine's internal texture layout. If you have few runtime images and don't see batching problems in your profiling, you don't need this.

## Mode 1: Shelf (Streaming)

Shelf mode adds images immediately using a shelf-based bin packing algorithm. Images can be freed, replaced, and updated over time.

### Creating

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

### Freeing Images

```go
err := atlas.Free("player_skin")
```

`Free` removes the region from the lookup and reclaims the pixel space for reuse. Pixels are cleared on the atlas page to prevent stale bleeding. Future `Add` calls may reuse the freed space.

### Replacing Images (Same Size)

```go
err := atlas.Replace("player_skin", newSkinImg)
```

`Replace` swaps pixels in-place without changing the TextureRegion. The new image must be exactly the same size as the existing region. Use this for texture updates that don't change dimensions.

### Updating Images (Any Size)

```go
region, err := atlas.Update("player_skin", differentSizeImg)
```

`Update` is a convenience method: if the new image is the same size, it acts like `Replace`. If the size differs, it frees the old region and adds the new one, returning the (potentially different) TextureRegion.

### Querying

```go
atlas.Has("player_skin")    // true if already added
atlas.RegionCount()          // number of named regions
atlas.Region("player_skin") // returns TextureRegion (magenta placeholder if missing)
```

### Extending a Static Atlas

You can dynamically add images to an atlas created by `LoadAtlas`. New images are packed onto new pages beyond the static ones:

```go
atlas, _ := scene.LoadAtlas(jsonData, []*ebiten.Image{pageImg})
// ... later at runtime:
atlas.Add("dlc_hat", hatImg)
```

## Mode 2: Batch (Rect Pack)

Batch mode stages all images first, then packs them in one shot using the MaxRects algorithm. This produces better packing efficiency since all items are known upfront. The atlas is immutable after packing.

### Creating and Staging

```go
atlas := willow.NewBatchAtlas(willow.PackerConfig{PageWidth: 1024, PageHeight: 1024})

atlas.Stage("a", imgA)   // buffers image (no packing yet)
atlas.Stage("b", imgB)
atlas.Stage("c", imgC)
```

### Packing

```go
err := atlas.Pack()  // runs MaxRects, copies pixels, sets regions
```

After `Pack`:
- `Region("a")` returns valid TextureRegions
- `Stage()` returns an error (atlas is immutable)
- `Pack()` again returns an error

### When to Use Batch vs Shelf

| | Shelf (`NewAtlas`) | Batch (`NewBatchAtlas`) |
|---|---|---|
| Images known upfront | No  -  add over time | Yes  -  all staged before Pack |
| Packing efficiency | Good (85-90%) | Better (MaxRects optimal) |
| Post-creation changes | Add, Free, Replace, Update | None (immutable after Pack) |
| Use case | Dynamic content, streaming | Asset loading, level init |

## Page Size Guidance

Each atlas page allocates `width * height * 4` bytes of VRAM:

| Page Size | VRAM per Page | Good For |
|-----------|--------------|----------|
| 256x256 | 256 KB | A handful of small sprites (icons, UI elements) |
| 512x512 | 1 MB | Moderate set of medium sprites |
| 1024x1024 | 4 MB | Large set of varied sprites |
| 2048x2048 | 16 MB | Many sprites that must all batch together |

Page dimensions should be powers of two (256, 512, 1024, 2048). GPUs are optimized for power-of-two textures  -  non-power-of-two sizes may be padded internally, wasting VRAM and hurting cache performance.

**Rule of thumb:** choose the smallest power-of-two page size that fits your expected image set. If you're adding 8 small icons, a 256x256 page wastes far less VRAM than the 2048x2048 default.

```go
// For a small set of UI icons:
atlas := willow.NewAtlas(willow.PackerConfig{
    PageWidth:  256,
    PageHeight: 256,
})
```

When a page fills up, the packer automatically allocates a new one. Images on different pages are separate batches.

## How It Works

### Shelf Mode

The packer uses a shelf algorithm  -  each page is divided into horizontal strips. Items are placed left-to-right on the shelf whose height best matches the item, minimizing wasted vertical space. When no shelf fits, a new one is opened. When the page is full, a new page is allocated. Freed regions become free slots that are checked first on subsequent `Add` calls.

### Batch Mode

The packer uses the MaxRects algorithm with Best Short Side Fit (BSSF) heuristic. Images are sorted largest-area-first for optimal placement. Each image is tried on existing pages before a new page is allocated.

### Pixel Copying

Pixels are copied onto the atlas page via `DrawImage` at `Add`/`Pack` time (one-time cost, not per frame). After that, the source image is no longer referenced.

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
| `NewBatchAtlas` + `Pack` | Good  -  optimal runtime packing | Page-sized allocation | Stage all, pack once |
| `NewAtlas` + `Add` | Good  -  shared runtime pages | Page-sized allocation | None  -  pack at load time |
| `RegisterPage` per image | Poor  -  one batch per image | Minimal  -  image-sized | None  -  simplest |
| `SetCustomImage` | Poor  -  breaks coalesced batching | Minimal  -  image-sized | None  -  simplest |

## Next Steps

- [Camera & Viewport](?page=camera-and-viewport)  -  viewport setup, follow, zoom, and culling
- [Text & Fonts](?page=text-and-fonts)  -  bitmap and TTF text rendering

## Related

- [Sprites & Atlas](?page=sprites-and-atlas)  -  static atlas loading and TextureRegion basics
- [Sprite Packing (atlaspack)](?page=atlaspack)  -  build-time atlas packing tool
- [Offscreen Rendering](?page=offscreen-rendering)  -  RenderTexture for procedural content
