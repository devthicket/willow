# Tilemap Data Access API

## Summary

Expose read-only access to tilemap layer data so external systems (pathfinding, gameplay logic, AI, collision generation) can query tile information without reaching into internals. Willow should **not** implement pathfinding or gameplay logic — but it should make it trivial to hook them up.

## Current State

`Layer` exposes:
- `SetTile(col, row, gid)` — write a single tile
- `SetData(data, w, h)` — replace entire layer data
- `SetAnimations(anims)` — set animation data
- `InvalidateBuffer()` — mark for rebuild
- `Node()` — get the scene graph node

Missing: no way to **read** tile data back. No `TileAt()`, no `Width()`/`Height()`, no property queries. The `Data` field is exported but only as an implementation detail — there's no documented read API. External systems would have to import `internal/tilemap` and access `layer.Data` directly, breaking encapsulation.

## Proposed API

```go
// --- Layer read access ---

// TileAt returns the GID at (col, row), with flip flags stripped.
// Returns 0 for out-of-bounds or empty tiles.
func (l *Layer) TileAt(col, row int) uint32

// RawTileAt returns the raw GID including flip flags.
func (l *Layer) RawTileAt(col, row int) uint32

// Dimensions returns (width, height) in tiles.
func (l *Layer) Dimensions() (cols, rows int)

// TileSize returns the pixel dimensions of a single tile.
// (Delegates to the parent Viewport's TileWidth/TileHeight.)
func (l *Layer) TileSize() (w, h int)

// --- Viewport read access ---

// LayerCount returns the number of layers.
func (v *Viewport) LayerCount() int

// Layer returns the layer at the given index.
func (v *Viewport) Layer(index int) *Layer

// TileToWorld converts tile coordinates to world-space pixel coordinates
// (top-left corner of the tile).
func (v *Viewport) TileToWorld(col, row int) (x, y float64)

// WorldToTile converts world-space coordinates to tile coordinates.
func (v *Viewport) WorldToTile(x, y float64) (col, row int)
```

### For external systems (pathfinding, collision, etc.)

```go
// TileQuery provides a read-only view of tilemap data suitable for
// external systems. Avoids exposing internal types.
type TileQuery struct {
    Cols, Rows     int
    TileW, TileH   int
    TileAt         func(col, row int) uint32  // GID lookup (flags stripped)
}

// Query returns a TileQuery for the given layer, usable by external packages.
func (v *Viewport) Query(layerIndex int) TileQuery
```

This lets a pathfinder or collision builder work with a simple struct + function — no import of `internal/tilemap` needed.

## Design Notes

- **Read-only**: These methods never modify state or mark buffers dirty. Safe to call from any goroutine after scene setup.
- **TileQuery as a bridge**: External packages (user's pathfinder, physics layer builder, etc.) receive a `TileQuery` value. They don't need to know about `Layer`, `Viewport`, or any Willow internals. This is the key design — Willow provides the data, the user provides the logic.
- **WorldToTile / TileToWorld**: These are critical for gameplay. "What tile is the player standing on?" and "Where in the world is tile (5, 3)?" are the two most common queries. They account for the viewport node's world transform.
- **Property access is out of scope**: Tiled custom properties (walkable, damage, terrain type) are application-specific. Users should maintain their own property lookup keyed by GID. Willow can optionally pass through raw Tiled properties at load time, but that's a separate concern.
- **No pathfinding, no collision generation**: Those are gameplay systems. The API here just makes the data available. A user's A* implementation takes a `TileQuery` and builds its own walkability grid.

## Implementation Scope

Small — ~40-60 lines of trivial accessor methods. The `TileQuery` bridge is ~10 lines. No new dependencies, no performance implications.

## Prior Art

- Ludum (`tilemap_node.go`): Tightly couples pathfinding (A*) into the tilemap package. `GetPathfinder()` returns a built-in pathfinder that reads tile data directly. This works but means the tilemap package owns gameplay logic. Willow should avoid this — expose data, don't own algorithms.
- Godot `TileMap`: Exposes `get_cell()`, `map_to_local()`, `local_to_map()`. Navigation/pathfinding is a separate `NavigationServer` that reads tilemap data through a clean interface.
