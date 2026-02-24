# Polygons

Willow can create polygon shapes from a list of points, automatically triangulated using ear clipping. Use polygons for irregular shapes, terrain outlines, or any geometry defined by a point list.

## Creating Polygons

```go
// Untextured polygon (uses WhitePixel)
poly := willow.NewPolygon("triangle", []willow.Vec2{
    {X: 0, Y: 0}, {X: 100, Y: 0}, {X: 50, Y: 80},
})
poly.Color = willow.Color{R: 0, G: 1, B: 0, A: 1}
```

```go
// Textured polygon
texPoly := willow.NewPolygonTextured("shape", textureImg, []willow.Vec2{
    {X: 0, Y: 0}, {X: 200, Y: 0}, {X: 200, Y: 150}, {X: 0, Y: 150},
})
```

## Updating Points

Update the polygon's shape at runtime:

```go
willow.SetPolygonPoints(poly, newPoints)
```

This re-triangulates the polygon with the new point list.

## Next Steps

- [Offscreen Rendering](?page=offscreen-rendering)  -  render targets and compositing
- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  making nodes interactive

## Related

- [Mesh & Distortion](?page=meshes)  -  raw vertex geometry and distortion grids
- [Ropes](?page=ropes)  -  textured strips along curved paths
- [Solid-Color Sprites](?page=solid-color-sprites)  -  simpler approach for rectangles
