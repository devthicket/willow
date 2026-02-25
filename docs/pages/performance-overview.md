# Performance

Willow is designed with performance in mind. This page explains why a retained-mode scene graph can be faster than hand-written rendering code, how Willow minimizes heap allocations per frame, and what tools are available to optimize your game.

## Why a Scene Graph Can Be Faster

A common assumption is that a scene graph adds overhead compared to calling `DrawImage` directly. In practice, Willow is **~2.6x faster** than naive raw `DrawImage` calls — because owning the entire command stream lets Willow accumulate vertices explicitly and submit them in bulk via `DrawTriangles32`, while also eliminating per-sprite heap allocations.

### Benchmark: Willow vs Raw Ebitengine (10K sprites)

*Tested on Apple M3 Max, Go 1.24, Ebitengine v2. Real-world atlas: 2× 4096×4096 pages, runs of 1,000 sprites per page. Results will vary by hardware, scene complexity, and atlas layout.*

| Layer | avg ns/op | vs Raw DrawImage |
|-------|-----------|-----------------|
| Raw `DrawTriangles32` (pre-computed) | ~391K | ~5.8x faster |
| **Willow** | **~875K** | **~2.6x faster** |
| Raw `DrawImage` | ~2,287K | — |

*Lower ns/op is better. Raw `DrawTriangles32` pre-computes all vertex positions outside the benchmark loop — it represents the theoretical floor if you did all the math yourself ahead of time.*

### Where Does This Come From?

1. **Vertex accumulation** — sprites sharing the same atlas page and blend mode have their vertices merged into a single `DrawTriangles32` submission, rather than 10,000 individual `DrawImage` calls
2. **Sort-based batching** — Willow sorts render commands by batch key (texture, blend mode, render layer), maximizing batch run lengths
3. **Allocation elimination** — preallocated buffers reduce per-frame allocations from ~30,000 (raw `DrawImage`) to ~31, cutting GC pressure significantly

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

| Scenario | Willow | Raw DrawImage |
|----------|--------|---------------|
| Real-world atlas (10K sprites, 2 pages) | 31 allocs/op | 30,000 allocs/op |
| 1,000 particles (coalesced) | 4 allocs/op | 3,001 allocs/op |

## Dirty Flag Transforms

World transforms are only recomputed when something actually changes. Willow tracks dirty state per node:

- Setting a transform property via a setter (`SetPosition`, `SetScale`, etc.) marks the node dirty
- Dirty state propagates to descendants during `updateWorldTransform`
- Clean nodes skip `computeLocalTransform` and the matrix multiply, though the tree is still walked to propagate parent state

### Measured Impact

| State | avg ns/op |
|-------|-----------|
| All nodes dirty | ~142K |
| All nodes clean | ~35K |

Clean frames are **~4x faster** than fully dirty frames — a significant win for scenes where most nodes are stationary each frame.

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
| Particles (1,000) | **~5x faster** | 45K vs 230K ns/op |
| Mixed sprites + particles | ~3% faster | Marginal |
| Single atlas page (10K) | Roughly equal | |
| Real-world atlas (2 pages) | Roughly equal | |
| Worst case (alternating pages) | ~13% slower | Pathological |

Coalesced mode is the default and recommended because of the large particle win, even though it's roughly neutral for sprite-only workloads. The worst case — alternating atlas pages every sprite — is pathological and unlikely in real games.

**Takeaway:** organize your atlases so sprites that render together share the same page. Longer batch runs generally mean better performance.

## Optimization Strategies

### For All Games

- **Use coalesced batch mode** (the default) — it tends to be faster in most realistic scenarios
- **Pack related sprites onto the same atlas page** — minimizes batch breaks
- **Use setter methods** (`SetPosition`, `SetScale`) or call `Invalidate()` after bulk field assignments — ensures dirty flags propagate correctly
- **Set `Visible = false`** on off-screen containers — skips entire subtrees

### For Large Worlds

- **Enable camera culling** — `cam.CullEnabled = true` skips nodes outside the viewport
- **Use `CacheAsTree`** for semi-static subtrees (stable structure, occasional changes)
- **Use `CacheAsTexture`** for fully static content (backgrounds, decorations)

### For Complex Effects

- **Combine filters with `CacheAsTexture`** — avoids re-applying shaders every frame on static content

## Reproduce These Benchmarks

All numbers on this page come from the benchmark suite in the Willow repository. To reproduce on your hardware:

```
go test -bench='BenchmarkDraw_RealWorldAtlas|BenchmarkRaw_RealWorldAtlas' -benchmem -count=3
go test -bench='BenchmarkTransform' -benchmem -count=3
go test -bench='BenchmarkParticle' -benchmem -count=3
```

## Next Steps

- [Scene](?page=scene) — scene configuration and batch modes
- [Nodes](?page=nodes) — node types, visual properties, and tree manipulation

## Related

- [Architecture](?page=architecture) — render pipeline and sort/batch details
- [CacheAsTree](?page=cache-as-tree) — command list caching for semi-static subtrees
- [CacheAsTexture](?page=cache-as-texture) — offscreen render caching for static content
