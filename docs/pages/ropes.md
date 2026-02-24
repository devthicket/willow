# Ropes

A rope renders a textured strip along a path with configurable curve modes. Use ropes for cables, chains, tentacles, laser beams, or any geometry that follows a curved path.

## Creating a Rope

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

## Curve Modes

| Mode | Description |
|------|-------------|
| `RopeCurveLine` | Straight line between endpoints |
| `RopeCurveCatenary` | Hanging cable (uses `Sag` field) |
| `RopeCurveQuadBezier` | Quadratic Bezier (`Controls[0]`) |
| `RopeCurveCubicBezier` | Cubic Bezier (`Controls[0]`, `Controls[1]`) |
| `RopeCurveWave` | Sine wave (`Amplitude`, `Frequency`, `Phase`) |
| `RopeCurveCustom` | User-provided path via `PointsFunc` |

## Updating a Rope

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

## Join Modes

| Mode | Description |
|------|-------------|
| `RopeJoinMiter` | Sharp corners (default) |
| `RopeJoinBevel` | Flat-cut corners |

## Next Steps

- [Lighting](?page=lighting)  -  2D light layer compositing
- [Clipping & Masks](?page=clipping-and-masks)  -  alpha-based masking

## Related

- [Mesh & Distortion](?page=meshes)  -  raw vertex geometry and distortion grids
- [Polygons](?page=polygons)  -  ear-clip triangulated polygon shapes
- [Tweens & Animation](?page=tweens-and-animation)  -  animate rope endpoints
