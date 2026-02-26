# Tilemap Viewport

<p align="center">
  <img src="gif/tilemap.gif" alt="Tilemap demo" width="400">
</p>

Willow provides `TileMapViewport` for efficient rendering of large tile-based maps. Only visible tiles are rendered, with buffering for smooth camera scrolling.

## TileMapViewport

```go
viewport := willow.NewTileMapViewport("map", 32, 32)  // tile size in pixels
scene.Root().AddChild(viewport.Node())
```

### Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `TileWidth` | `int` | *(set at creation)* | Tile width in pixels |
| `TileHeight` | `int` | *(set at creation)* | Tile height in pixels |
| `MaxZoomOut` | `float64` | `1.0` | Minimum expected zoom level; controls buffer sizing |
| `MarginTiles` | `int` | `2` | Extra tiles buffered beyond viewport edge |

## Binding a Camera

The tilemap viewport needs a camera to determine which tiles are visible:

```go
cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: 800, Height: 600})
viewport.SetCamera(cam)
```

## Adding Tile Layers

```go
// Tile data: flat array of GIDs, row-major order
// GID 0 = empty tile
tileData := []uint32{
    1, 2, 1, 2, 1,
    3, 0, 0, 0, 3,
    1, 2, 1, 2, 1,
}

// TextureRegion per GID (index 0 unused, GIDs start at 1)
regions := []willow.TextureRegion{
    {},                       // GID 0 (empty, unused)
    atlas.Region("grass"),    // GID 1
    atlas.Region("dirt"),     // GID 2
    atlas.Region("stone"),    // GID 3
}

layer := viewport.AddTileLayer(willow.TileLayerConfig{
    Name: "ground", Width: 5, Height: 3,
    Data: tileData, Regions: regions, AtlasImage: atlasPage,
})
```

### GID Format

GIDs follow the Tiled TMX convention:
- Bits 31, 30, 29: flip flags (horizontal, vertical, diagonal)
- Remaining bits: tile ID
- GID 0 = empty tile (not rendered)

## Modifying Tiles at Runtime

```go
layer.SetTile(col, row, newGID)           // change a single tile
layer.SetData(newData, newWidth, newHeight) // replace all tile data
layer.InvalidateBuffer()                    // force full redraw
```

## Animated Tiles

Define animation sequences for specific GIDs:

```go
anims := map[uint32][]willow.AnimFrame{
    4: {  // GID 4 animates through these frames
        {GID: 4, Duration: 200},  // 200ms
        {GID: 5, Duration: 200},
        {GID: 6, Duration: 200},
    },
}
layer.SetAnimations(anims)
```

`AnimFrame.Duration` is in milliseconds. The animation loops continuously.

## Sandwich Layers (Entity Layers)

Insert scene graph nodes between tile layers for depth-sorted entities:

```go
// Background tiles
viewport.AddTileLayer(willow.TileLayerConfig{
    Name: "bg", Width: w, Height: h,
    Data: bgData, Regions: regions, AtlasImage: page,
})

// Entity layer (scene graph nodes rendered between tile layers)
entityContainer := willow.NewContainer("entities")
viewport.AddChild(entityContainer)

// Foreground tiles
viewport.AddTileLayer(willow.TileLayerConfig{
    Name: "fg", Width: w, Height: h,
    Data: fgData, Regions: regions, AtlasImage: page,
})
```

`AddChild` interleaves scene graph nodes with tile layers in draw order.

## Camera Scrolling with Tilemap

```go
cam := scene.NewCamera(willow.Rect{X: 0, Y: 0, Width: 800, Height: 600})
viewport.SetCamera(cam)

// Scroll to a tile coordinate
cam.ScrollToTile(10, 5, 32, 32, willow.TweenConfig{Duration: 0.5, Ease: ease.InOutQuad})

// Set camera bounds to map size
cam.SetBounds(willow.Rect{
    X: 0, Y: 0,
    Width:  float64(mapWidth * 32),
    Height: float64(mapHeight * 32),
})
```

## Zoom Support

Set `MaxZoomOut` to the minimum expected zoom level to ensure enough tiles are buffered:

```go
viewport.MaxZoomOut = 0.5  // support zooming out to 50%
```

## Next Steps

- [Polygons](?page=polygons)  -  point-list polygon shapes
- [Offscreen Rendering](?page=offscreen-rendering)  -  render targets and compositing

## Related

- [Camera & Viewport](?page=camera-and-viewport)  -  camera follow, bounds, and coordinate conversion
- [Atlases](?page=atlases)  -  atlas regions used for tile GIDs
- [Tweens & Animation](?page=tweens-and-animation)  -  easing functions for camera scrolling
