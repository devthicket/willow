# Performance Caching

Willow provides two caching strategies to avoid redundant work for static or semi-static subtrees.

## CacheAsTree (Command List Cache)

Caches the **render command list** for a subtree. The scene graph traversal is skipped for cached subtrees  -  the pre-built command list is replayed directly. Best for subtrees whose structure and visual properties don't change frequently.

```go
node.SetCacheAsTree(true)
```

### Cache Tree Modes

```go
// Manual: you must call InvalidateCacheTree() when content changes
node.SetCacheAsTree(true, willow.CacheTreeManual)

// Auto (default): property setters on any descendant automatically invalidate
node.SetCacheAsTree(true, willow.CacheTreeAuto)
```

In auto mode, calling `SetPosition()`, `SetColor()`, `SetVisible()`, or any property setter on a descendant node automatically invalidates ancestor caches.

### Manual Invalidation

```go
node.InvalidateCacheTree()
```

Call this after any change that should trigger a re-traversal of the cached subtree.

### Querying

```go
enabled := node.IsCacheAsTreeEnabled()
```

## CacheAsTexture (Offscreen Render Cache)

Renders the entire subtree to an **offscreen image** once, then reuses that image each frame. Best for complex subtrees that rarely change (e.g., backgrounds, static UI panels).

```go
node.SetCacheAsTexture(true)
```

### Invalidation

When the cached content changes, call:

```go
node.InvalidateCache()
```

This triggers a re-render of the subtree to the offscreen image on the next frame.

### Querying

```go
enabled := node.IsCacheEnabled()
```

## One-Shot Render to Texture

Render a subtree to a new `*ebiten.Image` (caller owns the result):

```go
img := node.ToTexture(scene)
```

This is a one-time capture, not a live cache.

## When to Use Each

| Strategy | Best For | Cost | Updates |
|----------|----------|------|---------|
| **CacheAsTree** | Subtrees with stable structure but that still need per-node transforms applied | Saves traversal and command emission | Free in auto mode; manual for field assignments |
| **CacheAsTexture** | Fully static visual content (backgrounds, decorations) | Saves traversal, emission, *and* per-node vertex submissions | Must invalidate explicitly |
| **Neither** | Dynamic content that changes every frame | No overhead from cache management | N/A |

## Example: Static Background

```go
bg := willow.NewContainer("background")
for x := 0; x < 20; x++ {
    for y := 0; y < 15; y++ {
        tile := willow.NewSprite("tile", atlas.Region("grass"))
        tile.SetPosition(float64(x*32), float64(y*32))
        bg.AddChild(tile)
    }
}
bg.SetCacheAsTexture(true)
scene.Root().AddChild(bg)
```

The 300 tile sprites are rendered once to an offscreen image and reused every frame.

## Example: Semi-Static UI

```go
panel := willow.NewContainer("inventory")
// ... add inventory slots, icons, labels ...
panel.SetCacheAsTree(true)  // auto mode

// When the inventory changes:
panel.InvalidateCacheTree()
```
