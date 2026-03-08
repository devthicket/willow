# Issues — internal architecture v2 migration

Tracked issues discovered during Phases 1–8. Items marked `[fixed]` were
resolved during migration; unmarked items need attention in later phases.

---

## Phase 2a — particle/, text/

### `rebuildPixelVerts` not migrated as separate method
Root has `(tb *TextBlock) rebuildPixelVerts(f *PixelFont)` as a separate
method from `rebuildLocalVerts`. Internal `text.TextBlock.RebuildLocalVerts`
takes a `lineHeight` parameter and handles both SDF and pixel fonts. The
root calls `rebuildPixelVerts(f)` in `emitPixelTextCommand` but internal
render calls `tb.RebuildLocalVerts(float64(f.CellH))`. Functionally
equivalent but the root facade must stop calling the old method.

---

## Phase 6 — render/

### [fixed] Root `emitNodeCommand` only emits SDF for text nodes
`rendertarget.go:406-408` calls `emitSDFTextCommand` for ALL text nodes
regardless of font type. The main `traverse` (render.go:220-225) properly
dispatches via type switch on `*SpriteFont` / `*PixelFont`.
Internal `render.EmitNodeCommand` fixes this by checking
`text.IsPixelFont(tb.Font)` and dispatching to the correct emit function.

### `CustomEmit` receives `*Pipeline` instead of `*Scene`
Root signature: `customEmit func(s *Scene, treeOrder *int)`
Internal node: `CustomEmit func(s any, treeOrder *int)`
Pipeline calls `n.CustomEmit(p, treeOrder)` passing `*Pipeline`.

Users of `CustomEmit` (e.g. `TileMapLayer.emitCommands`) access
`s.viewTransform` and `s.commands`. When migrated, they must type-assert
the `any` to `*render.Pipeline` and use `p.ViewTransform` / `p.AppendCommand`.
This is the planned cycle-breaking pattern (parameters over back-pointers)
but the `TileMapLayer` code needs updating in the root facade phase.

### `ShouldCullFn` is a function pointer on render/
`render.ShouldCullFn` must be wired by the root to `camera.ShouldCull`.
If not wired, culling is silently disabled (nodes are never culled).
This is by design but easy to forget during integration.

### `BlendMaskFn` function pointer
Used in `renderSpecialNode` for mask compositing. Must be wired by root to
return `BlendMask` (a `types.BlendMode`). If nil, mask compositing uses
the zero-value blend mode which produces incorrect results.

### `AtlasPageFn` / `EnsureMagentaImageFn` function pointers
Must be wired by root. If nil, all atlas-based sprites render as nothing.
The `resolveAtlasPage` helper in batch.go silently returns nil.

### Particle batch uses `ParticleRenderData` accessor
Root accesses `e.particles[i].x, .y, .scale, .alpha, .colorR, .colorG, .colorB`
directly (unexported fields). Internal render uses
`e.ParticleRenderData(i)` which returns the same 7 values. Negligible
overhead (single method call per particle per frame, inlined by compiler).

---

## Phase 7 — input/

### `HitRect`, `HitCircle`, `HitPolygon` still defined in root
Root `input.go` defines `HitRect`, `HitCircle`, and `HitPolygon` as exported
types implementing `HitShape`. These are user-facing API types that belong in
the root package (or types/). The internal `input/` package relies on
`types.HitShape` interface only and does not redefine these. No migration
needed — they stay in root.

### `NodeDimensions` needed by hit testing
`nodeContainsLocal` in input/ needs node dimensions for AABB fallback when
no HitShape is set. Uses `NodeDimensionsFn` function pointer (same pattern
as render/ and camera/). This is a third copy of the dimension logic,
but input/ can't import camera/ or render/.

### `emitInteractionEvent` bridged via function pointer
Root's `emitInteractionEvent` checks `s.store` (EntityStore) and builds an
`InteractionEvent`. Internal input/ uses `EmitInteractionEventFn` which the
root wires to call `store.EmitEvent`. The `InteractionEvent` struct stays in
root since it references the ECS store.

---

## Phase 8 — core/

### TweenGroup convenience constructors stay in root
`TweenPosition`, `TweenScale`, `TweenColor`, `TweenAlpha`, `TweenRotation`
access node fields directly (`&node.x`, `&node.color.r`, etc.). These are
unexported from the node's perspective in different packages. The `TweenGroup`
struct and its `Tick`/`Update`/`Cancel` methods moved to core/, but the
constructors remain in root because they need to take pointers to unexported
`types.Color` fields (`r`, `g`, `b`, `a`). Root facade (Phase 9) will keep
these constructors and call `core.Scene.RegisterTween`.

### `autoRegister` pattern needs function pointer
Root's `autoRegister` calls `node.scene.registerTween(g)`. In internal,
`node.Scene_` is `any`. Core/ can't wire this directly without a function
pointer since node/ can't import core/. Phase 9 must wire a
`node.RegisterTweenFn` or have root constructors call `core.RegisterTween`
explicitly after creating the TweenGroup.

### `countBatches` / `countDrawCalls` use `commandBatchKey`
Root's `countBatches` calls `commandBatchKey(&cmd)` (unexported).
Internal `core.CountBatches` calls `render.CommandBatchKey(&cmd)` (exported).
Required exporting `CommandBatchKey` from render/ (already was exported).

### `submitBatchesCoalesced` exported
Root called `s.submitBatchesCoalesced(target)` (unexported). Core calls
`p.SubmitBatchesCoalesced(target)`. Required renaming to exported in render/.

---

## Phase 9 — root facade (in progress)

### Callback setter rename: `SetOn*` → `On*`
Root public API uses `n.OnPointerDown(fn)`, `n.OnClick(fn)`, etc. Internal
node/ originally used `SetOnPointerDown(fn)`, etc. Renamed all 10 setters
to match root's API. No callers in internal/ were affected (only defined,
never called from internal/).

### Getter methods added for renamed fields
Added `Visible()`, `Renderable()`, `CustomImage()`, `TextureRegion()`,
`SkewX()`, `SkewY()`, `PivotX()`, `PivotY()` getters to match root API.
Fields were renamed with underscore suffix in a prior sub-task.

### `RenderLayer` / `GlobalOrder` — no getter methods
Root has `RenderLayer()` and `GlobalOrder()` getter methods returning
unexported fields. In internal/node, these are exported fields (`n.RenderLayer`,
`n.GlobalOrder`), so adding a getter method with the same name would conflict.
Users access them directly via the exported field. No issue for type alias.

### `Scene()` method stays in root
`n.Scene()` returns `*Scene`, which is defined in root. Can't live in node/
since node/ doesn't know about Scene. Root facade will define this method
(or it stays as-is in the root's existing `node.go`). Since `type Node = node.Node`
is a type alias, methods can be defined on it in root only if root is the
defining package — but it's NOT with a type alias. **This means `Scene()` must
move to internal/node as `Scene() any` and root casts the return.** Or root
keeps its own Node type (not alias). Needs resolution.

### `ToTexture(s *Scene)` stays in root
Requires `*Scene` and calls `subtreeBounds`, `renderSubtree` — both reference
root-level render logic. Cannot move to node/ or core/ without significant
refactoring. Stays in root.

### Mask/Cache/MeshAABB methods added to node/
`SetMask`, `ClearMask`, `GetMask`, `SetCacheAsTexture`, `InvalidateCache`,
`IsCacheEnabled`, `InvalidateMeshAABB` added to internal/node/methods.go.
These use exported field names (`MaskNode`, `CacheEnabled`, `CacheTexture`,
`CacheDirty`, `Mesh.AabbDirty`).

### [done] Color alias
`type Color = types.Color` in root. All `c.r` → `c.R()` conversions done.
`types.ColorFieldPtrs(&c)` added for TweenColor field pointer access.
`toRGBA()` method converted to `colorToRGBA()` free function.

### [done] Foundation type aliases
Vec2, Rect, Range, BlendMode, NodeType, EventType, MouseButton,
KeyModifiers, TextAlign, TextureRegion, CacheTreeMode, HitShape aliased
from internal/types. Constants re-exported.

### Root facade — remaining work
- Alias Node (`type Node = node.Node`) — requires updating ~381 field
  references from unexported to exported names across root .go files
- Remove duplicate method implementations from root .go files
- Delete root files fully replaced by internal/ (render.go, batch.go,
  transform.go, mask.go, camera.go, input.go, text.go, particle.go,
  filter.go, lightlayer.go, tilemap.go, atlas.go, etc.)
- Wire all ~20 function pointers in root init or constructors
- Handle `TileMapLayer.emitCommands` which casts `any` to `*Scene`
  (must become `*render.Pipeline`)
- Integration tests
- Resolve `Scene()` method ownership (see above)

---

## Cross-phase — integration concerns

### Root facade must wire all function pointers
The following function pointers must be wired during root init:
- `render.AtlasPageFn` — atlas page resolution
- `render.EnsureMagentaImageFn` — placeholder image
- `render.ShouldCullFn` — camera culling
- `render.BlendMaskFn` — mask blend mode
- `render.PageFn` — used by `RenderTexture.DrawSprite` (rendertexture.go)
- `render.MagentaImageFn` — used by `ResolvePage` (rendertexture.go)
- `render.NewSpriteFn` — used by `RenderTexture.NewSpriteNode`
- `render.EnsureSDFShaderFn` — (currently unused, shaders compiled in render/)
- `render.EnsureMSDFShaderFn` — (currently unused, shaders compiled in render/)
- `node.InvalidateAncestorCacheFn`
- `node.RegisterAnimatedInCacheFn`
- `node.SetCacheAsTreeFn`
- `node.InvalidateCacheTreeFn`
- `node.IsCacheAsTreeEnabledFn`
- `node.PropagateSceneFn`
- `input.ScreenToWorldFn` — camera screen-to-world conversion
- `input.NodeDimensionsFn` — node dimensions for hit testing
- `input.EmitInteractionEventFn` — ECS interaction bridge
- `input.RebuildSortedChildrenFn` — ZIndex sort for hit collection

### Duplicate page resolution paths
`render/rendertexture.go` has its own `PageFn` / `MagentaImageFn` function
pointers, while `render/batch.go` uses `AtlasPageFn` / `EnsureMagentaImageFn`.
These are two separate sets of function pointers that do the same thing.
Should be consolidated during root facade phase (Phase 9) to wire once.

### `WorldAABB` exists in both camera/ and render/
`camera.WorldAABB` (cull.go) and `render.WorldAABB` (pipeline.go) are
identical functions. render/ needs it for `SubtreeBounds` but can't import
camera/ (would create a cycle). Options:
1. Move `WorldAABB` to a shared location (types/ or node/)
2. Accept the duplication (both are ~10 lines, hot path)
Current choice: (2) — accepted duplication.

### `NodeDimensions` exists in both camera/ and render/
Same situation as `WorldAABB`. Both are ~20 lines. render/ needs it for
subtree bounds but can't import camera/.
Current choice: accepted duplication.
