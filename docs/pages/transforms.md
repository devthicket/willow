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

| Property | Fields | Setter | Description |
|----------|--------|--------|-------------|
| Position | `X`, `Y` | `SetPosition(x, y)` | Local pixel offset from parent |
| Scale | `ScaleX`, `ScaleY` | `SetScale(sx, sy)` | Multiplier (1.0 = original) |
| Rotation | `Rotation` | `SetRotation(r)` | Radians, clockwise |
| Skew | `SkewX`, `SkewY` | `SetSkew(sx, sy)` | Shear in radians |
| Pivot | `PivotX`, `PivotY` | `SetPivot(px, py)` | Transform origin in local pixels |
| Alpha | `Alpha` | `SetAlpha(a)` | Opacity [0,1], inherited from parent |

## Setter Methods vs. Direct Assignment

**Setter methods** (`SetPosition`, `SetScale`, etc.) automatically:
1. Mark the node's transform as dirty
2. Invalidate ancestor cache-as-tree caches

**Direct field assignment** (`node.X = 10`) does *not* trigger dirty marking. Use it for bulk updates, then call `Invalidate()` once:

```go
// Bulk update  -  one dirty mark
node.X = 100
node.Y = 200
node.ScaleX = 2
node.Invalidate()

// Or use setters for automatic handling
node.SetPosition(100, 200)
node.SetScale(2, 2)
```

## Pivot Point

The pivot determines the center of rotation and scale:

```go
sprite.PivotX = 32  // center of a 64px sprite
sprite.PivotY = 32
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
parent.X = 100
parent.Y = 100

child := willow.NewSprite("child", region)
child.X = 50  // world position = (150, 100)
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
- [Sprites & Atlas](?page=sprites-and-atlas)  -  loading texture atlases and regions
- [Camera & Viewport](?page=camera-and-viewport)  -  camera transforms and screen-to-world conversion

## Related

- [Nodes](?page=nodes)  -  node types, visual properties, and tree manipulation
- [Architecture](?page=architecture)  -  how dirty flags fit into the render pipeline
