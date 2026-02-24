# Mesh & Distortion

Willow supports custom vertex geometry through mesh nodes, plus a high-level distortion grid for deformable surfaces like water, heat haze, or cloth.

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

## Distortion Grid

<p align="center">
  <img src="gif/watermesh.gif" alt="Water mesh demo" width="400">
</p>

A grid mesh that deforms a texture  -  useful for water, heat haze, or cloth effects:

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

## Next Steps

- [Ropes](?page=ropes)  -  textured strips along curved paths
- [Lighting](?page=lighting)  -  2D light layer compositing

## Related

- [Polygons](?page=polygons)  -  ear-clip triangulated polygon shapes
- [Nodes](?page=nodes)  -  node types and visual properties
- [Offscreen Rendering](?page=offscreen-rendering)  -  render targets for mesh effects
