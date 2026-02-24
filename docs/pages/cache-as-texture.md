# CacheAsTexture

CacheAsTexture renders an entire subtree to an **offscreen image** once, then reuses that image each frame. This is the most aggressive caching strategy  -  it skips traversal, command emission, *and* per-node vertex submissions. Use it for complex subtrees that rarely change, like backgrounds or static UI panels.

## Enabling

```go
node.SetCacheAsTexture(true)
```

## Invalidation

When the cached content changes, trigger a re-render:

```go
node.InvalidateCache()
```

This re-renders the subtree to the offscreen image on the next frame.

## Querying

```go
enabled := node.IsCacheEnabled()
```

## One-Shot Render to Texture

Render a subtree to a new `*ebiten.Image` without enabling live caching (caller owns the result):

```go
img := node.ToTexture(scene)
```

This is a one-time capture, not a live cache.

## Example: Static Background

```go
bg := willow.NewContainer("background")
for x := 0; x < 20; x++ {
    for y := 0; y < 15; y++ {
        tile := willow.NewSprite("tile", atlas.Region("grass"))
        tile.X = float64(x * 32)
        tile.Y = float64(y * 32)
        bg.AddChild(tile)
    }
}
bg.SetCacheAsTexture(true)
scene.Root().AddChild(bg)
```

The 300 tile sprites are rendered once to an offscreen image and reused every frame.

## When to Use CacheAsTexture

| Scenario | Recommendation |
|----------|---------------|
| Fully static visual content (backgrounds, decorations) | CacheAsTexture |
| Content with filters applied | CacheAsTexture avoids re-applying filters every frame |
| Subtree that still needs per-node transforms | [CacheAsTree](?page=cache-as-tree) instead |
| Dynamic content that changes every frame | Don't cache |

## Next Steps

- [Debug & Testing](?page=debug-and-testing)  -  debug overlays and visual testing tools
- [ECS Integration](?page=ecs-integration)  -  entity component system bridge

## Related

- [Performance](?page=performance-overview)  -  benchmarks, batching, and optimization strategies
- [CacheAsTree](?page=cache-as-tree)  -  lighter caching that preserves per-node transforms
- [Offscreen Rendering](?page=offscreen-rendering)  -  manual render targets
- [Post-Processing Filters](?page=post-processing-filters)  -  combine with CacheAsTexture for static filtered content
