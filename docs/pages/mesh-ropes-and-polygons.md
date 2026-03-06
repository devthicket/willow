# Mesh, Ropes & Polygons

<p align="center">
  <img src="gif/watermesh.gif" alt="Water mesh demo" width="400">
</p>

Willow supports custom vertex geometry through mesh nodes, plus high-level helpers for ropes, distortion grids, and polygons.

## Raw Mesh

Create a node with custom vertices and indices:

```go
vertices := []ebiten.Vertex{
    {DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
    {DstX: 100, DstY: 0, SrcX: 64, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
    {DstX: 100, DstY: 100, SrcX: 64, SrcY: 64, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
    {DstX: 0, DstY: 100, SrcX: 0, SrcY: 64, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
}

indices := []uint16{0, 1, 2, 0, 2, 3}

mesh := willow.NewMesh("quad", textureImage, vertices, indices)
scene.Root().AddChild(mesh)
```

The mesh fields are public and can be modified at runtime:

```go
mesh.Vertices[0].DstX = 10
mesh.InvalidateMeshAABB()  // call after modifying vertices
```

## Rope

A rope renders a textured strip along a path with configurable curve modes:

```go
start := &willow.Vec2{X: 100, Y: 100}
end := &willow.Vec2{X: 400, Y: 300}

rope := willow.NewRope("cable", ropeImage, nil, willow.RopeConfig{
    Width:     8,
    CurveMode: willow.RopeCurveCatenary,
    Segments:  30,
    Start:     start,
    End:       end,
    Sag:       50,
})
scene.Root().AddChild(rope.Node())
```

### Curve Modes

| Mode | Description |
|------|-------------|
| `RopeCurveLine` | Straight line between endpoints |
| `RopeCurveCatenary` | Hanging cable (uses `Sag` field) |
| `RopeCurveQuadBezier` | Quadratic Bezier (`Controls[0]`) |
| `RopeCurveCubicBezier` | Cubic Bezier (`Controls[0]`, `Controls[1]`) |
| `RopeCurveWave` | Sine wave (`Amplitude`, `Frequency`, `Phase`) |
| `RopeCurveCustom` | User-provided path via `PointsFunc` |

### Updating a Rope

Bind `Start` and `End` to `*Vec2` pointers  -  mutate the values, then call `Update()`:

```go
start.X = mouseX
start.Y = mouseY
rope.Update()
```

Or override the full path:

```go
rope.SetPoints([]willow.Vec2{{X: 0, Y: 0}, {X: 50, Y: 100}, {X: 100, Y: 0}})
```

### Join Modes

| Mode | Description |
|------|-------------|
| `RopeJoinMiter` | Sharp corners (default) |
| `RopeJoinBevel` | Flat-cut corners |

## DistortionGrid

A grid mesh that deforms a texture:

```go
grid := willow.NewDistortionGrid("water", waterImage, 10, 8)
scene.Root().AddChild(grid.Node())
```

### Manipulating Vertices

```go
// Displace a single vertex
grid.SetVertex(col, row, deltaX, deltaY)

// Displace all vertices with a function
grid.SetAllVertices(func(col, row int, restX, restY float64) (dx, dy float64) {
    return math.Sin(restY*0.1+time) * 5, 0
})

// Reset all displacements to zero
grid.Reset()
```

Query grid dimensions:

```go
cols := grid.Cols()
rows := grid.Rows()
```

## Polygon

Create polygon shapes from a list of points (ear-clip triangulated):

```go
// Untextured polygon (uses WhitePixel)
poly := willow.NewPolygon("triangle", []willow.Vec2{
    {X: 0, Y: 0}, {X: 100, Y: 0}, {X: 50, Y: 80},
})
poly.SetColor(willow.RGB(0, 1, 0))

// Textured polygon
texPoly := willow.NewPolygonTextured("shape", textureImg, []willow.Vec2{
    {X: 0, Y: 0}, {X: 200, Y: 0}, {X: 200, Y: 150}, {X: 0, Y: 150},
})
```

For common shapes, use the built-in constructors:

```go
hex := willow.NewRegularPolygon("hexagon", 6, 50)   // 6 sides, radius 50
star := willow.NewStar("star", 60, 25, 5)            // 5-pointed star
```

Update points at runtime:

```go
willow.SetPolygonPoints(poly, newPoints)
```
