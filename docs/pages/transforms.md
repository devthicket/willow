# Transforms

<p align="center">
  <img src="gif/shapes.gif" alt="Shapes demo" width="400">
</p>

Every node has a local transform defined by position, scale, rotation, skew, and pivot. Willow computes world transforms by multiplying parent and child matrices, using dirty flags to avoid unnecessary recomputation.

## Coordinate System

- **Origin**: top-left corner of the screen/parent
- **Y-axis**: increases downward
- **Rotation**: clockwise, in radians

## Transform Properties

| Property | Getter | Setter | Description |
|----------|--------|--------|-------------|
| Position | `X()`, `Y()` | `SetPosition(x, y)` / `SetX(x)` / `SetY(y)` | Local pixel offset from parent |
| Scale | `ScaleX()`, `ScaleY()` | `SetScale(sx, sy)` | Multiplier (1.0 = original) |
| Size | `Width()`, `Height()` | `SetSize(w, h)` | Pixel dimensions |
| Rotation | `Rotation()` | `SetRotation(r)` | Radians, clockwise |
| Skew | `SkewX()`, `SkewY()` | `SetSkew(sx, sy)` | Shear in radians |
| Pivot | `PivotX()`, `PivotY()` | `SetPivot(px, py)` | Transform origin in local pixels |
| Alpha | `Alpha()` | `SetAlpha(a)` | Opacity [0,1], inherited from parent |

## Setter Methods vs. Direct Assignment

**Setter methods** (`SetPosition`, `SetScale`, etc.) automatically:
1. Mark the node's transform as dirty
2. Invalidate ancestor cache-as-tree caches

Node fields are private; use **setter methods** for all mutations:

```go
node.SetPosition(100, 200)
node.SetScale(2, 2)
```

## Pivot Point

The pivot determines the center of rotation and scale:

```go
sprite.SetPivot(32, 32)  // center of a 64px sprite
sprite.SetRotation(math.Pi / 4)  // rotates around center
```

By default, pivot is `(0, 0)` (top-left corner).

## Coordinate Conversion

Convert between a node's local space and world space:

```go
// World to local (e.g., for hit testing)
localX, localY := node.WorldToLocal(worldX, worldY)

// Local to world (e.g., for spawning effects at a node's position)
worldX, worldY := node.LocalToWorld(0, 0)  // node's origin in world space
```

## Transform Inheritance

Child transforms are relative to their parent:

```go
parent := willow.NewContainer("parent")
parent.SetPosition(100, 100)

child := willow.NewSprite("child", region)
child.SetX(50)  // world position = (150, 100)
parent.AddChild(child)
```

Scaling and rotation are also inherited  -  scaling a parent scales all children. Alpha is multiplicatively inherited.

## Dirty Flag System

Willow tracks whether a node's world transform needs recomputation via dirty flags. A node is marked dirty when:

- Any transform property changes via a setter method
- `Invalidate()` is called explicitly
- A parent's transform changes (propagates to all descendants)

World matrices are lazily recomputed during `scene.Update()` only for nodes that are actually dirty, avoiding unnecessary matrix math for static nodes.

## Next Steps

- [Solid-Color Sprites](?page=solid-color-sprites)  -  creating shapes with scale and color
- [Sprites](?page=sprites)  -  TextureRegion and image display
- [Camera & Viewport](?page=camera-and-viewport)  -  camera transforms and screen-to-world conversion

## Related

- [Nodes](?page=nodes)  -  node types, visual properties, and tree manipulation
- [Architecture](?page=architecture)  -  how dirty flags fit into the render pipeline
