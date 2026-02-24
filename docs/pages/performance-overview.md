# Performance

Willow is designed with performance in mind. This page explains why a retained-mode scene graph can be faster than hand-written rendering code, how Willow minimizes heap allocations per frame, and what tools are available to optimize your game.

## Why a Scene Graph Can Be Faster

A common assumption is that a scene graph adds overhead compared to calling `DrawImage` directly. In practice, Willow's coalesced batching mode can be **faster** than raw Ebitengine `DrawImage` calls  -  because owning the entire command stream unlocks optimizations that hand-written code typically doesn't do.

The key insight: **individual `DrawImage` calls each produce their own `DrawTriangles` submission internally**. Willow instead accumulates vertices across many sprites and submits them in bulk via `DrawTriangles32`, reducing command pipeline entries.

### Benchmark: Willow vs Raw Ebitengine (10K sprites)

*Tested on Apple M3 Max, Go 1.24, Ebitengine v2. Results will vary by hardware, scene complexity, and atlas layout.*

| Layer | avg ns/op | vs Raw DrawImage |
|-------|-----------|-----------------|
| Raw `DrawTriangles32` (pre-computed) | ~357K | ~6x faster |
| **Willow Coalesced** | **~1,993K** | **~11% faster** |
| Raw `DrawImage` | ~2,230K |  -  |
| Willow Immediate | ~4,121K | ~1.8x slower |

In this test (2x 4096x4096 atlas pages, runs of 1,000 sprites per page), Willow's coalesced mode measured about 11% faster than calling `DrawImage` 10,000 times. The scene graph overhead was more than offset by batching gains. Your mileage may vary depending on atlas layout and sprite count.

### Where Does This Come From?

1. **Vertex accumulation**  -  sprites sharing the same atlas page and blend mode have their vertices merged into a single `DrawTriangles32` submission
2. **Sort-based batching**  -  Willow sorts render commands by batch key (texture, blend mode, render layer), maximizing batch run lengths
3. **Allocation elimination**  -  preallocated buffers mean the hot path produces minimal garbage, reducing GC pressure

## Minimal Heap Allocations

Willow's hot path is designed to produce **near-zero heap allocations per frame** in steady state. All major buffers are preallocated and reused:

| Buffer | Purpose |
|--------|---------|
| `commands` | Render command list (grows to high-water mark, reused with `[:0]`) |
| `sortBuf` | Merge sort index buffer |
| `batchVerts` / `batchInds` | Vertex and index accumulation for `DrawTriangles32` |
| Hit-test buffers | Pointer dispatch scratch space |
| Particle pools | Fixed-size per emitter (`MaxParticles`) |

### Measured Allocation Counts

*From `go test -bench` on the same hardware as above.*

| Scenario (10K sprites) | Immediate mode | Coalesced mode |
|------------------------|---------------|---------------|
| Single atlas page | 30,000 allocs/op | **3 allocs/op** |
| Real-world atlas (2 pages) | 30,000 allocs/op | **30 allocs/op** |
| 1,000 particles | 3,000 allocs/op | **3 allocs/op** |

## Dirty Flag Transforms

World transforms are only recomputed when something actually changes. Willow tracks dirty state per node:

- Setting a transform property via a setter (`SetPosition`, `SetScale`, etc.) marks the node dirty
- Dirty state propagates to descendants during traversal
- Static subtrees pay **zero cost**  -  no matrix math, no cache invalidation

In our benchmarks, the read-only traverse optimization (skipping clean nodes) measured around a **10x speedup** on 10K sprite traversal compared to unconditional recomputation.

## Batching In Depth

Willow's default `BatchModeCoalesced` groups compatible render commands into batches. A batch breaks when any of these change between consecutive commands:

- Texture / atlas page
- Blend mode
- Shader
- Render target

### Batch Performance by Scenario

*Measured coalesced vs immediate mode on the same test hardware.*

| Scenario | Coalesced vs Immediate | Notes |
|----------|----------------------|-------|
| Single atlas page (10K sprites) | **~2x faster** | One giant batch |
| Particles (1,000) | **~5x faster** | Single texture, single blend mode |
| Real-world atlas (2 pages, runs of 1K) | **~2x faster** | 10 batch breaks, still significant gains |
| Mixed sprites + particles | **~5% faster** | Interleaving causes more breaks |
| Worst case (alternating pages every sprite) | ~5% slower | Pathological  -  unlikely in practice |

**Takeaway:** organize your atlases so sprites that render together share the same page. Longer batch runs generally mean better performance.

## Optimization Strategies

### For All Games

- **Use coalesced batch mode** (the default)  -  it tends to be faster in most realistic scenarios
- **Pack related sprites onto the same atlas page**  -  minimizes batch breaks
- **Use setter methods** (`SetPosition`, `SetScale`) or call `Invalidate()` after bulk field assignments  -  ensures dirty flags propagate correctly
- **Set `Visible = false`** on off-screen containers  -  skips entire subtrees

### For Large Worlds

- **Enable camera culling**  -  `cam.CullEnabled = true` skips nodes outside the viewport
- **Use `CacheAsTree`** for semi-static subtrees (stable structure, occasional changes)
- **Use `CacheAsTexture`** for fully static content (backgrounds, decorations)

### For Complex Effects

- **Combine filters with `CacheAsTexture`**  -  avoids re-applying shaders every frame on static content
- **Use `WorldSpace: false`** on particle emitters when possible  -  avoids per-particle world transform

## Next Steps

- [Scene](?page=scene)  -  scene configuration and batch modes
- [Nodes](?page=nodes)  -  node types, visual properties, and tree manipulation

## Related

- [Architecture](?page=architecture)  -  render pipeline and sort/batch details
- [CacheAsTree](?page=cache-as-tree)  -  command list caching for semi-static subtrees
- [CacheAsTexture](?page=cache-as-texture)  -  offscreen render caching for static content
