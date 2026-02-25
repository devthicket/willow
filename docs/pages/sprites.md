# Sprites

Sprites are the primary way to display images in Willow. Create a sprite from a `TextureRegion`, attach it to the scene graph, and Willow handles rendering.

## Creating a Sprite

```go
region := atlas.Region("player_idle")
sprite := willow.NewSprite("player", region)
sprite.X = 100
sprite.Y = 200
scene.Root().AddChild(sprite)
```

`NewSprite` takes a name and a `TextureRegion`. The region tells Willow which rectangle of which atlas page to draw.

## TextureRegion

A `TextureRegion` describes a rectangular sub-image within an atlas page:

```go
type TextureRegion struct {
    Page      uint16  // atlas page index
    X, Y      uint16  // top-left corner in atlas page
    Width     uint16  // sub-image width (may differ from OriginalW if trimmed)
    Height    uint16  // sub-image height
    OriginalW uint16  // untrimmed width as authored
    OriginalH uint16  // untrimmed height as authored
    OffsetX   int16   // horizontal trim offset
    OffsetY   int16   // vertical trim offset
    Rotated   bool    // true if stored 90 degrees CW in atlas
}
```

An empty `TextureRegion{}` signals "use the 1x1 WhitePixel", which is the standard way to create [solid-color shapes](?page=solid-color-sprites).

## Standalone Images

For a few one-off images that aren't in a pre-packed atlas, two approaches keep things simple:

```go
// Option A: SetCustomImage (simplest for one-offs)
sprite := willow.NewSprite("preview", willow.TextureRegion{})
sprite.SetCustomImage(img)

// Option B: RegisterPage + TextureRegion (for bitmap fonts, tilemaps)
scene.RegisterPage(0, pageImage)
```

`SetCustomImage` is the fastest path for a handful of images. Each custom image becomes its own batch entry, which is fine for small counts. For many runtime images (10+), pack them onto shared pages with a [dynamic atlas](?page=dynamic-atlas) instead.

## Next Steps

- [Atlases](?page=atlases)  -  organize many textures for efficient rendering
- [Camera & Viewport](?page=camera-and-viewport)  -  viewport setup, follow, zoom, and culling

## Related

- [Solid-Color Sprites](?page=solid-color-sprites)  -  shapes without textures using `WhitePixel`
- [Nodes](?page=nodes)  -  node types and visual properties
