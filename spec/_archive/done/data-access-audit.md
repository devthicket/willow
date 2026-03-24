# Data Access Audit: Gameplay System Integration

## Goal

Audit every Willow system and ensure it exposes the data that external gameplay layers need — physics engines, collision detection, AI/pathfinding, ECS frameworks, inventory systems, dialog systems, etc. Willow is a rendering and UI layer, not a game engine. Its job is to make integration trivial, not to own gameplay logic.

The guiding principle: **if a gameplay system needs to read it, Willow should expose it cleanly. If it only matters for rendering internals, keep it private.**

## Current Gaps by System

### HIGH PRIORITY

#### Node World Position & Bounds
**Problem:** No direct way to get a node's world-space position or bounding box.
- `WorldTransform [6]float64` is a public field but undocumented as an API contract
- `WorldAlpha` has no getter
- World-space AABB exists internally (`camera/cull.go:WorldAABB()`, `render/pipeline.go:SubtreeBounds()`) but is not exported

**Why it matters:** Every physics engine, collision system, and AI system needs "where is this thing in the world" and "how big is it." These are the two most fundamental queries. Users currently must call `LocalToWorld(0, 0)` as a workaround for position, and manually compute bounds from `Width()`/`Height()` + transform — which doesn't account for rotation, pivot, or scale correctly.

**Proposed:**
```go
func (n *Node) WorldPosition() (x, y float64)       // center in world space
func (n *Node) WorldBounds() Rect                    // AABB in world space
func (n *Node) WorldAlpha() float64                  // computed alpha
```

#### Input State Queries
**Problem:** Input is callback-only. No way to poll current state.
- No `IsPointerDown()`, no `PointerPosition()`, no `IsKeyDown()`
- Gameplay code that needs "is the player pressing jump right now" must manually track state from callbacks

**Why it matters:** Callback-driven input works for UI (click handlers). Gameplay systems use polling — they check input state during their update tick. Every game framework exposes both patterns.

**Proposed:**
```go
func (s *Scene) PointerPosition() (x, y float64)     // current pointer in screen space
func (s *Scene) IsPointerDown(button MouseButton) bool
// Keyboard: Willow may intentionally leave this to Ebitengine's inpututil directly.
// If so, document that clearly.
```

### MEDIUM PRIORITY

#### Camera View Matrix
**Problem:** `viewMatrix` and `invViewMatrix` are private. Users can call `WorldToScreen()`/`ScreenToWorld()` per-point, but can't get the matrix itself for batch transforms or shader uniforms.

**Why it matters:** Custom shaders, minimap rendering, and batch coordinate conversion all want the raw matrix. Per-point conversion is O(n) when a matrix multiply would be O(1) setup.

**Proposed:**
```go
func (c *Camera) ViewMatrix() [6]float64
func (c *Camera) InverseViewMatrix() [6]float64
```

#### Node-to-Node Spatial Queries
**Problem:** No helper to compute distance or relative position between two nodes in world space.

**Why it matters:** AI systems constantly ask "how far is enemy from player" and "what direction is the target." Users can compute this from `LocalToWorld()` on both nodes, but it's boilerplate that every game writes.

**Proposed:**
```go
func DistanceBetween(a, b *Node) float64
func DirectionBetween(a, b *Node) Vec2  // normalized
```
These are convenience functions, not methods — they don't need to live on Node.

#### Tilemap Read Access
**Already covered in `queue/tilemap-data-access.md`.** Including here for completeness:
- `TileAt(col, row)` — read a tile
- `Dimensions()` — map size
- `WorldToTile()` / `TileToWorld()` — coordinate conversion
- `TileQuery` struct for external system bridging

### LOW PRIORITY

#### Tween Introspection
**Problem:** Can register and tick tweens, but can't query what's active. `scene.tweens` is private.

**Why it matters:** Debug tools, UI that shows "animation in progress," and systems that need to wait for a tween to finish.

**Proposed:**
```go
func (s *Scene) ActiveTweens() []*TweenGroup     // or just a count
func (g *TweenGroup) IsComplete() bool
```

#### Node Tree Utilities
**Problem:** No sibling queries, no depth calculation, no path-to-root.

**Why it matters:** Minor — tree traversal is adequate via `Children()`/`Parent`. These are convenience methods. Only add if users request them.

#### Filter Introspection
**Problem:** `node.Filters` is `[]any` — can't inspect or modify the chain at runtime.

**Why it matters:** Very niche. Only needed for runtime shader toggling or debug visualization. Low priority.

## What's Already Well-Exposed

These systems are fine — no changes needed:

- **Particle emitter state**: `ParticleRenderData(i)`, `ParticlePos(i)`, `ParticleVelocity(i)`, `Bounds()` — comprehensive per-particle access
- **Camera position/zoom/rotation**: Direct field access, plus `VisibleBounds()`, `WorldToScreen()`/`ScreenToWorld()`
- **Node tree structure**: `Children()`, `FindChild()`, `FindDescendant()`, `NodeIndex` for tag-based lookup
- **Sprite texture regions**: `TextureRegion()` getter returns full region data
- **Node local transforms**: Full getter/setter coverage for position, scale, rotation, alpha
- **EmitterConfig**: All fields public, fully mutable

## Approach

This is an audit + incremental fix, not a rewrite. Steps:

1. **Catalog** — list every `internal/` function/field that computes data a gameplay system would need
2. **Classify** — rendering-internal (keep private) vs gameplay-relevant (expose)
3. **Expose** — add public methods/functions, starting with HIGH priority
4. **Document** — each new accessor gets a godoc comment explaining the use case
5. **Test** — verify exposed data matches what the internal systems compute (no divergence)

The bar for adding a public method: "Would at least two different gameplay system categories (physics, AI, collision, UI, audio, etc.) need this data?" If yes, expose it. If only one very specific system needs it, it can stay internal.

## Non-Goals

- Do NOT add gameplay logic (pathfinding, physics, collision response)
- Do NOT expose rendering pipeline internals (command buffers, vertex arrays, sort state)
- Do NOT break existing API — all changes are additive
- Do NOT add abstractions or interfaces for hypothetical systems — expose concrete data
