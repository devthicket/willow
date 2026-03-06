# Polygons

Willow can create polygon shapes from a list of points, automatically triangulated using ear clipping. Use polygons for irregular shapes, terrain outlines, or any geometry defined by a point list.

## Creating Polygons

```go
// Untextured polygon (uses WhitePixel)
poly := willow.NewPolygon("triangle", []willow.Vec2{
    {X: 0, Y: 0}, {X: 100, Y: 0}, {X: 50, Y: 80},
})
poly.SetColor(willow.RGB(0, 1, 0))
```

```go
// Textured polygon
texPoly := willow.NewPolygonTextured("shape", textureImg, []willow.Vec2{
    {X: 0, Y: 0}, {X: 200, Y: 0}, {X: 200, Y: 150}, {X: 0, Y: 150},
})
```

## Regular Polygons and Stars

For common shapes, use the built-in constructors instead of computing vertices manually:

```go
// Regular hexagon centered at origin with radius 50
hex := willow.NewRegularPolygon("hexagon", 6, 50)
hex.SetColor(willow.RGB(0, 1, 0))

// 5-pointed star with outer radius 60 and inner radius 25
star := willow.NewStar("star", 60, 25, 5)
star.SetColor(willow.RGB(1, 1, 0))
```

`NewRegularPolygon` accepts any side count (minimum 3). The first vertex points straight up. `NewStar` alternates between outer tips and inner valleys.

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
