# CacheAsTree

CacheAsTree caches the **render command list** for a subtree. When enabled, scene graph traversal and command emission are skipped  -  the pre-built command list is replayed directly. Use this for subtrees whose structure and visual properties don't change frequently but still need per-node transforms applied.

## Enabling

```go
node.SetCacheAsTree(true)
```

## Cache Modes

```go
// Auto (default): property setters on any descendant automatically invalidate
node.SetCacheAsTree(true, willow.CacheTreeAuto)

// Manual: you must call InvalidateCacheTree() when content changes
node.SetCacheAsTree(true, willow.CacheTreeManual)
```

In auto mode, calling `SetPosition()`, `SetColor()`, `SetVisible()`, or any property setter on a descendant node automatically invalidates ancestor caches. Direct field assignment (e.g., `node.X = 10`) does **not** trigger auto-invalidation  -  call `Invalidate()` or `InvalidateCacheTree()` manually.

## Manual Invalidation

```go
node.InvalidateCacheTree()
```

Call this after any change that should trigger a re-traversal of the cached subtree.

## Querying

```go
enabled := node.IsCacheAsTreeEnabled()
```

## Example: Semi-Static UI

```go
panel := willow.NewContainer("inventory")
// ... add inventory slots, icons, labels ...
panel.SetCacheAsTree(true)  // auto mode

// When the inventory changes:
panel.InvalidateCacheTree()
```

## When to Use CacheAsTree

| Scenario | Recommendation |
|----------|---------------|
| Subtree structure is stable but nodes still move/transform | CacheAsTree |
| Content changes every frame | Don't cache  -  overhead outweighs savings |
| Fully static content that never changes | [CacheAsTexture](?page=cache-as-texture) is more efficient |

## Next Steps

- [CacheAsTexture](?page=cache-as-texture)  -  render subtrees to an offscreen image for maximum performance

## Related

- [Performance](?page=performance-overview)  -  benchmarks, batching, and optimization strategies
- [Transforms](?page=transforms)  -  setter methods vs direct assignment and dirty flags
- [Architecture](?page=architecture)  -  how caching fits into the render pipeline
